// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package models

type KeyInfo struct {
	KeyID         string `json:"key_id,omitempty"`
	VersionID     string `json:"version_id,omitempty"`
	SecretCmkKey  string `json:"secret_cmk_key,omitempty"`
	VersionNumber uint   `json:"version_number,omitempty"`
	Size          uint   `json:"size,omitempty"`
	Algorithm     string `json:"algorithm,omitempty"`
	Purpose       string `json:"purpose,omitempty"`
	State         string `json:"state,omitempty"`
}

type KeyVersionInfo struct {
	KeyID         string `json:"key_id"`
	VersionID     string `json:"version_id"`
	SecretCmkKey  string `json:"secret_cmk_key"`
	VersionNumber int    `json:"version_number"`
	Size          int    `json:"size"`
	PublicKey     string `json:"public_key"`
	Status        string `json:"status"`
	Algorithm     string `json:"algorithm"`
	Purpose       string `json:"purpose"`
	IsCurrent     bool   `json:"is_current"`
}

type WrappingKeyInfo struct {
	IDCmkWrappingKeyRef string `json:"id_kek,omitempty"`
	Provider            string `json:"provider,omitempty"`
	KeyRef              string `json:"key_ref,omitempty"`
	Version             string `json:"version,omitempty"`
	PublicKey           string `json:"public_key,omitempty"`
}

type OperationStatus struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type RESTResponse struct {
	ServiceName    string      `json:"serviceName"`
	ServiceVersion string      `json:"serviceVersion"`
	Results        any         `json:"results,omitempty"`
	Pagination     *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	TotalRegisters     uint `json:"totalRegisters" example:"30000"`
	TotalPages         uint `json:"totalPages" example:"300"`
	TotalRegistersPage uint `json:"totalRegistersPage" example:"100"`
	PageNow            uint `json:"pageNow" example:"1"`
}

type KeyReferenceRequest struct {
	KeyID        string `json:"key_id,omitempty"`
	VersionID    string `json:"version_id,omitempty"`
	SecretCmkKey string `json:"secret_cmk_key,omitempty"`
}

type DeleteKeyRequest struct {
	Request *KeyReferenceRequest `json:"request" binding:"required"`
}

type ScheduleKeyDeletionRequest struct {
	KeyReferenceRequest
	PendingWindowDays int32 `json:"pending_window_days"`
}

type UpdateKeyVersionStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type KeyResponse struct {
	Key KeyInfo `json:"key"`
}

type KeyVersionResponse struct {
	KeyVersion KeyVersionInfo `json:"key_version"`
}

type OperationResponse struct {
	Status OperationStatus `json:"status"`
}

type CreateKEKRequest struct {
	IDGenerate string `json:"id_generate" binding:"required"`
	Salt       string `json:"salt,omitempty"`
}

type RotateKEKRequest struct {
	IDGenerate string `json:"id_generate" binding:"required"`
	Salt       string `json:"salt,omitempty"`
}

type DeleteKEKRequest struct {
	IDCmkWrappingKeyRef string `json:"id_kek,omitempty"`
	Version             string `json:"version,omitempty"`
}

type ListKEKRequest struct {
	IDGenerate        string `json:"id_generate" form:"id_generate" binding:"required"`
	Page              uint   `json:"page" form:"page" binding:"required" example:"1"`
	TotalRegisterPage uint   `json:"totalResgisterPage" form:"totalResgisterPage" binding:"required" example:"100"`
}

type ListCmkKeysRequest struct {
	IDCmkWrappingKeyRef string `json:"id_kek" form:"id_kek" binding:"required"`
	Page                uint   `json:"page" form:"page" binding:"required" example:"1"`
	TotalRegisterPage   uint   `json:"totalResgisterPage" form:"totalResgisterPage" binding:"required" example:"100"`
}

type ListCreationKeyQueuesRequest struct {
	IDCmkKey          string `json:"id_cmk_key" form:"id_cmk_key"`
	Status            string `json:"status" form:"status"`
	Page              uint   `json:"page" form:"page" binding:"required" example:"1"`
	TotalRegisterPage uint   `json:"totalResgisterPage" form:"totalResgisterPage" binding:"required" example:"100"`
}

type WrappingKeyResponse struct {
	WrappingKey WrappingKeyInfo `json:"wrapping_key"`
}

type WrappingKeyOperationResponse struct {
	Status      OperationStatus `json:"status"`
	WrappingKey WrappingKeyInfo `json:"wrapping_key,omitempty"`
}

type TokenRequest struct {
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
}

type TokenResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresAt string `json:"expires_at"`
}

type APIClientInfo struct {
	IDAPIClient  string `json:"id_api_client"`
	ClientIDHash string `json:"client_id_hash"`
	Description  string `json:"description,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type CreateAPIClientRequest struct {
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	Description  string `json:"description,omitempty"`
}

type DeleteAPIClientRequest struct {
	ClientID string `json:"client_id" binding:"required"`
}

type ListAPIClientsRequest struct {
	Page              uint `json:"page" form:"page" binding:"required" example:"1"`
	TotalRegisterPage uint `json:"totalResgisterPage" form:"totalResgisterPage" binding:"required" example:"100"`
}

type APIClientResponse struct {
	APIClient APIClientInfo `json:"api_client"`
}
