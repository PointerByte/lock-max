package kek

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	goopenssl "github.com/PointerByte/GoForge/cmd/go-openssl/code"
	"github.com/PointerByte/lock-max/dragon-cmk/common"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"

	commonEncrypt "github.com/PointerByte/GoForge/encrypt/common"
	"github.com/google/uuid"
)

var store *sync.Map
var generateIDIndex *sync.Map

var (
	readDirFn                = os.ReadDir
	removeAllFn              = os.RemoveAll
	generateCertificatesFn   = goopenssl.GenerateCertificates
	readPrivateKeyFileFn     = goopenssl.ReadPrivateKeyFile
	marshalPKCS8PrivateKeyFn = x509.MarshalPKCS8PrivateKey
	marshalPKIXPublicKeyFn   = x509.MarshalPKIXPublicKey
)

const (
	certsDir      = "./certs"
	globalKeyRef  = "global"
	localKeyRef   = "local"
	localProvider = "local"
	globalVersion = "v1"

	errMsgCreateKeyFunctionNotConfigured     = "create key function not configured"
	errMsgRotateKeyFunctionNotConfigured     = "rotate key function not configured"
	errMsgRotateWrapKeyFunctionNotConfigured = "rotate wrap key function not configured"
	errMsgDeleteWrapKeyFunctionNotConfigured = "delete wrap key function not configured"
	errMsgVersionRequired                    = "version is required"
	errMsgCannotDeleteLatestKEK              = "cannot delete latest KEK version"
	errMsgKEKIDCannotBeNil                   = "id generate cannot be nil"
)

func init() {
	store = &sync.Map{}
	generateIDIndex = &sync.Map{}
}

type Repository struct {
	fnCreteKey      HandlerFuncCreteKey
	fnRotateKey     HandlerFuncRotateKey
	fnRotateWrapKey HandlerFuncRotateWrapKey
	fnDeleteWrapKey HandlerFuncDeleteWrapKey
}

type HandlerFuncCreteKey func(input models.CreateKeyInput, eventType commonEntity.EventType) (string, error)

type HandlerFuncRotateKey func(secretCmkKey string) (string, error)

type HandlerFuncRotateWrapKey func(kekData models.KEK) error

type HandlerFuncDeleteWrapKey func(kekData models.KEK) error

func NewRepository() IRepository {
	return &Repository{}
}

func (r *Repository) SetFuncCreateKey(fn HandlerFuncCreteKey) {
	r.fnCreteKey = fn
}

func (r *Repository) SetFuncRotate(fn HandlerFuncRotateKey) {
	r.fnRotateKey = fn
}

func (r *Repository) SetFuncRotateWrapKey(fn HandlerFuncRotateWrapKey) {
	r.fnRotateWrapKey = fn
}

func (r *Repository) SetFuncDeleteWrapKey(fn HandlerFuncDeleteWrapKey) {
	r.fnDeleteWrapKey = fn
}

func idDirectory(id uuid.UUID) string {
	if id == uuid.Nil {
		return fmt.Sprintf("%s/%s", certsDir, globalKeyRef)
	}
	return fmt.Sprintf("%s/%s", certsDir, id.String())
}

func wrappingDirectory(version string, id uuid.UUID) string {
	return fmt.Sprintf("%s/%s", idDirectory(id), version)
}

func storeKey(id uuid.UUID, version string) string {
	if id == uuid.Nil {
		return version
	}
	return fmt.Sprintf("%s/%s", version, id.String())
}

func storeGenerateID(idGenerate uuid.UUID, idKek uuid.UUID) {
	if idGenerate != uuid.Nil && idKek != uuid.Nil {
		generateIDIndex.Store(idGenerate.String(), idKek)
	}
}

func resolveGenerateID(idGenerate uuid.UUID) uuid.UUID {
	if idGenerate == uuid.Nil {
		return uuid.Nil
	}
	value, ok := generateIDIndex.Load(idGenerate.String())
	if !ok {
		return idGenerate
	}
	idKek, ok := value.(uuid.UUID)
	if !ok {
		return idGenerate
	}
	return idKek
}

func (r *Repository) getDirectories() []string {
	entries, err := readDirFn(certsDir)
	if err != nil {
		return nil
	}

	directories := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if entry.Name() == globalKeyRef {
			versionEntries, err := readDirFn(idDirectory(uuid.Nil))
			if err != nil {
				continue
			}
			for _, versionEntry := range versionEntries {
				if versionEntry.IsDir() {
					directories = append(directories, wrappingDirectory(versionEntry.Name(), uuid.Nil))
				}
			}
			continue
		}

		if strings.HasPrefix(entry.Name(), "v") {
			directories = append(directories, fmt.Sprintf("%s/%s", certsDir, entry.Name()))
			continue
		}

		id, err := uuid.Parse(entry.Name())
		if err != nil {
			continue
		}
		versionEntries, err := readDirFn(idDirectory(id))
		if err != nil {
			continue
		}
		for _, versionEntry := range versionEntries {
			if versionEntry.IsDir() {
				directories = append(directories, wrappingDirectory(versionEntry.Name(), id))
			}
		}
	}
	sort.Strings(directories)
	return directories
}

