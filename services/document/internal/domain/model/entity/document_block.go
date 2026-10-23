package entity

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/enum"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

// DocumentBlock 文档块实体
type DocumentBlock struct {
	ID                 int64          `gorm:"column:id"`                                      // 主键ID
	DocumentId         int64          `gorm:"column:document_id"`                             // 文档ID
	TaskId             int64          `gorm:"column:task_id"`                                 // 任务ID
	BlockNo            int            `gorm:"column:block_no"`                                // 块序号
	BlockType          string         `gorm:"column:block_type"`                              // 块类型
	ParentBlockId      int64          `gorm:"column:parent_block_id"`                         // 父块ID
	SectionPath        string         `gorm:"column:section_path"`                            // 章节路径
	CanonicalPath      string         `gorm:"column:canonical_path"`                          // 规范路径
	PageNo             int            `gorm:"column:page_no"`                                 // 页码
	PageRange          string         `gorm:"column:page_range"`                              // 页码范围
	BboxJson           string         `gorm:"column:bbox_json"`                               // 边界框 JSON
	Text               string         `gorm:"column:text"`                                    // 文本内容
	ImageObjectName    string         `gorm:"column:image_object_name"`                       // 图片对象名
	ImageCaption       string         `gorm:"column:image_caption"`                           // 图片说明
	Metadata           map[string]any `gorm:"column:metadata_json;type:json;serializer:json"` // 元数据
	ParentBlockNo      int            `gorm:"-"`                                              // 父块编号
	BoundingBoxJSON    string         `gorm:"-"`                                              // 边界框JSON
	ImageFileName      string         `gorm:"-"`                                              // 图片文件名
	ImageContentBase64 string         `gorm:"-"`                                              // 图片Base64内容
}

// HasBlockContent 检查块是否有内容
func (b *DocumentBlock) HasBlockContent() bool {
	return b != nil && strings.TrimSpace(b.RenderBlockContent()) != ""
}

// RenderBlockContent 渲染块内容
func (b *DocumentBlock) RenderBlockContent() string {
	if b == nil {
		return ""
	}
	text := utils.FirstNonBlank(b.Text, b.ImageCaption)
	if utils.IsBlank(text) {
		return ""
	}

	// 根据块类型处理内容
	switch strings.ToUpper(utils.Trim(b.BlockType)) {
	case "TITLE":
		if !strings.HasPrefix(utils.Trim(text), "#") {
			return "# " + text
		}
	case "IMAGE", "FIGURE":
		if utils.IsNotBlank(b.ImageCaption) {
			return "[IMAGE]\n" + b.ImageCaption
		}
	}

	return text
}

// HeadingLevel 获取标题层级
func (b *DocumentBlock) HeadingLevel() int {
	if b == nil {
		return 1
	}
	if metadata, ok := b.Metadata["headingLevel"]; ok {
		if level, valid := metadata.(int); valid && level >= 1 && level <= 6 {
			return level
		}
	}
	// 从文本推断层级
	text := strings.TrimSpace(b.Text)
	if strings.HasPrefix(text, "# ") {
		return 1
	}
	if strings.HasPrefix(text, "## ") {
		return 2
	}
	if strings.HasPrefix(text, "### ") {
		return 3
	}
	if strings.HasPrefix(text, "#### ") {
		return 4
	}
	if strings.HasPrefix(text, "##### ") {
		return 5
	}
	if strings.HasPrefix(text, "###### ") {
		return 6
	}
	return 1
}

// BuildContentText 构建内容文本（包含文本、图片说明）
func (b *DocumentBlock) BuildContentText() string {
	if b == nil {
		return ""
	}
	var parts []string
	if b.Text != "" {
		parts = append(parts, b.Text)
	}
	if b.ImageCaption != "" {
		parts = append(parts, fmt.Sprintf("[IMAGE] %s", b.ImageCaption))
	}
	return strings.Join(parts, "\n\n")
}

