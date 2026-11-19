package mod

import (
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/rules"
)

const (
	SpellMaxMana      = 32767
	SpellMaxRange     = 255
	SpellMaxRadius    = 255
	SpellMaxDamage    = 6898
	SpellMaxDuration  = 4095
	SpellMaxAreaLife  = 4000
	SpellMaxMagnitude = 32767
	SpellMaxRatio     = 1000000
	SpellMaxRays      = 100
)

func SpellEffectKinds() []string {
	return []string{"none", "health", "speed", "scanrange", "absorption",
		"protectionfire", "protectionwater", "protectionair", "protectionearth"}
}

func SpellEffectModes() []string {
	return []string{"duration", "continuous", "charges", "singleuse"}
}

type SpellInt struct {
	Set  bool
	Val  int32
	Line int
}

type SpellRatio struct {
	Set      bool
	Num, Den int64
	Line     int
}

type SpellBool struct {
	Set  bool
	Val  bool
	Line int
}

type SpellTable struct {
	Set  bool
	Vals []int32
	Line int
}

type SpellChoice struct {
	Set  bool
	Val  string
	Line int
}

func SpellAreaHitsNames() []string { return []string{"all", "not_own", "hostile"} }

type SpellEffectEdit struct {
	Line      int
	Kind      string
	Mode      string
	Magnitude SpellInt
	Duration  SpellInt
}

type SpellRow struct {
	Mod  string
	File string
	Line int

	Target string

	Mana, Range, Radius SpellInt
	Rays                SpellInt
	Duration, AreaLife  SpellInt
	DamageMin           SpellInt
	DamageMax           SpellInt
	Effect              *SpellEffectEdit
	HealHostile         SpellBool
	SelfCast            SpellBool
	AreaHits            SpellChoice

	Power, DamageFactor, RangeBonus SpellTable
	DurationFactor, Magnitude       SpellTable

	// Off takes the spell out of the book and the shops; BookPosition is its
	// book position.
	Off          SpellBool
	BookPosition SpellInt

	// Dir is the folder or packed file of the mod the row is read from.
	Dir string
	// Icon is the spell's book picture; Cast and Burst are the object that
	// flies and the burst, each drawn by its registry row. Imported marks
	// pictures copied from a game install.
	Icon, Cast, Burst SpellPicture
	CastRow, BurstRow *SpellSheet
	Imported          SpellBool
}

// SpellPicture is a row's picture field: the file of the mod it names, and
// the mod, file and folder of the row that names it, which a merged row
// keeps per field.
type SpellPicture struct {
	Set            bool
	Path           string
	Line           int
	Mod, File, Dir string
}

// SpellSheet is the registry row a cast picture is drawn by; a key not set
// is taken from the sheet.
type SpellSheet struct {
	Line                                                 int
	Phases, RotationPhases, Width, Height, Palette, Flip SpellInt
}

// ImportedPicturesWarning is the check's line on a row whose pictures are
// copies of a game install's.
const ImportedPicturesWarning = "the row's pictures are copied from a game install; publish the mod only where the game's rights allow it"

// ImportedSpells are the targets of d's rows whose pictures are copied from
// a game install.
func (d SpellData) ImportedSpells() []string {
	var out []string
	for _, r := range d.Rows {
		if r.Imported.Val && len(r.Pictures()) > 0 {
			out = append(out, r.Target)
		}
	}
	return out
}

// SpellSheetKeys are the keys of a cast picture's registry row.
func SpellSheetKeys() []string {
	return []string{"phases", "rotation_phases", "width", "height", "palette", "flip"}
}

// Ints are the row's values in the order of SpellSheetKeys.
func (s *SpellSheet) Ints() []*SpellInt {
	return []*SpellInt{&s.Phases, &s.RotationPhases, &s.Width, &s.Height, &s.Palette, &s.Flip}
}

// NewSpellSheet is the registry row of values, in the order of
// SpellSheetKeys; a value outside its key's bounds is left unset.
func NewSpellSheet(values ...int32) *SpellSheet {
	s := &SpellSheet{}
	for i, dst := range s.Ints() {
		if i < len(values) && int64(values[i]) >= spellSheetBounds[i][0] && int64(values[i]) <= spellSheetBounds[i][1] {
			*dst = SpellInt{Set: true, Val: values[i]}
		}
	}
	return s
}

