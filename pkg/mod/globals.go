package mod

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"againrom/pkg/locale"
	"againrom/pkg/scriptprofile"
)

// FirstGlobal and LastGlobal bound the id a mod's global variable takes, as
// the script limits state them; MaxGlobals is how many one mod declares.
const (
	FirstGlobal = scriptprofile.FirstGlobal
	LastGlobal  = scriptprofile.LastGlobal
	MaxGlobals  = LastGlobal - FirstGlobal + 1
)

// globalNameMax is the longest name a global takes.
const globalNameMax = 32

// Global is one global variable a mod declares: the id a map names it by, a
// name for the editor and the checks, the value a run starts with, and
// optional notes per language code.
type Global struct {
	Mod, File string
	Line      int
	ID        int32
	Name      string
	Initial   int32
	Notes     map[string]string
}

// GlobalData is the global variables the mods declare, each mod's in its
// file order.
type GlobalData struct {
	Globals []Global
}

// Empty reports whether no mod declares a global.
func (d GlobalData) Empty() bool { return len(d.Globals) == 0 }

// Of is the globals mod id declares, in its file order.
func (d GlobalData) Of(id string) []Global {
	var out []Global
	for _, g := range d.Globals {
		if g.Mod == id {
			out = append(out, g)
		}
	}
	return out
}

// Declares reports that mod id declares global n.
func (d GlobalData) Declares(id string, n int32) bool {
	return slices.ContainsFunc(d.Globals, func(g Global) bool { return g.Mod == id && g.ID == n })
}

// ValidGlobalName is [a-z][a-z0-9_]{0,31}: what a global's name may be.
func ValidGlobalName(s string) bool { return validGlobalName(s) }

// validGlobalName is [a-z][a-z0-9_]{0,31}.
func validGlobalName(s string) bool {
	if s == "" || len(s) > globalNameMax || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

// ParseGlobals reads data/globals.toml of the mod id: [[global]] tables with
// the keys id, name, initial and a note per language code. A refusal names
// the file and the line.
func ParseGlobals(id, file string, data []byte) (GlobalData, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return GlobalData{}, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return GlobalData{}, bad(p.line, "%s is outside a [[global]] table", p.key)
	}
	var out GlobalData
	ids, names := map[int32]bool{}, map[string]bool{}
	for _, t := range tables[1:] {
		if t.name != "global" || !t.array {
			return GlobalData{}, bad(t.line, "unknown table %q (this file holds [[global]])", t.name)
		}
		if len(out.Globals) == MaxGlobals {
			return GlobalData{}, bad(t.line, "a mod declares at most %d globals", MaxGlobals)
		}
		g := Global{Mod: id, File: file, Line: t.line}
		hasID, hasName := false, false
		for _, p := range t.pairs {
			var err error
			switch p.key {
			case "id":
				if p.val.kind != tomlInt {
					err = bad(p.line, "id must be an integer, not %s", p.val.kind)
				} else if p.val.num < FirstGlobal || p.val.num > LastGlobal {
					err = bad(p.line, "id is %d; a global takes an id from %d to %d", p.val.num, FirstGlobal, LastGlobal)
				}
				g.ID, hasID = int32(p.val.num), true
			case "name":
				if p.val.kind != tomlString || !validGlobalName(p.val.str) {
					err = bad(p.line, "name must be a lower-case letter followed by at most %d lower-case letters, digits or '_'", globalNameMax-1)
				}
				g.Name, hasName = p.val.str, true
			case "initial":
				if p.val.kind != tomlInt || p.val.num < math.MinInt32 || p.val.num > math.MaxInt32 {
					err = bad(p.line, "initial must be an integer from %d to %d", math.MinInt32, math.MaxInt32)
				}
				g.Initial = int32(p.val.num)
			default:
				if !locale.Known(p.key) {
					err = bad(p.line, "unknown key %q (keys: id, name, initial and a note per language code)", p.key)
				} else if p.val.kind != tomlString {
					err = bad(p.line, "the %s note must be a string", p.key)
				} else {
					if g.Notes == nil {
						g.Notes = map[string]string{}
					}
					g.Notes[p.key] = p.val.str
				}
			}
			if err != nil {
				return GlobalData{}, err
			}
		}
		if !hasID {
			return GlobalData{}, bad(t.line, "[[global]] has no id")
		}
		if !hasName {
			return GlobalData{}, bad(t.line, "[[global]] has no name")
		}
		if ids[g.ID] {
			return GlobalData{}, bad(t.line, "global %d stands twice in this file", g.ID)
		}
		if names[g.Name] {
			return GlobalData{}, bad(t.line, "the name %q stands twice in this file", g.Name)
		}
		ids[g.ID], names[g.Name] = true, true
		out.Globals = append(out.Globals, g)
	}
	return out, nil
}

// readGlobals reads the mod's global declarations.
func readGlobals(r DataReader, d *Data, rel string, src []byte) error {
	globals, err := ParseGlobals(r.Mod, rel, src)
	if err != nil {
		return err
	}
	d.Globals.Globals = append(d.Globals.Globals, globals.Globals...)
	return nil
}

// writeGlobals is the globals kind's writer: source made to hold
// d.Globals.Globals, each [[global]] table kept where it stands with its
// lines but for the keys it changes. A global the source does not hold is
// appended; one d lacks leaves with its table.
func writeGlobals(rel string, source []byte, d Data) ([]byte, error) {
	before, err := ParseGlobals("", rel, source)
	if err != nil {
		return nil, err
	}
	after := globalSourceRows(d.Globals.Globals)
	out, err := writeRows(source, globalSourceRows(before.Globals), after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseGlobals("", rel, out)
	if err != nil {
		return nil, err
	}
	if !sameRows(globalSourceRows(back.Globals), after) {
		return nil, fmt.Errorf("%s: the written rows do not read back as the rows given", rel)
	}
	return out, nil
}

// globalSourceRows is the globals as the row writer sees them: a global is
// named by its id and writes its name, a non-zero initial and its notes.
func globalSourceRows(list []Global) []sourceRow {
	rows := make([]sourceRow, len(list))
	for i, g := range list {
		pairs := []missionPair{{"id", strconv.Itoa(int(g.ID))}, {"name", writeString(g.Name)}}
		if g.Initial != 0 {
			pairs = append(pairs, missionPair{"initial", strconv.Itoa(int(g.Initial))})
		}
		for _, l := range locale.All() {
			if v, ok := g.Notes[l.Code]; ok {
				pairs = append(pairs, missionPair{l.Code, writeString(v)})
			}
		}
		rows[i] = sourceRow{table: "global", identity: strconv.Itoa(int(g.ID)), pairs: pairs}
	}
	return rows
}
