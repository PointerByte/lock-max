package kek

import (
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"

	goopenssl "github.com/PointerByte/GoForge/cmd/go-openssl/code"
	appCommon "github.com/PointerByte/lock-max/dragon-cmk/common"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/google/uuid"
)

type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.isDir }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func resetKEKTestState(t *testing.T) {
	t.Helper()
	originalStore := store
	originalGenerateIDIndex := generateIDIndex
	originalReadDirFn := readDirFn
	originalRemoveAllFn := removeAllFn
	originalGenerateCertificatesFn := generateCertificatesFn
	originalReadPrivateKeyFileFn := readPrivateKeyFileFn
	originalMarshalPKCS8PrivateKeyFn := marshalPKCS8PrivateKeyFn
	originalMarshalPKIXPublicKeyFn := marshalPKIXPublicKeyFn
	store = &sync.Map{}
	generateIDIndex = &sync.Map{}
	readDirFn = os.ReadDir
	removeAllFn = os.RemoveAll
	generateCertificatesFn = goopenssl.GenerateCertificates
	readPrivateKeyFileFn = goopenssl.ReadPrivateKeyFile
	marshalPKCS8PrivateKeyFn = originalMarshalPKCS8PrivateKeyFn
	marshalPKIXPublicKeyFn = originalMarshalPKIXPublicKeyFn
	t.Cleanup(func() {
		store = originalStore
		generateIDIndex = originalGenerateIDIndex
		readDirFn = originalReadDirFn
		removeAllFn = originalRemoveAllFn
		generateCertificatesFn = originalGenerateCertificatesFn
		readPrivateKeyFileFn = originalReadPrivateKeyFileFn
		marshalPKCS8PrivateKeyFn = originalMarshalPKCS8PrivateKeyFn
		marshalPKIXPublicKeyFn = originalMarshalPKIXPublicKeyFn
	})
}

