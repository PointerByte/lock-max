package controller

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/PointerByte/GoForge/logger/builder"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	pb "github.com/PointerByte/lock-max/dragon-cmk/proto"
	kek "github.com/PointerByte/lock-max/dragon-cmk/service/KEK"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeCMKRepository struct {
	statusResponse *models.StatusResponse
	statusErr      error

	createInput  models.CreateKeyInput
	createEvent  commonEntity.EventType
	createSecret string
	createErr    error

	rotateSecretIn  string
	rotateSecretOut string
	rotateErr       error

	rotateWrapID  uuid.UUID
	rotateWrapErr error

	encryptSecret      string
	encryptPlaintext   string
	encryptAdditional  *string
	encryptCiphertext  string
	encryptAnnotations map[string][]byte
	encryptErr         error

	decryptSecret      string
	decryptCiphertext  string
	decryptAdditional  *string
	decryptPlaintext   string
	decryptAnnotations map[string][]byte
	decryptErr         error

	jwtSecret string
	jwtAlg    string
	jwtClaims any
	jwtToken  string
	jwtErr    error

	verifyJWTSecret string
	verifyJWTAlg    string
	verifyJWTToken  string
	verifyJWTErr    error

	readJWTSecret       string
	readJWTAlg          string
	readJWTToken        string
	readJWTErr          error
	readJWTInvalidClaim bool

	signSecret    string
	signMessage   string
	signSignature string
	signErr       error

	verifySecret    string
	verifyMessage   string
	verifySignature string
	verifyValid     bool
	verifyErr       error

	enableSecret        string
	enableErr           error
	disableSecret       string
	disableErr          error
	scheduleSecret      string
	scheduleInterval    uint
	scheduleErr         error
	pendingSecrets      []string
	pendingErr          error
	cancelSecret        string
	cancelErr           error
	unavailableSecret   string
	unavailableErr      error
	deleteSecret        string
	deleteErr           error
	listIDKek           uuid.UUID
	listPage            uint
	listTotalPage       uint
	listResult          *models.PaginatedCmkKey
	listErr             error
	listQueueIDKey      uuid.UUID
	listQueueStatus     commonEntity.QueueStatus
	listQueuePage       uint
	listQueueTotal      uint
	listQueueResult     *models.PaginatedCreationKeyQueue
	listQueueErr        error
	getVersionID        uuid.UUID
	getVersionResult    *models.KeyVersionInfo
	getVersionErr       error
	updateVersionID     uuid.UUID
	updateVersionStatus commonEntity.KeyVersionStatus
	updateVersionResult *models.KeyVersionInfo
	updateVersionErr    error
}

func (f *fakeCMKRepository) Status() (*models.StatusResponse, error) {
	return f.statusResponse, f.statusErr
}

func (f *fakeCMKRepository) ListCmkKey(idKek uuid.UUID, page uint, totalRegisterPage uint) (*models.PaginatedCmkKey, error) {
	f.listIDKek = idKek
	f.listPage = page
	f.listTotalPage = totalRegisterPage
	if f.listResult != nil {
		return f.listResult, f.listErr
	}
	return &models.PaginatedCmkKey{
		Pagination: models.Pagination{
			TotalRegistersPage: totalRegisterPage,
			PageNow:            page,
		},
	}, f.listErr
}

func (f *fakeCMKRepository) ListCreationKeyQueues(idCmkKey uuid.UUID, status commonEntity.QueueStatus, page uint, totalRegisterPage uint) (*models.PaginatedCreationKeyQueue, error) {
	f.listQueueIDKey = idCmkKey
	f.listQueueStatus = status
	f.listQueuePage = page
	f.listQueueTotal = totalRegisterPage
	if f.listQueueResult != nil {
		return f.listQueueResult, f.listQueueErr
	}
	return &models.PaginatedCreationKeyQueue{
		Pagination: models.Pagination{
			TotalRegistersPage: totalRegisterPage,
			PageNow:            page,
		},
	}, f.listQueueErr
}

