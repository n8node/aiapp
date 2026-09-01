package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var ErrStudioAuth = errors.New("studio auth failed")

const studioLoginUser = "unsloth"

type StudioAuthService struct {
	base     string
	password string
	http     *http.Client
	auth     *AuthService
}

func NewStudioAuthService(base, password string, auth *AuthService) *StudioAuthService {
	return &StudioAuthService{
		base:     strings.TrimRight(strings.TrimSpace(base), "/"),
		password: password,
		http:     &http.Client{Timeout: 8 * time.Second},
		auth:     auth,
	}
}

type StudioSession struct {
	AccessToken        string `json:"access_token"`
	RefreshToken       string `json:"refresh_token"`
	MustChangePassword bool   `json:"must_change_password"`
}

func (s *StudioAuthService) Session(ctx context.Context, userID string) (*StudioSession, error) {
	user, err := s.auth.RequireUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.CanAccessStudio() {
		return nil, ErrForbidden
	}
	if s.base == "" || s.password == "" {
		return nil, ErrStudioAuth
	}
	body, err := json.Marshal(map[string]string{
		"username": studioLoginUser,
		"password": s.password,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, ErrStudioAuth
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ErrStudioAuth
	}
	var out StudioSession
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, ErrStudioAuth
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		return nil, ErrStudioAuth
	}
	return &out, nil
}