func fakePrivateKey(t *testing.T) *ecdh.PrivateKey {
	t.Helper()
	key, err := ecdh.P521().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

func configureCertificateSuccess(t *testing.T) {
	t.Helper()
	key := fakePrivateKey(t)
	generateCertificatesFn = func(options goopenssl.Options) (goopenssl.Result, error) {
		return goopenssl.Result{PrivateKeyPath: options.OutputDir + "/key.pem"}, nil
	}
	readPrivateKeyFileFn = func(path string, secret string) (any, error) {
		return key, nil
	}
	marshalPKCS8PrivateKeyFn = func(key any) ([]byte, error) {
		return []byte("private-der"), nil
	}
	marshalPKIXPublicKeyFn = func(key any) ([]byte, error) {
		return []byte("public-der"), nil
	}
}

func TestRepositoryConstructorsAndSetters(t *testing.T) {
	repository := NewRepository()
	r, ok := repository.(*Repository)
	if !ok {
		t.Fatalf("expected *Repository, got %T", repository)
	}

	createFn := func(input models.CreateKeyInput, eventType commonEntity.EventType) (string, error) {
		return "secret", nil
	}
	rotateFn := func(secretCmkKey string) (string, error) {
		return secretCmkKey, nil
	}
	rotateWrapFn := func(kekData models.KEK) error {
		return nil
	}
	deleteWrapFn := func(kekData models.KEK) error {
		return nil
	}
	r.SetFuncCreateKey(createFn)
	r.SetFuncRotate(rotateFn)
	r.SetFuncRotateWrapKey(rotateWrapFn)
	r.SetFuncDeleteWrapKey(deleteWrapFn)

	if err := r.requireCreateKeyFunc(); err != nil {
		t.Fatalf("unexpected create func error: %v", err)
	}
	if err := r.requireRotateKeyFunc(); err != nil {
		t.Fatalf("unexpected rotate func error: %v", err)
	}
	if err := r.requireRotateWrapKeyFunc(); err != nil {
		t.Fatalf("unexpected rotate wrap func error: %v", err)
	}
	if err := r.requireDeleteWrapKeyFunc(); err != nil {
		t.Fatalf("unexpected delete wrap func error: %v", err)
	}
}

func TestVersionDirectoryHelpers(t *testing.T) {
	resetKEKTestState(t)
	r := &Repository{}
	idV2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	idV10 := uuid.MustParse("10101010-1010-1010-1010-101010101010")
	readDirFn = func(string) ([]os.DirEntry, error) {
		return nil, errors.New("unexpected path")
	}
	readDirFn = func(path string) ([]os.DirEntry, error) {
		switch path {
		case certsDir:
			return []os.DirEntry{
				fakeDirEntry{name: idV2.String(), isDir: true},
				fakeDirEntry{name: "file", isDir: false},
				fakeDirEntry{name: idV10.String(), isDir: true},
				fakeDirEntry{name: "bad", isDir: true},
			}, nil
		case idDirectory(idV2):
			return []os.DirEntry{fakeDirEntry{name: "v2", isDir: true}}, nil
		case idDirectory(idV10):
			return []os.DirEntry{fakeDirEntry{name: "v10", isDir: true}}, nil
		default:
			return nil, errors.New("unexpected path")
		}
	}

	wrappingID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	if got := wrappingDirectory("v3", wrappingID); got != "./certs/11111111-1111-1111-1111-111111111111/v3" {
		t.Fatalf("unexpected wrapping directory: %s", got)
	}
	wantDirectories := []string{
		"./certs/10101010-1010-1010-1010-101010101010/v10",
		"./certs/22222222-2222-2222-2222-222222222222/v2",
	}
	if got := r.getDirectories(); !reflect.DeepEqual(got, wantDirectories) {
		t.Fatalf("unexpected directories: %#v", got)
	}
	if got := r.getLastVersion(); got != 10 {
		t.Fatalf("unexpected last version: %d", got)
	}
	if version, number := r.getLatestVersion(); version != "v10" || number != 10 {
		t.Fatalf("unexpected latest version: %s %d", version, number)
	}
	if version, number := r.getNextVersion(); version != "v11" || number != 11 {
		t.Fatalf("unexpected next version: %s %d", version, number)
	}
	if got := storeKey(wrappingID, "v3"); got != "v3/11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected store key: %s", got)
	}
}

func TestDirectoryErrorsAndSearch(t *testing.T) {
	resetKEKTestState(t)
	r := &Repository{}
	readDirFn = func(string) ([]os.DirEntry, error) {
		return nil, errors.New("read failed")
	}
	if got := r.getDirectories(); got != nil {
		t.Fatalf("expected nil directories, got %#v", got)
	}
	if version, number := r.getLatestVersion(); version != "v1" || number != 1 {
		t.Fatalf("unexpected default latest version: %s %d", version, number)
	}
	if _, _, err := r.searchDirectory(uuid.Nil, "v1"); err == nil {
		t.Fatal("expected read error")
	}

	readDirFn = func(string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{name: "v1", isDir: true}, fakeDirEntry{name: "v2", isDir: false}}, nil
	}
	if got, wrappingID, err := r.searchDirectory(uuid.Nil, "v1"); err != nil || got != "./certs/v1" || wrappingID != uuid.Nil {
		t.Fatalf("unexpected search result: %s %v", got, err)
	}
	if _, _, err := r.searchDirectory(uuid.Nil, "v2"); err == nil {
		t.Fatal("expected missing directory error")
	}

	id := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	readDirFn = func(path string) ([]os.DirEntry, error) {
		switch path {
		case certsDir:
			return []os.DirEntry{fakeDirEntry{name: id.String(), isDir: true}}, nil
		case "./certs/22222222-2222-2222-2222-222222222222/v1":
			return nil, nil
		default:
			return nil, errors.New("missing")
		}
	}
	if got, wrappingID, err := r.searchDirectory(uuid.Nil, "v1"); err != nil || got != "./certs/22222222-2222-2222-2222-222222222222/v1" || wrappingID != id {
		t.Fatalf("unexpected uuid search result: %s %s %v", got, wrappingID, err)
	}
	if got, wrappingID, err := r.searchDirectory(id, "v1"); err != nil || got != "./certs/22222222-2222-2222-2222-222222222222/v1" || wrappingID != id {
		t.Fatalf("unexpected explicit uuid search result: %s %s %v", got, wrappingID, err)
	}
}

