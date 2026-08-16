// Package token defines the cross-cutting token contract. Business modules
// only know Manager; the concrete implementation (JWT, opaque, ...) is
// chosen in the composition root.
package token

import (
	"context"
	"time"
)

// Claims describe what is encoded inside an access token: the owning user, the
// roles asserted at issue time, and the token id used for revocation.
type Claims struct {
	UserID string
	Roles  []string
	// JTI is the token id minted at issue time. The auth module stores it on
	// the refresh-token row so revoking a session can deny outstanding access
	// tokens immediately via the denylist.
	JTI string
}

// Manager issues and parses signed access tokens.
// The concrete implementation (JWT, opaque, ...) is chosen in the composition
// root, so business modules only ever depend on this contract.
type Manager interface {
	// IssueAccessToken mints an access token for the identity, valid for ttl.
	// The returned string is opaque to the caller and carries no side effects.
	IssueAccessToken(ctx context.Context, c Claims, ttl time.Duration) (string, error)

	// ParseAccessToken validates raw and returns its claims. It rejects tokens
	// that are malformed, expired, or signed with a different key/scheme.
	ParseAccessToken(ctx context.Context, raw string) (*Claims, error)
}
