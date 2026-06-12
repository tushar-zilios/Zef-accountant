package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// BeginTx starts a new database transaction.
func BeginTx(ctx context.Context) (pgx.Tx, error) {
	return GetPool().Begin(ctx)
}