func TestCertificateEncryptSecretFromEnv(t *testing.T) {
	t.Setenv(appCommon.EnvKekLocalEncryptSecret, " 12345678901234567890123456789012 ")
	if got := appCommon.KekLocalEncryptSecret(); got != "12345678901234567890123456789012" {
		t.Fatalf("unexpected certificate encrypt secret: %q", got)
	}
}

func TestLoadFiles(t *testing.T) {
	resetKEKTestState(t)
	r := &Repository{}
	key := fakePrivateKey(t)

	readPrivateKeyFileFn = func(string, string) (any, error) {
		return nil, errors.New("parse failed")
	}
	if _, _, err := r.loadFiles("key.pem"); err == nil {
		t.Fatal("expected parse error")
	}

	readPrivateKeyFileFn = func(string, string) (any, error) {
		return key, nil
	}
	marshalPKCS8PrivateKeyFn = func(any) ([]byte, error) {
		return nil, errors.New("private marshal failed")
	}
	if _, _, err := r.loadFiles("key.pem"); err == nil {
		t.Fatal("expected private marshal error")
	}

	marshalPKCS8PrivateKeyFn = func(any) ([]byte, error) {
		return []byte("private"), nil
	}
	marshalPKIXPublicKeyFn = func(any) ([]byte, error) {
		return nil, errors.New("public marshal failed")
	}
	if _, _, err := r.loadFiles("key.pem"); err == nil {
		t.Fatal("expected public marshal error")
	}

	marshalPKIXPublicKeyFn = func(any) ([]byte, error) {
		return []byte("public"), nil
	}
	privateDER, publicDER, err := r.loadFiles("key.pem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(privateDER) != "private" || string(publicDER) != "public" {
		t.Fatalf("unexpected der data: %q %q", privateDER, publicDER)
	}

	t.Setenv(appCommon.EnvKekLocalEncryptSecret, "12345678901234567890123456789012")
	readSecret := ""
	readPrivateKeyFileFn = func(path string, secret string) (any, error) {
		readSecret = secret
		return key, nil
	}
	if _, _, err := r.loadFiles("key.pem"); err != nil {
		t.Fatalf("unexpected encrypted load error: %v", err)
	}
	if readSecret != "12345678901234567890123456789012" {
		t.Fatalf("unexpected read secret: %q", readSecret)
	}

	readPrivateKeyFileFn = func(string, string) (any, error) {
		return "not-ecc", nil
	}
	if _, _, err := r.loadFiles("key.pem"); err == nil {
		t.Fatal("expected non ECC key error")
	}
}

func TestGenerateAndLoadCertificate(t *testing.T) {
	resetKEKTestState(t)
	r := &Repository{}
	id := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	generateCertificatesFn = func(goopenssl.Options) (goopenssl.Result, error) {
		return goopenssl.Result{}, errors.New("generate failed")
	}
	if _, _, _, err := r.generateCertificate(id, "v1", "salt"); err == nil {
		t.Fatal("expected generate error")
	}

	configureCertificateSuccess(t)
	t.Setenv(appCommon.EnvKekLocalEncryptSecret, "12345678901234567890123456789012")
	var generateOptions goopenssl.Options
	generateCertificatesFn = func(options goopenssl.Options) (goopenssl.Result, error) {
		generateOptions = options
		return goopenssl.Result{PrivateKeyPath: options.OutputDir + "/key.pem"}, nil
	}
	privateDER, publicDER, outputDir, err := r.generateCertificate(id, "v1", "salt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(privateDER) != "private-der" || string(publicDER) != "public-der" || outputDir != "./certs/33333333-3333-3333-3333-333333333333/v1" {
		t.Fatalf("unexpected certificate result: %q %q %s", privateDER, publicDER, outputDir)
	}
	if generateOptions.EncryptSecret != "12345678901234567890123456789012" {
		t.Fatalf("unexpected generate encrypt secret: %q", generateOptions.EncryptSecret)
	}

	readDirFn = func(string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{name: "v1", isDir: true}}, nil
	}
	kekData, err := r.loadCertificate(uuid.Nil, "v1")
	if err != nil {
		t.Fatalf("unexpected load certificate error: %v", err)
	}
	if kekData.Version != "v1" || kekData.Provider != localProvider || kekData.KeyRef != localKeyRef {
		t.Fatalf("unexpected kek data: %#v", kekData)
	}

	readPrivateKeyFileFn = func(string, string) (any, error) {
		return nil, errors.New("parse failed")
	}
	if _, _, _, err := r.generateCertificate(id, "v2", "salt"); err == nil {
		t.Fatal("expected generate certificate load error")
	}
	if _, err := r.loadCertificate(uuid.Nil, "v1"); err == nil {
		t.Fatal("expected load certificate file error")
	}

	readDirFn = func(string) ([]os.DirEntry, error) {
		return nil, errors.New("read failed")
	}
	if _, err := r.loadCertificate(uuid.Nil, "v1"); err == nil {
		t.Fatal("expected load certificate search error")
	}
}

