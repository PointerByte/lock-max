package cmk

import (
	"errors"
	"fmt"

	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/views"
	"github.com/PointerByte/lock-max/dragon-cmk/service/utilities"
	"github.com/google/uuid"
)

func (r *Repository) getKey(IdCmkKey uuid.UUID) (key *views.CmkKeyView, _ error) {
	queue, err := r.getQueueData(nil, &IdCmkKey)
	if err != nil {
		return nil, err
	}

	if queue.Status != commonEntity.QueueStatusProcessed {
		return nil, errors.New(errMsgKeyNotProcessed)
	}

	cmkKey, err := r.views.QueryCmkKeyView("WHERE id_cmk_key = $1", IdCmkKey)
	if err != nil {
		return nil, err
	}
	if len(cmkKey) == 0 {
		return nil, errors.New(errMsgKeyNotFound)
	}
	return &cmkKey[0], nil
}

func (r *Repository) getQueueData(id ...*uuid.UUID) (idQueue *views.CmkCreationKeyQueueView, _ error) {
	var idCmkKey, idCmkKeyCreationQueue uuid.UUID
	if len(id) > 0 && id[0] != nil {
		idCmkKey = *id[0]
	}
	if len(id) > 1 && id[1] != nil {
		idCmkKeyCreationQueue = *id[1]
	}

	queues, err := r.views.QueryCmkCreationKeyQueueView(
		"WHERE id_cmk_key = $1 or id_cmk_key_creation_queue = $2", idCmkKey, idCmkKeyCreationQueue,
	)
	if err != nil {
		return nil, err
	}
	if len(queues) == 0 {
		return nil, fmt.Errorf("creation key queue not found for cmk key: %s%s", idCmkKey, idCmkKeyCreationQueue)
	}
	return &queues[0], nil
}

func (r *Repository) getKeyVersion(IDCmkKeyVersion *uuid.UUID) (*views.CmkKeyVersionView, error) {
	versions, err := r.views.QueryCmkKeyVersionView("WHERE id_cmk_key_version = $1", IDCmkKeyVersion)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, errors.New(errMsgKeyVersionNotFound)
	}
	return &versions[0], nil
}

func (r *Repository) getWrappingKeyRef(idCmkWrappingKeyRef uuid.UUID) (*views.CmkWrappingKeyRefView, error) {
	data, err := r.views.QueryCmkWrappingKeyRefView("WHERE id_cmk_wrapping_key_ref = $1", idCmkWrappingKeyRef)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New(errMsgWrappingKeyRefNotFound)
	}
	return &data[0], nil
}

func (r *Repository) loadKeyMaterial(secretCmkKey string) (*keyMaterial, error) {
	idCmkKey, idCmkKeyVersion, err := utilities.GetSecretCmkKey(secretCmkKey)
	if err != nil {
		return nil, err
	}

	cmkKey, err := r.getKey(*idCmkKey)
	if err != nil {
		return nil, err
	}

	cmkKeyVersion, err := r.getKeyVersion(idCmkKeyVersion)
	if err != nil {
		return nil, err
	}
	if cmkKeyVersion.Status != commonEntity.KeyVersionStatusEnabled {
		return nil, errors.New(errMsgKeyVersionMustBeEnabled)
	}

	cmkWrappingKeyRef, err := r.getWrappingKeyRef(*cmkKeyVersion.IDCmkWrappingKeyRef)
	if err != nil {
		return nil, err
	}

	key, _, validChecksum, err := r.unwrapKey(
		cmkKeyVersion.SecretWrapped,
		cmkKeyVersion.WrapAlg,
		cmkWrappingKeyRef.Version,
		cmkKeyVersion.SecretChecksum,
	)
	if err != nil {
		return nil, err
	}
	if !validChecksum {
		return nil, errors.New(errMsgInvalidChecksum)
	}

	return &keyMaterial{
		cmkKey:     cmkKey,
		keyVersion: cmkKeyVersion,
		key:        key,
		algorithm:  string(cmkKey.Algorithm),
	}, nil
}
