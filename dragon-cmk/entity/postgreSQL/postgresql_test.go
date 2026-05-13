package postgreSQL

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type fakeViewModel struct {
	Name string `db:"name" gorm:"column:name"`
}

func (fakeViewModel) TableName() string {
	return "dragon_cmk.fake_view"
}

func TestBuildPlaceholders(t *testing.T) {
	if got := buildPlaceholders(0); got != "" {
		t.Fatalf("buildPlaceholders(0) = %q, want empty string", got)
	}

	if got := buildPlaceholders(3); got != "$1, $2, $3" {
		t.Fatalf("buildPlaceholders(3) = %q", got)
	}
}

func TestValidateObjectName(t *testing.T) {
	validNames := []string{
		"dragon_cmk.sp_create_cmk_key",
		"vw_cmk_key",
		"schema_1.object_2",
	}

	for _, name := range validNames {
		if err := validateObjectName(name); err != nil {
			t.Fatalf("validateObjectName(%q) unexpected error: %v", name, err)
		}
	}

	invalidNames := []string{
		"dragon-cmk.bad",
		"bad name",
		"schema.table;",
		"",
	}

	for _, name := range invalidNames {
		if err := validateObjectName(name); err == nil {
			t.Fatalf("validateObjectName(%q) expected error", name)
		}
	}
}

func TestCallStoredProcedure(t *testing.T) {
	t.Run("rejects invalid procedure name", func(t *testing.T) {
		_, err := CallStoredProcedure(context.Background(), "bad name")
		if err == nil {
			t.Fatal("expected error for invalid procedure name")
		}
	})

	t.Run("returns execution error", func(t *testing.T) {
		recorder := stubExecSQL(t, 0, errors.New("exec failure"))

		_, err := CallStoredProcedure(context.Background(), "dragon_cmk.sp_test", 1, "two")
		if err == nil || !strings.Contains(err.Error(), "exec failure") {
			t.Fatalf("unexpected error: %v", err)
		}

		if recorder.sql != "CALL dragon_cmk.sp_test($1, $2)" {
			t.Fatalf("unexpected SQL: %s", recorder.sql)
		}
	})

	t.Run("executes successfully", func(t *testing.T) {
		recorder := stubExecSQL(t, 1, nil)

		tag, err := CallStoredProcedure(context.Background(), "dragon_cmk.sp_test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tag.String() != "CALL" {
			t.Fatalf("unexpected command tag: %s", tag.String())
		}

		if recorder.calls != 1 {
			t.Fatalf("expected one Exec call, got %d", recorder.calls)
		}
	})
}

