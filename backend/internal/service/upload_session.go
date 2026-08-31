package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UploadSessionClaims struct {
	WorkspaceID string  `json:"workspace_id"`
	UserID      string  `json:"user_id"`
	S3Key       string  `json:"s3_key"`
	Name        string  `json:"name"`
	MimeType    string  `json:"mime_type"`
	Size        int64   `json:"size"`
	FolderID    *string `json:"folder_id"`
	jwt.RegisteredClaims
}

type UploadSessionService struct {
	secret []byte
	ttl    time.Duration
}

func NewUploadSessionService(secret string) *UploadSessionService {
	return &UploadSessionService{secret: []byte(secret), ttl: 15 * time.Minute}
}

func (s *UploadSessionService) Create(c UploadSessionClaims) (string, error) {
	c.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   "upload",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
}

func (s *UploadSessionService) Verify(tokenStr string) (*UploadSessionClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &UploadSessionClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*UploadSessionClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid upload session")
	}
	return claims, nil
}
