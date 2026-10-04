package markdown

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestGoldmarkParser_ParseFromFile 读取 Markdown 文件并打印解析结果。
//
// 默认读取同包 testdata/sample.md；可通过环境变量 MARKDOWN_TEST_FILE
// 指定任意 Markdown 文件，例如：
//
//	MARKDOWN_TEST_FILE=../../../../eval/dataset/80000_docs/documents_dup_part_1_part_1.txt go test -run TestGoldmarkParser_ParseFromFile -v
func TestGoldmarkParser_ParseFromFile(t *testing.T) {
	path := os.Getenv("MARKDOWN_TEST_FILE")
	if path == "" {
		//path = "E:\\GoCode\\ragent-convert\\know-agent\\eval\\dataset\\80000_docs\\documents_dup_part_1_part_1.md"
		path = "E:\\GoCode\\ragent-convert\\know-agent\\eval\\dataset\\documents_539\\无人驾驶航空器飞行管理暂行条例.md"
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取文件失败 %s: %v", path, err)
	}

	parser := NewGoldmarkParser()
	blocks, err := parser.Parse(context.Background(), raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	t.Logf("文件: %s", path)
	t.Logf("字节长度: %d", len(raw))
	t.Logf("解析得到 DocumentBlock 数量: %d", len(blocks))
	t.Log(strings.Repeat("=", 60))

	if len(blocks) == 0 {
		t.Fatal("期望至少解析出一个 DocumentBlock")
	}

	for _, b := range blocks {
		t.Logf("【块 #%d】类型=%s", b.BlockNo, b.BlockType)
		if b.Text != "" {
			t.Logf("  文本: %s", singleLine(b.Text))
		}
		if len(b.TableRows) > 0 {
			t.Logf("  表格行数: %d", len(b.TableRows))
			for i, row := range b.TableRows {
				t.Logf("    行%d: %v", i, row)
			}
		}
		if b.TableHTML != "" {
			t.Logf("  表格HTML: %s", singleLine(b.TableHTML))
		}
		if len(b.Metadata) > 0 {
			mb, _ := json.MarshalIndent(b.Metadata, "    ", "  ")
			t.Logf("  元数据: %s", mb)
		}
		t.Log(strings.Repeat("-", 60))
	}

	// 将完整解析结果以 JSON 形式输出，便于核查细节
	dump, _ := json.MarshalIndent(blocks, "", "  ")
	t.Logf("完整解析结果(JSON):\n%s", dump)
}

// singleLine 将多行文本折叠为单行，便于在测试日志中查看。
func singleLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