// spellSheetBounds are the bounds of each key of SpellSheetKeys.
var spellSheetBounds = [][2]int64{{1, 1024}, {1, 64}, {1, 1024}, {1, 1024}, {0, 1}, {0, 1}}

// Pictures are the row's pictures that are set, each with the field that
// names it.
func (r SpellRow) Pictures() []SpellPictureField {
	var out []SpellPictureField
	for _, p := range []SpellPictureField{{"icon", r.Icon, nil}, {"cast", r.Cast, r.CastRow}, {"burst", r.Burst, r.BurstRow}} {
		if p.Picture.Set {
			out = append(out, p)
		}
	}
	return out
}

// SpellPictureField is one picture of a row: its field, the file and, for
// a cast picture, the registry row it states.
type SpellPictureField struct {
	Field   string
	Picture SpellPicture
	Row     *SpellSheet
}

// SpellEdit is the row's switch and book position for the game's profile.
func (r SpellRow) SpellEdit() rules.SpellEdit {
	return rules.SpellEdit{Name: r.Target, Off: r.Off.Set && r.Off.Val, On: r.Off.Set && !r.Off.Val,
		Position: int(r.BookPosition.Val)}
}

// Bare reports that the row sets no field: it names its spell and changes
// nothing, so its writer may take it out.
func (r SpellRow) Bare() bool { return len(spellFieldsOf(r)) == 0 }

func (r SpellRow) Formulas() [5]SpellTable {
	return [5]SpellTable{rules.FormulaPower: r.Power, rules.FormulaDamage: r.DamageFactor,
		rules.FormulaRange: r.RangeBonus, rules.FormulaDuration: r.DurationFactor, rules.FormulaMagnitude: r.Magnitude}
}

type SpellGlobal struct {
	Damage   SpellRatio
	Heal     SpellRatio
	Mana     SpellRatio
	Range    SpellRatio
	Radius   SpellRatio
	Duration SpellRatio

	Power, DamageFactor, RangeBonus, DurationFactor SpellTable

	// BookSize is the book's number of positions, set in BookSizeFile of
	// mod BookSizeMod.
	BookSize                  SpellInt
	BookSizeMod, BookSizeFile string
	// Line is the line of the [global] table, 0 when the file holds none.
	Line int
}

func (g SpellGlobal) Formulas() [5]SpellTable {
	var out [5]SpellTable
	out[rules.FormulaPower], out[rules.FormulaDamage] = g.Power, g.DamageFactor
	out[rules.FormulaRange], out[rules.FormulaDuration] = g.RangeBonus, g.DurationFactor
	return out
}

type SpellData struct {
	Global SpellGlobal
	Rows   []SpellRow
	// Order is the load order MergeSpells lays the rows in.
	Order Precedence
	// Pictures are the pictures under assets/spells by their clean path in
	// the mod, read in place of the mod's file: the editor's drafts not
	// written yet.
	Pictures map[string][]byte
}

func (d SpellData) Empty() bool { return len(d.Rows) == 0 && len(d.Global.Globals()) == 0 }

// MergeSpells merges the mods' rows into one row per catalog name.
func MergeSpells(d SpellData, base []string) (SpellData, []Override, error) {
	records := make([]SpellRow, 0, len(base))
	held := map[string]bool{}
	for _, name := range base {
		if k := rules.SpellKey(name); k != "" && !held[k] {
			held[k] = true
			records = append(records, SpellRow{Target: k})
		}
	}
	patches := make([]Patch[string, SpellRow], 0, len(d.Rows))
	for _, r := range d.Rows {
		var fields []PatchField
		for _, f := range spellFieldsOf(r) {
			fields = append(fields, PatchField{Name: f.name, Line: f.line})
		}
		patches = append(patches, Patch[string, SpellRow]{Source: Source{Mod: r.Mod, File: r.File, Line: r.Line},
			Key: rules.SpellKey(r.Target), Fields: fields, Apply: func(dst *SpellRow) { dst.overlay(r) }})
	}
	l, err := Merge("spell", records, func(r SpellRow) string { return r.Target }, patches, d.Order, nil)
	if err != nil {
		return SpellData{}, nil, err
	}
	out := SpellData{Global: d.Global, Order: d.Order, Pictures: d.Pictures}
	for _, r := range l.Records {
		if _, named := l.Named[r.Target]; named {
			out.Rows = append(out.Rows, r)
		}
	}
	return out, l.Overrides, nil
}