func (r *Repository) getLatestVersion() (string, uint) {
	lastVersion := r.getLastVersion()
	if lastVersion == 0 {
		return "v1", 1
	}
	return fmt.Sprintf("v%d", lastVersion), uint(lastVersion)
}

func (r *Repository) getLastVersion() int {
	lastVersion := 0
	for _, directory := range r.getDirectories() {
		index := strings.LastIndex(directory, "/")
		name := directory
		if index >= 0 {
			name = directory[index+1:]
		}
		if !strings.HasPrefix(name, "v") {
			continue
		}

		version, err := strconv.Atoi(strings.TrimPrefix(name, "v"))
		if err != nil || version < 1 {
			continue
		}
		if version > lastVersion {
			lastVersion = version
		}
	}
	return lastVersion
}

func (r *Repository) getNextVersion() (string, uint) {
	nextVersion := r.getLastVersion() + 1
	return fmt.Sprintf("v%d", nextVersion), uint(nextVersion)
}

func (r *Repository) searchVersionDirectory(version string) (string, uuid.UUID, error) {
	entries, err := readDirFn(certsDir)
	if err != nil {
		return "", uuid.Nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == version {
			return fmt.Sprintf("%s/%s", certsDir, entry.Name()), uuid.Nil, nil
		}

		if entry.Name() == globalKeyRef {
			outputDir := wrappingDirectory(version, uuid.Nil)
			if _, err := readDirFn(outputDir); err == nil {
				return outputDir, uuid.Nil, nil
			}
			continue
		}

		id, err := uuid.Parse(entry.Name())
		if err != nil {
			continue
		}
		outputDir := wrappingDirectory(version, id)
		if _, err := readDirFn(outputDir); err == nil {
			return outputDir, id, nil
		}
	}
	return "", uuid.Nil, fmt.Errorf("directory not found for version: %s", version)
}

func (r *Repository) searchDirectory(id uuid.UUID, version string) (string, uuid.UUID, error) {
	if id != uuid.Nil {
		outputDir := wrappingDirectory(version, id)
		if _, err := readDirFn(outputDir); err != nil {
			return "", uuid.Nil, err
		}
		return outputDir, id, nil
	}

	return r.searchVersionDirectory(version)
}

func ecdhPrivateKey(privateKeyAny any) (*ecdh.PrivateKey, error) {
	switch privateKey := privateKeyAny.(type) {
	case *ecdh.PrivateKey:
		return privateKey, nil
	case *ecdsa.PrivateKey:
		ecdhPrivateKey, err := privateKey.ECDH()
		if err != nil {
			return nil, fmt.Errorf("convert private key to ECDH: %w", err)
		}
		return ecdhPrivateKey, nil
	default:
		return nil, errors.New("private key is not an ECC key")
	}
}

func (r *Repository) loadFiles(privateKeyPath string) (privateDER []byte, publicDER []byte, _ error) {
	privateKeyAny, err := readPrivateKeyFileFn(privateKeyPath, common.KekLocalEncryptSecret())
	if err != nil {
		return nil, nil, err
	}
	privateKey, err := ecdhPrivateKey(privateKeyAny)
	if err != nil {
		return nil, nil, err
	}

	privateDER, err = marshalPKCS8PrivateKeyFn(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal ECC private key: %w", err)
	}

	publicDER, err = marshalPKIXPublicKeyFn(privateKey.PublicKey())
	if err != nil {
		return nil, nil, fmt.Errorf("marshal ECC public key: %w", err)
	}
	return
}

func (r *Repository) generateCertificate(id uuid.UUID, version, salt string) (privateDER, publicDER []byte, outputDir string, _ error) {
	outputDir = wrappingDirectory(version, id)
	result, err := generateCertificatesFn(goopenssl.Options{
		Algorithm:     "ecc",
		ECCCurve:      "p521",
		OutputDir:     outputDir,
		CommonName:    "localhost",
		Salt:          salt,
		EncryptSecret: common.KekLocalEncryptSecret(),
	})
	if err != nil {
		return nil, nil, "", err
	}

	privateDER, publicDER, err = r.loadFiles(result.PrivateKeyPath)
	if err != nil {
		return nil, nil, "", err
	}
	return privateDER, publicDER, outputDir, nil
}

