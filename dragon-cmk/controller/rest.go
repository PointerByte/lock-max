// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	serverGin "github.com/PointerByte/GoForge/config/server/gin"
	middlewaresLogger "github.com/PointerByte/GoForge/logger/middlewares"
	middlewaresSecurity "github.com/PointerByte/GoForge/security/middlewares"
	_ "github.com/PointerByte/lock-max/dragon-cmk/api"
	"github.com/PointerByte/lock-max/dragon-cmk/controller/models"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	cmk "github.com/PointerByte/lock-max/dragon-cmk/service/CMK"
	kek "github.com/PointerByte/lock-max/dragon-cmk/service/KEK"
	auth "github.com/PointerByte/lock-max/dragon-cmk/service/auth"
	serviceModels "github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type RESTService struct {
	keyRepository      cmk.IRepository
	wrappingRepository kek.IRepository
	authRepository     auth.IRepository
}

var getRouteFn = serverGin.GetRoute

type restError struct {
	status  int
	message string
}

func (e restError) Error() string {
	return e.message
}

func badRequestError(message string) error {
	return restError{status: http.StatusBadRequest, message: message}
}

func RegisterRESTRoutes(keyRepository cmk.IRepository, wrappingRepository kek.IRepository, authRepositories ...auth.IRepository) {
	var authRepository auth.IRepository
	if len(authRepositories) > 0 {
		authRepository = authRepositories[0]
	}
	service := &RESTService{
		keyRepository:      keyRepository,
		wrappingRepository: wrappingRepository,
		authRepository:     authRepository,
	}

	docs := getRouteFn("/api")
	docs.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api/docs/doc.json")))

	authRoutes := getRouteFn("/api/auth/v1")
	authRoutes.POST("/service-token", service.CreateServiceToken)
	authRoutes.Use(middlewaresSecurity.RequireJWT())
	authRoutes.POST("/token", service.CreateAPIToken)
	authRoutes.POST("/clients", service.CreateAPIClient)
	authRoutes.GET("/clients/list", service.ListAPIClients)
	authRoutes.GET("/clients/:client_id", service.GetAPIClient)
	authRoutes.DELETE("/clients", service.DeleteAPIClient)

	keys := getRouteFn("/api/keys/v1")
	keys.Use(middlewaresSecurity.RequireJWT())
	keys.POST("/enable", service.EnableKey)
	keys.POST("/disable", service.DisableKey)
	keys.POST("/schedule-deletion", service.ScheduleKeyDeletion)
	keys.POST("/pending-deletion", service.PendingDeletion)
	keys.POST("/cancel-deletion", service.CancelKeyDeletion)
	keys.POST("/unavailable", service.UnavailableDelete)
	keys.DELETE("/", service.DeleteKey)
	keys.GET("/versions/:version_id", service.GetKeyVersion)
	keys.PATCH("/versions/:version_id/status", service.UpdateKeyVersionStatus)
	keys.GET("/list", service.ListCmkKeys)
	keys.GET("/creation-queues/list", service.ListCreationKeyQueues)

	keks := getRouteFn("/api/config/v1")
	keks.Use(middlewaresSecurity.RequireJWT())
	keks.POST("/create", service.CreateKEK)
	keks.GET("/list", service.ListKEK)
	keks.GET("/:id", service.GetKEK)
	keks.POST("/rotate", service.RotateKEK)
	keks.DELETE("/", service.DeleteKEK)
}

// CreateServiceToken creates a JWT for service APIs using HTTP Basic credentials.
// @Summary Create service API token
// @Description Creates a JWT signed with the configured service certificates. The request must use HTTP Basic auth with client_id/client_secret from environment variables.
// @Tags Auth
// @Produce json
// @Param Authorization header string true "Basic base64(client_id:client_secret)"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 401 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Router /api/auth/v1/service-token [post]
func (s *RESTService) CreateServiceToken(c *gin.Context) {
	if !requireAuthRepository(c, s.authRepository) {
		return
	}
	token, err := s.authRepository.CreateServiceToken(c.Request.Context(), c.GetHeader("Authorization"))
	if err != nil {
		writeRESTAuthError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "token created", tokenResponseModel(token))
}

