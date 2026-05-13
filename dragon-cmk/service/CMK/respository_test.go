package cmk

import (
	"context"
	"encoding/base64"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	serverGin "github.com/PointerByte/GoForge/config/server/gin"
	commonEncrypt "github.com/PointerByte/GoForge/encrypt/common"
	modelsEncrypt "github.com/PointerByte/GoForge/encrypt/models"
	"github.com/PointerByte/GoForge/logger/builder"
	jwtservice "github.com/PointerByte/GoForge/security/auth/jwt"
	"github.com/PointerByte/GoForge/tools/jobs"
	appCommon "github.com/PointerByte/lock-max/dragon-cmk/common"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/store"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/views"
	kek "github.com/PointerByte/lock-max/dragon-cmk/service/KEK"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	testIDKey              = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testIDVersion          = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testIDVersionSecondary = uuid.MustParse("22222222-2222-2222-2222-222222222223")
	testIDQueue            = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	testIDWrap             = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	testSecretRaw          = testIDKey.String() + "." + testIDVersion.String()
	testSecret             = base64.StdEncoding.EncodeToString([]byte(testSecretRaw))
)

func assertEncodedSecret(t *testing.T, secret string) (uuid.UUID, uuid.UUID) {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		t.Fatalf("expected base64 encoded secret, got %q: %v", secret, err)
	}

	parts := strings.Split(string(decoded), ".")
	if len(parts) != 2 {
		t.Fatalf("expected decoded secret with 2 parts, got %q", decoded)
	}

	idCmkKey, err := uuid.Parse(parts[0])
	if err != nil {
		t.Fatalf("expected decoded cmk key uuid, got %q: %v", parts[0], err)
	}

	idCmkKeyVersion, err := uuid.Parse(parts[1])
	if err != nil {
		t.Fatalf("expected decoded cmk key version uuid, got %q: %v", parts[1], err)
	}

	return idCmkKey, idCmkKeyVersion
}

type fakeKEKRepository struct {
	kekData          *models.KEK
	getErr           error
	createFn         kek.HandlerFuncCreteKey
	rotateFn         kek.HandlerFuncRotateKey
	rotateWrapFn     kek.HandlerFuncRotateWrapKey
	deleteWrapFn     kek.HandlerFuncDeleteWrapKey
	createKEKErr     error
	rotateKEKErr     error
	deleteKeyErr     error
	createKEKCalls   int
	rotateKEKCalls   int
	deleteKeyCalls   int
	requestedVersion []string
}

func (f *fakeKEKRepository) SetFuncCreateKey(fn kek.HandlerFuncCreteKey) {
	f.createFn = fn
}

func (f *fakeKEKRepository) SetFuncRotate(fn kek.HandlerFuncRotateKey) {
	f.rotateFn = fn
}

func (f *fakeKEKRepository) SetFuncRotateWrapKey(fn kek.HandlerFuncRotateWrapKey) {
	f.rotateWrapFn = fn
}

func (f *fakeKEKRepository) SetFuncDeleteWrapKey(fn kek.HandlerFuncDeleteWrapKey) {
	f.deleteWrapFn = fn
}

func (f *fakeKEKRepository) CreateKEK(id uuid.UUID, _ string, _ string) (*uuid.UUID, error) {
	f.createKEKCalls++
	if id == uuid.Nil {
		id = uuid.New()
	}
	return &id, f.createKEKErr
}

func (f *fakeKEKRepository) GetKEK(_ uuid.UUID, version string) (*models.KEK, error) {
	f.requestedVersion = append(f.requestedVersion, version)
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.kekData == nil {
		return nil, errors.New("missing kek")
	}
	copy := *f.kekData
	if version != "" {
		copy.Version = version
	}
	return &copy, nil
}

func (f *fakeKEKRepository) RotateKEK(id uuid.UUID, _ string) (*uuid.UUID, error) {
	f.rotateKEKCalls++
	if id == uuid.Nil {
		id = uuid.New()
	}
	return &id, f.rotateKEKErr
}

func (f *fakeKEKRepository) ListKEK(uuid.UUID, uint, uint) (*models.PaginatedKEK, error) {
	if f.kekData == nil {
		return &models.PaginatedKEK{}, f.getErr
	}
	return &models.PaginatedKEK{
		Results: []models.KEK{*f.kekData},
		Pagination: models.Pagination{
			TotalRegisters:     1,
			TotalPages:         1,
			TotalRegistersPage: 1,
			PageNow:            1,
		},
	}, f.getErr
}

func (f *fakeKEKRepository) DeleteKey(uuid.UUID, string) error {
	f.deleteKeyCalls++
	return f.deleteKeyErr
}

type fakeStoreRepository struct {
	createCmkKeyErr             error
	updateCmkKeyErr             error
	deleteCmkKeyErr             error
	createQueueErr              error
	updateQueueErr              error
	deleteQueueErr              error
	createVersionErr            error
	updateVersionMetadataErr    error
	updateVersionStatusErr      error
	retireVersionErr            error
	deleteRetiredVersionErr     error
	rotateVersionErr            error
	createWrappingKeyRefErr     error
	updateWrappingKeyRefErr     error
	deleteWrappingKeyRefErr     error
	createCmkKeyInput           *store.CreateCmkKeyInput
	updateCmkKeyInputs          []store.UpdateCmkKeyInput
	createQueueInput            *store.CreateKeyCreationQueueInput
	updateQueueInputs           []store.UpdateKeyCreationQueueInput
	createVersionInput          *store.CreateKeyVersionInput
	updateVersionMetadataInputs []store.UpdateKeyVersionMetadataInput
	updateVersionStatusInputs   []store.UpdateKeyVersionStatusInput
	rotateVersionInput          *store.RotateKeyVersionInput
	createWrappingKeyRefInput   *store.CreateWrappingKeyRefInput
	updateWrappingKeyRefInputs  []store.UpdateWrappingKeyRefInput
	deletedCmkKey               *uuid.UUID
	deletedQueue                *uuid.UUID
	retiredVersion              *uuid.UUID
	deletedRetiredVersion       *uuid.UUID
	deletedWrappingKeyRef       *uuid.UUID
}

func commandTag() *pgconn.CommandTag {
	tag := pgconn.NewCommandTag("CALL")
	return &tag
}

func (f *fakeStoreRepository) CreateCmkKey(input store.CreateCmkKeyInput) (*pgconn.CommandTag, error) {
	f.createCmkKeyInput = &input
	return commandTag(), f.createCmkKeyErr
}

func (f *fakeStoreRepository) UpdateCmkKey(input store.UpdateCmkKeyInput) (*pgconn.CommandTag, error) {
	f.updateCmkKeyInputs = append(f.updateCmkKeyInputs, input)
	return commandTag(), f.updateCmkKeyErr
}

func (f *fakeStoreRepository) DeleteCmkKey(idCmkKey uuid.UUID) (*pgconn.CommandTag, error) {
	f.deletedCmkKey = &idCmkKey
	return commandTag(), f.deleteCmkKeyErr
}

func (f *fakeStoreRepository) CreateKeyCreationQueue(input store.CreateKeyCreationQueueInput) (*pgconn.CommandTag, error) {
	f.createQueueInput = &input
	return commandTag(), f.createQueueErr
}

func (f *fakeStoreRepository) UpdateKeyCreationQueue(input store.UpdateKeyCreationQueueInput) (*pgconn.CommandTag, error) {
	f.updateQueueInputs = append(f.updateQueueInputs, input)
	return commandTag(), f.updateQueueErr
}

func (f *fakeStoreRepository) DeleteKeyCreationQueue(idCmkKeyCreationQueue uuid.UUID) (*pgconn.CommandTag, error) {
	f.deletedQueue = &idCmkKeyCreationQueue
	return commandTag(), f.deleteQueueErr
}

func (f *fakeStoreRepository) CreateKeyVersion(input store.CreateKeyVersionInput) (*pgconn.CommandTag, error) {
	f.createVersionInput = &input
	return commandTag(), f.createVersionErr
}

func (f *fakeStoreRepository) UpdateKeyVersionMetadata(input store.UpdateKeyVersionMetadataInput) (*pgconn.CommandTag, error) {
	f.updateVersionMetadataInputs = append(f.updateVersionMetadataInputs, input)
	return commandTag(), f.updateVersionMetadataErr
}

func (f *fakeStoreRepository) UpdateKeyVersionStatus(input store.UpdateKeyVersionStatusInput) (*pgconn.CommandTag, error) {
	f.updateVersionStatusInputs = append(f.updateVersionStatusInputs, input)
	return commandTag(), f.updateVersionStatusErr
}

func (f *fakeStoreRepository) RetireKeyVersion(idCmkKeyVersion uuid.UUID) (*pgconn.CommandTag, error) {
	f.retiredVersion = &idCmkKeyVersion
	return commandTag(), f.retireVersionErr
}

func (f *fakeStoreRepository) DeleteRetiredKeyVersion(idCmkKeyVersion uuid.UUID) (*pgconn.CommandTag, error) {
	f.deletedRetiredVersion = &idCmkKeyVersion
	return commandTag(), f.deleteRetiredVersionErr
}

func (f *fakeStoreRepository) RotateKeyVersion(input store.RotateKeyVersionInput) (*pgconn.CommandTag, error) {
	f.rotateVersionInput = &input
	return commandTag(), f.rotateVersionErr
}

func (f *fakeStoreRepository) CreateWrappingKeyRef(input store.CreateWrappingKeyRefInput) (*pgconn.CommandTag, error) {
	f.createWrappingKeyRefInput = &input
	return commandTag(), f.createWrappingKeyRefErr
}

func (f *fakeStoreRepository) UpdateWrappingKeyRef(input store.UpdateWrappingKeyRefInput) (*pgconn.CommandTag, error) {
	f.updateWrappingKeyRefInputs = append(f.updateWrappingKeyRefInputs, input)
	return commandTag(), f.updateWrappingKeyRefErr
}

func (f *fakeStoreRepository) DeleteWrappingKeyRef(idCmkWrappingKeyRef uuid.UUID) (*pgconn.CommandTag, error) {
	f.deletedWrappingKeyRef = &idCmkWrappingKeyRef
	return commandTag(), f.deleteWrappingKeyRefErr
}

type fakeViewsRepository struct {
	keys            []views.CmkKeyView
	queues          []views.CmkCreationKeyQueueView
	versions        []views.CmkKeyVersionView
	wrappingRefs    []views.CmkWrappingKeyRefView
	keyErr          error
	queueErr        error
	versionErr      error
	wrappingRefErr  error
	providerWrapErr error
	matchAnyQueue   bool
	keyQueries      []string
	queueQueries    []string
	versionQueries  []string
	wrapQueries     []string
}

func (f *fakeViewsRepository) ReadCmkKeyView() ([]views.CmkKeyView, error) {
	return f.keys, f.keyErr
}

