package views

import (
	"time"

	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/google/uuid"
)

// CmkKeyView represents the projected data exposed by the vw_cmk_key database view.
type CmkKeyView struct {
	IDCmkKey        uuid.UUID               `db:"id_cmk_key" gorm:"column:id_cmk_key" json:"id_cmk_key"`
	Algorithm       commonEntity.KeyType    `db:"algorithm" gorm:"column:algorithm" json:"algorithm"`
	Purpose         commonEntity.KeyPurpose `db:"purpose" gorm:"column:purpose" json:"purpose"`
	Status          commonEntity.KeyStatus  `db:"status" gorm:"column:status" json:"status"`
	IDCmkKeyVersion *uuid.UUID              `db:"id_cmk_key_version" gorm:"column:id_cmk_key_version" json:"id_cmk_key_version,omitempty"`
	CreatedAt       time.Time               `db:"created_at" gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time               `db:"updated_at" gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns the fully qualified database view name used for CMK keys.
func (CmkKeyView) TableName() string {
	return "dragon_cmk.vw_cmk_key"
}

// CmkCreationKeyQueueView represents the projected data exposed by the
// vw_cmk_creation_key_queue database view.
type CmkCreationKeyQueueView struct {
	IDCmkKeyCreationQueue uuid.UUID                `db:"id_cmk_key_creation_queue" gorm:"column:id_cmk_key_creation_queue" json:"id_cmk_key_creation_queue"`
	IDCmkKey              uuid.UUID                `db:"id_cmk_key" gorm:"column:id_cmk_key" json:"id_cmk_key"`
	EventType             commonEntity.EventType   `db:"event_type" gorm:"column:event_type" json:"event_type"`
	Status                commonEntity.QueueStatus `db:"status" gorm:"column:status" json:"status"`
	ErrorMessage          *string                  `db:"error_message" gorm:"column:error_message" json:"error_message,omitempty"`
	QueuedAt              time.Time                `db:"queued_at" gorm:"column:queued_at" json:"queued_at"`
	ProcessedAt           *time.Time               `db:"processed_at" gorm:"column:processed_at" json:"processed_at,omitempty"`
}

// TableName returns the fully qualified database view name used for key
// creation queue records.
func (CmkCreationKeyQueueView) TableName() string {
	return "dragon_cmk.vw_cmk_creation_key_queue"
}

// CmkKeyCreationQueueView is kept for compatibility with the previous name.
type CmkKeyCreationQueueView = CmkCreationKeyQueueView

// CmkKeyVersionView represents the projected data exposed by the
// vw_cmk_key_version database view.
type CmkKeyVersionView struct {
	IDCmkKeyVersion     uuid.UUID                     `db:"id_cmk_key_version" gorm:"column:id_cmk_key_version" json:"id_cmk_key_version"`
	IDCmkKey            uuid.UUID                     `db:"id_cmk_key" gorm:"column:id_cmk_key" json:"id_cmk_key"`
	VersionNumber       int                           `db:"version_number" gorm:"column:version_number" json:"version_number"`
	Size                int                           `db:"size" gorm:"column:size" json:"size"`
	Status              commonEntity.KeyVersionStatus `db:"status" gorm:"column:status" json:"status"`
	KID                 string                        `db:"kid" gorm:"column:kid" json:"kid"`
	SecretWrapped       string                        `db:"secret_wrapped" gorm:"column:secret_wrapped" json:"secret_wrapped"`
	WrapAlg             string                        `db:"wrap_alg" gorm:"column:wrap_alg" json:"wrap_alg"`
	IDCmkWrappingKeyRef *uuid.UUID                    `db:"id_cmk_wrapping_key_ref" gorm:"column:id_cmk_wrapping_key_ref" json:"id_cmk_wrapping_key_ref,omitempty"`
	Aditional           *string                       `db:"aditional" gorm:"column:aditional" json:"aditional,omitempty"`
	CreatedAt           time.Time                     `db:"created_at" gorm:"column:created_at" json:"created_at"`
	ActivatedAt         *time.Time                    `db:"activated_at" gorm:"column:activated_at" json:"activated_at,omitempty"`
	DeactivatedAt       *time.Time                    `db:"deactivated_at" gorm:"column:deactivated_at" json:"deactivated_at,omitempty"`
	RetiredAt           *time.Time                    `db:"retired_at" gorm:"column:retired_at" json:"retired_at,omitempty"`
	SecretChecksum      *string                       `db:"secret_checksum" gorm:"column:secret_checksum" json:"secret_checksum,omitempty"`
}

// TableName returns the fully qualified database view name used for CMK
// versions.
func (CmkKeyVersionView) TableName() string {
	return "dragon_cmk.vw_cmk_key_version"
}

// CmkWrappingKeyRefView represents the projected data exposed by the
// vw_cmk_wrapping_key_ref database view.
type CmkWrappingKeyRefView struct {
	IDCmkWrappingKeyRef uuid.UUID `db:"id_cmk_wrapping_key_ref" gorm:"column:id_cmk_wrapping_key_ref" json:"id_cmk_wrapping_key_ref"`
	Provider            string    `db:"provider" gorm:"column:provider" json:"provider"`
	KeyRef              string    `db:"key_ref" gorm:"column:key_ref" json:"key_ref"`
	Version             string    `db:"version" gorm:"column:version" json:"version"`
	CreatedAt           time.Time `db:"created_at" gorm:"column:created_at" json:"created_at"`
}

// TableName returns the fully qualified database view name used for wrapping
// key references.
func (CmkWrappingKeyRefView) TableName() string {
	return "dragon_cmk.vw_cmk_wrapping_key_ref"
}