func (r *Repository) loadCertificate(id uuid.UUID, version string) (*models.KEK, error) {
	outputDir, wrappingID, err := r.searchDirectory(id, version)
	if err != nil {
		return nil, err
	}

	privateDER, publicDER, err := r.loadFiles(fmt.Sprintf("%s/key.pem", outputDir))
	if err != nil {
		return nil, err
	}
	return newLocalKEK(wrappingID, version, "", privateDER, publicDER), nil
}

func newLocalKEK(id uuid.UUID, version, secretCmkKey string, privateDER, publicDER []byte) *models.KEK {
	if id == uuid.Nil {
		id = uuid.New()
	}
	return &models.KEK{
		PublicKey:           base64.StdEncoding.EncodeToString(publicDER),
		PrivateKey:          base64.StdEncoding.EncodeToString(privateDER),
		SecretCmkKey:        secretCmkKey,
		IdCmkWrappingKeyRef: id,
		KeyRef:              localKeyRef,
		Provider:            localProvider,
		Version:             version,
	}
}

func storeKEK(version string, kek *models.KEK) {
	store.Store(storeKey(kek.IdCmkWrappingKeyRef, version), *kek)
	store.Store(version, *kek)
}

func cleanupVersion(id uuid.UUID, version, outputDir string) {
	store.Delete(storeKey(id, version))
	store.Delete(version)
	_ = removeAllFn(outputDir)
}

func cleanupCreatedKEK(idGenerate uuid.UUID, idKek uuid.UUID, version string, outputDir string) {
	cleanupVersion(idKek, version, outputDir)
	if idGenerate != uuid.Nil {
		generateIDIndex.Delete(idGenerate.String())
	}
}

func (r *Repository) requireCreateKeyFunc() error {
	if r.fnCreteKey == nil {
		return errors.New(errMsgCreateKeyFunctionNotConfigured)
	}
	return nil
}

func (r *Repository) requireRotateKeyFunc() error {
	if r.fnRotateKey == nil {
		return errors.New(errMsgRotateKeyFunctionNotConfigured)
	}
	return nil
}

func (r *Repository) requireRotateWrapKeyFunc() error {
	if r.fnRotateWrapKey == nil {
		return errors.New(errMsgRotateWrapKeyFunctionNotConfigured)
	}
	return nil
}

func (r *Repository) requireDeleteWrapKeyFunc() error {
	if r.fnDeleteWrapKey == nil {
		return errors.New(errMsgDeleteWrapKeyFunctionNotConfigured)
	}
	return nil
}

func (r *Repository) createGlobalKEK(salt string) (*models.KEK, error) {
	if err := r.requireCreateKeyFunc(); err != nil {
		return nil, err
	}

	privateDER, publicDER, outputDir, err := r.generateCertificate(uuid.Nil, globalVersion, salt)
	if err != nil {
		return nil, err
	}

	idGenerate := uuid.New()
	secretCmkKey, err := r.fnCreteKey(models.CreateKeyInput{
		IDCmkKey:  &idGenerate,
		Algorithm: commonEntity.KeySymmetricDefault,
		Purpose:   commonEntity.KeyPurposeEncrypt,
		Size:      uint(commonEncrypt.Key256Bits),
		Version:   1,
	}, commonEntity.EventTypeCreateKey)
	if err != nil {
		cleanupVersion(uuid.Nil, globalVersion, outputDir)
		return nil, err
	}

	kekData := newLocalKEK(uuid.Nil, globalVersion, secretCmkKey, privateDER, publicDER)
	storeKEK(globalVersion, kekData)
	return kekData, nil
}

func (r *Repository) CreateKEK(idGenerate uuid.UUID, secretCmkKey string, salt string) (idKek *uuid.UUID, _ error) {
	if err := r.requireCreateKeyFunc(); err != nil {
		return nil, err
	}
	if secretCmkKey != "" {
		if err := r.requireRotateKeyFunc(); err != nil {
			return nil, err
		}
	}

	if idGenerate == uuid.Nil {
		return nil, errors.New(errMsgKEKIDCannotBeNil)
	}
	version, NumVersion := r.getNextVersion()
	_idKek := uuid.New()
	idKek = &_idKek
	privateDER, publicDER, outputDir, err := r.generateCertificate(_idKek, version, salt)
	if err != nil {
		return nil, err
	}
	storeGenerateID(idGenerate, _idKek)

	if secretCmkKey == "" {
		secretCmkKey, err := r.fnCreteKey(models.CreateKeyInput{
			IDCmkKey:  &idGenerate,
			Algorithm: commonEntity.KeySymmetricDefault,
			Purpose:   commonEntity.KeyPurposeEncrypt,
			Size:      uint(commonEncrypt.Key256Bits),
			Version:   NumVersion,
		}, commonEntity.EventTypeCreateKey)
		if err != nil {
			cleanupCreatedKEK(idGenerate, _idKek, version, outputDir)
			return nil, err
		}

		storeKEK(version, newLocalKEK(_idKek, version, secretCmkKey, privateDER, publicDER))
	} else {
		secretCmkKey, err = r.fnRotateKey(secretCmkKey)
		if err != nil {
			cleanupCreatedKEK(idGenerate, _idKek, version, outputDir)
			return nil, err
		}
		storeKEK(version, newLocalKEK(_idKek, version, secretCmkKey, privateDER, publicDER))
	}
	return idKek, nil
}

