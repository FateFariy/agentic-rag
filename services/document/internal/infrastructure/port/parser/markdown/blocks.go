package markdown

import (
	"strings"

	"github.com/swiftbit/know-agent/services/document/internal/domain/model/enum"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/shared"
)

// nodeSourceText 从 MarkdownSyntax 中提取节点的源文本
func nodeSourceText(syntax *shared.MarkdownSyntax, node *shared.MarkdownNode) string {
	if node.SourceSpan == nil {
		return ""
	}
	sourceBytes := []byte(syntax.SourceText)
	if node.SourceSpan.StartByte >= len(sourceBytes) || node.SourceSpan.EndByte > len(sourceBytes) {
		return ""
	}
	return string(sourceBytes[node.SourceSpan.StartByte:node.SourceSpan.EndByte])
}

// legacyBlockContent 根据节点类型生成块类型和文本内容
func legacyBlockContent(node *shared.MarkdownNode, rawText string) (string, string) {
	rawText = strings.TrimSpace(rawText)
	switch node.NodeType {
	case NodeHeading:
		return enum.BlockTypeTitle, strings.TrimSpace(node.Text)
	case NodeOrderedList, NodeUnorderedList:
		return enum.BlockTypeList, rawText
	case NodeCodeBlock:
		return enum.BlockTypeCode, rawText
	case NodeBlockquote:
		return enum.BlockTypeBlockquote, rawText
	case NodeThematicBreak:
		return enum.BlockTypeThematicBreak, rawText
	case NodeHTMLBlock:
		return enum.BlockTypeHTML, rawText
	default:
		return enum.BlockTypeText, rawText
	}
}
