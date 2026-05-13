package store

import (
	"errors"
	"testing"
	"time"

	"github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMockRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	status := common.KeyStatusEnabled
	keyVersionStatus := common.KeyVersionStatusDisabled
	queueStatus := common.QueueStatusPending
	processedQueueStatus := common.QueueStatusProcessed
	errorMessage := "boom"
	provider := "aws"
	keyRef := "key-ref"
	version := "v1"
	aditional := "extra-data"
	updatedSecretWrapped := "wrapped-2"
	secretChecksum := "checksum"
	updatedKeyVersionStatus := common.KeyVersionStatusUnavailable
	now := time.Now().UTC()
	id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	callTag := pgconn.NewCommandTag("CALL")
	mockErr := errors.New("mock error")

	repo.EXPECT().CreateCmkKey(CreateCmkKeyInput{
		IDCmkKey:  id1,
		Algorithm: common.KeyTypeRSAOAEP,
		Purpose:   common.KeyPurposeSign,
		Status:    &status,
	}).Return(&callTag, nil)

	repo.EXPECT().UpdateCmkKey(UpdateCmkKeyInput{
		IDCmkKey: id1,
		Status:   &status,
	}).Return(nil, mockErr)

	repo.EXPECT().DeleteCmkKey(id1).Return(&callTag, nil)

	repo.EXPECT().CreateKeyCreationQueue(CreateKeyCreationQueueInput{
		IDCmkKeyCreationQueue: id1,
		IDCmkKey:              id2,
		EventType:             common.EventTypeCreateKey,
		Status:                &queueStatus,
		ProcessedAt:           &now,
	}).Return(&callTag, nil)

	repo.EXPECT().UpdateKeyCreationQueue(UpdateKeyCreationQueueInput{
		IDCmkKeyCreationQueue: id1,
		Status:                &processedQueueStatus,
		ErrorMessage:          &errorMessage,
		ProcessedAt:           &now,
	}).Return(&callTag, nil)

	repo.EXPECT().DeleteKeyCreationQueue(id1).Return(&callTag, nil)

	repo.EXPECT().CreateKeyVersion(CreateKeyVersionInput{
		IDCmkKeyVersion:     id1,
		IDCmkKey:            id2,
		VersionNumber:       1,
		Size:                256,
		Status:              &keyVersionStatus,
		KID:                 "kid-1",
		SecretWrapped:       "wrapped",
		WrapAlg:             "RSA-OAEP",
		IDCmkWrappingKeyRef: &id2,
		Aditional:           &aditional,
		SecretChecksum:      &secretChecksum,
	}).Return(&callTag, nil)

	repo.EXPECT().UpdateKeyVersionMetadata(UpdateKeyVersionMetadataInput{
		IDCmkKeyVersion:     id1,
		IDCmkWrappingKeyRef: &id2,
		SecretWrapped:       &updatedSecretWrapped,
		SecretChecksum:      &secretChecksum,
	}).Return(&callTag, nil)

	repo.EXPECT().UpdateKeyVersionStatus(UpdateKeyVersionStatusInput{
		IDCmkKeyVersion: id1,
		Status:          updatedKeyVersionStatus,
	}).Return(&callTag, nil)

	repo.EXPECT().RetireKeyVersion(id1).Return(&callTag, nil)
	repo.EXPECT().DeleteRetiredKeyVersion(id1).Return(&callTag, nil)

	repo.EXPECT().RotateKeyVersion(RotateKeyVersionInput{
		IDCmkKey:        id1,
		IDCmkKeyVersion: id2,
	}).Return(&callTag, nil)

	repo.EXPECT().CreateWrappingKeyRef(CreateWrappingKeyRefInput{
		IDCmkWrappingKeyRef: id1,
		Provider:            provider,
		KeyRef:              keyRef,
		Version:             version,
	}).Return(&callTag, nil)

	repo.EXPECT().UpdateWrappingKeyRef(UpdateWrappingKeyRefInput{
		IDCmkWrappingKeyRef: id1,
		Provider:            &provider,
		KeyRef:              &keyRef,
		Version:             &version,
	}).Return(&callTag, nil)
	repo.EXPECT().DeleteWrappingKeyRef(id1).Return(&callTag, nil)

	if got, err := repo.CreateCmkKey(CreateCmkKeyInput{
		IDCmkKey:  id1,
		Algorithm: common.KeyTypeRSAOAEP,
		Purpose:   common.KeyPurposeSign,
		Status:    &status,
	}); err != nil || got == nil || got.String() != callTag.String() {
		t.Fatalf("CreateCmkKey() = (%v, %v)", got, err)
	}

	if got, err := repo.UpdateCmkKey(UpdateCmkKeyInput{
		IDCmkKey: id1,
		Status:   &status,
	}); !errors.Is(err, mockErr) || got != nil {
		t.Fatalf("UpdateCmkKey() = (%v, %v)", got, err)
	}

	if got, err := repo.DeleteCmkKey(id1); err != nil || got == nil {
		t.Fatalf("DeleteCmkKey() = (%v, %v)", got, err)
	}

	if got, err := repo.CreateKeyCreationQueue(CreateKeyCreationQueueInput{
		IDCmkKeyCreationQueue: id1,
		IDCmkKey:              id2,
		EventType:             common.EventTypeCreateKey,
		Status:                &queueStatus,
		ProcessedAt:           &now,
	}); err != nil || got == nil {
		t.Fatalf("CreateKeyCreationQueue() = (%v, %v)", got, err)
	}

	if got, err := repo.UpdateKeyCreationQueue(UpdateKeyCreationQueueInput{
		IDCmkKeyCreationQueue: id1,
		Status:                &processedQueueStatus,
		ErrorMessage:          &errorMessage,
		ProcessedAt:           &now,
	}); err != nil || got == nil {
		t.Fatalf("UpdateKeyCreationQueue() = (%v, %v)", got, err)
	}

	if got, err := repo.DeleteKeyCreationQueue(id1); err != nil || got == nil {
		t.Fatalf("DeleteKeyCreationQueue() = (%v, %v)", got, err)
	}

	if got, err := repo.CreateKeyVersion(CreateKeyVersionInput{
		IDCmkKeyVersion:     id1,
		IDCmkKey:            id2,
		VersionNumber:       1,
		Size:                256,
		Status:              &keyVersionStatus,
		KID:                 "kid-1",
		SecretWrapped:       "wrapped",
		WrapAlg:             "RSA-OAEP",
		IDCmkWrappingKeyRef: &id2,
		Aditional:           &aditional,
		SecretChecksum:      &secretChecksum,
	}); err != nil || got == nil {
		t.Fatalf("CreateKeyVersion() = (%v, %v)", got, err)
	}

	if got, err := repo.UpdateKeyVersionMetadata(UpdateKeyVersionMetadataInput{
		IDCmkKeyVersion:     id1,
		IDCmkWrappingKeyRef: &id2,
		SecretWrapped:       &updatedSecretWrapped,
		SecretChecksum:      &secretChecksum,
	}); err != nil || got == nil {
		t.Fatalf("UpdateKeyVersionMetadata() = (%v, %v)", got, err)
	}

	if got, err := repo.UpdateKeyVersionStatus(UpdateKeyVersionStatusInput{
		IDCmkKeyVersion: id1,
		Status:          updatedKeyVersionStatus,
	}); err != nil || got == nil {
		t.Fatalf("UpdateKeyVersionStatus() = (%v, %v)", got, err)
	}

	if got, err := repo.RetireKeyVersion(id1); err != nil || got == nil {
		t.Fatalf("RetireKeyVersion() = (%v, %v)", got, err)
	}

	if got, err := repo.DeleteRetiredKeyVersion(id1); err != nil || got == nil {
		t.Fatalf("DeleteRetiredKeyVersion() = (%v, %v)", got, err)
	}

	if got, err := repo.RotateKeyVersion(RotateKeyVersionInput{
		IDCmkKey:        id1,
		IDCmkKeyVersion: id2,
	}); err != nil || got == nil {
		t.Fatalf("RotateKeyVersion() = (%v, %v)", got, err)
	}

	if got, err := repo.CreateWrappingKeyRef(CreateWrappingKeyRefInput{
		IDCmkWrappingKeyRef: id1,
		Provider:            provider,
		KeyRef:              keyRef,
		Version:             version,
	}); err != nil || got == nil {
		t.Fatalf("CreateWrappingKeyRef() = (%v, %v)", got, err)
	}

	if got, err := repo.UpdateWrappingKeyRef(UpdateWrappingKeyRefInput{
		IDCmkWrappingKeyRef: id1,
		Provider:            &provider,
		KeyRef:              &keyRef,
		Version:             &version,
	}); err != nil || got == nil {
		t.Fatalf("UpdateWrappingKeyRef() = (%v, %v)", got, err)
	}

	if got, err := repo.DeleteWrappingKeyRef(id1); err != nil || got == nil {
		t.Fatalf("DeleteWrappingKeyRef() = (%v, %v)", got, err)
	}
}
