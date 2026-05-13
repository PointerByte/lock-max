// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type generatedMessage interface {
	goproto.Message
	Reset()
	String() string
	ProtoReflect() protoreflect.Message
}

func testStruct(t *testing.T) *structpb.Struct {
	t.Helper()
	value, err := structpb.NewStruct(map[string]any{"sub": "alice"})
	if err != nil {
		t.Fatalf("unexpected struct setup error: %v", err)
	}
	return value
}

func keyInfo() *KeyInfo {
	return &KeyInfo{
		KeyId:         "key",
		VersionId:     "version",
		SecretCmkKey:  "secret",
		VersionNumber: 2,
		Size:          256,
		Algorithm:     KeyAlgorithm_KEY_ALGORITHM_ECDH,
		Purpose:       KeyPurpose_KEY_PURPOSE_ENCRYPT,
		State:         KeyState_KEY_STATE_ACTIVE,
	}
}

func wrappingKeyInfo() *WrappingKeyInfo {
	return &WrappingKeyInfo{
		IdCmkWrappingKeyRef: "wrap",
		Provider:            "local",
		KeyRef:              "local-key",
		Version:             "v1",
		PublicKey:           "public",
	}
}

func operationStatus() *OperationStatus {
	return &OperationStatus{Success: true, Message: "ok"}
}

