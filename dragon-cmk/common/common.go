// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const Timeout = 5 * time.Minute

const (
	EnvKekLocalEncryptSecret = "KEK_LOCAL_ENCRYPT_SECRET"
	EnvServiceClientID       = "CMK_SERVICE_CLIENT_ID"
	EnvServiceClientSecret   = "CMK_SERVICE_CLIENT_SECRET"
	EnvJWTTTL                = "CMK_JWT_TTL"
	EnvVaultMode             = "VAULT_MODE"
	EnvWorkersLimit          = "WORKERS_LIMIT"
	EnvPGUser                = "PGUSER"
	EnvPGPassword            = "PGPASSWORD"
	EnvPGHost                = "PGHOST"
	EnvPGPort                = "PGPORT"
	EnvPGDatabase            = "PGDATABASE"
	EnvPGSSLMode             = "PGSSLMODE"
	EnvPGSchema              = "PGSHEMA"
	EnvPGMaxConnections      = "PG_MAX_CONNS"
	EnvPGMinConnections      = "PG_MIN_CONNS"
	EnvPGMaxConnLifetime     = "PG_MAX_CONN_LIFETIME"
	EnvPGMaxConnIdleTime     = "PG_MAX_CONN_IDLE_TIME"
)

const defaultJWTTTL = time.Hour

func ServiceClientID() string {
	return EnvString(EnvServiceClientID)
}

func ServiceClientSecret() string {
	return EnvString(EnvServiceClientSecret)
}

func JWTAlgorithm() string {
	return strings.TrimSpace(viper.GetString("jwt.algorithm"))
}

func JWTPrivateKeyPath() string {
	return strings.TrimSpace(viper.GetString("jwt.eddsa.private_key"))
}

func JWTPublicKeyPath() string {
	return strings.TrimSpace(viper.GetString("jwt.eddsa.public_key"))
}

func JWTTTL() time.Duration {
	ttl := EnvDuration(EnvJWTTTL, defaultJWTTTL)
	if ttl > 0 {
		return ttl
	}
	return defaultJWTTTL
}

func KekLocalEncryptSecret() string {
	return EnvString(EnvKekLocalEncryptSecret)
}

func VaultMode() string {
	return EnvString(EnvVaultMode)
}

func WorkerLimit() (int, bool, error) {
	value, ok := EnvLookup(EnvWorkersLimit)
	if !ok {
		return 0, false, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, true, err
	}
	return limit, true, nil
}

func PostgreSQLDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		EnvString(EnvPGUser),
		EnvString(EnvPGPassword),
		EnvString(EnvPGHost),
		EnvString(EnvPGPort),
		EnvString(EnvPGDatabase),
		EnvString(EnvPGSSLMode),
	)
}

func PostgreSQLSchema() string {
	return EnvString(EnvPGSchema)
}

func PostgreSQLMaxConnections(fallback int) int {
	return EnvInt(EnvPGMaxConnections, fallback)
}

func PostgreSQLMinConnections(fallback int) int {
	return EnvInt(EnvPGMinConnections, fallback)
}

func PostgreSQLMaxConnLifetime(fallback time.Duration) time.Duration {
	return EnvDuration(EnvPGMaxConnLifetime, fallback)
}

func PostgreSQLMaxConnIdleTime(fallback time.Duration) time.Duration {
	return EnvDuration(EnvPGMaxConnIdleTime, fallback)
}

func EnvString(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func EnvLookup(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}

func EnvInt(key string, fallback int) int {
	value, ok := EnvLookup(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func EnvDuration(key string, fallback time.Duration) time.Duration {
	value, ok := EnvLookup(key)
	if !ok {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
