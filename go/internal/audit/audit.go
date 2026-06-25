// Package audit writes the unbypassable audit trail. Records are written by the MCP
// server IN THE SAME TRANSACTION as the action (so "action happened <=> audit exists"),
// capturing allow AND deny. audit_log is append-only (REVOKE UPDATE/DELETE on app_user).
//
// M1 stores a per-row integrity hash (sha256 of the record). The tamper-evident hash
// CHAIN (prev_hash = previous row's row_hash) and the verifier are added in M10.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
)

const genesisPrevHash = "" // M10: replace with the previous row's row_hash

// Insert writes one audit record using the supplied Querier (pass the per-call tx so the
// audit is committed atomically with the action).
func Insert(ctx context.Context, q store.Querier, rec domain.AuditRecord) error {
	args := mustJSON(rec.Arguments)
	result := mustJSON(rec.Result)
	rowHash := hashRecord(rec, args, result)

	const sql = `
		INSERT INTO audit_log
			(correlation_id, user_id, action, tool, arguments, decision, reason, result, prev_hash, row_hash)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8::jsonb, $9, $10)`
	_, err := q.Exec(ctx, sql,
		rec.CorrelationID, nullable(rec.UserID), rec.Action, nullable(rec.Tool),
		args, rec.Decision, nullable(rec.Reason), result, genesisPrevHash, rowHash)
	return err
}

func hashRecord(rec domain.AuditRecord, args, result string) string {
	h := sha256.New()
	// Stable concatenation of the security-relevant fields.
	for _, f := range []string{rec.CorrelationID, rec.UserID, rec.Action, rec.Tool, args, rec.Decision, rec.Reason, result} {
		h.Write([]byte(f))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
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