// CreateAPIToken creates a JWT for a persisted API client.
// @Summary Create API client token
// @Description Validates client_id/client_secret against the hashed API client table and returns a JWT.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.TokenRequest true "Client credentials"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 401 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/auth/v1/token [post]
func (s *RESTService) CreateAPIToken(c *gin.Context) {
	if !requireAuthRepository(c, s.authRepository) {
		return
	}
	var request models.TokenRequest
	if !bindJSON(c, &request) {
		return
	}
	token, err := s.authRepository.CreateAPIToken(c.Request.Context(), serviceModels.ClientCredentials{
		ClientID:     request.ClientID,
		ClientSecret: request.ClientSecret,
	})
	if err != nil {
		writeRESTAuthError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "token created", tokenResponseModel(token))
}

// CreateAPIClient creates a client allowed to request API tokens.
// @Summary Create API client
// @Description Stores HMAC hashes for client_id and client_secret using the global CMK key as HMAC secret.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.CreateAPIClientRequest true "API client"
// @Success 201 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/auth/v1/clients [post]
func (s *RESTService) CreateAPIClient(c *gin.Context) {
	if !requireAuthRepository(c, s.authRepository) {
		return
	}
	var request models.CreateAPIClientRequest
	if !bindJSON(c, &request) {
		return
	}
	client, err := s.authRepository.CreateAPIClient(c.Request.Context(), serviceModels.CreateAPIClientInput{
		ClientID:     request.ClientID,
		ClientSecret: request.ClientSecret,
		Description:  request.Description,
	})
	if err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusCreated, "api client created", models.APIClientResponse{APIClient: apiClientModel(client)})
}

// ListAPIClients lists registered API clients.
// @Summary List API clients
// @Description Lists API clients with pagination. Secrets are never returned.
// @Tags Auth
// @Produce json
// @Param page query int true "Page number"
// @Param totalResgisterPage query int true "Total registers per page"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/auth/v1/clients/list [get]
func (s *RESTService) ListAPIClients(c *gin.Context) {
	if !requireAuthRepository(c, s.authRepository) {
		return
	}
	var request models.ListAPIClientsRequest
	if !bindQuery(c, &request) {
		return
	}
	result, err := s.authRepository.ListAPIClients(c.Request.Context(), request.Page, request.TotalRegisterPage)
	if err != nil {
		writeRESTError(c, err)
		return
	}
	items := make([]models.APIClientInfo, 0, len(result.Results))
	for _, client := range result.Results {
		item := client
		items = append(items, apiClientModel(&item))
	}
	writeRESTPaginatedSuccess(c, http.StatusOK, "api clients listed", items, paginationModel(result.Pagination))
}

// GetAPIClient gets an API client by raw client_id.
// @Summary Get API client
// @Description Hashes the provided client_id and returns the matching API client metadata.
// @Tags Auth
// @Produce json
// @Param client_id path string true "Client id"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/auth/v1/clients/{client_id} [get]
func (s *RESTService) GetAPIClient(c *gin.Context) {
	if !requireAuthRepository(c, s.authRepository) {
		return
	}
	client, err := s.authRepository.GetAPIClient(c.Request.Context(), c.Param("client_id"))
	if err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "api client found", models.APIClientResponse{APIClient: apiClientModel(client)})
}