// headingCodeRegex 匹配数字编号（如 1、1.2、1.2.3）
var headingCodeRegex = regexp.MustCompile(`^(\d+(\.\d+)*)`)

// ExtractHeadingCode 提取标题编码（如 "1.2.3"），用于文档结构层级识别
func (b *DocumentBlock) ExtractHeadingCode() string {
	if b == nil {
		return ""
	}
	if match := headingCodeRegex.FindString(b.Text); match != "" {
		return match
	}
	return ""
}

// IsTitleBlock 判断块是否为标题块
func (b *DocumentBlock) IsTitleBlock() bool {
	return b != nil && utils.EqualsIgnoreCase(b.BlockType, "TITLE")
}

// ResolveTitle 解析块标题
func (b *DocumentBlock) ResolveTitle() string {
	if b == nil {
		return ""
	}
	blocks := DocumentBlocks{b}
	return blocks.ResolveTitle(b.SectionPath)
}

// ResolveChunkType 解析块类型
func (b *DocumentBlock) ResolveChunkType() string {
	if b == nil {
		return ""
	}
	blocks := DocumentBlocks{b}
	return blocks.ResolveChunkType()
}

// Ids 获取块的 ID 列表
func (b *DocumentBlock) Ids() string {
	if b == nil {
		return ""
	}
	blocks := DocumentBlocks{b}
	return blocks.Ids()
}

// CloneWithText 克隆块并设置文本内容
func (b *DocumentBlock) CloneWithText(text string) *DocumentBlock {
	if b == nil {
		return nil
	}
	clone := *b
	clone.Text = text
	return &clone
}

var nonAllowedCharsRegex = regexp.MustCompile(`[^0-9a-zA-Z\p{Han}]+`)

// BuildCanonicalPath 生成规范路径，格式：章节路径-块编号
func (b *DocumentBlock) BuildCanonicalPath() string {
	section := b.SectionPath
	// 正则：匹配非字母、数字、汉字的字符（连续多个替换为一个 "-"）
	section = nonAllowedCharsRegex.ReplaceAllString(section, "-")
	section = strings.Trim(section, "-")
	if section == "" {
		section = "root"
	}
	return fmt.Sprintf("/%s/%d", section, b.BlockNo)
}

// ResumeRawText 从原始字节中恢复块的文本内容
func (b *DocumentBlock) ResumeRawText(rawBytes []byte) {
	if b == nil || len(rawBytes) == 0 {
		return
	}
	var sourceSpan vo.SourceSpan
	marshal, _ := json.Marshal(b.Metadata["sourceSpan"])
	_ = json.Unmarshal(marshal, &sourceSpan)
	b.Text = string(rawBytes[sourceSpan.StartByte:sourceSpan.EndByte])
}

type DocumentBlocks []*DocumentBlock

// FirstPageNo 获取块列表第一个块的页码
func (b DocumentBlocks) FirstPageNo() int {
	for _, block := range b {
		if block == nil {
			continue
		}
		pageNo := block.PageNo
		if pageNo > 0 {
			return pageNo
		}
	}
	return 0
}

// PageRange 获取块列表的页码范围
func (b DocumentBlocks) PageRange() string {
	// 提取有效页码并去重
	pageSet := make(map[int]bool, len(b))
	pages := make([]int, 0, len(b))
	for _, block := range b {
		if block == nil {
			continue
		}
		if pn := block.PageNo; pn > 0 && !pageSet[pn] {
			pageSet[pn] = true
			pages = append(pages, pn)
		}
	}

	// 若存在有效页码，则排序并计算范围
	if len(pages) > 0 {
		slices.Sort(pages)

		first, last := pages[0], pages[len(pages)-1]
		if first == last {
			return strconv.Itoa(first)
		}
		return fmt.Sprintf("%d-%d", first, last)
	}

	// 回退逻辑：使用块自带的 pageRange
	for _, block := range b {
		if block == nil {
			continue
		}
		if r := strings.TrimSpace(block.PageRange); r != "" {
			return r
		}
	}
	return ""
}