func (f *fakeViewsRepository) QueryCmkKeyView(query string, args ...any) ([]views.CmkKeyView, error) {
	f.keyQueries = append(f.keyQueries, query)
	if f.keyErr != nil {
		return nil, f.keyErr
	}
	if strings.Contains(query, "id_cmk_key = $1") && len(args) > 0 {
		id, ok := args[0].(uuid.UUID)
		if ok {
			for _, key := range f.keys {
				if key.IDCmkKey == id {
					return []views.CmkKeyView{key}, nil
				}
			}
			return nil, nil
		}
	}
	return f.keys, f.keyErr
}

func (f *fakeViewsRepository) ReadCmkCreationKeyQueueView() ([]views.CmkCreationKeyQueueView, error) {
	return f.queues, f.queueErr
}

func (f *fakeViewsRepository) QueryCmkCreationKeyQueueView(query string, args ...any) ([]views.CmkCreationKeyQueueView, error) {
	f.queueQueries = append(f.queueQueries, query)
	if f.queueErr != nil {
		return nil, f.queueErr
	}
	if strings.Contains(query, "ORDER BY queued_at") {
		matches := f.filterQueues(args...)
		return paginateQueues(matches, args...), nil
	}
	if len(args) > 0 {
		for _, arg := range args {
			id, ok := arg.(uuid.UUID)
			if !ok || id == uuid.Nil {
				continue
			}
			for _, queue := range f.queues {
				if queue.IDCmkKey == id || queue.IDCmkKeyCreationQueue == id {
					return []views.CmkCreationKeyQueueView{queue}, nil
				}
			}
		}
		if f.matchAnyQueue && len(f.queues) > 0 {
			return []views.CmkCreationKeyQueueView{f.queues[0]}, nil
		}
		return nil, nil
	}
	return f.queues, f.queueErr
}

func (f *fakeViewsRepository) CountCmkCreationKeyQueueView(query string, args ...any) (uint, error) {
	f.queueQueries = append(f.queueQueries, query)
	if f.queueErr != nil {
		return 0, f.queueErr
	}
	return uint(len(f.filterQueues(args...))), nil
}

func (f *fakeViewsRepository) filterQueues(args ...any) []views.CmkCreationKeyQueueView {
	matches := append([]views.CmkCreationKeyQueueView{}, f.queues...)
	for _, arg := range args {
		switch value := arg.(type) {
		case uuid.UUID:
			if value == uuid.Nil {
				continue
			}
			filtered := make([]views.CmkCreationKeyQueueView, 0, len(matches))
			for _, queue := range matches {
				if queue.IDCmkKey == value || queue.IDCmkKeyCreationQueue == value {
					filtered = append(filtered, queue)
				}
			}
			matches = filtered
		case commonEntity.QueueStatus:
			if value == "" {
				continue
			}
			filtered := make([]views.CmkCreationKeyQueueView, 0, len(matches))
			for _, queue := range matches {
				if queue.Status == value {
					filtered = append(filtered, queue)
				}
			}
			matches = filtered
		}
	}
	return matches
}

func paginateQueues(values []views.CmkCreationKeyQueueView, args ...any) []views.CmkCreationKeyQueueView {
	if len(args) < 2 {
		return values
	}
	limit, okLimit := args[len(args)-2].(int)
	offset, okOffset := args[len(args)-1].(int)
	if !okLimit || !okOffset {
		return values
	}
	if offset >= len(values) {
		return nil
	}
	end := offset + limit
	if end > len(values) {
		end = len(values)
	}
	return values[offset:end]
}

func (f *fakeViewsRepository) ReadCmkKeyCreationQueueView() ([]views.CmkKeyCreationQueueView, error) {
	return f.ReadCmkCreationKeyQueueView()
}

func (f *fakeViewsRepository) QueryCmkKeyCreationQueueView(query string, args ...any) ([]views.CmkKeyCreationQueueView, error) {
	return f.QueryCmkCreationKeyQueueView(query, args...)
}

func (f *fakeViewsRepository) ReadCmkKeyVersionView() ([]views.CmkKeyVersionView, error) {
	return f.versions, f.versionErr
}

func (f *fakeViewsRepository) QueryCmkKeyVersionView(query string, args ...any) ([]views.CmkKeyVersionView, error) {
	f.versionQueries = append(f.versionQueries, query)
	if f.versionErr != nil {
		return nil, f.versionErr
	}
	if strings.Contains(query, "id_cmk_key_version = $1") && len(args) > 0 {
		id, ok := args[0].(*uuid.UUID)
		if ok && id != nil {
			for _, version := range f.versions {
				if version.IDCmkKeyVersion == *id {
					return []views.CmkKeyVersionView{version}, nil
				}
			}
			return nil, nil
		}
	}
	if strings.Contains(query, "id_cmk_wrapping_key_ref = $1") && len(args) > 0 {
		id, ok := args[0].(uuid.UUID)
		if ok {
			matches := make([]views.CmkKeyVersionView, 0)
			for _, version := range f.versions {
				if version.IDCmkWrappingKeyRef != nil && *version.IDCmkWrappingKeyRef == id {
					matches = append(matches, version)
				}
			}
			return matches, nil
		}
	}
	return f.versions, f.versionErr
}

func (f *fakeViewsRepository) ReadCmkWrappingKeyRefView() ([]views.CmkWrappingKeyRefView, error) {
	return f.wrappingRefs, f.wrappingRefErr
}

func (f *fakeViewsRepository) QueryCmkWrappingKeyRefView(query string, args ...any) ([]views.CmkWrappingKeyRefView, error) {
	f.wrapQueries = append(f.wrapQueries, query)
	if f.wrappingRefErr != nil {
		return nil, f.wrappingRefErr
	}
	if strings.Contains(query, "id_cmk_wrapping_key_ref = $1") && len(args) > 0 {
		id, ok := args[0].(uuid.UUID)
		if ok {
			for _, wrappingRef := range f.wrappingRefs {
				if wrappingRef.IDCmkWrappingKeyRef == id {
					return []views.CmkWrappingKeyRefView{wrappingRef}, nil
				}
			}
			return nil, nil
		}
	}
	if strings.Contains(query, "provider = $1") && len(args) >= 3 {
		if f.providerWrapErr != nil {
			return nil, f.providerWrapErr
		}
		provider, _ := args[0].(string)
		keyRef, _ := args[1].(string)
		version, _ := args[2].(string)
		for _, wrappingRef := range f.wrappingRefs {
			if wrappingRef.Provider == provider && wrappingRef.KeyRef == keyRef && wrappingRef.Version == version {
				return []views.CmkWrappingKeyRefView{wrappingRef}, nil
			}
		}
		return nil, nil
	}
	return f.wrappingRefs, f.wrappingRefErr
}

type fakeSecurityRepository struct {
	err         error
	generateErr error
	encryptErr  error
	decryptErr  error
	signErr     error
	verifyErr   error
}

var securityContextRecorder func(context.Context)

func recordSecurityContext(ctx context.Context) {
	if securityContextRecorder != nil {
		securityContextRecorder(ctx)
	}
}

func firstErr(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func (f fakeSecurityRepository) GenerateSymetrycKeys(ctx context.Context, _ commonEncrypt.SizeSymetrycKey) (*modelsEncrypt.KeyData, error) {
	recordSecurityContext(ctx)
	return &modelsEncrypt.KeyData{PublicKey: "sym-public", KeyID: "sym-key"}, firstErr(f.generateErr, f.err)
}

func (f fakeSecurityRepository) EncryptAES(ctx context.Context, secretKey, value string, additional *string) (string, error) {
	recordSecurityContext(ctx)
	return "aes:" + value, firstErr(f.encryptErr, f.err)
}

func (f fakeSecurityRepository) DecryptAES(ctx context.Context, secretKey, cipherValue string, additional *string) (string, error) {
	recordSecurityContext(ctx)
	return "plain:" + cipherValue, firstErr(f.decryptErr, f.err)
}

func (f fakeSecurityRepository) GenerateRSAKeys(ctx context.Context, _ commonEncrypt.SizeAsymetrycKey) (*modelsEncrypt.KeyData, error) {
	recordSecurityContext(ctx)
	return &modelsEncrypt.KeyData{PublicKey: "rsa-public", KeyID: "rsa-private"}, firstErr(f.generateErr, f.err)
}

func (f fakeSecurityRepository) GenerateECCKeys(ctx context.Context, _ commonEncrypt.CurveAsymmetricKey) (*modelsEncrypt.KeyData, error) {
	recordSecurityContext(ctx)
	return &modelsEncrypt.KeyData{PublicKey: "ecc-public", KeyID: "ecc-private"}, firstErr(f.generateErr, f.err)
}

func (f fakeSecurityRepository) RSA_OAEP_Encode(ctx context.Context, publicKey, text string) (string, error) {
	recordSecurityContext(ctx)
	return "rsa:" + text, firstErr(f.encryptErr, f.err)
}

func (f fakeSecurityRepository) RSA_OAEP_Decode(ctx context.Context, privateKey, cipherText string) (string, error) {
	recordSecurityContext(ctx)
	return "rsa-plain:" + cipherText, firstErr(f.decryptErr, f.err)
}

func (f fakeSecurityRepository) ECC_Encode(ctx context.Context, publicKey, text string) (string, error) {
	recordSecurityContext(ctx)
	return "ecc:" + text, firstErr(f.encryptErr, f.err)
}

func (f fakeSecurityRepository) ECC_Decode(ctx context.Context, privateKey, cipherText string) (string, error) {
	recordSecurityContext(ctx)
	return "ecc-plain:" + cipherText, firstErr(f.decryptErr, f.err)
}

func (f fakeSecurityRepository) HMAC(ctx context.Context, secretKey, message string) string {
	recordSecurityContext(ctx)
	return "hmac"
}

func (f fakeSecurityRepository) Sha256Hex(ctx context.Context, message string) string {
	recordSecurityContext(ctx)
	return "sha256"
}

func (f fakeSecurityRepository) Blake3(ctx context.Context, message string) string {
	recordSecurityContext(ctx)
	return "blake3"
}

func (f fakeSecurityRepository) GenerateEd255Keys(ctx context.Context) (*modelsEncrypt.KeyData, error) {
	recordSecurityContext(ctx)
	return &modelsEncrypt.KeyData{PublicKey: "ed-public", KeyID: "ed-private"}, firstErr(f.generateErr, f.err)
}

func (f fakeSecurityRepository) SignEd25519(ctx context.Context, privateKey, text string) (string, error) {
	recordSecurityContext(ctx)
	return "ed-signature", firstErr(f.signErr, f.err)
}

func (f fakeSecurityRepository) VerifyEd25519(ctx context.Context, publicKey, text, signature string) error {
	recordSecurityContext(ctx)
	return firstErr(f.verifyErr, f.err)
}

func (f fakeSecurityRepository) SignRSAPSS(ctx context.Context, privateKey, text string) (string, error) {
	recordSecurityContext(ctx)
	return "pss-signature", firstErr(f.signErr, f.err)
}

func (f fakeSecurityRepository) VerifyRSAPSS(ctx context.Context, publicKey, text, signature string) error {
	recordSecurityContext(ctx)
	return firstErr(f.verifyErr, f.err)
}

func (f fakeSecurityRepository) Sign_RSA_PKCS1v15_SHA256(ctx context.Context, privateKey, data string) (string, error) {
	recordSecurityContext(ctx)
	return "pkcs-signature", firstErr(f.signErr, f.err)
}

func (f fakeSecurityRepository) Verify_RSA_PKCS1v15_SHA256(ctx context.Context, data, publicKey string, signature string) error {
	recordSecurityContext(ctx)
	return firstErr(f.verifyErr, f.err)
}

type fakeJWTService struct {
	createErr error
	verifyErr error
	readErr   error
}

var jwtContextRecorder func(context.Context)

func (f fakeJWTService) Create(claims any) (string, error) {
	return "jwt-token", f.createErr
}

func (f fakeJWTService) CreateWithContext(ctx context.Context, claims any) (string, error) {
	if jwtContextRecorder != nil {
		jwtContextRecorder(ctx)
	}
	return "jwt-token", f.createErr
}

func (f fakeJWTService) ValidateSignatureWithContext(ctx context.Context, token string) error {
	if jwtContextRecorder != nil {
		jwtContextRecorder(ctx)
	}
	return f.verifyErr
}

func (f fakeJWTService) Read(token string, destination any) error {
	return f.readErr
}

func (f fakeJWTService) ReadWithContext(ctx context.Context, token string, destination any) error {
	if jwtContextRecorder != nil {
		jwtContextRecorder(ctx)
	}
	return f.readErr
}

func testKEKData() *models.KEK {
	return &models.KEK{
		IdCmkWrappingKeyRef: testIDWrap,
		SecretCmkKey:        testSecret,
		PublicKey:           "wrap-public",
		PrivateKey:          "wrap-private",
		KeyRef:              "local",
		Provider:            "local",
		Version:             "v1",
	}
}

func testViews(algorithm commonEntity.KeyType, purpose commonEntity.KeyPurpose, status commonEntity.KeyStatus) *fakeViewsRepository {
	return &fakeViewsRepository{
		keys: []views.CmkKeyView{{
			IDCmkKey:        testIDKey,
			Algorithm:       algorithm,
			Purpose:         purpose,
			Status:          status,
			IDCmkKeyVersion: &testIDVersion,
		}},
		queues: []views.CmkCreationKeyQueueView{{
			IDCmkKeyCreationQueue: testIDQueue,
			IDCmkKey:              testIDKey,
			EventType:             commonEntity.EventTypeCreateKey,
			Status:                commonEntity.QueueStatusProcessed,
		}},
		versions: []views.CmkKeyVersionView{{
			IDCmkKeyVersion:     testIDVersion,
			IDCmkKey:            testIDKey,
			VersionNumber:       1,
			Size:                256,
			Status:              commonEntity.KeyVersionStatusEnabled,
			KID:                 "public-key",
			SecretWrapped:       "wrapped-key",
			WrapAlg:             string(commonEntity.KeyTypeECDH),
			IDCmkWrappingKeyRef: &testIDWrap,
			SecretChecksum:      stringPtr("checksum"),
		}},
		wrappingRefs: []views.CmkWrappingKeyRefView{{
			IDCmkWrappingKeyRef: testIDWrap,
			Provider:            "local",
			KeyRef:              "local",
			Version:             "v1",
		}},
	}
}

func stringPtr(value string) *string {
	return &value
}

func newTestRepository(algorithm commonEntity.KeyType, purpose commonEntity.KeyPurpose, status commonEntity.KeyStatus) (*Repository, *fakeStoreRepository, *fakeViewsRepository, *fakeKEKRepository) {
	sp := &fakeStoreRepository{}
	viewRepo := testViews(algorithm, purpose, status)
	kekRepo := &fakeKEKRepository{kekData: testKEKData()}
	repo := &Repository{
		ctx:      context.Background(),
		kek:      kekRepo,
		sp:       sp,
		views:    viewRepo,
		security: fakeSecurityRepository{},
	}
	return repo, sp, viewRepo, kekRepo
}

func assertCommonTimeoutContext(t *testing.T, ctx context.Context) {
	t.Helper()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context deadline")
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		t.Fatalf("expected active timeout context, remaining=%s", remaining)
	}
	if remaining > appCommon.Timeout {
		t.Fatalf("expected timeout up to %s, got %s", appCommon.Timeout, remaining)
	}
	if remaining < appCommon.Timeout-time.Second {
		t.Fatalf("expected timeout close to %s, got %s", appCommon.Timeout, remaining)
	}
}

