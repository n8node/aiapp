package archivekit

import (
	"fmt"
	"path"
	"strings"
	"unicode/utf8"
)

const (
	MaxFiles        = 400
	MaxUncompressed = 500 << 20
	MaxFileBytes    = 100 << 20
	MaxDepth        = 8
)

func ArchiveBaseName(name string) string {
	n := strings.TrimSpace(name)
	lower := strings.ToLower(n)
	for _, suf := range []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tgz", ".tbz2", ".tbz"} {
		if strings.HasSuffix(lower, suf) {
			base := strings.TrimSpace(n[:len(n)-len(suf)])
			if base == "" {
				return "архив"
			}
			return base
		}
	}
	if i := strings.LastIndex(n, "."); i > 0 {
		base := strings.TrimSpace(n[:i])
		if base != "" {
			return base
		}
	}
	if n == "" {
		return "архив"
	}
	return n
}

func NumberedName(base string, n int) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "архив"
	}
	if n <= 0 {
		return base
	}
	return fmt.Sprintf("%s (%d)", base, n)
}

func NextAvailableName(base string, taken func(string) bool) (string, error) {
	if !taken(base) {
		return base, nil
	}
	for i := 1; i <= 99; i++ {
		cand := NumberedName(base, i)
		if utf8.RuneCountInString(cand) > 255 {
			return "", errName
		}
		if !taken(cand) {
			return cand, nil
		}
	}
	return "", errName
}

func SafeRelPath(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, "\\", "/")
	if s == "" || strings.Contains(s, "\x00") {
		return "", errPath
	}
	for _, p := range strings.Split(s, "/") {
		if p == ".." {
			return "", errPath
		}
	}
	cleaned := path.Clean("/" + s)
	if cleaned == "/" || cleaned == "." || cleaned == "/." {
		return "", errPath
	}
	if !strings.HasPrefix(cleaned, "/") || strings.Contains(cleaned, "..") {
		return "", errPath
	}
	rel := strings.TrimPrefix(cleaned, "/")
	parts := strings.Split(rel, "/")
	if len(parts) > MaxDepth {
		return "", errPath
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return "", errPath
		}
		if utf8.RuneCountInString(p) > 255 {
			return "", errPath
		}
		if strings.ContainsAny(p, "/\\\x00") {
			return "", errPath
		}
	}
	return rel, nil
}

func SkipNoise(rel string) bool {
	lower := strings.ToLower(rel)
	if strings.HasPrefix(lower, "__macosx/") || lower == "__macosx" {
		return true
	}
	base := strings.ToLower(path.Base(rel))
	switch base {
	case ".ds_store", "thumbs.db", "desktop.ini":
		return true
	}
	return strings.HasPrefix(base, "._")
}
