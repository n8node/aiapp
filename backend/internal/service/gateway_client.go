package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type GatewayClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewGatewayClient(baseURL, token string) *GatewayClient {
	return &GatewayClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   token,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type gatewayEmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
}

func (g *GatewayClient) Ready(ctx context.Context) bool {
	if g == nil || g.baseURL == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+"/live", nil)
	if err != nil {
		return false
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	return resp.StatusCode == http.StatusOK
}

func (g *GatewayClient) Embed(ctx context.Context, texts []string, inputType string) ([][]float64, error) {
	if g == nil || g.baseURL == "" {
		return nil, ErrGatewayUnavailable
	}
	if len(texts) == 0 {
		return nil, nil
	}
	body, _ := json.Marshal(map[string]any{
		"model":      "embeddings",
		"input":      texts,
		"input_type": inputType,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, ErrGatewayUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, ErrGatewayUnavailable
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, ErrGatewayUnavailable
	}
	var out gatewayEmbedResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, ErrGatewayUnavailable
	}
	vecs := make([][]float64, 0, len(out.Data))
	for _, d := range out.Data {
		vecs = append(vecs, d.Embedding)
	}
	if len(vecs) != len(texts) {
		return nil, ErrGatewayUnavailable
	}
	return vecs, nil
}

type GatewayChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type gatewayChatResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (g *GatewayClient) ChatComplete(ctx context.Context, modelName string, messages []GatewayChatMessage, onDelta func(string) error) (string, error) {
	if g == nil || g.baseURL == "" {
		return "", ErrGatewayUnavailable
	}
	full, _ := g.chatOnce(ctx, modelName, messages, true, onDelta)
	if strings.TrimSpace(full) != "" {
		return full, nil
	}
	return g.chatOnce(ctx, modelName, messages, false, onDelta)
}

func (g *GatewayClient) chatOnce(ctx context.Context, modelName string, messages []GatewayChatMessage, stream bool, onDelta func(string) error) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":    modelName,
		"messages": messages,
		"stream":   stream,
	})
	if err != nil {
		return "", ErrGatewayUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", ErrGatewayUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	client := g.client
	if stream {
		client = &http.Client{Timeout: 3 * time.Minute}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", ErrGatewayUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return "", ErrGatewayUnavailable
	}
	if stream {
		return readChatSSE(resp.Body, onDelta)
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	var out gatewayChatResponse
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Choices) == 0 {
		return "", ErrGatewayUnavailable
	}
	text := out.Choices[0].Message.Content
	if text == "" {
		text = out.Choices[0].Delta.Content
	}
	if onDelta != nil && text != "" {
		_ = onDelta(text)
	}
	return text, nil
}

func readChatSSE(r io.Reader, onDelta func(string) error) (string, error) {
	limited := io.LimitReader(r, 16<<20)
	buf := make([]byte, 4096)
	var carry string
	var full strings.Builder
	for {
		n, err := limited.Read(buf)
		if n > 0 {
			carry += string(buf[:n])
			for {
				idx := strings.Index(carry, "\n")
				if idx < 0 {
					break
				}
				line := strings.TrimSpace(carry[:idx])
				carry = carry[idx+1:]
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if payload == "[DONE]" {
					return full.String(), nil
				}
				var chunk gatewayChatResponse
				if json.Unmarshal([]byte(payload), &chunk) != nil || len(chunk.Choices) == 0 {
					continue
				}
				piece := chunk.Choices[0].Delta.Content
				if piece == "" {
					piece = chunk.Choices[0].Message.Content
				}
				if piece == "" {
					continue
				}
				full.WriteString(piece)
				if onDelta != nil {
					if derr := onDelta(piece); derr != nil {
						return full.String(), derr
					}
				}
			}
		}
		if err == io.EOF {
			return full.String(), nil
		}
		if err != nil {
			if full.Len() > 0 {
				return full.String(), nil
			}
			return "", ErrGatewayUnavailable
		}
	}
}

func (g *GatewayClient) GenerateMedia(ctx context.Context, kind, modelName, prompt string) (string, error) {
	if g == nil || g.baseURL == "" {
		return "", ErrMediaUnavailable
	}
	path := "/v1/images/generations"
	if kind == modelMediaVideo {
		path = "/v1/videos/generations"
	}
	body, _ := json.Marshal(map[string]any{
		"model":  modelName,
		"prompt": prompt,
		"n":      1,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return "", ErrMediaUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", ErrMediaUnavailable
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
	if resp.StatusCode == http.StatusNotImplemented || resp.StatusCode == http.StatusNotFound {
		return "", ErrMediaUnavailable
	}
	if resp.StatusCode >= 400 {
		return "", ErrMediaUnavailable
	}
	var out struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &out) != nil || len(out.Data) == 0 {
		return "", ErrMediaUnavailable
	}
	if u := strings.TrimSpace(out.Data[0].URL); u != "" && (strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://")) {
		return u, nil
	}
	if b64 := strings.TrimSpace(out.Data[0].B64JSON); b64 != "" && len(b64) < 4<<20 {
		return "data:image/png;base64," + b64, nil
	}
	return "", ErrMediaUnavailable
}

const modelMediaVideo = "video"