func TestNewRepositoryWiresKEKCallbacks(t *testing.T) {
	kekRepo := &fakeKEKRepository{kekData: testKEKData()}
	repo := NewRepository(context.Background(), builder.New(context.Background()), fakeSecurityRepository{}, kekRepo)
	if repo == nil {
		t.Fatal("expected repository")
	}
	if kekRepo.createFn == nil {
		t.Fatal("expected create callback to be registered")
	}
	if kekRepo.rotateFn == nil {
		t.Fatal("expected rotate callback to be registered")
	}
	if kekRepo.rotateWrapFn == nil {
		t.Fatal("expected rotate wrap callback to be registered")
	}
	if kekRepo.deleteWrapFn == nil {
		t.Fatal("expected delete wrap callback to be registered")
	}
}

func TestRepositoryUsesCommonTimeoutContext(t *testing.T) {
	recorded := 0
	securityContextRecorder = func(ctx context.Context) {
		recorded++
		assertCommonTimeoutContext(t, ctx)
	}
	t.Cleanup(func() {
		securityContextRecorder = nil
	})

	repo, _, _, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	if _, err := repo.generateKey(commonEntity.KeySymmetricDefault, 256); err != nil {
		t.Fatalf("unexpected generate key error: %v", err)
	}
	if _, _, _, err := repo.wrapKey("key"); err != nil {
		t.Fatalf("unexpected wrap key error: %v", err)
	}
	if _, _, err := repo.Encrypt(testSecret, "plain", nil); err != nil {
		t.Fatalf("unexpected encrypt error: %v", err)
	}
	if recorded == 0 {
		t.Fatal("expected security context calls to be recorded")
	}
}

func TestVerifyJWTUsesCommonTimeoutContext(t *testing.T) {
	originalNewJWTServiceFn := newJWTServiceFn
	newJWTServiceFn = func(input jwtservice.ConfigServiceInput) (jwtTokenService, error) {
		return fakeJWTService{}, nil
	}
	t.Cleanup(func() {
		newJWTServiceFn = originalNewJWTServiceFn
		jwtContextRecorder = nil
	})

	recorded := 0
	jwtContextRecorder = func(ctx context.Context) {
		recorded++
		assertCommonTimeoutContext(t, ctx)
	}

	repo, _, _, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeSign, commonEntity.KeyStatusEnabled)
	if err := repo.VerifyJWT(context.Background(), testSecret, "HS256", "token"); err != nil {
		t.Fatalf("unexpected verify jwt error: %v", err)
	}
	if recorded != 1 {
		t.Fatalf("expected one jwt context call, got %d", recorded)
	}
}

func TestSaveKeyWorker(t *testing.T) {
	ch := make(chan chKeyResult, 1)
	expected := &secret{secretWrapped: "wrapped"}
	saveKeyWorker(expected, nil, ch)
	result := <-ch
	if result.err != nil || result.secret != expected {
		t.Fatalf("unexpected result: %#v", result)
	}
	if _, ok := <-ch; ok {
		t.Fatal("expected closed channel")
	}

	ch = make(chan chKeyResult, 1)
	expectedErr := errors.New("worker failed")
	saveKeyWorker(nil, expectedErr, ch)
	result = <-ch
	if result.err != expectedErr || result.secret != nil {
		t.Fatalf("unexpected error result: %#v", result)
	}
}

func TestGenerateKey(t *testing.T) {
	repo, _, _, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)

	tests := []struct {
		name      string
		algorithm commonEntity.KeyType
		size      uint
		wantKeyID string
	}{
		{"symmetric", commonEntity.KeySymmetricDefault, 256, "sym-key"},
		{"rsa", commonEntity.KeyTypeRSAOAEP, 2048, "rsa-private"},
		{"ecdh-p256", commonEntity.KeyTypeECDH, 256, "ecc-private"},
		{"ecdh-p384", commonEntity.KeyTypeECDH, 384, "ecc-private"},
		{"ecdh-p521", commonEntity.KeyTypeECDH, 521, "ecc-private"},
		{"eddsa", commonEntity.KeyTypeEdDSA, 0, "ed-private"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := repo.generateKey(tt.algorithm, tt.size)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if key.KeyID != tt.wantKeyID {
				t.Fatalf("unexpected key id: %s", key.KeyID)
			}
		})
	}

	if _, err := repo.generateKey(commonEntity.KeyTypeECDH, 128); err == nil || err.Error() != errMsgUnsupportedECDHKeySize {
		t.Fatalf("expected unsupported ecdh size, got %v", err)
	}
	if _, err := repo.generateKey(commonEntity.KeyType("bad"), 0); err == nil || err.Error() != errMsgUnsupportedAlgorithm {
		t.Fatalf("expected unsupported algorithm, got %v", err)
	}

	repo.security = fakeSecurityRepository{err: errors.New("security failed")}
	if _, err := repo.generateKey(commonEntity.KeySymmetricDefault, 256); err == nil {
		t.Fatal("expected security error")
	}
	if _, err := repo.generateKey(commonEntity.KeyTypeRSAOAEP, 2048); err == nil {
		t.Fatal("expected rsa error")
	}
	if _, err := repo.generateKey(commonEntity.KeyTypeECDH, 256); err == nil {
		t.Fatal("expected ecc error")
	}
	if _, err := repo.generateKey(commonEntity.KeyTypeEdDSA, 0); err == nil {
		t.Fatal("expected eddsa error")
	}
}

func TestQueueAndSaveHelpers(t *testing.T) {
	repo, sp, viewRepo, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)

	idQueue, err := repo.createQueue(testIDKey, commonEntity.EventTypeCreateKey)
	if err != nil {
		t.Fatalf("unexpected queue error: %v", err)
	}
	if idQueue == nil || sp.createQueueInput == nil || sp.createQueueInput.IDCmkKey != testIDKey {
		t.Fatalf("unexpected queue input: %#v", sp.createQueueInput)
	}

	if err := repo.queueProcessingKey(testIDQueue, testIDKey); err != nil {
		t.Fatalf("unexpected processing error: %v", err)
	}
	if got := *sp.updateCmkKeyInputs[len(sp.updateCmkKeyInputs)-1].Status; got != commonEntity.KeyStatusPendingImport {
		t.Fatalf("unexpected processing status: %s", got)
	}

	if err := repo.queueProcessedKey(testIDQueue, testIDKey); err != nil {
		t.Fatalf("unexpected processed error: %v", err)
	}
	if got := *sp.updateCmkKeyInputs[len(sp.updateCmkKeyInputs)-1].Status; got != commonEntity.KeyStatusEnabled {
		t.Fatalf("unexpected processed status: %s", got)
	}

	if err := repo.queueFailedKey(testIDQueue, testIDKey, "boom"); err != nil {
		t.Fatalf("unexpected failed error: %v", err)
	}
	if got := *sp.updateCmkKeyInputs[len(sp.updateCmkKeyInputs)-1].Status; got != commonEntity.KeyStatusUnavailable {
		t.Fatalf("unexpected failed status: %s", got)
	}
	if last := sp.updateQueueInputs[len(sp.updateQueueInputs)-1]; last.ErrorMessage == nil || *last.ErrorMessage != "boom" {
		t.Fatalf("unexpected queue error message: %#v", last)
	}

	viewRepo.queueErr = errors.New("queue read failed")
	if err := repo.updateKeyCreationQueue(testIDQueue, commonEntity.QueueStatusFailed, nil); err == nil {
		t.Fatal("expected queue read error")
	}
	viewRepo.queueErr = nil
	sp.updateQueueErr = errors.New("queue update failed")
	if err := repo.updateKeyCreationQueue(testIDQueue, commonEntity.QueueStatusFailed, nil); err == nil {
		t.Fatal("expected queue update error")
	}
	sp.updateQueueErr = nil

	viewRepo.keys = nil
	if err := repo.updateKeyStatusIfExists(testIDKey, commonEntity.KeyStatusEnabled); err != nil {
		t.Fatalf("missing key should be ignored, got %v", err)
	}
	viewRepo.keys = testViews(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled).keys
	sp.updateCmkKeyErr = errors.New("status update failed")
	if err := repo.updateKeyStatusIfExists(testIDKey, commonEntity.KeyStatusEnabled); err == nil {
		t.Fatal("expected update status error")
	}
}

