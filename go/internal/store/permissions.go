package store

import "context"

// HasPermission reports whether any of the given roles grants the permission.
func HasPermission(ctx context.Context, q Querier, roles []string, permission string) (bool, error) {
	const sql = `
		SELECT EXISTS (
			SELECT 1 FROM role_permissions
			WHERE permission_id = $1 AND role_id = ANY($2)
		)`
	var ok bool
	if err := q.QueryRow(ctx, sql, permission, roles).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

// HasAnyCustomer reports whether any of the given roles has the any_customer flag,
// meaning the role may act on resources owned by other customers (e.g. manager, admin).
func HasAnyCustomer(ctx context.Context, q Querier, roles []string) (bool, error) {
	const sql = `SELECT EXISTS (SELECT 1 FROM roles WHERE id = ANY($1) AND any_customer = TRUE)`
	var ok bool
	if err := q.QueryRow(ctx, sql, roles).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}