// DeleteAPIClient deletes an API client by raw client_id.
// @Summary Delete API client
// @Description Deletes a client after hashing the provided client_id with the global CMK key.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.DeleteAPIClientRequest true "Client id"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/auth/v1/clients [delete]
func (s *RESTService) DeleteAPIClient(c *gin.Context) {
	if !requireAuthRepository(c, s.authRepository) {
		return
	}
	var request models.DeleteAPIClientRequest
	if !bindJSON(c, &request) {
		return
	}
	if err := s.authRepository.DeleteAPIClient(c.Request.Context(), request.ClientID); err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "api client deleted", models.OperationResponse{Status: operationStatusModel("api client deleted")})
}

// EnableKey enables a customer managed key.
// @Summary Enable a CMK
// @Description Enables a customer managed key by secret_cmk_key or key_id/version_id.
// @Tags CMK Keys
// @Accept json
// @Produce json
// @Param request body models.KeyReferenceRequest true "Key reference"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/enable [post]
func (s *RESTService) EnableKey(c *gin.Context) {
	request, ok := bindKeyRequest(c)
	if !ok {
		return
	}
	secret := secretCmkKey(request.SecretCmkKey, request.KeyID, request.VersionID)
	if !requireSecret(c, secret) {
		return
	}
	if err := s.keyRepository.EnableKey(secret); err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "key enabled", models.KeyResponse{Key: keyInfoModel(secret, request.KeyID, request.VersionID, string(commonEntity.KeyStatusEnabled))})
}

// DisableKey disables a customer managed key.
// @Summary Disable a CMK
// @Description Disables a customer managed key by secret_cmk_key or key_id/version_id.
// @Tags CMK Keys
// @Accept json
// @Produce json
// @Param request body models.KeyReferenceRequest true "Key reference"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/disable [post]
func (s *RESTService) DisableKey(c *gin.Context) {
	request, ok := bindKeyRequest(c)
	if !ok {
		return
	}
	secret := secretCmkKey(request.SecretCmkKey, request.KeyID, request.VersionID)
	if !requireSecret(c, secret) {
		return
	}
	if err := s.keyRepository.DisableKey(secret); err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "key disabled", models.KeyResponse{Key: keyInfoModel(secret, request.KeyID, request.VersionID, string(commonEntity.KeyStatusDisabled))})
}

// ScheduleKeyDeletion schedules deletion for a customer managed key.
// @Summary Schedule CMK deletion
// @Description Marks a customer managed key for deletion after the pending window.
// @Tags CMK Keys
// @Accept json
// @Produce json
// @Param request body models.ScheduleKeyDeletionRequest true "Deletion schedule"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/schedule-deletion [post]
func (s *RESTService) ScheduleKeyDeletion(c *gin.Context) {
	var request models.ScheduleKeyDeletionRequest
	if !bindJSON(c, &request) {
		return
	}
	if request.PendingWindowDays <= 0 {
		writeRESTError(c, badRequestError("pending_window_days must be greater than zero"))
		return
	}
	secret := secretCmkKey(request.SecretCmkKey, request.KeyID, request.VersionID)
	if !requireSecret(c, secret) {
		return
	}
	if err := s.keyRepository.ScheduleKeyDeletion(secret, uint(request.PendingWindowDays)); err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "key deletion scheduled", models.KeyResponse{Key: keyInfoModel(secret, request.KeyID, request.VersionID, string(commonEntity.KeyStatusPendingDeletion))})
}

// PendingDeletion marks a customer managed key as pending deletion.
// @Summary Mark CMK as pending deletion
// @Description Moves a customer managed key into pending deletion by secret_cmk_key or key_id/version_id.
// @Tags CMK Keys
// @Accept json
// @Produce json
// @Param request body models.KeyReferenceRequest true "Key reference"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/pending-deletion [post]
func (s *RESTService) PendingDeletion(c *gin.Context) {
	request, ok := bindKeyRequest(c)
	if !ok {
		return
	}
	secret := secretCmkKey(request.SecretCmkKey, request.KeyID, request.VersionID)
	if !requireSecret(c, secret) {
		return
	}
	if err := s.keyRepository.PendingDeletion(secret); err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "key marked pending deletion", models.KeyResponse{Key: keyInfoModel(secret, request.KeyID, request.VersionID, string(commonEntity.KeyStatusPendingDeletion))})
}

