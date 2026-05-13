package postgreSQL

import (
	"context"
	"fmt"
	"strings"
)

// ViewModel defines the minimum contract required to resolve the database view
// name for a typed query.
type ViewModel interface {
	TableName() string
}

var querySQLFn = querySQL

// ReadView returns all rows from the view represented by the generic model.
func ReadView[T ViewModel](ctx context.Context) ([]T, error) {
	var model T
	return QueryView[T](ctx, model.TableName())
}

// QueryModelView executes a SELECT against the view represented by the generic
// model and appends an optional query clause such as WHERE/ORDER BY/LIMIT.
func QueryModelView[T ViewModel](ctx context.Context, query string, args ...any) ([]T, error) {
	var model T
	return QueryViewWithClause[T](ctx, model.TableName(), query, args...)
}

// CountModelView executes a COUNT(*) against the view represented by the
// generic model and appends an optional filtering clause such as WHERE.
func CountModelView[T ViewModel](ctx context.Context, query string, args ...any) (uint, error) {
	var model T
	return CountViewWithClause(ctx, model.TableName(), query, args...)
}

// QueryView executes a SELECT against a database view or table-like object and
// maps the result set into the requested model type.
func QueryView[T any](ctx context.Context, sourceName string, args ...any) ([]T, error) {
	return QueryViewWithClause[T](ctx, sourceName, "", args...)
}

// QueryViewWithClause executes a SELECT against a database view or table-like
// object and appends an optional query clause such as WHERE/ORDER BY/LIMIT.
func QueryViewWithClause[T any](ctx context.Context, sourceName, queryClause string, args ...any) ([]T, error) {
	if err := validateObjectName(sourceName); err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM %s", sourceName)
	if strings.TrimSpace(queryClause) != "" {
		query = fmt.Sprintf("%s %s", query, queryClause)
	}

	var items []T
	if queryErr := querySQLFn(ctx, &items, query, args...); queryErr != nil {
		return nil, fmt.Errorf("query view %s: %w", sourceName, queryErr)
	}
	return items, nil
}

// CountViewWithClause executes a COUNT(*) against a database view or table-like
// object and appends an optional filtering clause such as WHERE.
func CountViewWithClause(ctx context.Context, sourceName, queryClause string, args ...any) (uint, error) {
	if err := validateObjectName(sourceName); err != nil {
		return 0, err
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", sourceName)
	if strings.TrimSpace(queryClause) != "" {
		query = fmt.Sprintf("%s %s", query, queryClause)
	}

	var total int64
	if queryErr := querySQLFn(ctx, &total, query, args...); queryErr != nil {
		return 0, fmt.Errorf("count view %s: %w", sourceName, queryErr)
	}
	return uint(total), nil
}

func querySQL(ctx context.Context, destination any, query string, args ...any) error {
	db, err := GetPostgreSQLPool(ctx)
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Raw(query, args...).Scan(destination).Error
}
