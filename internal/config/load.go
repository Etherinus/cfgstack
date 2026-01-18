package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/etherinus/cfg-fuse/internal/errx"
	"github.com/etherinus/cfg-fuse/internal/merge"
	"github.com/etherinus/cfg-fuse/internal/prov"
)

type Source struct {
	Layer string
	File  string
}

func LoadLayers(dir, profile string, failOnMissingDefault bool, allowEmptyProfile bool) (map[string]any, *prov.Map, []Source, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	if strings.TrimSpace(profile) == "" && !allowEmptyProfile {
		return nil, nil, nil, &errx.Err{Op: "load", File: absDir, Msg: "missing profile (or pass --allow-empty-profile)"}
	}

	layers := []string{"default", "local"}
	if profile != "" {
		layers = append(layers, profile)
	}

	var sources []Source
	mergedCfg := map[string]any{}
	p := prov.New()

	defaultFiles, _, err := findLayerFilesDetailed(absDir, "default")
	if err != nil {
		return nil, nil, nil, err
	}
	if failOnMissingDefault && len(defaultFiles) == 0 {
		return nil, nil, nil, &errx.Err{Op: "load", File: absDir, Msg: "missing default.*"}
	}

	for _, layer := range layers {
		files, err := findLayerFiles(absDir, layer)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, file := range files {
			m, err := parseFileToMap(file)
			if err != nil {
				return nil, nil, nil, err
			}
			src := prov.Source{Layer: layer, File: file}
			mergedCfg, err = merge.DeepWithProv(mergedCfg, m, p, src, "")
			if err != nil {
				return nil, nil, nil, err
			}
			sources = append(sources, Source{Layer: layer, File: file})
		}
	}

	return mergedCfg, p, sources, nil
}

func findLayerFiles(dir, layer string) ([]string, error) {
	sup, _, err := findLayerFilesDetailed(dir, layer)
	return sup, err
}

func findLayerFilesDetailed(dir, layer string) ([]string, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, errx.Wrap("read", dir, "", "failed to read directory", err)
	}
	prefix := layer + "."
	var supported []string
	var unsupported []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		full := filepath.Join(dir, name)
		if isSupportedExt(ext) {
			supported = append(supported, full)
		} else {
			unsupported = append(unsupported, full)
		}
	}
	sort.Strings(supported)
	sort.Strings(unsupported)
	return supported, unsupported, nil
}

func isSupportedExt(ext string) bool {
	switch ext {
	case ".json", ".yaml", ".yml", ".toml":
		return true
	default:
		return false
	}
}

func parseFileToMap(path string) (map[string]any, error) {
	v, err := parseFile(path)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, &errx.Err{Op: "load", File: path, Msg: "root value must be an object"}
	}
	return m, nil
}

func parseFile(path string) (any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errx.Wrap("read", path, "", "failed to read file", err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return parseJSON(path, b)
	case ".yaml", ".yml":
		return parseYAML(path, b)
	case ".toml":
		return parseTOML(path, b)
	default:
		return nil, &errx.Err{Op: "parse", File: path, Msg: fmt.Sprintf("unsupported file extension: %s", ext)}
	}
}