// CancelKeyDeletion cancels deletion for a customer managed key.
// @Summary Cancel CMK deletion
// @Description Cancels pending deletion for a customer managed key.
// @Tags CMK Keys
// @Accept json
// @Produce json
// @Param request body models.KeyReferenceRequest true "Key reference"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/cancel-deletion [post]
func (s *RESTService) CancelKeyDeletion(c *gin.Context) {
	request, ok := bindKeyRequest(c)
	if !ok {
		return
	}
	secret := secretCmkKey(request.SecretCmkKey, request.KeyID, request.VersionID)
	if !requireSecret(c, secret) {
		return
	}
	if err := s.keyRepository.CancelKeyDeletion(secret); err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "key deletion canceled", models.KeyResponse{Key: keyInfoModel(secret, request.KeyID, request.VersionID, string(commonEntity.KeyStatusDisabled))})
}

// UnavailableDelete marks a customer managed key as unavailable.
// @Summary Mark CMK unavailable
// @Description Moves a customer managed key into unavailable state by secret_cmk_key or key_id/version_id.
// @Tags CMK Keys
// @Accept json
// @Produce json
// @Param request body models.KeyReferenceRequest true "Key reference"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/unavailable [post]
func (s *RESTService) UnavailableDelete(c *gin.Context) {
	request, ok := bindKeyRequest(c)
	if !ok {
		return
	}
	secret := secretCmkKey(request.SecretCmkKey, request.KeyID, request.VersionID)
	if !requireSecret(c, secret) {
		return
	}
	if err := s.keyRepository.UnavailableDelete(secret); err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "key marked unavailable", models.KeyResponse{Key: keyInfoModel(secret, request.KeyID, request.VersionID, string(commonEntity.KeyStatusUnavailable))})
}

// DeleteKey deletes a customer managed key.
// @Summary Delete a CMK
// @Description Deletes a customer managed key using the JSON request body.
// @Tags CMK Keys
// @Accept json
// @Produce json
// @Param request body models.DeleteKeyRequest true "Delete key request"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/ [delete]
func (s *RESTService) DeleteKey(c *gin.Context) {
	var request models.DeleteKeyRequest
	if !bindJSON(c, &request) {
		return
	}
	secret := secretCmkKey(request.Request.SecretCmkKey, request.Request.KeyID, request.Request.VersionID)
	if !requireSecret(c, secret) {
		return
	}
	if err := s.keyRepository.DeleteKey(secret); err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "key deleted", models.OperationResponse{Status: operationStatusModel("key deleted")})
}

// GetKeyVersion gets non-sensitive data for a CMK key version.
// @Summary Get CMK key version
// @Description Gets secret_cmk_key and non-sensitive version data by version id.
// @Tags CMK Keys
// @Produce json
// @Param version_id path string true "CMK key version id"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/versions/{version_id} [get]
func (s *RESTService) GetKeyVersion(c *gin.Context) {
	idCmkKeyVersion, err := requiredRESTUUID(c.Param("version_id"), "version_id")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	version, err := s.keyRepository.GetKeyVersionInfo(idCmkKeyVersion)
	if err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "key version found", models.KeyVersionResponse{KeyVersion: keyVersionInfoModel(version)})
}

// UpdateKeyVersionStatus updates a non-current CMK key version status.
// @Summary Update CMK key version status
// @Description Updates a specific non-current key version status. pendingDeletion is not accepted.
// @Tags CMK Keys
// @Accept json
// @Produce json
// @Param version_id path string true "CMK key version id"
// @Param request body models.UpdateKeyVersionStatusRequest true "Status update"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/versions/{version_id}/status [patch]
func (s *RESTService) UpdateKeyVersionStatus(c *gin.Context) {
	idCmkKeyVersion, err := requiredRESTUUID(c.Param("version_id"), "version_id")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	var request models.UpdateKeyVersionStatusRequest
	if !bindJSON(c, &request) {
		return
	}
	status, err := keyVersionStatus(request.Status)
	if err != nil {
		writeRESTError(c, err)
		return
	}
	version, err := s.keyRepository.UpdateKeyVersionStatus(idCmkKeyVersion, status)
	if err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "key version status updated", models.KeyVersionResponse{KeyVersion: keyVersionInfoModel(version)})
}