// overlay writes the fields src sets into r.
func (r *SpellRow) overlay(src SpellRow) {
	if r.Mod == "" {
		r.Mod, r.File, r.Line, r.Target = src.Mod, src.File, src.Line, rules.SpellKey(src.Target)
	}
	ints := func(dst *SpellInt, v SpellInt) {
		if v.Set {
			*dst = v
		}
	}
	ints(&r.Mana, src.Mana)
	ints(&r.Range, src.Range)
	ints(&r.Radius, src.Radius)
	ints(&r.Rays, src.Rays)
	ints(&r.Duration, src.Duration)
	ints(&r.AreaLife, src.AreaLife)
	if src.DamageMax.Set {
		r.DamageMin, r.DamageMax = src.DamageMin, src.DamageMax
	}
	if e := src.Effect; e != nil {
		if r.Effect == nil {
			r.Effect = &SpellEffectEdit{Line: e.Line}
		}
		if e.Kind != "" {
			r.Effect.Kind = e.Kind
		}
		if e.Mode != "" {
			r.Effect.Mode = e.Mode
		}
		ints(&r.Effect.Magnitude, e.Magnitude)
		ints(&r.Effect.Duration, e.Duration)
	}
	if src.HealHostile.Set {
		r.HealHostile = src.HealHostile
	}
	if src.SelfCast.Set {
		r.SelfCast = src.SelfCast
	}
	if src.AreaHits.Set {
		r.AreaHits = src.AreaHits
	}
	tables := func(dst *SpellTable, v SpellTable) {
		if v.Set {
			*dst = v
		}
	}
	tables(&r.Power, src.Power)
	tables(&r.DamageFactor, src.DamageFactor)
	tables(&r.RangeBonus, src.RangeBonus)
	tables(&r.DurationFactor, src.DurationFactor)
	tables(&r.Magnitude, src.Magnitude)
	if src.Off.Set {
		r.Off = src.Off
	}
	ints(&r.BookPosition, src.BookPosition)
	pictures := func(dst *SpellPicture, row **SpellSheet, v SpellPicture, vrow *SpellSheet) {
		if v.Set {
			*dst = v
			if dst.Mod == "" {
				dst.Mod, dst.File, dst.Dir = src.Mod, src.File, src.Dir
			}
			if row != nil {
				*row = vrow
			}
		}
	}
	pictures(&r.Icon, nil, src.Icon, nil)
	pictures(&r.Cast, &r.CastRow, src.Cast, src.CastRow)
	pictures(&r.Burst, &r.BurstRow, src.Burst, src.BurstRow)
	if src.Imported.Set {
		r.Imported = src.Imported
	}
}

func (g SpellGlobal) Globals() []SpellGlobalKey {
	var out []SpellGlobalKey
	for _, k := range []struct {
		key string
		r   SpellRatio
	}{{"damage_mul", g.Damage}, {"heal_mul", g.Heal}, {"mana_mul", g.Mana},
		{"range_mul", g.Range}, {"radius_mul", g.Radius}, {"duration_mul", g.Duration}} {
		if k.r.Set {
			out = append(out, SpellGlobalKey{Key: k.key, Line: k.r.Line})
		}
	}
	for f, t := range g.Formulas() {
		if t.Set {
			out = append(out, SpellGlobalKey{Key: rules.SpellFormula(f).String(), Line: t.Line})
		}
	}
	if g.BookSize.Set {
		out = append(out, SpellGlobalKey{Key: "book_size", Line: g.BookSize.Line})
	}
	return out
}

type SpellGlobalKey struct {
	Key  string
	Line int
}

