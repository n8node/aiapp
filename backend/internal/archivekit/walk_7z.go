package archivekit

import (
	"bytes"
	"io"

	"github.com/bodgit/sevenzip"
	"github.com/n8node/aiapp/internal/filesniff"
)

func walk7z(data []byte) ([]Entry, error) {
	zr, err := sevenzip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrNotArchive
	}
	var total int64
	var out []Entry
	for _, f := range zr.File {
		name := f.Name
		if f.FileInfo().IsDir() {
			rel, err := SafeRelPath(name)
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
		if int64(f.UncompressedSize) > MaxFileBytes {
			return nil, ErrTooLarge
		}
		total += int64(f.UncompressedSize)
		if total > MaxUncompressed {
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