func (f *fakeCMKRepository) GetKeyVersionInfo(idCmkKeyVersion uuid.UUID) (*models.KeyVersionInfo, error) {
	f.getVersionID = idCmkKeyVersion
	return f.getVersionResult, f.getVersionErr
}

func (f *fakeCMKRepository) UpdateKeyVersionStatus(idCmkKeyVersion uuid.UUID, status commonEntity.KeyVersionStatus) (*models.KeyVersionInfo, error) {
	f.updateVersionID = idCmkKeyVersion
	f.updateVersionStatus = status
	return f.updateVersionResult, f.updateVersionErr
}

func (f *fakeCMKRepository) CreateKey(input models.CreateKeyInput, eventType commonEntity.EventType) (string, error) {
	f.createInput = input
	f.createEvent = eventType
	return f.createSecret, f.createErr
}

func (f *fakeCMKRepository) RotateKey(secretCmkKey string) (string, error) {
	f.rotateSecretIn = secretCmkKey
	return f.rotateSecretOut, f.rotateErr
}

func (f *fakeCMKRepository) RotateWrapKey(idCmkWrappingKeyRef uuid.UUID) error {
	f.rotateWrapID = idCmkWrappingKeyRef
	return f.rotateWrapErr
}

func (f *fakeCMKRepository) Encrypt(secretCmkKey string, plaintext string, additional *string) (string, map[string][]byte, error) {
	f.encryptSecret = secretCmkKey
	f.encryptPlaintext = plaintext
	f.encryptAdditional = additional
	return f.encryptCiphertext, f.encryptAnnotations, f.encryptErr
}

func (f *fakeCMKRepository) Decrypt(secretCmkKey string, ciphertext string, additional *string) (string, map[string][]byte, error) {
	f.decryptSecret = secretCmkKey
	f.decryptCiphertext = ciphertext
	f.decryptAdditional = additional
	return f.decryptPlaintext, f.decryptAnnotations, f.decryptErr
}

func (f *fakeCMKRepository) CreateJWT(_ context.Context, secretCmkKey string, algorithm string, claims any) (string, error) {
	f.jwtSecret = secretCmkKey
	f.jwtAlg = algorithm
	f.jwtClaims = claims
	return f.jwtToken, f.jwtErr
}

func (f *fakeCMKRepository) VerifyJWT(_ context.Context, secretCmkKey string, algorithm string, token string) error {
	f.verifyJWTSecret = secretCmkKey
	f.verifyJWTAlg = algorithm
	f.verifyJWTToken = token
	return f.verifyJWTErr
}

func (f *fakeCMKRepository) ReadJWT(_ context.Context, secretCmkKey string, algorithm string, token string, claims any) error {
	f.readJWTSecret = secretCmkKey
	f.readJWTAlg = algorithm
	f.readJWTToken = token
	if values, ok := claims.(map[string]any); ok {
		values["sub"] = "alice"
		if f.readJWTInvalidClaim {
			values["bad"] = make(chan int)
		}
	}
	return f.readJWTErr
}

func (f *fakeCMKRepository) Sing(secretCmkKey string, message string) (string, error) {
	f.signSecret = secretCmkKey
	f.signMessage = message
	return f.signSignature, f.signErr
}

func (f *fakeCMKRepository) Verify(secretCmkKey string, message string, signature string) (bool, error) {
	f.verifySecret = secretCmkKey
	f.verifyMessage = message
	f.verifySignature = signature
	return f.verifyValid, f.verifyErr
}

func (f *fakeCMKRepository) DisableKey(secretCmkKey string) error {
	f.disableSecret = secretCmkKey
	return f.disableErr
}

func (f *fakeCMKRepository) EnableKey(secretCmkKey string) error {
	f.enableSecret = secretCmkKey
	return f.enableErr
}

func (f *fakeCMKRepository) ScheduleKeyDeletion(secretCmkKey string, interval uint) error {
	f.scheduleSecret = secretCmkKey
	f.scheduleInterval = interval
	return f.scheduleErr
}

func (f *fakeCMKRepository) PendingDeletion(secretCmkKey string) error {
	f.pendingSecrets = append(f.pendingSecrets, secretCmkKey)
	return f.pendingErr
}