func SpellFieldNames() []string {
	return []string{"mana", "range", "damage", "radius", "duration", "area_duration", "effect",
		"heal_hostile", "self_cast", "area_hits", "rays", "power", "damage_factor", "range_bonus", "duration_factor", "magnitude",
		"off", "book_position", "icon", "cast", "cast_row", "burst", "burst_row", "imported"}
}

func ParseSpells(id, file string, data []byte, setting func(name string) (int64, bool)) (SpellData, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return SpellData{}, fileError(file, err)
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return SpellData{}, spellBad(file, p.line, "%s is outside a [global] or [[spell]] table", p.key)
	}
	r := spellReader{file: file, setting: setting}
	var out SpellData
	for _, t := range tables[1:] {
		switch {
		case t.name == "global" && !t.array:
			g, err := r.global(t)
			if err != nil {
				return SpellData{}, err
			}
			if g.BookSize.Set {
				g.BookSizeMod, g.BookSizeFile = id, file
			}
			g.Line = t.line
			out.Global = g
		case t.name == "spell" && t.array:
			row, err := r.row(id, t)
			if err != nil {
				return SpellData{}, err
			}
			out.Rows = append(out.Rows, row)
		default:
			return SpellData{}, spellBad(file, t.line, "unknown table %q (this file holds [global] and [[spell]])", t.name)
		}
	}
	return out, nil
}

func spellBad(file string, line int, format string, args ...any) error {
	return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
}

type spellReader struct {
	file    string
	setting func(string) (int64, bool)
}

func (r spellReader) bad(line int, format string, args ...any) error {
	return spellBad(r.file, line, format, args...)
}

func (r spellReader) integer(name string, v tomlValue, line int) (int64, error) {
	switch v.kind {
	case tomlInt:
		return v.num, nil
	case tomlString:
		if !strings.HasPrefix(v.str, "@") {
			return 0, r.bad(line, "%s must be an integer or \"@setting\", not the text %q", name, v.str)
		}
		key := v.str[1:]
		if r.setting == nil {
			return 0, r.bad(line, "%s names the setting %q, and this file has none to read", name, key)
		}
		n, ok := r.setting(key)
		if !ok {
			return 0, r.bad(line, "%s names the setting %q, which is not an integer setting of this mod", name, key)
		}
		return n, nil
	}
	return 0, r.bad(line, "%s must be an integer or \"@setting\", not %s", name, v.kind)
}

func (r spellReader) ranged(name string, v tomlValue, line int, lo, hi int64) (SpellInt, error) {
	n, err := r.integer(name, v, line)
	if err != nil {
		return SpellInt{}, err
	}
	if n < lo || n > hi {
		return SpellInt{}, r.bad(line, "%s is %d, outside %d..%d", name, n, lo, hi)
	}
	return SpellInt{Set: true, Val: int32(n), Line: line}, nil
}

func (r spellReader) table(name string, v tomlValue, line int) (SpellTable, error) {
	var f rules.SpellFormula
	for f = 0; f < rules.FormulaMagnitude; f++ {
		if f.String() == name {
			break
		}
	}
	lo, hi, entries := rules.FormulaBounds(f)
	if v.kind != tomlArray {
		return SpellTable{}, r.bad(line, "%s must be an array of integers, not %s", name, v.kind)
	}
	if len(v.list) == 0 || len(v.list) > entries {
		return SpellTable{}, r.bad(line, "%s has %d entries, want 1..%d", name, len(v.list), entries)
	}
	vals := make([]int32, len(v.list))
	for i, item := range v.list {
		n, err := r.integer(fmt.Sprintf("%s entry %d", name, i), item, line)
		if err != nil {
			return SpellTable{}, err
		}
		if n < int64(lo) || n > int64(hi) {
			return SpellTable{}, r.bad(line, "%s entry %d is %d, outside %d..%d", name, i, n, lo, hi)
		}
		vals[i] = int32(n)
	}
	return SpellTable{Set: true, Vals: vals, Line: line}, nil
}

