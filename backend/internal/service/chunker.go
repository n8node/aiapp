package service

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	defaultChunkSize    = 500
	defaultChunkOverlap = 50
	maxChunksPerFile    = 200
)

type TextChunk struct {
	Text      string
	Index     int
	StartChar int
	EndChar   int
}

func chunkText(text string, size, overlap int) []TextChunk {
	if size <= 0 {
		size = defaultChunkSize
	}
	if overlap < 0 || overlap >= size {
		overlap = defaultChunkOverlap
		if overlap >= size {
			overlap = size / 10
		}
	}
	cleaned := strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	for strings.Contains(cleaned, "\n\n\n") {
		cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")
	}
	if cleaned == "" {
		return nil
	}
	runes := []rune(cleaned)
	if len(runes) <= size {
		return []TextChunk{{Text: cleaned, Index: 0, StartChar: 0, EndChar: len(runes)}}
	}
	var out []TextChunk
	start := 0
	idx := 0
	for start < len(runes) && idx < maxChunksPerFile {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		} else {
			end = breakPoint(runes, start, end)
		}
		piece := strings.TrimSpace(string(runes[start:end]))
		if piece != "" {
			out = append(out, TextChunk{Text: piece, Index: idx, StartChar: start, EndChar: end})
			idx++
		}
		if end >= len(runes) {
			break
		}
		next := end - overlap
		if next <= start {
			next = end
		}
		start = next
	}
	return out
}

func breakPoint(runes []rune, start, end int) int {
	windowStart := end - 80
	if windowStart < start {
		windowStart = start
	}
	window := runes[windowStart:end]
	for i := len(window) - 1; i >= 1; i-- {
		if window[i] == '\n' && window[i-1] == '\n' {
			return windowStart + i + 1
		}
	}
	for i := len(window) - 1; i >= 1; i-- {
		if isSentenceEnd(window[i-1]) && unicode.IsSpace(window[i]) {
			return windowStart + i + 1
		}
	}
	for i := len(window) - 1; i >= 0; i-- {
		if unicode.IsSpace(window[i]) {
			return windowStart + i + 1
		}
	}
	return end
}

func isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?' || r == '…'
}

func clipName(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}
