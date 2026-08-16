package command

import (
	"context"
	"log/slog"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
)

const jtiDenyPrefix = "auth:jti:deny:"

// JTIDenyKey returns the cache key marking an access-token id as revoked. The
// key is namespaced so it never collides with other cache entries.
func JTIDenyKey(jti string) string { return jtiDenyPrefix + jti }

// jtiDenylist records revoked access-token ids in the shared cache so
// Authenticate can reject them before their natural expiry. Revocations live
// at least as long as the access token TTL, so an entry always covers the
// token's remaining life.
type jtiDenylist struct {
	cache cache.Cache
	ttl   time.Duration
}

// newJTIDenylist builds a denylist that keeps entries for at least accessTTL.
// A nil cache disables the denylist (revocation then relies on token expiry).
func newJTIDenylist(c cache.Cache, accessTTL time.Duration) *jtiDenylist {
	return &jtiDenylist{cache: c, ttl: accessTTL}
}

// NewJTIDenylist is the exported constructor used by the composition root and
// module wiring. It delegates to the unexported newJTIDenylist.
func NewJTIDenylist(c cache.Cache, accessTTL time.Duration) *jtiDenylist {
	return newJTIDenylist(c, accessTTL)
}

// deny marks the given access-token ids as revoked. Failures are logged but
// never fail the caller: the refresh token is already revoked in the database,
// so the security boundary holds even if the denylist write fails.
func (d *jtiDenylist) deny(ctx context.Context, jtis []string) {
	if d == nil || d.cache == nil {
		return
	}
	for _, j := range jtis {
		if j == "" {
			continue
		}
		if err := d.cache.Set(ctx, JTIDenyKey(j), []byte("1"), d.ttl); err != nil {
			slog.Warn("access token denylist write failed", "err", err)
		}
	}
}
