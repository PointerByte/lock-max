// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package auth

import "github.com/google/uuid"

//go:generate mockgen -source=interface.go -destination=mock_repository.go -package=auth

type IRepository interface {
	CreateAPIClient(input CreateAPIClientInput) error
	GetAPIClientByClientIDHash(clientIDHash string) (*APIClient, error)
	ListAPIClients(page uint, totalRegisterPage uint) ([]APIClient, error)
	CountAPIClients() (uint, error)
	DeleteAPIClient(clientIDHash string) error
}

type CreateAPIClientInput struct {
	IDAPIClient      uuid.UUID
	ClientIDHash     string
	ClientSecretHash string
	Description      string
}
