package auth

import (
	"context"
	"slices"

	"rbac-testing/internal/store"
)

type Subject interface {
	GetSubjectID() int64
}

// Ensure Claims implements Subject
var _ Subject = (*Claims)(nil)

type Authorizer interface {
	Can(ctx context.Context, sub Subject, action string, resource ...any) (bool, error)
}

type StoreAuthorizer struct {
	store store.Querier
}

func NewAuthorizer(store store.Querier) *StoreAuthorizer {
	return &StoreAuthorizer{store: store}
}

func (a *StoreAuthorizer) Can(ctx context.Context, sub Subject, action string, resource ...any) (bool, error) {
	// 1. Check basic RBAC permission
	perms, err := a.store.GetUserPermissions(ctx, sub.GetSubjectID())
	if err != nil {
		return false, err
	}
	if !slices.Contains(perms, action) {
		return false, nil
	}

	// 2. If no specific resource was passed, general permission is enough!
	if len(resource) == 0 {
		return true, nil
	}

	// 3. Optional: Fine-grained checks based on resource type
	switch res := resource[0].(type) {
	case store.ShoppingList:
		_ = res
		// Example: Only creator can touch it unless user is an Admin
		return true, nil
	}

	return true, nil
}
