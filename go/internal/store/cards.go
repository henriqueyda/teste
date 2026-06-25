package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
)

var ErrCardNotFound = errors.New("card not found")

// GetCardByID loads a card by its ID. The caller is responsible for performing an
// ownership check before acting on the returned card's data.
func GetCardByID(ctx context.Context, q Querier, cardID string) (domain.Card, error) {
	const sql = `SELECT id, customer_id, limit_cents, max_limit_cents FROM cards WHERE id = $1`
	var c domain.Card
	err := q.QueryRow(ctx, sql, cardID).Scan(&c.ID, &c.CustomerID, &c.LimitCents, &c.MaxLimitCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Card{}, ErrCardNotFound
	}
	if err != nil {
		return domain.Card{}, err
	}
	return c, nil
}

// GetCardByCustomerID loads the card owned by the given customer.
func GetCardByCustomerID(ctx context.Context, q Querier, customerID string) (domain.Card, error) {
	const sql = `SELECT id, customer_id, limit_cents, max_limit_cents FROM cards WHERE customer_id = $1 LIMIT 1`
	var c domain.Card
	err := q.QueryRow(ctx, sql, customerID).Scan(&c.ID, &c.CustomerID, &c.LimitCents, &c.MaxLimitCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Card{}, ErrCardNotFound
	}
	if err != nil {
		return domain.Card{}, err
	}
	return c, nil
}

// UpdateCardLimit sets the card's limit. It enforces that the new limit is positive and
// does not exceed the card's eligibility ceiling (max_limit_cents).
func UpdateCardLimit(ctx context.Context, q Querier, cardID string, newLimitCents int64) error {
	const sql = `
		UPDATE cards SET limit_cents = $1
		WHERE id = $2 AND $1 > 0 AND $1 <= max_limit_cents`
	tag, err := q.Exec(ctx, sql, newLimitCents, cardID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("limit must be between R$0 and the card's maximum eligibility ceiling")
	}
	return nil
}
