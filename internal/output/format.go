package output

import (
	"path/filepath"
	"strings"

	"github.com/etherinus/cfg-fuse/internal/errx"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
	FormatTOML Format = "toml"
)

func ResolveFormat(outPath, override string) (Format, error) {
	ov := strings.ToLower(strings.TrimSpace(override))
	if ov != "" {
		switch ov {
		case "json":
			return FormatJSON, nil
		case "yaml", "yml":
			return FormatYAML, nil
		case "toml":
			return FormatTOML, nil
		default:
			return "", &errx.Err{Op: "write", Msg: "invalid --format, expected json|yaml|toml"}
		}
	}

	if outPath == "-" || strings.TrimSpace(outPath) == "" {
		return FormatJSON, nil
	}

	ext := strings.ToLower(filepath.Ext(outPath))
	switch ext {
	case ".json":
		return FormatJSON, nil
	case ".yaml", ".yml":
		return FormatYAML, nil
	case ".toml":
		return FormatTOML, nil
	case "":
		return FormatJSON, nil
	default:
		return "", &errx.Err{Op: "write", File: outPath, Msg: "unsupported output extension: " + ext}
	}
}
