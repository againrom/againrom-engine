package mod

import (
	"fmt"
	"slices"
)

// A catalog kind is built in layers. Layer 0 is the base's decoder's
// records; each mod's records of the kind are one layer over it, applied in
// load order. A record names its key and the fields it sets: a key no layer
// holds adds a record, a key one holds changes only the fields named, and
// the field "off" switches a record off or back on. Nothing is deleted: a
// record switched off is left out of the merged records. Merge is the one
// layer step; a kind supplies its records, its key and how a patch writes
// its fields.

// OffField is the field that switches a record off (true) or back on.
const OffField = "off"

// Source is where a layer record stands: its mod, file and line.
type Source struct {
	Mod  string
	File string
	Line int
}

// Errorf is a refusal at the source, naming the mod, the file and the line.
func (s Source) Errorf(format string, args ...any) error {
	return &LayerError{Source: s, Msg: fmt.Sprintf(format, args...)}
}

// LayerError is a refusal of a layer record.
type LayerError struct {
	Source
	Msg string
}

func (e *LayerError) Error() string {
	return fmt.Sprintf("mod %q: %s", e.Mod, (&ItemFileError{File: e.File, Line: e.Line, Msg: e.Msg}).Error())
}

// PatchField is one field a layer record sets, with the line that sets it.
type PatchField struct {
	Name string
	Line int
}

// Patch is one layer record over records of type R keyed by K.
type Patch[K comparable, R any] struct {
	Source
	Key K
	// Fields are the fields the record sets, OffField among them when it
	// switches the record off or on.
	Fields []PatchField
	// Off is the value of OffField when Fields names it.
	Off bool
	// Apply writes the fields the record sets, other than OffField, into r.
	Apply func(r *R)
}

// Override is one field a later mod sets over an earlier mod's value.
type Override struct {
	Kind    string
	Key     string
	Field   string
	Earlier string
	Later   Source
}

func (o Override) String() string {
	return fmt.Sprintf("%s %s %s: mod %q (%s:%d) overrides mod %q", o.Kind, o.Key, o.Field, o.Later.Mod, o.Later.File, o.Later.Line, o.Earlier)
}

// Precedence is the load order the mods state: each mod by the mods it names
// in requires or in load-after, among the enabled ones.
type Precedence map[string][]string

// After reports whether mod later is ordered after mod earlier, directly or
// through other mods.
func (p Precedence) After(later, earlier string) bool {
	seen := map[string]bool{}
	stack := []string{later}
	for len(stack) > 0 {
		m := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, d := range p[m] {
			if d == earlier {
				return true
			}
			if !seen[d] {
				seen[d] = true
				stack = append(stack, d)
			}
		}
	}
	return false
}

// PrecedenceOf is the precedence the manifests of the ordered mods state.
func PrecedenceOf(ordered []Entry) Precedence {
	enabled := map[string]bool{}
	for _, m := range ordered {
		enabled[m.Manifest.ID] = true
	}
	p := Precedence{}
	for _, m := range ordered {
		id := m.Manifest.ID
		for _, r := range m.Manifest.Requires {
			if d, err := ParseDep(r); err == nil && enabled[d.ID] {
				p[id] = append(p[id], d.ID)
			}
		}
		for _, l := range m.Manifest.LoadAfter {
			if enabled[l] {
				p[id] = append(p[id], l)
			}
		}
	}
	return p
}

// Layered is the result of a layer step.
type Layered[K comparable, R any] struct {
	// Records are the merged records that are not switched off: layer 0's
	// in its order, then the added ones in the order they were added.
	Records []R
	// Overrides are the fields a later mod set over an earlier mod's.
	Overrides []Override
	// Named is the last record that named each key.
	Named map[K]Source
	// Off are the keys switched off, each with the record that did.
	Off map[K]Source
}

// Merge applies patches, in load order, over base, layer 0, whose records
// key names. kind names the records in a refusal. After each mod's last
// patch, check, when given, judges the records so far; its error stands as
// the refusal of that mod. Two mods that set one field, or that both name a
// key no earlier layer holds, are refused unless order puts the later one
// after the earlier; then the field is an Override.
func Merge[K comparable, R any](kind string, base []R, key func(R) K, patches []Patch[K, R], order Precedence,
	check func(mod string, l *Layered[K, R]) error) (*Layered[K, R], error) {
	type slot struct {
		rec   R
		off   bool
		added string
		owner map[string]string
	}
	var keys []K
	slots := map[K]*slot{}
	for _, r := range base {
		k := key(r)
		keys = append(keys, k)
		slots[k] = &slot{rec: r, owner: map[string]string{}}
	}
	l := &Layered[K, R]{Named: map[K]Source{}, Off: map[K]Source{}}
	view := func() {
		l.Records = l.Records[:0]
		for _, k := range keys {
			if s := slots[k]; !s.off {
				l.Records = append(l.Records, s.rec)
			}
		}
	}
	for i, p := range patches {
		s, ok := slots[p.Key]
		switch {
		case !ok && p.Off:
			return nil, p.Errorf("%s %v is switched off, but no layer holds it", kind, p.Key)
		case !ok:
			var zero R
			s = &slot{rec: zero, added: p.Mod, owner: map[string]string{}}
			slots[p.Key] = s
			keys = append(keys, p.Key)
		case s.added != "" && s.added != p.Mod && !order.After(p.Mod, s.added):
			return nil, p.Errorf("%s %v is added by mod %q and named by mod %q, and neither loads after the other: add load-after to one of them",
				kind, p.Key, s.added, p.Mod)
		}
		for _, f := range p.Fields {
			prior := s.owner[f.Name]
			if prior != "" && prior != p.Mod {
				if !order.After(p.Mod, prior) {
					return nil, p.Errorf("%s %v %s is set by mod %q and by mod %q, and neither loads after the other: add load-after to one of them",
						kind, p.Key, f.Name, prior, p.Mod)
				}
				l.Overrides = append(l.Overrides, Override{Kind: kind, Key: fmt.Sprint(p.Key), Field: f.Name, Earlier: prior,
					Later: Source{Mod: p.Mod, File: p.File, Line: f.Line}})
			}
			s.owner[f.Name] = p.Mod
		}
		if p.Apply != nil {
			p.Apply(&s.rec)
		}
		if slices.ContainsFunc(p.Fields, func(f PatchField) bool { return f.Name == OffField }) {
			s.off = p.Off
			if p.Off {
				l.Off[p.Key] = p.Source
			} else {
				delete(l.Off, p.Key)
			}
		}
		l.Named[p.Key] = p.Source
		if check != nil && (i == len(patches)-1 || patches[i+1].Mod != p.Mod) {
			view()
			if err := check(p.Mod, l); err != nil {
				return nil, err
			}
		}
	}
	view()
	return l, nil
}
