package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ErrUserNotFound is returned when no user matches the lookup. Login handlers should
// treat this the same as a wrong password (avoid leaking which usernames exist).
var ErrUserNotFound = errors.New("user not found")

// AuthUser is the login-time view of a user: enough to verify the password and to mint
// a token (roles + customer_id). customer_id is empty for staff.
type AuthUser struct {
	ID           string
	Username     string
	PasswordHash string
	CustomerID   string
	Roles        []string
}

// GetUserByUsername loads a user and their roles for authentication.
func GetUserByUsername(ctx context.Context, q Querier, username string) (AuthUser, error) {
	const sql = `
		SELECT u.id, u.username, u.password_hash, COALESCE(u.customer_id, ''),
		       COALESCE(array_agg(ur.role_id) FILTER (WHERE ur.role_id IS NOT NULL), '{}')
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		WHERE u.username = $1
		GROUP BY u.id`
	var u AuthUser
	err := q.QueryRow(ctx, sql, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CustomerID, &u.Roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthUser{}, ErrUserNotFound
	}
	if err != nil {
		return AuthUser{}, err
	}
	return u, nil
}
