package archivekit

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"io"
	"strings"

	"github.com/n8node/aiapp/internal/filesniff"
)

var (
	ErrNotArchive = errors.New("not an archive")
	ErrTooLarge   = errors.New("archive too large")
	ErrTooMany    = errors.New("archive too many files")
	ErrEncrypted  = errors.New("archive encrypted")
	errPath       = errors.New("unsafe path")
	errName       = errors.New("name taken")
)

type Entry struct {
	RelPath string
	IsDir   bool
	Size    int64
	Data    []byte
}

func IsArchiveName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if strings.HasSuffix(n, ".tar.gz") || strings.HasSuffix(n, ".tar.bz2") || strings.HasSuffix(n, ".tar.xz") {
		return true
	}
	switch filesniff.Extension(name) {
	case "zip", "rar", "7z", "tar", "tgz", "tbz", "tbz2", "gz":
		return true
	}
	return false
}

func Walk(name string, data []byte) ([]Entry, error) {
	if len(data) == 0 {
		return nil, ErrNotArchive
	}
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".tar.gz") || strings.HasSuffix(n, ".tgz"):
		return walkGzipTar(data)
	case strings.HasSuffix(n, ".tar.bz2") || strings.HasSuffix(n, ".tbz") || strings.HasSuffix(n, ".tbz2"):
		return walkBzipTar(data)
	case strings.HasSuffix(n, ".tar"):
		return walkTar(bytes.NewReader(data))
	case strings.HasSuffix(n, ".gz") && !strings.HasSuffix(n, ".tar.gz"):
		return walkGzipFile(name, data)
	case strings.HasSuffix(n, ".zip") || bytes.HasPrefix(data, []byte("PK")):
		if filesniff.Extension(name) != "zip" && !strings.HasSuffix(n, ".zip") {
			return nil, ErrNotArchive
		}
		return walkZip(data)
	case strings.HasSuffix(n, ".7z") || bytes.HasPrefix(data, []byte("7z\xbc\xaf\x27\x1c")):
		return walk7z(data)
	case strings.HasSuffix(n, ".rar") || bytes.HasPrefix(data, []byte("Rar!")):
		return walkRar(data)
	default:
		if filesniff.Extension(name) == "zip" {
			return walkZip(data)
		}
		return nil, ErrNotArchive
	}
}

func walkZip(data []byte) ([]Entry, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrNotArchive
	}
	var total uint64
	out := make([]Entry, 0, len(zr.File))
	for _, f := range zr.File {
		if f.Flags&1 != 0 {
			return nil, ErrEncrypted
		}
		name := f.Name
		if strings.HasSuffix(name, "/") || f.FileInfo().IsDir() {
			rel, err := SafeRelPath(strings.TrimSuffix(name, "/"))
			if err != nil || SkipNoise(rel) {
				continue
			}
			out = append(out, Entry{RelPath: rel, IsDir: true})
			continue
		}
		rel, err := SafeRelPath(name)
		if err != nil || SkipNoise(rel) {
			continue
		}
		if !filesniff.AllowedExtension(rel) {
			continue
		}
		if f.UncompressedSize64 > uint64(MaxFileBytes) {
			return nil, ErrTooLarge
		}
		total += f.UncompressedSize64
		if total > uint64(MaxUncompressed) {
			return nil, ErrTooLarge
		}
		if len(out) >= MaxFiles {
			return nil, ErrTooMany
		}
		rc, err := f.Open()
		if err != nil {
			if isZipEncrypted(err) {
				return nil, ErrEncrypted
			}
			continue
		}
		body, err := io.ReadAll(io.LimitReader(rc, MaxFileBytes+1))
		_ = rc.Close()
		if err != nil || int64(len(body)) > MaxFileBytes {
			return nil, ErrTooLarge
		}
		if !filesniff.HeadMatches(rel, headOf(body)) {
			continue
		}
		out = append(out, Entry{RelPath: rel, Size: int64(len(body)), Data: body})
	}
	return out, nil
}

func walkTar(r io.Reader) ([]Entry, error) {
	tr := tar.NewReader(r)
	var total int64
	var out []Entry
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrNotArchive
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			rel, err := SafeRelPath(hdr.Name)
			if err != nil || SkipNoise(rel) {
				continue
			}
			out = append(out, Entry{RelPath: rel, IsDir: true})
		case tar.TypeReg, tar.TypeRegA:
			rel, err := SafeRelPath(hdr.Name)
			if err != nil || SkipNoise(rel) {
				continue
			}
			if !filesniff.AllowedExtension(rel) {
				_, _ = io.Copy(io.Discard, io.LimitReader(tr, MaxFileBytes+1))
				continue
			}
			if hdr.Size > MaxFileBytes {
				return nil, ErrTooLarge
			}
			total += hdr.Size
			if total > MaxUncompressed {
				return nil, ErrTooLarge
			}
			if len(out) >= MaxFiles {
				return nil, ErrTooMany
			}
			body, err := io.ReadAll(io.LimitReader(tr, MaxFileBytes+1))
			if err != nil || int64(len(body)) > MaxFileBytes {
				return nil, ErrTooLarge
			}
			if !filesniff.HeadMatches(rel, headOf(body)) {
				continue
			}
			out = append(out, Entry{RelPath: rel, Size: int64(len(body)), Data: body})
		default:
			continue
		}
	}
	return out, nil
}

func walkGzipTar(data []byte) ([]Entry, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, ErrNotArchive
	}
	defer gr.Close()
	return walkTar(gr)
}

func walkBzipTar(data []byte) ([]Entry, error) {
	return walkTar(bzip2.NewReader(bytes.NewReader(data)))
}

func walkGzipFile(name string, data []byte) ([]Entry, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, ErrNotArchive
	}
	defer gr.Close()
	body, err := io.ReadAll(io.LimitReader(gr, MaxFileBytes+1))
	if err != nil || int64(len(body)) > MaxFileBytes {
		return nil, ErrTooLarge
	}
	inner := ArchiveBaseName(name)
	if inner == "" || inner == "архив" {
		inner = "file"
	}
	if !filesniff.AllowedExtension(inner) {
		return nil, ErrNotArchive
	}
	if !filesniff.HeadMatches(inner, headOf(body)) {
		return nil, ErrNotArchive
	}
	return []Entry{{RelPath: inner, Size: int64(len(body)), Data: body}}, nil
}

func headOf(body []byte) []byte {
	if len(body) > filesniff.MaxHead {
		return body[:filesniff.MaxHead]
	}
	return body
}

func isZipEncrypted(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "password") || strings.Contains(msg, "encrypted")
}
