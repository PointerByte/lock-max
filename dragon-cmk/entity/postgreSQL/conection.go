package postgreSQL

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/PointerByte/lock-max/dragon-cmk/common"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	defaultPostgresMaxConnections  = int32(200)
	defaultPostgresMinConnections  = int32(10)
	defaultPostgresMaxConnLifetime = time.Hour
	defaultPostgresMaxConnIdleTime = 30 * time.Minute
)

var (
	postgresPool     *gorm.DB
	postgresSQLDB    *sql.DB
	postgresPoolOnce sync.Once
	postgresPoolErr  error

	loadPostgreSQLConfigFn = loadPostgreSQLConfig
	openGormFn             = gorm.Open
	postgresDialectorFn    = postgres.Open
	getSQLDBFn             = func(db *gorm.DB) (*sql.DB, error) { return db.DB() }
	pingSQLDBFn            = func(ctx context.Context, db *sql.DB) error { return db.PingContext(ctx) }
	closeSQLDBFn           = func(db *sql.DB) error { return db.Close() }
	newPostgreSQLPoolFn    = NewPostgreSQLPool
)

// NewPostgreSQLPool creates a GORM-backed PostgreSQL connection pool using environment
// variables and validates connectivity with a ping.
func NewPostgreSQLPool(ctx context.Context) (*gorm.DB, error) {
	config, err := loadPostgreSQLConfigFn()
	if err != nil {
		return nil, err
	}

	db, err := openGormFn(postgresDialectorFn(config.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("create postgresql gorm db: %w", err)
	}

	sqlDB, err := getSQLDBFn(db)
	if err != nil {
		return nil, fmt.Errorf("get postgresql sql db: %w", err)
	}
	configureSQLDB(sqlDB, config)

	if err := pingSQLDBFn(ctx, sqlDB); err != nil {
		_ = closeSQLDBFn(sqlDB)
		return nil, fmt.Errorf("ping postgresql: %w", err)
	}
	return db, nil
}

// GetPostgreSQLPool returns a singleton GORM-backed PostgreSQL connection for the
// application lifetime.
func GetPostgreSQLPool(ctx context.Context) (*gorm.DB, error) {
	postgresPoolOnce.Do(func() {
		postgresPool, postgresPoolErr = newPostgreSQLPoolFn(ctx)
		if postgresPoolErr == nil {
			postgresSQLDB, postgresPoolErr = getSQLDBFn(postgresPool)
		}
	})
	return postgresPool, postgresPoolErr
}

// ClosePostgreSQLPool closes the shared PostgreSQL connection when it is
// no longer needed.
func ClosePostgreSQLPool() {
	if postgresSQLDB != nil {
		_ = closeSQLDBFn(postgresSQLDB)
		postgresSQLDB = nil
	}

	postgresPool = nil
	postgresPoolErr = nil
	postgresPoolOnce = sync.Once{}
}

// RestartPostgreSQLPool closes the current shared pool and creates a new one.
func RestartPostgreSQLPool() error {
	ClosePostgreSQLPool()

	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()
	pool, err := newPostgreSQLPoolFn(ctx)
	if err != nil {
		return err
	}

	postgresPool = pool
	postgresSQLDB, postgresPoolErr = getSQLDBFn(pool)
	if postgresPoolErr != nil {
		postgresPool = nil
		return postgresPoolErr
	}
	postgresPoolErr = nil
	postgresPoolOnce = sync.Once{}
	postgresPoolOnce.Do(func() {})
	return nil
}

type postgreSQLConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func loadPostgreSQLConfig() (*postgreSQLConfig, error) {
	return &postgreSQLConfig{
		DSN:             common.PostgreSQLDSN(),
		MaxOpenConns:    common.PostgreSQLMaxConnections(int(defaultPostgresMaxConnections)),
		MaxIdleConns:    common.PostgreSQLMinConnections(int(defaultPostgresMinConnections)),
		ConnMaxLifetime: common.PostgreSQLMaxConnLifetime(defaultPostgresMaxConnLifetime),
		ConnMaxIdleTime: common.PostgreSQLMaxConnIdleTime(defaultPostgresMaxConnIdleTime),
	}, nil
}

func configureSQLDB(db *sql.DB, config *postgreSQLConfig) {
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}
