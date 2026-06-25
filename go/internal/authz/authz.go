// Package authz is the Policy Decision Point: the single choke point every tool call
// passes through. It enforces RBAC (role -> permission) and resource ownership
// (resource.customer_id vs the verified caller), deny-by-default. The Subject comes only
// from the verified token — never from tool arguments.
package authz

import (
	"context"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
)

// PermissionChecker answers "do any of these roles grant this permission?".
type PermissionChecker func(ctx context.Context, roles []string, permission string) (bool, error)

// AnyCustomerChecker answers "does any of these roles carry any_customer = true?".
// Roles with that flag (manager, admin) may act on resources owned by other customers.
type AnyCustomerChecker func(ctx context.Context, roles []string) (bool, error)

// Authorize decides whether the subject may perform the tool's action.
//
// Step 1 — RBAC: the subject's roles must include a role that grants the required permission.
// Step 2 — Ownership: if ToolSpec.OwnerCustomerID is set, the subject must either own
// the resource (CustomerID matches) or hold a cross-customer role (teller/admin).
// Deny-by-default: if either check fails the call is denied and the reason is recorded.
func Authorize(
	ctx context.Context,
	hasPermission PermissionChecker,
	hasAnyCustomer AnyCustomerChecker,
	s domain.Subject,
	spec domain.ToolSpec,
) (domain.Decision, error) {
	// --- RBAC ---
	if spec.RequiredPermission != "" {
		ok, err := hasPermission(ctx, s.Roles, spec.RequiredPermission)
		if err != nil {
			return domain.Decision{}, err
		}
		if !ok {
			return domain.Deny("missing_permission"), nil
		}
	}

	// --- Ownership ---
	// The OwnerCustomerID is derived from the resource record loaded BEFORE this check.
	// It is never supplied by the LLM — the tool handler sets it after loading the resource.
	if spec.OwnerCustomerID != "" && spec.OwnerCustomerID != s.CustomerID {
		cross, err := hasAnyCustomer(ctx, s.Roles)
		if err != nil {
			return domain.Decision{}, err
		}
		if !cross {
			return domain.Deny("ownership_violation"), nil
		}
	}

	return domain.Allow(), nil
}
