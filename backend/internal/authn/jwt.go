package authn

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "user_id"
const sessionIDKey contextKey = "session_id"

const AccessCookie = "access_token"

type JWT struct {
	secret []byte
	secure bool
}

func NewJWT(secret string, secure bool) *JWT {
	return &JWT{secret: []byte(secret), secure: secure}
}

type claims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

func (a *JWT) Issue(userID, sessionID string, ttl time.Duration) (string, error) {
	now := time.Now()
	c := claims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(a.secret)
}

func (a *JWT) Parse(token string) (userID, sessionID string, err error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, jwt.ErrSignatureInvalid
		}
		return a.secret, nil
	})
	if err != nil {
		return "", "", err
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return "", "", jwt.ErrTokenInvalidClaims
	}
	return c.UserID, c.SessionID, nil
}

const cookiePath = "/" // site root: Unsloth Vite assets live at /assets, not /app

func cookieSameSite(secure bool) http.SameSite {
	if secure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func (a *JWT) SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessCookie,
		Value:    token,
		Path:     cookiePath,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: cookieSameSite(a.secure),
	})
}

func (a *JWT) ClearCookie(w http.ResponseWriter) {
	for _, path := range []string{cookiePath, "/app"} {
		http.SetCookie(w, &http.Cookie{
			Name:     AccessCookie,
			Value:    "",
			Path:     path,
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   a.secure,
			SameSite: cookieSameSite(a.secure),
		})
	}
}

func ExtractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if c, err := r.Cookie(AccessCookie); err == nil {
		return c.Value
	}
	return ""
}

func WithUser(ctx context.Context, userID, sessionID string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok && id != ""
}

func SessionID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(sessionIDKey).(string)
	return id, ok && id != ""
}
