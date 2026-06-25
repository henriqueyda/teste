package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
)

// ErrCustomerNotFound is returned when no customer matches the id.
var ErrCustomerNotFound = errors.New("customer not found")

// GetCustomerByID loads a customer profile.
func GetCustomerByID(ctx context.Context, q Querier, id string) (domain.Customer, error) {
	const sql = `SELECT id, full_name, document, is_retiree FROM customers WHERE id = $1`
	var c domain.Customer
	err := q.QueryRow(ctx, sql, id).Scan(&c.ID, &c.FullName, &c.Document, &c.IsRetiree)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Customer{}, ErrCustomerNotFound
	}
	if err != nil {
		return domain.Customer{}, err
	}
	return c, nil
}
