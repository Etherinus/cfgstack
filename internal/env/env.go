package env

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/etherinus/cfg-fuse/internal/errx"
	"github.com/etherinus/cfg-fuse/internal/prov"
)

type segment struct {
	key     string
	index   int
	isIndex bool
}

func Apply(root map[string]any, prefix, delim string, mode CaseMode, p *prov.Map) (map[string]any, int, error) {
	if root == nil {
		root = map[string]any{}
	}
	if strings.TrimSpace(prefix) == "" {
		return nil, 0, &errx.Err{Op: "env", Msg: "env prefix cannot be empty"}
	}
	if delim == "" {
		return nil, 0, &errx.Err{Op: "env", Msg: "env delimiter cannot be empty"}
	}

	out := root
	pfx := prefix + delim
	applied := 0

	envs := os.Environ()
	sort.Strings(envs)

	for _, kv := range envs {
		i := strings.IndexByte(kv, '=')
		if i <= 0 {
			continue
		}
		name := kv[:i]
		val := kv[i+1:]
		if !strings.HasPrefix(name, pfx) {
			continue
		}

		rawPath := strings.TrimPrefix(name, pfx)
		if rawPath == "" {
			return nil, applied, &errx.Err{Op: "env", File: name, Msg: "empty path"}
		}

		segs, keyPath, err := parseSegments(rawPath, delim, mode)
		if err != nil {
			return nil, applied, errx.Wrap("env", name, "", "invalid env path", err)
		}
		if len(segs) == 0 {
			return nil, applied, &errx.Err{Op: "env", File: name, Msg: "empty path"}
		}
		if segs[0].isIndex {
			return nil, applied, &errx.Err{Op: "env", File: name, Msg: "root segment cannot be an index"}
		}

		parsed, err := parseValue(val)
		if err != nil {
			return nil, applied, errx.Wrap("env", name, keyPath, "invalid env value", err)
		}

		src := prov.Source{Layer: "env", File: name}
		ptr := ""
		for _, s := range segs {
			if s.isIndex {
				ptr = ptr + "/" + strconv.Itoa(s.index)
			} else {
				ptr = ptr + "/" + prov.Escape(s.key)
			}
			if p != nil {
				p.Set(ptr, src)
			}
		}

		newRoot, err := setAny(out, segs, parsed)
		if err != nil {
			return nil, applied, errx.Wrap("env", name, keyPath, "failed to apply env override", err)
		}
		m, ok := newRoot.(map[string]any)
		if !ok {
			return nil, applied, &errx.Err{Op: "env", File: name, Key: keyPath, Msg: "root must remain an object"}
		}
		out = m
		applied++
	}

	return out, applied, nil
}

func parseSegments(rawPath, delim string, mode CaseMode) ([]segment, string, error) {
	parts := strings.Split(rawPath, delim)
	segs := make([]segment, 0, len(parts))
	var b strings.Builder
	firstKeyWritten := false

	for _, p := range parts {
		if p == "" {
			return nil, "", &errx.Err{Op: "env", Msg: "empty path segment"}
		}
		if isDigits(p) {
			idx64, err := strconv.ParseInt(p, 10, 32)
			if err != nil || idx64 < 0 {
				return nil, "", &errx.Err{Op: "env", Msg: "invalid array index"}
			}
			idx := int(idx64)
			segs = append(segs, segment{index: idx, isIndex: true})
			b.WriteString("[")
			b.WriteString(strconv.Itoa(idx))
			b.WriteString("]")
			continue
		}

		key := p
		if mode == CaseLower {
			key = strings.ToLower(key)
		}
		segs = append(segs, segment{key: key})

		if !firstKeyWritten {
			b.WriteString(key)
			firstKeyWritten = true
		} else {
			b.WriteString(".")
			b.WriteString(key)
		}
	}

	return segs, b.String(), nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func parseValue(s string) (any, error) {
	ts := strings.TrimSpace(s)
	if strings.EqualFold(ts, "true") {
		return true, nil
	}
	if strings.EqualFold(ts, "false") {
		return false, nil
	}
	if strings.EqualFold(ts, "null") {
		return nil, nil
	}
	if i, err := strconv.ParseInt(ts, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(ts, 64); err == nil {
		return f, nil
	}
	if len(ts) > 0 {
		c := ts[0]
		if c == '{' || c == '[' {
			var v any
			if err := json.Unmarshal([]byte(ts), &v); err != nil {
				return nil, err
			}
			return v, nil
		}
	}
	return s, nil
}

func setAny(cur any, segs []segment, value any) (any, error) {
	if len(segs) == 0 {
		return value, nil
	}
	s := segs[0]

	if s.isIndex {
		a, ok := cur.([]any)
		if !ok {
			a = make([]any, 0, s.index+1)
		}
		if len(a) <= s.index {
			a = append(a, make([]any, s.index-len(a)+1)...)
		}
		child := a[s.index]
		newChild, err := setAny(child, segs[1:], value)
		if err != nil {
			return nil, err
		}
		a[s.index] = newChild
		return a, nil
	}

	m, ok := cur.(map[string]any)
	if !ok {
		m = map[string]any{}
	}
	child, _ := m[s.key]
	newChild, err := setAny(child, segs[1:], value)
	if err != nil {
		return nil, err
	}
	m[s.key] = newChild
	return m, nil
}
