package merge

import (
	"sort"

	"github.com/etherinus/cfg-fuse/internal/prov"
)

func Deep(base, overlay map[string]any) (map[string]any, error) {
	return DeepWithProv(base, overlay, nil, prov.Source{}, "")
}

func DeepWithProv(base, overlay map[string]any, p *prov.Map, src prov.Source, ptr string) (map[string]any, error) {
	if base == nil {
		base = map[string]any{}
	}
	res := deepCopyMap(base)

	keys := make([]string, 0, len(overlay))
	for k := range overlay {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := overlay[k]
		childPtr := ptr + "/" + prov.Escape(k)
		if p != nil {
			p.Set(childPtr, src)
		}

		bv, ok := res[k]
		if ok {
			bm, bok := bv.(map[string]any)
			om, ook := v.(map[string]any)
			if bok && ook {
				merged, err := DeepWithProv(bm, om, p, src, childPtr)
				if err != nil {
					return nil, err
				}
				res[k] = merged
				continue
			}
		}
		res[k] = deepCopyAny(v)
	}
	return res, nil
}

func deepCopyAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return deepCopyMap(t)
	case []any:
		a := make([]any, len(t))
		for i := range t {
			a[i] = deepCopyAny(t[i])
		}
		return a
	default:
		return v
	}
}

func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = deepCopyAny(v)
	}
	return out
}