func TestLocalStoreAndCleanup(t *testing.T) {
	resetKEKTestState(t)
	id := uuid.New()
	kekData := &models.KEK{IdCmkWrappingKeyRef: id, Version: "v1"}
	storeKEK("v1", kekData)
	value, ok := store.Load("v1")
	if !ok {
		t.Fatal("expected value in store")
	}
	if value.(models.KEK).IdCmkWrappingKeyRef != id {
		t.Fatalf("unexpected stored value: %#v", value)
	}

	removed := ""
	removeAllFn = func(path string) error {
		removed = path
		return nil
	}
	cleanupVersion(id, "v1", "./certs/"+id.String()+"/v1")
	if _, ok := store.Load("v1"); ok {
		t.Fatal("expected value removed from store")
	}
	if _, ok := store.Load(storeKey(id, "v1")); ok {
		t.Fatal("expected keyed value removed from store")
	}
	if removed != "./certs/"+id.String()+"/v1" {
		t.Fatalf("unexpected removed path: %s", removed)
	}
}

func TestCreateKEK(t *testing.T) {
	resetKEKTestState(t)
	readDirFn = func(string) ([]os.DirEntry, error) {
		return nil, nil
	}
	configureCertificateSuccess(t)
	r := &Repository{}
	idGenerate := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	if _, err := r.CreateKEK(uuid.Nil, "", "salt"); err == nil || err.Error() != errMsgCreateKeyFunctionNotConfigured {
		t.Fatalf("expected create func error, got %v", err)
	}

	r.SetFuncCreateKey(func(input models.CreateKeyInput, eventType commonEntity.EventType) (string, error) {
		if input.IDCmkKey == nil || *input.IDCmkKey != idGenerate {
			t.Fatalf("unexpected id generate: %#v", input.IDCmkKey)
		}
		if input.Algorithm != commonEntity.KeySymmetricDefault || input.Purpose != commonEntity.KeyPurposeEncrypt || input.Size != 32 || input.Version != 1 {
			t.Fatalf("unexpected create input: %#v", input)
		}
		if eventType != commonEntity.EventTypeCreateKey {
			t.Fatalf("unexpected event type: %s", eventType)
		}
		return "secret", nil
	})
	idKek, err := r.CreateKEK(idGenerate, "", "salt")
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if idKek == nil || *idKek == idGenerate {
		t.Fatalf("expected generated id_kek different from id_generate, got %v", idKek)
	}
	if _, ok := store.Load("v1"); !ok {
		t.Fatal("expected created KEK in store")
	}

	if _, err := r.CreateKEK(idGenerate, "existing", "salt"); err == nil || err.Error() != errMsgRotateKeyFunctionNotConfigured {
		t.Fatalf("expected rotate func error, got %v", err)
	}

	r.SetFuncRotate(func(secretCmkKey string) (string, error) {
		return "", errors.New("rotate failed")
	})
	if _, err := r.CreateKEK(idGenerate, "existing", "salt"); err == nil {
		t.Fatal("expected rotate error")
	}

	r.SetFuncRotate(func(secretCmkKey string) (string, error) {
		return secretCmkKey + "-rotated", nil
	})
	if _, err := r.CreateKEK(idGenerate, "existing", "salt"); err != nil {
		t.Fatalf("unexpected rotate success error: %v", err)
	}
}

