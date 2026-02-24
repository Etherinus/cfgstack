package output

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveFormat(t *testing.T) {
	tests := []struct {
		name     string
		outPath  string
		override string
		want     Format
		wantErr  bool
	}{
		{name: "stdout-default", outPath: "-", want: FormatJSON},
		{name: "from-extension-yaml", outPath: "merged.yaml", want: FormatYAML},
		{name: "override-wins", outPath: "merged.json", override: "toml", want: FormatTOML},
		{name: "invalid-extension", outPath: "merged.txt", wantErr: true},
		{name: "invalid-override", outPath: "-", override: "xml", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveFormat(tt.outPath, tt.override)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected format: got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestWriteFileAtomicJSON(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "merged.json")

	if err := WriteFile(out, FormatJSON, map[string]any{"a": int64(1)}); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"a": 1`) {
		t.Fatalf("unexpected output content: %s", s)
	}
	if !strings.HasSuffix(s, "\n") {
		t.Fatalf("expected trailing newline in json output")
	}

	tmpFiles, err := filepath.Glob(filepath.Join(dir, ".merged.json.*.tmp"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(tmpFiles) != 0 {
		t.Fatalf("expected no temp files left, got: %v", tmpFiles)
	}
}

func TestWriteFileMissingPath(t *testing.T) {
	err := WriteFile("", FormatJSON, map[string]any{"x": 1})
	if err == nil {
		t.Fatal("expected error for empty output path")
	}
}

func TestWriteToUnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := WriteTo(&buf, Format("xml"), map[string]any{"x": 1})
	if err == nil {
		t.Fatal("expected unsupported format error")
	}
}
