package netgear

import "testing"

func TestMergeHash(t *testing.T) {
	// Interleave "ab"+"12" => "a1b2", md5 of that.
	got := mergeHash("ab", "12")
	if len(got) != 32 {
		t.Fatalf("expected 32 hex chars, got %q", got)
	}
	if mergeHash("ab", "12") != got {
		t.Fatal("not deterministic")
	}
	if mergeHash("ab", "12") == mergeHash("ab", "13") {
		t.Fatal("rand must change hash")
	}
}