func (r spellReader) boolean(name string, v tomlValue, line int) (SpellBool, error) {
	if v.kind == tomlBool {
		return SpellBool{Set: true, Val: v.flag, Line: line}, nil
	}
	if v.kind != tomlString {
		return SpellBool{}, r.bad(line, "%s must be true, false or \"@setting\", not %s", name, v.kind)
	}
	n, err := r.integer(name, v, line)
	if err != nil {
		return SpellBool{}, err
	}
	if n != 0 && n != 1 {
		return SpellBool{}, r.bad(line, "%s is %d from its setting, which must be 0 or 1", name, n)
	}
	return SpellBool{Set: true, Val: n == 1, Line: line}, nil
}

func (r spellReader) choice(name string, v tomlValue, line int, names []string) (SpellChoice, error) {
	if v.kind != tomlString || !containsName(names, v.str) {
		return SpellChoice{}, r.bad(line, "%s must be one of %s", name, strings.Join(names, ", "))
	}
	return SpellChoice{Set: true, Val: v.str, Line: line}, nil
}

func (r spellReader) ratio(name string, v tomlValue, line int) (SpellRatio, error) {
	if v.kind != tomlArray || len(v.list) != 2 {
		return SpellRatio{}, r.bad(line, "%s must be [numerator, denominator]", name)
	}
	num, err := r.integer(name+" numerator", v.list[0], line)
	if err != nil {
		return SpellRatio{}, err
	}
	den, err := r.integer(name+" denominator", v.list[1], line)
	if err != nil {
		return SpellRatio{}, err
	}
	if num < 0 || num > SpellMaxRatio {
		return SpellRatio{}, r.bad(line, "%s numerator is %d, outside 0..%d", name, num, SpellMaxRatio)
	}
	if den < 1 || den > SpellMaxRatio {
		return SpellRatio{}, r.bad(line, "%s denominator is %d, outside 1..%d", name, den, SpellMaxRatio)
	}
	return SpellRatio{Set: true, Num: num, Den: den, Line: line}, nil
}

func (r spellReader) global(t tomlTable) (SpellGlobal, error) {
	var g SpellGlobal
	for _, p := range t.pairs {
		var dst *SpellRatio
		switch p.key {
		case "damage_mul":
			dst = &g.Damage
		case "heal_mul":
			dst = &g.Heal
		case "mana_mul":
			dst = &g.Mana
		case "range_mul":
			dst = &g.Range
		case "radius_mul":
			dst = &g.Radius
		case "duration_mul":
			dst = &g.Duration
		case "power", "damage_factor", "range_bonus", "duration_factor":
			t, err := r.table(p.key, p.val, p.line)
			if err != nil {
				return SpellGlobal{}, err
			}
			switch p.key {
			case "power":
				g.Power = t
			case "damage_factor":
				g.DamageFactor = t
			case "range_bonus":
				g.RangeBonus = t
			default:
				g.DurationFactor = t
			}
			continue
		case "magnitude":
			return SpellGlobal{}, r.bad(p.line, "magnitude has no [global] table: each spell's magnitude arm has its own shape, so give it in a [[spell]]")
		case "book_size":
			v, err := r.ranged("book_size", p.val, p.line, 1, int64(rules.SpellBookPositions()))
			if err != nil {
				return SpellGlobal{}, err
			}
			g.BookSize = v
			continue
		default:
			return SpellGlobal{}, r.bad(p.line, "unknown key %q in [global] (keys: damage_mul, heal_mul, mana_mul, range_mul, radius_mul, duration_mul, power, damage_factor, range_bonus, duration_factor, book_size)", p.key)
		}
		v, err := r.ratio(p.key, p.val, p.line)
		if err != nil {
			return SpellGlobal{}, err
		}
		*dst = v
	}
	return g, nil
}

