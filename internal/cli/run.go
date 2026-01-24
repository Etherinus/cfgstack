package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/etherinus/cfgstack/internal/config"
	"github.com/etherinus/cfgstack/internal/doctor"
	"github.com/etherinus/cfgstack/internal/env"
	"github.com/etherinus/cfgstack/internal/inspect"
	"github.com/etherinus/cfgstack/internal/output"
	"github.com/etherinus/cfgstack/internal/prov"
	"github.com/etherinus/cfgstack/internal/schema"
)

func Run(args []string) int {
	stdout := os.Stdout
	stderr := os.Stderr
	if len(args) < 2 {
		printUsage(stderr)
		return 2
	}

	cmd := args[1]
	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		printUsage(stdout)
		return 0
	}

	switch cmd {
	case "build":
		return runBuild(stdout, stderr, args[2:])
	case "explain":
		return runExplain(stdout, stderr, args[2:])
	case "doctor":
		return runDoctor(stdout, stderr, args[2:])
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", cmd)
		printUsage(stderr)
		return 2
	}
}

func runBuild(stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("cfgstack build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	inDir := fs.String("in", "config", "input config directory")
	profile := fs.String("profile", "", "profile name, for example prod or dev")
	allowEmptyProfile := fs.Bool("allow-empty-profile", false, "allow empty profile and skip <profile>.* layer")
	failMissingDefault := fs.Bool("fail-on-missing-default", false, "fail if no default.* files found")
	outPath := fs.String("out", "merged.json", "output path or '-' for stdout")
	outFormat := fs.String("format", "", "output format override: json|yaml|toml (useful with --out -)")
	envPrefix := fs.String("env-prefix", "APP", "environment variable prefix")
	envDelim := fs.String("env-delim", "__", "environment variable delimiter")
	envCase := fs.String("env-case", "lower", "env key casing: lower|keep")
	schemaPath := fs.String("schema", "", "path/URL to JSON Schema (optional): file path, file://, http(s)://")
	strictSchema := fs.Bool("schema-strict", false, "fail on unknown schema refs")
	verbose := fs.Bool("verbose", false, "print applied layers in order")
	printSources := fs.Bool("print-sources", false, "print provenance map as JSON to stdout (requires --out not '-')")
	showHelp := fs.Bool("h", false, "show help")
	showHelp2 := fs.Bool("help", false, "show help")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "%s\n\n", err.Error())
		printBuildUsage(stderr)
		return 2
	}
	if *showHelp || *showHelp2 {
		printBuildUsage(stdout)
		return 0
	}

	if strings.TrimSpace(*profile) == "" && !*allowEmptyProfile {
		fmt.Fprintln(stderr, "missing required flag: --profile (or pass --allow-empty-profile)")
		printBuildUsage(stderr)
		return 2
	}
	if strings.TrimSpace(*envPrefix) == "" {
		fmt.Fprintln(stderr, "--env-prefix cannot be empty")
		return 2
	}
	if *envDelim == "" {
		fmt.Fprintln(stderr, "--env-delim cannot be empty")
		return 2
	}
	caseMode, err := env.ParseCaseMode(*envCase)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", err.Error())
		return 2
	}
	if *printSources && *outPath == "-" {
		fmt.Fprintln(stderr, "--print-sources requires --out not '-' (stdout is used for sources JSON)")
		return 2
	}

	cfg, provMap, sources, err := config.LoadLayers(*inDir, *profile, *failMissingDefault, *allowEmptyProfile)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", formatErr(err))
		return 1
	}

	cfg, nEnv, err := env.Apply(cfg, *envPrefix, *envDelim, caseMode, provMap)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", formatErr(err))
		return 1
	}
	sources = append(sources, config.Source{Layer: "env", File: fmt.Sprintf("%s (%d vars)", *envPrefix, nEnv)})

	if *verbose {
		for i, s := range sources {
			fmt.Fprintf(stderr, "[%d] %s: %s\n", i+1, s.Layer, s.File)
		}
	}

	if strings.TrimSpace(*schemaPath) != "" {
		v := schema.Validator{
			SchemaRef:  *schemaPath,
			StrictRefs: *strictSchema,
			Provenance: provMap,
		}
		if err := v.Validate(cfg); err != nil {
			fmt.Fprintf(stderr, "%s\n", formatErr(err))
			return 1
		}
	}

	fmtStr, err := output.ResolveFormat(*outPath, *outFormat)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", formatErr(err))
		return 2
	}

	if *printSources {
		if err := writeSourcesJSON(stdout, provMap); err != nil {
			fmt.Fprintf(stderr, "%s\n", formatErr(err))
			return 1
		}
	}

	if *outPath == "-" {
		if err := output.WriteTo(stdout, fmtStr, cfg); err != nil {
			fmt.Fprintf(stderr, "%s\n", formatErr(err))
			return 1
		}
		return 0
	}

	if err := output.WriteFile(*outPath, fmtStr, cfg); err != nil {
		fmt.Fprintf(stderr, "%s\n", formatErr(err))
		return 1
	}

	if *printSources {
		fmt.Fprintf(stderr, "ok: %s\n", *outPath)
	} else {
		fmt.Fprintf(stdout, "ok: %s\n", *outPath)
	}
	return 0
}

