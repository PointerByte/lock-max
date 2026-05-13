// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	"github.com/PointerByte/GoForge/logger/builder"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	pb "github.com/PointerByte/lock-max/dragon-cmk/proto"
	cmk "github.com/PointerByte/lock-max/dragon-cmk/service/CMK"
	auth "github.com/PointerByte/lock-max/dragon-cmk/service/auth"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type KeyService struct {
	pb.UnimplementedKeyServiceServer
	repository     cmk.IRepository
	authRepository auth.IRepository
}

const OperationSuccess = "Operation success"

func NewGRPCServices(ctx context.Context, ctxLogger *builder.Context) pb.KeyServiceServer {
	cmkRepository, _, authRepository := NewRepositories(ctx, ctxLogger)
	return NewKeyService(cmkRepository, authRepository)
}

func NewKeyService(repository cmk.IRepository, authRepositories ...auth.IRepository) pb.KeyServiceServer {
	var authRepository auth.IRepository
	if len(authRepositories) > 0 {
		authRepository = authRepositories[0]
	}
	return &KeyService{repository: repository, authRepository: authRepository}
}

func (s *KeyService) CreateServiceToken(ctx context.Context, _ *pb.CreateServiceTokenRequest) (*pb.TokenResponse, error) {
	ctxLogger := builder.New(ctx)
	if err := requireGRPCAuthRepository(s.authRepository); err != nil {
		ctxLogger.Error(err)
		return nil, err
	}
	token, err := s.authRepository.CreateServiceToken(ctx, authorizationFromIncomingContext(ctx))
	if err != nil {
		ctxLogger.Error(err)
		return nil, authServiceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return tokenResponse(token), nil
}

func (s *KeyService) CreateAPIToken(ctx context.Context, request *pb.CreateAPITokenRequest) (*pb.TokenResponse, error) {
	ctxLogger := builder.New(ctx)
	if err := requireGRPCAuthRepository(s.authRepository); err != nil {
		ctxLogger.Error(err)
		return nil, err
	}
	token, err := s.authRepository.CreateAPIToken(ctx, models.ClientCredentials{
		ClientID:     request.GetClientId(),
		ClientSecret: request.GetClientSecret(),
	})
	if err != nil {
		ctxLogger.Error(err)
		return nil, authServiceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return tokenResponse(token), nil
}

func (s *KeyService) CreateAPIClient(ctx context.Context, request *pb.CreateAPIClientRequest) (*pb.CreateAPIClientResponse, error) {
	ctxLogger := builder.New(ctx)
	if err := requireGRPCAuthRepository(s.authRepository); err != nil {
		ctxLogger.Error(err)
		return nil, err
	}
	client, err := s.authRepository.CreateAPIClient(ctx, models.CreateAPIClientInput{
		ClientID:     request.GetClientId(),
		ClientSecret: request.GetClientSecret(),
		Description:  request.GetDescription(),
	})
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.CreateAPIClientResponse{ApiClient: apiClientInfo(client)}, nil
}

func (s *KeyService) ListAPIClients(ctx context.Context, request *pb.ListAPIClientsRequest) (*pb.ListAPIClientsResponse, error) {
	ctxLogger := builder.New(ctx)
	if err := requireGRPCAuthRepository(s.authRepository); err != nil {
		ctxLogger.Error(err)
		return nil, err
	}
	response, err := s.authRepository.ListAPIClients(ctx, uint(request.GetPage()), uint(request.GetTotalResgisterPage()))
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	results := make([]*pb.APIClientInfo, 0, len(response.Results))
	for _, item := range response.Results {
		client := item
		results = append(results, apiClientInfo(&client))
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.ListAPIClientsResponse{
		Results: results,
		Pagination: &pb.Pagination{
			TotalRegisters:     uint64(response.Pagination.TotalRegisters),
			TotalPages:         uint64(response.Pagination.TotalPages),
			TotalRegistersPage: uint64(response.Pagination.TotalRegistersPage),
			PageNow:            uint64(response.Pagination.PageNow),
		},
	}, nil
}

func (s *KeyService) GetAPIClient(ctx context.Context, request *pb.GetAPIClientRequest) (*pb.CreateAPIClientResponse, error) {
	ctxLogger := builder.New(ctx)
	if err := requireGRPCAuthRepository(s.authRepository); err != nil {
		ctxLogger.Error(err)
		return nil, err
	}
	client, err := s.authRepository.GetAPIClient(ctx, request.GetClientId())
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.CreateAPIClientResponse{ApiClient: apiClientInfo(client)}, nil
}

func (s *KeyService) DeleteAPIClient(ctx context.Context, request *pb.DeleteAPIClientRequest) (*pb.OperationStatus, error) {
	ctxLogger := builder.New(ctx)
	if err := requireGRPCAuthRepository(s.authRepository); err != nil {
		ctxLogger.Error(err)
		return nil, err
	}
	if err := s.authRepository.DeleteAPIClient(ctx, request.GetClientId()); err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return operationStatus("api client deleted"), nil
}

func (s *KeyService) Status(ctx context.Context, _ *pb.StatusRequest) (*pb.StatusResponse, error) {
	ctxLogger := builder.New(ctx)
	response, err := s.repository.Status()
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.StatusResponse{
		Version: response.Version,
		Healthz: response.Healthz,
		KeyId:   response.ID.String(),
	}, nil
}

func (s *KeyService) ListCmkKeys(ctx context.Context, request *pb.ListCmkKeysRequest) (*pb.ListCmkKeysResponse, error) {
	ctxLogger := builder.New(ctx)
	idKek, err := parseUUID(request.GetIdKek(), "id_kek")
	if err != nil {
		ctxLogger.Error(err)
		return nil, err
	}
	response, err := s.repository.ListCmkKey(idKek, uint(request.GetPage()), uint(request.GetTotalResgisterPage()))
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}

	results := make([]*pb.CmkKeyRecord, 0, len(response.Results))
	for _, item := range response.Results {
		key := item.CmkKey
		versionID := ""
		if key.IDCmkKeyVersion != nil {
			versionID = key.IDCmkKeyVersion.String()
		}
		results = append(results, &pb.CmkKeyRecord{
			IdCmkKey:        key.IDCmkKey.String(),
			IdCmkKeyVersion: versionID,
			Algorithm:       keyAlgorithmProto(key.Algorithm),
			Purpose:         keyPurposeProto(key.Purpose),
			State:           keyStateProto(key.Status),
		})
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.ListCmkKeysResponse{
		Results: results,
		Pagination: &pb.Pagination{
			TotalRegisters:     uint64(response.Pagination.TotalRegisters),
			TotalPages:         uint64(response.Pagination.TotalPages),
			TotalRegistersPage: uint64(response.Pagination.TotalRegistersPage),
			PageNow:            uint64(response.Pagination.PageNow),
		},
	}, nil
}

func (s *KeyService) CreateKey(ctx context.Context, request *pb.CreateKeyRequest) (*pb.CreateKeyResponse, error) {
	ctxLogger := builder.New(ctx)
	algorithm, err := keyAlgorithm(request.GetAlgorithm())
	if err != nil {
		ctxLogger.Error(err)
		return nil, err
	}
	purpose, err := keyPurpose(request.GetPurpose())
	if err != nil {
		ctxLogger.Error(err)
		return nil, err
	}
	eventType, err := keyEventType(request.GetEventType())
	if err != nil {
		ctxLogger.Error(err)
		return nil, err
	}

	input := models.CreateKeyInput{
		Algorithm: algorithm,
		Size:      uint(request.GetSize()),
		Purpose:   purpose,
		Version:   uint(request.GetVersion()),
	}

	secret, err := s.repository.CreateKey(input, eventType)
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	key := keyInfoFromSecret(secret, pb.KeyState_KEY_STATE_ACTIVE)
	key.Algorithm = request.GetAlgorithm()
	key.Purpose = request.GetPurpose()
	key.Size = request.GetSize()
	key.VersionNumber = request.GetVersion()
	ctxLogger.Info(OperationSuccess)
	return &pb.CreateKeyResponse{SecretCmkKey: secret, Key: key}, nil
}

func (s *KeyService) RotateKey(ctx context.Context, request *pb.RotateKeyRequest) (*pb.RotateKeyResponse, error) {
	ctxLogger := builder.New(ctx)
	secret, err := s.repository.RotateKey(request.GetSecretCmkKey())
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.RotateKeyResponse{
		SecretCmkKey: secret,
		NewKey:       keyInfoFromSecret(secret, pb.KeyState_KEY_STATE_ACTIVE),
	}, nil
}

func (s *KeyService) RotateWrapKey(ctx context.Context, request *pb.RotateWrapKeyRequest) (*pb.RotateWrapKeyResponse, error) {
	ctxLogger := builder.New(ctx)
	id, err := parseUUID(request.GetIdCmkWrappingKeyRef(), "id_cmk_wrapping_key_ref")
	if err != nil {
		ctxLogger.Error(err)
		return nil, err
	}
	if err := s.repository.RotateWrapKey(id); err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.RotateWrapKeyResponse{
		Status: operationStatus("wrapping key rotated"),
		WrappingKey: &pb.WrappingKeyInfo{
			IdCmkWrappingKeyRef: id.String(),
		},
	}, nil
}

func (s *KeyService) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	ctxLogger := builder.New(ctx)
	ciphertext, annotations, err := s.repository.Encrypt(
		request.GetSecretCmkKey(),
		string(request.GetPlaintext()),
		optionalString(request.GetAdditional()),
	)
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.EncryptResponse{
		Ciphertext:  []byte(ciphertext),
		KeyId:       requestKeyID(request.GetUid(), request.GetSecretCmkKey()),
		Annotations: annotations,
	}, nil
}

func (s *KeyService) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	ctxLogger := builder.New(ctx)
	plaintext, annotations, err := s.repository.Decrypt(
		request.GetSecretCmkKey(),
		string(request.GetCiphertext()),
		optionalString(request.GetAdditional()),
	)
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.DecryptResponse{
		Plaintext:   []byte(plaintext),
		Annotations: annotations,
	}, nil
}

func (s *KeyService) Sing(ctx context.Context, request *pb.SingRequest) (*pb.SingResponse, error) {
	ctxLogger := builder.New(ctx)
	signature, err := s.repository.Sing(request.GetSecretCmkKey(), string(request.GetMessage()))
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.SingResponse{
		Signature: []byte(signature),
		KeyId:     requestKeyID(request.GetUid(), request.GetSecretCmkKey()),
	}, nil
}

func (s *KeyService) Verify(ctx context.Context, request *pb.VerifyRequest) (*pb.VerifyResponse, error) {
	ctxLogger := builder.New(ctx)
	valid, err := s.repository.Verify(
		request.GetSecretCmkKey(),
		string(request.GetMessage()),
		string(request.GetSignature()),
	)
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.VerifyResponse{
		Valid: valid,
		KeyId: requestKeyID(request.GetUid(), request.GetSecretCmkKey()),
	}, nil
}

func (s *KeyService) CreateJWT(ctx context.Context, request *pb.CreateJWTRequest) (*pb.CreateJWTResponse, error) {
	ctxLogger := builder.New(ctx)
	claims := map[string]any{}
	if request.GetClaims() != nil {
		claims = request.GetClaims().AsMap()
	}
	token, err := s.repository.CreateJWT(ctx, request.GetSecretCmkKey(), request.GetAlgorithm(), claims)
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.CreateJWTResponse{Token: token}, nil
}

func (s *KeyService) VerifyJWT(ctx context.Context, request *pb.VerifyJWTRequest) (*pb.VerifyJWTResponse, error) {
	ctxLogger := builder.New(ctx)
	if err := s.repository.VerifyJWT(ctx, request.GetSecretCmkKey(), request.GetAlgorithm(), request.GetToken()); err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.VerifyJWTResponse{Valid: true}, nil
}

func (s *KeyService) ReadJWT(ctx context.Context, request *pb.ReadJWTRequest) (*pb.ReadJWTResponse, error) {
	ctxLogger := builder.New(ctx)
	claims := map[string]any{}
	if err := s.repository.ReadJWT(ctx, request.GetSecretCmkKey(), request.GetAlgorithm(), request.GetToken(), claims); err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	structClaims, err := structpb.NewStruct(claims)
	if err != nil {
		ctxLogger.Error(err)
		return nil, serviceError(err)
	}
	ctxLogger.Info(OperationSuccess)
	return &pb.ReadJWTResponse{Claims: structClaims}, nil
}

func keyAlgorithm(value pb.KeyAlgorithm) (commonEntity.KeyType, error) {
	switch value {
	case pb.KeyAlgorithm_KEY_ALGORITHM_SYMMETRIC_DEFAULT:
		return commonEntity.KeySymmetricDefault, nil
	case pb.KeyAlgorithm_KEY_ALGORITHM_RSA_OAEP:
		return commonEntity.KeyTypeRSAOAEP, nil
	case pb.KeyAlgorithm_KEY_ALGORITHM_RSA_PKCS1V15_SHA256:
		return commonEntity.KeyTypeRSAPKCS1v15SHA256, nil
	case pb.KeyAlgorithm_KEY_ALGORITHM_ECDH:
		return commonEntity.KeyTypeECDH, nil
	case pb.KeyAlgorithm_KEY_ALGORITHM_EDDSA:
		return commonEntity.KeyTypeEdDSA, nil
	default:
		return "", grpcstatus.Error(codes.InvalidArgument, "unsupported key algorithm")
	}
}

func keyPurpose(value pb.KeyPurpose) (commonEntity.KeyPurpose, error) {
	switch value {
	case pb.KeyPurpose_KEY_PURPOSE_SIGN:
		return commonEntity.KeyPurposeSign, nil
	case pb.KeyPurpose_KEY_PURPOSE_ENCRYPT:
		return commonEntity.KeyPurposeEncrypt, nil
	case pb.KeyPurpose_KEY_PURPOSE_WRAP:
		return commonEntity.KeyPurposeWrap, nil
	default:
		return "", grpcstatus.Error(codes.InvalidArgument, "unsupported key purpose")
	}
}

func keyAlgorithmProto(value commonEntity.KeyType) pb.KeyAlgorithm {
	switch value {
	case commonEntity.KeySymmetricDefault:
		return pb.KeyAlgorithm_KEY_ALGORITHM_SYMMETRIC_DEFAULT
	case commonEntity.KeyTypeRSAOAEP:
		return pb.KeyAlgorithm_KEY_ALGORITHM_RSA_OAEP
	case commonEntity.KeyTypeRSAPKCS1v15SHA256:
		return pb.KeyAlgorithm_KEY_ALGORITHM_RSA_PKCS1V15_SHA256
	case commonEntity.KeyTypeECDH:
		return pb.KeyAlgorithm_KEY_ALGORITHM_ECDH
	case commonEntity.KeyTypeEdDSA:
		return pb.KeyAlgorithm_KEY_ALGORITHM_EDDSA
	default:
		return pb.KeyAlgorithm_KEY_ALGORITHM_UNSPECIFIED
	}
}

func keyPurposeProto(value commonEntity.KeyPurpose) pb.KeyPurpose {
	switch value {
	case commonEntity.KeyPurposeSign:
		return pb.KeyPurpose_KEY_PURPOSE_SIGN
	case commonEntity.KeyPurposeEncrypt:
		return pb.KeyPurpose_KEY_PURPOSE_ENCRYPT
	case commonEntity.KeyPurposeWrap:
		return pb.KeyPurpose_KEY_PURPOSE_WRAP
	default:
		return pb.KeyPurpose_KEY_PURPOSE_UNSPECIFIED
	}
}

func keyStateProto(value commonEntity.KeyStatus) pb.KeyState {
	switch value {
	case commonEntity.KeyStatusEnabled:
		return pb.KeyState_KEY_STATE_ACTIVE
	case commonEntity.KeyStatusDisabled:
		return pb.KeyState_KEY_STATE_DISABLED
	case commonEntity.KeyStatusPendingDeletion:
		return pb.KeyState_KEY_STATE_PENDING_DELETION
	case commonEntity.KeyStatusPendingImport:
		return pb.KeyState_KEY_STATE_PENDING_IMPORT
	case commonEntity.KeyStatusUnavailable:
		return pb.KeyState_KEY_STATE_UNAVAILABLE
	default:
		return pb.KeyState_KEY_STATE_UNSPECIFIED
	}
}

func keyEventType(value pb.KeyEventType) (commonEntity.EventType, error) {
	switch value {
	case pb.KeyEventType_KEY_EVENT_TYPE_UNSPECIFIED:
		return "", nil
	case pb.KeyEventType_KEY_EVENT_TYPE_CREATE_KEY:
		return commonEntity.EventTypeCreateKey, nil
	case pb.KeyEventType_KEY_EVENT_TYPE_ROTATE_KEY:
		return commonEntity.EventTypeRotateKey, nil
	default:
		return "", grpcstatus.Error(codes.InvalidArgument, "unsupported key event type")
	}
}

func parseUUID(value string, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, grpcstatus.Error(codes.InvalidArgument, fmt.Sprintf("invalid %s", field))
	}
	return id, nil
}

