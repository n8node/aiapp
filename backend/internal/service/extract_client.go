package service

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/n8node/aiapp/internal/model"
)

type HTTPExtractor struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewHTTPExtractor(baseURL, token string) *HTTPExtractor {
	return &HTTPExtractor{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   token,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (e *HTTPExtractor) Extract(ctx context.Context, path, originalName, mimeType string) (*model.ExtractResult, error) {
	if e == nil || e.baseURL == "" {
		return nil, ErrExtractUnavailable
	}
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		var pipeErr error
		defer func() {
			_ = mw.Close()
			_ = pw.CloseWithError(pipeErr)
		}()
		part, err := mw.CreateFormFile("file", originalName)
		if err != nil {
			pipeErr = err
			return
		}
		f, err := os.Open(path)
		if err != nil {
			pipeErr = err
			return
		}
		defer f.Close()
		if _, err := io.Copy(part, f); err != nil {
			pipeErr = err
			return
		}
		_ = mw.WriteField("original_name", originalName)
		_ = mw.WriteField("mime_type", mimeType)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/internal/v1/extract", pr)
	if err != nil {
		return nil, ErrExtractUnavailable
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if e.token != "" {
		req.Header.Set("Authorization", "Bearer "+e.token)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrExtractUnavailable
	}
	defer resp.Body.Close()
	body := io.LimitReader(resp.Body, 1<<20)
	if resp.StatusCode == http.StatusRequestEntityTooLarge {
		_, _ = io.Copy(io.Discard, body)
		return nil, ErrObjectTooLarge
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		_, _ = io.Copy(io.Discard, body)
		return nil, ErrExtractFailed
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, body)
		return nil, ErrExtractUnavailable
	}
	var out model.ExtractResult
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		return nil, ErrExtractUnavailable
	}
	if out.Warnings == nil {
		out.Warnings = []string{}
	}
	return &out, nil
}
