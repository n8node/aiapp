package archivekit

import (
	"bytes"
	"errors"
	"io"

	"github.com/n8node/aiapp/internal/filesniff"
	"github.com/nwaples/rardecode/v2"
)

func walkRar(data []byte) ([]Entry, error) {
	rr, err := rardecode.NewReader(bytes.NewReader(data))
	if err != nil {
		msg := err.Error()
		if isZipEncrypted(err) {
			return nil, ErrEncrypted
		}
		if msg != "" {
			return nil, ErrNotArchive
		}
		return nil, ErrNotArchive
	}
	var total int64
	var out []Entry
	for {
		hdr, err := rr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if isZipEncrypted(err) {
				return nil, ErrEncrypted
			}
			return nil, ErrNotArchive
		}
		if hdr.IsDir {
			rel, err := SafeRelPath(hdr.Name)
			if err != nil || SkipNoise(rel) {
				continue
			}
			out = append(out, Entry{RelPath: rel, IsDir: true})
			continue
		}
		rel, err := SafeRelPath(hdr.Name)
		if err != nil || SkipNoise(rel) {
			_, _ = io.Copy(io.Discard, io.LimitReader(rr, MaxFileBytes+1))
			continue
		}
		if !filesniff.AllowedExtension(rel) {
			_, _ = io.Copy(io.Discard, io.LimitReader(rr, MaxFileBytes+1))
			continue
		}
		if hdr.UnPackedSize > MaxFileBytes {
			return nil, ErrTooLarge
		}
		total += hdr.UnPackedSize
		if total > MaxUncompressed {
			return nil, ErrTooLarge
		}
		if len(out) >= MaxFiles {
			return nil, ErrTooMany
		}
		body, err := io.ReadAll(io.LimitReader(rr, MaxFileBytes+1))
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