func TestCreateKEKCleanupOnCreateError(t *testing.T) {
	resetKEKTestState(t)
	readDirFn = func(string) ([]os.DirEntry, error) {
		return nil, nil
	}
	configureCertificateSuccess(t)
	removed := ""
	id := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	removeAllFn = func(path string) error {
		removed = path
		return nil
	}
	r := &Repository{}
	r.SetFuncCreateKey(func(models.CreateKeyInput, commonEntity.EventType) (string, error) {
		return "", errors.New("create failed")
	})

	if _, err := r.CreateKEK(id, "", "salt"); err == nil {
		t.Fatal("expected create error")
	}
	if removed == "" {
		t.Fatalf("unexpected cleanup path: %s", removed)
	}
}

func TestGetRotateAndDeleteKEK(t *testing.T) {
	resetKEKTestState(t)
	idWrap := uuid.New()
	r := &Repository{}
	kekData := &models.KEK{
		IdCmkWrappingKeyRef: idWrap,
		SecretCmkKey:        "secret",
		PublicKey:           "public",
		PrivateKey:          "private",
		KeyRef:              localKeyRef,
		Provider:            localProvider,
		Version:             "v1",
	}
	storeKEK("v1", kekData)
	readDirFn = func(string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{name: "v1", isDir: true}, fakeDirEntry{name: "v2", isDir: true}}, nil
	}

	got, err := r.GetKEK(uuid.Nil, "v1")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if got.SecretCmkKey != "secret" {
		t.Fatalf("unexpected KEK: %#v", got)
	}

	if _, err := r.RotateKEK(uuid.Nil, "salt"); err == nil || err.Error() != errMsgRotateWrapKeyFunctionNotConfigured {
		t.Fatalf("expected rotate wrap func error, got %v", err)
	}

	r.SetFuncCreateKey(func(models.CreateKeyInput, commonEntity.EventType) (string, error) {
		return "new-secret", nil
	})
	r.SetFuncRotate(func(secretCmkKey string) (string, error) {
		return secretCmkKey, nil
	})
	rotatedWrap := false
	r.SetFuncRotateWrapKey(func(kekData models.KEK) error {
		rotatedWrap = true
		return nil
	})
	deletedWrap := false
	r.SetFuncDeleteWrapKey(func(kekData models.KEK) error {
		deletedWrap = true
		return nil
	})
	configureCertificateSuccess(t)
	idGenerate := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	if _, err := r.RotateKEK(idGenerate, "salt"); err != nil {
		t.Fatalf("unexpected rotate error: %v", err)
	}
	if !rotatedWrap {
		t.Fatal("expected wrapping keys to rotate")
	}
	if deletedWrap {
		t.Fatal("rotate should not delete wrapping keys")
	}

	if err := r.DeleteKey(uuid.Nil, ""); err == nil || err.Error() != errMsgVersionRequired {
		t.Fatalf("expected version required error, got %v", err)
	}
	if err := r.DeleteKey(uuid.Nil, "v2"); err == nil || err.Error() != errMsgCannotDeleteLatestKEK {
		t.Fatalf("expected latest delete error, got %v", err)
	}

	removed := ""
	deleteOrder := make([]string, 0, 3)
	removeAllFn = func(path string) error {
		deleteOrder = append(deleteOrder, "remove")
		removed = path
		return nil
	}
	storeKEK("v1", kekData)
	rotatedWrap = false
	deletedWrap = false
	r.SetFuncRotateWrapKey(func(kekData models.KEK) error {
		deleteOrder = append(deleteOrder, "rotate")
		rotatedWrap = true
		return nil
	})
	r.SetFuncDeleteWrapKey(func(kekData models.KEK) error {
		deleteOrder = append(deleteOrder, "delete")
		deletedWrap = true
		return nil
	})
	if err := r.DeleteKey(uuid.Nil, "v1"); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if removed != "./certs/v1" {
		t.Fatalf("unexpected removed path: %s", removed)
	}
	if _, ok := store.Load("v1"); ok {
		t.Fatal("expected deleted KEK removed from store")
	}
	if !deletedWrap {
		t.Fatal("expected delete wrapping key callback")
	}
	if !reflect.DeepEqual(deleteOrder, []string{"rotate", "delete", "remove"}) {
		t.Fatalf("unexpected delete order: %#v", deleteOrder)
	}
}

