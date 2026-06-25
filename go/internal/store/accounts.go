package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
)

var ErrAccountNotFound = errors.New("account not found")

// GetAccountByCustomerID loads the account owned by the given customer. Because we look
// up by customer_id (not an LLM-supplied account_id), there is no IDOR surface here.
func GetAccountByCustomerID(ctx context.Context, q Querier, customerID string) (domain.Account, error) {
	const sql = `SELECT id, customer_id, balance_cents, pix_daily_limit_cents FROM accounts WHERE customer_id = $1 LIMIT 1`
	var a domain.Account
	err := q.QueryRow(ctx, sql, customerID).Scan(&a.ID, &a.CustomerID, &a.BalanceCents, &a.PixDailyLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, ErrAccountNotFound
	}
	if err != nil {
		return domain.Account{}, err
	}
	return a, nil
}
