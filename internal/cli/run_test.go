package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRunNoArgs(t *testing.T) {
	code := Run([]string{"cfgstack"})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunHelp(t *testing.T) {
	code := Run([]string{"cfgstack", "help"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunVersionCommand(t *testing.T) {
	code := Run([]string{"cfgstack", "version"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunVersionFlag(t *testing.T) {
	code := Run([]string{"cfgstack", "--version"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunVersionJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runVersion(&stdout, &stderr, []string{"--json"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}

	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid json output: %v, out=%q", err, stdout.String())
	}

	if out["name"] != "cfgstack" {
		t.Fatalf("expected name=cfgstack, got %v", out["name"])
	}
	if _, ok := out["version"]; !ok {
		t.Fatalf("expected version field in json output")
	}
	if _, ok := out["commit"]; !ok {
		t.Fatalf("expected commit field in json output")
	}
	if _, ok := out["date"]; !ok {
		t.Fatalf("expected date field in json output")
	}
}

func TestRunVersionHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runVersion(&stdout, &stderr, []string{"--help"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.Len() == 0 {
		t.Fatalf("expected help output")
	}
}

func TestRunVersionRejectsPositionalArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runVersion(&stdout, &stderr, []string{"extra"})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected error output")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	code := Run([]string{"cfgstack", "unknown"})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunBuildMissingProfile(t *testing.T) {
	code := Run([]string{"cfgstack", "build", "--in", "config"})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunBuildInvalidPrintSourcesWithStdout(t *testing.T) {
	code := Run([]string{"cfgstack", "build", "--allow-empty-profile", "--print-sources", "--out", "-"})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunExplainInvalidPointer(t *testing.T) {
	code := Run([]string{"cfgstack", "explain", "--allow-empty-profile", "--at", "db/host"})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunDoctorInvalidMaxConflicts(t *testing.T) {
	code := Run([]string{"cfgstack", "doctor", "--allow-empty-profile", "--max-conflicts", "-1"})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}
