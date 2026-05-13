package utilities

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestGetSecretCmkKey(t *testing.T) {
	idCmkKey := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	idCmkKeyVersion := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	secret := idCmkKey.String() + "." + idCmkKeyVersion.String()

	gotID, gotVersion, err := GetSecretCmkKey(base64.StdEncoding.EncodeToString([]byte(secret)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *gotID != idCmkKey {
		t.Fatalf("unexpected id cmk key: %s", gotID)
	}
	if *gotVersion != idCmkKeyVersion {
		t.Fatalf("unexpected id cmk key version: %s", gotVersion)
	}
}

func TestGetSecretCmkKeyAcceptsPlainSecret(t *testing.T) {
	idCmkKey := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	idCmkKeyVersion := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	gotID, gotVersion, err := GetSecretCmkKey(idCmkKey.String() + "." + idCmkKeyVersion.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *gotID != idCmkKey {
		t.Fatalf("unexpected id cmk key: %s", gotID)
	}
	if *gotVersion != idCmkKeyVersion {
		t.Fatalf("unexpected id cmk key version: %s", gotVersion)
	}
}

func TestGetSecretCmkKeyErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  error
	}{
		{
			name:  "missing version",
			input: base64.StdEncoding.EncodeToString([]byte(uuid.NewString())),
			want:  errInvalidSecretCmkKey,
		},
		{
			name:  "invalid cmk key uuid",
			input: base64.StdEncoding.EncodeToString([]byte("bad." + uuid.NewString())),
		},
		{
			name:  "invalid cmk key version uuid",
			input: base64.StdEncoding.EncodeToString([]byte(uuid.NewString() + ".bad")),
		},
		{
			name:  "too many parts",
			input: base64.StdEncoding.EncodeToString([]byte(uuid.NewString() + "." + uuid.NewString() + ".extra")),
			want:  errInvalidSecretCmkKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotVersion, err := GetSecretCmkKey(tt.input)
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			if gotID != nil || gotVersion != nil {
				t.Fatalf("expected nil ids, got %v %v", gotID, gotVersion)
			}
		})
	}
}
