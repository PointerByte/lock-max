// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"strings"

	"github.com/PointerByte/GoForge/encrypt"
	encryptLocal "github.com/PointerByte/GoForge/encrypt/local"
	"github.com/PointerByte/GoForge/logger/builder"
	cmk "github.com/PointerByte/lock-max/dragon-cmk/service/CMK"
	kek "github.com/PointerByte/lock-max/dragon-cmk/service/KEK"
	auth "github.com/PointerByte/lock-max/dragon-cmk/service/auth"
)

func NewRepositories(ctx context.Context, ctxLogger *builder.Context) (cmk.IRepository, kek.IRepository, auth.IRepository) {
	kekRepository := kek.NewRepository()
	encryptRepository := encrypt.NewRepository(encryptLocal.NewRepository())
	cmkRepository := cmk.NewRepository(
		ctx,
		ctxLogger,
		encryptRepository,
		kekRepository,
	)
	authRepository := auth.NewRepository(ctx, ctxLogger, encryptRepository, kekRepository)
	return cmkRepository, kekRepository, authRepository
}

func secretCmkKey(secret string, keyID string, versionID string) string {
	if secret != "" {
		return secret
	}
	if keyID == "" || versionID == "" {
		return ""
	}
	return keyID + "." + versionID
}

func secretCmkKeyParts(secret string) (keyID string, versionID string) {
	parts := strings.Split(secret, ".")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
