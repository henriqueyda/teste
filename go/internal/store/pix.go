package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetTransactionPINHash returns the bcrypt hash of the user's transaction PIN.
// Returns an error if the user has no PIN configured (e.g. staff accounts).
func GetTransactionPINHash(ctx context.Context, q Querier, userID string) (string, error) {
	const sql = `SELECT transaction_pin_hash FROM users WHERE id = $1`
	var hash *string
	if err := q.QueryRow(ctx, sql, userID).Scan(&hash); err != nil {
		return "", err
	}
	if hash == nil {
		return "", fmt.Errorf("this account has no transaction PIN configured")
	}
	return *hash, nil
}

// CreatePinSession generates a random one-use token valid for 60 seconds and inserts it into
// pin_sessions. The caller (gateway /step-up) has already verified the user's PIN via bcrypt.
func CreatePinSession(ctx context.Context, q Querier, userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("internal error")
	}
	token := hex.EncodeToString(b)
	const sql = `INSERT INTO pin_sessions (id, user_id, expires_at) VALUES ($1, $2, now() + interval '60 seconds')`
	if _, err := q.Exec(ctx, sql, token, userID); err != nil {
		return "", fmt.Errorf("internal error")
	}
	return token, nil
}

// ConsumePinSession marks a pin_session as used atomically.
// Returns an error if the token is unknown, already used, or expired.
func ConsumePinSession(ctx context.Context, q Querier, userID, token string) error {
	const sql = `
		UPDATE pin_sessions SET used_at = now()
		WHERE id = $1 AND user_id = $2 AND expires_at > now() AND used_at IS NULL
		RETURNING id`
	var id string
	err := q.QueryRow(ctx, sql, token, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("token de sessão inválido ou expirado")
	}
	return err
}

// DebitAccount decrements the account balance for the given customer, returning the new balance.
// Uses customer_id (not an LLM-supplied account_id) to prevent IDOR.
func DebitAccount(ctx context.Context, q Querier, customerID string, amountCents int64) (int64, error) {
	const sql = `
		UPDATE accounts SET balance_cents = balance_cents - $1
		WHERE customer_id = $2 AND balance_cents >= $1
		RETURNING balance_cents`
	var newBalance int64
	err := q.QueryRow(ctx, sql, amountCents, customerID).Scan(&newBalance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("saldo insuficiente para a transferência")
	}
	return newBalance, err
}
