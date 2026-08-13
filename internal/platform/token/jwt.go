// Package token implements the application/token contract with HS256 JWTs.
package token

import (
	"context"
	"errors"
	"fmt"
	"time"

	apptoken "github.com/fatkulnurk/go-project-starter/internal/application/token"
	"github.com/golang-jwt/jwt/v5"
)

// Manager issues and parses signed access tokens.
type Manager struct {
	secret []byte
}

// NewManager builds a JWT manager with the given secret.
func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

type accessClaims struct {
	UserID string   `json:"uid"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// IssueAccessToken implements apptoken.Manager.
func (m *Manager) IssueAccessToken(_ context.Context, c apptoken.Claims, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := accessClaims{
		UserID: c.UserID,
		Roles:  c.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// ParseAccessToken implements apptoken.Manager.
func (m *Manager) ParseAccessToken(_ context.Context, raw string) (*apptoken.Claims, error) {
	var claims accessClaims
	_, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse access token: %w", err)
	}
	return &apptoken.Claims{UserID: claims.UserID, Roles: claims.Roles}, nil
}

// ErrInvalid is returned for structurally invalid tokens.
var ErrInvalid = errors.New("invalid token")