func (f *fakeCMKRepository) CancelKeyDeletion(secretCmkKey string) error {
	f.cancelSecret = secretCmkKey
	return f.cancelErr
}

func (f *fakeCMKRepository) UnavailableDelete(secretCmkKey string) error {
	f.unavailableSecret = secretCmkKey
	return f.unavailableErr
}

func (f *fakeCMKRepository) DeleteKey(secretCmkKey string) error {
	f.deleteSecret = secretCmkKey
	return f.deleteErr
}

type fakeWrappingRepository struct {
	createFn kek.HandlerFuncCreteKey
	rotateFn kek.HandlerFuncRotateKey

	createSecret string
	createSalt   string
	createID     uuid.UUID
	createResult uuid.UUID
	createErr    error

	getVersions []string
	getIDs      []uuid.UUID
	getData     *models.KEK
	getErr      error

	rotateID     uuid.UUID
	rotateSalt   string
	rotateResult uuid.UUID
	rotateErr    error

	deleteID      uuid.UUID
	deleteVersion string
	deleteErr     error

	listID        uuid.UUID
	listPage      uint
	listTotalPage uint
}

func (f *fakeWrappingRepository) SetFuncCreateKey(fn kek.HandlerFuncCreteKey) {
	f.createFn = fn
}

func (f *fakeWrappingRepository) SetFuncRotate(fn kek.HandlerFuncRotateKey) {
	f.rotateFn = fn
}

func (f *fakeWrappingRepository) SetFuncRotateWrapKey(kek.HandlerFuncRotateWrapKey) {}

func (f *fakeWrappingRepository) SetFuncDeleteWrapKey(kek.HandlerFuncDeleteWrapKey) {}

func (f *fakeWrappingRepository) CreateKEK(id uuid.UUID, secretCmkKey string, salt string) (*uuid.UUID, error) {
	f.createID = id
	f.createSecret = secretCmkKey
	f.createSalt = salt
	if f.createResult == uuid.Nil {
		f.createResult = id
	}
	return &f.createResult, f.createErr
}

func (f *fakeWrappingRepository) GetKEK(id uuid.UUID, version string) (*models.KEK, error) {
	f.getIDs = append(f.getIDs, id)
	f.getVersions = append(f.getVersions, version)
	return f.getData, f.getErr
}

func (f *fakeWrappingRepository) RotateKEK(id uuid.UUID, salt string) (*uuid.UUID, error) {
	f.rotateID = id
	f.rotateSalt = salt
	if f.rotateResult == uuid.Nil {
		f.rotateResult = id
	}
	return &f.rotateResult, f.rotateErr
}

func (f *fakeWrappingRepository) ListKEK(id uuid.UUID, page uint, totalRegisterPage uint) (*models.PaginatedKEK, error) {
	f.listID = id
	f.listPage = page
	f.listTotalPage = totalRegisterPage
	if f.getData == nil {
		return &models.PaginatedKEK{Pagination: models.Pagination{PageNow: page, TotalRegistersPage: totalRegisterPage}}, f.getErr
	}
	return &models.PaginatedKEK{
		Results: []models.KEK{*f.getData},
		Pagination: models.Pagination{
			TotalRegisters:     1,
			TotalPages:         1,
			TotalRegistersPage: totalRegisterPage,
			PageNow:            page,
		},
	}, f.getErr
}

func (f *fakeWrappingRepository) DeleteKey(id uuid.UUID, version string) error {
	f.deleteID = id
	f.deleteVersion = version
	return f.deleteErr
}

func TestNewGRPCServices(t *testing.T) {
	keyService := NewGRPCServices(context.Background(), builder.New(context.Background()))
	if keyService == nil {
		t.Fatal("expected grpc services")
	}
}

