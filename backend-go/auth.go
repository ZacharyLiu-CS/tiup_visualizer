package main

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthService handles JWT creation and verification.
type AuthService struct {
	cfg *AppConfig
}

func NewAuthService(cfg *AppConfig) *AuthService {
	return &AuthService{cfg: cfg}
}

func (a *AuthService) Authenticate(username, password string) bool {
	return username == a.cfg.Auth.Username && password == a.cfg.Auth.Password
}

func (a *AuthService) CreateToken(username string) (string, int, error) {
	expireDuration := time.Duration(a.cfg.Auth.TokenExpireHours) * time.Hour
	expiresAt := time.Now().Add(expireDuration)

	claims := jwt.MapClaims{
		"sub": username,
		"exp": expiresAt.Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(a.cfg.Auth.SecretKey))
	if err != nil {
		return "", 0, err
	}
	return signed, int(expireDuration.Seconds()), nil
}

func (a *AuthService) VerifyToken(tokenStr string) (string, bool) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(a.cfg.Auth.SecretKey), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			slog.Warn("Token expired")
		} else {
			slog.Warn("Invalid token", "error", err)
		}
		return "", false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", false
	}

	sub, _ := claims.GetSubject()
	if sub == "" {
		return "", false
	}
	return sub, true
}

// ExtractToken extracts JWT token from Authorization header or query parameter.
func ExtractToken(r *http.Request) string {
	// Try Authorization header first
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	// Fallback to query parameter
	return r.URL.Query().Get("token")
}
