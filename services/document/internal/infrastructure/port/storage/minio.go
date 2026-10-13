package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/document/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

type MinioStorage struct {
	*infra.MinioClient
}

var _ adapter.Storage = (*MinioStorage)(nil)

func NewMinioStorage(minioConf infra.MinioConf) *MinioStorage {
	return &MinioStorage{MinioClient: infra.NewMinioClient(minioConf)}
}

// UploadOriginalFile 上传原始文件
func (s *MinioStorage) UploadOriginalFile(ctx context.Context, documentId int64, fileName string, bytes []byte, contentType string) (*vo.StoredObjectInfo, error) {
	objectName := fmt.Sprintf("%s/%d/%d-%s", s.ObjectPrefix, documentId, time.Now().Unix(), fileName)
	if err := s.Upload(ctx, objectName, bytes, contentType); err != nil {
		return nil, err
	}
	return &vo.StoredObjectInfo{BucketName: s.BucketName, ObjectName: objectName, ObjectUrl: s.ObjectURL(objectName)}, nil
}

// UploadParsedText 上传解析后的文本
func (s *MinioStorage) UploadParsedText(ctx context.Context, documentId int64, parsedText string) (string, error) {
	objectName := fmt.Sprintf("%s/%d/%d.txt", s.ParsedTextPrefix, documentId, time.Now().Unix())
	if err := s.Upload(ctx, objectName, []byte(parsedText), "text/plain;charset=UTF-8"); err != nil {
		return "", err
	}
	return objectName, nil
}

// UploadParseArtifact 上传解析产物
func (s *MinioStorage) UploadParseArtifact(ctx context.Context, documentId, taskId int64, name, contentType string, content []byte) (string, error) {
	safeFileName := utils.BlankToDefault(name, "artifact.bin")
	safeFileName = strings.ReplaceAll(safeFileName, "/", "-")
	safeFileName = strings.ReplaceAll(safeFileName, "\\", "-")
	objectName := fmt.Sprintf("%s/%d/%d/%d-%s", s.ParseArtifactPrefix, documentId, taskId, time.Now().UnixMilli(), safeFileName)
	if err := s.Upload(ctx, objectName, content, contentType); err != nil {
		return "", err
	}
	return objectName, nil
}