// ListCmkKeys lists CMK keys by KEK id.
// @Summary List CMK keys
// @Description Lists CMK keys that belong to a KEK id.
// @Tags CMK Keys
// @Produce json
// @Param id_kek query string true "KEK id"
// @Param page query int true "Page number"
// @Param totalResgisterPage query int true "Total registers per page"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/list [get]
func (s *RESTService) ListCmkKeys(c *gin.Context) {
	var request models.ListCmkKeysRequest
	if !bindQuery(c, &request) {
		return
	}
	idKek, err := optionalRESTUUID(request.IDCmkWrappingKeyRef, "id_kek")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	result, err := s.keyRepository.ListCmkKey(idKek, request.Page, request.TotalRegisterPage)
	if err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTPaginatedSuccess(c, http.StatusOK, "cmk keys listed", result.Results, paginationModel(result.Pagination))
}

// ListCreationKeyQueues lists CMK creation key queue records.
// @Summary List CMK creation key queues
// @Description Lists records from dragon_cmk.cmk_creation_key_queue with pagination.
// @Tags CMK Keys
// @Produce json
// @Param id_cmk_key query string false "CMK key id"
// @Param status query string false "Queue status"
// @Param page query int true "Page number"
// @Param totalResgisterPage query int true "Total registers per page"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/keys/v1/creation-queues/list [get]
func (s *RESTService) ListCreationKeyQueues(c *gin.Context) {
	var request models.ListCreationKeyQueuesRequest
	if !bindQuery(c, &request) {
		return
	}
	idCmkKey, err := optionalRESTUUID(request.IDCmkKey, "id_cmk_key")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	status, err := optionalQueueStatus(request.Status)
	if err != nil {
		writeRESTError(c, err)
		return
	}
	result, err := s.keyRepository.ListCreationKeyQueues(idCmkKey, status, request.Page, request.TotalRegisterPage)
	if err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTPaginatedSuccess(c, http.StatusOK, "creation key queues listed", result.Results, paginationModel(result.Pagination))
}

// CreateKEK creates a wrapping key reference.
// @Summary Create wrapping key reference
// @Description Creates a wrapping key reference for CMK encryption.
// @Tags KEK Config
// @Accept json
// @Produce json
// @Param request body models.CreateKEKRequest true "Wrapping key data"
// @Success 201 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/config/v1/create [post]
func (s *RESTService) CreateKEK(c *gin.Context) {
	var request models.CreateKEKRequest
	if !bindJSON(c, &request) {
		return
	}
	idGenerate, err := optionalRESTUUID(request.IDGenerate, "id_generate")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	idKek, err := s.wrappingRepository.CreateKEK(idGenerate, "", request.Salt)
	if err != nil {
		writeRESTError(c, err)
		return
	}
	wrappingKey, err := s.wrappingRepository.GetKEK(*idKek, "")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusCreated, "wrapping key created", models.WrappingKeyOperationResponse{
		Status:      operationStatusModel("wrapping key created"),
		WrappingKey: wrappingKeyInfoModel(wrappingKey),
	})
}

