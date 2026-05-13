package postgreSQL

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var objectNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)*$`)

var execSQLFn = execSQL

// CallStoredProcedure executes a PostgreSQL stored procedure using CALL and
// returns the database command tag.
func CallStoredProcedure(ctx context.Context, procedureName string, args ...any) (*pgconn.CommandTag, error) {
	if err := validateObjectName(procedureName); err != nil {
		return nil, err
	}

	query := fmt.Sprintf("CALL %s(%s)", procedureName, buildPlaceholders(len(args)))

	_, execErr := execSQLFn(ctx, query, args...)
	if execErr != nil {
		return nil, fmt.Errorf("call stored procedure %s: %w", procedureName, execErr)
	}

	commandTag := pgconn.NewCommandTag("CALL")
	return &commandTag, nil
}

// ReadStoredProcedure executes a PostgreSQL routine that returns rows and maps
// the result set into the requested model type.
func ReadStoredProcedure[T any](ctx context.Context, procedureName string, args ...any) ([]T, error) {
	if err := validateObjectName(procedureName); err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM %s(%s)", procedureName, buildPlaceholders(len(args)))

	var items []T
	if queryErr := querySQLFn(ctx, &items, query, args...); queryErr != nil {
		return nil, fmt.Errorf("read stored procedure %s: %w", procedureName, queryErr)
	}

	return items, nil
}

func execSQL(ctx context.Context, query string, args ...any) (int64, error) {
	db, err := GetPostgreSQLPool(ctx)
	if err != nil {
		return 0, err
	}

	result := db.WithContext(ctx).Exec(query, args...)
	return result.RowsAffected, result.Error
}

func buildPlaceholders(total int) string {
	if total == 0 {
		return ""
	}

	placeholders := make([]string, total)
	for index := range total {
		placeholders[index] = fmt.Sprintf("$%d", index+1)
	}

	return strings.Join(placeholders, ", ")
}

func validateObjectName(name string) error {
	if !objectNamePattern.MatchString(name) {
		return fmt.Errorf("invalid database object name: %s", name)
	}

	return nil
}