func TestKeyServiceDelegatesToCMK(t *testing.T) {
	ctx := context.Background()
	idKey := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	idVersion := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	idWrap := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	secret := idKey.String() + "." + idVersion.String()
	annotations := map[string][]byte{"alg": []byte("aes")}
	repository := &fakeCMKRepository{
		statusResponse: &models.StatusResponse{
			ID:      idKey,
			Healthz: "ok",
			Version: "v1",
		},
		createSecret:       secret,
		rotateSecretOut:    secret,
		encryptCiphertext:  "cipher",
		encryptAnnotations: annotations,
		decryptPlaintext:   "plain",
		decryptAnnotations: annotations,
		jwtToken:           "token",
		signSignature:      "signature",
		verifyValid:        true,
	}
	service := NewKeyService(repository)

	statusResponse, err := service.Status(ctx, &pb.StatusRequest{})
	if err != nil {
		t.Fatalf("unexpected status error: %v", err)
	}
	if statusResponse.GetKeyId() != idKey.String() || statusResponse.GetHealthz() != "ok" || statusResponse.GetVersion() != "v1" {
		t.Fatalf("unexpected status response: %#v", statusResponse)
	}

	createResponse, err := service.CreateKey(ctx, &pb.CreateKeyRequest{
		IdCmkKey:  idKey.String(),
		Algorithm: pb.KeyAlgorithm_KEY_ALGORITHM_ECDH,
		Size:      256,
		Purpose:   pb.KeyPurpose_KEY_PURPOSE_ENCRYPT,
		Version:   2,
		EventType: pb.KeyEventType_KEY_EVENT_TYPE_ROTATE_KEY,
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if repository.createInput.IDCmkKey != nil {
		t.Fatalf("expected grpc CreateKey to let the server generate id_cmk_key, got %#v", repository.createInput.IDCmkKey)
	}
	if repository.createInput.Algorithm != commonEntity.KeyTypeECDH || repository.createInput.Purpose != commonEntity.KeyPurposeEncrypt || repository.createInput.Size != 256 || repository.createInput.Version != 2 {
		t.Fatalf("unexpected create input: %#v", repository.createInput)
	}
	if repository.createEvent != commonEntity.EventTypeRotateKey {
		t.Fatalf("unexpected event type: %s", repository.createEvent)
	}
	if createResponse.GetSecretCmkKey() != secret || createResponse.GetKey().GetKeyId() != idKey.String() {
		t.Fatalf("unexpected create response: %#v", createResponse)
	}

	rotateResponse, err := service.RotateKey(ctx, &pb.RotateKeyRequest{SecretCmkKey: secret})
	if err != nil {
		t.Fatalf("unexpected rotate error: %v", err)
	}
	if repository.rotateSecretIn != secret || rotateResponse.GetSecretCmkKey() != secret {
		t.Fatalf("unexpected rotate result: secret=%s response=%#v", repository.rotateSecretIn, rotateResponse)
	}

	rotateWrapResponse, err := service.RotateWrapKey(ctx, &pb.RotateWrapKeyRequest{IdCmkWrappingKeyRef: idWrap.String()})
	if err != nil {
		t.Fatalf("unexpected rotate wrap error: %v", err)
	}
	if repository.rotateWrapID != idWrap || !rotateWrapResponse.GetStatus().GetSuccess() || rotateWrapResponse.GetWrappingKey().GetIdCmkWrappingKeyRef() != idWrap.String() {
		t.Fatalf("unexpected rotate wrap result: id=%s response=%#v", repository.rotateWrapID, rotateWrapResponse)
	}

	encryptResponse, err := service.Encrypt(ctx, &pb.EncryptRequest{
		Plaintext:    []byte("plain"),
		Uid:          "request-key",
		SecretCmkKey: secret,
		Additional:   "aad",
	})
	if err != nil {
		t.Fatalf("unexpected encrypt error: %v", err)
	}
	if repository.encryptSecret != secret || repository.encryptPlaintext != "plain" || repository.encryptAdditional == nil || *repository.encryptAdditional != "aad" {
		t.Fatalf("unexpected encrypt input: %#v", repository)
	}
	if string(encryptResponse.GetCiphertext()) != "cipher" || encryptResponse.GetKeyId() != "request-key" || !reflect.DeepEqual(encryptResponse.GetAnnotations(), annotations) {
		t.Fatalf("unexpected encrypt response: %#v", encryptResponse)
	}

	decryptResponse, err := service.Decrypt(ctx, &pb.DecryptRequest{
		Ciphertext:   []byte("cipher"),
		SecretCmkKey: secret,
	})
	if err != nil {
		t.Fatalf("unexpected decrypt error: %v", err)
	}
	if repository.decryptSecret != secret || repository.decryptCiphertext != "cipher" || repository.decryptAdditional != nil {
		t.Fatalf("unexpected decrypt input: %#v", repository)
	}
	if string(decryptResponse.GetPlaintext()) != "plain" || !reflect.DeepEqual(decryptResponse.GetAnnotations(), annotations) {
		t.Fatalf("unexpected decrypt response: %#v", decryptResponse)
	}

	signResponse, err := service.Sing(ctx, &pb.SingRequest{
		Message:      []byte("message"),
		SecretCmkKey: secret,
	})
	if err != nil {
		t.Fatalf("unexpected sign error: %v", err)
	}
	if repository.signSecret != secret || repository.signMessage != "message" || string(signResponse.GetSignature()) != "signature" || signResponse.GetKeyId() != idKey.String() {
		t.Fatalf("unexpected sign result: response=%#v repo=%#v", signResponse, repository)
	}

	verifyResponse, err := service.Verify(ctx, &pb.VerifyRequest{
		Message:      []byte("message"),
		Signature:    []byte("signature"),
		Uid:          "request-key",
		SecretCmkKey: secret,
	})
	if err != nil {
		t.Fatalf("unexpected verify error: %v", err)
	}
	if repository.verifySecret != secret || repository.verifyMessage != "message" || repository.verifySignature != "signature" || !verifyResponse.GetValid() || verifyResponse.GetKeyId() != "request-key" {
		t.Fatalf("unexpected verify result: response=%#v repo=%#v", verifyResponse, repository)
	}

	claims, err := structpb.NewStruct(map[string]any{"sub": "alice"})
	if err != nil {
		t.Fatalf("unexpected claims setup error: %v", err)
	}
	jwtResponse, err := service.CreateJWT(ctx, &pb.CreateJWTRequest{
		SecretCmkKey: secret,
		Algorithm:    "HS256",
		Claims:       claims,
	})
	if err != nil {
		t.Fatalf("unexpected create jwt error: %v", err)
	}
	if repository.jwtSecret != secret || repository.jwtAlg != "HS256" || jwtResponse.GetToken() != "token" {
		t.Fatalf("unexpected create jwt result: response=%#v repo=%#v", jwtResponse, repository)
	}
	if gotClaims := repository.jwtClaims.(map[string]any); gotClaims["sub"] != "alice" {
		t.Fatalf("unexpected jwt claims: %#v", gotClaims)
	}

	verifyJWTResponse, err := service.VerifyJWT(ctx, &pb.VerifyJWTRequest{SecretCmkKey: secret, Algorithm: "HS256", Token: "token"})
	if err != nil {
		t.Fatalf("unexpected verify jwt error: %v", err)
	}
	if repository.verifyJWTSecret != secret || repository.verifyJWTAlg != "HS256" || repository.verifyJWTToken != "token" || !verifyJWTResponse.GetValid() {
		t.Fatalf("unexpected verify jwt result: response=%#v repo=%#v", verifyJWTResponse, repository)
	}

	readJWTResponse, err := service.ReadJWT(ctx, &pb.ReadJWTRequest{SecretCmkKey: secret, Algorithm: "HS256", Token: "token"})
	if err != nil {
		t.Fatalf("unexpected read jwt error: %v", err)
	}
	if repository.readJWTSecret != secret || repository.readJWTAlg != "HS256" || repository.readJWTToken != "token" || readJWTResponse.GetClaims().AsMap()["sub"] != "alice" {
		t.Fatalf("unexpected read jwt result: response=%#v repo=%#v", readJWTResponse, repository)
	}

}

func TestKeyServiceValidationAndErrors(t *testing.T) {
	ctx := context.Background()
	repository := &fakeCMKRepository{statusErr: errors.New("status failed")}
	service := NewKeyService(repository)

	if _, err := service.Status(ctx, &pb.StatusRequest{}); grpcstatus.Code(err) != codes.Internal {
		t.Fatalf("expected internal status error, got %v", err)
	}
	if _, err := service.CreateKey(ctx, &pb.CreateKeyRequest{Algorithm: pb.KeyAlgorithm_KEY_ALGORITHM_UNSPECIFIED, Purpose: pb.KeyPurpose_KEY_PURPOSE_ENCRYPT}); grpcstatus.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid algorithm error, got %v", err)
	}
	if _, err := service.CreateKey(ctx, &pb.CreateKeyRequest{Algorithm: pb.KeyAlgorithm_KEY_ALGORITHM_SYMMETRIC_DEFAULT, Purpose: pb.KeyPurpose_KEY_PURPOSE_UNSPECIFIED}); grpcstatus.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid purpose error, got %v", err)
	}
	if _, err := service.CreateKey(ctx, &pb.CreateKeyRequest{Algorithm: pb.KeyAlgorithm_KEY_ALGORITHM_SYMMETRIC_DEFAULT, Purpose: pb.KeyPurpose_KEY_PURPOSE_ENCRYPT, EventType: pb.KeyEventType(99)}); grpcstatus.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid event type error, got %v", err)
	}
	if _, err := service.RotateWrapKey(ctx, &pb.RotateWrapKeyRequest{IdCmkWrappingKeyRef: "bad"}); grpcstatus.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid wrapping id error, got %v", err)
	}

	repository = &fakeCMKRepository{createErr: errors.New("create failed")}
	service = NewKeyService(repository)
	if _, err := service.CreateKey(ctx, &pb.CreateKeyRequest{Algorithm: pb.KeyAlgorithm_KEY_ALGORITHM_SYMMETRIC_DEFAULT, Purpose: pb.KeyPurpose_KEY_PURPOSE_ENCRYPT}); grpcstatus.Code(err) != codes.Internal {
		t.Fatalf("expected internal create error, got %v", err)
	}

	repository = &fakeCMKRepository{readJWTInvalidClaim: true}
	service = NewKeyService(repository)
	if _, err := service.ReadJWT(ctx, &pb.ReadJWTRequest{}); grpcstatus.Code(err) != codes.Internal {
		t.Fatalf("expected read jwt claim conversion error, got %v", err)
	}
}