// GetKEK gets a wrapping key reference from the path.
// @Summary Get wrapping key reference
// @Description Gets a wrapping key reference by id_generate and optional version.
// @Tags KEK Config
// @Produce json
// @Param id path string true "Generate UUID"
// @Param version query string false "Wrapping key version"
// @Success 200 {object} models.RESTResponse
// @Success 204 "Wrapping key not found"
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/config/v1/{id} [get]
func (s *RESTService) GetKEK(c *gin.Context) {
	idGenerate, err := optionalRESTUUID(restID(c), "id_generate")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	wrappingKey, err := s.wrappingRepository.GetKEK(idGenerate, c.Query("version"))
	if err != nil {
		writeRESTError(c, err)
		return
	}
	if isEmptyKEK(wrappingKey) {
		writeREST204(c, "wrapping key not found")
		return
	}
	writeRESTSuccess(c, http.StatusOK, "wrapping key found", models.WrappingKeyResponse{WrappingKey: wrappingKeyInfoModel(wrappingKey)})
}

// RotateKEK rotates a wrapping key reference from request data.
// @Summary Rotate wrapping key reference
// @Description Rotates a wrapping key reference using the request id or creates a new id when omitted.
// @Tags KEK Config
// @Accept json
// @Produce json
// @Param request body models.RotateKEKRequest true "Rotation data"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/config/v1/rotate [post]
func (s *RESTService) RotateKEK(c *gin.Context) {
	var request models.RotateKEKRequest
	if !bindJSON(c, &request) {
		return
	}
	idGenerate, err := optionalRESTUUID(request.IDGenerate, "id_generate")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	idKek, err := s.wrappingRepository.RotateKEK(idGenerate, request.Salt)
	if err != nil {
		writeRESTError(c, err)
		return
	}
	wrappingKey, err := s.wrappingRepository.GetKEK(*idKek, "")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "wrapping key rotated", models.WrappingKeyOperationResponse{
		Status:      operationStatusModel("wrapping key rotated"),
		WrappingKey: wrappingKeyInfoModel(wrappingKey),
	})
}

// DeleteKEK deletes a wrapping key reference from request data.
// @Summary Delete wrapping key reference
// @Description Deletes a wrapping key reference using the JSON request body.
// @Tags KEK Config
// @Accept json
// @Produce json
// @Param request body models.DeleteKEKRequest true "Delete data"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/config/v1/ [delete]
func (s *RESTService) DeleteKEK(c *gin.Context) {
	var request models.DeleteKEKRequest
	if !bindJSON(c, &request) {
		return
	}
	id := request.IDCmkWrappingKeyRef
	if pathID := c.Param("id"); pathID != "" {
		id = pathID
	}
	version := request.Version

	idCmkWrappingKeyRef, err := optionalRESTUUID(id, "id_kek")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	if err := s.wrappingRepository.DeleteKey(idCmkWrappingKeyRef, version); err != nil {
		writeRESTError(c, err)
		return
	}
	writeRESTSuccess(c, http.StatusOK, "wrapping key deleted", models.OperationResponse{Status: operationStatusModel("wrapping key deleted")})
}

// ListKEK lists wrapping keys by generate id.
// @Summary List wrapping keys
// @Description Lists wrapping key versions for an id_generate.
// @Tags KEK Config
// @Produce json
// @Param id_generate query string true "Generate id"
// @Param page query int true "Page number"
// @Param totalResgisterPage query int true "Total registers per page"
// @Success 200 {object} models.RESTResponse
// @Failure 400 {object} models.RESTResponse
// @Failure 500 {object} models.RESTResponse
// @Security BearerAuth
// @Router /api/config/v1/list [get]
func (s *RESTService) ListKEK(c *gin.Context) {
	var request models.ListKEKRequest
	if !bindQuery(c, &request) {
		return
	}
	idGenerate, err := optionalRESTUUID(request.IDGenerate, "id_generate")
	if err != nil {
		writeRESTError(c, err)
		return
	}
	result, err := s.wrappingRepository.ListKEK(idGenerate, request.Page, request.TotalRegisterPage)
	if err != nil {
		writeRESTError(c, err)
		return
	}
	items := make([]models.WrappingKeyInfo, 0, len(result.Results))
	for _, kekData := range result.Results {
		item := kekData
		items = append(items, wrappingKeyInfoModel(&item))
	}
	writeRESTPaginatedSuccess(c, http.StatusOK, "wrapping keys listed", items, paginationModel(result.Pagination))
}

