// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/PointerByte/GoForge/encrypt"
	"github.com/PointerByte/GoForge/logger/builder"
	jwtservice "github.com/PointerByte/GoForge/security/auth/jwt"
	appCommon "github.com/PointerByte/lock-max/dragon-cmk/common"
	entityAuth "github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/auth"
	kek "github.com/PointerByte/lock-max/dragon-cmk/service/KEK"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

const (
	tokenTypeBearer = "Bearer"

	errMsgAuthorizationRequired       = "authorization header is required"
	errMsgAuthorizationMustBeBasic    = "authorization header must use Basic scheme"
	errMsgInvalidBasicAuthorization   = "invalid basic authorization"
	errMsgInvalidClientCredentials    = "invalid client credentials"
	errMsgClientCredentialsRequired   = "client_id and client_secret are required"
	errMsgClientIDRequired            = "client_id is required"
	errMsgClientSecretRequired        = "client_secret is required"
	errMsgGlobalCMKKeyRequired        = "global CMK key is required"
	errMsgPageRequired                = "page must be greater than zero"
	errMsgTotalRegisterPageRequired   = "totalResgisterPage must be greater than zero"
	errMsgJWTSigningConfigUnavailable = "jwt signing configuration is unavailable"
)

type Repository struct {
	ctx      context.Context
	store    entityAuth.IRepository
	hash     encrypt.HashRepository
	wrapping kek.IRepository
	now      func() time.Time
}

func NewRepository(
	ctx context.Context,
	ctxLogger *builder.Context,
	hash encrypt.HashRepository,
	wrapping kek.IRepository,
) IRepository {
	return NewRepositoryWithStore(ctx, entityAuth.NewRepository(ctx, ctxLogger), hash, wrapping)
}

func NewRepositoryWithStore(
	ctx context.Context,
	store entityAuth.IRepository,
	hash encrypt.HashRepository,
	wrapping kek.IRepository,
) IRepository {
	return &Repository{
		ctx:      ctx,
		store:    store,
		hash:     hash,
		wrapping: wrapping,
		now:      time.Now,
	}
}

func (r *Repository) CreateServiceToken(ctx context.Context, authorization string) (*models.AuthToken, error) {
	clientID, clientSecret, err := parseBasicAuthorization(authorization)
	if err != nil {
		return nil, err
	}
	if !constantTimeEqual(clientID, appCommon.ServiceClientID()) ||
		!constantTimeEqual(clientSecret, appCommon.ServiceClientSecret()) {
		return nil, errors.New(errMsgInvalidClientCredentials)
	}

	return r.createToken(ctx, clientID)
}

func (r *Repository) CreateAPIToken(ctx context.Context, input models.ClientCredentials) (*models.AuthToken, error) {
	if err := validateCredentials(input.ClientID, input.ClientSecret); err != nil {
		return nil, err
	}

	clientIDHash, clientSecretHash, err := r.hashCredentials(ctx, input.ClientID, input.ClientSecret)
	if err != nil {
		return nil, err
	}
	client, err := r.store.GetAPIClientByClientIDHash(clientIDHash)
	if err != nil {
		return nil, errors.New(errMsgInvalidClientCredentials)
	}
	if !constantTimeEqual(client.ClientSecretHash, clientSecretHash) {
		return nil, errors.New(errMsgInvalidClientCredentials)
	}

	return r.createToken(ctx, input.ClientID)
}

func (r *Repository) CreateAPIClient(ctx context.Context, input models.CreateAPIClientInput) (*models.APIClient, error) {
	if err := validateCredentials(input.ClientID, input.ClientSecret); err != nil {
		return nil, err
	}

	clientIDHash, clientSecretHash, err := r.hashCredentials(ctx, input.ClientID, input.ClientSecret)
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	if err := r.store.CreateAPIClient(entityAuth.CreateAPIClientInput{
		IDAPIClient:      id,
		ClientIDHash:     clientIDHash,
		ClientSecretHash: clientSecretHash,
		Description:      strings.TrimSpace(input.Description),
	}); err != nil {
		return nil, err
	}

	client, err := r.store.GetAPIClientByClientIDHash(clientIDHash)
	if err != nil {
		return &models.APIClient{
			IDAPIClient:  id,
			ClientIDHash: clientIDHash,
			Description:  strings.TrimSpace(input.Description),
			CreatedAt:    r.now().UTC(),
			UpdatedAt:    r.now().UTC(),
		}, nil
	}
	return apiClientModel(client), nil
}

func (r *Repository) ListAPIClients(_ context.Context, page uint, totalRegisterPage uint) (*models.PaginatedAPIClients, error) {
	if err := validatePagination(page, totalRegisterPage); err != nil {
		return nil, err
	}
	total, err := r.store.CountAPIClients()
	if err != nil {
		return nil, err
	}
	clients, err := r.store.ListAPIClients(page, totalRegisterPage)
	if err != nil {
		return nil, err
	}

	results := make([]models.APIClient, 0, len(clients))
	for _, client := range clients {
		results = append(results, *apiClientModel(&client))
	}

	return &models.PaginatedAPIClients{
		Results:    results,
		Pagination: newPagination(total, page, totalRegisterPage),
	}, nil
}

func (r *Repository) GetAPIClient(ctx context.Context, clientID string) (*models.APIClient, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, errors.New(errMsgClientIDRequired)
	}
	clientIDHash, err := r.hashValue(ctx, clientID)
	if err != nil {
		return nil, err
	}
	client, err := r.store.GetAPIClientByClientIDHash(clientIDHash)
	if err != nil {
		return nil, err
	}
	return apiClientModel(client), nil
}

