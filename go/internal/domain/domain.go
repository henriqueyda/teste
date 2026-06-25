// Package domain holds the core, dependency-free types shared across the Go tier:
// Subject (the verified caller), banking entities, ToolSpec, and the authorization
// Decision. Keeping these pure (no DB/HTTP imports) is the center of the hexagonal
// architecture — everything else depends inward on this package.
package domain

import "time"

// Subject is the authenticated caller. It is built ONLY from a cryptographically
// verified token — never from tool arguments or chat content. This is the identity
// every authorization decision is made against.
type Subject struct {
	UserID     string
	CustomerID string // "" for staff (teller/admin) who are not themselves customers
	Roles      []string
}

// Customer is a banking client (the system-of-record entity behind get_customer_profile).
type Customer struct {
	ID        string
	FullName  string
	Document  string
	IsRetiree bool
}

// Account is the banking account owned by a customer.
type Account struct {
	ID            string
	CustomerID    string
	BalanceCents  int64
	PixDailyLimit int64
}

// Card is a credit/debit card owned by a customer.
type Card struct {
	ID            string
	CustomerID    string
	LimitCents    int64
	MaxLimitCents int64
}

// ToolSpec declares a tool's security requirements. The MCP server uses it to decide
// what to enforce before executing. RequiredPermission drives the RBAC check;
// OwnerCustomerID carries the resource owner so authz can deny cross-customer access;
// Critical flags actions that will require confirmation/step-up (wired in M6/M7).
type ToolSpec struct {
	Name               string
	RequiredPermission string
	OwnerCustomerID    string // "" means no ownership check; set after loading the resource
	Critical           bool
}

// Decision is the outcome of authorization. Deny carries a machine-readable Reason that
// is recorded in the audit trail (e.g. "missing_permission", "ownership_violation").
type Decision struct {
	Allow  bool
	Reason string
}

// Allow returns an allow decision.
func Allow() Decision { return Decision{Allow: true} }

// Deny returns a deny decision with a reason.
func Deny(reason string) Decision { return Decision{Allow: false, Reason: reason} }

// AuditRecord is one entry in the tamper-evident audit trail. It is written by the MCP
// server in the SAME transaction as the action, for both allow and deny outcomes.
// See contracts/audit-record.schema.json.
type AuditRecord struct {
	CorrelationID string
	UserID        string
	Action        string
	Tool          string
	Arguments     map[string]any
	Decision      string // "allow" | "deny"
	Reason        string
	Result        map[string]any
	Timestamp     time.Time
}
