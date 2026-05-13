package cmk

import (
	"encoding/base64"
	"errors"
	"fmt"

	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/store"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/views"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/google/uuid"
)

func (r *Repository) GetKeyVersionInfo(idCmkKeyVersion uuid.UUID) (*models.KeyVersionInfo, error) {
	version, err := r.getKeyVersion(&idCmkKeyVersion)
	if err != nil {
		return nil, err
	}

	cmkKey, err := r.getKey(version.IDCmkKey)
	if err != nil {
		return nil, err
	}

	return keyVersionInfoModel(cmkKey, version), nil
}

func (r *Repository) UpdateKeyVersionStatus(idCmkKeyVersion uuid.UUID, status commonEntity.KeyVersionStatus) (*models.KeyVersionInfo, error) {
	if err := validateUpdateKeyVersionStatus(status); err != nil {
		return nil, err
	}

	version, err := r.getKeyVersion(&idCmkKeyVersion)
	if err != nil {
		return nil, err
	}

	cmkKey, err := r.getKey(version.IDCmkKey)
	if err != nil {
		return nil, err
	}
	if isCurrentKeyVersion(cmkKey, idCmkKeyVersion) {
		return nil, errors.New(errMsgKeyVersionMainCannotBeUpdated)
	}

	_, err = r.sp.UpdateKeyVersionStatus(store.UpdateKeyVersionStatusInput{
		IDCmkKeyVersion: idCmkKeyVersion,
		Status:          status,
	})
	if err != nil {
		return nil, err
	}

	updated := *version
	updated.Status = status
	return keyVersionInfoModel(cmkKey, &updated), nil
}

func validateUpdateKeyVersionStatus(status commonEntity.KeyVersionStatus) error {
	if status == commonEntity.KeyVersionStatus("pendingDeletion") {
		return errors.New(errMsgKeyVersionStatusPendingDeletion)
	}
	switch status {
	case commonEntity.KeyVersionStatusEnabled,
		commonEntity.KeyVersionStatusDisabled,
		commonEntity.KeyVersionStatusRetired,
		commonEntity.KeyVersionStatusUnavailable:
		return nil
	default:
		return errors.New(errMsgKeyVersionStatusInvalid)
	}
}

func keyVersionInfoModel(cmkKey *views.CmkKeyView, version *views.CmkKeyVersionView) *models.KeyVersionInfo {
	if cmkKey == nil || version == nil {
		return nil
	}
	return &models.KeyVersionInfo{
		IDCmkKey:        version.IDCmkKey,
		IDCmkKeyVersion: version.IDCmkKeyVersion,
		SecretCmkKey:    encodeSecretCmkKey(version.IDCmkKey, version.IDCmkKeyVersion),
		VersionNumber:   version.VersionNumber,
		Size:            version.Size,
		PublicKey:       version.KID,
		Status:          version.Status,
		Algorithm:       cmkKey.Algorithm,
		Purpose:         cmkKey.Purpose,
		IsCurrent:       isCurrentKeyVersion(cmkKey, version.IDCmkKeyVersion),
	}
}

func encodeSecretCmkKey(idCmkKey uuid.UUID, idCmkKeyVersion uuid.UUID) string {
	secret := fmt.Sprintf("%s.%s", idCmkKey.String(), idCmkKeyVersion.String())
	return base64.StdEncoding.EncodeToString([]byte(secret))
}

func isCurrentKeyVersion(cmkKey *views.CmkKeyView, idCmkKeyVersion uuid.UUID) bool {
	return cmkKey != nil &&
		cmkKey.IDCmkKeyVersion != nil &&
		*cmkKey.IDCmkKeyVersion == idCmkKeyVersion
}
