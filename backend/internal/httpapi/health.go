package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	ping  Pinger
	start time.Time
}

func NewHealthHandler(ping Pinger) *HealthHandler {
	return &HealthHandler{ping: ping, start: time.Now()}
}

type healthResponse struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Postgres      string `json:"postgres"`
	App           string `json:"app"`
}

func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:        "ok",
		Version:       version,
		UptimeSeconds: int64(time.Since(h.start).Seconds()),
		Postgres:      "unchecked",
		App:           "rigintel",
	})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	postgresStatus := "ok"
	code := http.StatusOK
	status := "ok"
	if err := h.ping.Ping(r.Context()); err != nil {
		postgresStatus = "error"
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, healthResponse{
		Status:        status,
		Version:       version,
		UptimeSeconds: int64(time.Since(h.start).Seconds()),
		Postgres:      postgresStatus,
		App:           "rigintel",
	})
}

type StatusHandler struct {
	publicAppURL string
}

func NewStatusHandler(publicAppURL string) *StatusHandler {
	return &StatusHandler{publicAppURL: publicAppURL}
}

func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{
			"service": "rigintel-api",
			"version": version,
			"app_url": h.publicAppURL,
		},
		"meta": map[string]string{
			"request_id": requestID(r),
		},
	})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