type explainJSONOut struct {
	Pointer         string       `json:"pointer"`
	Found           bool         `json:"found"`
	Value           any          `json:"value"`
	Context         inspect.Ctx  `json:"context"`
	Source          *prov.Source `json:"source"`
	ChildrenSources []prov.Entry `json:"children_sources,omitempty"`
}

func runExplain(stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("cfgstack explain", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	inDir := fs.String("in", "config", "input config directory")
	profile := fs.String("profile", "", "profile name, for example prod or dev")
	allowEmptyProfile := fs.Bool("allow-empty-profile", false, "allow empty profile and skip <profile>.* layer")
	at := fs.String("at", "", "JSON Pointer to inspect, for example /db/host or /a/b/0/x (required)")
	envPrefix := fs.String("env-prefix", "APP", "environment variable prefix")
	envDelim := fs.String("env-delim", "__", "environment variable delimiter")
	envCase := fs.String("env-case", "lower", "env key casing: lower|keep")
	asJSON := fs.Bool("json", false, "print structured JSON output")
	showSources := fs.Bool("sources", false, "print nearest sources for direct children at pointer (root: top-level)")
	showHelp := fs.Bool("h", false, "show help")
	showHelp2 := fs.Bool("help", false, "show help")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "%s\n\n", err.Error())
		printExplainUsage(stderr)
		return 2
	}
	if *showHelp || *showHelp2 {
		printExplainUsage(stdout)
		return 0
	}

	if strings.TrimSpace(*profile) == "" && !*allowEmptyProfile {
		fmt.Fprintln(stderr, "missing required flag: --profile (or pass --allow-empty-profile)")
		printExplainUsage(stderr)
		return 2
	}
	if strings.TrimSpace(*at) == "" {
		fmt.Fprintln(stderr, "missing required flag: --at")
		printExplainUsage(stderr)
		return 2
	}
	if !strings.HasPrefix(*at, "/") && *at != "" {
		fmt.Fprintln(stderr, "--at must be a JSON Pointer starting with '/'")
		return 2
	}

	caseMode, err := env.ParseCaseMode(*envCase)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", err.Error())
		return 2
	}

	cfg, provMap, _, err := config.LoadLayers(*inDir, *profile, false, *allowEmptyProfile)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", formatErr(err))
		return 1
	}

	cfg, _, err = env.Apply(cfg, *envPrefix, *envDelim, caseMode, provMap)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", formatErr(err))
		return 1
	}

	ptr := *at
	if ptr == "" {
		ptr = "/"
	}

	val, ok := inspect.ValueAtPointer(cfg, ptr)
	ctx := inspect.ContextAtPointer(cfg, ptr)

	var srcPtr *prov.Source
	if s, ok2 := provMap.LookupNearest(ptr); ok2 {
		s2 := s
		srcPtr = &s2
	}

	childrenSources := []prov.Entry(nil)
	if *showSources {
		childrenSources = computeChildrenSources(cfg, ptr, provMap)
	}

	if *asJSON {
		out := explainJSONOut{
			Pointer:         ptr,
			Found:           ok,
			Value:           val,
			Context:         ctx,
			Source:          srcPtr,
			ChildrenSources: childrenSources,
		}
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "%s\n", err.Error())
			return 1
		}
		b = append(b, '\n')
		_, _ = stdout.Write(b)
		return 0
	}

	fmt.Fprintf(stdout, "pointer: %s\n", ptr)
	if !ok {
		fmt.Fprintln(stdout, "value: <not found>")
		if srcPtr != nil {
			fmt.Fprintf(stdout, "source: %s %s\n", srcPtr.Layer, srcPtr.File)
		} else {
			fmt.Fprintln(stdout, "source: <unknown>")
		}
		return 0
	}

	fmt.Fprintf(stdout, "value: %s\n", inspect.CompactJSON(val))
	fmt.Fprintln(stdout, inspect.FormatContext(cfg, ptr))

	if srcPtr != nil {
		fmt.Fprintf(stdout, "source: %s %s\n", srcPtr.Layer, srcPtr.File)
	} else {
		fmt.Fprintln(stdout, "source: <unknown>")
	}

	if *showSources {
		if len(childrenSources) == 0 {
			fmt.Fprintln(stdout, "children_sources: <none>")
		} else {
			fmt.Fprintln(stdout, "children_sources:")
			for _, e := range childrenSources {
				if e.Layer == "<unknown>" {
					fmt.Fprintf(stdout, "  %s: <unknown>\n", e.Ptr)
				} else {
					fmt.Fprintf(stdout, "  %s: %s %s\n", e.Ptr, e.Layer, e.File)
				}
			}
		}
	}

	return 0
}

