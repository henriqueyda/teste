// Package audit writes the unbypassable audit trail. Records are written by the MCP
// server IN THE SAME TRANSACTION as the action (so "action happened <=> audit exists"),
// capturing allow AND deny. audit_log is append-only (REVOKE UPDATE/DELETE on app_user).
package audit

import (
	"context"
	"encoding/json"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
)

// Insert writes one audit record using the supplied Querier (pass the per-call tx so the
// audit is committed atomically with the action).
func Insert(ctx context.Context, q store.Querier, rec domain.AuditRecord) error {
	args := mustJSON(rec.Arguments)
	result := mustJSON(rec.Result)

	const sql = `
		INSERT INTO audit_log
			(correlation_id, user_id, action, tool, arguments, decision, reason, result)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8::jsonb)`
	_, err := q.Exec(ctx, sql,
		rec.CorrelationID, nullable(rec.UserID), rec.Action, nullable(rec.Tool),
		args, rec.Decision, nullable(rec.Reason), result)
	return err
}

func mustJSON(m map[string]any) string {
	if m == nil {
		return "null"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "null"
	}
	return string(b)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