func TestSaveKey(t *testing.T) {
	repo, sp, viewRepo, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)

	if err := repo.saveKey(testIDKey, commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt); err != nil {
		t.Fatalf("unexpected save existing key error: %v", err)
	}
	if len(sp.updateCmkKeyInputs) == 0 {
		t.Fatal("expected existing key status update")
	}

	viewRepo.keys = nil
	if err := repo.saveKey(testIDKey, commonEntity.KeyTypeRSAOAEP, commonEntity.KeyPurposeSign); err != nil {
		t.Fatalf("unexpected create key error: %v", err)
	}
	if sp.createCmkKeyInput == nil || sp.createCmkKeyInput.Algorithm != commonEntity.KeyTypeRSAOAEP {
		t.Fatalf("unexpected create key input: %#v", sp.createCmkKeyInput)
	}

	sp.createCmkKeyErr = errors.New("create cmk failed")
	if err := repo.saveKey(testIDKey, commonEntity.KeyTypeRSAOAEP, commonEntity.KeyPurposeSign); err == nil {
		t.Fatal("expected create cmk error")
	}
}

func TestWrappingHelpers(t *testing.T) {
	repo, sp, viewRepo, kekRepo := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)

	wrapped, alg, checksum, err := repo.wrapKey("key-material")
	if err != nil {
		t.Fatalf("unexpected wrap error: %v", err)
	}
	if wrapped != "ecc:key-material" || alg != string(commonEntity.KeyTypeECDH) || checksum == nil || *checksum != "ed-signature" {
		t.Fatalf("unexpected wrap result: %s %s %v", wrapped, alg, checksum)
	}

	key, algorithm, valid, err := repo.unwrapKey("wrapped", string(commonEntity.KeyTypeECDH), "v1", stringPtr("checksum"))
	if err != nil {
		t.Fatalf("unexpected unwrap error: %v", err)
	}
	if key != "ecc-plain:wrapped" || algorithm != string(commonEntity.KeyTypeECDH) || !valid {
		t.Fatalf("unexpected unwrap result: %s %s %v", key, algorithm, valid)
	}
	_, _, valid, err = repo.unwrapKey("wrapped", string(commonEntity.KeyTypeECDH), "v1", nil)
	if err != nil {
		t.Fatalf("unexpected nil checksum error: %v", err)
	}
	if valid {
		t.Fatal("expected invalid checksum when checksum is nil")
	}

	if err := repo.saveWrappingKeyRef(); err != nil {
		t.Fatalf("unexpected save wrapping key ref error: %v", err)
	}

	viewRepo.wrappingRefs = nil
	idWrap, err := repo.ensureWrappingKeyRef(kekRepo.kekData)
	if err != nil {
		t.Fatalf("unexpected ensure wrapping ref error: %v", err)
	}
	if *idWrap != testIDWrap || sp.createWrappingKeyRefInput == nil {
		t.Fatalf("unexpected wrapping ref creation: %s %#v", idWrap, sp.createWrappingKeyRefInput)
	}

	viewRepo.wrappingRefs = []views.CmkWrappingKeyRefView{{IDCmkWrappingKeyRef: uuid.New(), Provider: "local", KeyRef: "local", Version: "v1"}}
	idWrap, err = repo.ensureWrappingKeyRef(kekRepo.kekData)
	if err != nil {
		t.Fatalf("unexpected existing wrapping ref error: %v", err)
	}
	if *idWrap != viewRepo.wrappingRefs[0].IDCmkWrappingKeyRef {
		t.Fatalf("unexpected existing wrapping id: %s", idWrap)
	}

	repo.security = fakeSecurityRepository{err: errors.New("security failed")}
	if _, _, _, err := repo.wrapKeyWithKEK(kekRepo.kekData, "key"); err == nil {
		t.Fatal("expected wrap security error")
	}
	if _, _, _, err := repo.unwrapKey("wrapped", string(commonEntity.KeyTypeECDH), "v1", stringPtr("checksum")); err == nil {
		t.Fatal("expected unwrap security error")
	}
}

func TestRotateWrapKey(t *testing.T) {
	repo, sp, viewRepo, kekRepo := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)

	if err := repo.RotateWrapKey(testIDWrap); err != nil {
		t.Fatalf("same wrapping key should be a no-op: %v", err)
	}

	oldWrap := uuid.New()
	viewRepo.wrappingRefs = []views.CmkWrappingKeyRefView{{IDCmkWrappingKeyRef: oldWrap, Provider: "local", KeyRef: "local", Version: "v1"}}
	kekRepo.kekData.IdCmkWrappingKeyRef = testIDWrap
	kekRepo.kekData.Version = "v2"
	viewRepo.versions[0].IDCmkWrappingKeyRef = &oldWrap
	if err := repo.RotateWrapKey(oldWrap); err != nil {
		t.Fatalf("unexpected rotate wrap error: %v", err)
	}
	if len(sp.updateVersionMetadataInputs) != 1 {
		t.Fatalf("expected metadata update, got %d", len(sp.updateVersionMetadataInputs))
	}
	if got := sp.updateVersionMetadataInputs[0].IDCmkWrappingKeyRef; got == nil || *got != oldWrap {
		t.Fatalf("expected metadata to keep wrapping ref id %s, got %v", oldWrap, got)
	}
	if len(sp.updateWrappingKeyRefInputs) != 1 {
		t.Fatalf("expected wrapping ref update, got %d", len(sp.updateWrappingKeyRefInputs))
	}
	if got := sp.updateWrappingKeyRefInputs[0]; got.IDCmkWrappingKeyRef != oldWrap || got.Version == nil || *got.Version != "v2" {
		t.Fatalf("unexpected wrapping ref update: %#v", got)
	}

	viewRepo.wrappingRefs = nil
	if err := repo.RotateWrapKeyByKEK(*kekRepo.kekData); err != nil {
		t.Fatalf("not found by kek should be ignored, got %v", err)
	}
	viewRepo.wrappingRefErr = errors.New("wrap query failed")
	if err := repo.RotateWrapKeyByKEK(*kekRepo.kekData); err == nil {
		t.Fatal("expected wrapping ref query error")
	}
}

func TestRotateWrapKeyMovesToExistingTarget(t *testing.T) {
	repo, sp, viewRepo, kekRepo := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	oldWrap := uuid.New()
	targetWrap := uuid.New()
	viewRepo.wrappingRefs = []views.CmkWrappingKeyRefView{
		{IDCmkWrappingKeyRef: oldWrap, Provider: "local", KeyRef: "local", Version: "v1"},
		{IDCmkWrappingKeyRef: targetWrap, Provider: "local", KeyRef: "local", Version: "v2"},
	}
	viewRepo.versions[0].IDCmkWrappingKeyRef = &oldWrap
	kekRepo.kekData.IdCmkWrappingKeyRef = targetWrap
	kekRepo.kekData.Version = "v2"

	if err := repo.RotateWrapKey(oldWrap); err != nil {
		t.Fatalf("unexpected rotate wrap error: %v", err)
	}
	if len(sp.updateVersionMetadataInputs) != 1 {
		t.Fatalf("expected metadata update, got %d", len(sp.updateVersionMetadataInputs))
	}
	if got := sp.updateVersionMetadataInputs[0].IDCmkWrappingKeyRef; got == nil || *got != targetWrap {
		t.Fatalf("expected metadata moved to target %s, got %v", targetWrap, got)
	}
	if sp.deletedWrappingKeyRef == nil || *sp.deletedWrappingKeyRef != oldWrap {
		t.Fatalf("expected old wrapping ref delete, got %v", sp.deletedWrappingKeyRef)
	}
	if len(sp.updateWrappingKeyRefInputs) != 0 {
		t.Fatalf("did not expect mutable update when target exists: %#v", sp.updateWrappingKeyRefInputs)
	}
}

func TestDeleteWrapKeyByKEK(t *testing.T) {
	repo, sp, viewRepo, kekRepo := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	oldWrap := uuid.New()
	viewRepo.wrappingRefs = []views.CmkWrappingKeyRefView{
		{IDCmkWrappingKeyRef: oldWrap, Provider: "local", KeyRef: "local", Version: "v1"},
		{IDCmkWrappingKeyRef: testIDWrap, Provider: "local", KeyRef: "local", Version: "v2"},
	}
	kekRepo.kekData.Version = "v2"

	if err := repo.DeleteWrapKeyByKEK(models.KEK{Provider: "local", KeyRef: "local", Version: "v1"}); err != nil {
		t.Fatalf("unexpected delete wrap key error: %v", err)
	}
	if sp.deletedWrappingKeyRef == nil || *sp.deletedWrappingKeyRef != oldWrap {
		t.Fatalf("expected old wrapping ref deleted, got %v", sp.deletedWrappingKeyRef)
	}

	sp.deletedWrappingKeyRef = nil
	if err := repo.DeleteWrapKeyByKEK(*kekRepo.kekData); err != nil {
		t.Fatalf("active wrapping key should not be deleted: %v", err)
	}
	if sp.deletedWrappingKeyRef != nil {
		t.Fatalf("active wrapping key should not be deleted, got %v", sp.deletedWrappingKeyRef)
	}

	if err := repo.DeleteWrapKeyByKEK(models.KEK{Provider: "missing", KeyRef: "missing", Version: "v0"}); err != nil {
		t.Fatalf("missing wrapping key should be no-op, got %v", err)
	}

	viewRepo.providerWrapErr = errors.New("provider lookup failed")
	if err := repo.DeleteWrapKeyByKEK(models.KEK{Provider: "local", KeyRef: "local", Version: "v1"}); err == nil {
		t.Fatal("expected lookup error")
	}
	viewRepo.providerWrapErr = nil

	kekRepo.getErr = errors.New("current kek failed")
	if err := repo.DeleteWrapKeyByKEK(models.KEK{Provider: "local", KeyRef: "local", Version: "v1"}); err == nil {
		t.Fatal("expected current kek error")
	}
	kekRepo.getErr = nil

	sp.deleteWrappingKeyRefErr = errors.New("delete failed")
	if err := repo.DeleteWrapKeyByKEK(models.KEK{Provider: "local", KeyRef: "local", Version: "v1"}); err == nil {
		t.Fatal("expected delete wrapping ref error")
	}
}

