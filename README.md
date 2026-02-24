# cfgstack

`cfgstack` is a production-focused CLI for building a final configuration from layered files and environment variable overrides.

I built it to solve a common problem in real systems: config drift across `dev`, `staging`, and `prod`, plus hard-to-debug implicit overrides.

## Why this exists

The goals are straightforward:

- deterministic behavior in local runs, CI, and production;
- explicit merge rules (no hidden magic);
- fast root-cause analysis when config is wrong.

`cfgstack` is designed around those constraints.

## Who this is for

This tool is useful for teams that:

- run the same service across multiple environments;
- need profile-based config (`prod`, `dev`, etc.);
- rely on env overrides but still want control and traceability;
- require schema validation and strong diagnostics.

Typical use cases: backend services, workers, bots, game servers, internal CLIs.

## What cfgstack does

- Loads layered config files:
  - `default.*`
  - `local.*`
  - `<profile>.*` (for example `prod.*`, `dev.*`)
- Supports `JSON`, `YAML`, `TOML`.
- Deep-merges objects using deterministic rules.
- Applies hierarchical env overrides.
- Tracks provenance (the last source that wrote each JSON Pointer).
- Validates the final config against JSON Schema.
- Provides debugging commands:
  - `explain` for pointer-level inspection;
  - `doctor` for directory diagnostics and conflict analysis.

## How it works

### Layer order

Layers are always applied in this order:

1. `default.*`
2. `local.*`
3. `<profile>.*`
4. env overrides

If multiple files match a layer, files are applied in lexicographic filename order.

### Merge semantics

- object + object: recursive merge
- arrays: full replacement by later layer
- scalars (`string`, `bool`, `number`, `null`): replacement by later layer
- type mismatch: later layer wins

This behavior is intentionally simple and predictable.

### Environment overrides

Defaults:

- `--env-prefix APP`
- `--env-delim __`
- `--env-case lower`

Examples:

- `APP__DB__HOST=localhost` -> `{ "db": { "host": "localhost" } }`
- `APP__SERVERS__0__HOST=10.0.0.1` -> `{ "servers": [ { "host": "10.0.0.1" } ] }`

Value typing:

- `true/false/null` -> bool/null
- integer -> `int64`
- float -> `float64`
- values starting with `{` or `[` -> JSON parse
- otherwise -> string

## Installation

### Install from source

```bash
go install github.com/etherinus/cfgstack/cmd/cfgstack@latest
```

### Build locally

```bash
go build -o cfgstack ./cmd/cfgstack
```

Build with metadata:

```bash
go build -ldflags "-X github.com/etherinus/cfgstack/internal/buildinfo.Version=v0.1.0 -X github.com/etherinus/cfgstack/internal/buildinfo.Commit=$(git rev-parse HEAD) -X github.com/etherinus/cfgstack/internal/buildinfo.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o cfgstack ./cmd/cfgstack
```

## Quick start

Example layout:

```text
config/
  default.yaml
  local.yaml
  prod.yaml
  dev.yaml
```

Build merged config:

```bash
cfgstack build --in config --profile prod --out merged.json
```

Write to stdout:

```bash
cfgstack build --in config --profile prod --out - --format json
```

Dry run without writing:

```bash
cfgstack build --in config --allow-empty-profile --out merged.json --dry-run
```

Dry run with pointer-level diff against existing output file:

```bash
cfgstack build --in config --allow-empty-profile --out merged.json --dry-run --diff
```

## Commands

### `cfgstack build`

Builds final config from file layers and env overrides.

Important flags:

- `--profile` is required unless `--allow-empty-profile` is set.
- `--fail-on-missing-default` fails when no `default.*` files are found.
- `--print-sources` outputs provenance map JSON.
- `--dry-run` computes result without writing output file.
- `--diff` (with `--dry-run`) prints pointer-level diff vs current `--out`.
- `--schema` and `--schema-strict` enable JSON Schema validation.

Notes:

- `--print-sources` requires `--out` not `-`.
- `--diff` requires `--dry-run` and `--out` not `-`.

### `cfgstack explain`

Inspect a JSON Pointer in resolved config:

```bash
cfgstack explain --in config --profile prod --at /db/host
cfgstack explain --in config --profile prod --at / --sources
cfgstack explain --in config --profile prod --at /db/host --json
```

Includes:

- value;
- local context (parent/type/key/index);
- nearest provenance source;
- optional child sources (`--sources`).

### `cfgstack doctor`

Diagnose config directory health:

```bash
cfgstack doctor --in config --profile prod --fail-on-missing-default --max-conflicts 100
cfgstack doctor --in config --allow-empty-profile --json
```

Checks:

- discovered files per layer;
- supported vs unsupported extensions;
- effective apply order;
- pointer-level override conflicts.

Exit behavior:

- non-zero if conflicts exist;
- non-zero if `--fail-on-missing-default` and no `default.*`.

`--json` returns structured scan/sources/conflict data and final `ok` status.

### `cfgstack version`

```bash
cfgstack version
cfgstack --version
cfgstack version --json
```

JSON fields:

- `name`
- `version`
- `commit`
- `date`

## JSON Schema validation

Enable validation:

```bash
cfgstack build --in config --profile prod --out merged.json --schema schema.json
```

Supported schema refs:

- local path: `schema.json`
- file URL: `file:///.../schema.json`
- HTTP(S): `https://...`

Validation errors include:

- JSON Pointer;
- nested causes;
- value context;
- nearest provenance source.

## Exit codes

- `0` success
- `1` runtime/validation/doctor failure
- `2` CLI argument error

## CI and releases

- CI: `.github/workflows/ci.yml`
  - `go mod tidy` check
  - `go vet ./...`
  - `go test ./...`
  - release-build smoke check
- Release: `.github/workflows/release.yml`
  - triggered by tags `v*`
  - builds Linux/macOS/Windows binaries
  - publishes artifacts to GitHub Release

## Repository layout

```text
cmd/cfgstack/main.go - entrypoint
internal/cli         - commands and flags
internal/config      - layer discovery and parsing
internal/env         - env override application
internal/merge       - deep merge
internal/output      - output read/write/encode
internal/schema      - schema compile/validate
internal/inspect     - JSON Pointer utilities and diff
internal/prov        - provenance map/history
internal/doctor      - diagnostics/reporting
internal/errx        - structured errors
```

## Final notes

Design priorities:

- deterministic behavior;
- explicit override semantics;
- traceability of final values.

If you need a controlled and reproducible configuration pipeline, `cfgstack` is built for exactly that.
