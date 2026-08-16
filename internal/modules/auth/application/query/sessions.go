// Package queries contains the read-side use cases of the auth module.
package query

import (
	"context"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// Session is one active login session (a refresh-token family) together with
// its creation, last-use and expiry timestamps.
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Sessions is the read-side use case that lists the active sessions of a user.
// It is the counterpart of SessionRevoke for inspecting active logins.
type Sessions struct {
	refreshTokens domain.RefreshTokenRepository
	clock         clock.Clock
}

// NewSessions builds the sessions query from the refresh-token repository and
// the clock used to decide which families are still active.
func NewSessions(refreshTokens domain.RefreshTokenRepository, clk clock.Clock) *Sessions {
	return &Sessions{refreshTokens: refreshTokens, clock: clk}
}

// Execute returns the user's active sessions; an empty (non-nil) slice is
// returned when there are none. Repository errors are passed through.
func (q *Sessions) Execute(ctx context.Context, userID string) ([]Session, error) {
	families, err := q.refreshTokens.ListActiveFamilies(ctx, userID, q.clock.Now())
	if err != nil {
		return nil, err
	}
	out := make([]Session, 0, len(families))
	for _, f := range families {
		out = append(out, Session{
			ID:        f.ID,
			CreatedAt: f.CreatedAt,
			LastUsed:  f.LastUsed,
			ExpiresAt: f.ExpiresAt,
		})
	}
	return out, nil
}