func TestGettersAndLoadKeyMaterial(t *testing.T) {
	repo, _, viewRepo, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)

	key, err := repo.getKey(testIDKey)
	if err != nil {
		t.Fatalf("unexpected key error: %v", err)
	}
	if key.IDCmkKey != testIDKey {
		t.Fatalf("unexpected key: %#v", key)
	}
	queue, err := repo.getQueueData(&testIDKey, nil)
	if err != nil {
		t.Fatalf("unexpected queue error: %v", err)
	}
	if queue.IDCmkKeyCreationQueue != testIDQueue {
		t.Fatalf("unexpected queue: %#v", queue)
	}
	version, err := repo.getKeyVersion(&testIDVersion)
	if err != nil {
		t.Fatalf("unexpected version error: %v", err)
	}
	if version.IDCmkKeyVersion != testIDVersion {
		t.Fatalf("unexpected version: %#v", version)
	}
	wrap, err := repo.getWrappingKeyRef(testIDWrap)
	if err != nil {
		t.Fatalf("unexpected wrap error: %v", err)
	}
	if wrap.IDCmkWrappingKeyRef != testIDWrap {
		t.Fatalf("unexpected wrap: %#v", wrap)
	}
	material, err := repo.loadKeyMaterial(testSecret)
	if err != nil {
		t.Fatalf("unexpected material error: %v", err)
	}
	if material.key != "ecc-plain:wrapped-key" || material.algorithm != string(commonEntity.KeySymmetricDefault) {
		t.Fatalf("unexpected material: %#v", material)
	}

	viewRepo.queues[0].Status = commonEntity.QueueStatusPending
	if _, err := repo.getKey(testIDKey); err == nil || err.Error() != errMsgKeyNotProcessed {
		t.Fatalf("expected key not processed, got %v", err)
	}
	viewRepo.queues[0].Status = commonEntity.QueueStatusProcessed
	viewRepo.keys = nil
	if _, err := repo.getKey(testIDKey); err == nil || err.Error() != errMsgKeyNotFound {
		t.Fatalf("expected key not found, got %v", err)
	}
	viewRepo.keys = testViews(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled).keys
	viewRepo.queues = nil
	if _, err := repo.getQueueData(&testIDKey, nil); err == nil {
		t.Fatal("expected queue not found")
	}
	viewRepo.queues = testViews(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled).queues
	viewRepo.versions = nil
	if _, err := repo.getKeyVersion(&testIDVersion); err == nil || err.Error() != errMsgKeyVersionNotFound {
		t.Fatalf("expected version not found, got %v", err)
	}
	viewRepo.versionErr = errors.New("version query failed")
	if _, err := repo.getKeyVersion(&testIDVersion); err == nil {
		t.Fatal("expected version query error")
	}
	viewRepo.versionErr = nil
	viewRepo.versions = testViews(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled).versions
	viewRepo.wrappingRefs = nil
	if _, err := repo.getWrappingKeyRef(testIDWrap); err == nil || err.Error() != errMsgWrappingKeyRefNotFound {
		t.Fatalf("expected wrap not found, got %v", err)
	}
}

func TestStatusAndHealthz(t *testing.T) {
	for status, want := range map[commonEntity.KeyStatus]string{
		commonEntity.KeyStatusPendingImport:   "PENDING_IMPORT",
		commonEntity.KeyStatusEnabled:         "ok",
		commonEntity.KeyStatusDisabled:        "DISABLED",
		commonEntity.KeyStatusPendingDeletion: "PENDING_DELETION",
		commonEntity.KeyStatusUnavailable:     "UNAVAILABLE",
		commonEntity.KeyStatus("other"):       "UNKNOWN",
	} {
		if got := getHealthz(status); got != want {
			t.Fatalf("getHealthz(%s) = %s, want %s", status, got, want)
		}
	}

	repo, _, _, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	status, err := repo.Status()
	if err != nil {
		t.Fatalf("unexpected status error: %v", err)
	}
	if status.ID != testIDKey || status.Healthz != "ok" || status.Version != "v1" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestKeyVersionInfoAndStatusUpdate(t *testing.T) {
	repo, sp, viewRepo, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	secondary := viewRepo.versions[0]
	secondary.IDCmkKeyVersion = testIDVersionSecondary
	secondary.VersionNumber = 2
	secondary.Status = commonEntity.KeyVersionStatusDisabled
	secondary.KID = "secondary-public-key"
	viewRepo.versions = append(viewRepo.versions, secondary)

	info, err := repo.GetKeyVersionInfo(testIDVersionSecondary)
	if err != nil {
		t.Fatalf("unexpected get version info error: %v", err)
	}
	if info.IDCmkKey != testIDKey || info.IDCmkKeyVersion != testIDVersionSecondary || info.PublicKey != "secondary-public-key" || info.IsCurrent {
		t.Fatalf("unexpected key version info: %#v", info)
	}
	if idKey, idVersion := assertEncodedSecret(t, info.SecretCmkKey); idKey != testIDKey || idVersion != testIDVersionSecondary {
		t.Fatalf("unexpected encoded secret ids: %s %s", idKey, idVersion)
	}

	updated, err := repo.UpdateKeyVersionStatus(testIDVersionSecondary, commonEntity.KeyVersionStatusUnavailable)
	if err != nil {
		t.Fatalf("unexpected update version status error: %v", err)
	}
	if updated.Status != commonEntity.KeyVersionStatusUnavailable {
		t.Fatalf("unexpected updated status: %#v", updated)
	}
	if len(sp.updateVersionStatusInputs) != 1 ||
		sp.updateVersionStatusInputs[0].IDCmkKeyVersion != testIDVersionSecondary ||
		sp.updateVersionStatusInputs[0].Status != commonEntity.KeyVersionStatusUnavailable {
		t.Fatalf("unexpected update status input: %#v", sp.updateVersionStatusInputs)
	}

	if _, err := repo.UpdateKeyVersionStatus(testIDVersionSecondary, commonEntity.KeyVersionStatus("pendingDeletion")); err == nil || err.Error() != errMsgKeyVersionStatusPendingDeletion {
		t.Fatalf("expected pendingDeletion validation error, got %v", err)
	}
	if _, err := repo.UpdateKeyVersionStatus(testIDVersionSecondary, commonEntity.KeyVersionStatus("bad")); err == nil || err.Error() != errMsgKeyVersionStatusInvalid {
		t.Fatalf("expected invalid status error, got %v", err)
	}
	if _, err := repo.UpdateKeyVersionStatus(testIDVersion, commonEntity.KeyVersionStatusDisabled); err == nil || err.Error() != errMsgKeyVersionMainCannotBeUpdated {
		t.Fatalf("expected main version update error, got %v", err)
	}
}

func TestRequirePurposeAndJWTConfig(t *testing.T) {
	if err := requirePurpose(commonEntity.KeyPurposeEncrypt, commonEntity.KeyPurposeEncrypt, "bad purpose"); err != nil {
		t.Fatalf("unexpected purpose error: %v", err)
	}
	if err := requirePurpose(commonEntity.KeyPurposeSign, commonEntity.KeyPurposeEncrypt, "bad purpose"); err == nil || err.Error() != "bad purpose" {
		t.Fatalf("expected bad purpose error, got %v", err)
	}
	config := newJWTConfig("HS256", "private", "public")
	if config.Algorithm != "hs256" || *config.HMACSecretKey != "private" || *config.RSAPublicKeyKey != "public" || config.Validator == nil {
		t.Fatalf("unexpected jwt config: %#v", config)
	}
}

func TestJWTExpirationValidator(t *testing.T) {
	if err := validateJWTExpirationClaim(context.Background(), jwtservice.Token{Claims: []byte(`{"sub":"1"}`)}); err != nil {
		t.Fatalf("unexpected missing exp error: %v", err)
	}

	future := time.Now().Add(time.Hour).Unix()
	if err := validateJWTExpirationClaim(context.Background(), jwtservice.Token{Claims: []byte(`{"exp":` + strconv.FormatInt(future, 10) + `}`)}); err != nil {
		t.Fatalf("unexpected future exp error: %v", err)
	}

	expired := time.Now().Add(-time.Hour).Unix()
	if err := validateJWTExpirationClaim(context.Background(), jwtservice.Token{Claims: []byte(`{"exp":` + strconv.FormatInt(expired, 10) + `}`)}); err == nil || err.Error() != "jwt has expired" {
		t.Fatalf("expected expired jwt error, got %v", err)
	}

	if err := validateJWTExpirationClaim(context.Background(), jwtservice.Token{Claims: []byte(`{"exp":"bad"}`)}); err == nil || err.Error() != "invalid jwt expiration claim" {
		t.Fatalf("expected invalid exp error, got %v", err)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	for _, tt := range []struct {
		name      string
		algorithm commonEntity.KeyType
		wantEnc   string
		wantDec   string
	}{
		{"aes", commonEntity.KeySymmetricDefault, "aes:plain", "plain:cipher"},
		{"rsa", commonEntity.KeyTypeRSAOAEP, "rsa:plain", "rsa-plain:cipher"},
		{"ecdh", commonEntity.KeyTypeECDH, "ecc:plain", "ecc-plain:cipher"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, _, _ := newTestRepository(tt.algorithm, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
			ciphertext, annotations, err := repo.Encrypt(testSecret, "plain", nil)
			if err != nil {
				t.Fatalf("unexpected encrypt error: %v", err)
			}
			if ciphertext != tt.wantEnc || string(annotations["PublicKey"]) != "public-key" || string(annotations["Size"]) != "256" {
				t.Fatalf("unexpected encrypt result: %s %#v", ciphertext, annotations)
			}
			plaintext, _, err := repo.Decrypt(testSecret, "cipher", nil)
			if err != nil {
				t.Fatalf("unexpected decrypt error: %v", err)
			}
			if plaintext != tt.wantDec {
				t.Fatalf("unexpected plaintext: %s", plaintext)
			}
		})
	}

	repo, _, _, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeSign, commonEntity.KeyStatusEnabled)
	if _, _, err := repo.Encrypt(testSecret, "plain", nil); err == nil || err.Error() != errMsgKeyPurposeNotAllowEncryption {
		t.Fatalf("expected encryption purpose error, got %v", err)
	}
	if _, _, err := repo.Decrypt(testSecret, "cipher", nil); err == nil || err.Error() != errMsgKeyPurposeNotAllowDecryption {
		t.Fatalf("expected decryption purpose error, got %v", err)
	}
	repo, _, _, _ = newTestRepository(commonEntity.KeyType("bad"), commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	if _, _, err := repo.Encrypt(testSecret, "plain", nil); err == nil || err.Error() != errMsgUnsupportedAlgorithm {
		t.Fatalf("expected unsupported encrypt algorithm, got %v", err)
	}
	if _, _, err := repo.Decrypt(testSecret, "cipher", nil); err == nil || err.Error() != errMsgUnsupportedAlgorithm {
		t.Fatalf("expected unsupported decrypt algorithm, got %v", err)
	}
}

func TestJWTMethods(t *testing.T) {
	originalNewJWTServiceFn := newJWTServiceFn
	t.Cleanup(func() {
		newJWTServiceFn = originalNewJWTServiceFn
	})

	repo, _, _, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeSign, commonEntity.KeyStatusEnabled)
	newJWTServiceFn = func(input jwtservice.ConfigServiceInput) (jwtTokenService, error) {
		return fakeJWTService{}, nil
	}
	token, err := repo.CreateJWT(context.Background(), testSecret, "HS256", map[string]any{"sub": "1"})
	if err != nil || token != "jwt-token" {
		t.Fatalf("unexpected create jwt result: %s %v", token, err)
	}
	if err := repo.VerifyJWT(context.Background(), testSecret, "HS256", token); err != nil {
		t.Fatalf("unexpected verify jwt error: %v", err)
	}
	if err := repo.ReadJWT(context.Background(), testSecret, "HS256", token, map[string]any{}); err != nil {
		t.Fatalf("unexpected read jwt error: %v", err)
	}

	newJWTServiceFn = func(input jwtservice.ConfigServiceInput) (jwtTokenService, error) {
		return nil, errors.New("jwt factory failed")
	}
	if _, err := repo.CreateJWT(context.Background(), testSecret, "HS256", nil); err == nil {
		t.Fatal("expected create jwt factory error")
	}
	if err := repo.VerifyJWT(context.Background(), testSecret, "HS256", token); err == nil {
		t.Fatal("expected verify jwt factory error")
	}
	if err := repo.ReadJWT(context.Background(), testSecret, "HS256", token, map[string]any{}); err == nil {
		t.Fatal("expected read jwt factory error")
	}

	repo, _, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	if _, err := repo.CreateJWT(context.Background(), testSecret, "HS256", nil); err == nil || err.Error() != errMsgKeyPurposeNotAllowJWTSigning {
		t.Fatalf("expected jwt purpose error, got %v", err)
	}
}

func TestJWTMethodsValidateExpirationClaim(t *testing.T) {
	repo, _, _, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeSign, commonEntity.KeyStatusEnabled)

	expiredClaims := map[string]any{"sub": "1", "exp": time.Now().Add(-time.Hour).Unix()}
	if _, err := repo.CreateJWT(context.Background(), testSecret, "HS256", expiredClaims); err == nil || err.Error() != "jwt has expired" {
		t.Fatalf("expected create jwt expired error, got %v", err)
	}

	config := newJWTConfig("HS256", "ecc-plain:wrapped-key", "public-key")
	config.Validator = nil
	service, err := newJWTService(config)
	if err != nil {
		t.Fatalf("unexpected jwt service error: %v", err)
	}
	expiredToken, err := service.CreateWithContext(context.Background(), expiredClaims)
	if err != nil {
		t.Fatalf("unexpected expired token setup error: %v", err)
	}

	if err := repo.VerifyJWT(context.Background(), testSecret, "HS256", expiredToken); err == nil || err.Error() != "jwt has expired" {
		t.Fatalf("expected verify jwt expired error, got %v", err)
	}
}

