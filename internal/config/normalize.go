package config

import (
	"encoding/json"
	"fmt"
)

func normalizeAny(v any, file string) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, vv := range t {
			nv, err := normalizeAny(vv, file)
			if err != nil {
				return nil, err
			}
			m[k] = nv
		}
		return m, nil
	case map[any]any:
		m := make(map[string]any, len(t))
		for k, vv := range t {
			ks, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("non-string key in map: %T", k)
			}
			nv, err := normalizeAny(vv, file)
			if err != nil {
				return nil, err
			}
			m[ks] = nv
		}
		return m, nil
	case []any:
		a := make([]any, len(t))
		for i := range t {
			nv, err := normalizeAny(t[i], file)
			if err != nil {
				return nil, err
			}
			a[i] = nv
		}
		return a, nil
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i, nil
		}
		if f, err := t.Float64(); err == nil {
			return f, nil
		}
		return t.String(), nil
	default:
		return v, nil
	}
}
