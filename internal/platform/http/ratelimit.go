package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/apierr"
	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
)

// RateLimitByIP throttles requests per client IP. max attempts within window;
// excess requests get a 429. The counter lives in the shared cache so the
// limit holds across multiple API replicas.
func RateLimitByIP(c cache.Cache, max int64, window time.Duration) func(http.Handler) http.Handler {
	keyPrefix := "rl:ip:" + sha256Hex(window.String())
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if max <= 0 || window <= 0 {
				next.ServeHTTP(w, r)
				return
			}
			if err := incrementLimit(r.Context(), c, keyPrefix+ClientIP(r), max, window); err != nil {
				WriteMappedError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func incrementLimit(ctx context.Context, c cache.Cache, key string, max int64, window time.Duration) error {
	n, err := c.Increment(ctx, key, 1)
	if err != nil {
		return err
	}
	if n == 1 {
		_ = c.Expire(ctx, key, window)
	}
	if n > max {
		return apierr.ErrTooManyRequests
	}
	return nil
}
