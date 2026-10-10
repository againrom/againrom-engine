// Package modrt runs mods: it orders the resolved mods, executes each mod's
// Starlark entry file and calls init(game, settings), and returns the rules the
// mods set together with the mod set a save records.
//
// A script reaches the game through three values only. settings holds the mod's
// own resolved settings. game.rules holds the declared rule parameters; each is
// an integer within a declared range and every other type or value is a script
// error. game.data.add reads one of the mod's data files, which add the items,
// screens or companion join conditions the file describes, or edit the
// characters it names. The interpreter is given no file, network, clock or
// randomness: the
// predeclared names are the Starlark language's own, load() reads only files of
// the mod's own folder, and a script that runs too long is stopped. After a
// mod's init returns, the values it built and its handle on game are frozen.
//
// This is the only package that imports the interpreter. The simulation never
// sees it: it receives the finished rules.Rules value and nothing else.
package modrt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"

	"againrom/pkg/locale"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
)

// EntryName is the entry file of a mod that does not name one.
const EntryName = "main.star"

// MaxSteps bounds the work one mod's script may do.
const MaxSteps = 20_000_000

// Options configures a run.
type Options struct {
	// Print receives the output of the script's print function. Nil discards it.
	Print func(modID, msg string)
}

// Result is what running a mod set produced.
type Result struct {
	// Rules are the parameters the mods set; the original game's when no mod
	// sets one.
	Rules rules.Rules
	// Set describes the active mods in load order with their settings.
	Set mod.Set
	// Ordered are the mods in load order.
	Ordered []mod.Entry
	// Items are the items the mods add and edit, in load order.
	Items mod.ItemData
	// Characters are the character edits the mods make, in load order.
	Characters mod.CharacterData
	// Screens are the screens the mods declare, in load order.
	Screens mod.ScreenData
	// Companions are the join conditions the mods declare, in load order.
	Companions mod.CompanionData
	Spells     mod.SpellData
}

// Load orders mods for the base, resolves their settings from given and runs
// each mod in order. Every refusal names the mod, and a script error names the
// file and line.
func Load(mods []mod.Entry, base string, given []mod.SettingFlag, opt Options) (Result, error) {
	if len(mods) == 0 {
		if len(given) != 0 {
			return Result{}, fmt.Errorf("-mod-setting names mod %q, which is not enabled", given[0].Mod)
		}
		return Result{Rules: rules.Default(), Set: mod.Set{Base: base}}, nil
	}
	ordered, err := mod.Order(mods, base)
	if err != nil {
		return Result{}, err
	}
	byMod := map[string]map[string]string{}
	enabled := map[string]bool{}
	for _, m := range ordered {
		enabled[m.Manifest.ID] = true
	}
	for _, g := range given {
		if !enabled[g.Mod] {
			return Result{}, fmt.Errorf("-mod-setting %s.%s names a mod that is not enabled", g.Mod, g.Key)
		}
		if byMod[g.Mod] == nil {
			byMod[g.Mod] = map[string]string{}
		}
		if _, dup := byMod[g.Mod][g.Key]; dup {
			return Result{}, fmt.Errorf("-mod-setting %s.%s is given twice", g.Mod, g.Key)
		}
		byMod[g.Mod][g.Key] = g.Value
	}
	sh := &shared{params: rules.Defaults(), lang: LanguageFor(base), spellGlobals: map[string]string{}, spellFields: map[string]string{}}
	set := mod.Set{Base: base}
	for _, m := range ordered {
		id := m.Manifest.ID
		decl, err := mod.LoadSettings(m.Dir)
		if err != nil {
			return Result{}, fmt.Errorf("mod %q: %w", id, err)
		}
		values, err := mod.ResolveSettings(id, decl, byMod[id])
		if err != nil {
			return Result{}, err
		}
		digest, err := mod.ContentDigest(m.Dir)
		if err != nil {
			return Result{}, fmt.Errorf("mod %q: %w", id, err)
		}
		if err := runMod(m, values, sh, opt); err != nil {
			return Result{}, err
		}
		set.Mods = append(set.Mods, mod.SetEntry{ID: id, Version: m.Manifest.Version, Digest: digest, Settings: values})
	}
	r, err := rules.New(sh.params)
	if err != nil {
		return Result{}, fmt.Errorf("the mods set rules the game refuses: %w", err)
	}
	return Result{Rules: r, Set: set, Ordered: ordered, Items: sh.items, Characters: sh.characters, Screens: sh.screens, Companions: sh.companions, Spells: sh.spells}, nil
}