func TestTransactionCommands(t *testing.T) {
	tests := []struct {
		name      string
		call      func(context.Context) (*pgconn.CommandTag, error)
		sql       string
		errorText string
	}{
		{
			name:      "commit",
			call:      Commit,
			sql:       "COMMIT",
			errorText: "COMMIT transaction",
		},
		{
			name:      "rollback",
			call:      Rollback,
			sql:       "ROLLBACK",
			errorText: "ROLLBACK transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_exec_error", func(t *testing.T) {
			recorder := stubExecSQL(t, 0, errors.New("exec failure"))

			_, err := tt.call(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.errorText) || !strings.Contains(err.Error(), "exec failure") {
				t.Fatalf("unexpected error: %v", err)
			}
			if recorder.sql != tt.sql {
				t.Fatalf("unexpected SQL: %s", recorder.sql)
			}
		})

		t.Run(tt.name+"_success", func(t *testing.T) {
			recorder := stubExecSQL(t, 1, nil)

			tag, err := tt.call(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tag == nil || tag.String() != tt.sql {
				t.Fatalf("unexpected command tag: %#v", tag)
			}
			if recorder.calls != 1 {
				t.Fatalf("expected one Exec call, got %d", recorder.calls)
			}
			if recorder.sql != tt.sql {
				t.Fatalf("unexpected SQL: %s", recorder.sql)
			}
		})
	}
}

func TestReadStoredProcedure(t *testing.T) {
	type result struct {
		Name string `db:"name" gorm:"column:name"`
	}

	t.Run("rejects invalid procedure name", func(t *testing.T) {
		_, err := ReadStoredProcedure[result](context.Background(), "bad name")
		if err == nil {
			t.Fatal("expected error for invalid procedure name")
		}
	})

	t.Run("returns query error", func(t *testing.T) {
		recorder := stubQuerySQL(t, nil, errors.New("query failure"))

		_, err := ReadStoredProcedure[result](context.Background(), "dragon_cmk.fn_test", 7)
		if err == nil || !strings.Contains(err.Error(), "query failure") {
			t.Fatalf("unexpected error: %v", err)
		}

		if recorder.sql != "SELECT * FROM dragon_cmk.fn_test($1)" {
			t.Fatalf("unexpected SQL: %s", recorder.sql)
		}
	})

	t.Run("scans successfully", func(t *testing.T) {
		recorder := stubQuerySQL(t, []result{{Name: "alice"}, {Name: "bob"}}, nil)

		items, err := ReadStoredProcedure[result](context.Background(), "dragon_cmk.fn_test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(items) != 2 || items[0].Name != "alice" || items[1].Name != "bob" {
			t.Fatalf("unexpected items: %#v", items)
		}
		if recorder.calls != 1 {
			t.Fatalf("expected one query call, got %d", recorder.calls)
		}
	})
}

func TestQueryViewAndReadView(t *testing.T) {
	t.Run("query view rejects invalid source name", func(t *testing.T) {
		_, err := QueryView[fakeViewModel](context.Background(), "bad name")
		if err == nil {
			t.Fatal("expected error for invalid source name")
		}
	})

	t.Run("query view returns query error", func(t *testing.T) {
		recorder := stubQuerySQL(t, nil, errors.New("query failure"))

		_, err := QueryView[fakeViewModel](context.Background(), "dragon_cmk.fake_view")
		if err == nil || !strings.Contains(err.Error(), "query failure") {
			t.Fatalf("unexpected error: %v", err)
		}

		if recorder.sql != "SELECT * FROM dragon_cmk.fake_view" {
			t.Fatalf("unexpected SQL: %s", recorder.sql)
		}
	})

	t.Run("query view scans successfully", func(t *testing.T) {
		stubQuerySQL(t, []fakeViewModel{{Name: "alice"}}, nil)

		items, err := QueryView[fakeViewModel](context.Background(), "dragon_cmk.fake_view")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(items) != 1 || items[0].Name != "alice" {
			t.Fatalf("unexpected items: %#v", items)
		}
	})

	t.Run("query view with clause appends SQL and arguments", func(t *testing.T) {
		recorder := stubQuerySQL(t, []fakeViewModel{{Name: "alice"}}, nil)

		items, err := QueryViewWithClause[fakeViewModel](context.Background(), "dragon_cmk.fake_view", "WHERE name = $1 ORDER BY name", "alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 || items[0].Name != "alice" {
			t.Fatalf("unexpected items: %#v", items)
		}
		if recorder.sql != "SELECT * FROM dragon_cmk.fake_view WHERE name = $1 ORDER BY name" {
			t.Fatalf("unexpected SQL: %s", recorder.sql)
		}
		if len(recorder.args) != 1 || recorder.args[0] != "alice" {
			t.Fatalf("unexpected args: %#v", recorder.args)
		}
	})

	t.Run("read view uses model table name", func(t *testing.T) {
		recorder := stubQuerySQL(t, []fakeViewModel{{Name: "alice"}}, nil)

		items, err := ReadView[fakeViewModel](context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(items) != 1 || items[0].Name != "alice" {
			t.Fatalf("unexpected items: %#v", items)
		}

		if recorder.sql != "SELECT * FROM dragon_cmk.fake_view" {
			t.Fatalf("unexpected SQL: %s", recorder.sql)
		}
	})

	t.Run("query model view uses model table name and clause", func(t *testing.T) {
		recorder := stubQuerySQL(t, []fakeViewModel{{Name: "alice"}}, nil)

		items, err := QueryModelView[fakeViewModel](context.Background(), "WHERE name = $1", "alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 || items[0].Name != "alice" {
			t.Fatalf("unexpected items: %#v", items)
		}
		if recorder.sql != "SELECT * FROM dragon_cmk.fake_view WHERE name = $1" {
			t.Fatalf("unexpected SQL: %s", recorder.sql)
		}
	})

	t.Run("count view with clause appends SQL and arguments", func(t *testing.T) {
		recorder := stubQuerySQL(t, int64(7), nil)

		total, err := CountViewWithClause(context.Background(), "dragon_cmk.fake_view", "WHERE name = $1", "alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 7 {
			t.Fatalf("unexpected total: %d", total)
		}
		if recorder.sql != "SELECT COUNT(*) FROM dragon_cmk.fake_view WHERE name = $1" {
			t.Fatalf("unexpected SQL: %s", recorder.sql)
		}
		if len(recorder.args) != 1 || recorder.args[0] != "alice" {
			t.Fatalf("unexpected args: %#v", recorder.args)
		}
	})

	t.Run("count model view uses model table name", func(t *testing.T) {
		recorder := stubQuerySQL(t, int64(3), nil)

		total, err := CountModelView[fakeViewModel](context.Background(), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 3 {
			t.Fatalf("unexpected total: %d", total)
		}
		if recorder.sql != "SELECT COUNT(*) FROM dragon_cmk.fake_view" {
			t.Fatalf("unexpected SQL: %s", recorder.sql)
		}
	})

	t.Run("count view rejects invalid source name", func(t *testing.T) {
		if _, err := CountViewWithClause(context.Background(), "bad name", ""); err == nil {
			t.Fatal("expected error for invalid source name")
		}
	})

	t.Run("count view returns query error", func(t *testing.T) {
		recorder := stubQuerySQL(t, int64(0), errors.New("query failure"))

		_, err := CountViewWithClause(context.Background(), "dragon_cmk.fake_view", "")
		if err == nil || !strings.Contains(err.Error(), "query failure") {
			t.Fatalf("unexpected error: %v", err)
		}
		if recorder.sql != "SELECT COUNT(*) FROM dragon_cmk.fake_view" {
			t.Fatalf("unexpected SQL: %s", recorder.sql)
		}
	})
}

func TestLoadPostgreSQLConfig(t *testing.T) {
	setEnv(t, "PGUSER", "dragon")
	setEnv(t, "PGPASSWORD", "secret")
	setEnv(t, "PGHOST", "localhost")
	setEnv(t, "PGPORT", "5432")
	setEnv(t, "PGDATABASE", "origin")
	setEnv(t, "PGSSLMODE", "disable")
	setEnv(t, "PG_MAX_CONNS", "20")
	setEnv(t, "PG_MIN_CONNS", "5")
	setEnv(t, "PG_MAX_CONN_LIFETIME", "2h")
	setEnv(t, "PG_MAX_CONN_IDLE_TIME", "45m")

	config, err := loadPostgreSQLConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(config.DSN, "postgres://dragon:secret@localhost:5432/origin?sslmode=disable") {
		t.Fatalf("unexpected dsn: %s", config.DSN)
	}

	if config.MaxOpenConns != 20 || config.MaxIdleConns != 5 {
		t.Fatalf("unexpected pool sizes: max=%d min=%d", config.MaxOpenConns, config.MaxIdleConns)
	}

	if config.ConnMaxLifetime != 2*time.Hour {
		t.Fatalf("unexpected ConnMaxLifetime: %s", config.ConnMaxLifetime)
	}
}

func TestNewGetAndClosePostgreSQLPool(t *testing.T) {
	t.Run("new pool returns load config error", func(t *testing.T) {
		original := loadPostgreSQLConfigFn
		loadPostgreSQLConfigFn = func() (*postgreSQLConfig, error) {
			return nil, errors.New("config failure")
		}
		t.Cleanup(func() {
			loadPostgreSQLConfigFn = original
		})

		_, err := NewPostgreSQLPool(context.Background())
		if err == nil || !strings.Contains(err.Error(), "config failure") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("new pool returns creation error", func(t *testing.T) {
		originalLoad := loadPostgreSQLConfigFn
		originalOpen := openGormFn
		loadPostgreSQLConfigFn = func() (*postgreSQLConfig, error) {
			return &postgreSQLConfig{DSN: "postgres://test"}, nil
		}
		openGormFn = func(gorm.Dialector, ...gorm.Option) (*gorm.DB, error) {
			return nil, errors.New("create failure")
		}
		t.Cleanup(func() {
			loadPostgreSQLConfigFn = originalLoad
			openGormFn = originalOpen
		})

		_, err := NewPostgreSQLPool(context.Background())
		if err == nil || !strings.Contains(err.Error(), "create failure") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("new pool closes on ping error", func(t *testing.T) {
		var closed bool
		originalLoad := loadPostgreSQLConfigFn
		originalOpen := openGormFn
		originalSQLDB := getSQLDBFn
		originalPing := pingSQLDBFn
		originalClose := closeSQLDBFn

		loadPostgreSQLConfigFn = func() (*postgreSQLConfig, error) {
			return &postgreSQLConfig{DSN: "postgres://test"}, nil
		}
		openGormFn = func(gorm.Dialector, ...gorm.Option) (*gorm.DB, error) {
			return &gorm.DB{}, nil
		}
		getSQLDBFn = func(*gorm.DB) (*sql.DB, error) {
			return &sql.DB{}, nil
		}
		pingSQLDBFn = func(context.Context, *sql.DB) error {
			return errors.New("ping failure")
		}
		closeSQLDBFn = func(*sql.DB) error {
			closed = true
			return nil
		}
		t.Cleanup(func() {
			loadPostgreSQLConfigFn = originalLoad
			openGormFn = originalOpen
			getSQLDBFn = originalSQLDB
			pingSQLDBFn = originalPing
			closeSQLDBFn = originalClose
		})

		_, err := NewPostgreSQLPool(context.Background())
		if err == nil || !strings.Contains(err.Error(), "ping failure") {
			t.Fatalf("unexpected error: %v", err)
		}

		if !closed {
			t.Fatal("expected pool close on ping failure")
		}
	})

	t.Run("new pool succeeds", func(t *testing.T) {
		originalLoad := loadPostgreSQLConfigFn
		originalOpen := openGormFn
		originalSQLDB := getSQLDBFn
		originalPing := pingSQLDBFn

		loadPostgreSQLConfigFn = func() (*postgreSQLConfig, error) {
			return &postgreSQLConfig{DSN: "postgres://test"}, nil
		}
		openGormFn = func(gorm.Dialector, ...gorm.Option) (*gorm.DB, error) {
			return &gorm.DB{}, nil
		}
		getSQLDBFn = func(*gorm.DB) (*sql.DB, error) {
			return &sql.DB{}, nil
		}
		pingSQLDBFn = func(context.Context, *sql.DB) error {
			return nil
		}
		t.Cleanup(func() {
			loadPostgreSQLConfigFn = originalLoad
			openGormFn = originalOpen
			getSQLDBFn = originalSQLDB
			pingSQLDBFn = originalPing
		})

		pool, err := NewPostgreSQLPool(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pool == nil {
			t.Fatal("expected non-nil pool")
		}
	})

	t.Run("get pool caches singleton and close resets it", func(t *testing.T) {
		resetPoolState()

		var created int
		var closed int
		originalNew := newPostgreSQLPoolFn
		originalSQLDB := getSQLDBFn
		originalClose := closeSQLDBFn

		newPostgreSQLPoolFn = func(context.Context) (*gorm.DB, error) {
			created++
			return &gorm.DB{}, nil
		}
		getSQLDBFn = func(*gorm.DB) (*sql.DB, error) {
			return &sql.DB{}, nil
		}
		closeSQLDBFn = func(*sql.DB) error {
			closed++
			return nil
		}
		t.Cleanup(func() {
			newPostgreSQLPoolFn = originalNew
			getSQLDBFn = originalSQLDB
			closeSQLDBFn = originalClose
			resetPoolState()
		})

		first, err := GetPostgreSQLPool(context.Background())
		if err != nil || first == nil {
			t.Fatalf("unexpected result: pool=%v err=%v", first, err)
		}

		second, err := GetPostgreSQLPool(context.Background())
		if err != nil || second == nil {
			t.Fatalf("unexpected result: pool=%v err=%v", second, err)
		}

		if created != 1 {
			t.Fatalf("expected one pool creation, got %d", created)
		}

		ClosePostgreSQLPool()
		if closed != 1 {
			t.Fatalf("expected one pool close, got %d", closed)
		}

		_, err = GetPostgreSQLPool(context.Background())
		if err != nil {
			t.Fatalf("unexpected error after reset: %v", err)
		}

		if created != 2 {
			t.Fatalf("expected pool recreation after close, got %d", created)
		}
	})
}

type sqlRecorder struct {
	sql   string
	args  []any
	calls int
}

func stubExecSQL(t *testing.T, rowsAffected int64, err error) *sqlRecorder {
	t.Helper()

	recorder := &sqlRecorder{}
	original := execSQLFn
	execSQLFn = func(_ context.Context, query string, args ...any) (int64, error) {
		recorder.calls++
		recorder.sql = query
		recorder.args = args
		return rowsAffected, err
	}
	t.Cleanup(func() {
		execSQLFn = original
	})
	return recorder
}

func stubQuerySQL(t *testing.T, data any, err error) *sqlRecorder {
	t.Helper()

	recorder := &sqlRecorder{}
	original := querySQLFn
	querySQLFn = func(_ context.Context, destination any, query string, args ...any) error {
		recorder.calls++
		recorder.sql = query
		recorder.args = args
		if err != nil {
			return err
		}
		reflect.ValueOf(destination).Elem().Set(reflect.ValueOf(data))
		return nil
	}
	t.Cleanup(func() {
		querySQLFn = original
	})
	return recorder
}

func resetPoolState() {
	postgresPool = nil
	postgresSQLDB = nil
	postgresPoolErr = nil
	postgresPoolOnce = sync.Once{}
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()

	previous, existed := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Setenv(%s) error: %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if existed {
			err = os.Setenv(key, previous)
		} else {
			err = os.Unsetenv(key)
		}

		if err != nil {
			t.Fatalf("restore env %s error: %v", key, err)
		}
	})
}
