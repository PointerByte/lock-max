package views

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/PointerByte/GoForge/logger/builder"
	viperdata "github.com/PointerByte/GoForge/logger/viperData"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func TestPostgreSQLRepositoryViews(t *testing.T) {
	setupViewsLoggerTest(t)

	keyID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	t.Run("ReadCmkKeyView success and error", func(t *testing.T) {
		ctx := context.Background()
		ctxLogger := builder.New(ctx)
		repo := Repository{ctx: ctx, ctxLogger: ctxLogger}
		expected := []CmkKeyView{{IDCmkKey: keyID}}

		original := readCmkKeyViewFn
		t.Cleanup(func() {
			readCmkKeyViewFn = original
		})

		readCmkKeyViewFn = func(context.Context) ([]CmkKeyView, error) {
			return expected, nil
		}

		got, err := repo.ReadCmkKeyView()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected result: %#v", got)
		}

		readCmkKeyViewFn = func(context.Context) ([]CmkKeyView, error) {
			return nil, errors.New("forced failure")
		}

		got, err = repo.ReadCmkKeyView()
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Fatalf("expected nil result, got %#v", got)
		}
	})

	t.Run("QueryCmkKeyView success and error", func(t *testing.T) {
		ctx := context.Background()
		ctxLogger := builder.New(ctx)
		repo := Repository{ctx: ctx, ctxLogger: ctxLogger}
		expected := []CmkKeyView{{IDCmkKey: keyID}}

		original := queryCmkKeyViewFn
		t.Cleanup(func() {
			queryCmkKeyViewFn = original
		})

		queryCmkKeyViewFn = func(_ context.Context, query string, args ...any) ([]CmkKeyView, error) {
			if query != "WHERE id_cmk_key = $1" || len(args) != 1 || args[0] != keyID {
				t.Fatalf("unexpected query call: query=%q args=%#v", query, args)
			}
			return expected, nil
		}

		got, err := repo.QueryCmkKeyView("WHERE id_cmk_key = $1", keyID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected result: %#v", got)
		}

		queryCmkKeyViewFn = func(context.Context, string, ...any) ([]CmkKeyView, error) {
			return nil, errors.New("forced failure")
		}

		got, err = repo.QueryCmkKeyView("WHERE id_cmk_key = $1", keyID)
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Fatalf("expected nil result, got %#v", got)
		}
	})

	t.Run("ReadCmkCreationKeyQueueView success and error", func(t *testing.T) {
		ctx := context.Background()
		ctxLogger := builder.New(ctx)
		repo := Repository{ctx: ctx, ctxLogger: ctxLogger}
		expected := []CmkCreationKeyQueueView{{IDCmkKeyCreationQueue: keyID}}

		original := readCmkCreationKeyQueueViewFn
		t.Cleanup(func() {
			readCmkCreationKeyQueueViewFn = original
		})

		readCmkCreationKeyQueueViewFn = func(context.Context) ([]CmkCreationKeyQueueView, error) {
			return expected, nil
		}

		got, err := repo.ReadCmkCreationKeyQueueView()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected result: %#v", got)
		}

		readCmkCreationKeyQueueViewFn = func(context.Context) ([]CmkCreationKeyQueueView, error) {
			return nil, errors.New("forced failure")
		}

		got, err = repo.ReadCmkCreationKeyQueueView()
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Fatalf("expected nil result, got %#v", got)
		}
	})

	t.Run("QueryCmkCreationKeyQueueView success and error", func(t *testing.T) {
		ctx := context.Background()
		ctxLogger := builder.New(ctx)
		repo := Repository{ctx: ctx, ctxLogger: ctxLogger}
		expected := []CmkCreationKeyQueueView{{IDCmkKeyCreationQueue: keyID}}

		original := queryCmkCreationKeyQueueViewFn
		t.Cleanup(func() {
			queryCmkCreationKeyQueueViewFn = original
		})

		queryCmkCreationKeyQueueViewFn = func(_ context.Context, query string, args ...any) ([]CmkCreationKeyQueueView, error) {
			if query != "WHERE status = $1" || len(args) != 1 || args[0] != "pending" {
				t.Fatalf("unexpected query call: query=%q args=%#v", query, args)
			}
			return expected, nil
		}

		got, err := repo.QueryCmkCreationKeyQueueView("WHERE status = $1", "pending")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected result: %#v", got)
		}

		queryCmkCreationKeyQueueViewFn = func(context.Context, string, ...any) ([]CmkCreationKeyQueueView, error) {
			return nil, errors.New("forced failure")
		}

		got, err = repo.QueryCmkCreationKeyQueueView("WHERE status = $1", "pending")
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Fatalf("expected nil result, got %#v", got)
		}
	})

	t.Run("CountCmkCreationKeyQueueView success and error", func(t *testing.T) {
		ctx := context.Background()
		ctxLogger := builder.New(ctx)
		repo := Repository{ctx: ctx, ctxLogger: ctxLogger}

		original := countCmkCreationKeyQueueViewFn
		t.Cleanup(func() {
			countCmkCreationKeyQueueViewFn = original
		})

		countCmkCreationKeyQueueViewFn = func(_ context.Context, query string, args ...any) (uint, error) {
			if query != "WHERE status = $1" || len(args) != 1 || args[0] != "pending" {
				t.Fatalf("unexpected count call: query=%q args=%#v", query, args)
			}
			return 3, nil
		}

		got, err := repo.CountCmkCreationKeyQueueView("WHERE status = $1", "pending")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 3 {
			t.Fatalf("unexpected count: %d", got)
		}

		countCmkCreationKeyQueueViewFn = func(context.Context, string, ...any) (uint, error) {
			return 0, errors.New("forced failure")
		}

		got, err = repo.CountCmkCreationKeyQueueView("WHERE status = $1", "pending")
		if err == nil {
			t.Fatal("expected error")
		}
		if got != 0 {
			t.Fatalf("expected zero count, got %d", got)
		}
	})

	t.Run("ReadCmkKeyVersionView success and error", func(t *testing.T) {
		ctx := context.Background()
		ctxLogger := builder.New(ctx)
		repo := Repository{ctx: ctx, ctxLogger: ctxLogger}
		expected := []CmkKeyVersionView{{IDCmkKeyVersion: keyID, KID: "kid-1"}}

		original := readCmkKeyVersionViewFn
		t.Cleanup(func() {
			readCmkKeyVersionViewFn = original
		})

		readCmkKeyVersionViewFn = func(context.Context) ([]CmkKeyVersionView, error) {
			return expected, nil
		}

		got, err := repo.ReadCmkKeyVersionView()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected result: %#v", got)
		}

		readCmkKeyVersionViewFn = func(context.Context) ([]CmkKeyVersionView, error) {
			return nil, errors.New("forced failure")
		}

		got, err = repo.ReadCmkKeyVersionView()
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Fatalf("expected nil result, got %#v", got)
		}
	})

	t.Run("QueryCmkKeyVersionView success and error", func(t *testing.T) {
		ctx := context.Background()
		ctxLogger := builder.New(ctx)
		repo := Repository{ctx: ctx, ctxLogger: ctxLogger}
		expected := []CmkKeyVersionView{{IDCmkKeyVersion: keyID, KID: "kid-1"}}

		original := queryCmkKeyVersionViewFn
		t.Cleanup(func() {
			queryCmkKeyVersionViewFn = original
		})

		queryCmkKeyVersionViewFn = func(_ context.Context, query string, args ...any) ([]CmkKeyVersionView, error) {
			if query != "WHERE kid = $1" || len(args) != 1 || args[0] != "kid-1" {
				t.Fatalf("unexpected query call: query=%q args=%#v", query, args)
			}
			return expected, nil
		}

		got, err := repo.QueryCmkKeyVersionView("WHERE kid = $1", "kid-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected result: %#v", got)
		}

		queryCmkKeyVersionViewFn = func(context.Context, string, ...any) ([]CmkKeyVersionView, error) {
			return nil, errors.New("forced failure")
		}

		got, err = repo.QueryCmkKeyVersionView("WHERE kid = $1", "kid-1")
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Fatalf("expected nil result, got %#v", got)
		}
	})

	t.Run("ReadCmkWrappingKeyRefView success and error", func(t *testing.T) {
		ctx := context.Background()
		ctxLogger := builder.New(ctx)
		repo := Repository{ctx: ctx, ctxLogger: ctxLogger}
		expected := []CmkWrappingKeyRefView{{IDCmkWrappingKeyRef: keyID, Provider: "aws", Version: "v1"}}

		original := readCmkWrappingKeyRefViewFn
		t.Cleanup(func() {
			readCmkWrappingKeyRefViewFn = original
		})

		readCmkWrappingKeyRefViewFn = func(context.Context) ([]CmkWrappingKeyRefView, error) {
			return expected, nil
		}

		got, err := repo.ReadCmkWrappingKeyRefView()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected result: %#v", got)
		}

		readCmkWrappingKeyRefViewFn = func(context.Context) ([]CmkWrappingKeyRefView, error) {
			return nil, errors.New("forced failure")
		}

		got, err = repo.ReadCmkWrappingKeyRefView()
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Fatalf("expected nil result, got %#v", got)
		}
	})

	t.Run("QueryCmkWrappingKeyRefView success and error", func(t *testing.T) {
		ctx := context.Background()
		ctxLogger := builder.New(ctx)
		repo := Repository{ctx: ctx, ctxLogger: ctxLogger}
		expected := []CmkWrappingKeyRefView{{IDCmkWrappingKeyRef: keyID, Provider: "aws", Version: "v1"}}

		original := queryCmkWrappingKeyRefViewFn
		t.Cleanup(func() {
			queryCmkWrappingKeyRefViewFn = original
		})

		queryCmkWrappingKeyRefViewFn = func(_ context.Context, query string, args ...any) ([]CmkWrappingKeyRefView, error) {
			if query != "WHERE provider = $1" || len(args) != 1 || args[0] != "aws" {
				t.Fatalf("unexpected query call: query=%q args=%#v", query, args)
			}
			return expected, nil
		}

		got, err := repo.QueryCmkWrappingKeyRefView("WHERE provider = $1", "aws")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected result: %#v", got)
		}

		queryCmkWrappingKeyRefViewFn = func(context.Context, string, ...any) ([]CmkWrappingKeyRefView, error) {
			return nil, errors.New("forced failure")
		}

		got, err = repo.QueryCmkWrappingKeyRefView("WHERE provider = $1", "aws")
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Fatalf("expected nil result, got %#v", got)
		}
	})
}

func TestNewRepository(t *testing.T) {
	ctx := context.Background()
	ctxLogger := builder.New(ctx)
	repository := NewRepository(ctx, ctxLogger)
	if repository == nil {
		t.Fatal("expected repository instance")
	}

	postgreSQLRepository, ok := repository.(*Repository)
	if !ok {
		t.Fatalf("unexpected repository type: %T", repository)
	}
	if postgreSQLRepository.ctx != ctx {
		t.Fatal("expected repository context to be preserved")
	}
	if postgreSQLRepository.ctxLogger != ctxLogger {
		t.Fatal("expected repository logger context to be preserved")
	}
}

func setupViewsLoggerTest(t *testing.T) {
	t.Helper()

	viper.Reset()
	viperdata.ResetViperDataSingleton()
	viper.Set(string(viperdata.AppAtribute), "dragon-cmk")
	builder.EnableModeTest()

	t.Cleanup(func() {
		viper.Reset()
		viperdata.ResetViperDataSingleton()
	})
}
