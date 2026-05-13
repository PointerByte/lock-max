package views

import "testing"

func TestTableNameMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "cmk key",
			got:  CmkKeyView{}.TableName(),
			want: "dragon_cmk.vw_cmk_key",
		},
		{
			name: "cmk creation key queue",
			got:  CmkCreationKeyQueueView{}.TableName(),
			want: "dragon_cmk.vw_cmk_creation_key_queue",
		},
		{
			name: "cmk key version",
			got:  CmkKeyVersionView{}.TableName(),
			want: "dragon_cmk.vw_cmk_key_version",
		},
		{
			name: "cmk wrapping key ref",
			got:  CmkWrappingKeyRefView{}.TableName(),
			want: "dragon_cmk.vw_cmk_wrapping_key_ref",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.got != tt.want {
				t.Fatalf("TableName() = %q, want %q", tt.got, tt.want)
			}
		})
	}
}