func runDoctor(stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("cfgstack doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	inDir := fs.String("in", "config", "input config directory")
	profile := fs.String("profile", "", "profile name, for example prod or dev")
	allowEmptyProfile := fs.Bool("allow-empty-profile", false, "allow empty profile and skip <profile>.* layer")
	failMissingDefault := fs.Bool("fail-on-missing-default", false, "fail if no default.* files found")
	maxConflicts := fs.Int("max-conflicts", 50, "max number of conflicts to print")
	showHelp := fs.Bool("h", false, "show help")
	showHelp2 := fs.Bool("help", false, "show help")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "%s\n\n", err.Error())
		printDoctorUsage(stderr)
		return 2
	}
	if *showHelp || *showHelp2 {
		printDoctorUsage(stdout)
		return 0
	}

	if strings.TrimSpace(*profile) == "" && !*allowEmptyProfile {
		fmt.Fprintln(stderr, "missing required flag: --profile (or pass --allow-empty-profile)")
		printDoctorUsage(stderr)
		return 2
	}
	if *maxConflicts < 0 {
		fmt.Fprintln(stderr, "--max-conflicts must be >= 0")
		return 2
	}

	scan, err := config.ScanLayers(*inDir, *profile, *allowEmptyProfile)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", formatErr(err))
		return 1
	}

	cfg, provMap, sources, err := config.LoadLayers(*inDir, *profile, *failMissingDefault, *allowEmptyProfile)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", formatErr(err))
		return 1
	}

	rep := doctor.Report{
		Scan:               scan,
		Sources:            sources,
		Config:             cfg,
		Provenance:         provMap,
		MaxConflicts:       *maxConflicts,
		FailMissingDefault: *failMissingDefault,
	}
	ok := rep.Print(stdout)

	if !ok {
		return 1
	}
	return 0
}

func computeChildrenSources(cfg any, ptr string, p *prov.Map) []prov.Entry {
	children, ok := inspect.ChildPointers(cfg, ptr)
	if !ok || len(children) == 0 {
		return nil
	}
	out := make([]prov.Entry, 0, len(children))
	for _, cptr := range children {
		if s, ok := p.LookupNearest(cptr); ok {
			out = append(out, prov.Entry{Ptr: cptr, Layer: s.Layer, File: s.File})
		} else {
			out = append(out, prov.Entry{Ptr: cptr, Layer: "<unknown>", File: ""})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Ptr < out[j].Ptr
	})
	return out
}

