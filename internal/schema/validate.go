package schema

import (
	"errors"
	"strings"

	"github.com/etherinus/cfg-fuse/internal/errx"
	"github.com/etherinus/cfg-fuse/internal/inspect"
	"github.com/etherinus/cfg-fuse/internal/prov"
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
			loc := ve.InstanceLocation
			if loc == "" {
				loc = "/"
			}

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
	loc := ve.InstanceLocation
	if loc == "" {
		loc = "/"
	}
	*out = append(*out, indent+loc+": "+ve.Message)
	for _, c := range ve.Causes {
		addValidationLines(out, c, depth+1)
	}
}