func apiClientInfo() *APIClientInfo {
	now := timestamppb.New(time.Unix(1, 0).UTC())
	return &APIClientInfo{
		IdApiClient:  "client-id",
		ClientIdHash: "client-id-hash",
		Description:  "client",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func testMessages(t *testing.T) []generatedMessage {
	t.Helper()
	claims := testStruct(t)
	timeout := durationpb.New(time.Minute)
	annotations := map[string][]byte{"alg": []byte("aes")}

	return []generatedMessage{
		keyInfo(),
		wrappingKeyInfo(),
		operationStatus(),
		&CreateServiceTokenRequest{},
		&CreateAPITokenRequest{ClientId: "client", ClientSecret: "secret"},
		&TokenResponse{Token: "token", TokenType: "Bearer", ExpiresAt: timestamppb.New(time.Unix(1, 0).UTC())},
		apiClientInfo(),
		&CreateAPIClientRequest{ClientId: "client", ClientSecret: "secret", Description: "client"},
		&CreateAPIClientResponse{ApiClient: apiClientInfo()},
		&ListAPIClientsRequest{Page: 1, TotalResgisterPage: 10},
		&ListAPIClientsResponse{Results: []*APIClientInfo{apiClientInfo()}, Pagination: &Pagination{TotalRegisters: 1, TotalPages: 1, TotalRegistersPage: 10, PageNow: 1}},
		&GetAPIClientRequest{ClientId: "client"},
		&DeleteAPIClientRequest{ClientId: "client"},
		&StatusRequest{},
		&StatusResponse{Version: "v1", Healthz: "ok", KeyId: "key"},
		&CreateKeyRequest{IdCmkKey: "key", Algorithm: KeyAlgorithm_KEY_ALGORITHM_SYMMETRIC_DEFAULT, Size: 256, Purpose: KeyPurpose_KEY_PURPOSE_ENCRYPT, Version: 1, EventType: KeyEventType_KEY_EVENT_TYPE_CREATE_KEY},
		&CreateKeyResponse{SecretCmkKey: "secret", Key: keyInfo()},
		&RotateKeyRequest{SecretCmkKey: "secret"},
		&RotateKeyResponse{SecretCmkKey: "secret-v2", NewKey: keyInfo()},
		&RotateWrapKeyRequest{IdCmkWrappingKeyRef: "wrap"},
		&RotateWrapKeyResponse{Status: operationStatus(), WrappingKey: wrappingKeyInfo()},
		&EncryptRequest{Plaintext: []byte("plain"), Uid: "uid", SecretCmkKey: "secret", Additional: "aad"},
		&EncryptResponse{Ciphertext: []byte("cipher"), KeyId: "key", Annotations: annotations},
		&DecryptRequest{Ciphertext: []byte("cipher"), Uid: "uid", KeyId: "key", Annotations: annotations, SecretCmkKey: "secret", Additional: "aad"},
		&DecryptResponse{Plaintext: []byte("plain"), Annotations: annotations},
		&SingRequest{Message: []byte("message"), Uid: "uid", SecretCmkKey: "secret"},
		&SingResponse{Signature: []byte("signature"), KeyId: "key", Annotations: annotations},
		&VerifyRequest{Message: []byte("message"), Signature: []byte("signature"), Uid: "uid", SecretCmkKey: "secret"},
		&VerifyResponse{Valid: true, KeyId: "key"},
		&CreateJWTRequest{SecretCmkKey: "secret", Algorithm: "HS256", Claims: claims, Timeout: timeout},
		&CreateJWTResponse{Token: "token"},
		&VerifyJWTRequest{SecretCmkKey: "secret", Algorithm: "HS256", Token: "token"},
		&VerifyJWTResponse{Valid: true},
		&ReadJWTRequest{SecretCmkKey: "secret", Algorithm: "HS256", Token: "token"},
		&ReadJWTResponse{Claims: claims},
		&DisableKeyRequest{KeyId: "key", VersionId: "version", SecretCmkKey: "secret"},
		&DisableKeyResponse{Key: keyInfo()},
		&EnableKeyRequest{KeyId: "key", VersionId: "version", SecretCmkKey: "secret"},
		&EnableKeyResponse{Key: keyInfo()},
		&ScheduleKeyDeletionRequest{KeyId: "key", VersionId: "version", PendingWindowDays: 7, SecretCmkKey: "secret"},
		&ScheduleKeyDeletionResponse{Key: keyInfo()},
		&PendingDeletionRequest{SecretCmkKey: "secret"},
		&PendingDeletionResponse{Key: keyInfo()},
		&CancelKeyDeletionRequest{KeyId: "key", VersionId: "version", SecretCmkKey: "secret"},
		&CancelKeyDeletionResponse{Key: keyInfo()},
		&UnavailableDeleteRequest{KeyId: "key", VersionId: "version", SecretCmkKey: "secret"},
		&UnavailableDeleteResponse{Key: keyInfo()},
		&DeleteKeyRequest{SecretCmkKey: "secret"},
		&DeleteKeyResponse{Status: operationStatus()},
		&CreateKEKRequest{IdCmkWrappingKeyRef: "wrap", SecretCmkKey: "secret", Salt: "salt"},
		&CreateKEKResponse{Status: operationStatus(), WrappingKey: wrappingKeyInfo()},
		&GetKEKRequest{IdCmkWrappingKeyRef: "wrap", Version: "v1"},
		&GetKEKResponse{WrappingKey: wrappingKeyInfo()},
		&RotateKEKRequest{IdCmkWrappingKeyRef: "wrap", Salt: "salt"},
		&RotateKEKResponse{Status: operationStatus(), WrappingKey: wrappingKeyInfo()},
		&DeleteKEKRequest{IdCmkWrappingKeyRef: "wrap", Version: "v1"},
		&DeleteKEKResponse{Status: operationStatus()},
	}
}

func callGetterMethods(t *testing.T, value reflect.Value) {
	t.Helper()
	for i := 0; i < value.Type().NumMethod(); i++ {
		method := value.Type().Method(i)
		if !strings.HasPrefix(method.Name, "Get") {
			continue
		}
		results := value.Method(i).Call(nil)
		if len(results) != 1 {
			t.Fatalf("%s returned %d values", method.Name, len(results))
		}
	}
}

func TestGeneratedMessagesAccessorsAndReflection(t *testing.T) {
	for _, message := range testMessages(t) {
		value := reflect.ValueOf(message)
		callGetterMethods(t, value)
		callGetterMethods(t, reflect.Zero(value.Type()))

		descriptor := message.ProtoReflect().Descriptor()
		if descriptor.FullName() == "" {
			t.Fatalf("empty descriptor for %T", message)
		}
		if message.String() == "" && descriptor.Fields().Len() > 0 {
			t.Fatalf("expected non-empty string for %T", message)
		}
		data, err := goproto.Marshal(message)
		if err != nil {
			t.Fatalf("marshal %T: %v", message, err)
		}
		clone := reflect.New(value.Type().Elem()).Interface().(goproto.Message)
		if err := goproto.Unmarshal(data, clone); err != nil {
			t.Fatalf("unmarshal %T: %v", message, err)
		}
		if descriptorMethod := value.MethodByName("Descriptor"); descriptorMethod.IsValid() {
			results := descriptorMethod.Call(nil)
			if len(results) != 2 || results[0].Len() == 0 {
				t.Fatalf("unexpected Descriptor result for %T", message)
			}
		}
		message.Reset()
		if message.ProtoReflect().Descriptor().FullName() != descriptor.FullName() {
			t.Fatalf("descriptor changed after reset for %T", message)
		}
	}
}

func TestGeneratedEnumMethods(t *testing.T) {
	tests := []struct {
		name  string
		value interface {
			Descriptor() protoreflect.EnumDescriptor
			Type() protoreflect.EnumType
			Number() protoreflect.EnumNumber
			String() string
			EnumDescriptor() ([]byte, []int)
		}
		enumPtr any
		want    string
	}{
		{name: "state", value: KeyState_KEY_STATE_ACTIVE, enumPtr: KeyState_KEY_STATE_ACTIVE.Enum(), want: "KEY_STATE_ACTIVE"},
		{name: "purpose", value: KeyPurpose_KEY_PURPOSE_SIGN, enumPtr: KeyPurpose_KEY_PURPOSE_SIGN.Enum(), want: "KEY_PURPOSE_SIGN"},
		{name: "algorithm", value: KeyAlgorithm_KEY_ALGORITHM_RSA_OAEP, enumPtr: KeyAlgorithm_KEY_ALGORITHM_RSA_OAEP.Enum(), want: "KEY_ALGORITHM_RSA_OAEP"},
		{name: "event", value: KeyEventType_KEY_EVENT_TYPE_ROTATE_KEY, enumPtr: KeyEventType_KEY_EVENT_TYPE_ROTATE_KEY.Enum(), want: "KEY_EVENT_TYPE_ROTATE_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.String() != tt.want {
				t.Fatalf("unexpected enum string: %s", tt.value.String())
			}
			if reflect.ValueOf(tt.enumPtr).IsNil() {
				t.Fatal("expected enum pointer")
			}
			if tt.value.Descriptor().FullName() == "" {
				t.Fatal("expected enum descriptor")
			}
			if tt.value.Type().Descriptor().FullName() != tt.value.Descriptor().FullName() {
				t.Fatal("expected enum type descriptor to match")
			}
			if tt.value.Number() == 0 {
				t.Fatal("expected non-zero enum number")
			}
			raw, indexes := tt.value.EnumDescriptor()
			if len(raw) == 0 || len(indexes) == 0 {
				t.Fatalf("unexpected enum descriptor data: raw=%d indexes=%v", len(raw), indexes)
			}
		})
	}

	if KeyState_name[int32(KeyState_KEY_STATE_RETIRED)] != "KEY_STATE_RETIRED" {
		t.Fatal("expected KeyState name map")
	}
	if KeyPurpose_value["KEY_PURPOSE_WRAP"] != int32(KeyPurpose_KEY_PURPOSE_WRAP) {
		t.Fatal("expected KeyPurpose value map")
	}
	if KeyAlgorithm_value["KEY_ALGORITHM_EDDSA"] != int32(KeyAlgorithm_KEY_ALGORITHM_EDDSA) {
		t.Fatal("expected KeyAlgorithm value map")
	}
	if KeyEventType_name[int32(KeyEventType_KEY_EVENT_TYPE_CREATE_KEY)] != "KEY_EVENT_TYPE_CREATE_KEY" {
		t.Fatal("expected KeyEventType name map")
	}
}

