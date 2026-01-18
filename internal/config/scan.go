package config

import (
	"os"
	"path/filepath"
)

type ScanResult struct {
	Dir         string
	Profile     string
	Layers      []string
	Supported   map[string][]string
	Unsupported map[string][]string
}

func ScanLayers(dir, profile string, allowEmptyProfile bool) (ScanResult, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	_, err = os.ReadDir(absDir)
	if err != nil {
		return ScanResult{}, err
	}

	layers := []string{"default", "local"}
	if profile != "" {
		layers = append(layers, profile)
	} else if !allowEmptyProfile {
		layers = append(layers, "")
	}

	supported := map[string][]string{}
	unsupported := map[string][]string{}

	for _, layer := range layers {
		if layer == "" {
			continue
		}
		sup, uns, err := findLayerFilesDetailed(absDir, layer)
		if err != nil {
			return ScanResult{}, err
		}
		supported[layer] = sup
		unsupported[layer] = uns
	}

	return ScanResult{
		Dir:         absDir,
		Profile:     profile,
		Layers:      layersResolved(profile, allowEmptyProfile),
		Supported:   supported,
		Unsupported: unsupported,
	}, nil
}

func layersResolved(profile string, allowEmptyProfile bool) []string {
	out := []string{"default", "local"}
	if profile != "" {
		out = append(out, profile)
	} else if !allowEmptyProfile {
		out = append(out, "<profile>")
	}
	return out
}
