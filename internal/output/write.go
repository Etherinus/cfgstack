package output

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/etherinus/cfg-fuse/internal/errx"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func WriteTo(w io.Writer, format Format, data any) error {
	b, err := encode(format, data)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	if err != nil {
		return errx.Wrap("write", "", "", "failed to write output", err)
	}
	return nil
}

func WriteFile(path string, format Format, data any) error {
	if strings.TrimSpace(path) == "" {
		return &errx.Err{Op: "write", Msg: "missing output path"}
	}
	b, err := encode(format, data)
	if err != nil {
		return errx.Wrap("write", path, "", "failed to encode output", err)
	}
	return writeAtomic(path, b)
}

func encode(format Format, data any) ([]byte, error) {
	switch format {
	case FormatJSON:
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, err
		}
		b = append(b, '\n')
		return b, nil
	case FormatYAML:
		return yaml.Marshal(data)
	case FormatTOML:
		return toml.Marshal(data)
	default:
		return nil, &errx.Err{Op: "write", Msg: "unsupported format: " + string(format)}
	}
}

func writeAtomic(path string, b []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	f, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return errx.Wrap("write", path, "", "failed to create temp file", err)
	}
	tmp := f.Name()
	closeOK := false
	defer func() {
		_ = f.Close()
		if !closeOK {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(b); err != nil {
		return errx.Wrap("write", path, "", "failed to write temp file", err)
	}
	if err := f.Close(); err != nil {
		return errx.Wrap("write", path, "", "failed to close temp file", err)
	}
	closeOK = true
	if err := os.Rename(tmp, path); err != nil {
		return errx.Wrap("write", path, "", "failed to rename temp file", err)
	}
	return nil
}
