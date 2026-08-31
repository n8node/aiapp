package filesniff

import (
	"bytes"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const MaxHead = 512

var allowedExt = map[string]struct{}{
	"pdf": {}, "doc": {}, "docx": {}, "xls": {}, "xlsx": {}, "ppt": {}, "pptx": {},
	"odt": {}, "ods": {}, "odp": {}, "rtf": {}, "txt": {}, "csv": {}, "md": {},
	"jpg": {}, "jpeg": {}, "png": {}, "gif": {}, "webp": {}, "tif": {}, "tiff": {},
	"zip": {}, "mp4": {}, "webm": {}, "mp3": {}, "wav": {}, "m4a": {},
}

func Extension(name string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
}

func AllowedExtension(name string) bool {
	_, ok := allowedExt[Extension(name)]
	return ok
}

func MaxSizeBytes(mimeType, name string) int64 {
	m := strings.ToLower(mimeType)
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(m, "image/") || isExt(n, "jpg", "jpeg", "png", "gif", "webp", "tif", "tiff"):
		return 25 << 20
	case strings.HasPrefix(m, "video/") || isExt(n, "mp4", "webm"):
		return 200 << 20
	case strings.HasPrefix(m, "audio/") || isExt(n, "mp3", "wav", "m4a"):
		return 50 << 20
	case strings.Contains(m, "zip") || isExt(n, "zip"):
		return 100 << 20
	default:
		return 50 << 20
	}
}

func MIMEMatchesExtension(mimeType, name string) bool {
	m := strings.ToLower(strings.TrimSpace(mimeType))
	ext := Extension(name)
	if m == "" || m == "application/octet-stream" {
		return true
	}
	switch ext {
	case "jpg", "jpeg", "png", "gif", "webp", "tif", "tiff":
		return strings.HasPrefix(m, "image/")
	case "mp4", "webm":
		return strings.HasPrefix(m, "video/")
	case "mp3", "wav", "m4a":
		return strings.HasPrefix(m, "audio/")
	case "zip":
		return strings.Contains(m, "zip") || strings.Contains(m, "compressed")
	case "pdf":
		return m == "application/pdf"
	case "doc":
		return strings.Contains(m, "msword")
	case "docx":
		return strings.Contains(m, "wordprocessingml")
	case "xls":
		return strings.Contains(m, "ms-excel") || strings.Contains(m, "spreadsheet")
	case "xlsx":
		return strings.Contains(m, "spreadsheetml")
	case "ppt":
		return strings.Contains(m, "ms-powerpoint")
	case "pptx":
		return strings.Contains(m, "presentationml")
	case "odt", "ods", "odp":
		return strings.Contains(m, "opendocument")
	case "rtf":
		return strings.Contains(m, "rtf") || strings.HasPrefix(m, "text/")
	case "txt", "csv", "md":
		return strings.HasPrefix(m, "text/") || m == "text/csv"
	default:
		return true
	}
}

func HeadMatches(name string, head []byte) bool {
	ext := Extension(name)
	if len(head) == 0 {
		return false
	}
	switch ext {
	case "pdf":
		return bytes.HasPrefix(head, []byte("%PDF"))
	case "png":
		return bytes.HasPrefix(head, []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})
	case "jpg", "jpeg":
		return bytes.HasPrefix(head, []byte{0xff, 0xd8, 0xff})
	case "gif":
		return bytes.HasPrefix(head, []byte("GIF87a")) || bytes.HasPrefix(head, []byte("GIF89a"))
	case "webp":
		return bytes.HasPrefix(head, []byte("RIFF")) && len(head) >= 12 && bytes.Equal(head[8:12], []byte("WEBP"))
	case "tif", "tiff":
		return bytes.HasPrefix(head, []byte{0x49, 0x49, 0x2a, 0x00}) || bytes.HasPrefix(head, []byte{0x4d, 0x4d, 0x00, 0x2a})
	case "zip", "docx", "xlsx", "pptx", "odt", "ods", "odp":
		return bytes.HasPrefix(head, []byte("PK"))
	case "doc", "xls", "ppt":
		return bytes.HasPrefix(head, []byte{0xd0, 0xcf, 0x11, 0xe0})
	case "rtf":
		return bytes.HasPrefix(bytes.TrimLeft(head, "\xef\xbb\xbf"), []byte(`{\rtf`))
	case "mp4", "webm", "m4a":
		return looksLikeMediaContainer(head)
	case "mp3":
		return bytes.HasPrefix(head, []byte("ID3")) || (len(head) >= 2 && head[0] == 0xff && head[1]&0xe0 == 0xe0)
	case "wav":
		return bytes.HasPrefix(head, []byte("RIFF"))
	case "txt", "csv", "md":
		return looksLikeText(head)
	default:
		return true
	}
}

func looksLikeMediaContainer(head []byte) bool {
	if bytes.Contains(head, []byte("ftyp")) || bytes.HasPrefix(head, []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		return true
	}
	return bytes.HasPrefix(head, []byte("RIFF"))
}

func looksLikeText(head []byte) bool {
	if bytes.Contains(head, []byte{0}) {
		return false
	}
	if utf8.Valid(head) {
		return true
	}
	printable := 0
	for _, b := range head {
		if b == '\n' || b == '\r' || b == '\t' || (b >= 32 && b < 127) {
			printable++
		}
	}
	return printable*10 >= len(head)*8
}

func isExt(name string, exts ...string) bool {
	ext := Extension(name)
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}
