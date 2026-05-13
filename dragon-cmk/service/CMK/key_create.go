package cmk

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	commonEncrypt "github.com/PointerByte/GoForge/encrypt/common"
	modelsEncrypt "github.com/PointerByte/GoForge/encrypt/models"
	"github.com/PointerByte/GoForge/tools/workers"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/store"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/google/uuid"
)

func (r *Repository) CreateKey(input models.CreateKeyInput, eventType commonEntity.EventType) (string, error) {
	if input.IDCmkKey == nil {
		id := uuid.New()
		input.IDCmkKey = &id
	}

	chKey := make(chan chKeyResult)
	worker := func() {
		if err := r.saveKey(*input.IDCmkKey, input.Algorithm, input.Purpose); err != nil {
			saveKeyWorker(nil, err, chKey)
			return
		}
		if err := r.saveWrappingKeyRef(); err != nil {
			saveKeyWorker(nil, err, chKey)
			return
		}
		if eventType == "" {
			eventType = commonEntity.EventTypeCreateKey
		}

		idQueue, err := r.createQueue(*input.IDCmkKey, eventType)
		if err != nil {
			saveKeyWorker(nil, err, chKey)
			return
		}

		keyData, err := r.generateKey(input.Algorithm, input.Size)
		if err != nil {
			if _err := r.queueFailedKey(*idQueue, *input.IDCmkKey, err.Error()); _err != nil {
				err = errors.Join(err, _err)
			}
			saveKeyWorker(nil, err, chKey)
			return
		}

		if err := r.queueProcessingKey(*idQueue, *input.IDCmkKey); err != nil {
			saveKeyWorker(nil, err, chKey)
			return
		}

		secretWrapped, wrapAlg, secretChecksum, err := r.wrapKey(keyData.KeyID)
		if err != nil {
			saveKeyWorker(nil, err, chKey)
			return
		}

		result := &secret{
			idQueue:        idQueue,
			keyData:        *keyData,
			secretWrapped:  secretWrapped,
			wrapAlg:        wrapAlg,
			secretChecksum: secretChecksum,
		}
		saveKeyWorker(result, nil, chKey)
		if err := r.queueProcessedKey(*idQueue, *input.IDCmkKey); err != nil {
			saveKeyWorker(nil, err, chKey)
			return
		}
	}
	workers.AddTask(worker)

	result := <-chKey
	if result.err != nil {
		return "", result.err
	}

	idCmkKeyVersion := uuid.New()
	if input.Version == 0 {
		input.Version = 1
	}

	kekData, err := r.kek.GetKEK(uuid.Nil, "")
	if err != nil {
		return "", err
	}
	secretCmkKey := fmt.Sprintf("%s.%s", result.secret.idQueue.String(), idCmkKeyVersion.String())
	return base64.StdEncoding.EncodeToString([]byte(secretCmkKey)),
		r.saveKeyVersionKey(
			idCmkKeyVersion,
			*input.IDCmkKey,
			kekData.IdCmkWrappingKeyRef,
			input.Version,
			input.Size,
			result.secret.keyData.PublicKey,
			result.secret.secretWrapped,
			result.secret.wrapAlg,
			result.secret.secretChecksum,
		)
}

func (r *Repository) generateKey(algorithm commonEntity.KeyType, size uint) (*modelsEncrypt.KeyData, error) {
	ctx, cancel := r.contextWithTimeout()
	defer cancel()

	switch algorithm {
	case commonEntity.KeySymmetricDefault:
		key, err := r.security.GenerateSymetrycKeys(ctx, commonEncrypt.SizeSymetrycKey(size))
		if err != nil {
			return nil, err
		}
		return key, nil
	case commonEntity.KeyTypeRSAOAEP:
		key, err := r.security.GenerateRSAKeys(ctx, commonEncrypt.SizeAsymetrycKey(size))
		if err != nil {
			return nil, err
		}
		return key, nil
	case commonEntity.KeyTypeECDH:
		var curve commonEncrypt.CurveAsymmetricKey
		switch size {
		case 256:
			curve = commonEncrypt.CurveP256
		case 384:
			curve = commonEncrypt.CurveP384
		case 521:
			curve = commonEncrypt.CurveP521
		default:
			return nil, errors.New(errMsgUnsupportedECDHKeySize)
		}
		key, err := r.security.GenerateECCKeys(ctx, curve)
		if err != nil {
			return nil, err
		}
		return key, nil
	case commonEntity.KeyTypeEdDSA:
		key, err := r.security.GenerateEd255Keys(ctx)
		if err != nil {
			return nil, err
		}
		return key, nil
	default:
		return nil, errors.New(errMsgUnsupportedAlgorithm)
	}
}

