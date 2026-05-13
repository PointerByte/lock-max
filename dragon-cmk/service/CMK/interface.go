package cmk

import (
	"context"

	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/google/uuid"
)

//go:generate mockgen -source=interface.go -destination=mock_repository.go -package=cmk

type IRepository interface {
	CreateKey(input models.CreateKeyInput, eventType commonEntity.EventType) (string, error)
	RotateKey(secretCmkKey string) (string, error)
	RotateWrapKey(IdCmkWrappingKeyRef uuid.UUID) error
	Status() (*models.StatusResponse, error)
	GetKeyVersionInfo(idCmkKeyVersion uuid.UUID) (*models.KeyVersionInfo, error)
	UpdateKeyVersionStatus(idCmkKeyVersion uuid.UUID, status commonEntity.KeyVersionStatus) (*models.KeyVersionInfo, error)
	ListCmkKey(idKek uuid.UUID, page uint, totalRegisterPage uint) (*models.PaginatedCmkKey, error)
	ListCreationKeyQueues(idCmkKey uuid.UUID, status commonEntity.QueueStatus, page uint, totalRegisterPage uint) (*models.PaginatedCreationKeyQueue, error)

	Encrypt(secretCmkKey string, plaintext string, additional *string) (ciphertext string, annotations map[string][]byte, err error)
	Decrypt(secretCmkKey string, ciphertext string, additional *string) (plaintext string, annotations map[string][]byte, err error)
	CreateJWT(ctx context.Context, secretCmkKey string, algorithm string, claims any) (string, error)
	VerifyJWT(ctx context.Context, secretCmkKey string, algorithm string, token string) error
	ReadJWT(ctx context.Context, secretCmkKey string, algorithm string, token string, claims any) error
	Sing(secretCmkKey string, message string) (signature string, err error)
	Verify(secretCmkKey string, message, signature string) (valid bool, err error)

	DisableKey(secretCmkKey string) error
	EnableKey(secretCmkKey string) error
	ScheduleKeyDeletion(secretCmkKey string, interval uint) error
	PendingDeletion(secretCmkKey string) error
	CancelKeyDeletion(secretCmkKey string) error
	UnavailableDelete(secretCmkKey string) error
	DeleteKey(secretCmkKey string) error
}
