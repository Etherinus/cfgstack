package config

import (
	"github.com/etherinus/cfgstack/internal/errx"
	"gopkg.in/yaml.v3"
)

func parseYAML(path string, b []byte) (any, error) {
	var v any
	if err := yaml.Unmarshal(b, &v); err != nil {
		return nil, errx.Wrap("parse", path, "", "invalid YAML", err)
	}
	nv, err := normalizeAny(v, path)
	if err != nil {
		return nil, errx.Wrap("parse", path, "", "invalid YAML", err)
	}
	return nv, nil
}
