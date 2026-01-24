package config

import (
	"github.com/etherinus/cfgstack/internal/errx"
	"github.com/pelletier/go-toml/v2"
)

func parseTOML(path string, b []byte) (any, error) {
	var m map[string]any
	if err := toml.Unmarshal(b, &m); err != nil {
		return nil, errx.Wrap("parse", path, "", "invalid TOML", err)
	}
	nv, err := normalizeAny(m, path)
	if err != nil {
		return nil, errx.Wrap("parse", path, "", "invalid TOML", err)
	}
	return nv, nil
}
