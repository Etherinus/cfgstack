package env

import (
	"os"
	"testing"

	"github.com/etherinus/cfg-fuse/internal/prov"
)

func TestApplyLower(t *testing.T) {
	prefix := "CFG_FUSE_TEST"
	delim := "__"
	k1 := prefix + delim + "A" + delim + "B"
	k2 := prefix + delim + "X"
	k3 := prefix + delim + "ARR" + delim + "0" + delim + "K"

	os.Setenv(k1, "1")
	os.Setenv(k2, "true")
	os.Setenv(k3, "9")
	defer func() {
		os.Unsetenv(k1)
		os.Unsetenv(k2)
		os.Unsetenv(k3)
	}()

	root := map[string]any{}
	p := prov.New()

	m, _, err := Apply(root, prefix, delim, CaseLower, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	am := m["a"].(map[string]any)
	if am["b"].(int64) != 1 {
		t.Fatalf("expected a.b=1, got %v", am["b"])
	}
	if m["x"].(bool) != true {
		t.Fatalf("expected x=true")
	}

	arr := m["arr"].([]any)
	obj := arr[0].(map[string]any)
	if obj["k"].(int64) != 9 {
		t.Fatalf("expected arr[0].k=9, got %v", obj["k"])
	}

	if _, ok := p.LookupNearest("/a/b"); !ok {
		t.Fatalf("expected provenance for /a/b")
	}
	if _, ok := p.LookupNearest("/arr/0/k"); !ok {
		t.Fatalf("expected provenance for /arr/0/k")
	}
}

func TestApplyKeep(t *testing.T) {
	prefix := "CFG_FUSE_TEST2"
	delim := "__"
	k1 := prefix + delim + "CamelCase" + delim + "InnerKey"
	os.Setenv(k1, "1")
	defer func() {
		os.Unsetenv(k1)
	}()

	root := map[string]any{}
	m, _, err := Apply(root, prefix, delim, CaseKeep, prov.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cc := m["CamelCase"].(map[string]any)
	if cc["InnerKey"].(int64) != 1 {
		t.Fatalf("expected CamelCase.InnerKey=1, got %v", cc["InnerKey"])
	}
}