func TestFileDescriptorHelpers(t *testing.T) {
	if File_proto_route_proto.Path() != "proto/route.proto" {
		t.Fatalf("unexpected proto path: %s", File_proto_route_proto.Path())
	}
	if len(file_proto_route_proto_rawDescGZIP()) == 0 {
		t.Fatal("expected raw descriptor")
	}
	file_proto_route_proto_init()
}

type testKeyServer struct {
	UnimplementedKeyServiceServer
}

func (testKeyServer) Status(context.Context, *StatusRequest) (*StatusResponse, error) {
	return &StatusResponse{Version: "v1", Healthz: "ok", KeyId: "key"}, nil
}

func (testKeyServer) CreateKey(_ context.Context, request *CreateKeyRequest) (*CreateKeyResponse, error) {
	return &CreateKeyResponse{
		SecretCmkKey: "secret",
		Key: &KeyInfo{
			KeyId:         request.GetIdCmkKey(),
			Algorithm:     request.GetAlgorithm(),
			Purpose:       request.GetPurpose(),
			Size:          request.GetSize(),
			VersionNumber: request.GetVersion(),
		},
	}, nil
}

func (testKeyServer) RotateKey(_ context.Context, request *RotateKeyRequest) (*RotateKeyResponse, error) {
	return &RotateKeyResponse{SecretCmkKey: request.GetSecretCmkKey() + ".rotated", NewKey: keyInfo()}, nil
}

