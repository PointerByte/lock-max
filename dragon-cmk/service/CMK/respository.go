package cmk

import (
	"context"
	"errors"
	"fmt"

	"github.com/PointerByte/GoForge/encrypt"
	modelsEncrypt "github.com/PointerByte/GoForge/encrypt/models"
	"github.com/PointerByte/GoForge/logger/builder"
	"github.com/PointerByte/GoForge/tools/workers"
	appCommon "github.com/PointerByte/lock-max/dragon-cmk/common"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/store"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/views"
	kek "github.com/PointerByte/lock-max/dragon-cmk/service/KEK"
	"github.com/google/uuid"
)

const (
	errMsgInvalidChecksum                 = "invalid checksum"
	errMsgUnsupportedAlgorithm            = "unsupported algorithm"
	errMsgUnsupportedECDHKeySize          = "unsupported key size for ECDH"
	errMsgKeyNotProcessed                 = "key is not processed yet"
	errMsgKeyNotFound                     = "key not found"
	errMsgKeyVersionNotFound              = "key version not found"
	errMsgKeyVersionMustBeEnabled         = "key version must be enabled"
	errMsgKeyVersionMainCannotBeUpdated   = "main key version cannot be updated"
	errMsgKeyVersionStatusInvalid         = "invalid key version status"
	errMsgKeyVersionStatusPendingDeletion = "pendingDeletion cannot be used to update a key version"
	errMsgWrappingKeyRefNotFound          = "wrapping key reference not found"
	errMsgKeyPurposeNotAllowEncryption    = "key purpose does not allow encryption"
	errMsgKeyPurposeNotAllowDecryption    = "key purpose does not allow decryption"
	errMsgKeyPurposeNotAllowSigning       = "key purpose does not allow signing"
	errMsgKeyPurposeNotAllowJWTSigning    = "key purpose does not allow JWT signing"
	errMsgKeyMustBeDisabledToEnable       = "key must be disabled to enable"
	errMsgKeyMustBeEnabledToDisable       = "key must be enabled to disable"
	errMsgKeyMustBeDisabledToMarkDeletion = "key must be disabled to mark pending deletion"
)

func init() {
	limit, ok, err := appCommon.WorkerLimit()
	if ok {
		if err != nil {
			panic(fmt.Sprintf("invalid %s: %v", appCommon.EnvWorkersLimit, err))
		}
		workers.SetWorkersLimit(limit)
	}
	workers.RunWorkers()
}

type Repository struct {
	ctx      context.Context
	kek      kek.IRepository
	sp       store.IRepository
	views    views.IRepository
	security encrypt.IRepository
}

func NewRepository(ctx context.Context, ctxLogger *builder.Context, repo encrypt.IRepository, kek kek.IRepository) IRepository {
	r := &Repository{
		ctx:      ctx,
		kek:      kek,
		sp:       store.NewRepository(ctx, ctxLogger),
		views:    views.NewRepository(ctx, ctxLogger),
		security: repo,
	}
	kek.SetFuncCreateKey(r.CreateKey)
	kek.SetFuncRotate(r.RotateKey)
	kek.SetFuncRotateWrapKey(r.RotateWrapKeyByKEK)
	kek.SetFuncDeleteWrapKey(r.DeleteWrapKeyByKEK)
	return r
}

type secret struct {
	idQueue        *uuid.UUID
	keyData        modelsEncrypt.KeyData
	secretWrapped  string
	wrapAlg        string
	secretChecksum *string
}

type chKeyResult struct {
	secret *secret
	err    error
}

type keyMaterial struct {
	cmkKey     *views.CmkKeyView
	keyVersion *views.CmkKeyVersionView
	key        string
	algorithm  string
}

func saveKeyWorker(
	keyData *secret,
	err error,
	chKey chan<- chKeyResult,
) {
	defer close(chKey)
	if err != nil {
		chKey <- chKeyResult{
			secret: nil,
			err:    err,
		}
		return
	}
	chKey <- chKeyResult{
		secret: keyData,
		err:    nil,
	}
}

func (r *Repository) contextWithTimeout() (context.Context, context.CancelFunc) {
	ctx := r.ctx
	return context.WithTimeout(ctx, appCommon.Timeout)
}

func contextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, appCommon.Timeout)
}

func requirePurpose(actual, expected commonEntity.KeyPurpose, errMsg string) error {
	if actual != expected {
		return errors.New(errMsg)
	}
	return nil
}
