package common

type EventType string

const (
	EventTypeCreateKey EventType = "create_key"
	EventTypeRotateKey EventType = "rotate_key"
)

type KeyPurpose string

const (
	KeyPurposeSign    KeyPurpose = "sign"
	KeyPurposeEncrypt KeyPurpose = "encrypt"
	KeyPurposeWrap    KeyPurpose = "wrap"
)

type KeyType string

const (
	KeySymmetricDefault      KeyType = "SYMMETRIC_DEFAULT"
	KeyTypeRSAOAEP           KeyType = "RSA_OAEP"
	KeyTypeRSAPKCS1v15SHA256 KeyType = "RSA_PKCS1v15_SHA256"
	KeyTypeECDH              KeyType = "ECDH"
	KeyTypeEdDSA             KeyType = "EdDSA"
)

type KeyStatus string

const (
	KeyStatusEnabled         KeyStatus = "enabled"
	KeyStatusDisabled        KeyStatus = "disabled"
	KeyStatusPendingDeletion KeyStatus = "pendingDeletion"
	KeyStatusPendingImport   KeyStatus = "pendingImport"
	KeyStatusUnavailable     KeyStatus = "Unavailable"
)

type KeyVersionStatus string

// Key version status values mirror dragon_cmk.key_version_status in the DB.
// Internal DB semantics:
//   - enabled: only versions with this status can be used for crypto operations.
//   - disabled: version exists but cannot be used.
//   - pendingDeletion: set internally when the parent cmk_key is pending deletion;
//     it is intentionally not exposed as an update option for a single version.
//   - retired: version is permanently retired and eligible for retired-version cleanup.
//   - Unavailable: version is unavailable because the parent cmk_key became unavailable.
const (
	KeyVersionStatusEnabled     KeyVersionStatus = "enabled"
	KeyVersionStatusDisabled    KeyVersionStatus = "disabled"
	KeyVersionStatusRetired     KeyVersionStatus = "retired"
	KeyVersionStatusUnavailable KeyVersionStatus = "Unavailable"
)

type QueueStatus string
 
const (
	QueueStatusPending    QueueStatus = "pending"
	QueueStatusProcessing QueueStatus = "processing"
	QueueStatusProcessed  QueueStatus = "processed"
	QueueStatusFailed     QueueStatus = "failed"
)