func (testKeyServer) RotateWrapKey(_ context.Context, request *RotateWrapKeyRequest) (*RotateWrapKeyResponse, error) {
	return &RotateWrapKeyResponse{Status: operationStatus(), WrappingKey: &WrappingKeyInfo{IdCmkWrappingKeyRef: request.GetIdCmkWrappingKeyRef()}}, nil
}

func (testKeyServer) Encrypt(_ context.Context, request *EncryptRequest) (*EncryptResponse, error) {
	return &EncryptResponse{Ciphertext: append([]byte("cipher:"), request.GetPlaintext()...), KeyId: request.GetUid(), Annotations: map[string][]byte{"aad": []byte(request.GetAdditional())}}, nil
}

func (testKeyServer) Decrypt(_ context.Context, request *DecryptRequest) (*DecryptResponse, error) {
	return &DecryptResponse{Plaintext: append([]byte("plain:"), request.GetCiphertext()...), Annotations: request.GetAnnotations()}, nil
}

func (testKeyServer) Sing(_ context.Context, request *SingRequest) (*SingResponse, error) {
	return &SingResponse{Signature: append([]byte("sig:"), request.GetMessage()...), KeyId: request.GetUid(), Annotations: map[string][]byte{"sig": []byte("ok")}}, nil
}

func (testKeyServer) Verify(_ context.Context, request *VerifyRequest) (*VerifyResponse, error) {
	return &VerifyResponse{Valid: string(request.GetSignature()) != "", KeyId: request.GetUid()}, nil
}

func (testKeyServer) CreateJWT(context.Context, *CreateJWTRequest) (*CreateJWTResponse, error) {
	return &CreateJWTResponse{Token: "token"}, nil
}

func (testKeyServer) VerifyJWT(context.Context, *VerifyJWTRequest) (*VerifyJWTResponse, error) {
	return &VerifyJWTResponse{Valid: true}, nil
}

func (testKeyServer) ReadJWT(context.Context, *ReadJWTRequest) (*ReadJWTResponse, error) {
	claims, _ := structpb.NewStruct(map[string]any{"sub": "alice"})
	return &ReadJWTResponse{Claims: claims}, nil
}

func (testKeyServer) EnableKey(_ context.Context, request *EnableKeyRequest) (*EnableKeyResponse, error) {
	return &EnableKeyResponse{Key: &KeyInfo{KeyId: request.GetKeyId(), VersionId: request.GetVersionId(), SecretCmkKey: request.GetSecretCmkKey(), State: KeyState_KEY_STATE_ACTIVE}}, nil
}

func (testKeyServer) DisableKey(_ context.Context, request *DisableKeyRequest) (*DisableKeyResponse, error) {
	return &DisableKeyResponse{Key: &KeyInfo{KeyId: request.GetKeyId(), VersionId: request.GetVersionId(), SecretCmkKey: request.GetSecretCmkKey(), State: KeyState_KEY_STATE_DISABLED}}, nil
}

func (testKeyServer) ScheduleKeyDeletion(_ context.Context, request *ScheduleKeyDeletionRequest) (*ScheduleKeyDeletionResponse, error) {
	return &ScheduleKeyDeletionResponse{Key: &KeyInfo{KeyId: request.GetKeyId(), VersionId: request.GetVersionId(), SecretCmkKey: request.GetSecretCmkKey(), State: KeyState_KEY_STATE_PENDING_DELETION}}, nil
}

