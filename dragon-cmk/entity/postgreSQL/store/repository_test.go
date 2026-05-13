package store

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/PointerByte/GoForge/logger/builder"
	viperdata "github.com/PointerByte/GoForge/logger/viperData"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/spf13/viper"
)

func TestPostgreSQLRepositoryStoredProcedures(t *testing.T) {
	setupStoreLoggerTest(t)

	status := common.KeyStatusEnabled
	keyVersionStatus := common.KeyVersionStatusDisabled
	queueStatus := common.QueueStatusPending
	processedQueueStatus := common.QueueStatusProcessed
	updatedKeyVersionStatus := common.KeyVersionStatusUnavailable
	errorMessage := "boom"
	provider := "aws"
	keyRef := "key-ref"
	version := "v1"
	aditional := "extra-data"
	updatedSecretWrapped := "wrapped-2"
	secretChecksum := "checksum"
	now := time.Now().UTC()
	id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name         string
		call         func(repo PostgreSQLRepository) (*pgconn.CommandTag, error)
		expectedProc string
		expectedArgs []any
	}{
		{
			name: "CreateCmkKey",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.CreateCmkKey(CreateCmkKeyInput{
					IDCmkKey:  id1,
					Algorithm: common.KeyTypeRSAOAEP,
					Purpose:   common.KeyPurposeSign,
					Status:    &status,
				})
			},
			expectedProc: "dragon_cmk.sp_create_cmk_key",
			expectedArgs: []any{id1, common.KeyTypeRSAOAEP, common.KeyPurposeSign, &status},
		},
		{
			name: "UpdateCmkKey",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.UpdateCmkKey(UpdateCmkKeyInput{
					IDCmkKey: id1,
					Status:   &status,
				})
			},
			expectedProc: "dragon_cmk.sp_update_cmk_key",
			expectedArgs: []any{id1, &status},
		},
		{
			name: "DeleteCmkKey",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.DeleteCmkKey(id1)
			},
			expectedProc: "dragon_cmk.sp_delete_cmk_key",
			expectedArgs: []any{id1},
		},
		{
			name: "CreateKeyCreationQueue",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.CreateKeyCreationQueue(CreateKeyCreationQueueInput{
					IDCmkKeyCreationQueue: id1,
					IDCmkKey:              id2,
					EventType:             common.EventTypeCreateKey,
					Status:                &queueStatus,
					ProcessedAt:           &now,
				})
			},
			expectedProc: "dragon_cmk.sp_create_creation_key_queue",
			expectedArgs: []any{id1, id2, common.EventTypeCreateKey, &queueStatus, &now},
		},
		{
			name: "UpdateKeyCreationQueue",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.UpdateKeyCreationQueue(UpdateKeyCreationQueueInput{
					IDCmkKeyCreationQueue: id1,
					Status:                &processedQueueStatus,
					ErrorMessage:          &errorMessage,
					ProcessedAt:           &now,
				})
			},
			expectedProc: "dragon_cmk.sp_update_creation_key_queue",
			expectedArgs: []any{id1, &processedQueueStatus, &errorMessage, &now},
		},
		{
			name: "DeleteKeyCreationQueue",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.DeleteKeyCreationQueue(id1)
			},
			expectedProc: "dragon_cmk.sp_delete_creation_key_queue",
			expectedArgs: []any{id1},
		},
		{
			name: "CreateKeyVersion",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.CreateKeyVersion(CreateKeyVersionInput{
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
				})
			},
			expectedProc: "dragon_cmk.sp_create_key_version",
			expectedArgs: []any{id1, id2, 1, 256, &keyVersionStatus, "kid-1", "wrapped", "RSA-OAEP", &id2, &aditional, &secretChecksum},
		},
		{
			name: "UpdateKeyVersionMetadata",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.UpdateKeyVersionMetadata(UpdateKeyVersionMetadataInput{
					IDCmkKeyVersion:     id1,
					IDCmkWrappingKeyRef: &id2,
					SecretWrapped:       &updatedSecretWrapped,
					SecretChecksum:      &secretChecksum,
				})
			},
			expectedProc: "dragon_cmk.sp_update_key_version_metadata",
			expectedArgs: []any{id1, &id2, &updatedSecretWrapped, &secretChecksum},
		},
		{
			name: "UpdateKeyVersionStatus",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.UpdateKeyVersionStatus(UpdateKeyVersionStatusInput{
					IDCmkKeyVersion: id1,
					Status:          updatedKeyVersionStatus,
				})
			},
			expectedProc: "dragon_cmk.sp_update_key_version_status",
			expectedArgs: []any{id1, updatedKeyVersionStatus},
		},
		{
			name: "RetireKeyVersion",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.RetireKeyVersion(id1)
			},
			expectedProc: "dragon_cmk.sp_retire_key_version",
			expectedArgs: []any{id1},
		},
		{
			name: "DeleteRetiredKeyVersion",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.DeleteRetiredKeyVersion(id1)
			},
			expectedProc: "dragon_cmk.sp_delete_retired_key_version",
			expectedArgs: []any{id1},
		},
		{
			name: "RotateKeyVersion",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.RotateKeyVersion(RotateKeyVersionInput{
					IDCmkKey:        id1,
					IDCmkKeyVersion: id2,
				})
			},
			expectedProc: "dragon_cmk.sp_rotate_key_version",
			expectedArgs: []any{id1, id2},
		},
		{
			name: "CreateWrappingKeyRef",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.CreateWrappingKeyRef(CreateWrappingKeyRefInput{
					IDCmkWrappingKeyRef: id1,
					Provider:            provider,
					KeyRef:              keyRef,
					Version:             version,
				})
			},
			expectedProc: "dragon_cmk.sp_create_wrapping_key_ref",
			expectedArgs: []any{id1, provider, keyRef, version},
		},
		{
			name: "UpdateWrappingKeyRef",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.UpdateWrappingKeyRef(UpdateWrappingKeyRefInput{
					IDCmkWrappingKeyRef: id1,
					Provider:            &provider,
					KeyRef:              &keyRef,
					Version:             &version,
				})
			},
			expectedProc: "dragon_cmk.sp_update_wrapping_key_ref",
			expectedArgs: []any{id1, &provider, &keyRef, &version},
		},
		{
			name: "DeleteWrappingKeyRef",
			call: func(repo PostgreSQLRepository) (*pgconn.CommandTag, error) {
				return repo.DeleteWrappingKeyRef(id1)
			},
			expectedProc: "dragon_cmk.sp_delete_wrapping_key_ref",
			expectedArgs: []any{id1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_success", func(t *testing.T) {
			ctx := context.Background()
			repo := PostgreSQLRepository{ctx: ctx, ctxLogger: builder.New(ctx)}
			wantTag := pgconn.NewCommandTag("CALL")

			original := callStoredProcedureFn
			t.Cleanup(func() {
				callStoredProcedureFn = original
			})

			callStoredProcedureFn = func(_ context.Context, procedureName string, args ...any) (*pgconn.CommandTag, error) {
				if procedureName != tt.expectedProc {
					t.Fatalf("unexpected procedure: %s", procedureName)
				}
				if !reflect.DeepEqual(args, tt.expectedArgs) {
					t.Fatalf("unexpected args: %#v", args)
				}
				tag := wantTag
				return &tag, nil
			}

			got, err := tt.call(repo)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil || got.String() != wantTag.String() {
				t.Fatalf("unexpected command tag: %#v", got)
			}
		})

		t.Run(tt.name+"_error", func(t *testing.T) {
			ctx := context.Background()
			repo := PostgreSQLRepository{ctx: ctx, ctxLogger: builder.New(ctx)}

			original := callStoredProcedureFn
			t.Cleanup(func() {
				callStoredProcedureFn = original
			})

			callStoredProcedureFn = func(_ context.Context, procedureName string, args ...any) (*pgconn.CommandTag, error) {
				return nil, errors.New("forced failure")
			}

			got, err := tt.call(repo)
			if err == nil {
				t.Fatal("expected error")
			}
			if got != nil {
				t.Fatalf("expected nil result, got %#v", got)
			}
		})
	}
}

func TestNewRepository(t *testing.T) {
	ctx := context.Background()
	ctxLogger := builder.New(ctx)
	repository := NewRepository(ctx, ctxLogger)
	if repository == nil {
		t.Fatal("expected repository instance")
	}

	postgreSQLRepository, ok := repository.(*PostgreSQLRepository)
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

func setupStoreLoggerTest(t *testing.T) {
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
