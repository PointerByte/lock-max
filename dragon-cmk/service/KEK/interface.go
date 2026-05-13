package kek

import (
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/google/uuid"
)

type IRepository interface {
	SetFuncCreateKey(fn HandlerFuncCreteKey)
	SetFuncRotate(fn HandlerFuncRotateKey)
	SetFuncRotateWrapKey(fn HandlerFuncRotateWrapKey)
	SetFuncDeleteWrapKey(fn HandlerFuncDeleteWrapKey)

	CreateKEK(idGenerate uuid.UUID, secretCmkKey string, salt string) (*uuid.UUID, error)
	GetKEK(id uuid.UUID, version string) (*models.KEK, error)
	RotateKEK(id uuid.UUID, salt string) (*uuid.UUID, error)
	ListKEK(id uuid.UUID, page uint, totalRegisterPage uint) (*models.PaginatedKEK, error)
	DeleteKey(id uuid.UUID, version string) error
}
