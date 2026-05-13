// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"

	"github.com/PointerByte/GoForge/logger/builder"
	"github.com/PointerByte/GoForge/logger/formatter"
	"github.com/PointerByte/lock-max/dragon-cmk/entity"
	postgresql "github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL"
)

const errMsgAPIClientNotFound = "api client not found"

var (
	callStoredProcedureFn = postgresql.CallStoredProcedure
	queryAPIClientFn      = postgresql.QueryModelView[APIClient]
	countAPIClientFn      = postgresql.CountModelView[APIClient]
)

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

func (r PostgreSQLRepository) CreateAPIClient(input CreateAPIClientInput) error {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Create API client")
	process.Request = input
	defer traceEnd(process)

	_, err := callStoredProcedureFn(
		r.ctx,
		"dragon_cmk.sp_create_api_client",
		input.IDAPIClient,
		input.ClientIDHash,
		input.ClientSecretHash,
		input.Description,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return err
	}

	process.Response = "created"
	return nil
}

func (r PostgreSQLRepository) GetAPIClientByClientIDHash(clientIDHash string) (*APIClient, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Get API client")
	process.Request = clientIDHash
	defer traceEnd(process)

	result, err := queryAPIClientFn(r.ctx, "WHERE client_id_hash = $1", clientIDHash)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}
	if len(result) == 0 {
		process.Status = formatter.ERROR
		process.Response = errMsgAPIClientNotFound
		return nil, errors.New(errMsgAPIClientNotFound)
	}

	process.Response = result[0]
	return &result[0], nil
}

func (r PostgreSQLRepository) ListAPIClients(page uint, totalRegisterPage uint) ([]APIClient, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "List API clients")
	process.Request = map[string]uint{"page": page, "totalRegisterPage": totalRegisterPage}
	defer traceEnd(process)

	offset := (page - 1) * totalRegisterPage
	result, err := queryAPIClientFn(
		r.ctx,
		"ORDER BY created_at DESC, id_api_client DESC LIMIT $1 OFFSET $2",
		totalRegisterPage,
		offset,
	)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) CountAPIClients() (uint, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Count API clients")
	defer traceEnd(process)

	result, err := countAPIClientFn(r.ctx, "")
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return 0, err
	}

	process.Response = result
	return result, nil
}

func (r PostgreSQLRepository) DeleteAPIClient(clientIDHash string) error {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Delete API client")
	process.Request = clientIDHash
	defer traceEnd(process)

	_, err := callStoredProcedureFn(r.ctx, "dragon_cmk.sp_delete_api_client", clientIDHash)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return err
	}

	process.Response = "deleted"
	return nil
}