func (r *Repository) GetKEK(id uuid.UUID, version string) (*models.KEK, error) {
	id = resolveGenerateID(id)
	if id == uuid.Nil && version == "" {
		version = globalVersion
	} else if version == "" {
		version, _ = r.getLatestVersion()
	}

	value, ok := store.Load(storeKey(id, version))
	if !ok {
		kek, err := r.loadCertificate(id, version)
		if err != nil {
			if id != uuid.Nil || version != globalVersion {
				return nil, err
			}
			kek, err = r.createGlobalKEK("")
			if err != nil {
				return nil, err
			}
		}
		storeKEK(version, kek)
		value = *kek
	}

	kek := value.(models.KEK)
	return &kek, nil
}

func (r *Repository) RotateKEK(id uuid.UUID, salt string) (*uuid.UUID, error) {
	if err := r.requireRotateWrapKeyFunc(); err != nil {
		return nil, err
	}

	kekData, err := r.GetKEK(uuid.Nil, "")
	if err != nil {
		return nil, err
	}

	idKek, err := r.CreateKEK(id, kekData.SecretCmkKey, salt)
	if err != nil {
		return nil, err
	}

	return idKek, r.fnRotateWrapKey(*kekData)
}

func validatePagination(page uint, totalRegisterPage uint) error {
	if page == 0 {
		return errors.New("page must be greater than zero")
	}
	if totalRegisterPage == 0 {
		return errors.New("totalResgisterPage must be greater than zero")
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

func paginateStrings(values []string, page uint, totalRegisterPage uint) []string {
	start := int((page - 1) * totalRegisterPage)
	if start >= len(values) {
		return nil
	}
	end := start + int(totalRegisterPage)
	if end > len(values) {
		end = len(values)
	}
	return values[start:end]
}

func versionNumber(version string) int {
	number, err := strconv.Atoi(strings.TrimPrefix(version, "v"))
	if err != nil {
		return 0
	}
	return number
}

func (r *Repository) listVersions(id uuid.UUID) ([]string, error) {
	id = resolveGenerateID(id)
	versionEntries, err := readDirFn(idDirectory(id))
	if err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(versionEntries))
	for _, versionEntry := range versionEntries {
		if versionEntry.IsDir() && strings.HasPrefix(versionEntry.Name(), "v") {
			versions = append(versions, versionEntry.Name())
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		return versionNumber(versions[i]) < versionNumber(versions[j])
	})
	return versions, nil
}

func (r *Repository) ListKEK(id uuid.UUID, page uint, totalRegisterPage uint) (*models.PaginatedKEK, error) {
	if err := validatePagination(page, totalRegisterPage); err != nil {
		return nil, err
	}

	versions, err := r.listVersions(id)
	if err != nil {
		return nil, err
	}

	results := make([]models.KEK, 0, totalRegisterPage)
	idKek := resolveGenerateID(id)
	for _, version := range paginateStrings(versions, page, totalRegisterPage) {
		kekData, err := r.GetKEK(idKek, version)
		if err != nil {
			return nil, err
		}
		results = append(results, *kekData)
	}

	return &models.PaginatedKEK{
		Results:    results,
		Pagination: newPagination(uint(len(versions)), page, totalRegisterPage),
	}, nil
}

func (r *Repository) DeleteKey(id uuid.UUID, version string) error {
	if err := r.requireRotateWrapKeyFunc(); err != nil {
		return err
	}
	if err := r.requireDeleteWrapKeyFunc(); err != nil {
		return err
	}

	if version == "" {
		return errors.New(errMsgVersionRequired)
	}

	latestVersion, _ := r.getLatestVersion()
	if version == latestVersion {
		return errors.New(errMsgCannotDeleteLatestKEK)
	}

	kekData, err := r.GetKEK(id, version)
	if err != nil {
		return err
	}

	if err := r.fnRotateWrapKey(*kekData); err != nil {
		return err
	}

	if err := r.fnDeleteWrapKey(*kekData); err != nil {
		return err
	}

	outputDir, wrappingID, err := r.searchDirectory(id, version)
	if err != nil {
		return err
	}

	if err := removeAllFn(outputDir); err != nil {
		return err
	}

	store.Delete(storeKey(wrappingID, version))
	store.Delete(version)
	return nil
}
