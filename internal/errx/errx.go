package errx

import (
	"errors"
	"fmt"
)

type Err struct {
	Op   string
	File string
	Key  string
	Msg  string
	Err  error
}

func (e *Err) Error() string {
	base := e.Msg
	if base == "" && e.Err != nil {
		base = e.Err.Error()
	}
	prefix := ""
	if e.Op != "" {
		prefix = e.Op + ": "
	}
	ctx := ""
	if e.File != "" {
		ctx = e.File
	}
	if e.Key != "" {
		if ctx != "" {
			ctx = ctx + ":" + e.Key
		} else {
			ctx = e.Key
		}
	}
	if ctx != "" {
		return fmt.Sprintf("%s%s: %s", prefix, ctx, base)
	}
	return prefix + base
}

func (e *Err) Unwrap() error {
	return e.Err
}

func Wrap(op, file, key, msg string, err error) error {
	if err == nil {
		return &Err{Op: op, File: file, Key: key, Msg: msg}
	}
	var ex *Err
	if errors.As(err, &ex) {
		if file == "" {
			file = ex.File
		}
		if key == "" {
			key = ex.Key
		}
		if msg == "" {
			msg = ex.Msg
		}
		return &Err{Op: op, File: file, Key: key, Msg: msg, Err: err}
	}
	return &Err{Op: op, File: file, Key: key, Msg: msg, Err: err}
}
