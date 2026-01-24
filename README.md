# cfgstack

`cfgstack` is a small Go CLI that builds a final configuration by merging file-based layers (JSON/YAML/TOML) and applying environment-variable overrides.

Use case: services, bots, games, CLIs, anything you deploy across environments.

## What you get

* Layered configs from a directory:

  * `default.*`
  * `local.*` (typically not committed)
  * `<profile>.*` (for example `prod.*`, `dev.*`)
* Supported formats: **JSON**, **YAML**, **TOML**
* Predictable deep merge rules for objects
* Env overrides with hierarchical paths:

  * `APP__DB__HOST=localhost` -> `{ "db": { "host": "localhost" } }`
* Array indices via env paths:

  * `APP__SERVERS__0__HOST=...`
* Optional JSON Schema validation (local path, `file://`, `http(s)://`)
* Good error messages:

  * JSON Pointer location
  * value context (value + parent info)
  * nearest source (which file or env var last touched the pointer)
* Debug tooling:

  * `explain` shows value + provenance for a pointer
  * `doctor` scans your config directory and reports conflicts and issues

## Installation

### Install from source

```bash
go install github.com/etherinus/cfgstack/cmd/cfgstack@latest
```

Make sure `$GOBIN` (or `$GOPATH/bin`) is on your `PATH`.

### Build locally

```bash
go build -o cfgstack ./cmd/cfgstack
```

## Quick start

Example `config/` structure:

```text
config/
  default.yaml
  local.yaml
  prod.yaml
  dev.yaml
```

Build merged config for prod:

```bash
cfgstack build --in config --profile prod --out merged.json
```

Write to stdout:

```bash
cfgstack build --in config --profile prod --out - --format json
```

Apply env overrides:

```bash
export APP__DB__HOST=localhost
export APP__DB__PORT=5432
cfgstack build --in config --profile prod --out merged.json
```

## Layering model

### Layer order

1. `default.*`
2. `local.*`
3. `<profile>.*` (skipped if profile is empty and `--allow-empty-profile` is enabled)
4. env overrides

If a layer matches multiple files, they are applied in **lexicographic filename order**.

### Merge rules

* object + object: recursive merge
* arrays: replaced as a whole by later layer
* scalars (string/bool/number/null): replaced by later layer
* type mismatch: later layer wins

This is intentionally simple and predictable (arrays are not "smart merged").

## Env overrides

### Prefix and delimiter

Only variables with `<ENV_PREFIX><DELIM>` prefix are used.

Defaults:

* `--env-prefix APP`
* `--env-delim __`

Example:

* `APP__A__B=1` -> `{ "a": { "b": 1 } }` (with `--env-case lower`)

### Arrays by index

If a path segment is all digits, it is treated as an array index:

* `APP__A__B__0=1` -> `{ "a": { "b": [1] } }`
* `APP__A__B__0__X=1` -> `{ "a": { "b": [ { "x": 1 } ] } }`

### Key casing

* `--env-case lower` (default): env keys are lowercased
* `--env-case keep`: env keys are kept as-is

### Value typing

Env values are parsed as:

* `true/false/null` -> bool/null
* integer -> int64
* float -> float64
* if the trimmed value starts with `{` or `[` -> attempt JSON parse
* otherwise -> string

## JSON Schema validation

Enable with `--schema`:

```bash
cfgstack build --in config --profile prod --out merged.json --schema schema.json
```

Schema references:

* local path: `schema.json`
* file URL: `file:///home/me/schema.json`
* HTTP(S): `https://example.com/schema.json`

`--schema-strict` enables stricter compiler assertions.

On validation failure, the error includes:

* JSON Pointer (instance location)
* nested validation causes
* context block (value + parent info)
* nearest provenance source (file/env)

Schema compilation is cached within the process:

* local files: abs path + mtime + size + strict mode
* URLs: URL string + strict mode

## Commands

### `cfgstack build`

Build the final config.

Examples:

```bash
cfgstack build --in config --profile prod --out merged.json
cfgstack build --in config --profile prod --out - --format yaml
cfgstack build --in config --allow-empty-profile --out merged.json
cfgstack build --in config --profile prod --fail-on-missing-default --out merged.json
```

Notable flags:

* `--fail-on-missing-default` - fail if no `default.*` files are found
* `--allow-empty-profile` - allow empty profile and skip `<profile>.*`
* `--print-sources` - print provenance map (pointer -> source) as JSON to stdout

Provenance dump:

```bash
cfgstack build --in config --profile prod --out merged.json --print-sources > sources.json
```

Note: when `--print-sources` is set, the status line (`ok: ...`) is printed to stderr so stdout remains valid JSON.

### `cfgstack explain`

Inspect a JSON Pointer in the merged config.

Examples:

```bash
cfgstack explain --in config --profile prod --at /db/host
cfgstack explain --in config --profile prod --at / --sources
cfgstack explain --in config --profile prod --at /db/host --json
cfgstack explain --in config --profile prod --at / --json --sources
```

* `--json` prints structured JSON output.
* `--sources` prints nearest sources for direct children at the pointer:

  * for `/` it lists top-level keys
  * for an object/array node it lists its direct children

### `cfgstack doctor`

Diagnose a config directory.

What it checks:

* directory exists and is readable
* which files are found per layer
* unsupported extensions per layer
* applied order
* pointer-level overrides (conflicts): where the same pointer was set by multiple files

Example:

```bash
cfgstack doctor --in config --profile prod --fail-on-missing-default --max-conflicts 100
```

`doctor` exits non-zero if:

* conflicts are found
* `--fail-on-missing-default` is set and no `default.*` exists

## Exit codes

* `0` - success
* `1` - runtime error (I/O, parse, schema validation, doctor found issues)
* `2` - CLI argument error

## Repository layout

Typical Go CLI structure:

```text
cmd/cfgstack/main.go - entrypoint
internal/cli         - flags and command orchestration
internal/config      - layer discovery and config parsing
internal/env         - env overrides
internal/merge       - deep merge
internal/output      - encoding and writing
internal/schema      - schema validation and caching
internal/inspect     - JSON Pointer utilities and context
internal/prov        - provenance map and history
internal/doctor      - diagnostics and conflict reporting
internal/errx        - unified error formatting
```

## Notes and gotchas

* Arrays are replaced, not merged. If you need additive semantics, model it as objects or use explicit indices from env.
* If you use `--env-case lower`, ensure your schema and file keys match the expected casing.
* If you want deterministic env application order, `cfgstack` sorts the environment variable list before applying.