func TestSignAndVerify(t *testing.T) {
	for _, tt := range []struct {
		name      string
		algorithm commonEntity.KeyType
		signature string
	}{
		{"rsa-pss", commonEntity.KeyTypeRSAOAEP, "pss-signature"},
		{"rsa-pkcs", commonEntity.KeyTypeRSAPKCS1v15SHA256, "pkcs-signature"},
		{"eddsa", commonEntity.KeyTypeEdDSA, "ed-signature"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, _, _ := newTestRepository(tt.algorithm, commonEntity.KeyPurposeSign, commonEntity.KeyStatusEnabled)
			signature, err := repo.Sing(testSecret, "message")
			if err != nil {
				t.Fatalf("unexpected sign error: %v", err)
			}
			if signature != tt.signature {
				t.Fatalf("unexpected signature: %s", signature)
			}
			valid, err := repo.Verify(testSecret, "message", signature)
			if err != nil || !valid {
				t.Fatalf("unexpected verify result: %v %v", valid, err)
			}
		})
	}

	repo, _, _, _ := newTestRepository(commonEntity.KeyTypeEdDSA, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	if _, err := repo.Sing(testSecret, "message"); err == nil || err.Error() != errMsgKeyPurposeNotAllowSigning {
		t.Fatalf("expected sign purpose error, got %v", err)
	}
	if _, err := repo.Verify(testSecret, "message", "signature"); err == nil || err.Error() != errMsgKeyPurposeNotAllowSigning {
		t.Fatalf("expected verify purpose error, got %v", err)
	}

	repo, _, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeSign, commonEntity.KeyStatusEnabled)
	if _, err := repo.Sing(testSecret, "message"); err == nil || err.Error() != errMsgUnsupportedAlgorithm {
		t.Fatalf("expected unsupported sign algorithm, got %v", err)
	}
	if _, err := repo.Verify(testSecret, "message", "signature"); err == nil || err.Error() != errMsgUnsupportedAlgorithm {
		t.Fatalf("expected unsupported verify algorithm, got %v", err)
	}
}

func TestTransitionsAndDelete(t *testing.T) {
	repo, sp, _, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusDisabled)
	if err := repo.EnableKey(testSecret); err != nil {
		t.Fatalf("unexpected enable error: %v", err)
	}
	if got := *sp.updateCmkKeyInputs[len(sp.updateCmkKeyInputs)-1].Status; got != commonEntity.KeyStatusEnabled {
		t.Fatalf("unexpected enable status: %s", got)
	}

	repo, sp, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	if err := repo.DisableKey(testSecret); err != nil {
		t.Fatalf("unexpected disable error: %v", err)
	}
	if got := *sp.updateCmkKeyInputs[len(sp.updateCmkKeyInputs)-1].Status; got != commonEntity.KeyStatusDisabled {
		t.Fatalf("unexpected disable status: %s", got)
	}
	if err := repo.EnableKey(testSecret); err == nil || err.Error() != errMsgKeyMustBeDisabledToEnable {
		t.Fatalf("expected enable transition error, got %v", err)
	}

	repo, sp, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusDisabled)
	if err := repo.PendingDeletion(testSecret); err != nil {
		t.Fatalf("unexpected pending deletion error: %v", err)
	}
	if sp.retiredVersion == nil || *sp.retiredVersion != testIDVersion {
		t.Fatalf("expected retired version %s, got %v", testIDVersion, sp.retiredVersion)
	}
	if got := *sp.updateCmkKeyInputs[len(sp.updateCmkKeyInputs)-1].Status; got != commonEntity.KeyStatusPendingDeletion {
		t.Fatalf("unexpected pending deletion status: %s", got)
	}

	repo, sp, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusPendingDeletion)
	if err := repo.CancelKeyDeletion(testSecret); err != nil {
		t.Fatalf("unexpected cancel deletion error: %v", err)
	}
	if got := *sp.updateCmkKeyInputs[len(sp.updateCmkKeyInputs)-1].Status; got != commonEntity.KeyStatusDisabled {
		t.Fatalf("unexpected cancel deletion status: %s", got)
	}

	repo, sp, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusPendingDeletion)
	if err := repo.UnavailableDelete(testSecret); err != nil {
		t.Fatalf("unexpected unavailable delete error: %v", err)
	}
	if got := *sp.updateCmkKeyInputs[len(sp.updateCmkKeyInputs)-1].Status; got != commonEntity.KeyStatusUnavailable {
		t.Fatalf("unexpected unavailable delete status: %s", got)
	}

	repo, sp, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	if err := repo.DeleteKey(testSecret); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if sp.deletedRetiredVersion == nil || *sp.deletedRetiredVersion != testIDVersion || sp.deletedQueue == nil || *sp.deletedQueue != testIDQueue || sp.deletedCmkKey == nil || *sp.deletedCmkKey != testIDKey {
		t.Fatalf("unexpected delete calls: version=%v queue=%v key=%v", sp.deletedRetiredVersion, sp.deletedQueue, sp.deletedCmkKey)
	}

	if err := repo.DisableKey("bad-secret"); err == nil {
		t.Fatal("expected invalid secret error")
	}
}

func TestScheduleKeyDeletion(t *testing.T) {
	serverGin.SetModeTest()
	jobs.StopAllJobs(true)
	t.Cleanup(func() {
		jobs.StopAllJobs(true)
	})

	repo, sp, viewRepo, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusDisabled)
	if err := repo.ScheduleKeyDeletion(testSecret, 7); err != nil {
		t.Fatalf("unexpected schedule deletion error: %v", err)
	}
	if sp.retiredVersion == nil || *sp.retiredVersion != testIDVersion {
		t.Fatalf("expected retired version %s, got %v", testIDVersion, sp.retiredVersion)
	}
	if got := *sp.updateCmkKeyInputs[len(sp.updateCmkKeyInputs)-1].Status; got != commonEntity.KeyStatusPendingDeletion {
		t.Fatalf("unexpected pending deletion status: %s", got)
	}
	if sp.deletedRetiredVersion != nil || sp.deletedQueue != nil || sp.deletedCmkKey != nil {
		t.Fatalf("expected no immediate delete calls: version=%v queue=%v key=%v", sp.deletedRetiredVersion, sp.deletedQueue, sp.deletedCmkKey)
	}

	viewRepo.keys[0].Status = commonEntity.KeyStatusPendingDeletion
	jobs.StartJobs()
	time.Sleep(20 * time.Millisecond)
	if sp.deletedRetiredVersion != nil || sp.deletedQueue != nil || sp.deletedCmkKey != nil {
		t.Fatalf("expected scheduled job to be disabled in mode test: version=%v queue=%v key=%v", sp.deletedRetiredVersion, sp.deletedQueue, sp.deletedCmkKey)
	}

	repo.deleteScheduledKey(testSecret)
	if sp.deletedRetiredVersion == nil || *sp.deletedRetiredVersion != testIDVersion || sp.deletedQueue == nil || *sp.deletedQueue != testIDQueue || sp.deletedCmkKey == nil || *sp.deletedCmkKey != testIDKey {
		t.Fatalf("unexpected scheduled delete calls: version=%v queue=%v key=%v", sp.deletedRetiredVersion, sp.deletedQueue, sp.deletedCmkKey)
	}

	repo, sp, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusDisabled)
	repo.deleteScheduledKey(testSecret)
	if sp.deletedRetiredVersion != nil || sp.deletedQueue != nil || sp.deletedCmkKey != nil {
		t.Fatalf("expected canceled scheduled deletion to be skipped: version=%v queue=%v key=%v", sp.deletedRetiredVersion, sp.deletedQueue, sp.deletedCmkKey)
	}

	repo, sp, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusDisabled)
	if err := repo.ScheduleKeyDeletion(testSecret, 0); err == nil {
		t.Fatal("expected invalid schedule interval error")
	}
	if sp.retiredVersion != nil || len(sp.updateCmkKeyInputs) != 0 {
		t.Fatalf("expected invalid interval to skip pending deletion: retired=%v updates=%d", sp.retiredVersion, len(sp.updateCmkKeyInputs))
	}
}

