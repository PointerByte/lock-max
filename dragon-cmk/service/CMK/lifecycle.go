package cmk

import (
	"errors"
	"time"

	"github.com/PointerByte/GoForge/logger/builder"
	"github.com/PointerByte/GoForge/tools/jobs"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/store"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/PointerByte/lock-max/dragon-cmk/service/utilities"
	"github.com/google/uuid"
)

func (r *Repository) RotateKey(secretCmkKeyIn string) (secretCmkKeyOut string, err error) {
	material, err := r.loadKeyMaterial(secretCmkKeyIn)
	if err != nil {
		return "", err
	}

	secretCmkKeyOut, err = r.CreateKey(models.CreateKeyInput{
		IDCmkKey:  &material.cmkKey.IDCmkKey,
		Algorithm: commonEntity.KeyType(material.algorithm),
		Size:      uint(material.keyVersion.Size),
		Purpose:   material.cmkKey.Purpose,
		Version:   uint(material.keyVersion.VersionNumber + 1),
	}, commonEntity.EventTypeRotateKey)
	if err != nil {
		return "", err
	}

	_, idCmkKeyVersion, err := utilities.GetSecretCmkKey(secretCmkKeyOut)
	if err != nil {
		return "", err
	}

	_, err = r.sp.RotateKeyVersion(store.RotateKeyVersionInput{
		IDCmkKey:        material.cmkKey.IDCmkKey,
		IDCmkKeyVersion: *idCmkKeyVersion,
	})
	return
}

func (r *Repository) EnableKey(secretCmkKey string) error {
	idCmkKey, _, err := utilities.GetSecretCmkKey(secretCmkKey)
	if err != nil {
		return err
	}

	return r.transitionKeyStatus(
		*idCmkKey,
		commonEntity.KeyStatusDisabled,
		commonEntity.KeyStatusEnabled,
		errMsgKeyMustBeDisabledToEnable,
	)
}

func (r *Repository) DisableKey(secretCmkKey string) error {
	idCmkKey, _, err := utilities.GetSecretCmkKey(secretCmkKey)
	if err != nil {
		return err
	}
	return r.transitionKeyStatus(
		*idCmkKey,
		commonEntity.KeyStatusEnabled,
		commonEntity.KeyStatusDisabled,
		errMsgKeyMustBeEnabledToDisable,
	)
}

const timeDay = 24 * time.Hour

func (r *Repository) ScheduleKeyDeletion(secretCmkKey string, interval uint) error {
	if interval == 0 {
		return errors.New("pending window days must be greater than zero")
	}
	if err := r.PendingDeletion(secretCmkKey); err != nil {
		return err
	}
	fn := func() { r.deleteScheduledKey(secretCmkKey) }
	jobs.Job(fn, time.Duration(interval)*timeDay, nil)
	return nil
}

func (r *Repository) deleteScheduledKey(secretCmkKey string) {
	ctxLogger := builder.New(r.ctx)
	idCmkKey, _, err := utilities.GetSecretCmkKey(secretCmkKey)
	if err != nil {
		ctxLogger.Error(err)
		return
	}

	cmkKey, err := r.getKey(*idCmkKey)
	if err != nil {
		ctxLogger.Error(err)
		return
	}
	if cmkKey.Status != commonEntity.KeyStatusPendingDeletion {
		ctxLogger.Info("ScheduleKeyDeletion skipped because key deletion is not pending")
		return
	}

	if err := r.DeleteKey(secretCmkKey); err != nil {
		ctxLogger.Error(err)
		return
	}
	ctxLogger.Info("ScheduleKeyDeletion was executed")
}

func (r *Repository) PendingDeletion(secretCmkKey string) error {
	idCmkKey, _, err := utilities.GetSecretCmkKey(secretCmkKey)
	if err != nil {
		return err
	}

	cmkKey, err := r.getKey(*idCmkKey)
	if err != nil {
		return err
	}

	cmkKeyVersion, err := r.getKeyVersion(cmkKey.IDCmkKeyVersion)
	if err != nil {
		return err
	}

	_, err = r.sp.RetireKeyVersion(cmkKeyVersion.IDCmkKeyVersion)
	if err != nil {
		return err
	}
	return r.transitionKeyStatus(
		*idCmkKey,
		commonEntity.KeyStatusDisabled,
		commonEntity.KeyStatusPendingDeletion,
		errMsgKeyMustBeDisabledToMarkDeletion,
	)
}

func (r *Repository) CancelKeyDeletion(secretCmkKey string) error {
	idCmkKey, _, err := utilities.GetSecretCmkKey(secretCmkKey)
	if err != nil {
		return err
	}

	return r.transitionKeyStatus(
		*idCmkKey,
		commonEntity.KeyStatusPendingDeletion,
		commonEntity.KeyStatusDisabled,
		"key must be pending deletion to cancel deletion",
	)
}

func (r *Repository) UnavailableDelete(secretCmkKey string) error {
	idCmkKey, _, err := utilities.GetSecretCmkKey(secretCmkKey)
	if err != nil {
		return err
	}

	return r.transitionKeyStatus(
		*idCmkKey,
		commonEntity.KeyStatusPendingDeletion,
		commonEntity.KeyStatusUnavailable,
		"key must be pending deletion to mark unavailable",
	)
}

func (r *Repository) transitionKeyStatus(
	idCmkKey uuid.UUID,
	required,
	next commonEntity.KeyStatus,
	errMsg string,
) error {
	cmkKey, err := r.getKey(idCmkKey)
	if err != nil {
		return err
	}
	if cmkKey.Status != required {
		return errors.New(errMsg)
	}
	return r.updateKeyStatus(idCmkKey, next)
}

func (r *Repository) updateKeyStatus(idCmkKey uuid.UUID, status commonEntity.KeyStatus) error {
	_, err := r.sp.UpdateCmkKey(store.UpdateCmkKeyInput{
		IDCmkKey: idCmkKey,
		Status:   &status,
	})
	return err
}

func (r *Repository) DeleteKey(secretCmkKey string) error {
	idCmkKey, _, err := utilities.GetSecretCmkKey(secretCmkKey)
	if err != nil {
		return err
	}

	cmkKey, err := r.getKey(*idCmkKey)
	if err != nil {
		return err
	}

	queue, err := r.getQueueData(&cmkKey.IDCmkKey, nil)
	if err != nil {
		return err
	}

	_, err = r.sp.DeleteRetiredKeyVersion(*cmkKey.IDCmkKeyVersion)
	if err != nil {
		return err
	}

	_, err = r.sp.DeleteKeyCreationQueue(queue.IDCmkKeyCreationQueue)
	if err != nil {
		return err
	}

	_, err = r.sp.DeleteCmkKey(*idCmkKey)
	return err
}
