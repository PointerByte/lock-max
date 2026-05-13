package views

import (
	"context"

	"github.com/PointerByte/GoForge/logger/builder"
	"github.com/PointerByte/GoForge/logger/formatter"
	"github.com/PointerByte/lock-max/dragon-cmk/entity"
	postgresql "github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL"
)

type Repository struct {
	ctx       context.Context
	ctxLogger *builder.Context
}

var (
	readCmkKeyViewFn               = postgresql.ReadView[CmkKeyView]
	readCmkCreationKeyQueueViewFn  = postgresql.ReadView[CmkCreationKeyQueueView]
	readCmkKeyVersionViewFn        = postgresql.ReadView[CmkKeyVersionView]
	readCmkWrappingKeyRefViewFn    = postgresql.ReadView[CmkWrappingKeyRefView]
	queryCmkKeyViewFn              = postgresql.QueryModelView[CmkKeyView]
	queryCmkCreationKeyQueueViewFn = postgresql.QueryModelView[CmkCreationKeyQueueView]
	countCmkCreationKeyQueueViewFn = postgresql.CountModelView[CmkCreationKeyQueueView]
	queryCmkKeyVersionViewFn       = postgresql.QueryModelView[CmkKeyVersionView]
	queryCmkWrappingKeyRefViewFn   = postgresql.QueryModelView[CmkWrappingKeyRefView]
)

func NewRepository(ctx context.Context, ctxLogger *builder.Context) IRepository {
	return &Repository{
		ctx:       ctx,
		ctxLogger: ctxLogger,
	}
}

func (r Repository) ReadCmkKeyView() ([]CmkKeyView, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Read CMK key view")
	process.Request = (CmkKeyView{}).TableName()
	defer traceEnd(process)

	result, err := readCmkKeyViewFn(r.ctx)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r Repository) QueryCmkKeyView(query string, args ...any) ([]CmkKeyView, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Query CMK key view")
	process.Request = map[string]any{"view": (CmkKeyView{}).TableName(), "query": query, "args": args}
	defer traceEnd(process)

	result, err := queryCmkKeyViewFn(r.ctx, query, args...)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r Repository) ReadCmkCreationKeyQueueView() ([]CmkCreationKeyQueueView, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Read CMK creation key queue view")
	process.Request = (CmkCreationKeyQueueView{}).TableName()
	defer traceEnd(process)

	result, err := readCmkCreationKeyQueueViewFn(r.ctx)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r Repository) QueryCmkCreationKeyQueueView(query string, args ...any) ([]CmkCreationKeyQueueView, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Query CMK creation key queue view")
	process.Request = map[string]any{"view": (CmkCreationKeyQueueView{}).TableName(), "query": query, "args": args}
	defer traceEnd(process)

	result, err := queryCmkCreationKeyQueueViewFn(r.ctx, query, args...)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r Repository) CountCmkCreationKeyQueueView(query string, args ...any) (uint, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Count CMK creation key queue view")
	process.Request = map[string]any{"view": (CmkCreationKeyQueueView{}).TableName(), "query": query, "args": args}
	defer traceEnd(process)

	result, err := countCmkCreationKeyQueueViewFn(r.ctx, query, args...)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return 0, err
	}

	process.Response = result
	return result, nil
}

func (r Repository) ReadCmkKeyCreationQueueView() ([]CmkKeyCreationQueueView, error) {
	return r.ReadCmkCreationKeyQueueView()
}

func (r Repository) QueryCmkKeyCreationQueueView(query string, args ...any) ([]CmkKeyCreationQueueView, error) {
	return r.QueryCmkCreationKeyQueueView(query, args...)
}

func (r Repository) ReadCmkKeyVersionView() ([]CmkKeyVersionView, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Read CMK key version view")
	process.Request = (CmkKeyVersionView{}).TableName()
	defer traceEnd(process)

	result, err := readCmkKeyVersionViewFn(r.ctx)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r Repository) QueryCmkKeyVersionView(query string, args ...any) ([]CmkKeyVersionView, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Query CMK key version view")
	process.Request = map[string]any{"view": (CmkKeyVersionView{}).TableName(), "query": query, "args": args}
	defer traceEnd(process)

	result, err := queryCmkKeyVersionViewFn(r.ctx, query, args...)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r Repository) ReadCmkWrappingKeyRefView() ([]CmkWrappingKeyRefView, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Read CMK wrapping key reference view")
	process.Request = (CmkWrappingKeyRefView{}).TableName()
	defer traceEnd(process)

	result, err := readCmkWrappingKeyRefViewFn(r.ctx)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}

func (r Repository) QueryCmkWrappingKeyRefView(query string, args ...any) ([]CmkWrappingKeyRefView, error) {
	process, traceEnd := entity.TraceClient(r.ctxLogger, "Query CMK wrapping key reference view")
	process.Request = map[string]any{"view": (CmkWrappingKeyRefView{}).TableName(), "query": query, "args": args}
	defer traceEnd(process)

	result, err := queryCmkWrappingKeyRefViewFn(r.ctx, query, args...)
	if err != nil {
		process.Status = formatter.ERROR
		process.Response = err.Error()
		return nil, err
	}

	process.Response = result
	return result, nil
}
