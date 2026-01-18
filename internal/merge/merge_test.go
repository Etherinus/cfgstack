package merge

import "testing"

func TestDeepMerge(t *testing.T) {
	base := map[string]any{
		"a": map[string]any{
			"b": int64(1),
			"c": []any{int64(1), int64(2)},
		},
		"x": "y",
	}
	over := map[string]any{
		"a": map[string]any{
			"b": int64(2),
			"d": true,
			"c": []any{int64(9)},
		},
		"z": int64(3),
	}
	m, err := Deep(base, over)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	am := m["a"].(map[string]any)
	if am["b"].(int64) != 2 {
		t.Fatalf("expected a.b=2, got %v", am["b"])
	}
	if am["d"].(bool) != true {
		t.Fatalf("expected a.d=true")
	}
	c := am["c"].([]any)
	if len(c) != 1 || c[0].(int64) != 9 {
		t.Fatalf("expected a.c=[9], got %#v", c)
	}
	if m["x"].(string) != "y" {
		t.Fatalf("expected x=y")
	}
	if m["z"].(int64) != 3 {
		t.Fatalf("expected z=3")
	}
}
