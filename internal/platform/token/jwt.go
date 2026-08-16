// Package token implements the application/token contract with HS256 JWTs.
package token

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/id"
	apptoken "github.com/fatkulnurk/go-project-starter/internal/application/token"
	"github.com/golang-jwt/jwt/v5"
)

// Manager issues and parses signed access tokens. The issuer and audience are
// embedded in every token and enforced on parse, which ties tokens to the
// environment they were minted for.
type Manager struct {
	secret   []byte
	issuer   string
	audience string
}

// NewManager builds a JWT manager. issuer and audience are validated against
// on parse; pass empty strings to disable the checks.
func NewManager(secret, issuer, audience string) *Manager {
	return &Manager{secret: []byte(secret), issuer: issuer, audience: audience}
}

type accessClaims struct {
	UserID string   `json:"uid"`
	Roles  []string `json:"roles"`
	JTI    string   `json:"jti,omitempty"`
	jwt.RegisteredClaims
}

// IssueAccessToken implements apptoken.Manager.
// It mints an HS256 token carrying the user ID, roles and a jti (generated
// when c.JTI is empty) valid for ttl. The context is ignored.
func (m *Manager) IssueAccessToken(_ context.Context, c apptoken.Claims, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	jti := c.JTI
	if jti == "" {
		jti = newID()
	}
	claims := accessClaims{
		UserID: c.UserID,
		Roles:  c.Roles,
		JTI:    jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			Subject:   c.UserID,
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
// It verifies the HS256 signature, expiry and issuer/audience (when
// configured) and returns ErrInvalid for any failure. The context is ignored.
func (m *Manager) ParseAccessToken(_ context.Context, raw string) (*apptoken.Claims, error) {
	var claims accessClaims
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		// Small clock-skew tolerance so a barely-expired token from a slightly
		// out-of-sync client is not spuriously rejected.
		jwt.WithLeeway(30*time.Second),
	)
	tok, err := parser.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse access token: %w", err)
	}
	if !tok.Valid {
		return nil, ErrInvalid
	}
	if m.issuer != "" && claims.Issuer != m.issuer {
		return nil, fmt.Errorf("%w: issuer mismatch", ErrInvalid)
	}
	if m.audience != "" && !containsString(claims.Audience, m.audience) {
		return nil, fmt.Errorf("%w: audience mismatch", ErrInvalid)
	}
	return &apptoken.Claims{UserID: claims.UserID, Roles: claims.Roles, JTI: claims.JTI}, nil
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// ErrInvalid is returned for structurally invalid tokens.
// ParseAccessToken returns it (possibly wrapped) when signature, expiry or
// issuer/audience validation fails.
var ErrInvalid = errors.New("invalid token")

// newID returns a version-7 UUID string, used for the jti claim.
func newID() string { return id.New() }