func bindKeyRequest(c *gin.Context) (models.KeyReferenceRequest, bool) {
	var request models.KeyReferenceRequest
	if !bindJSON(c, &request) {
		return models.KeyReferenceRequest{}, false
	}
	return request, true
}

func bindJSON(c *gin.Context, destination any) bool {
	if err := c.ShouldBindJSON(destination); err != nil {
		writeRESTError(c, badRequestError(err.Error()))
		return false
	}
	return true
}

func bindQuery(c *gin.Context, destination any) bool {
	if err := c.ShouldBindQuery(destination); err != nil {
		writeRESTError(c, badRequestError(err.Error()))
		return false
	}
	return true
}

func requireSecret(c *gin.Context, secret string) bool {
	if secret != "" {
		return true
	}
	writeRESTError(c, badRequestError("secret_cmk_key or key_id/version_id is required"))
	return false
}

func restID(c *gin.Context) string {
	if id := c.Param("id"); id != "" {
		return id
	}
	if id := c.Query("id_kek"); id != "" {
		return id
	}
	return c.Query("id_cmk_wrapping_key_ref")
}

func optionalRESTUUID(value string, field string) (uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, badRequestError("invalid " + field)
	}
	return id, nil
}

func requiredRESTUUID(value string, field string) (uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return uuid.Nil, badRequestError(field + " is required")
	}
	return optionalRESTUUID(value, field)
}

func optionalQueueStatus(value string) (commonEntity.QueueStatus, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	status := commonEntity.QueueStatus(value)
	switch status {
	case commonEntity.QueueStatusPending,
		commonEntity.QueueStatusProcessing,
		commonEntity.QueueStatusProcessed,
		commonEntity.QueueStatusFailed:
		return status, nil
	default:
		return "", badRequestError("invalid status")
	}
}

func keyVersionStatus(value string) (commonEntity.KeyVersionStatus, error) {
	status := commonEntity.KeyVersionStatus(strings.TrimSpace(value))
	if status == commonEntity.KeyVersionStatus("pendingDeletion") {
		return "", badRequestError("pendingDeletion cannot be used to update a key version")
	}
	switch status {
	case commonEntity.KeyVersionStatusEnabled,
		commonEntity.KeyVersionStatusDisabled,
		commonEntity.KeyVersionStatusRetired,
		commonEntity.KeyVersionStatusUnavailable:
		return status, nil
	default:
		return "", badRequestError("invalid status")
	}
}

func writeRESTError(c *gin.Context, err error) {
	code := http.StatusInternalServerError
	var restErr restError
	if errors.As(err, &restErr) {
		code = restErr.status
	}
	middlewaresLogger.PrintError(c, err)
	c.JSON(code, restResponse(models.ErrorResponse{Error: err.Error()}, nil))
}

func writeRESTAuthError(c *gin.Context, err error) {
	message := err.Error()
	if strings.Contains(message, "authorization") || strings.Contains(message, "credentials") {
		writeRESTError(c, restError{status: http.StatusUnauthorized, message: message})
		return
	}
	writeRESTError(c, err)
}

func requireAuthRepository(c *gin.Context, repository auth.IRepository) bool {
	if repository != nil {
		return true
	}
	writeRESTError(c, errors.New("auth repository is not configured"))
	return false
}

func writeRESTSuccess(c *gin.Context, code int, msg string, resp any) {
	writeRESTPaginatedSuccess(c, code, msg, resp, nil)
}

func writeRESTPaginatedSuccess(c *gin.Context, code int, msg string, resp any, pagination *models.Pagination) {
	middlewaresLogger.PrintInfo(c, msg)
	c.JSON(code, restResponse(resp, pagination))
}

