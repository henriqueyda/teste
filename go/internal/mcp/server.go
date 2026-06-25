// Package mcp is the security kernel, exposed over the Model Context Protocol using the
// official Go SDK (Streamable HTTP transport). It is the only path from agent reasoning to
// action. Every tool call runs the same pipeline — verify run-as token -> RBAC/ownership ->
// execute -> audit, all in one transaction — so the guarantees hold regardless of transport.
package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/henrique-yda/teste-tecnico-itau/internal/audit"
	"github.com/henrique-yda/teste-tecnico-itau/internal/auth"
	"github.com/henrique-yda/teste-tecnico-itau/internal/authz"
	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
)

// Deps are the dependencies every tool handler needs.
type Deps struct {
	Pool     *pgxpool.Pool
	Secret   []byte // HMAC-SHA256 secret for token verification
	Issuer   string // expected JWT issuer
	Audience string // expected JWT audience
}

// NewHTTPHandler builds the MCP server, registers the allowlisted tools, and returns the
// Streamable HTTP handler. A single server instance serves every request; identity is read
// per-call from the request headers (see subjectFromRequest), never bound at build time.
func NewHTTPHandler(d Deps) http.Handler {
	s := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "banking-agent-mcp", Version: "0.1.0"}, nil)
	registerTools(s, d)
	return mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server { return s }, nil)
}

// registerTools is the allowlist: only tools added here can be called.
func registerTools(s *mcpsdk.Server, d Deps) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_customer_profile",
		Description: "Return the authenticated caller's own customer profile. Takes no identity argument; the customer is resolved from the verified session token.",
	}, d.getCustomerProfile)

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_account_balance",
		Description: "Return the authenticated caller's own bank account balance and daily PIX limit. Takes no arguments; the account is resolved from the verified session token.",
	}, d.getAccountBalance)

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name: "get_card_limit",
		Description: "Return the current and maximum credit card limit for the given card. " +
			"Requires card_id. The ownership check ensures only the card owner (or a teller/admin) can view it.",
	}, d.getCardLimit)

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name: "update_card_limit",
		Description: "Set a new credit card limit. " +
			"Requires card_id and new_limit_cents (must be positive and within the card's eligibility ceiling). " +
			"Only the card owner (or admin) may increase their own card's limit.",
	}, d.updateCardLimit)

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name: "create_pix",
		Description: "Send a PIX transfer from the authenticated customer's account. " +
			"Requires recipient_key (PIX key: CPF, phone, e-mail, or random key) and amount_cents (positive integer in centavos). " +
			"Requires transaction_pin (the customer's 4-digit transaction password).",
	}, d.createPix)
}

// subjectFromRequest derives the verified caller from the run-as token carried in the
// Authorization header (out-of-band; never in tool arguments). The kernel re-verifies the
// token itself — it does not trust the caller's claims about identity.
func (d Deps) subjectFromRequest(req *mcpsdk.CallToolRequest) (domain.Subject, string, error) {
	if req.Extra == nil || req.Extra.Header == nil {
		return domain.Subject{}, "", fmt.Errorf("missing request headers")
	}
	token, ok := strings.CutPrefix(req.Extra.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" {
		return domain.Subject{}, "", fmt.Errorf("missing run-as token")
	}
	subject, err := auth.VerifyUserToken(d.Secret, d.Issuer, d.Audience, token)
	if err != nil {
		return domain.Subject{}, "", fmt.Errorf("invalid run-as token")
	}
	corrID := req.Extra.Header.Get("X-Correlation-Id")
	if corrID == "" {
		corrID = newCorrelationID()
	}
	return subject, corrID, nil
}

// runToolPipeline runs authorization + execution + audit for one tool call inside a single
// transaction. Both allow and deny outcomes are audited. exec does the real work using the
// provided tx and returns the result map recorded in the audit row.
func (d Deps) runToolPipeline(
	ctx context.Context,
	subject domain.Subject,
	corrID string,
	spec domain.ToolSpec,
	args map[string]any,
	exec func(tx pgx.Tx) (map[string]any, error),
) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("internal error")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	auditWithin := func(decision, reason string, result map[string]any) {
		_ = audit.Insert(ctx, tx, domain.AuditRecord{
			CorrelationID: corrID,
			UserID:        subject.UserID,
			Action:        spec.Name,
			Tool:          spec.Name,
			Arguments:     args,
			Decision:      decision,
			Reason:        reason,
			Result:        result,
			Timestamp:     time.Now().UTC(),
		})
	}

	hasPerm := func(ctx context.Context, roles []string, perm string) (bool, error) {
		return store.HasPermission(ctx, tx, roles, perm)
	}
	hasAnyCustomer := func(ctx context.Context, roles []string) (bool, error) {
		return store.HasAnyCustomer(ctx, tx, roles)
	}
	decision, err := authz.Authorize(ctx, hasPerm, hasAnyCustomer, subject, spec)
	if err != nil {
		return fmt.Errorf("internal error")
	}
	if !decision.Allow {
		auditWithin("deny", decision.Reason, nil)
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("internal error")
		}
		committed = true
		return fmt.Errorf("access denied: %s", decision.Reason)
	}

	result, err := exec(tx)
	if err != nil {
		return err // rollback via defer; no allow-audit
	}
	auditWithin("allow", "", result)
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("internal error")
	}
	committed = true
	return nil
}

func newCorrelationID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// traceCtxFromRequest extracts W3C TraceContext from MCP request headers so Go spans
// appear as children of the Python agent's span in Jaeger.
func traceCtxFromRequest(ctx context.Context, req *mcpsdk.CallToolRequest) context.Context {
	if req.Extra == nil || req.Extra.Header == nil {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Extra.Header))
}

// startToolSpan extracts the parent trace from the MCP request and starts a child span
// named after the tool. Caller must defer span.End().
func startToolSpan(ctx context.Context, req *mcpsdk.CallToolRequest, toolName string) (context.Context, trace.Span) {
	ctx = traceCtxFromRequest(ctx, req)
	ctx, span := otel.Tracer("banking-mcp").Start(ctx, "mcp."+toolName)
	return ctx, span
}
