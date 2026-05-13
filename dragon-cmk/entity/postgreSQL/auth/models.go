// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"time"

	"github.com/google/uuid"
)

type APIClient struct {
	IDAPIClient      uuid.UUID `db:"id_api_client" gorm:"column:id_api_client" json:"id_api_client"`
	ClientIDHash     string    `db:"client_id_hash" gorm:"column:client_id_hash" json:"client_id_hash"`
	ClientSecretHash string    `db:"client_secret_hash" gorm:"column:client_secret_hash" json:"-"`
	Description      string    `db:"description" gorm:"column:description" json:"description,omitempty"`
	CreatedAt        time.Time `db:"created_at" gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" gorm:"column:updated_at" json:"updated_at"`
}

func (APIClient) TableName() string {
	return "dragon_cmk.api_client"
}
