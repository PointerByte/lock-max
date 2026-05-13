// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"time"

	"github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/views"
	"github.com/google/uuid"
)

type KEK struct {
	IdCmkWrappingKeyRef uuid.UUID `json:"-"`
	SecretCmkKey        string    `json:"-"`
	PublicKey           string    `json:"PublicKey"`
	PrivateKey          string    `json:"-"`
	KeyRef              string    `json:"KeyRef"`
	Provider            string    `json:"Provider"`
	Version             string    `json:"Version"`
}

type Pagination struct {
	TotalRegisters     uint `json:"totalRegisters"`
	TotalPages         uint `json:"totalPages"`
	TotalRegistersPage uint `json:"totalRegistersPage"`
	PageNow            uint `json:"pageNow"`
}

type PaginatedKEK struct {
	Results    []KEK
	Pagination Pagination
}

type PaginatedCmkKey struct {
	Results    []CmkKey
	Pagination Pagination
}

type PaginatedCreationKeyQueue struct {
	Results    []views.CmkCreationKeyQueueView
	Pagination Pagination
}

type StatusResponse struct {
	ID      uuid.UUID `json:"id"`
	Healthz string    `json:"healthz"`
	Version string    `json:"version"`
}

type KeyVersionInfo struct {
	IDCmkKey        uuid.UUID               `json:"id_cmk_key"`
	IDCmkKeyVersion uuid.UUID               `json:"id_cmk_key_version"`
	SecretCmkKey    string                  `json:"secret_cmk_key"`
	VersionNumber   int                     `json:"version_number"`
	Size            int                     `json:"size"`
	PublicKey       string                  `json:"public_key"`
	Status          common.KeyVersionStatus `json:"status"`
	Algorithm       common.KeyType          `json:"algorithm"`
	Purpose         common.KeyPurpose       `json:"purpose"`
	IsCurrent       bool                    `json:"is_current"`
}

type CreateKeyInput struct {
	IDCmkKey  *uuid.UUID
	Algorithm common.KeyType
	Size      uint
	Purpose   common.KeyPurpose
	Version   uint
}

type CmkKey struct {
	CmkKey views.CmkKeyView              `json:"CmkKey"`
	Queue  views.CmkCreationKeyQueueView `json:"Queue"`
}

type ClientCredentials struct {
	ClientID     string
	ClientSecret string
}

type AuthToken struct {
	Token     string
	TokenType string
	ExpiresAt time.Time
}

type CreateAPIClientInput struct {
	ClientID     string
	ClientSecret string
	Description  string
}

type APIClient struct {
	IDAPIClient      uuid.UUID `json:"id_api_client"`
	ClientIDHash     string    `json:"client_id_hash"`
	ClientSecretHash string    `json:"-"`
	Description      string    `json:"description,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type PaginatedAPIClients struct {
	Results    []APIClient
	Pagination Pagination
}
