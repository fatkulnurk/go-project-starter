// Package token defines the cross-cutting token contract. Business modules
// only know Manager; the concrete implementation (JWT, opaque, ...) is
// chosen in the composition root.
package token

import (
	"context"
	"time"
)

// Claims describe what is encoded inside an access token.
type Claims struct {
	UserID string
	Roles  []string
}

// Manager issues and parses access tokens.
type Manager interface {
	// IssueAccessToken mints an access token for the identity, valid for ttl.
	IssueAccessToken(ctx context.Context, c Claims, ttl time.Duration) (string, error)
	// ParseAccessToken validates raw and returns its claims.
	ParseAccessToken(ctx context.Context, raw string) (*Claims, error)
}
