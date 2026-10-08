package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/duke-git/lancet/v2/condition"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/swiftbit/know-agent/pkg/common"
	"github.com/swiftbit/know-agent/pkg/utils"
)

var ErrDocumentStorageFailed = common.NewBizError(20011, "文件存储失败：%s")

// MinioConf MinIO 配置
type MinioConf struct {
	Addr                string `json:",default=127.0.0.1:9000"`
	AccessKeyID         string `json:",default=minioadmin"`
	SecretAccessKey     string `json:",default=minioadmin"`
	BucketName          string `json:",default=super-agent-document"`
	ObjectPrefix        string `json:",default=rag/document"`
	ParsedTextPrefix    string `json:",default=rag/parsed-text"`
	ParseArtifactPrefix string `json:",default=rag/parse-artifact"`
	UseSSL              bool   `json:",default=false"`
}

// MinioStore 通用对象存储网关：内嵌 MinIO 客户端，封装上传、下载、删除、
// 桶初始化与对象 URL 构建，供各服务的存储适配器复用，业务侧只需处理对象名拼接等业务规则
type MinioStore struct {
	*minio.Client
	MinioConf
}

// NewMinioStore 基于配置创建对象存储网关，并在内部完成 MinIO 客户端初始化
func NewMinioStore(c MinioConf) *MinioStore {
	client, err := minio.New(c.Addr, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKeyID, c.SecretAccessKey, ""),
		Secure: c.UseSSL,
	})
	if err != nil {
		panic(err)
	}
	return &MinioStore{Client: client, MinioConf: c}
}

// Upload 上传对象；contentType 为空时按二进制流处理
func (s *MinioStore) Upload(ctx context.Context, objectName string, data []byte, contentType string) error {
	if err := s.EnsureBucket(ctx); err != nil {
		return common.WrapErr(err, ErrDocumentStorageFailed.Code, "上传 MinIO 文件失败: "+err.Error())
	}

	_, err := s.PutObject(ctx, s.BucketName, objectName, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: condition.Ternary(contentType == "", "application/octet-stream", contentType),
	})
	if err != nil {
		return common.WrapErr(err, ErrDocumentStorageFailed.Code, "上传 MinIO 文件失败: "+err.Error())
	}
	return nil
}

// DownloadObject 下载对象内容
func (s *MinioStore) DownloadObject(ctx context.Context, objectName string) ([]byte, error) {
	object, err := s.GetObject(ctx, s.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, common.WrapErr(err, ErrDocumentStorageFailed.Code, "下载 MinIO 文件失败: "+err.Error())
	}
	defer func() {
		_ = object.Close()
	}()

	b, err := io.ReadAll(object)
	if err != nil {
		return nil, common.WrapErr(err, ErrDocumentStorageFailed.Code, "下载 MinIO 文件失败: "+err.Error())
	}
	return b, nil
}

// DownloadText 下载对象并按文本返回
func (s *MinioStore) DownloadText(ctx context.Context, objectName string) (string, error) {
	b, err := s.DownloadObject(ctx, objectName)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DeleteObjects 批量删除对象；空名称会被过滤去重，桶不存在时视为已删除
func (s *MinioStore) DeleteObjects(ctx context.Context, objectNameList []string) error {
	validObjectNameList := utils.FilterUniqueLimit(objectNameList, -1, func(name string) (string, bool) {
		trimmed := strings.TrimSpace(name)
		return trimmed, trimmed != ""
	})
	if len(validObjectNameList) == 0 {
		return nil
	}

	exists, err := s.BucketExists(ctx, s.BucketName)
	if err != nil {
		return common.WrapErr(err, ErrDocumentStorageFailed.Code, "删除 MinIO 文件失败: "+err.Error())
	}
	if !exists {
		return nil
	}

	for _, objectName := range validObjectNameList {
		if err = s.RemoveObject(ctx, s.BucketName, objectName, minio.RemoveObjectOptions{}); err != nil {
			return common.WrapErr(err, ErrDocumentStorageFailed.Code, "删除 MinIO 文件失败: "+err.Error())
		}
	}
	return nil
}

// EnsureBucket 确保桶存在，不存在则创建
func (s *MinioStore) EnsureBucket(ctx context.Context) error {
	exists, err := s.BucketExists(ctx, s.BucketName)
	if err != nil {
		return err
	}
	if !exists {
		return s.MakeBucket(ctx, s.BucketName, minio.MakeBucketOptions{})
	}
	return nil
}

// ObjectURL 构建对象可访问 URL
func (s *MinioStore) ObjectURL(objectName string) string {
	return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(s.Addr, "/"), s.BucketName, objectName)
}
