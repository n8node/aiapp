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
