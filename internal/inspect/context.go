package inspect

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

type Ctx struct {
	ValueFound  bool     `json:"value_found"`
	ValueType   string   `json:"value_type"`
	ParentFound bool     `json:"parent_found"`
	ParentType  string   `json:"parent_type,omitempty"`
	Key         string   `json:"key,omitempty"`
	Keys        []string `json:"keys,omitempty"`
	Index       *int     `json:"index,omitempty"`
	Len         *int     `json:"len,omitempty"`
}

func CompactJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "<unmarshalable>"
	}
	var out bytes.Buffer
	if err := json.Compact(&out, b); err != nil {
		return string(b)
	}
	return out.String()
}

func FormatContext(data any, ptr string) string {
	val, ok := ValueAtPointer(data, ptr)
	if !ok {
		return "context:\n  value: <not found>"
	}

	valStr := CompactJSON(val)
	parent, token, pok := ParentAtPointer(data, ptr)
	if !pok {
		return "context:\n  value: " + valStr
	}

	switch t := parent.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return "context:\n  value: " + valStr + "\n  parent: object\n  key: " + token + "\n  keys: [" + strings.Join(keys, ", ") + "]"
	case []any:
		idx, _ := strconv.Atoi(token)
		return "context:\n  value: " + valStr + "\n  parent: array\n  index: " + strconv.Itoa(idx) + "\n  len: " + strconv.Itoa(len(t))
	default:
		return "context:\n  value: " + valStr + "\n  parent: " + typeName(parent)
	}
}

func ContextAtPointer(data any, ptr string) Ctx {
	val, ok := ValueAtPointer(data, ptr)
	ctx := Ctx{
		ValueFound: ok,
		ValueType:  typeName(val),
	}
	parent, token, pok := ParentAtPointer(data, ptr)
	ctx.ParentFound = pok
	if !pok {
		return ctx
	}
	ctx.ParentType = typeName(parent)

	switch t := parent.(type) {
	case map[string]any:
		ctx.Key = token
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ctx.Keys = keys
	case []any:
		idx, err := strconv.Atoi(token)
		if err == nil {
			ctx.Index = &idx
		}
		l := len(t)
		ctx.Len = &l
	default:
	}
	return ctx
}

func ChildPointers(root any, ptr string) ([]string, bool) {
	node, ok := ValueAtPointer(root, ptr)
	if !ok {
		return nil, false
	}
	if ptr == "" {
		ptr = "/"
	}
	if ptr != "/" && strings.HasSuffix(ptr, "/") {
		ptr = strings.TrimSuffix(ptr, "/")
	}

	switch t := node.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]string, 0, len(keys))
		for _, k := range keys {
			out = append(out, ptr+"/"+escapePointerToken(k))
		}
		return out, true
	case []any:
		out := make([]string, 0, len(t))
		for i := 0; i < len(t); i++ {
			out = append(out, ptr+"/"+strconv.Itoa(i))
		}
		return out, true
	default:
		return nil, true
	}
}

func ValueAtPointer(root any, ptr string) (any, bool) {
	if ptr == "" || ptr == "/" {
		return root, true
	}
	if !strings.HasPrefix(ptr, "/") {
		return nil, false
	}
	tokens := strings.Split(ptr, "/")[1:]
	cur := root
	for _, tok := range tokens {
		tok = unescapePointerToken(tok)
		switch t := cur.(type) {
		case map[string]any:
			v, ok := t[tok]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			if tok == "" || !isDigits(tok) {
				return nil, false
			}
			i, err := strconv.Atoi(tok)
			if err != nil || i < 0 || i >= len(t) {
				return nil, false
			}
			cur = t[i]
		default:
			return nil, false
		}
	}
	return cur, true
}

func ParentAtPointer(root any, ptr string) (any, string, bool) {
	if ptr == "" || ptr == "/" {
		return nil, "", false
	}
	if !strings.HasPrefix(ptr, "/") {
		return nil, "", false
	}
	tokens := strings.Split(ptr, "/")[1:]
	if len(tokens) == 0 {
		return nil, "", false
	}
	parentPtr := "/" + strings.Join(tokens[:len(tokens)-1], "/")
	parent, ok := ValueAtPointer(root, parentPtr)
	if !ok {
		return nil, "", false
	}
	token := unescapePointerToken(tokens[len(tokens)-1])
	return parent, token, true
}

func unescapePointerToken(s string) string {
	if strings.IndexByte(s, '~') == -1 {
		return s
	}
	s = strings.ReplaceAll(s, "~1", "/")
	s = strings.ReplaceAll(s, "~0", "~")
	return s
}

func escapePointerToken(s string) string {
	if strings.IndexByte(s, '~') == -1 && strings.IndexByte(s, '/') == -1 {
		return s
	}
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
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

func typeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case int64:
		return "int"
	case float64:
		return "float"
	case string:
		return "string"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return "unknown"
	}
}
