package analysis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/swiftbit/know-agent/pkg/logx"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/document/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/enum"
)

// ProfileGenerator 文档画像生成器
type ProfileGenerator interface {
	Generate(ctx context.Context, documentId int64, analysisResult *Result, structureNodes []*entity.StructureNode) (*entity.DocumentProfile, error)
}

// SaveStage 保存阶段：按顺序执行文本上传、产物持久化、结构节点持久化、文档画像生成
type SaveStage struct {
	repo    adapter.DocumentRepository
	storage adapter.Storage
	gen     ProfileGenerator
}

// NewSaveStage 创建保存阶段
func NewSaveStage(repo adapter.DocumentRepository, storage adapter.Storage, gen ProfileGenerator) *SaveStage {
	return &SaveStage{
		repo:    repo,
		storage: storage,
		gen:     gen,
	}
}

// Name 阶段名称
func (p *SaveStage) Name() string {
	return "保存阶段"
}

// Execute 执行保存阶段，依次执行各子阶段
func (p *SaveStage) Execute(ctx context.Context, parseCtx *Context) error {
	documentId := parseCtx.DocumentId
	taskId := parseCtx.TaskId
	analysisResult := parseCtx.AnalysisResult

	// 解析文本上传
	parsedTextPath, err := p.executeParsedTextUpload(ctx, documentId, analysisResult)
	if err != nil {
		return fmt.Errorf("解析文本上传失败: %w", err)
	}

	// 解析产物保存（文档块）
	if err = p.executeArtifactPersist(ctx, documentId, taskId, analysisResult); err != nil {
		return fmt.Errorf("解析产物保存失败: %w", err)
	}

	// 结构节点持久化
	structureNodes, err := p.executeStructurePersist(ctx, documentId, taskId, analysisResult)
	if err != nil {
		return fmt.Errorf("结构节点持久化失败: %w", err)
	}

	// 文档画像生成
	if err = p.executeProfileGenerate(ctx, documentId, analysisResult, structureNodes); err != nil {
		return fmt.Errorf("文档画像生成失败: %w", err)
	}

	parseCtx.StructureNodes = structureNodes
	parseCtx.ParsedTextPath = parsedTextPath

	detail, _ := json.Marshal(map[string]any{
		"parsedTextPath":     parsedTextPath,
		"blockCount":         len(analysisResult.Blocks),
		"structureNodeCount": len(structureNodes),
	})

	saveLog := &entity.DocumentTaskLog{
		TaskId:       parseCtx.Task.ID,
		DocumentId:   parseCtx.DocumentId,
		StageType:    enum.TaskStageContentParse,
		EventType:    enum.TaskEventComplete,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "解析产物入库完成，已保存 parsed text、block、structure 和文档画像。",
		DetailJson:   string(detail),
	}
	_ = p.repo.InsertTaskLog(ctx, saveLog)

	return nil
}

// executeParsedTextUpload 上传解析后的文本，返回文本存储路径
func (p *SaveStage) executeParsedTextUpload(ctx context.Context, documentId int64, analysisResult *Result) (string, error) {
	return p.storage.UploadParsedText(ctx, documentId, analysisResult.ParsedText)
}

// executeArtifactPersist 持久化解析产物（文档块）
func (p *SaveStage) executeArtifactPersist(ctx context.Context, documentId, taskId int64, analysisResult *Result) error {
	if analysisResult == nil {
		return nil
	}
	blocks, err := p.buildDocumentBlockEntities(ctx, documentId, taskId, analysisResult.Blocks)
	if err != nil {
		return err
	}
	txFn := func(txCtx context.Context) error {
		if err = p.repo.DeleteDocumentBlocksByTask(txCtx, documentId, taskId); err != nil {
			return err
		}
		if err = p.repo.InsertDocumentBlockBatch(txCtx, blocks); err != nil {
			return err
		}
		return nil
	}

	return p.repo.Do(ctx, txFn)
}

func (p *SaveStage) buildDocumentBlockEntities(ctx context.Context, documentId, taskId int64,
	candidates []*entity.DocumentBlock) ([]*entity.DocumentBlock, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}
	idsMap := make(map[int]int64, len(candidates))
	for _, candidate := range candidates {
		if candidate != nil && candidate.BlockNo > 0 {
			idsMap[candidate.BlockNo] = utils.GetSnowflakeNextID()
		}
	}
	blocks := make([]*entity.DocumentBlock, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.BlockNo <= 0 || candidate.BlockType == "" {
			continue
		}
		candidate.ID = idsMap[candidate.BlockNo]
		candidate.DocumentId = documentId
		candidate.TaskId = taskId
		candidate.ParentBlockId = idsMap[candidate.ParentBlockNo]

		if candidate.ImageContentBase64 != "" {
			content, err := decodeBase64(candidate.ImageContentBase64, fmt.Sprintf("图片 %d", candidate.BlockNo))
			if err != nil {
				return nil, err
			}
			fileName := utils.BlankToDefault(candidate.ImageFileName, fmt.Sprintf("image_%d.png", candidate.BlockNo))
			objectName, err := p.storage.UploadParseArtifact(ctx, documentId, taskId, fileName, "image/png", content)
			if err != nil {
				return nil, err
			}
			candidate.ImageObjectName = objectName
		}
		blocks = append(blocks, candidate)
	}
	return blocks, nil
}

// decodeBase64 Base64 解码，带错误包装
func decodeBase64(encoded, context string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%s Base64 解码失败: %w", context, err)
	}
	return decoded, nil
}

// executeStructurePersist 持久化结构节点，返回持久化后的节点列表
func (p *SaveStage) executeStructurePersist(ctx context.Context, documentId, taskId int64, analysisResult *Result) ([]*entity.StructureNode, error) {
	if documentId == 0 || taskId == 0 {
		return nil, nil
	}

	candidates := analysisResult.StructureNodes
	if len(candidates) == 0 {
		logx.Infof("结构节点阶段跳过，无候选节点，documentId=%d", documentId)
		return nil, nil
	}

	// 按文档ID清除旧的结构节点
	if err := p.repo.DeleteStructureNodeByDocumentId(ctx, documentId); err != nil {
		return nil, err
	}

	// 分配节点ID，并建立 nodeNo -> id 映射，便于父子/兄弟关系回写
	nodeIdMap := make(map[int]int64, len(candidates))
	for _, candidate := range candidates {
		if candidate != nil && candidate.NodeNo != 0 {
			nodeIdMap[candidate.NodeNo] = utils.GetSnowflakeNextID()
		}
	}

	// 回写节点ID、父节点ID、兄弟节点ID
	nodes := make([]*entity.StructureNode, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate != nil && candidate.NodeNo != 0 {
			candidate.ID = nodeIdMap[candidate.NodeNo]
			candidate.DocumentId = documentId
			candidate.ParseTaskId = taskId
			candidate.ParentNodeId = nodeIdMap[candidate.ParentNodeNo]
			candidate.PrevSiblingNodeId = nodeIdMap[candidate.PrevSiblingNodeNo]
			candidate.NextSiblingNodeId = nodeIdMap[candidate.NextSiblingNodeNo]
			nodes = append(nodes, candidate)
		}
	}

	// 批量插入
	if err := p.repo.InsertStructureNodeBatch(ctx, nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

// executeProfileGenerate 生成文档画像
func (p *SaveStage) executeProfileGenerate(ctx context.Context, documentId int64, analysisResult *Result, structureNodes []*entity.StructureNode) error {
	if documentId == 0 {
		return nil
	}
	_, err := p.gen.Generate(ctx, documentId, analysisResult, structureNodes)
	return err
}