func TestCreateAndRotateKey(t *testing.T) {
	repo, sp, viewRepo, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	viewRepo.matchAnyQueue = true

	secret, err := repo.CreateKey(models.CreateKeyInput{
		IDCmkKey:  &testIDKey,
		Algorithm: commonEntity.KeySymmetricDefault,
		Size:      256,
		Purpose:   commonEntity.KeyPurposeEncrypt,
	}, "")
	if err != nil {
		t.Fatalf("unexpected create key error: %v", err)
	}
	if secret == "" || sp.createVersionInput == nil {
		t.Fatalf("expected secret and version creation, got secret=%q input=%#v", secret, sp.createVersionInput)
	}
	_, createdVersion := assertEncodedSecret(t, secret)
	if createdVersion != sp.createVersionInput.IDCmkKeyVersion {
		t.Fatalf("expected encoded create version %s, got %s", sp.createVersionInput.IDCmkKeyVersion, createdVersion)
	}

	rotated, err := repo.RotateKey(testSecret)
	if err != nil {
		t.Fatalf("unexpected rotate key error: %v", err)
	}
	if rotated == "" || sp.rotateVersionInput == nil || sp.rotateVersionInput.IDCmkKey != testIDKey {
		t.Fatalf("unexpected rotate result: %s %#v", rotated, sp.rotateVersionInput)
	}
	_, rotatedVersion := assertEncodedSecret(t, rotated)
	if rotatedVersion != sp.rotateVersionInput.IDCmkKeyVersion {
		t.Fatalf("expected encoded rotate version %s, got %s", sp.rotateVersionInput.IDCmkKeyVersion, rotatedVersion)
	}

	sp.rotateVersionErr = errors.New("rotate version failed")
	if _, err := repo.RotateKey(testSecret); err == nil {
		t.Fatal("expected rotate version error")
	}
}

func TestCreateKeyErrors(t *testing.T) {
	repo, sp, viewRepo, kekRepo := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	sp.createQueueErr = errors.New("queue failed")
	if _, err := repo.CreateKey(models.CreateKeyInput{IDCmkKey: &testIDKey, Algorithm: commonEntity.KeySymmetricDefault, Size: 256, Purpose: commonEntity.KeyPurposeEncrypt}, ""); err == nil {
		t.Fatal("expected queue error")
	}

	sp.createQueueErr = nil
	repo.security = fakeSecurityRepository{err: errors.New("generate failed")}
	if _, err := repo.CreateKey(models.CreateKeyInput{IDCmkKey: &testIDKey, Algorithm: commonEntity.KeySymmetricDefault, Size: 256, Purpose: commonEntity.KeyPurposeEncrypt}, ""); err == nil {
		t.Fatal("expected generate error")
	}

	repo.security = fakeSecurityRepository{}
	kekRepo.getErr = errors.New("kek failed")
	if _, err := repo.CreateKey(models.CreateKeyInput{IDCmkKey: &testIDKey, Algorithm: commonEntity.KeySymmetricDefault, Size: 256, Purpose: commonEntity.KeyPurposeEncrypt}, ""); err == nil {
		t.Fatal("expected kek error")
	}
	kekRepo.getErr = nil
	viewRepo.queues = nil
	if _, err := repo.CreateKey(models.CreateKeyInput{IDCmkKey: &testIDKey, Algorithm: commonEntity.KeySymmetricDefault, Size: 256, Purpose: commonEntity.KeyPurposeEncrypt}, ""); err == nil {
		t.Fatal("expected queue lookup error")
	}
}

func TestListCmkKey(t *testing.T) {
	repo, _, viewRepo, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	results, err := repo.ListCmkKey(testIDWrap, 1, 10)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(results.Results) != 1 || !reflect.DeepEqual(results.Results[0].CmkKey, viewRepo.keys[0]) || !reflect.DeepEqual(results.Results[0].Queue, viewRepo.queues[0]) {
		t.Fatalf("unexpected list result: %#v", results)
	}

	viewRepo.keyErr = errors.New("key query failed")
	if _, err := repo.ListCmkKey(testIDWrap, 1, 10); err == nil {
		t.Fatal("expected key query error")
	}
	viewRepo.keyErr = nil
	viewRepo.queueErr = errors.New("queue query failed")
	if _, err := repo.ListCmkKey(testIDWrap, 1, 10); err == nil {
		t.Fatal("expected queue query error")
	}
	viewRepo.queueErr = nil
	viewRepo.queues = nil
	if _, err := repo.ListCmkKey(testIDWrap, 1, 10); err == nil {
		t.Fatal("expected missing queue relation error")
	}
}

func TestListCreationKeyQueues(t *testing.T) {
	repo, _, viewRepo, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	secondKey := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	viewRepo.queues = append(viewRepo.queues, views.CmkCreationKeyQueueView{
		IDCmkKeyCreationQueue: uuid.MustParse("66666666-6666-6666-6666-666666666666"),
		IDCmkKey:              secondKey,
		EventType:             commonEntity.EventTypeRotateKey,
		Status:                commonEntity.QueueStatusPending,
	})

	results, err := repo.ListCreationKeyQueues(uuid.Nil, "", 1, 1)
	if err != nil {
		t.Fatalf("unexpected list queue error: %v", err)
	}
	if len(results.Results) != 1 || results.Pagination.TotalRegisters != 2 || results.Pagination.TotalPages != 2 {
		t.Fatalf("unexpected paginated queue result: %#v", results)
	}
	if len(viewRepo.queueQueries) != 2 {
		t.Fatalf("expected count and list queries, got %#v", viewRepo.queueQueries)
	}
	if !strings.Contains(viewRepo.queueQueries[1], "ORDER BY queued_at DESC") {
		t.Fatalf("expected stable queue order query, got %q", viewRepo.queueQueries[1])
	}

	results, err = repo.ListCreationKeyQueues(secondKey, commonEntity.QueueStatusPending, 1, 10)
	if err != nil {
		t.Fatalf("unexpected filtered list queue error: %v", err)
	}
	if len(results.Results) != 1 || results.Results[0].IDCmkKey != secondKey {
		t.Fatalf("unexpected filtered queue result: %#v", results)
	}

	if _, err := repo.ListCreationKeyQueues(uuid.Nil, "", 0, 10); err == nil {
		t.Fatal("expected pagination validation error")
	}
	viewRepo.queueErr = errors.New("queue query failed")
	if _, err := repo.ListCreationKeyQueues(uuid.Nil, "", 1, 10); err == nil {
		t.Fatal("expected queue count error")
	}
}

func TestStatusErrors(t *testing.T) {
	repo, _, viewRepo, kekRepo := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	kekRepo.getErr = errors.New("kek failed")
	if _, err := repo.Status(); err == nil {
		t.Fatal("expected kek error")
	}

	kekRepo.getErr = nil
	kekRepo.kekData.SecretCmkKey = "bad-secret"
	if _, err := repo.Status(); err == nil {
		t.Fatal("expected invalid secret error")
	}

	kekRepo.kekData.SecretCmkKey = testSecret
	viewRepo.keys = nil
	if _, err := repo.Status(); err == nil {
		t.Fatal("expected key error")
	}

	viewRepo.keys = testViews(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled).keys
	viewRepo.versions = nil
	if _, err := repo.Status(); err == nil {
		t.Fatal("expected version error")
	}
}

func TestLoadKeyMaterialErrors(t *testing.T) {
	repo, _, viewRepo, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)

	if _, err := repo.loadKeyMaterial("bad-secret"); err == nil {
		t.Fatal("expected invalid secret error")
	}

	viewRepo.keys = nil
	if _, err := repo.loadKeyMaterial(testSecret); err == nil {
		t.Fatal("expected key error")
	}

	viewRepo.keys = testViews(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled).keys
	viewRepo.versions = nil
	if _, err := repo.loadKeyMaterial(testSecret); err == nil {
		t.Fatal("expected version error")
	}

	viewRepo.versions = testViews(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled).versions
	viewRepo.versions[0].Status = commonEntity.KeyVersionStatusDisabled
	if _, err := repo.loadKeyMaterial(testSecret); err == nil || err.Error() != errMsgKeyVersionMustBeEnabled {
		t.Fatalf("expected key version status error, got %v", err)
	}

	viewRepo.versions = testViews(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled).versions
	viewRepo.wrappingRefs = nil
	if _, err := repo.loadKeyMaterial(testSecret); err == nil {
		t.Fatal("expected wrapping ref error")
	}

	viewRepo.wrappingRefs = testViews(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled).wrappingRefs
	repo.security = fakeSecurityRepository{decryptErr: errors.New("unwrap failed")}
	if _, err := repo.loadKeyMaterial(testSecret); err == nil {
		t.Fatal("expected unwrap error")
	}

	repo.security = fakeSecurityRepository{verifyErr: errors.New("checksum failed")}
	if _, err := repo.loadKeyMaterial(testSecret); err == nil || err.Error() != errMsgInvalidChecksum {
		t.Fatalf("expected invalid checksum, got %v", err)
	}
}

func TestWrappingErrorBranches(t *testing.T) {
	repo, sp, viewRepo, kekRepo := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)

	kekRepo.getErr = errors.New("kek failed")
	if _, _, _, err := repo.wrapKey("key"); err == nil {
		t.Fatal("expected wrap key kek error")
	}
	if err := repo.saveWrappingKeyRef(); err == nil {
		t.Fatal("expected save wrapping key ref kek error")
	}

	kekRepo.getErr = nil
	viewRepo.wrappingRefErr = errors.New("wrapping read failed")
	if _, err := repo.ensureWrappingKeyRef(kekRepo.kekData); err == nil {
		t.Fatal("expected ensure wrapping read error")
	}
	if _, err := repo.getWrappingKeyRefByKEK(kekRepo.kekData); err == nil {
		t.Fatal("expected get by kek read error")
	}

	viewRepo.wrappingRefErr = nil
	viewRepo.wrappingRefs = nil
	sp.createWrappingKeyRefErr = errors.New("create wrap failed")
	if _, err := repo.ensureWrappingKeyRef(kekRepo.kekData); err == nil {
		t.Fatal("expected create wrapping ref error")
	}

	sp.createWrappingKeyRefErr = nil
	viewRepo.wrappingRefs = []views.CmkWrappingKeyRefView{{IDCmkWrappingKeyRef: uuid.New(), Provider: "local", KeyRef: "local", Version: "v1"}}
	got, err := repo.getWrappingKeyRefByKEK(kekRepo.kekData)
	if err != nil {
		t.Fatalf("unexpected get by kek fallback error: %v", err)
	}
	if got.IDCmkWrappingKeyRef != viewRepo.wrappingRefs[0].IDCmkWrappingKeyRef {
		t.Fatalf("unexpected fallback wrapping ref: %#v", got)
	}
}

