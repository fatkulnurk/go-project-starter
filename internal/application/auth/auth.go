// Package auth defines the cross-cutting authentication contract: who is
// calling the system. Implementations live behind this interface (JWT, opaque
// tokens, sessions) so business modules never depend on a token library.
package auth

import "context"

// Identity describes the authenticated caller.
type Identity struct {
	UserID string
	Roles  []string
}

// Authenticator resolves a credential (e.g. a bearer token) to an Identity.
type Authenticator interface {
	// Authenticate returns the identity behind raw, or ErrUnauthenticated.
	Authenticate(ctx context.Context, raw string) (*Identity, error)
}

// ErrUnauthenticated is returned when the credential is missing/invalid/expired.
var ErrUnauthenticated = &unauthError{}

type unauthError struct{}

func (*unauthError) Error() string { return "unauthenticated" }

type ctxKey struct{}

// WithIdentity stores the identity in the context.
func WithIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// IdentityFrom returns the identity stored by WithIdentity, or nil.
func IdentityFrom(ctx context.Context) *Identity {
	id, _ := ctx.Value(ctxKey{}).(*Identity)
	return id
}