// LanguageFor is the language a mod's text is read in on a base: the locale
// whose code ends the base id, else the reference language.
func LanguageFor(base string) string {
	if l, ok := locale.ByBaseSuffix(base); ok {
		return l.Code
	}
	return locale.Fallback
}

// shared holds what every mod's game handle edits.
type shared struct {
	params       rules.Params
	lang         string
	items        mod.ItemData
	characters   mod.CharacterData
	screens      mod.ScreenData
	companions   mod.CompanionData
	spells       mod.SpellData
	spellGlobals map[string]string
	spellFields  map[string]string
}

// ruleField declares one parameter of game.rules. Adding a parameter is adding
// an entry here and a field to rules.Params.
type ruleField struct {
	name     string
	min, max int64
	get      func(rules.Params) int64
	set      func(*rules.Params, int64)
}

var ruleFields = []ruleField{
	{
		name: "skill_cap",
		min:  int64(rules.MinSkillCap), max: int64(rules.MaxSkillCap),
		get: func(p rules.Params) int64 { return int64(p.SkillCap) },
		set: func(p *rules.Params, v int64) { p.SkillCap = int32(v) },
	},
}

// FieldNames lists the parameters game.rules declares, in order.
func FieldNames() []string {
	out := make([]string, len(ruleFields))
	for i, f := range ruleFields {
		out[i] = f.name
	}
	return out
}

func fieldByName(name string) *ruleField {
	for i := range ruleFields {
		if ruleFields[i].name == name {
			return &ruleFields[i]
		}
	}
	return nil
}

// gameValue is the game argument of init.
type gameValue struct {
	rules *rulesValue
	data  *dataValue
}

func (g *gameValue) String() string        { return "<game>" }
func (g *gameValue) Type() string          { return "game" }
func (g *gameValue) Freeze()               { g.rules.Freeze(); g.data.Freeze() }
func (g *gameValue) Truth() starlark.Bool  { return true }
func (g *gameValue) Hash() (uint32, error) { return 0, errors.New("unhashable type: game") }
func (g *gameValue) AttrNames() []string   { return []string{"data", "rules"} }
func (g *gameValue) Attr(name string) (starlark.Value, error) {
	switch name {
	case "rules":
		return g.rules, nil
	case "data":
		return g.data, nil
	}
	return nil, nil
}

// rulesValue is game.rules: a record of integer parameters, each assigned
// within its declared range.
type rulesValue struct {
	sh     *shared
	frozen bool
}

func (r *rulesValue) String() string        { return "<game.rules>" }
func (r *rulesValue) Type() string          { return "game.rules" }
func (r *rulesValue) Freeze()               { r.frozen = true }
func (r *rulesValue) Truth() starlark.Bool  { return true }
func (r *rulesValue) Hash() (uint32, error) { return 0, errors.New("unhashable type: game.rules") }

func (r *rulesValue) AttrNames() []string {
	names := FieldNames()
	sort.Strings(names)
	return names
}

func (r *rulesValue) Attr(name string) (starlark.Value, error) {
	f := fieldByName(name)
	if f == nil {
		return nil, starlark.NoSuchAttrError(fmt.Sprintf("game.rules has no parameter %q (declared: %s)", name, strings.Join(FieldNames(), ", ")))
	}
	return starlark.MakeInt64(f.get(r.sh.params)), nil
}

func (r *rulesValue) SetField(name string, val starlark.Value) error {
	f := fieldByName(name)
	if f == nil {
		return starlark.NoSuchAttrError(fmt.Sprintf("game.rules has no parameter %q (declared: %s)", name, strings.Join(FieldNames(), ", ")))
	}
	if r.frozen {
		return fmt.Errorf("game.rules.%s cannot be set: the rules are frozen once init has returned", name)
	}
	i, ok := val.(starlark.Int)
	if !ok {
		return fmt.Errorf("game.rules.%s must be an int, got %s", name, val.Type())
	}
	n, ok := i.Int64()
	if !ok || n < f.min || n > f.max {
		return fmt.Errorf("game.rules.%s: %s is outside the declared range %d..%d", name, i.String(), f.min, f.max)
	}
	f.set(&r.sh.params, n)
	return nil
}

var fileOptions = &syntax.FileOptions{}

