package inspect

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
)

type DiffKind string

const (
	DiffAdd    DiffKind = "add"
	DiffRemove DiffKind = "remove"
	DiffChange DiffKind = "change"
)

type DiffEntry struct {
	Ptr  string
	Kind DiffKind
	Old  any
	New  any
}

func Diff(oldDoc, newDoc any) []DiffEntry {
	oldFlat := map[string]any{}
	newFlat := map[string]any{}
	flattenForDiff(oldDoc, "", oldFlat)
	flattenForDiff(newDoc, "", newFlat)

	keys := make([]string, 0, len(oldFlat)+len(newFlat))
	seen := map[string]bool{}
	for k := range oldFlat {
		seen[k] = true
		keys = append(keys, k)
	}
	for k := range newFlat {
		if seen[k] {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]DiffEntry, 0, len(keys))
	for _, ptr := range keys {
		ov, okOld := oldFlat[ptr]
		nv, okNew := newFlat[ptr]
		switch {
		case !okOld && okNew:
			out = append(out, DiffEntry{Ptr: ptr, Kind: DiffAdd, New: nv})
		case okOld && !okNew:
			out = append(out, DiffEntry{Ptr: ptr, Kind: DiffRemove, Old: ov})
		default:
			if !reflect.DeepEqual(canonicalForDiff(ov), canonicalForDiff(nv)) {
				out = append(out, DiffEntry{Ptr: ptr, Kind: DiffChange, Old: ov, New: nv})
			}
		}
	}
	return out
}

func flattenForDiff(v any, ptr string, out map[string]any) {
	if ptr == "" {
		ptr = "/"
	}

	switch t := v.(type) {
	case map[string]any:
		if len(t) == 0 {
			out[ptr] = map[string]any{}
			return
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			flattenForDiff(t[k], joinChildPointer(ptr, escapePointerToken(k)), out)
		}
	case []any:
		if len(t) == 0 {
			out[ptr] = []any{}
			return
		}
		for i := range t {
			flattenForDiff(t[i], joinChildPointer(ptr, strconv.Itoa(i)), out)
		}
	default:
		out[ptr] = v
	}
}

func joinChildPointer(parent, token string) string {
	if parent == "/" {
		return "/" + token
	}
	return parent + "/" + token
}

func canonicalForDiff(v any) any {
	switch t := v.(type) {
	case nil, bool, string:
		return t
	case int:
		return int64(t)
	case int8:
		return int64(t)
	case int16:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	case uint:
		return uint64(t)
	case uint8:
		return uint64(t)
	case uint16:
		return uint64(t)
	case uint32:
		return uint64(t)
	case uint64:
		return t
	case float32:
		return float64(t)
	case float64:
		return t
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, vv := range t {
			m[k] = canonicalForDiff(vv)
		}
		return m
	case []any:
		a := make([]any, len(t))
		for i := range t {
			a[i] = canonicalForDiff(t[i])
		}
		return a
	default:
		return t
	}
}
