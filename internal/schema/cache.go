package schema

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/etherinus/cfgstack/internal/errx"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type keyKind int

const (
	keyFile keyKind = iota
	keyURL
)

type cacheKey struct {
	kind   keyKind
	ref    string
	strict bool
	mtime  int64
	size   int64
}

var (
	cacheMu sync.Mutex
	cache   = map[cacheKey]*jsonschema.Schema{}
)

func compileCached(schemaRef string, strict bool) (*jsonschema.Schema, string, error) {
	kind, ref, compileTarget, mtime, size, err := normalizeSchemaRef(schemaRef)
	if err != nil {
		return nil, "", err
	}

	key := cacheKey{
		kind:   kind,
		ref:    ref,
		strict: strict,
		mtime:  mtime,
		size:   size,
	}

	cacheMu.Lock()
	if s, ok := cache[key]; ok {
		cacheMu.Unlock()
		return s, compileTarget, nil
	}
	cacheMu.Unlock()

	comp := jsonschema.NewCompiler()
	if strict {
		comp.AssertFormat()
		comp.AssertContent()
		comp.AssertVocabs()
	}

	s, err := comp.Compile(compileTarget)
	if err != nil {
		return nil, compileTarget, errx.Wrap("schema", compileTarget, "", "failed to compile JSON Schema", err)
	}

	cacheMu.Lock()
	cache[key] = s
	cacheMu.Unlock()

	return s, compileTarget, nil
}

func normalizeSchemaRef(schemaRef string) (keyKind, string, string, int64, int64, error) {
	ref := strings.TrimSpace(schemaRef)
	if ref == "" {
		return keyFile, "", "", 0, 0, &errx.Err{Op: "schema", Msg: "empty schema reference"}
	}

	u, err := url.Parse(ref)
	if err == nil && u.Scheme != "" {
		switch strings.ToLower(u.Scheme) {
		case "file":
			p, err := fileURLToPath(u)
			if err != nil {
				return keyFile, "", "", 0, 0, err
			}
			abs, err := filepath.Abs(p)
			if err != nil {
				abs = p
			}
			mtime, size, err := statKey(abs)
			if err != nil {
				return keyFile, abs, abs, 0, 0, err
			}
			return keyFile, abs, abs, mtime, size, nil
		case "http", "https":
			return keyURL, ref, ref, 0, 0, nil
		default:
			return keyURL, ref, ref, 0, 0, nil
		}
	}

	abs, err := filepath.Abs(ref)
	if err != nil {
		abs = ref
	}
	mtime, size, err := statKey(abs)
	if err != nil {
		return keyFile, abs, abs, 0, 0, err
	}
	return keyFile, abs, abs, mtime, size, nil
}

func statKey(path string) (int64, int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, 0, errx.Wrap("schema", path, "", "failed to stat schema file", err)
	}
	mtime := fi.ModTime()
	if mtime.IsZero() {
		mtime = time.Unix(0, 0)
	}
	return mtime.UnixNano(), fi.Size(), nil
}

func fileURLToPath(u *url.URL) (string, error) {
	p := u.Path
	if p == "" {
		return "", &errx.Err{Op: "schema", Msg: "invalid file URL"}
	}
	p = filepath.FromSlash(p)
	if u.Host != "" {
		return `\\` + u.Host + p, nil
	}
	return p, nil
}