// Ids 获取块列表的 ID 列表
func (b DocumentBlocks) Ids() string {
	seen := make(map[int64]struct{})
	var builder strings.Builder
	count := 0
	builder.WriteString("[")
	for _, block := range b {
		if block == nil {
			continue
		}
		id := block.ID
		if _, exists := seen[id]; id > 0 && !exists {
			seen[id] = struct{}{}
			if count > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(strconv.FormatInt(id, 10))
			count++
		}
	}
	builder.WriteString("]")
	return builder.String()
}

// FirstBlankCanonicalPath 获取块列表第一个非空的规范路径
func (b DocumentBlocks) FirstBlankCanonicalPath() string {
	for _, block := range b {
		if block == nil {
			continue
		}
		space := strings.TrimSpace(block.CanonicalPath)
		if space != "" {
			return space
		}
	}
	return ""
}

// ResolveTitle 解析块列表的标题

func (b DocumentBlocks) ResolveTitle(sectionPath string) string {
	normalizeTitle := func(title string) string {
		normalized := strings.TrimSpace(title)
		return strings.TrimLeft(normalized, "#")
	}
	for _, block := range b {
		if block != nil && block.IsTitleBlock() && strutil.IsNotBlank(block.Text) {
			return normalizeTitle(block.Text)
		}
	}

	sectionPath = strings.TrimSpace(sectionPath)
	if sectionPath == "" {
		return ""
	}

	re := regexp.MustCompile(`[>/|]`)
	parts := re.Split(sectionPath, -1)
	for i := len(parts) - 1; i >= 0; i-- {
		segment := strings.TrimSpace(parts[i])
		if segment != "" {
			return normalizeTitle(segment)
		}
	}
	return normalizeTitle(sectionPath)
}

// JoinBlockTexts 将块列表的文本内容用 "\n\n" 连接成一个字符串

func (b DocumentBlocks) JoinBlockTexts() string {
	var builder strings.Builder
	first := true
	for _, block := range b {
		if block == nil {
			continue
		}
		content := strings.TrimSpace(block.RenderBlockContent())
		if content == "" {
			continue
		}

		if !first {
			builder.WriteString("\n\n")
		}
		builder.WriteString(content)
		first = false
	}

	return builder.String()
}

// CleanupAndSort 对块列表进行清理和排序
func (b DocumentBlocks) CleanupAndSort() DocumentBlocks {
	if len(b) == 0 {
		return nil
	}
	predicate := func(item *DocumentBlock) bool {
		return item != nil && item.HasBlockContent()
	}
	result := make(DocumentBlocks, 0, len(b))
	for _, block := range b {
		if predicate(block) {
			result = append(result, block)
		}
	}
	less := func(a, b *DocumentBlock) int {
		if a.BlockNo == 0 {
			return 1
		} else if a.BlockNo != b.BlockNo {
			return a.BlockNo - b.BlockNo
		}
		return int(a.ID - b.ID)
	}
	slices.SortFunc(result, less)
	return result
}

// ToMap 将块列表转换为 ID 到块的映射
func (b DocumentBlocks) ToMap() map[int64]*DocumentBlock {
	result := make(map[int64]*DocumentBlock)
	for _, block := range b {
		if block != nil {
			result[block.ID] = block
		}
	}
	return result
}

// ResolveChunkType 解析块列表的块类型
func (b DocumentBlocks) ResolveChunkType() string {
	seen := make(map[string]struct{})
	var blockTypes []string

	for _, block := range b {
		if block == nil {
			continue
		}
		blockType := strutil.Trim(block.BlockType)
		if blockType == "" {
			continue
		}
		upperType := strings.ToUpper(blockType)
		if _, exists := seen[upperType]; !exists {
			seen[upperType] = struct{}{}
			blockTypes = append(blockTypes, upperType)
		}
	}

	if len(blockTypes) == 0 {
		return "TEXT"
	}
	if len(blockTypes) == 1 {
		return blockTypes[0]
	}
	return "MIXED"
}

