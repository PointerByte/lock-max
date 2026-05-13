package store

import (
	"context"

	"github.com/PointerByte/GoForge/logger/builder"
	"github.com/PointerByte/GoForge/logger/formatter"
	"github.com/PointerByte/lock-max/dragon-cmk/entity"
	postgresql "github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var callStoredProcedureFn = postgresql.CallStoredProcedure

func NewRepository(ctx context.Context, ctxLogger *builder.Context) IRepository {
	return &PostgreSQLRepository{
		ctx:       ctx,
		ctxLogger: ctxLogger,
	}
}

type PostgreSQLRepository struct {
	ctx       context.Context
	ctxLogger *builder.Context
}

func (r PostgreSQLRepository) CreateCmkKey(input CreateCmkKeyInput) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Register key in storage")
	process.Request = input
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_create_cmk_key",
		input.IDCmkKey,
		input.Algorithm,
		input.Purpose,
		input.Status,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) UpdateCmkKey(input UpdateCmkKeyInput) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Update key in storage")
	process.Request = input
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_update_cmk_key",
		input.IDCmkKey,
		input.Status,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) DeleteCmkKey(idCmkKey uuid.UUID) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Delete key from storage")
	process.Request = idCmkKey
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_delete_cmk_key",
		idCmkKey,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) CreateKeyCreationQueue(input CreateKeyCreationQueueInput) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Create key creation queue record")
	process.Request = input
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_create_creation_key_queue",
		input.IDCmkKeyCreationQueue,
		input.IDCmkKey,
		input.EventType,
		input.Status,
		input.ProcessedAt,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) UpdateKeyCreationQueue(input UpdateKeyCreationQueueInput) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Update key creation queue record")
	process.Request = input
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_update_creation_key_queue",
		input.IDCmkKeyCreationQueue,
		input.Status,
		input.ErrorMessage,
		input.ProcessedAt,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) DeleteKeyCreationQueue(idCmkKeyCreationQueue uuid.UUID) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Delete key creation queue record")
	process.Request = idCmkKeyCreationQueue
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_delete_creation_key_queue",
		idCmkKeyCreationQueue,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) CreateKeyVersion(input CreateKeyVersionInput) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Create key version")
	process.Request = input
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_create_key_version",
		input.IDCmkKeyVersion,
		input.IDCmkKey,
		input.VersionNumber,
		input.Size,
		input.Status,
		input.KID,
		input.SecretWrapped,
		input.WrapAlg,
		input.IDCmkWrappingKeyRef,
		input.Aditional,
		input.SecretChecksum,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) UpdateKeyVersionMetadata(input UpdateKeyVersionMetadataInput) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Update key version metadata")
	process.Request = input
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_update_key_version_metadata",
		input.IDCmkKeyVersion,
		input.IDCmkWrappingKeyRef,
		input.SecretWrapped,
		input.SecretChecksum,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) UpdateKeyVersionStatus(input UpdateKeyVersionStatusInput) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Update key version status")
	process.Request = input
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_update_key_version_status",
		input.IDCmkKeyVersion,
		input.Status,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) RetireKeyVersion(idCmkKeyVersion uuid.UUID) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Retire key version")
	process.Request = idCmkKeyVersion
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_retire_key_version",
		idCmkKeyVersion,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) DeleteRetiredKeyVersion(idCmkKeyVersion uuid.UUID) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Delete retired key version")
	process.Request = idCmkKeyVersion
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_delete_retired_key_version",
		idCmkKeyVersion,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) RotateKeyVersion(input RotateKeyVersionInput) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Rotate key version")
	process.Request = input
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_rotate_key_version",
		input.IDCmkKey,
		input.IDCmkKeyVersion,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) CreateWrappingKeyRef(input CreateWrappingKeyRefInput) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Create wrapping key reference")
	process.Request = input
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_create_wrapping_key_ref",
		input.IDCmkWrappingKeyRef,
		input.Provider,
		input.KeyRef,
		input.Version,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) UpdateWrappingKeyRef(input UpdateWrappingKeyRefInput) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Update wrapping key reference")
	process.Request = input
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_update_wrapping_key_ref",
		input.IDCmkWrappingKeyRef,
		input.Provider,
		input.KeyRef,
		input.Version,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) DeleteWrappingKeyRef(idCmkWrappingKeyRef uuid.UUID) (*pgconn.CommandTag, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Delete wrapping key reference")
	process.Request = idCmkWrappingKeyRef
	defer traceEnd(process)

	result, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_delete_wrapping_key_ref",
		idCmkWrappingKeyRef,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}