func TestKeyServiceOperationErrors(t *testing.T) {
	ctx := context.Background()
	idWrap := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	expected := errors.New("operation failed")

	tests := []struct {
		name       string
		repository *fakeCMKRepository
		call       func(pb.KeyServiceServer) error
	}{
		{
			name:       "rotate key",
			repository: &fakeCMKRepository{rotateErr: expected},
			call: func(service pb.KeyServiceServer) error {
				_, err := service.RotateKey(ctx, &pb.RotateKeyRequest{})
				return err
			},
		},
		{
			name:       "rotate wrap key",
			repository: &fakeCMKRepository{rotateWrapErr: expected},
			call: func(service pb.KeyServiceServer) error {
				_, err := service.RotateWrapKey(ctx, &pb.RotateWrapKeyRequest{IdCmkWrappingKeyRef: idWrap.String()})
				return err
			},
		},
		{
			name:       "encrypt",
			repository: &fakeCMKRepository{encryptErr: expected},
			call: func(service pb.KeyServiceServer) error {
				_, err := service.Encrypt(ctx, &pb.EncryptRequest{})
				return err
			},
		},
		{
			name:       "decrypt",
			repository: &fakeCMKRepository{decryptErr: expected},
			call: func(service pb.KeyServiceServer) error {
				_, err := service.Decrypt(ctx, &pb.DecryptRequest{})
				return err
			},
		},
		{
			name:       "sign",
			repository: &fakeCMKRepository{signErr: expected},
			call: func(service pb.KeyServiceServer) error {
				_, err := service.Sing(ctx, &pb.SingRequest{})
				return err
			},
		},
		{
			name:       "verify",
			repository: &fakeCMKRepository{verifyErr: expected},
			call: func(service pb.KeyServiceServer) error {
				_, err := service.Verify(ctx, &pb.VerifyRequest{})
				return err
			},
		},
		{
			name:       "create jwt",
			repository: &fakeCMKRepository{jwtErr: expected},
			call: func(service pb.KeyServiceServer) error {
				_, err := service.CreateJWT(ctx, &pb.CreateJWTRequest{})
				return err
			},
		},
		{
			name:       "verify jwt",
			repository: &fakeCMKRepository{verifyJWTErr: expected},
			call: func(service pb.KeyServiceServer) error {
				_, err := service.VerifyJWT(ctx, &pb.VerifyJWTRequest{})
				return err
			},
		},
		{
			name:       "read jwt",
			repository: &fakeCMKRepository{readJWTErr: expected},
			call: func(service pb.KeyServiceServer) error {
				_, err := service.ReadJWT(ctx, &pb.ReadJWTRequest{})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(NewKeyService(tt.repository)); grpcstatus.Code(err) != codes.Internal {
				t.Fatalf("expected internal error, got %v", err)
			}
		})
	}
}