// FirstBlankSectionPath 获取块列表第一个非空的章节路径
func (b DocumentBlocks) FirstBlankSectionPath() string {
	for _, block := range b {
		if block == nil {
			continue
		}
		trimmed := strings.TrimSpace(block.SectionPath)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// Normalize 对块列表进行规范化处理
func (b DocumentBlocks) Normalize() DocumentBlocks {
	normalized := make(DocumentBlocks, 0, len(b))
	currentSection := ""

	for _, block := range b {
		text := utils.CleanupSpace(block.Text)
		text = utils.ClipHead(text, 200)
		// 过滤掉无文本、无图片说明的空块
		if text == "" && block.ImageCaption == "" {
			continue
		}

		// 重新编号（从1开始）
		block.BlockNo = len(normalized) + 1
		block.Text = text

		// 若当前块是标题，更新当前章节信息
		if block.BlockType == enum.BlockTypeTitle {
			currentSection = text
		}

		// 补全 section_path（若为空则使用当前章节路径）
		if block.SectionPath == "" {
			block.SectionPath = currentSection
		}

		// 补全 canonical_path（若为空则自动生成）
		if block.CanonicalPath == "" {
			block.CanonicalPath = block.BuildCanonicalPath()
		}

		normalized = append(normalized, block)
	}
	return normalized
}

// ExtractParsedText 从块列表提取解析后的文本
func (b DocumentBlocks) ExtractParsedText() string {
	if len(b) == 0 {
		return ""
	}
	var parts []string
	for _, block := range b {
		if block == nil {
			continue
		}
		text := strings.TrimSpace(block.Text)
		if text != "" {
			parts = append(parts, text)
		} else if block.ImageCaption != "" {
			parts = append(parts, block.ImageCaption)
		}
	}
	return strings.Join(parts, "\n\n")
}

// CalcStats 统计块级指标：标题数、段落数、最大段落长度（字符数）
func (b DocumentBlocks) CalcStats() (headingCount, paragraphCount, maxParagraphLen int) {
	for _, block := range b {
		if block == nil {
			continue
		}
		switch block.BlockType {
		case enum.BlockTypeTitle:
			headingCount++
		default:
			textLen := utils.Len(strings.TrimSpace(block.Text))
			if block.ImageCaption != "" {
				textLen = max(textLen, utils.Len(block.ImageCaption))
			}
			if textLen > 0 {
				paragraphCount++
				maxParagraphLen = max(maxParagraphLen, textLen)
			}
		}
	}
	return
}

// CalcStructureLevel 计算结构等级（基于标题数和块总数）
func (b DocumentBlocks) CalcStructureLevel() int {
	headingCount, _, _ := b.CalcStats()
	blockCount := len(b)
	if headingCount >= 10 || blockCount >= 30 {
		return enum.StructureLevelHigh
	}
	if headingCount >= 3 || blockCount >= 10 {
		return enum.StructureLevelMedium
	}
	if headingCount >= 1 || blockCount >= 5 {
		return enum.StructureLevelLow
	}
	return 0
}

// CalcContentQualityLevel 计算内容质量等级（基于字符数、块数、最大段落长度）
func (b DocumentBlocks) CalcContentQualityLevel() int {
	charCount, _, maxParagraphLen := b.CalcStats()
	blockCount := len(b)
	if charCount >= 5000 && blockCount >= 10 && maxParagraphLen >= 100 {
		return enum.ContentQualityLevelHigh
	}
	if charCount >= 1000 && blockCount >= 3 {
		return enum.ContentQualityLevelMedium
	}
	if charCount >= 100 {
		return enum.ContentQualityLevelLow
	}
	return 0
}
