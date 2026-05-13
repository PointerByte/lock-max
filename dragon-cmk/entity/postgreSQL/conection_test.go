package postgreSQL

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestRestartPostgreSQLPool(t *testing.T) {
	t.Run("returns error when pool restart fails", func(t *testing.T) {
		originalNewPostgreSQLPoolFn := newPostgreSQLPoolFn
		originalCloseSQLDBFn := closeSQLDBFn
		originalPool := postgresPool
		originalSQLDB := postgresSQLDB
		originalErr := postgresPoolErr
		originalOnce := postgresPoolOnce

		t.Cleanup(func() {
			newPostgreSQLPoolFn = originalNewPostgreSQLPoolFn
			closeSQLDBFn = originalCloseSQLDBFn
			postgresPool = originalPool
			postgresSQLDB = originalSQLDB
			postgresPoolErr = originalErr
			postgresPoolOnce = originalOnce
		})

		newPostgreSQLPoolFn = func(_ context.Context) (*gorm.DB, error) {
			return nil, errors.New("restart error")
		}
		closeSQLDBFn = func(*sql.DB) error { return nil }

		if err := RestartPostgreSQLPool(); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("recreates the shared pool", func(t *testing.T) {
		originalNewPostgreSQLPoolFn := newPostgreSQLPoolFn
		originalCloseSQLDBFn := closeSQLDBFn
		originalGetSQLDBFn := getSQLDBFn
		originalPool := postgresPool
		originalSQLDB := postgresSQLDB
		originalErr := postgresPoolErr
		originalOnce := postgresPoolOnce

		t.Cleanup(func() {
			newPostgreSQLPoolFn = originalNewPostgreSQLPoolFn
			closeSQLDBFn = originalCloseSQLDBFn
			getSQLDBFn = originalGetSQLDBFn
			postgresPool = originalPool
			postgresSQLDB = originalSQLDB
			postgresPoolErr = originalErr
			postgresPoolOnce = originalOnce
		})

		closeCalled := false
		closeSQLDBFn = func(*sql.DB) error {
			closeCalled = true
			return nil
		}
		postgresPool = &gorm.DB{}
		postgresSQLDB = &sql.DB{}
		newPostgreSQLPoolFn = func(_ context.Context) (*gorm.DB, error) {
			return &gorm.DB{}, nil
		}
		getSQLDBFn = func(*gorm.DB) (*sql.DB, error) {
			return &sql.DB{}, nil
		}

		if err := RestartPostgreSQLPool(); err != nil {
			t.Fatalf("RestartPostgreSQLPool() error = %v", err)
		}
		if !closeCalled {
			t.Fatal("expected ClosePostgreSQLPool to close the previous pool")
		}
		if postgresPoolErr != nil {
			t.Fatalf("expected postgresPoolErr to be reset, got %v", postgresPoolErr)
		}
	})
}