func writeREST204(c *gin.Context, msg string) {
	middlewaresLogger.PrintInfo(c, msg)
	c.Header("X-Service-Name", serviceName())
	c.Header("X-Service-Version", serviceVersion())
	c.Header("X-Results-Message", msg)
	c.Status(http.StatusNoContent)
}

func restResponse(results any, pagination *models.Pagination) models.RESTResponse {
	return models.RESTResponse{
		ServiceName:    serviceName(),
		ServiceVersion: serviceVersion(),
		Results:        results,
		Pagination:     pagination,
	}
}

func serviceName() string {
	return viper.GetString("app.name")
}

func serviceVersion() string {
	return viper.GetString("app.version")
}

func paginationModel(pagination serviceModels.Pagination) *models.Pagination {
	return &models.Pagination{
		TotalRegisters:     pagination.TotalRegisters,
		TotalPages:         pagination.TotalPages,
		TotalRegistersPage: pagination.TotalRegistersPage,
		PageNow:            pagination.PageNow,
	}
}

func operationStatusModel(message string) models.OperationStatus {
	return models.OperationStatus{Success: true, Message: message}
}

func tokenResponseModel(token *serviceModels.AuthToken) models.TokenResponse {
	if token == nil {
		return models.TokenResponse{}
	}
	return models.TokenResponse{
		Token:     token.Token,
		TokenType: token.TokenType,
		ExpiresAt: token.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

func apiClientModel(client *serviceModels.APIClient) models.APIClientInfo {
	if client == nil {
		return models.APIClientInfo{}
	}
	return models.APIClientInfo{
		IDAPIClient:  client.IDAPIClient.String(),
		ClientIDHash: client.ClientIDHash,
		Description:  client.Description,
		CreatedAt:    client.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    client.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func keyInfoModel(secret string, keyID string, versionID string, state string) models.KeyInfo {
	key := models.KeyInfo{
		KeyID:        keyID,
		VersionID:    versionID,
		SecretCmkKey: secret,
		State:        state,
	}
	if key.KeyID == "" || key.VersionID == "" {
		secretKeyID, secretVersionID := secretCmkKeyParts(secret)
		if key.KeyID == "" {
			key.KeyID = secretKeyID
		}
		if key.VersionID == "" {
			key.VersionID = secretVersionID
		}
	}
	return key
}

func wrappingKeyInfoModel(kekData *serviceModels.KEK) models.WrappingKeyInfo {
	if kekData == nil {
		return models.WrappingKeyInfo{}
	}
	return models.WrappingKeyInfo{
		IDCmkWrappingKeyRef: kekData.IdCmkWrappingKeyRef.String(),
		Provider:            kekData.Provider,
		KeyRef:              kekData.KeyRef,
		Version:             kekData.Version,
		PublicKey:           kekData.PublicKey,
	}
}

func keyVersionInfoModel(version *serviceModels.KeyVersionInfo) models.KeyVersionInfo {
	if version == nil {
		return models.KeyVersionInfo{}
	}
	return models.KeyVersionInfo{
		KeyID:         version.IDCmkKey.String(),
		VersionID:     version.IDCmkKeyVersion.String(),
		SecretCmkKey:  version.SecretCmkKey,
		VersionNumber: version.VersionNumber,
		Size:          version.Size,
		PublicKey:     version.PublicKey,
		Status:        string(version.Status),
		Algorithm:     string(version.Algorithm),
		Purpose:       string(version.Purpose),
		IsCurrent:     version.IsCurrent,
	}
}

func isEmptyKEK(kekData *serviceModels.KEK) bool {
	return kekData == nil ||
		(kekData.IdCmkWrappingKeyRef == uuid.Nil &&
			kekData.Provider == "" &&
			kekData.KeyRef == "" &&
			kekData.Version == "" &&
			kekData.PublicKey == "")
}
