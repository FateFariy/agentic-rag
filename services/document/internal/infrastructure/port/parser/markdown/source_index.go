package markdown

import (
	"errors"
	"sort"

	"github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

type sourceIndex struct {
	sourceText  string
	lineStarts  []int
	byteOffsets []int
}

func newSourceIndex(sourceText string) *sourceIndex {
	si := &sourceIndex{sourceText: sourceText}
	si.buildLineStarts()
	si.buildByteOffsets()
	return si
}

func (s *sourceIndex) buildLineStarts() {
	s.lineStarts = []int{0}
	runes := []rune(s.sourceText)
	n := len(runes)
	i := 0
	for i < n {
		r := runes[i]
		if r == '\n' {
			i++
			s.lineStarts = append(s.lineStarts, i)
		} else if r == '\r' {
			i++
			if i < n && runes[i] == '\n' {
				i++ // 吞掉 \r\n 中紧随的 \n
			}
			s.lineStarts = append(s.lineStarts, i)
		} else {
			i++
		}
	}
}

func (s *sourceIndex) buildByteOffsets() {
	s.byteOffsets = []int{0}
	total := 0
	for _, r := range s.sourceText {
		total += len(string(r))
		s.byteOffsets = append(s.byteOffsets, total)
	}
}

func (s *sourceIndex) CharacterSpan(lineMap []int) (int, int, error) {
	if len(lineMap) != 2 {
		return 0, 0, errors.New("invalid line map: must have 2 elements")
	}
	startLine, endLine := lineMap[0], lineMap[1]
	if startLine < 0 || endLine < startLine {
		return 0, 0, errors.New("invalid markdown token line map")
	}
	start := s.lineStarts[startLine]
	if startLine >= len(s.lineStarts) {
		start = len([]rune(s.sourceText))
	}
	end := len([]rune(s.sourceText))
	if endLine < len(s.lineStarts) {
		end = s.lineStarts[endLine]
	}
	return start, end, nil
}

func (s *sourceIndex) SourceSpan(charSpan [2]int) *vo.SourceSpan {
	start, end := charSpan[0], charSpan[1]
	startLine, startCol := s.LineColumn(start)
	endLine, endCol := s.LineColumn(end)
	return &vo.SourceSpan{
		StartByte:   s.byteOffsets[start],
		EndByte:     s.byteOffsets[end],
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     endLine,
		EndColumn:   endCol,
	}
}

func (s *sourceIndex) LineColumn(charOffset int) (int, int) {
	idx := sort.Search(len(s.lineStarts), func(i int) bool {
		return s.lineStarts[i] > charOffset
	})
	lineIndex := idx - 1
	if lineIndex < 0 {
		lineIndex = 0
	}
	return lineIndex + 1, charOffset - s.lineStarts[lineIndex] + 1
}
