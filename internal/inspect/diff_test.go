package inspect

import "testing"

func TestDiffAddChangeRemove(t *testing.T) {
	oldDoc := map[string]any{
		"a": int64(1),
		"b": map[string]any{"x": int64(1)},
		"c": "gone",
	}
	newDoc := map[string]any{
		"a": int64(2),
		"b": map[string]any{"x": int64(1), "y": true},
	}

	got := Diff(oldDoc, newDoc)
	if len(got) != 3 {
		t.Fatalf("expected 3 diff entries, got %d (%+v)", len(got), got)
	}

	if got[0].Kind != DiffChange || got[0].Ptr != "/a" {
		t.Fatalf("unexpected first diff entry: %+v", got[0])
	}
	if got[1].Kind != DiffAdd || got[1].Ptr != "/b/y" {
		t.Fatalf("unexpected second diff entry: %+v", got[1])
	}
	if got[2].Kind != DiffRemove || got[2].Ptr != "/c" {
		t.Fatalf("unexpected third diff entry: %+v", got[2])
	}
}

func TestDiffCanonicalNumbers(t *testing.T) {
	oldDoc := map[string]any{"a": int64(1)}
	newDoc := map[string]any{"a": int(1)}

	got := Diff(oldDoc, newDoc)
	if len(got) != 0 {
		t.Fatalf("expected no diff for semantically equal numbers, got %+v", got)
	}
}
