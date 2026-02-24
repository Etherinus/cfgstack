package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunDoctorJSONOk(t *testing.T) {
	dir := t.TempDir()
	mustWriteDoctorFile(t, dir, "default.json", `{"a":1}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runDoctor(&stdout, &stderr, []string{
		"--in", dir,
		"--allow-empty-profile",
		"--json",
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}

	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid json output: %v, out=%q", err, stdout.String())
	}
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got %v", out["ok"])
	}
	if out["conflict_count"] != float64(0) {
		t.Fatalf("expected conflict_count=0, got %v", out["conflict_count"])
	}
}

func TestRunDoctorJSONConflict(t *testing.T) {
	dir := t.TempDir()
	mustWriteDoctorFile(t, dir, "default.json", `{"a":1}`)
	mustWriteDoctorFile(t, dir, "local.json", `{"a":2}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runDoctor(&stdout, &stderr, []string{
		"--in", dir,
		"--allow-empty-profile",
		"--json",
	})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d, stderr=%s", code, stderr.String())
	}

	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid json output: %v, out=%q", err, stdout.String())
	}
	if out["ok"] != false {
		t.Fatalf("expected ok=false, got %v", out["ok"])
	}
	if out["conflict_count"] != float64(1) {
		t.Fatalf("expected conflict_count=1, got %v", out["conflict_count"])
	}
}

func TestRunDoctorJSONMissingDefault(t *testing.T) {
	dir := t.TempDir()
	mustWriteDoctorFile(t, dir, "local.json", `{"a":1}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runDoctor(&stdout, &stderr, []string{
		"--in", dir,
		"--allow-empty-profile",
		"--fail-on-missing-default",
		"--json",
	})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d, stderr=%s", code, stderr.String())
	}

	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid json output: %v, out=%q", err, stdout.String())
	}
	if out["missing_default"] != true {
		t.Fatalf("expected missing_default=true, got %v", out["missing_default"])
	}
}

func mustWriteDoctorFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
