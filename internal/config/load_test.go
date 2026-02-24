package config

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestLoadLayersOrderAndProvenance(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, dir, "default.a.json", `{"a":0,"nested":{"seed":true}}`)
	mustWriteFile(t, dir, "default.b.json", `{"a":1,"nested":{"from":"default-b"}}`)
	mustWriteFile(t, dir, "local.10.json", `{"nested":{"from":"local"}}`)
	mustWriteFile(t, dir, "prod.20.json", `{"a":3}`)

	cfg, p, sources, err := LoadLayers(dir, "prod", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := cfg["a"].(int64); got != 3 {
		t.Fatalf("expected a=3, got %v", got)
	}
	nested := cfg["nested"].(map[string]any)
	if got := nested["from"].(string); got != "local" {
		t.Fatalf("expected nested.from=local, got %v", got)
	}
	if got := nested["seed"].(bool); !got {
		t.Fatalf("expected nested.seed=true, got %v", got)
	}

	gotOrder := make([]string, 0, len(sources))
	for _, s := range sources {
		gotOrder = append(gotOrder, filepath.Base(s.File))
	}
	wantOrder := []string{"default.a.json", "default.b.json", "local.10.json", "prod.20.json"}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("unexpected source order: got %v, want %v", gotOrder, wantOrder)
	}

	srcA, ok := p.LookupNearest("/a")
	if !ok || filepath.Base(srcA.File) != "prod.20.json" {
		t.Fatalf("expected provenance /a from prod.20.json, got %+v (ok=%v)", srcA, ok)
	}
	srcFrom, ok := p.LookupNearest("/nested/from")
	if !ok || filepath.Base(srcFrom.File) != "local.10.json" {
		t.Fatalf("expected provenance /nested/from from local.10.json, got %+v (ok=%v)", srcFrom, ok)
	}
}

func TestLoadLayersMissingDefault(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, dir, "local.json", `{"x":1}`)

	_, _, _, err := LoadLayers(dir, "prod", true, false)
	if err == nil {
		t.Fatal("expected missing default error, got nil")
	}
}

func TestLoadLayersAllowEmptyProfile(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, dir, "default.json", `{"x":1}`)

	cfg, _, sources, err := LoadLayers(dir, "", false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cfg["x"].(int64); got != 1 {
		t.Fatalf("expected x=1, got %v", got)
	}
	if len(sources) != 1 || filepath.Base(sources[0].File) != "default.json" {
		t.Fatalf("unexpected sources: %+v", sources)
	}
}

func TestScanLayersSupportedAndUnsupported(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, dir, "default.json", `{"x":1}`)
	mustWriteFile(t, dir, "default.ini", `x=1`)
	mustWriteFile(t, dir, "prod.toml", `x=2`)

	scan, err := ScanLayers(dir, "prod", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotDefaultSup := baseNames(scan.Supported["default"])
	gotDefaultUns := baseNames(scan.Unsupported["default"])
	gotProdSup := baseNames(scan.Supported["prod"])

	if !reflect.DeepEqual(gotDefaultSup, []string{"default.json"}) {
		t.Fatalf("unexpected default supported: %v", gotDefaultSup)
	}
	if !reflect.DeepEqual(gotDefaultUns, []string{"default.ini"}) {
		t.Fatalf("unexpected default unsupported: %v", gotDefaultUns)
	}
	if !reflect.DeepEqual(gotProdSup, []string{"prod.toml"}) {
		t.Fatalf("unexpected prod supported: %v", gotProdSup)
	}
}

func mustWriteFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func baseNames(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, filepath.Base(p))
	}
	sort.Strings(out)
	return out
}
