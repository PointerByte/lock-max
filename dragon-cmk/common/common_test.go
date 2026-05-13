// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestPostgreSQLDSN(t *testing.T) {
	t.Setenv(EnvPGUser, "dragon")
	t.Setenv(EnvPGPassword, "secret")
	t.Setenv(EnvPGHost, "localhost")
	t.Setenv(EnvPGPort, "5432")
	t.Setenv(EnvPGDatabase, "origin")
	t.Setenv(EnvPGSSLMode, "disable")

	dsn := PostgreSQLDSN()
	if !strings.Contains(dsn, "postgres://dragon:secret@localhost:5432/origin?sslmode=disable") {
		t.Fatalf("unexpected dsn: %s", dsn)
	}
}

func TestEnvInt(t *testing.T) {
	t.Setenv("TEST_INT", "")
	if got := EnvInt("TEST_INT", 11); got != 11 {
		t.Fatalf("unexpected int fallback: %d", got)
	}

	t.Setenv("TEST_INT", "bad")
	if got := EnvInt("TEST_INT", 12); got != 12 {
		t.Fatalf("unexpected int fallback for invalid value: %d", got)
	}

	t.Setenv("TEST_INT", "13")
	if got := EnvInt("TEST_INT", 0); got != 13 {
		t.Fatalf("unexpected int value: %d", got)
	}
}

func TestEnvDuration(t *testing.T) {
	t.Setenv("TEST_DURATION", "")
	if got := EnvDuration("TEST_DURATION", time.Second); got != time.Second {
		t.Fatalf("unexpected duration fallback: %s", got)
	}

	t.Setenv("TEST_DURATION", "bad")
	if got := EnvDuration("TEST_DURATION", 2*time.Second); got != 2*time.Second {
		t.Fatalf("unexpected duration fallback for invalid value: %s", got)
	}

	t.Setenv("TEST_DURATION", "3s")
	if got := EnvDuration("TEST_DURATION", 0); got != 3*time.Second {
		t.Fatalf("unexpected duration value: %s", got)
	}
}

func TestWorkerLimit(t *testing.T) {
	t.Setenv(EnvWorkersLimit, "")
	if _, ok, err := WorkerLimit(); ok || err != nil {
		t.Fatalf("unexpected unset worker limit: ok=%v err=%v", ok, err)
	}

	t.Setenv(EnvWorkersLimit, "bad")
	if _, ok, err := WorkerLimit(); !ok || err == nil {
		t.Fatalf("expected invalid worker limit error, ok=%v err=%v", ok, err)
	}

	t.Setenv(EnvWorkersLimit, "25")
	limit, ok, err := WorkerLimit()
	if err != nil || !ok || limit != 25 {
		t.Fatalf("unexpected worker limit: limit=%d ok=%v err=%v", limit, ok, err)
	}
}

func TestJWTConfigReadsViper(t *testing.T) {
	originalAlgorithm := viper.Get("jwt.algorithm")
	originalPrivateKey := viper.Get("jwt.eddsa.private_key")
	originalPublicKey := viper.Get("jwt.eddsa.public_key")
	t.Cleanup(func() {
		viper.Set("jwt.algorithm", originalAlgorithm)
		viper.Set("jwt.eddsa.private_key", originalPrivateKey)
		viper.Set("jwt.eddsa.public_key", originalPublicKey)
	})

	viper.Set("jwt.algorithm", "EDDSA")
	viper.Set("jwt.eddsa.private_key", "./certs/jwt/key.pem")
	viper.Set("jwt.eddsa.public_key", "./certs/jwt/public.pem")

	if got := JWTAlgorithm(); got != "EDDSA" {
		t.Fatalf("unexpected jwt algorithm: %s", got)
	}
	if got := JWTPrivateKeyPath(); got != "./certs/jwt/key.pem" {
		t.Fatalf("unexpected private key path: %s", got)
	}
	if got := JWTPublicKeyPath(); got != "./certs/jwt/public.pem" {
		t.Fatalf("unexpected public key path: %s", got)
	}
}