func (testKeyServer) PendingDeletion(_ context.Context, request *PendingDeletionRequest) (*PendingDeletionResponse, error) {
	return &PendingDeletionResponse{Key: &KeyInfo{SecretCmkKey: request.GetSecretCmkKey(), State: KeyState_KEY_STATE_PENDING_DELETION}}, nil
}

func (testKeyServer) CancelKeyDeletion(_ context.Context, request *CancelKeyDeletionRequest) (*CancelKeyDeletionResponse, error) {
	return &CancelKeyDeletionResponse{Key: &KeyInfo{KeyId: request.GetKeyId(), VersionId: request.GetVersionId(), SecretCmkKey: request.GetSecretCmkKey(), State: KeyState_KEY_STATE_DISABLED}}, nil
}

func (testKeyServer) UnavailableDelete(_ context.Context, request *UnavailableDeleteRequest) (*UnavailableDeleteResponse, error) {
	return &UnavailableDeleteResponse{Key: &KeyInfo{KeyId: request.GetKeyId(), VersionId: request.GetVersionId(), SecretCmkKey: request.GetSecretCmkKey(), State: KeyState_KEY_STATE_UNAVAILABLE}}, nil
}

func (testKeyServer) DeleteKey(context.Context, *DeleteKeyRequest) (*DeleteKeyResponse, error) {
	return &DeleteKeyResponse{Status: operationStatus()}, nil
}

func startProtoServer(t *testing.T, withInterceptor bool) *grpc.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	options := []grpc.ServerOption{}
	if withInterceptor {
		options = append(options, grpc.UnaryInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}))
	}
	server := grpc.NewServer(options...)
	RegisterKeyServiceServer(server, testKeyServer{})

	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufnet: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	return conn
}

