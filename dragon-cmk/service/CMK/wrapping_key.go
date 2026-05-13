package cmk

import (
	"errors"

	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/store"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL/views"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/google/uuid"
)

func (r *Repository) RotateWrapKey(IdCmkWrappingKeyRef uuid.UUID) error {
	wrappingKeyRef, err := r.getWrappingKeyRef(IdCmkWrappingKeyRef)
	if err != nil {
		return err
	}
	return r.rotateWrapKey(wrappingKeyRef)
}

func (r *Repository) RotateWrapKeyByKEK(kekData models.KEK) error {
	wrappingKeyRef, err := r.getWrappingKeyRefByKEK(&kekData)
	if err != nil {
		if err.Error() == errMsgWrappingKeyRefNotFound {
			return nil
		}
		return err
	}
	return r.rotateWrapKey(wrappingKeyRef)
}

func (r *Repository) rotateWrapKey(wrappingKeyRef *views.CmkWrappingKeyRefView) error {
	kekData, err := r.kek.GetKEK(uuid.Nil, "")
	if err != nil {
		return err
	}
	if sameWrappingKeyRef(wrappingKeyRef, kekData) {
		return nil
	}

	targetWrappingKeyRef, err := r.findWrappingKeyRefByKEK(kekData)
	if err != nil && err.Error() != errMsgWrappingKeyRefNotFound {
		return err
	}

	idCmkWrappingKeyRef := wrappingKeyRef.IDCmkWrappingKeyRef
	useExistingTarget := err == nil && targetWrappingKeyRef.IDCmkWrappingKeyRef != wrappingKeyRef.IDCmkWrappingKeyRef
	if useExistingTarget {
		idCmkWrappingKeyRef = targetWrappingKeyRef.IDCmkWrappingKeyRef
	}

	versions, err := r.views.QueryCmkKeyVersionView("WHERE id_cmk_wrapping_key_ref = $1", wrappingKeyRef.IDCmkWrappingKeyRef)
	if err != nil {
		return err
	}

	for _, version := range versions {
		key, _, validChecksum, err := r.unwrapKey(
			version.SecretWrapped,
			version.WrapAlg,
			wrappingKeyRef.Version,
			version.SecretChecksum,
		)
		if err != nil {
			return err
		}
		if !validChecksum {
			return errors.New(errMsgInvalidChecksum)
		}

		secretWrapped, _, secretChecksum, err := r.wrapKeyWithKEK(kekData, key)
		if err != nil {
			return err
		}

		_, err = r.sp.UpdateKeyVersionMetadata(store.UpdateKeyVersionMetadataInput{
			IDCmkKeyVersion:     version.IDCmkKeyVersion,
			IDCmkWrappingKeyRef: &idCmkWrappingKeyRef,
			SecretWrapped:       &secretWrapped,
			SecretChecksum:      secretChecksum,
		})
		if err != nil {
			return err
		}
	}

	if useExistingTarget {
		_, err = r.sp.DeleteWrappingKeyRef(wrappingKeyRef.IDCmkWrappingKeyRef)
		return err
	}

	provider := kekData.Provider
	keyRef := kekData.KeyRef
	version := kekData.Version
	_, err = r.sp.UpdateWrappingKeyRef(store.UpdateWrappingKeyRefInput{
		IDCmkWrappingKeyRef: wrappingKeyRef.IDCmkWrappingKeyRef,
		Provider:            &provider,
		KeyRef:              &keyRef,
		Version:             &version,
	})
	return err
}

func (r *Repository) getWrappingKeyRefByKEK(kekData *models.KEK) (*views.CmkWrappingKeyRefView, error) {
	wrappingKeyRef, err := r.findWrappingKeyRefByKEK(kekData)
	if err == nil {
		return wrappingKeyRef, nil
	}
	if err.Error() != errMsgWrappingKeyRefNotFound {
		return nil, err
	}

	wrappingKeyRef, err = r.getWrappingKeyRef(kekData.IdCmkWrappingKeyRef)
	if err != nil {
		return nil, err
	}
	if !sameWrappingKeyRef(wrappingKeyRef, kekData) {
		return nil, errors.New(errMsgWrappingKeyRefNotFound)
	}
	return wrappingKeyRef, nil
}

func (r *Repository) findWrappingKeyRefByKEK(kekData *models.KEK) (*views.CmkWrappingKeyRefView, error) {
	wrappingKeyRefs, err := r.views.QueryCmkWrappingKeyRefView(
		"WHERE provider = $1 AND key_ref = $2 AND version = $3",
		kekData.Provider,
		kekData.KeyRef,
		kekData.Version,
	)
	if err != nil {
		return nil, err
	}
	if len(wrappingKeyRefs) == 0 {
		return nil, errors.New(errMsgWrappingKeyRefNotFound)
	}
	return &wrappingKeyRefs[0], nil
}

