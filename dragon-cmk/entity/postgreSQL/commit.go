package postgreSQL

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

func Commit(ctx context.Context) (*pgconn.CommandTag, error) {
	_, execErr := execSQLFn(ctx, "COMMIT")
	if execErr != nil {
		return nil, fmt.Errorf("COMMIT transaction: %w", execErr)
	}

	commandTag := pgconn.NewCommandTag("COMMIT")
	return &commandTag, nil
}

func Rollback(ctx context.Context) (*pgconn.CommandTag, error) {
	_, execErr := execSQLFn(ctx, "ROLLBACK")
	if execErr != nil {
		return nil, fmt.Errorf("ROLLBACK transaction: %w", execErr)
	}

	commandTag := pgconn.NewCommandTag("ROLLBACK")
	return &commandTag, nil
}