// runMod executes one mod's entry file and its init.
func runMod(m mod.Entry, values []mod.SettingValue, sh *shared, opt Options) error {
	id := m.Manifest.ID
	entry := m.Manifest.Entry
	if entry == "" {
		if _, err := os.Stat(filepath.Join(m.Dir, EntryName)); err != nil {
			return nil
		}
		entry = EntryName
	}
	l := &loader{id: id, dir: m.Dir, opt: opt, cache: map[string]*module{}}
	thread := l.newThread()
	src, rel, err := l.read(entry)
	if err != nil {
		return fmt.Errorf("mod %q: entry %s: %w", id, entry, err)
	}
	globals, err := starlark.ExecFileOptions(fileOptions, thread, rel, src, nil)
	if err != nil {
		return describe(id, err)
	}
	initFn, ok := globals["init"].(starlark.Callable)
	if !ok {
		return fmt.Errorf("mod %q: %s defines no init(game, settings)", id, rel)
	}
	dict := starlark.StringDict{}
	for _, v := range values {
		switch v.Value.Kind {
		case mod.KindInt:
			dict[v.Key] = starlark.MakeInt64(v.Value.Int)
		case mod.KindBool:
			dict[v.Key] = starlark.Bool(v.Value.Bool)
		default:
			dict[v.Key] = starlark.String(v.Value.Str)
		}
	}
	settings := starlarkstruct.FromStringDict(starlarkstruct.Default, dict)
	game := &gameValue{rules: &rulesValue{sh: sh}, data: &dataValue{sh: sh, id: id, dir: m.Dir, loader: l, loaded: map[string]bool{}, settings: values}}
	if _, err := starlark.Call(thread, initFn, starlark.Tuple{game, settings}, nil); err != nil {
		game.Freeze()
		return describe(id, err)
	}
	game.Freeze()
	settings.Freeze()
	return nil
}

// module is a loaded file's globals, or the error loading it gave.
type module struct {
	globals starlark.StringDict
	err     error
	loading bool
}

type loader struct {
	id    string
	dir   string
	opt   Options
	cache map[string]*module
}

func (l *loader) newThread() *starlark.Thread {
	t := &starlark.Thread{Name: l.id}
	t.SetMaxExecutionSteps(MaxSteps)
	t.Print = func(_ *starlark.Thread, msg string) {
		if l.opt.Print != nil {
			l.opt.Print(l.id, msg)
		}
	}
	t.Load = l.load
	return t
}

// read returns the bytes of a file of the mod folder and its clean relative
// slash path; see mod.ReadInside.
func (l *loader) read(name string) ([]byte, string, error) {
	return mod.ReadInside(l.dir, name)
}

func (l *loader) load(thread *starlark.Thread, name string) (starlark.StringDict, error) {
	if !strings.HasSuffix(name, ".star") {
		return nil, fmt.Errorf("load(%q): only .star files of the mod folder can be loaded", name)
	}
	src, rel, err := l.read(name)
	if err != nil {
		return nil, fmt.Errorf("load(%q): %w", name, err)
	}
	if c, ok := l.cache[rel]; ok {
		if c.loading {
			return nil, fmt.Errorf("load(%q): cycle: the file is already being loaded", name)
		}
		return c.globals, c.err
	}
	c := &module{loading: true}
	l.cache[rel] = c
	inner := l.newThread()
	c.globals, c.err = starlark.ExecFileOptions(fileOptions, inner, rel, src, nil)
	if c.err != nil {
		c.err = errors.New(where(c.err))
	}
	c.loading = false
	return c.globals, c.err
}

// describe turns an interpreter error into one that names the mod, the file and
// the line of the failing statement.
func describe(modID string, err error) error {
	return fmt.Errorf("mod %q: %s", modID, where(err))
}

// where renders an interpreter error as "file:line: message".
func where(err error) string {
	var fe *mod.ItemFileError
	if errors.As(err, &fe) {
		return fe.Error()
	}
	var ev *starlark.EvalError
	if errors.As(err, &ev) {
		for i := len(ev.CallStack) - 1; i >= 0; i-- {
			p := ev.CallStack[i].Pos
			if p.Filename() != "<builtin>" {
				return fmt.Sprintf("%s:%d: %s", p.Filename(), p.Line, ev.Msg)
			}
		}
		return ev.Msg
	}
	var se syntax.Error
	if errors.As(err, &se) {
		return fmt.Sprintf("%s:%d: %s", se.Pos.Filename(), se.Pos.Line, se.Msg)
	}
	var rl resolve.ErrorList
	if errors.As(err, &rl) && len(rl) > 0 {
		return fmt.Sprintf("%s:%d: %s", rl[0].Pos.Filename(), rl[0].Pos.Line, rl[0].Msg)
	}
	return err.Error()
}
