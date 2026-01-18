package prov

import (
	"sort"
	"strings"
)

type Source struct {
	Layer string `json:"layer"`
	File  string `json:"file"`
}

type Entry struct {
	Ptr   string `json:"ptr"`
	Layer string `json:"layer"`
	File  string `json:"file"`
}

type HistoryEntry struct {
	Ptr     string   `json:"ptr"`
	Sources []Source `json:"sources"`
}

type Map struct {
	last map[string]Source
	hist map[string][]Source
}

func New() *Map {
	return &Map{
		last: map[string]Source{},
		hist: map[string][]Source{},
	}
}

func (p *Map) Set(ptr string, src Source) {
	if p == nil {
		return
	}
	if ptr == "" {
		ptr = "/"
	}
	p.last[ptr] = src

	seq := p.hist[ptr]
	if len(seq) == 0 {
		p.hist[ptr] = []Source{src}
		return
	}
	prev := seq[len(seq)-1]
	if prev.Layer == src.Layer && prev.File == src.File {
		return
	}
	p.hist[ptr] = append(seq, src)
}

func (p *Map) LookupNearest(ptr string) (Source, bool) {
	if p == nil {
		return Source{}, false
	}
	if ptr == "" {
		ptr = "/"
	}
	if s, ok := p.last[ptr]; ok {
		return s, true
	}
	if ptr == "/" {
		return Source{}, false
	}

	for {
		i := strings.LastIndex(ptr, "/")
		if i <= 0 {
			ptr = "/"
		} else {
			ptr = ptr[:i]
		}
		if s, ok := p.last[ptr]; ok {
			return s, true
		}
		if ptr == "/" {
			break
		}
	}
	return Source{}, false
}

func (p *Map) EntriesSorted() []Entry {
	if p == nil {
		return nil
	}
	ptrs := make([]string, 0, len(p.last))
	for k := range p.last {
		ptrs = append(ptrs, k)
	}
	sort.Strings(ptrs)
	out := make([]Entry, 0, len(ptrs))
	for _, k := range ptrs {
		s := p.last[k]
		out = append(out, Entry{Ptr: k, Layer: s.Layer, File: s.File})
	}
	return out
}

func (p *Map) HistoryConflictsSorted() []HistoryEntry {
	if p == nil {
		return nil
	}
	ptrs := make([]string, 0, len(p.hist))
	for k := range p.hist {
		ptrs = append(ptrs, k)
	}
	sort.Strings(ptrs)

	out := make([]HistoryEntry, 0, 64)
	for _, ptr := range ptrs {
		seq := p.hist[ptr]
		if len(seq) < 2 {
			continue
		}
		uniq := uniqueSources(seq)
		if len(uniq) < 2 {
			continue
		}
		out = append(out, HistoryEntry{Ptr: ptr, Sources: uniq})
	}
	return out
}

func uniqueSources(seq []Source) []Source {
	if len(seq) == 0 {
		return nil
	}
	out := make([]Source, 0, len(seq))
	var last *Source
	for i := range seq {
		s := seq[i]
		if last != nil && last.Layer == s.Layer && last.File == s.File {
			continue
		}
		out = append(out, s)
		last = &out[len(out)-1]
	}
	if len(out) < 2 {
		return out
	}
	seen := map[string]bool{}
	final := make([]Source, 0, len(out))
	for _, s := range out {
		k := s.Layer + "\n" + s.File
		if seen[k] {
			continue
		}
		seen[k] = true
		final = append(final, s)
	}
	return final
}

func Escape(seg string) string {
	if seg == "" {
		return ""
	}
	if strings.IndexByte(seg, '~') == -1 && strings.IndexByte(seg, '/') == -1 {
		return seg
	}
	seg = strings.ReplaceAll(seg, "~", "~0")
	seg = strings.ReplaceAll(seg, "/", "~1")
	return seg
}
