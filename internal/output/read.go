package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/etherinus/cfgstack/internal/errx"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func ParseBytes(format Format, b []byte) (any, error) {
	switch format {
	case FormatJSON:
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, errx.Wrap("read", "", "", "failed to parse JSON output", err)
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			if err == nil {
				return nil, &errx.Err{Op: "read", Msg: "failed to parse JSON output: extra content"}
			}
			return nil, errx.Wrap("read", "", "", "failed to parse JSON output", err)
		}
		return v, nil
	case FormatYAML:
		var v any
		if err := yaml.Unmarshal(b, &v); err != nil {
			return nil, errx.Wrap("read", "", "", "failed to parse YAML output", err)
		}
		return v, nil
	case FormatTOML:
		var v map[string]any
		if err := toml.Unmarshal(b, &v); err != nil {
			return nil, errx.Wrap("read", "", "", "failed to parse TOML output", err)
		}
		return v, nil
	default:
		return nil, &errx.Err{Op: "read", Msg: "unsupported format: " + string(format)}
	}
}

func ReadFile(path string, format Format) (any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errx.Wrap("read", path, "", "failed to read output file", err)
	}
	v, err := ParseBytes(format, b)
	if err != nil {
		return nil, errx.Wrap("read", path, "", "failed to parse output file", err)
	}
	return v, nil
}
