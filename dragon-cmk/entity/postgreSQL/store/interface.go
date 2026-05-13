package store

//go:generate mockgen -source=interface.go -destination=mock_repository.go -package=store

import (
	"time"

	"github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// IRepository defines write/command operations backed by stored procedures
// declared under db/storeProrcedures.
type IRepository interface {
	// Uses db/storeProrcedures/cmk_key/sp_create_cmk_key.sql
	CreateCmkKey(input CreateCmkKeyInput) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_key/sp_update_cmk_key.sql
	UpdateCmkKey(input UpdateCmkKeyInput) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_key/sp_delete_cmk_key.sql
	DeleteCmkKey(idCmkKey uuid.UUID) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_creation_key_queue/sp_create_creation_key_queue.sql
	CreateKeyCreationQueue(input CreateKeyCreationQueueInput) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_creation_key_queue/sp_update_creation_key_queue.sql
	UpdateKeyCreationQueue(input UpdateKeyCreationQueueInput) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_creation_key_queue/sp_delete_creation_key_queue.sql
	DeleteKeyCreationQueue(idCmkKeyCreationQueue uuid.UUID) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_key_version/sp_create_key_version.sql
	CreateKeyVersion(input CreateKeyVersionInput) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_key_version/sp_update_key_version_metadata.sql
	UpdateKeyVersionMetadata(input UpdateKeyVersionMetadataInput) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_key_version/sp_update_key_version_status.sql
	UpdateKeyVersionStatus(input UpdateKeyVersionStatusInput) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_key_version/sp_retire_key_version.sql
	RetireKeyVersion(idCmkKeyVersion uuid.UUID) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_key_version/sp_delete_retired_key_version.sql
	DeleteRetiredKeyVersion(idCmkKeyVersion uuid.UUID) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/sp_rotate_key_version.sql
	RotateKeyVersion(input RotateKeyVersionInput) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_wrapping_key_ref/sp_create_wrapping_key_ref.sql
	CreateWrappingKeyRef(input CreateWrappingKeyRefInput) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_wrapping_key_ref/sp_update_wrapping_key_ref.sql
	UpdateWrappingKeyRef(input UpdateWrappingKeyRefInput) (*pgconn.CommandTag, error)

	// Uses db/storeProrcedures/cmk_wrapping_key_ref/sp_delete_wrapping_key_ref.sql
	DeleteWrappingKeyRef(idCmkWrappingKeyRef uuid.UUID) (*pgconn.CommandTag, error)
}
type CreateCmkKeyInput struct {
	IDCmkKey  uuid.UUID
	Algorithm common.KeyType
	Purpose   common.KeyPurpose
	Status    *common.KeyStatus
}

type UpdateCmkKeyInput struct {
	IDCmkKey uuid.UUID
	Status   *common.KeyStatus
}

type CreateKeyCreationQueueInput struct {
	IDCmkKeyCreationQueue uuid.UUID
	IDCmkKey              uuid.UUID
	EventType             common.EventType
	Status                *common.QueueStatus
	ProcessedAt           *time.Time
}

type UpdateKeyCreationQueueInput struct {
	IDCmkKeyCreationQueue uuid.UUID
	Status                *common.QueueStatus
	ErrorMessage          *string
	ProcessedAt           *time.Time
}

type CreateKeyVersionInput struct {
	IDCmkKeyVersion     uuid.UUID
	IDCmkKey            uuid.UUID
	VersionNumber       int
	Size                int
	Status              *common.KeyVersionStatus
	KID                 string
	SecretWrapped       string
	WrapAlg             string
	IDCmkWrappingKeyRef *uuid.UUID
	Aditional           *string
	SecretChecksum      *string
}

type UpdateKeyVersionMetadataInput struct {
	IDCmkKeyVersion     uuid.UUID
	IDCmkWrappingKeyRef *uuid.UUID
	SecretWrapped       *string
	SecretChecksum      *string
}

type UpdateKeyVersionStatusInput struct {
	IDCmkKeyVersion uuid.UUID
	Status          common.KeyVersionStatus
}

type RotateKeyVersionInput struct {
	IDCmkKey        uuid.UUID
	IDCmkKeyVersion uuid.UUID
}

type CreateWrappingKeyRefInput struct {
	IDCmkWrappingKeyRef uuid.UUID
	Provider            string
	KeyRef              string
	Version             string
}

type UpdateWrappingKeyRefInput struct {
	IDCmkWrappingKeyRef uuid.UUID
	Provider            *string
	KeyRef              *string
	Version             *string
}
