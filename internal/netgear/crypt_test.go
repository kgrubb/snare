package netgear

import (
	"crypto/md5"
	"encoding/hex"
	"testing"
)

func TestMergeHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		rand     string
		want     string
	}{
		{
			name:     "interleave equal length",
			password: "ab",
			rand:     "12",
			want:     md5Hex("a1b2"),
		},
		{
			name:     "password longer",
			password: "abc",
			rand:     "1",
			want:     md5Hex("a1bc"),
		},
		{
			name:     "rand longer",
			password: "a",
			rand:     "123",
			want:     md5Hex("a123"),
		},
		{
			name:     "empty rand",
			password: "pw",
			rand:     "",
			want:     md5Hex("pw"),
		},
		{
			name:     "empty password",
			password: "",
			rand:     "xy",
			want:     md5Hex("xy"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mergeHash(tt.password, tt.rand)
			if got != tt.want {
				t.Fatalf("mergeHash(%q, %q) = %q, want %q", tt.password, tt.rand, got, tt.want)
			}
			if mergeHash(tt.password, tt.rand) != got {
				t.Fatal("not deterministic")
			}
		})
	}
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
