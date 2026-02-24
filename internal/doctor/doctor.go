package doctor

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/etherinus/cfgstack/internal/config"
	"github.com/etherinus/cfgstack/internal/inspect"
	"github.com/etherinus/cfgstack/internal/prov"
)

type Report struct {
	Scan               config.ScanResult
	Sources            []config.Source
	Config             map[string]any
	Provenance         *prov.Map
	MaxConflicts       int
	FailMissingDefault bool
}

type JSONConflict struct {
	Ptr   string        `json:"ptr"`
	Value any           `json:"value"`
	Chain []prov.Source `json:"chain"`
}

type JSONReport struct {
	Scan               config.ScanResult `json:"scan"`
	Sources            []config.Source   `json:"sources"`
	ConflictCount      int               `json:"conflict_count"`
	ShownConflictCount int               `json:"shown_conflict_count"`
	Conflicts          []JSONConflict    `json:"conflicts"`
	ConflictsTruncated bool              `json:"conflicts_truncated"`
	MissingDefault     bool              `json:"missing_default"`
	Ok                 bool              `json:"ok"`
}

func (r Report) Print(w io.Writer) bool {
	ok := true

	fmt.Fprintf(w, "dir: %s\n", r.Scan.Dir)
	if r.Scan.Profile == "" {
		fmt.Fprintln(w, "profile: <empty>")
	} else {
		fmt.Fprintf(w, "profile: %s\n", r.Scan.Profile)
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "layers:")

	for _, li := range r.Scan.Layers {
		sup := r.Scan.Supported[li]
		uns := r.Scan.Unsupported[li]

		fmt.Fprintf(w, "  %s:\n", li)
		if len(sup) == 0 && len(uns) == 0 {
			fmt.Fprintln(w, "    supported: <none>")
			fmt.Fprintln(w, "    unsupported: <none>")
			if li == "default" && r.FailMissingDefault {
				ok = false
			}
			continue
		}

		if len(sup) == 0 {
			fmt.Fprintln(w, "    supported: <none>")
		} else {
			fmt.Fprintln(w, "    supported:")
			for _, f := range sup {
				fmt.Fprintf(w, "      - %s\n", filepath.Base(f))
			}
		}

		if len(uns) == 0 {
			fmt.Fprintln(w, "    unsupported: <none>")
		} else {
			fmt.Fprintln(w, "    unsupported:")
			for _, f := range uns {
				fmt.Fprintf(w, "      - %s\n", filepath.Base(f))
			}
		}
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "applied order:")
	if len(r.Sources) == 0 {
		fmt.Fprintln(w, "  <none>")
	} else {
		for i, s := range r.Sources {
			fmt.Fprintf(w, "  [%d] %s: %s\n", i+1, s.Layer, filepath.Base(s.File))
		}
	}

	conflicts := r.allConflicts()

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "conflicts: %d\n", len(conflicts))

	if len(conflicts) > 0 {
		ok = false
	}

	limit := r.limitFor(len(conflicts))

	for i := 0; i < limit; i++ {
		c := conflicts[i]

		val, _ := inspect.ValueAtPointer(r.Config, c.Ptr)
		v := inspect.CompactJSON(val)
		v = trimString(v, 160)

		fmt.Fprintf(w, "\n[%d] %s\n", i+1, c.Ptr)
		fmt.Fprintf(w, "  value: %s\n", v)
		fmt.Fprintln(w, "  chain:")
		for _, s := range c.Sources {
			if s.File == "" {
				fmt.Fprintf(w, "    - %s\n", s.Layer)
			} else {
				fmt.Fprintf(w, "    - %s %s\n", s.Layer, s.File)
			}
		}
	}

	if len(conflicts) > limit {
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "conflicts truncated: showing %d of %d (use --max-conflicts)\n", limit, len(conflicts))
	}

	if r.FailMissingDefault {
		if len(r.Scan.Supported["default"]) == 0 {
			fmt.Fprintln(w, "")
			fmt.Fprintln(w, "error: missing default.* and --fail-on-missing-default is set")
			ok = false
		}
	}

	return ok
}

func trimString(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func (r Report) JSON() JSONReport {
	conflicts := r.allConflicts()
	limit := r.limitFor(len(conflicts))
	shown := conflicts[:limit]

	outConflicts := make([]JSONConflict, 0, len(shown))
	for _, c := range shown {
		val, _ := inspect.ValueAtPointer(r.Config, c.Ptr)
		outConflicts = append(outConflicts, JSONConflict{
			Ptr:   c.Ptr,
			Value: val,
			Chain: c.Sources,
		})
	}

	missingDefault := r.FailMissingDefault && len(r.Scan.Supported["default"]) == 0
	ok := len(conflicts) == 0 && !missingDefault

	return JSONReport{
		Scan:               r.Scan,
		Sources:            r.Sources,
		ConflictCount:      len(conflicts),
		ShownConflictCount: len(shown),
		Conflicts:          outConflicts,
		ConflictsTruncated: len(conflicts) > len(shown),
		MissingDefault:     missingDefault,
		Ok:                 ok,
	}
}

func (r Report) allConflicts() []prov.HistoryEntry {
	if r.Provenance == nil {
		return nil
	}
	return r.Provenance.HistoryConflictsSorted()
}

func (r Report) limitFor(total int) int {
	limit := r.MaxConflicts
	if limit < 0 {
		limit = total
	}
	if limit > total {
		limit = total
	}
	if limit < 0 {
		return 0
	}
	return limit
}
