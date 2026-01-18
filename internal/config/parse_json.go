package config

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/etherinus/cfg-fuse/internal/errx"
)

func parseJSON(path string, b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	dec.DisallowUnknownFields()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, errx.Wrap("parse", path, "", "invalid JSON", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		if err == nil {
			return nil, &errx.Err{Op: "parse", File: path, Msg: "invalid JSON: extra content"}
		}
		return nil, errx.Wrap("parse", path, "", "invalid JSON", err)
	}
	nv, err := normalizeAny(v, path)
	if err != nil {
		return nil, errx.Wrap("parse", path, "", "invalid JSON", err)
	}
	return nv, nil
}