func TestGRPCHelpers(t *testing.T) {
	algorithms := map[pb.KeyAlgorithm]commonEntity.KeyType{
		pb.KeyAlgorithm_KEY_ALGORITHM_SYMMETRIC_DEFAULT:   commonEntity.KeySymmetricDefault,
		pb.KeyAlgorithm_KEY_ALGORITHM_RSA_OAEP:            commonEntity.KeyTypeRSAOAEP,
		pb.KeyAlgorithm_KEY_ALGORITHM_RSA_PKCS1V15_SHA256: commonEntity.KeyTypeRSAPKCS1v15SHA256,
		pb.KeyAlgorithm_KEY_ALGORITHM_ECDH:                commonEntity.KeyTypeECDH,
		pb.KeyAlgorithm_KEY_ALGORITHM_EDDSA:               commonEntity.KeyTypeEdDSA,
	}
	for input, want := range algorithms {
		got, err := keyAlgorithm(input)
		if err != nil || got != want {
			t.Fatalf("unexpected algorithm mapping for %s: %s %v", input, got, err)
		}
	}

	purposes := map[pb.KeyPurpose]commonEntity.KeyPurpose{
		pb.KeyPurpose_KEY_PURPOSE_SIGN:    commonEntity.KeyPurposeSign,
		pb.KeyPurpose_KEY_PURPOSE_ENCRYPT: commonEntity.KeyPurposeEncrypt,
		pb.KeyPurpose_KEY_PURPOSE_WRAP:    commonEntity.KeyPurposeWrap,
	}
	for input, want := range purposes {
		got, err := keyPurpose(input)
		if err != nil || got != want {
			t.Fatalf("unexpected purpose mapping for %s: %s %v", input, got, err)
		}
	}

	events := map[pb.KeyEventType]commonEntity.EventType{
		pb.KeyEventType_KEY_EVENT_TYPE_UNSPECIFIED: "",
		pb.KeyEventType_KEY_EVENT_TYPE_CREATE_KEY:  commonEntity.EventTypeCreateKey,
		pb.KeyEventType_KEY_EVENT_TYPE_ROTATE_KEY:  commonEntity.EventTypeRotateKey,
	}
	for input, want := range events {
		got, err := keyEventType(input)
		if err != nil || got != want {
			t.Fatalf("unexpected event mapping for %s: %s %v", input, got, err)
		}
	}

	if got := optionalString(""); got != nil {
		t.Fatalf("expected nil optional string, got %v", *got)
	}
	if got := optionalString("value"); got == nil || *got != "value" {
		t.Fatalf("expected optional string value, got %v", got)
	}
	if err := serviceError(nil); err != nil {
		t.Fatalf("unexpected nil service error: %v", err)
	}
	notFoundErr := grpcstatus.Error(codes.NotFound, "missing")
	if err := serviceError(notFoundErr); grpcstatus.Code(err) != codes.NotFound {
		t.Fatalf("expected status error passthrough, got %v", err)
	}
	if wrappingKeyInfo(nil) != nil {
		t.Fatal("expected nil wrapping key info")
	}
	if got := secretCmkKey("", "", ""); got != "" {
		t.Fatalf("expected empty secret, got %s", got)
	}
}