func (r *Repository) createQueue(
	idCmkKey uuid.UUID,
	eventType commonEntity.EventType,
) (*uuid.UUID, error) {
	id := uuid.New()
	now := time.Now()
	status := commonEntity.QueueStatusPending
	_, err := r.sp.CreateKeyCreationQueue(store.CreateKeyCreationQueueInput{
		IDCmkKeyCreationQueue: id,
		IDCmkKey:              idCmkKey,
		EventType:             eventType,
		Status:                &status,
		ProcessedAt:           &now,
	})
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *Repository) queueProcessingKey(idQueue, idCmkKey uuid.UUID) error {
	if err := r.updateKeyCreationQueue(idQueue, commonEntity.QueueStatusPending, nil); err != nil {
		return err
	}
	return r.updateKeyStatusIfExists(idCmkKey, commonEntity.KeyStatusPendingImport)
}

func (r *Repository) queueProcessedKey(idQueue, idCmkKey uuid.UUID) error {
	if err := r.updateKeyCreationQueue(idQueue, commonEntity.QueueStatusPending, nil); err != nil {
		return err
	}
	return r.updateKeyStatusIfExists(idCmkKey, commonEntity.KeyStatusEnabled)
}

func (r *Repository) queueFailedKey(idQueue, idCmkKey uuid.UUID, errorMessage string) error {
	if err := r.updateKeyCreationQueue(idQueue, commonEntity.QueueStatusPending, &errorMessage); err != nil {
		return err
	}
	return r.updateKeyStatusIfExists(idCmkKey, commonEntity.KeyStatusUnavailable)
}

func (r *Repository) updateKeyCreationQueue(idQueue uuid.UUID, status commonEntity.QueueStatus, errorMessage *string) error {
	now := time.Now()
	_, err := r.getQueueData(&idQueue, nil)
	if err != nil {
		return err
	}
	_, err = r.sp.UpdateKeyCreationQueue(store.UpdateKeyCreationQueueInput{
		IDCmkKeyCreationQueue: idQueue,
		Status:                &status,
		ProcessedAt:           &now,
		ErrorMessage:          errorMessage,
	})
	return err
}

func (r *Repository) updateKeyStatusIfExists(idCmkKey uuid.UUID, status commonEntity.KeyStatus) error {
	_, err := r.getKey(idCmkKey)
	if err == nil {
		return r.updateKeyStatus(idCmkKey, status)
	}
	return nil
}

func (r *Repository) saveKey(
	idCmkKey uuid.UUID,
	algorithm commonEntity.KeyType,
	purpose commonEntity.KeyPurpose,
) error {
	status := commonEntity.KeyStatusPendingImport
	_, err := r.getKey(idCmkKey)
	if err == nil {
		return r.updateKeyStatus(idCmkKey, status)
	}

	_, err = r.sp.CreateCmkKey(store.CreateCmkKeyInput{
		IDCmkKey:  idCmkKey,
		Algorithm: algorithm,
		Purpose:   purpose,
		Status:    &status,
	})
	return err
}

func (r *Repository) saveKeyVersionKey(
	idCmkKeyVersion,
	idCmkKey uuid.UUID,
	idCmkWrappingKeyRef uuid.UUID,
	version uint,
	size uint,
	kid,
	secretWrapped,
	wrapAlg string,
	secretChecksum *string,
) error {
	status := commonEntity.KeyVersionStatusEnabled
	_, err := r.sp.CreateKeyVersion(store.CreateKeyVersionInput{
		IDCmkKeyVersion:     idCmkKeyVersion,
		IDCmkKey:            idCmkKey,
		IDCmkWrappingKeyRef: &idCmkWrappingKeyRef,
		VersionNumber:       int(version),
		Size:                int(size),
		KID:                 kid,
		SecretWrapped:       secretWrapped,
		WrapAlg:             wrapAlg,
		SecretChecksum:      secretChecksum,
		Status:              &status,
	})
	return err
}