func TestRotateWrapKeyErrorBranches(t *testing.T) {
	repo, sp, viewRepo, kekRepo := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)

	viewRepo.wrappingRefs = nil
	if err := repo.RotateWrapKey(testIDWrap); err == nil {
		t.Fatal("expected rotate wrap missing ref error")
	}

	oldWrap := uuid.New()
	viewRepo.wrappingRefs = []views.CmkWrappingKeyRefView{{IDCmkWrappingKeyRef: oldWrap, Provider: "local", KeyRef: "local", Version: "v1"}}
	viewRepo.versions[0].IDCmkWrappingKeyRef = &oldWrap
	kekRepo.kekData.IdCmkWrappingKeyRef = testIDWrap
	kekRepo.kekData.Version = "v2"
	kekRepo.getErr = errors.New("kek failed")
	if err := repo.RotateWrapKey(oldWrap); err == nil {
		t.Fatal("expected rotate wrap kek error")
	}

	kekRepo.getErr = nil
	viewRepo.versionErr = errors.New("versions failed")
	if err := repo.RotateWrapKey(oldWrap); err == nil {
		t.Fatal("expected versions query error")
	}

	viewRepo.versionErr = nil
	repo.security = fakeSecurityRepository{verifyErr: errors.New("checksum failed")}
	if err := repo.RotateWrapKey(oldWrap); err == nil || err.Error() != errMsgInvalidChecksum {
		t.Fatalf("expected invalid checksum, got %v", err)
	}

	repo.security = fakeSecurityRepository{encryptErr: errors.New("wrap failed")}
	if err := repo.RotateWrapKey(oldWrap); err == nil {
		t.Fatal("expected wrap error")
	}

	repo.security = fakeSecurityRepository{}
	sp.updateVersionMetadataErr = errors.New("metadata failed")
	if err := repo.RotateWrapKey(oldWrap); err == nil {
		t.Fatal("expected metadata update error")
	}
}

func TestJWTMethodErrors(t *testing.T) {
	originalNewJWTServiceFn := newJWTServiceFn
	t.Cleanup(func() {
		newJWTServiceFn = originalNewJWTServiceFn
	})
	repo, _, _, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeSign, commonEntity.KeyStatusEnabled)

	newJWTServiceFn = func(input jwtservice.ConfigServiceInput) (jwtTokenService, error) {
		return fakeJWTService{createErr: errors.New("create failed")}, nil
	}
	if _, err := repo.CreateJWT(context.Background(), testSecret, "HS256", nil); err == nil {
		t.Fatal("expected create jwt service error")
	}

	newJWTServiceFn = func(input jwtservice.ConfigServiceInput) (jwtTokenService, error) {
		return fakeJWTService{readErr: errors.New("verify failed")}, nil
	}
	if err := repo.VerifyJWT(context.Background(), testSecret, "HS256", "token"); err == nil {
		t.Fatal("expected verify jwt service error")
	}

	newJWTServiceFn = func(input jwtservice.ConfigServiceInput) (jwtTokenService, error) {
		return fakeJWTService{readErr: errors.New("read failed")}, nil
	}
	if err := repo.ReadJWT(context.Background(), testSecret, "HS256", "token", map[string]any{}); err == nil {
		t.Fatal("expected read jwt service error")
	}
}

func TestTransitionPendingAndDeleteErrors(t *testing.T) {
	repo, sp, _, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	if err := repo.PendingDeletion(testSecret); err == nil || err.Error() != errMsgKeyMustBeDisabledToMarkDeletion {
		t.Fatalf("expected pending deletion transition error, got %v", err)
	}

	repo, sp, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusDisabled)
	sp.retireVersionErr = errors.New("retire failed")
	if err := repo.PendingDeletion(testSecret); err == nil {
		t.Fatal("expected retire error")
	}

	repo, sp, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	sp.deleteRetiredVersionErr = errors.New("delete retired failed")
	if err := repo.DeleteKey(testSecret); err == nil {
		t.Fatal("expected delete retired error")
	}

	sp.deleteRetiredVersionErr = nil
	sp.deleteQueueErr = errors.New("delete queue failed")
	if err := repo.DeleteKey(testSecret); err == nil {
		t.Fatal("expected delete queue error")
	}

	sp.deleteQueueErr = nil
	sp.deleteCmkKeyErr = errors.New("delete cmk failed")
	if err := repo.DeleteKey(testSecret); err == nil {
		t.Fatal("expected delete cmk error")
	}
}

func TestCreateKeyAdditionalErrors(t *testing.T) {
	repo, sp, viewRepo, _ := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	viewRepo.matchAnyQueue = true

	sp.updateCmkKeyErr = errors.New("save key failed")
	if _, err := repo.CreateKey(models.CreateKeyInput{IDCmkKey: &testIDKey, Algorithm: commonEntity.KeySymmetricDefault, Size: 256, Purpose: commonEntity.KeyPurposeEncrypt}, ""); err == nil {
		t.Fatal("expected save key error")
	}

	sp.updateCmkKeyErr = nil
	sp.updateQueueErr = errors.New("processing failed")
	if _, err := repo.CreateKey(models.CreateKeyInput{IDCmkKey: &testIDKey, Algorithm: commonEntity.KeySymmetricDefault, Size: 256, Purpose: commonEntity.KeyPurposeEncrypt}, ""); err == nil {
		t.Fatal("expected queue processing error")
	}

	sp.updateQueueErr = nil
	repo.security = fakeSecurityRepository{encryptErr: errors.New("wrap failed")}
	if _, err := repo.CreateKey(models.CreateKeyInput{IDCmkKey: &testIDKey, Algorithm: commonEntity.KeySymmetricDefault, Size: 256, Purpose: commonEntity.KeyPurposeEncrypt}, ""); err == nil {
		t.Fatal("expected wrap key error")
	}

	repo.security = fakeSecurityRepository{}
	sp.createVersionErr = errors.New("version failed")
	if _, err := repo.CreateKey(models.CreateKeyInput{IDCmkKey: &testIDKey, Algorithm: commonEntity.KeySymmetricDefault, Size: 256, Purpose: commonEntity.KeyPurposeEncrypt}, ""); err == nil {
		t.Fatal("expected save version error")
	}
}

func TestAdditionalFunctionCoverage(t *testing.T) {
	repo, sp, viewRepo, kekRepo := newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)

	sp.updateQueueErr = errors.New("processed failed")
	if err := repo.queueProcessedKey(testIDQueue, testIDKey); err == nil {
		t.Fatal("expected queue processed error")
	}
	sp.updateQueueErr = nil

	viewRepo.wrappingRefs = nil
	viewRepo.providerWrapErr = errors.New("provider query failed")
	if _, err := repo.getWrappingKeyRefByKEK(kekRepo.kekData); err == nil {
		t.Fatal("expected provider query error")
	}
	viewRepo.providerWrapErr = nil

	viewRepo.wrappingRefs = []views.CmkWrappingKeyRefView{{IDCmkWrappingKeyRef: testIDWrap, Provider: "local", KeyRef: "local", Version: "v1"}}
	if err := repo.RotateWrapKeyByKEK(*kekRepo.kekData); err != nil {
		t.Fatalf("unexpected rotate by kek success error: %v", err)
	}

	if _, err := repo.RotateKey("bad-secret"); err == nil {
		t.Fatal("expected rotate key load material error")
	}

	sp.createQueueErr = errors.New("create during rotate failed")
	if _, err := repo.RotateKey(testSecret); err == nil {
		t.Fatal("expected rotate key create error")
	}
	sp.createQueueErr = nil

	_, _, viewRepo, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	repo.views = viewRepo
	viewRepo.keys = nil
	if err := repo.transitionKeyStatus(testIDKey, commonEntity.KeyStatusEnabled, commonEntity.KeyStatusDisabled, "transition failed"); err == nil {
		t.Fatal("expected transition get key error")
	}

	repo, _, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	if err := repo.EnableKey("bad-secret"); err == nil {
		t.Fatal("expected enable invalid secret error")
	}

	repo, _, viewRepo, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusDisabled)
	viewRepo.keys = nil
	if err := repo.PendingDeletion(testSecret); err == nil {
		t.Fatal("expected pending deletion get key error")
	}
	viewRepo.keys = testViews(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusDisabled).keys
	viewRepo.versions = nil
	if err := repo.PendingDeletion(testSecret); err == nil {
		t.Fatal("expected pending deletion version error")
	}

	repo, sp, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusDisabled)
	sp.updateCmkKeyErr = errors.New("pending deletion status failed")
	if err := repo.PendingDeletion(testSecret); err == nil {
		t.Fatal("expected pending deletion status update error")
	}

	repo, _, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	if err := repo.DeleteKey("bad-secret"); err == nil {
		t.Fatal("expected delete invalid secret error")
	}

	repo, _, viewRepo, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	viewRepo.keys = nil
	if err := repo.DeleteKey(testSecret); err == nil {
		t.Fatal("expected delete get key error")
	}

	repo, _, viewRepo, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	viewRepo.queues = nil
	if err := repo.DeleteKey(testSecret); err == nil {
		t.Fatal("expected delete queue error")
	}

	originalNewJWTServiceFn := newJWTServiceFn
	t.Cleanup(func() {
		newJWTServiceFn = originalNewJWTServiceFn
	})
	repo, _, _, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	if err := repo.VerifyJWT(context.Background(), testSecret, "HS256", "token"); err == nil || err.Error() != errMsgKeyPurposeNotAllowJWTSigning {
		t.Fatalf("expected verify jwt purpose error, got %v", err)
	}
	if err := repo.ReadJWT(context.Background(), testSecret, "HS256", "token", map[string]any{}); err == nil || err.Error() != errMsgKeyPurposeNotAllowJWTSigning {
		t.Fatalf("expected read jwt purpose error, got %v", err)
	}
	if err := repo.VerifyJWT(context.Background(), "bad-secret", "HS256", "token"); err == nil {
		t.Fatal("expected verify jwt load material error")
	}
	if err := repo.ReadJWT(context.Background(), "bad-secret", "HS256", "token", map[string]any{}); err == nil {
		t.Fatal("expected read jwt load material error")
	}

	repo, sp, viewRepo, _ = newTestRepository(commonEntity.KeySymmetricDefault, commonEntity.KeyPurposeEncrypt, commonEntity.KeyStatusEnabled)
	viewRepo.matchAnyQueue = true
	secret, err := repo.CreateKey(models.CreateKeyInput{Algorithm: commonEntity.KeySymmetricDefault, Size: 256, Purpose: commonEntity.KeyPurposeEncrypt}, "")
	if err != nil {
		t.Fatalf("unexpected create key with generated id error: %v", err)
	}
	if secret == "" || sp.createVersionInput == nil {
		t.Fatalf("expected generated secret and version input, got %q %#v", secret, sp.createVersionInput)
	}
	_, generatedVersion := assertEncodedSecret(t, secret)
	if generatedVersion != sp.createVersionInput.IDCmkKeyVersion {
		t.Fatalf("expected encoded generated version %s, got %s", sp.createVersionInput.IDCmkKeyVersion, generatedVersion)
	}
}
