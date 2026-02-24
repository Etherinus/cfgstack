package schema

import (
	"errors"
	"strings"

	"github.com/etherinus/cfgstack/internal/errx"
	"github.com/etherinus/cfgstack/internal/inspect"
	"github.com/etherinus/cfgstack/internal/prov"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Validator struct {
	SchemaRef  string
	StrictRefs bool
	Provenance *prov.Map
}

func (v Validator) Validate(data any) error {
	if strings.TrimSpace(v.SchemaRef) == "" {
		return nil
	}

	sch, compileTarget, err := compileCached(v.SchemaRef, v.StrictRefs)
	if err != nil {
		return err
	}

	if err := sch.Validate(data); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			loc := instanceLocationToPointer(ve.InstanceLocation)

			msg := formatValidationError(ve)
			ctx := inspect.FormatContext(data, loc)

			srcLine := ""
			if v.Provenance != nil {
				if src, ok := v.Provenance.LookupNearest(loc); ok {
					srcLine = "source: " + src.Layer + " " + src.File
				}
			}

			full := msg + "\n" + ctx
			if srcLine != "" {
				full = full + "\n" + srcLine
			}

			return &errx.Err{Op: "schema", File: compileTarget, Key: loc, Msg: full}
		}
		return errx.Wrap("schema", compileTarget, "", "schema validation failed", err)
	}

	return nil
}

func formatValidationError(ve *jsonschema.ValidationError) string {
	lines := make([]string, 0, 8)
	addValidationLines(&lines, ve, 0)
	return strings.Join(lines, "\n")
}

func addValidationLines(out *[]string, ve *jsonschema.ValidationError, depth int) {
	indent := strings.Repeat("  ", depth)
	loc := instanceLocationToPointer(ve.InstanceLocation)
	*out = append(*out, indent+loc+": "+firstLine(ve.Error()))
	for _, c := range ve.Causes {
		addValidationLines(out, c, depth+1)
	}
}

func instanceLocationToPointer(loc []string) string {
	if len(loc) == 0 {
		return "/"
	}
	var b strings.Builder
	for _, tok := range loc {
		b.WriteByte('/')
		b.WriteString(prov.Escape(tok))
	}
	return b.String()
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "validation failed"
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