func exerciseProtoClients(t *testing.T, conn *grpc.ClientConn) {
	t.Helper()
	ctx := context.Background()
	keyClient := NewKeyServiceClient(conn)

	statusResponse, err := keyClient.Status(ctx, &StatusRequest{}, grpc.WaitForReady(false))
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if statusResponse.GetVersion() != "v1" || statusResponse.GetHealthz() != "ok" {
		t.Fatalf("unexpected status response: %#v", statusResponse)
	}

	createResponse, err := keyClient.CreateKey(ctx, &CreateKeyRequest{IdCmkKey: "key", Algorithm: KeyAlgorithm_KEY_ALGORITHM_ECDH, Size: 256, Purpose: KeyPurpose_KEY_PURPOSE_ENCRYPT, Version: 1})
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if createResponse.GetSecretCmkKey() != "secret" || createResponse.GetKey().GetAlgorithm() != KeyAlgorithm_KEY_ALGORITHM_ECDH {
		t.Fatalf("unexpected create key response: %#v", createResponse)
	}

	rotateResponse, err := keyClient.RotateKey(ctx, &RotateKeyRequest{SecretCmkKey: "secret"})
	if err != nil {
		t.Fatalf("RotateKey: %v", err)
	}
	if rotateResponse.GetSecretCmkKey() != "secret.rotated" {
		t.Fatalf("unexpected rotate response: %#v", rotateResponse)
	}

	rotateWrapResponse, err := keyClient.RotateWrapKey(ctx, &RotateWrapKeyRequest{IdCmkWrappingKeyRef: "wrap"})
	if err != nil {
		t.Fatalf("RotateWrapKey: %v", err)
	}
	if !rotateWrapResponse.GetStatus().GetSuccess() || rotateWrapResponse.GetWrappingKey().GetIdCmkWrappingKeyRef() != "wrap" {
		t.Fatalf("unexpected rotate wrap response: %#v", rotateWrapResponse)
	}

	encryptResponse, err := keyClient.Encrypt(ctx, &EncryptRequest{Plaintext: []byte("plain"), Uid: "uid", Additional: "aad"})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if string(encryptResponse.GetCiphertext()) != "cipher:plain" || encryptResponse.GetKeyId() != "uid" {
		t.Fatalf("unexpected encrypt response: %#v", encryptResponse)
	}

	decryptResponse, err := keyClient.Decrypt(ctx, &DecryptRequest{Ciphertext: []byte("cipher"), Annotations: map[string][]byte{"a": []byte("b")}})
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decryptResponse.GetPlaintext()) != "plain:cipher" || string(decryptResponse.GetAnnotations()["a"]) != "b" {
		t.Fatalf("unexpected decrypt response: %#v", decryptResponse)
	}

	singResponse, err := keyClient.Sing(ctx, &SingRequest{Message: []byte("message"), Uid: "uid"})
	if err != nil {
		t.Fatalf("Sing: %v", err)
	}
	if string(singResponse.GetSignature()) != "sig:message" || singResponse.GetKeyId() != "uid" {
		t.Fatalf("unexpected sing response: %#v", singResponse)
	}

	verifyResponse, err := keyClient.Verify(ctx, &VerifyRequest{Signature: []byte("sig"), Uid: "uid"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !verifyResponse.GetValid() || verifyResponse.GetKeyId() != "uid" {
		t.Fatalf("unexpected verify response: %#v", verifyResponse)
	}

	createJWTResponse, err := keyClient.CreateJWT(ctx, &CreateJWTRequest{Claims: testStruct(t), Timeout: durationpb.New(time.Minute)})
	if err != nil {
		t.Fatalf("CreateJWT: %v", err)
	}
	if createJWTResponse.GetToken() != "token" {
		t.Fatalf("unexpected jwt response: %#v", createJWTResponse)
	}

	verifyJWTResponse, err := keyClient.VerifyJWT(ctx, &VerifyJWTRequest{Token: "token"})
	if err != nil {
		t.Fatalf("VerifyJWT: %v", err)
	}
	if !verifyJWTResponse.GetValid() {
		t.Fatalf("unexpected verify jwt response: %#v", verifyJWTResponse)
	}

	readJWTResponse, err := keyClient.ReadJWT(ctx, &ReadJWTRequest{Token: "token"})
	if err != nil {
		t.Fatalf("ReadJWT: %v", err)
	}
	if readJWTResponse.GetClaims().AsMap()["sub"] != "alice" {
		t.Fatalf("unexpected read jwt response: %#v", readJWTResponse)
	}

}

func TestGRPCClientsAndHandlers(t *testing.T) {
	t.Run("without interceptor", func(t *testing.T) {
		exerciseProtoClients(t, startProtoServer(t, false))
	})
	t.Run("with interceptor", func(t *testing.T) {
		exerciseProtoClients(t, startProtoServer(t, true))
	})
}

type failingClientConn struct{}

func (failingClientConn) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	return errors.New("invoke failed")
}

func (failingClientConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("stream failed")
}

func TestGRPCClientsReturnInvokeErrors(t *testing.T) {
	ctx := context.Background()
	keyClient := NewKeyServiceClient(failingClientConn{})

	keyCalls := []func() error{
		func() error { _, err := keyClient.Status(ctx, &StatusRequest{}); return err },
		func() error { _, err := keyClient.CreateKey(ctx, &CreateKeyRequest{}); return err },
		func() error { _, err := keyClient.RotateKey(ctx, &RotateKeyRequest{}); return err },
		func() error { _, err := keyClient.RotateWrapKey(ctx, &RotateWrapKeyRequest{}); return err },
		func() error { _, err := keyClient.Encrypt(ctx, &EncryptRequest{}); return err },
		func() error { _, err := keyClient.Decrypt(ctx, &DecryptRequest{}); return err },
		func() error { _, err := keyClient.Sing(ctx, &SingRequest{}); return err },
		func() error { _, err := keyClient.Verify(ctx, &VerifyRequest{}); return err },
		func() error { _, err := keyClient.CreateJWT(ctx, &CreateJWTRequest{}); return err },
		func() error { _, err := keyClient.VerifyJWT(ctx, &VerifyJWTRequest{}); return err },
		func() error { _, err := keyClient.ReadJWT(ctx, &ReadJWTRequest{}); return err },
	}
	for i, call := range keyCalls {
		if err := call(); err == nil {
			t.Fatalf("expected key client error at call %d", i)
		}
	}
}

