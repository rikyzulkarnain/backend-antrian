package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// isUniqueViolation reports whether the error originates from a Postgres
// unique-constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