func (r spellReader) row(id string, t tomlTable) (SpellRow, error) {
	row := SpellRow{Mod: id, File: r.file, Line: t.line}
	changes := false
	for _, p := range t.pairs {
		var err error
		switch p.key {
		case "target":
			if p.val.kind != tomlString {
				return SpellRow{}, r.bad(p.line, "target must be a string, not %s", p.val.kind)
			}
			if strings.TrimSpace(p.val.str) == "" {
				return SpellRow{}, r.bad(p.line, "target is empty")
			}
			row.Target = p.val.str
			continue
		case "mana":
			row.Mana, err = r.ranged("mana", p.val, p.line, 0, SpellMaxMana)
		case "range":
			row.Range, err = r.ranged("range", p.val, p.line, 0, SpellMaxRange)
		case "radius":
			row.Radius, err = r.ranged("radius", p.val, p.line, 0, SpellMaxRadius)
		case "duration":
			row.Duration, err = r.ranged("duration", p.val, p.line, 0, SpellMaxDuration)
		case "area_duration":
			row.AreaLife, err = r.ranged("area_duration", p.val, p.line, 0, SpellMaxAreaLife)
		case "damage":
			err = r.damage(&row, p)
		case "effect":
			row.Effect, err = r.effect(p)
		case "heal_hostile":
			row.HealHostile, err = r.boolean("heal_hostile", p.val, p.line)
		case "self_cast":
			row.SelfCast, err = r.boolean("self_cast", p.val, p.line)
		case "area_hits":
			row.AreaHits, err = r.choice("area_hits", p.val, p.line, SpellAreaHitsNames())
		case "rays":
			row.Rays, err = r.ranged("rays", p.val, p.line, 1, SpellMaxRays)
		case "power":
			row.Power, err = r.table(p.key, p.val, p.line)
		case "damage_factor":
			row.DamageFactor, err = r.table(p.key, p.val, p.line)
		case "range_bonus":
			row.RangeBonus, err = r.table(p.key, p.val, p.line)
		case "duration_factor":
			row.DurationFactor, err = r.table(p.key, p.val, p.line)
		case "magnitude":
			row.Magnitude, err = r.table(p.key, p.val, p.line)
		case "off":
			row.Off, err = r.boolean("off", p.val, p.line)
		case "book_position":
			row.BookPosition, err = r.ranged("book_position", p.val, p.line, 1, int64(rules.SpellBookPositions()))
		case "icon":
			row.Icon, err = r.picture(id, SpellIcon, p)
		case "cast":
			row.Cast, err = r.picture(id, SpellEffect, p)
		case "burst":
			row.Burst, err = r.picture(id, SpellEffect, p)
		case "cast_row":
			row.CastRow, err = r.sheet(p)
		case "burst_row":
			row.BurstRow, err = r.sheet(p)
		case "imported":
			row.Imported, err = r.boolean("imported", p.val, p.line)
		default:
			err = r.bad(p.line, "unknown key %q in [[spell]] (keys: target, %s)", p.key, strings.Join(SpellFieldNames(), ", "))
		}
		if err != nil {
			return SpellRow{}, err
		}
		changes = true
	}
	if row.Target == "" {
		return SpellRow{}, r.bad(t.line, "[[spell]] has no target")
	}
	if !changes {
		return SpellRow{}, r.bad(t.line, "[[spell]] of %q changes nothing", row.Target)
	}
	if row.Off.Val && row.BookPosition.Set {
		return SpellRow{}, r.bad(row.BookPosition.Line, "%s is off, so book_position does not apply to it", row.Target)
	}
	if row.CastRow != nil && !row.Cast.Set {
		return SpellRow{}, r.bad(row.CastRow.Line, "cast_row of %s needs cast, the picture it is the row of", row.Target)
	}
	if row.BurstRow != nil && !row.Burst.Set {
		return SpellRow{}, r.bad(row.BurstRow.Line, "burst_row of %s needs burst, the picture it is the row of", row.Target)
	}
	if row.Imported.Val && len(row.Pictures()) == 0 {
		return SpellRow{}, r.bad(row.Imported.Line, "imported marks the pictures of %s, and the row names none", row.Target)
	}
	return row, nil
}

// picture reads a row's picture field: a path the reference rule admits.
func (r spellReader) picture(id string, ref Reference, p tomlPair) (SpellPicture, error) {
	if p.val.kind != tomlString {
		return SpellPicture{}, r.bad(p.line, "%s must be a path in the mod, not %s", p.key, p.val.kind)
	}
	rel, err := ref.Check(r.file, p.line, p.key, p.val.str)
	if err != nil {
		return SpellPicture{}, err
	}
	return SpellPicture{Set: true, Path: rel, Line: p.line, Mod: id, File: r.file}, nil
}

