// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
)

//go:generate mockgen -source=interface.go -destination=mock_repository.go -package=auth

type IRepository interface {
	CreateServiceToken(ctx context.Context, authorization string) (*models.AuthToken, error)
	CreateAPIToken(ctx context.Context, input models.ClientCredentials) (*models.AuthToken, error)
	CreateAPIClient(ctx context.Context, input models.CreateAPIClientInput) (*models.APIClient, error)
	ListAPIClients(ctx context.Context, page uint, totalRegisterPage uint) (*models.PaginatedAPIClients, error)
	GetAPIClient(ctx context.Context, clientID string) (*models.APIClient, error)
	DeleteAPIClient(ctx context.Context, clientID string) error
}
