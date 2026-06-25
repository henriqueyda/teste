// Package store is the only place that touches SQL (Repository pattern, pgx). It keeps
// package domain free of persistence concerns. The services connect as the
// least-privilege 'app_user' role, which can INSERT+SELECT audit_log but not UPDATE or
// DELETE it (append-only enforced at the DB level).
// (RAG vectors are NOT here — they live in FAISS in the Python tier.)
package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, so repository functions work
// the same whether they run standalone or inside the per-tool-call transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Connect opens a pgx connection pool.
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, dsn)
}