func writeSourcesJSON(w io.Writer, p *prov.Map) error {
	entries := p.EntriesSorted()
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "cfgstack")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  cfgstack build --in config/ --profile prod --out merged.json")
	fmt.Fprintln(w, "  cfgstack explain --in config/ --profile prod --at /db/host")
	fmt.Fprintln(w, "  cfgstack doctor --in config/ --profile prod")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  build      merge config layers and apply env overrides")
	fmt.Fprintln(w, "  explain    inspect value, context, and provenance at JSON Pointer")
	fmt.Fprintln(w, "  doctor     check config directory and report layer conflicts")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Run 'cfgstack <cmd> --help' for options")
}

func printBuildUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  cfgstack build --in config/ --profile prod --out merged.json")
	fmt.Fprintln(w, "  cfgstack build --in config/ --allow-empty-profile --out merged.json")
	fmt.Fprintln(w, "  cfgstack build --in config/ --profile prod --out - --format json")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Layers, in order:")
	fmt.Fprintln(w, "  default.*")
	fmt.Fprintln(w, "  local.*")
	fmt.Fprintln(w, "  <profile>.* (skipped if profile empty and --allow-empty-profile)")
	fmt.Fprintln(w, "  env vars: <ENV_PREFIX><DELIM>path")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --in                      input config directory (default: config)")
	fmt.Fprintln(w, "  --profile                 profile name, for example prod or dev")
	fmt.Fprintln(w, "  --allow-empty-profile     allow empty profile and skip <profile>.* layer")
	fmt.Fprintln(w, "  --fail-on-missing-default fail if no default.* files found")
	fmt.Fprintln(w, "  --out                     output path or '-' for stdout")
	fmt.Fprintln(w, "  --format                  output format override: json|yaml|toml (useful with --out -)")
	fmt.Fprintln(w, "  --env-prefix              env var prefix (default: APP)")
	fmt.Fprintln(w, "  --env-delim               env var delimiter (default: __)")
	fmt.Fprintln(w, "  --env-case                env key casing: lower|keep (default: lower)")
	fmt.Fprintln(w, "  --schema                  path/URL to JSON Schema (optional): file path, file://, http(s)://")
	fmt.Fprintln(w, "  --schema-strict           fail on unknown schema refs")
	fmt.Fprintln(w, "  --verbose                 print applied layers in order")
	fmt.Fprintln(w, "  --print-sources           print provenance map JSON to stdout (requires --out not '-')")
}

func printExplainUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  cfgstack explain --in config/ --profile prod --at /a/b")
	fmt.Fprintln(w, "  cfgstack explain --in config/ --allow-empty-profile --at /")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --in                  input config directory (default: config)")
	fmt.Fprintln(w, "  --profile             profile name, for example prod or dev")
	fmt.Fprintln(w, "  --allow-empty-profile allow empty profile and skip <profile>.* layer")
	fmt.Fprintln(w, "  --at                  JSON Pointer to inspect (required)")
	fmt.Fprintln(w, "  --json                print structured JSON output")
	fmt.Fprintln(w, "  --sources             print nearest sources for direct children at pointer (root: top-level)")
	fmt.Fprintln(w, "  --env-prefix          env var prefix (default: APP)")
	fmt.Fprintln(w, "  --env-delim           env var delimiter (default: __)")
	fmt.Fprintln(w, "  --env-case            env key casing: lower|keep (default: lower)")
}

func printDoctorUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  cfgstack doctor --in config/ --profile prod")
	fmt.Fprintln(w, "  cfgstack doctor --in config/ --allow-empty-profile")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --in                      input config directory (default: config)")
	fmt.Fprintln(w, "  --profile                 profile name, for example prod or dev")
	fmt.Fprintln(w, "  --allow-empty-profile     allow empty profile and skip <profile>.* layer")
	fmt.Fprintln(w, "  --fail-on-missing-default fail if no default.* files found")
	fmt.Fprintln(w, "  --max-conflicts           max number of conflicts to print (default: 50)")
}
