package schema

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

func formatContext(data any, ptr string) string {
	val, ok := valueAtPointer(data, ptr)
	if !ok {
		return "context:\n  value: <not found>"
	}

	valStr := compactJSON(val)

	parent, token, pok := parentAtPointer(data, ptr)
	if !pok {
		return "context:\n  value: " + valStr
	}

	switch t := parent.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		return "context:\n  value: " + valStr + "\n  parent: object\n  key: " + token + "\n  keys: [" + strings.Join(keys, ", ") + "]"
	case []any:
		idx, _ := strconv.Atoi(token)
		return "context:\n  value: " + valStr + "\n  parent: array\n  index: " + strconv.Itoa(idx) + "\n  len: " + strconv.Itoa(len(t))
	default:
		return "context:\n  value: " + valStr + "\n  parent: " + typeName(parent)
	}
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

func compactJSON(v any) string {
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

func valueAtPointer(root any, ptr string) (any, bool) {
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

func parentAtPointer(root any, ptr string) (any, string, bool) {
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
	parent, ok := valueAtPointer(root, parentPtr)
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
