package rbac

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
)

// Authorizer implements the cross-cutting authorization contract using RBAC
// data. Super admins implicitly hold every permission, mirroring Spatie's
// Gate::before.
type Authorizer struct {
	svc Service
}

// HasPermission implements authorization.Authorizer. It returns nil when the
// identity's user holds the permission (directly or via roles) or is a super
// admin; an empty permission allows everything (there is nothing to check).
// It returns authorization.ErrForbidden when access is denied, and propagates
// any underlying service error (e.g. a failed cache or database read).
func (a *Authorizer) HasPermission(ctx context.Context, identity authorization.Identity, permission string) error {
	if permission == "" {
		return nil
	}
	ok, err := a.svc.HasPermission(ctx, identity.UserID, permission)
	if err != nil {
		return err
	}
	if !ok {
		return authorization.ErrForbidden
	}
	return nil
}

// HasRole implements authorization.Authorizer. It returns nil when the
// identity's user holds the role (matched by code); an empty role allows
// everything. It returns authorization.ErrForbidden when the role is missing
// and propagates any underlying service error.
func (a *Authorizer) HasRole(ctx context.Context, identity authorization.Identity, role string) error {
	if role == "" {
		return nil
	}
	ok, err := a.svc.HasRole(ctx, identity.UserID, role)
	if err != nil {
		return err
	}
	if !ok {
		return authorization.ErrForbidden
	}
	return nil
}

var _ authorization.Authorizer = (*Authorizer)(nil)