func sameWrappingKeyRef(wrappingKeyRef *views.CmkWrappingKeyRefView, kekData *models.KEK) bool {
	if wrappingKeyRef == nil || kekData == nil {
		return false
	}
	return wrappingKeyRef.Provider == kekData.Provider &&
		wrappingKeyRef.KeyRef == kekData.KeyRef &&
		wrappingKeyRef.Version == kekData.Version
}

func (r *Repository) DeleteWrapKeyByKEK(kekData models.KEK) error {
	wrappingKeyRef, err := r.getWrappingKeyRefByKEK(&kekData)
	if err != nil {
		if err.Error() == errMsgWrappingKeyRefNotFound {
			return nil
		}
		return err
	}

	currentKEK, err := r.kek.GetKEK(uuid.Nil, "")
	if err != nil {
		return err
	}
	if sameWrappingKeyRef(wrappingKeyRef, currentKEK) {
		return nil
	}

	_, err = r.sp.DeleteWrappingKeyRef(wrappingKeyRef.IDCmkWrappingKeyRef)
	return err
}

func (r *Repository) wrapKey(key string) (secretWrapped, wrapAlg string, secretChecksum *string, _ error) {
	kekData, err := r.kek.GetKEK(uuid.Nil, "")
	if err != nil {
		return "", "", nil, err
	}
	return r.wrapKeyWithKEK(kekData, key)
}

func (r *Repository) wrapKeyWithKEK(kekData *models.KEK, key string) (secretWrapped, wrapAlg string, secretChecksum *string, err error) {
	ctx, cancel := r.contextWithTimeout()
	defer cancel()

	secretWrapped, err = r.security.ECC_Encode(ctx, kekData.PublicKey, key)
	if err != nil {
		return "", "", nil, err
	}
	wrapAlg = string(commonEntity.KeyTypeECDH)
	secretChecksum = new(string)
	*secretChecksum, err = r.security.SignEd25519(ctx, kekData.PrivateKey, string(commonEntity.KeyTypeECDH))
	return
}

func (r *Repository) unwrapKey(wrapKey, wrapAlg, version string, secretChecksum *string) (key, algorithm string, validChecksum bool, _ error) {
	kekData, err := r.kek.GetKEK(uuid.Nil, version)
	if err != nil {
		return "", "", false, err
	}

	ctx, cancel := r.contextWithTimeout()
	defer cancel()

	key, err = r.security.ECC_Decode(ctx, kekData.PrivateKey, wrapKey)
	if err != nil {
		return "", "", false, err
	}
	algorithm = wrapAlg
	if secretChecksum == nil {
		return key, algorithm, false, nil
	}
	err = r.security.VerifyEd25519(ctx, kekData.PublicKey, algorithm, *secretChecksum)
	if err == nil {
		validChecksum = true
	}
	return
}

func (r *Repository) saveWrappingKeyRef() error {
	kekData, err := r.kek.GetKEK(uuid.Nil, "")
	if err != nil {
		return err
	}

	_, err = r.ensureWrappingKeyRef(kekData)
	return err
}

func (r *Repository) ensureWrappingKeyRef(kekData *models.KEK) (*uuid.UUID, error) {
	_, err := r.getWrappingKeyRef(kekData.IdCmkWrappingKeyRef)
	if err != nil {
		if err.Error() != errMsgWrappingKeyRefNotFound {
			return nil, err
		}
	}

	wrappingKeyRefs, err := r.views.QueryCmkWrappingKeyRefView(
		"WHERE provider = $1 AND key_ref = $2 AND version = $3",
		kekData.Provider,
		kekData.KeyRef,
		kekData.Version,
	)
	if err != nil {
		return nil, err
	}
	if len(wrappingKeyRefs) > 0 {
		kekData.IdCmkWrappingKeyRef = wrappingKeyRefs[0].IDCmkWrappingKeyRef
		return &kekData.IdCmkWrappingKeyRef, nil
	}

	_, err = r.sp.CreateWrappingKeyRef(store.CreateWrappingKeyRefInput{
		IDCmkWrappingKeyRef: kekData.IdCmkWrappingKeyRef,
		KeyRef:              kekData.KeyRef,
		Provider:            kekData.Provider,
		Version:             kekData.Version,
	})
	if err != nil {
		return nil, err
	}
	return &kekData.IdCmkWrappingKeyRef, nil
}
