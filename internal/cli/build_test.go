package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBuildDryRunNoWrite(t *testing.T) {
	cfgDir := t.TempDir()
	outFile := filepath.Join(t.TempDir(), "merged.json")
	mustWriteJSONFile(t, cfgDir, "default.json", `{"a":1}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runBuild(&stdout, &stderr, []string{
		"--in", cfgDir,
		"--allow-empty-profile",
		"--env-prefix", "CFGSTACK_TEST_NO_ENV",
		"--out", outFile,
		"--dry-run",
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}

	if _, err := os.Stat(outFile); !os.IsNotExist(err) {
		t.Fatalf("expected output file to not be written in dry-run mode")
	}
	if !strings.Contains(stdout.String(), "ok: dry-run") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunBuildDryRunDiff(t *testing.T) {
	cfgDir := t.TempDir()
	outFile := filepath.Join(t.TempDir(), "merged.json")
	mustWriteJSONFile(t, cfgDir, "default.json", `{"a":2,"b":3}`)
	mustWriteJSONFile(t, filepath.Dir(outFile), filepath.Base(outFile), `{"a":1}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runBuild(&stdout, &stderr, []string{
		"--in", cfgDir,
		"--allow-empty-profile",
		"--env-prefix", "CFGSTACK_TEST_NO_ENV",
		"--out", outFile,
		"--dry-run",
		"--diff",
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "~ /a: 1 -> 2") {
		t.Fatalf("expected changed pointer in diff output, got %q", out)
	}
	if !strings.Contains(out, "+ /b = 3") {
		t.Fatalf("expected added pointer in diff output, got %q", out)
	}

	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(b) != "{\"a\":1}" {
		t.Fatalf("expected dry-run to not change file, got %q", string(b))
	}
}

func TestRunBuildDiffRequiresDryRun(t *testing.T) {
	cfgDir := t.TempDir()
	mustWriteJSONFile(t, cfgDir, "default.json", `{"a":1}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runBuild(&stdout, &stderr, []string{
		"--in", cfgDir,
		"--allow-empty-profile",
		"--env-prefix", "CFGSTACK_TEST_NO_ENV",
		"--out", filepath.Join(t.TempDir(), "merged.json"),
		"--diff",
	})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--diff requires --dry-run") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunBuildDiffRequiresOutFile(t *testing.T) {
	cfgDir := t.TempDir()
	mustWriteJSONFile(t, cfgDir, "default.json", `{"a":1}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runBuild(&stdout, &stderr, []string{
		"--in", cfgDir,
		"--allow-empty-profile",
		"--env-prefix", "CFGSTACK_TEST_NO_ENV",
		"--out", "-",
		"--dry-run",
		"--diff",
	})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--diff requires --out not '-'") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func mustWriteJSONFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
