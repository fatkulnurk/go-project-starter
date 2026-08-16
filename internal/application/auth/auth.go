// Package auth defines the cross-cutting authentication contract: who is
// calling the system. Implementations live behind this interface (JWT, opaque
// tokens, sessions) so business modules never depend on a token library.
package auth

import "context"

// Identity describes the authenticated caller: their user id plus the roles
// asserted when the credential was issued.
type Identity struct {
	UserID string
	Roles  []string
}

// Authenticator resolves a credential (e.g. a bearer token) to an Identity.
// Implementations live behind this interface so modules never depend on a
// specific token library or scheme.
type Authenticator interface {
	// Authenticate returns the identity behind raw, or ErrUnauthenticated
	// when the credential is missing, invalid, revoked, or expired.
	Authenticate(ctx context.Context, raw string) (*Identity, error)
}

// ErrUnauthenticated is returned when the credential is missing/invalid/expired.
var ErrUnauthenticated = &unauthError{}

type unauthError struct{}

// Error implements the error interface for ErrUnauthenticated, returning the
// stable message that maps to the "unauthenticated" API code.
func (*unauthError) Error() string { return "unauthenticated" }

type ctxKey struct{}

// WithIdentity stores the identity in the context.
// HTTP middleware uses this so downstream handlers can recover the caller via
// IdentityFrom without threading it through every signature.
func WithIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// IdentityFrom returns the identity stored by WithIdentity, or nil.
// A nil result means the request is unauthenticated; callers that require an
// identity should treat nil as ErrUnauthenticated.
func IdentityFrom(ctx context.Context) *Identity {
	id, _ := ctx.Value(ctxKey{}).(*Identity)
	return id
}