// sheet reads a cast picture's registry row.
func (r spellReader) sheet(p tomlPair) (*SpellSheet, error) {
	keys := SpellSheetKeys()
	if p.val.kind != tomlInline {
		return nil, r.bad(p.line, "%s must be { %s }, not %s", p.key, strings.Join(keys, ", "), p.val.kind)
	}
	s := &SpellSheet{Line: p.line}
	dst := s.Ints()
	for _, q := range p.val.tbl {
		i := slices.Index(keys, q.key)
		if i < 0 {
			return nil, r.bad(p.line, "unknown key %q in %s (keys: %s)", q.key, p.key, strings.Join(keys, ", "))
		}
		v, err := r.ranged(p.key+"."+q.key, q.val, p.line, spellSheetBounds[i][0], spellSheetBounds[i][1])
		if err != nil {
			return nil, err
		}
		*dst[i] = v
	}
	if len(p.val.tbl) == 0 {
		return nil, r.bad(p.line, "%s changes nothing", p.key)
	}
	return s, nil
}

func (r spellReader) damage(row *SpellRow, p tomlPair) error {
	if p.val.kind != tomlInline {
		return r.bad(p.line, "damage must be { min = N, max = N }, not %s", p.val.kind)
	}
	for _, q := range p.val.tbl {
		switch q.key {
		case "min":
			v, err := r.ranged("damage.min", q.val, p.line, 0, SpellMaxDamage)
			if err != nil {
				return err
			}
			row.DamageMin = v
		case "max":
			v, err := r.ranged("damage.max", q.val, p.line, 0, SpellMaxDamage)
			if err != nil {
				return err
			}
			row.DamageMax = v
		default:
			return r.bad(p.line, "unknown key %q in damage (keys: min, max)", q.key)
		}
	}
	if !row.DamageMin.Set || !row.DamageMax.Set {
		return r.bad(p.line, "damage needs both min and max")
	}
	if row.DamageMin.Val > row.DamageMax.Val {
		return r.bad(p.line, "damage.min %d is above damage.max %d", row.DamageMin.Val, row.DamageMax.Val)
	}
	if row.DamageMax.Val == 0 {
		return r.bad(p.line, "damage is 0 to 0; a spell that does no damage or healing is not edited this way")
	}
	return nil
}

func (r spellReader) effect(p tomlPair) (*SpellEffectEdit, error) {
	if p.val.kind != tomlInline {
		return nil, r.bad(p.line, "effect must be { kind, magnitude, mode, duration }, not %s", p.val.kind)
	}
	e := &SpellEffectEdit{Line: p.line}
	for _, q := range p.val.tbl {
		switch q.key {
		case "kind", "mode":
			if q.val.kind != tomlString {
				return nil, r.bad(p.line, "effect.%s must be a string, not %s", q.key, q.val.kind)
			}
			names, dst := SpellEffectKinds(), &e.Kind
			if q.key == "mode" {
				names, dst = SpellEffectModes(), &e.Mode
			}
			if !containsName(names, q.val.str) {
				return nil, r.bad(p.line, "effect.%s is %q (use %s)", q.key, q.val.str, strings.Join(names, ", "))
			}
			*dst = q.val.str
		case "magnitude":
			v, err := r.ranged("effect.magnitude", q.val, p.line, -SpellMaxMagnitude-1, SpellMaxMagnitude)
			if err != nil {
				return nil, err
			}
			e.Magnitude = v
		case "duration":
			v, err := r.ranged("effect.duration", q.val, p.line, 0, SpellMaxDuration)
			if err != nil {
				return nil, err
			}
			e.Duration = v
		default:
			return nil, r.bad(p.line, "unknown key %q in effect (keys: kind, magnitude, mode, duration)", q.key)
		}
	}
	if e.Kind == "" && e.Mode == "" && !e.Magnitude.Set && !e.Duration.Set {
		return nil, r.bad(p.line, "effect changes nothing")
	}
	return e, nil
}

func containsName(names []string, s string) bool {
	for _, n := range names {
		if n == s {
			return true
		}
	}
	return false
}