func operationStatus(message string) *pb.OperationStatus {
	return &pb.OperationStatus{Success: true, Message: message}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func serviceError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := grpcstatus.FromError(err); ok {
		return err
	}
	return grpcstatus.Error(codes.Internal, err.Error())
}

func authServiceError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if message == "authorization header is required" ||
		message == "authorization header must use Basic scheme" ||
		message == "invalid basic authorization" ||
		message == "invalid client credentials" {
		return grpcstatus.Error(codes.Unauthenticated, message)
	}
	if message == "client_id and client_secret are required" ||
		message == "client_id is required" ||
		message == "client_secret is required" {
		return grpcstatus.Error(codes.InvalidArgument, message)
	}
	return serviceError(err)
}

func requireGRPCAuthRepository(repository auth.IRepository) error {
	if repository != nil {
		return nil
	}
	return grpcstatus.Error(codes.Internal, "auth repository is not configured")
}

func authorizationFromIncomingContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func tokenResponse(token *models.AuthToken) *pb.TokenResponse {
	if token == nil {
		return &pb.TokenResponse{}
	}
	return &pb.TokenResponse{
		Token:     token.Token,
		TokenType: token.TokenType,
		ExpiresAt: timestamppb.New(token.ExpiresAt),
	}
}

func apiClientInfo(client *models.APIClient) *pb.APIClientInfo {
	if client == nil {
		return nil
	}
	return &pb.APIClientInfo{
		IdApiClient:  client.IDAPIClient.String(),
		ClientIdHash: client.ClientIDHash,
		Description:  client.Description,
		CreatedAt:    timestamppb.New(client.CreatedAt),
		UpdatedAt:    timestamppb.New(client.UpdatedAt),
	}
}

func keyInfoFromSecret(secret string, state pb.KeyState) *pb.KeyInfo {
	key := &pb.KeyInfo{
		SecretCmkKey: secret,
		State:        state,
	}
	key.KeyId, key.VersionId = secretCmkKeyParts(secret)
	return key
}

func requestKeyID(uid string, secret string) string {
	if uid != "" {
		return uid
	}
	return keyInfoFromSecret(secret, pb.KeyState_KEY_STATE_UNSPECIFIED).GetKeyId()
}

func wrappingKeyInfo(kekData *models.KEK) *pb.WrappingKeyInfo {
	if kekData == nil {
		return nil
	}
	return &pb.WrappingKeyInfo{
		IdCmkWrappingKeyRef: kekData.IdCmkWrappingKeyRef.String(),
		Provider:            kekData.Provider,
		KeyRef:              kekData.KeyRef,
		Version:             kekData.Version,
		PublicKey:           kekData.PublicKey,
	}
}