func TestRotateKEKErrors(t *testing.T) {
	resetKEKTestState(t)
	r := &Repository{}
	r.SetFuncRotateWrapKey(func(models.KEK) error {
		return nil
	})
	readDirFn = func(string) ([]os.DirEntry, error) {
		return nil, errors.New("read failed")
	}
	if _, err := r.RotateKEK(uuid.Nil, "salt"); err == nil {
		t.Fatal("expected get kek error")
	}

	storeKEK("v1", &models.KEK{SecretCmkKey: "secret", Version: "v1"})
	readDirFn = func(string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{name: "v1", isDir: true}}, nil
	}
	if _, err := r.RotateKEK(uuid.MustParse("55555555-5555-5555-5555-555555555555"), "salt"); err == nil || err.Error() != errMsgCreateKeyFunctionNotConfigured {
		t.Fatalf("expected create kek error, got %v", err)
	}
}

func TestDeleteKeyErrors(t *testing.T) {
	resetKEKTestState(t)
	r := &Repository{}
	if err := r.DeleteKey(uuid.Nil, "v1"); err == nil || err.Error() != errMsgRotateWrapKeyFunctionNotConfigured {
		t.Fatalf("expected missing rotate wrap func error, got %v", err)
	}

	r.SetFuncRotateWrapKey(func(models.KEK) error {
		return errors.New("rotate wrap failed")
	})
	if err := r.DeleteKey(uuid.Nil, "v1"); err == nil || err.Error() != errMsgDeleteWrapKeyFunctionNotConfigured {
		t.Fatalf("expected missing delete wrap func error, got %v", err)
	}

	r.SetFuncDeleteWrapKey(func(models.KEK) error {
		return nil
	})
	readDirFn = func(string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{name: "v1", isDir: true}, fakeDirEntry{name: "v2", isDir: true}}, nil
	}
	storeKEK("v1", &models.KEK{Version: "v1"})
	if err := r.DeleteKey(uuid.Nil, "v1"); err == nil {
		t.Fatal("expected rotate wrap error")
	}

	r.SetFuncRotateWrapKey(func(models.KEK) error {
		return nil
	})
	readDirFn = func(string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{name: "v2", isDir: true}}, nil
	}
	if err := r.DeleteKey(uuid.Nil, "v1"); err == nil {
		t.Fatal("expected get kek error")
	}

	storeKEK("v1", &models.KEK{Version: "v1"})
	r.SetFuncDeleteWrapKey(func(models.KEK) error {
		return errors.New("delete wrap failed")
	})
	if err := r.DeleteKey(uuid.Nil, "v1"); err == nil {
		t.Fatal("expected delete wrap error")
	}

	r.SetFuncDeleteWrapKey(func(models.KEK) error {
		return nil
	})
	if err := r.DeleteKey(uuid.Nil, "v1"); err == nil {
		t.Fatal("expected search directory error")
	}

	readDirFn = func(string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{name: "v1", isDir: true}, fakeDirEntry{name: "v2", isDir: true}}, nil
	}
	removeAllFn = func(string) error {
		return errors.New("remove failed")
	}
	if err := r.DeleteKey(uuid.Nil, "v1"); err == nil {
		t.Fatal("expected remove error")
	}
}