func TestGeneratedHandlersReturnDecodeErrors(t *testing.T) {
	expected := errors.New("decode failed")
	decode := func(any) error { return expected }

	keyHandlers := []func(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error){
		_KeyService_CreateServiceToken_Handler,
		_KeyService_CreateAPIToken_Handler,
		_KeyService_CreateAPIClient_Handler,
		_KeyService_ListAPIClients_Handler,
		_KeyService_GetAPIClient_Handler,
		_KeyService_DeleteAPIClient_Handler,
		_KeyService_Status_Handler,
		_KeyService_CreateKey_Handler,
		_KeyService_RotateKey_Handler,
		_KeyService_RotateWrapKey_Handler,
		_KeyService_Encrypt_Handler,
		_KeyService_Decrypt_Handler,
		_KeyService_Sing_Handler,
		_KeyService_Verify_Handler,
		_KeyService_CreateJWT_Handler,
		_KeyService_VerifyJWT_Handler,
		_KeyService_ReadJWT_Handler,
	}
	for i, handler := range keyHandlers {
		if _, err := handler(testKeyServer{}, context.Background(), decode, nil); !errors.Is(err, expected) {
			t.Fatalf("expected decode error from key handler %d, got %v", i, err)
		}
	}
}

func TestUnimplementedServersReturnUnimplemented(t *testing.T) {
	ctx := context.Background()
	keyServer := UnimplementedKeyServiceServer{}
	keyCalls := []func() error{
		func() error { _, err := keyServer.CreateServiceToken(ctx, &CreateServiceTokenRequest{}); return err },
		func() error { _, err := keyServer.CreateAPIToken(ctx, &CreateAPITokenRequest{}); return err },
		func() error { _, err := keyServer.CreateAPIClient(ctx, &CreateAPIClientRequest{}); return err },
		func() error { _, err := keyServer.ListAPIClients(ctx, &ListAPIClientsRequest{}); return err },
		func() error { _, err := keyServer.GetAPIClient(ctx, &GetAPIClientRequest{}); return err },
		func() error { _, err := keyServer.DeleteAPIClient(ctx, &DeleteAPIClientRequest{}); return err },
		func() error { _, err := keyServer.Status(ctx, &StatusRequest{}); return err },
		func() error { _, err := keyServer.ListCmkKeys(ctx, &ListCmkKeysRequest{}); return err },
		func() error { _, err := keyServer.CreateKey(ctx, &CreateKeyRequest{}); return err },
		func() error { _, err := keyServer.RotateKey(ctx, &RotateKeyRequest{}); return err },
		func() error { _, err := keyServer.RotateWrapKey(ctx, &RotateWrapKeyRequest{}); return err },
		func() error { _, err := keyServer.Encrypt(ctx, &EncryptRequest{}); return err },
		func() error { _, err := keyServer.Decrypt(ctx, &DecryptRequest{}); return err },
		func() error { _, err := keyServer.Sing(ctx, &SingRequest{}); return err },
		func() error { _, err := keyServer.Verify(ctx, &VerifyRequest{}); return err },
		func() error { _, err := keyServer.CreateJWT(ctx, &CreateJWTRequest{}); return err },
		func() error { _, err := keyServer.VerifyJWT(ctx, &VerifyJWTRequest{}); return err },
		func() error { _, err := keyServer.ReadJWT(ctx, &ReadJWTRequest{}); return err },
	}
	for i, call := range keyCalls {
		if code := status.Code(call()); code != codes.Unimplemented {
			t.Fatalf("expected key unimplemented at call %d, got %s", i, code)
		}
	}
}

func TestServiceDescriptors(t *testing.T) {
	if KeyService_ServiceDesc.ServiceName != "key.v1.KeyService" || len(KeyService_ServiceDesc.Methods) != 18 || KeyService_ServiceDesc.Metadata != "proto/route.proto" {
		t.Fatalf("unexpected key service descriptor: %#v", KeyService_ServiceDesc)
	}
	if KeyService_Status_FullMethodName != "/key.v1.KeyService/Status" ||
		KeyService_CreateServiceToken_FullMethodName != "/key.v1.KeyService/CreateServiceToken" {
		t.Fatal("unexpected full method constants")
	}
}
