package env

import (
	"strings"

	"github.com/etherinus/cfg-fuse/internal/errx"
)

type CaseMode int

const (
	CaseLower CaseMode = iota
	CaseKeep
)

func ParseCaseMode(s string) (CaseMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "lower":
		return CaseLower, nil
	case "keep":
		return CaseKeep, nil
	default:
		return CaseLower, &errx.Err{Op: "env", Msg: "invalid --env-case, expected lower|keep"}
	}
}