func (r *Repository) DeleteAPIClient(ctx context.Context, clientID string) error {
	if strings.TrimSpace(clientID) == "" {
		return errors.New(errMsgClientIDRequired)
	}
	clientIDHash, err := r.hashValue(ctx, clientID)
	if err != nil {
		return err
	}
	return r.store.DeleteAPIClient(clientIDHash)
}

func parseBasicAuthorization(authorization string) (string, string, error) {
	authorization = strings.TrimSpace(authorization)
	if authorization == "" {
		return "", "", errors.New(errMsgAuthorizationRequired)
	}
	if !strings.HasPrefix(authorization, "Basic ") {
		return "", "", errors.New(errMsgAuthorizationMustBeBasic)
	}

	payload := strings.TrimSpace(strings.TrimPrefix(authorization, "Basic "))
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", "", errors.New(errMsgInvalidBasicAuthorization)
	}
	clientID, clientSecret, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return "", "", errors.New(errMsgInvalidBasicAuthorization)
	}
	if err := validateCredentials(clientID, clientSecret); err != nil {
		return "", "", err
	}
	return clientID, clientSecret, nil
}

func validateCredentials(clientID string, clientSecret string) error {
	switch {
	case strings.TrimSpace(clientID) == "" && strings.TrimSpace(clientSecret) == "":
		return errors.New(errMsgClientCredentialsRequired)
	case strings.TrimSpace(clientID) == "":
		return errors.New(errMsgClientIDRequired)
	case strings.TrimSpace(clientSecret) == "":
		return errors.New(errMsgClientSecretRequired)
	default:
		return nil
	}
}

func (r *Repository) createToken(ctx context.Context, subject string) (*models.AuthToken, error) {
	service, err := newConfiguredJWTService()
	if err != nil {
		return nil, err
	}

	now := r.now().UTC()
	expiresAt := now.Add(appCommon.JWTTTL())
	token, err := service.CreateWithContext(ctx, map[string]any{
		"sub": subject,
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
		"iss": "dragon-cmk",
		"aud": "dragon-cmk-api",
	})
	if err != nil {
		return nil, err
	}

	return &models.AuthToken{
		Token:     token,
		TokenType: tokenTypeBearer,
		ExpiresAt: expiresAt,
	}, nil
}

func newConfiguredJWTService() (*jwtservice.Service, error) {
	algorithm := strings.TrimSpace(appCommon.JWTAlgorithm())
	privateKey := strings.TrimSpace(appCommon.JWTPrivateKeyPath())
	publicKey := strings.TrimSpace(appCommon.JWTPublicKeyPath())
	if algorithm == "" || privateKey == "" || publicKey == "" {
		return nil, errors.New(errMsgJWTSigningConfigUnavailable)
	}

	viper.Set("jwt.algorithm", algorithm)
	viper.Set("jwt.eddsa.private_key", privateKey)
	viper.Set("jwt.eddsa.public_key", publicKey)
	return jwtservice.NewConfiguredService(jwtservice.ConfigServiceInput{
		Algorithm: algorithm,
	})
}

func (r *Repository) hashCredentials(ctx context.Context, clientID string, clientSecret string) (string, string, error) {
	clientIDHash, err := r.hashValue(ctx, clientID)
	if err != nil {
		return "", "", err
	}
	clientSecretHash, err := r.hashValue(ctx, clientSecret)
	if err != nil {
		return "", "", err
	}
	return clientIDHash, clientSecretHash, nil
}

func (r *Repository) hashValue(ctx context.Context, value string) (string, error) {
	secret, err := r.globalCMKSecret()
	if err != nil {
		return "", err
	}
	if r.hash == nil {
		return "", errors.New(errMsgGlobalCMKKeyRequired)
	}
	if ctx == nil {
		ctx = r.ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return r.hash.HMAC(ctx, secret, value), nil
}

func (r *Repository) globalCMKSecret() (string, error) {
	if r.wrapping == nil {
		return "", errors.New(errMsgGlobalCMKKeyRequired)
	}
	globalKEK, err := r.wrapping.GetKEK(uuid.Nil, "")
	if err != nil {
		return "", err
	}
	if globalKEK == nil || strings.TrimSpace(globalKEK.SecretCmkKey) == "" {
		return "", errors.New(errMsgGlobalCMKKeyRequired)
	}
	return globalKEK.SecretCmkKey, nil
}

func constantTimeEqual(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return hmac.Equal([]byte(left), []byte(right))
}

func validatePagination(page uint, totalRegisterPage uint) error {
	if page == 0 {
		return errors.New(errMsgPageRequired)
	}
	if totalRegisterPage == 0 {
		return errors.New(errMsgTotalRegisterPageRequired)
	}
	return nil
}

func newPagination(totalRegisters uint, page uint, totalRegisterPage uint) models.Pagination {
	totalPages := totalRegisters / totalRegisterPage
	if totalRegisters%totalRegisterPage != 0 {
		totalPages++
	}
	return models.Pagination{
		TotalRegisters:     totalRegisters,
		TotalPages:         totalPages,
		TotalRegistersPage: totalRegisterPage,
		PageNow:            page,
	}
}

func apiClientModel(client *entityAuth.APIClient) *models.APIClient {
	if client == nil {
		return nil
	}
	return &models.APIClient{
		IDAPIClient:      client.IDAPIClient,
		ClientIDHash:     client.ClientIDHash,
		ClientSecretHash: client.ClientSecretHash,
		Description:      client.Description,
		CreatedAt:        client.CreatedAt,
		UpdatedAt:        client.UpdatedAt,
	}
}
