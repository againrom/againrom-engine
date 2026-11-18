package game

import (
	"errors"
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/ui"
)

// The keys of W's spells page: the catalog grid, the book, the fold of
// the globals, and the prefixes of the chosen spell's fields and of the
// globals.
const (
	spellsGridKey    = "catalog"
	spellsBookKey    = "book"
	spellsGlobalsKey = "globals"
	spellFieldPrefix = "spell."
	spellGlobalKey   = "global."
)

// spellsLook is the install's book art, book icon strip and spell names,
// read once for the spells page.
type spellsLook struct {
	art   *ui.BottomHUDArt
	atlas *bmp.Image
	names *TextTable
}

// spellsLook is the install's look of the book; empty parts without an
// install.
func (x *MapInspector) spellsLook() *spellsLook {
	if x.spellArt != nil {
		return x.spellArt
	}
	x.spellArt = &spellsLook{}
	if a := x.archives; a != nil {
		x.spellArt.art, _ = LoadBottomHUDArt(a.artSource(), ScreensDescription(a.Base.Profile))
		x.spellArt.atlas = loadSpellAtlas(a.Containers)
		x.spellArt.names = LoadTextTable(a.Containers, SpellNamesTextPath, InstallTextCode(a.Containers, a.Game().Edition()))
	}
	return x.spellArt
}

// spellLabel is catalog spell name as the game names spell number id, in
// the font's bytes; the catalog name, spaced, for a spell the game lacks.
func (x *MapInspector) spellLabel(name string, id int) string {
	if t, ok := x.spellsLook().names.At(id - 1); ok && id > 0 {
		return t
	}
	return x.inspectionUTF8(strings.ReplaceAll(name, "_", " "))
}

// spellInitials is the text a catalog cell without a picture shows: the
// first letter of each word of the name.
func spellInitials(name string) string {
	var out []byte
	for _, w := range strings.Split(rules.SpellKey(name), "_") {
		if w != "" {
			out = append(out, w[0])
		}
	}
	return string(out)
}

// spellsBook is the book the mod's rows d give the game of table base: the
// spell number at each cell, 0 at an empty one, by the one rule the game
// reads a mod's book with; a refusal names the row that breaks it.
func spellsBook(base *mapload.Table, d mod.SpellData) ([]uint16, error) {
	p, _, msg := mapload.ModSpellProfile(base, d)
	if msg != "" {
		return nil, errors.New(msg)
	}
	return p.BookRows(), nil
}

// spellsFields is W's spells page: the chosen spell's fields, the globals
// folded beside them, the catalog of both games' spells and the mod's book
// as the game draws it.
func (s *mapInspectionEdits) spellsFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	project := s.workspace.Project
	if project == nil {
		project = x.mod.document()
	}
	read, err := s.heldRows(project, nil, mod.SpellsFile)
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	base := x.contextBase().table
	g := mapload.SpellGameOf(base)
	rows, err := spellsBook(base, read.Spells)
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	out := s.chosenSpellFields(base, read.Spells)
	out = append(out, s.spellGlobalFields(read.Spells.Global)...)

	look := x.spellsLook()
	ground := spellBookRows(base)
	catalog := rules.SpellCatalog()
	grid := ui.InspectionField{Key: spellsGridKey, Label: x.word("editor.spells.catalog"), Grid: true, Small: true,
		Icons: map[int64]*image.RGBA{}, Marks: map[int64]string{}}
	for i, name := range catalog {
		v := int64(i + 1)
		id, ok := rules.SpellNumber(g, name)
		label := x.spellLabel(name, id)
		if !ok {
			label = x.inspectionUTF8(fmt.Sprintf(x.word("editor.spells.lacking"), strings.ReplaceAll(name, "_", " ")))
			grid.Shut = append(grid.Shut, v)
		}
		grid.Choices = append(grid.Choices, ui.InspectionChoice{Value: v, Label: label})
		if pic := cutBookIcon(look.atlas, ground, uint16(id)); ok && pic != nil {
			grid.Icons[v] = pic
		}
		grid.Marks[v] = spellInitials(name)
		if ok && slices.Contains(rows, uint16(id)) {
			grid.Ticked = append(grid.Ticked, v)
		}
		if name == s.spell {
			grid.Value = v
		}
	}
	out = append(out, grid)

	book := &ui.InspectionBook{Art: look.art, Grow: x.inspectionUTF8(fmt.Sprintf(x.word("editor.spells.full"), rules.SpellBookPositions())), Remove: "remove",
		Drops: []ui.InspectionDrop{
			{From: ui.DropBook, To: ui.DropBook, Word: "move"},
			{From: ui.DropGrid, To: ui.DropBook, Word: "add"},
			{From: ui.DropBook, To: ui.DropGrid, Word: "remove"},
		}}
	book.Cells = make([]ui.SpellEntry, len(rows))
	for i, id := range rows {
		if id == 0 {
			book.Tips = append(book.Tips, x.inspectionUTF8(x.word("editor.spells.empty")))
			continue
		}
		name := rules.SpellProfile(g)[id-1].Name
		label := x.spellLabel(name, int(id))
		icon := cutBookIcon(look.atlas, ground, id)
		book.Cells[i] = ui.SpellEntry{ID: uint32(id), Name: label, Icon: icon}
		tip := label
		if !slices.Contains(ground, id) {
			tip = x.inspectionUTF8(fmt.Sprintf(x.word("editor.spells.nopicture"), strings.ReplaceAll(name, "_", " ")))
		}
		book.Tips = append(book.Tips, tip)
		if name == s.spell {
			book.Chosen = uint32(id)
		}
	}
	markMovedSpells(book.Cells, rows, ground)
	out = append(out, ui.InspectionField{Key: spellsBookKey, Label: x.word("editor.spells.book"), Book: book})
	return out
}

// chosenSpellFields are the fields of the page's chosen spell: each number
// the mod may set, empty while the game's stands, the game's value its
// hint; its book cell and its switch.
func (s *mapInspectionEdits) chosenSpellFields(base *mapload.Table, d mod.SpellData) []ui.InspectionField {
	x := s.inspector
	g := mapload.SpellGameOf(base)
	id, ok := rules.SpellNumber(g, s.spell)
	if s.spell == "" || !ok {
		return []ui.InspectionField{x.pageLine("chosen", "", x.word("editor.spells.none"))}
	}
	// The install's name is in the font's bytes already.
	head := fmt.Sprintf(x.inspectionUTF8(x.word("editor.spells.spell")), x.spellLabel(s.spell, id))
	out := []ui.InspectionField{{Key: "chosen", ReadOnly: true, Text: true, TextValue: head, Lines: []ui.InspectionLine{{Text: head}}}}
	var game data.Spell
	if spells, err := data.LoadSpells(base.Spells); err == nil && id <= len(spells) {
		game = spells[id-1]
	}
	row := mod.SpellRow{}
	if i := spellRowIndex(d, s.spell); i >= 0 {
		row = d.Rows[i]
	}
	text := func(v mod.SpellInt) string {
		if !v.Set {
			return ""
		}
		return strconv.Itoa(int(v.Val))
	}
	num := func(key, word, value, inherited string) {
		f := workspaceText(spellFieldPrefix+key, x.word(word), value)
		f.Inherit, f.Inherited = true, inherited
		out = append(out, f)
	}
	num("mana", "editor.spells.mana", text(row.Mana), strconv.Itoa(int(game.ManaCost)))
	num("range", "editor.spells.range", text(row.Range), strconv.Itoa(int(game.MaxRange)))
	num("radius", "editor.spells.radius", text(row.Radius), strconv.Itoa(int(game.Radius)))
	num("duration", "editor.spells.duration", text(row.Duration), strconv.Itoa(int(game.SpellDuration)))
	num("area_duration", "editor.spells.areaduration", text(row.AreaLife), strconv.Itoa(int(game.AreaDuration)))
	damage := ""
	if row.DamageMax.Set {
		damage = fmt.Sprintf("%d-%d", row.DamageMin.Val, row.DamageMax.Val)
	}
	num("damage", "editor.spells.damage", damage, fmt.Sprintf("%d-%d", game.DamageMin, game.DamageMax))
	slot := ""
	if p, ok := rules.GameSpellOf(g, id); ok && p.Slot != 0 {
		slot = strconv.Itoa(int(p.Slot))
	}
	num("book_position", "editor.spells.cell", text(row.BookPosition), slot)
	off := ui.InspectionField{Key: spellFieldPrefix + mod.OffField, Label: x.word("editor.spells.off"), Meaning: x.word("editor.spells.off.help"), Value: 1,
		Choices: []ui.InspectionChoice{{Value: 1, Label: x.inspectionUTF8(x.word("editor.spells.no"))}, {Value: 2, Label: x.inspectionUTF8(x.word("editor.spells.off"))}}}
	if row.Off.Set && row.Off.Val {
		off.Value = 2
	}
	return append(out, off)
}

// spellGlobalFields are the page's globals in the right column: one button
// that folds them, and while unfolded each ratio and the book's cells.
func (s *mapInspectionEdits) spellGlobalFields(g mod.SpellGlobal) []ui.InspectionField {
	x := s.inspector
	out := []ui.InspectionField{{Key: spellsGlobalsKey, Label: x.word("editor.spells.all"), Action: true, Column: 1}}
	if !s.spellGlobals {
		return out
	}
	ratio := func(v mod.SpellRatio) string {
		switch {
		case !v.Set:
			return ""
		case v.Den == 1:
			return strconv.FormatInt(v.Num, 10)
		}
		return fmt.Sprintf("%d/%d", v.Num, v.Den)
	}
	for _, r := range spellRatios(&g) {
		f := workspaceText(spellGlobalKey+r.key, x.word(r.word), ratio(*r.v))
		f.Inherit, f.Inherited, f.Meaning, f.Column = true, "1", x.word("editor.spells.ratio"), 1
		out = append(out, f)
	}
	size := ""
	if g.BookSize.Set {
		size = strconv.Itoa(int(g.BookSize.Val))
	}
	f := workspaceText(spellGlobalKey+"book_size", x.word("editor.spells.booksize"), size)
	f.Inherit, f.Inherited, f.Column = true, strconv.Itoa(rules.GameProfile(mapload.SpellGameOf(x.contextBase().table)).BookSize), 1
	return append(out, f)
}

// spellRatio is one global ratio of the spells file: its key, the word
// that labels it and the value.
type spellRatio struct {
	key, word string
	v         *mod.SpellRatio
}

// spellRatioWords are the global ratios' keys and the words that label
// them, in the page's order.
var spellRatioWords = [...][2]string{{"damage_mul", "editor.spells.damage"}, {"heal_mul", "editor.spells.heal"},
	{"mana_mul", "editor.spells.mana"}, {"range_mul", "editor.spells.range"},
	{"radius_mul", "editor.spells.radius"}, {"duration_mul", "editor.spells.duration"}}

// spellRatios are the global ratios of g.
func spellRatios(g *mod.SpellGlobal) []spellRatio {
	values := [len(spellRatioWords)]*mod.SpellRatio{&g.Damage, &g.Heal, &g.Mana, &g.Range, &g.Radius, &g.Duration}
	out := make([]spellRatio, len(values))
	for i, w := range spellRatioWords {
		out[i] = spellRatio{w[0], w[1], values[i]}
	}
	return out
}

// spellRowIndex is the place of the row of d naming catalog spell name,
// -1 for none.
func spellRowIndex(d mod.SpellData, name string) int {
	return slices.IndexFunc(d.Rows, func(r mod.SpellRow) bool { return rules.SpellKey(r.Target) == rules.SpellKey(name) })
}

// spellRow is the row of d naming catalog spell name, a new one at the end
// when d has none.
func spellRow(d *mod.SpellData, name string) *mod.SpellRow {
	if i := spellRowIndex(*d, name); i >= 0 {
		return &d.Rows[i]
	}
	d.Rows = append(d.Rows, mod.SpellRow{Target: rules.SpellKey(name)})
	return &d.Rows[len(d.Rows)-1]
}

// placeSpell writes catalog spell name of game g into book cell n: on, at
// n, the row's position left out where n is the game's own cell for it.
func placeSpell(d *mod.SpellData, g rules.SpellGame, name string, n int) {
	r := spellRow(d, name)
	r.Off = mod.SpellBool{}
	r.BookPosition = mod.SpellInt{Set: true, Val: int32(n), Line: r.BookPosition.Line}
	if id, ok := rules.SpellNumber(g, name); ok {
		if p, _ := rules.GameSpellOf(g, id); int(p.Slot) == n {
			r.BookPosition = mod.SpellInt{}
		}
	}
}

// takeSpellOut switches catalog spell name off: out of the book and the
// shops.
func takeSpellOut(d *mod.SpellData, name string) {
	r := spellRow(d, name)
	r.BookPosition = mod.SpellInt{}
	r.Off = mod.SpellBool{Set: true, Val: true, Line: r.Off.Line}
}

// spellRowText is what a gesture wrote for catalog spell name: its row's
// switch and book position as the file spells them, asGame where the row
// leaves both to the game.
func spellRowText(d mod.SpellData, name, asGame string) string {
	i := spellRowIndex(d, name)
	switch {
	case i >= 0 && d.Rows[i].Off.Set && d.Rows[i].Off.Val:
		return rules.SpellKey(name) + ": off = true"
	case i >= 0 && d.Rows[i].BookPosition.Set:
		return fmt.Sprintf("%s: book_position = %d", rules.SpellKey(name), d.Rows[i].BookPosition.Val)
	}
	return rules.SpellKey(name) + ": " + asGame
}

// setSpellsField makes one edit of the spells page. A press on the catalog
// or the book chooses a spell and writes nothing; a drop on the book, the
// Remove button and each field are one draft step of the spells file,
// refused where the game's rule for a mod's book or a row refuses it.
func (s *mapInspectionEdits) setSpellsField(key, value string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	x := s.inspector
	base := x.contextBase().table
	g := mapload.SpellGameOf(base)
	catalog := rules.SpellCatalog()
	value = strings.TrimSpace(value)
	gone := fmt.Errorf(x.word("editor.spells.err.gone"), value)
	project := s.workspace.Project
	if project == nil {
		project = x.mod.document()
	}
	read, err := s.heldRows(project, nil, mod.SpellsFile)
	if err != nil {
		return nil, err
	}
	rows, err := spellsBook(base, read.Spells)
	if err != nil {
		return nil, err
	}
	// cell is the catalog name of the spell at book cell word n, from 1.
	cell := func(word string) (string, bool) {
		n, err := strconv.Atoi(word)
		if err != nil || n < 1 || n > len(rows) || rows[n-1] == 0 {
			return "", false
		}
		return rules.SpellProfile(g)[rows[n-1]-1].Name, true
	}
	position := func(word string) (int, bool) {
		n, err := strconv.Atoi(word)
		return n, err == nil && n >= 1 && n <= len(rows)
	}
	switch key {
	case spellsGridKey:
		v, err := strconv.Atoi(value)
		if err != nil || v < 1 || v > len(catalog) {
			return nil, gone
		}
		s.spell = catalog[v-1]
		return nil, nil
	case spellsGlobalsKey:
		s.spellGlobals = !s.spellGlobals
		return nil, nil
	}
	if err := x.modWritable(); err != nil {
		return nil, err
	}
	var touched []string
	var edit func(d *mod.SpellData) error
	switch {
	case key == spellsBookKey:
		words := strings.Fields(value)
		switch {
		case len(words) == 2 && words[0] == "choose":
			name, ok := cell(words[1])
			if !ok {
				return nil, nil
			}
			s.spell = name
			return nil, nil
		case len(words) == 3 && words[0] == "move":
			name, ok := cell(words[1])
			to, in := position(words[2])
			if !ok || !in {
				return nil, gone
			}
			from, _ := position(words[1])
			other, held := cell(words[2])
			touched = []string{name}
			if held {
				touched = append(touched, other)
			}
			edit = func(d *mod.SpellData) error {
				placeSpell(d, g, name, to)
				if held {
					placeSpell(d, g, other, from)
				}
				return nil
			}
			s.spell = name
		case len(words) == 3 && words[0] == "add":
			v, err := strconv.Atoi(words[1])
			to, in := position(words[2])
			if err != nil || v < 1 || v > len(catalog) || !in {
				return nil, gone
			}
			name := catalog[v-1]
			if _, ok := rules.SpellNumber(g, name); !ok {
				return nil, fmt.Errorf(x.word("editor.spells.lacking"), strings.ReplaceAll(name, "_", " "))
			}
			touched = []string{name}
			// A held cell gives its spell up: it leaves the book as a drop on
			// the catalog writes it.
			out, held := cell(words[2])
			held = held && out != name
			if held {
				touched = append(touched, out)
			}
			edit = func(d *mod.SpellData) error {
				if held {
					takeSpellOut(d, out)
				}
				placeSpell(d, g, name, to)
				return nil
			}
			s.spell = name
		case len(words) == 3 && words[0] == "remove", value == "remove":
			name := s.spell
			if len(words) == 3 {
				var ok bool
				if name, ok = cell(words[1]); !ok {
					return nil, gone
				}
			} else if id, ok := rules.SpellNumber(g, name); !ok || !slices.Contains(rows, uint16(id)) {
				return nil, errors.New(x.word("editor.spells.err.notinbook"))
			}
			touched = []string{name}
			edit = func(d *mod.SpellData) error { takeSpellOut(d, name); return nil }
		default:
			return nil, gone
		}
	case strings.HasPrefix(key, spellFieldPrefix):
		name := s.spell
		if _, ok := rules.SpellNumber(g, name); !ok {
			return nil, errors.New(x.word("editor.spells.none"))
		}
		field := strings.TrimPrefix(key, spellFieldPrefix)
		edit = func(d *mod.SpellData) error { return setSpellRowField(spellRow(d, name), field, value) }
	case strings.HasPrefix(key, spellGlobalKey):
		field := strings.TrimPrefix(key, spellGlobalKey)
		edit = func(d *mod.SpellData) error { return setSpellGlobal(&d.Global, field, value) }
	default:
		return nil, gone
	}
	doc, err := s.editSpells(base, edit)
	if err != nil {
		return nil, err
	}
	s.saveNote = ""
	if len(touched) > 0 {
		after, err := s.heldRows(s.workspace.Project, nil, mod.SpellsFile)
		if err == nil {
			var parts []string
			for _, name := range touched {
				parts = append(parts, spellRowText(after.Spells, name, x.word("editor.spells.asgame")))
			}
			s.saveNote = fmt.Sprintf(x.word("editor.spells.written"), strings.Join(parts, "; "))
		}
	}
	return doc, nil
}

// editSpells is one draft step of the spells file: edit changes its rows,
// a row left setting nothing leaves the file, and the rows must pass the
// game's checks for a mod's spell rows and its book.
func (s *mapInspectionEdits) editSpells(base *mapload.Table, edit func(*mod.SpellData) error) (*ui.InspectionDocument, error) {
	return s.editKind(func(k kindFiles) error {
		return k.editRows(nil, []string{mod.SpellsFile}, func(d *mod.Data) error {
			if err := edit(&d.Spells); err != nil {
				return err
			}
			d.Spells.Rows = slices.DeleteFunc(d.Spells.Rows, func(r mod.SpellRow) bool { return r.Bare() })
			if _, err := spellsBook(base, d.Spells); err != nil {
				return err
			}
			if spells, err := data.LoadSpells(base.Spells); err == nil && len(spells) > 0 {
				for _, r := range d.Spells.Rows {
					if _, msg := mapload.ModSpellRefusal(base, r); msg != "" {
						return errors.New(msg)
					}
				}
			}
			return nil
		})
	})
}

// setSpellRowField sets field of row r to the typed value; empty leaves
// the field to the game.
func setSpellRowField(r *mod.SpellRow, field, value string) error {
	num := func(dst *mod.SpellInt) error {
		if value == "" {
			*dst = mod.SpellInt{}
			return nil
		}
		v, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return fmt.Errorf("%s: %q is not a whole number", field, value)
		}
		*dst = mod.SpellInt{Set: true, Val: int32(v), Line: dst.Line}
		return nil
	}
	switch field {
	case "mana":
		return num(&r.Mana)
	case "range":
		return num(&r.Range)
	case "radius":
		return num(&r.Radius)
	case "duration":
		return num(&r.Duration)
	case "area_duration":
		return num(&r.AreaLife)
	case "book_position":
		if err := num(&r.BookPosition); err != nil || value == "" {
			return err
		}
		r.Off = mod.SpellBool{}
		return nil
	case "damage":
		if value == "" {
			r.DamageMin, r.DamageMax = mod.SpellInt{}, mod.SpellInt{}
			return nil
		}
		lo, hi, ok := strings.Cut(value, "-")
		a, errA := strconv.ParseInt(strings.TrimSpace(lo), 10, 32)
		b, errB := strconv.ParseInt(strings.TrimSpace(hi), 10, 32)
		if !ok || errA != nil || errB != nil {
			return fmt.Errorf("damage: %q is not min-max", value)
		}
		r.DamageMin = mod.SpellInt{Set: true, Val: int32(a), Line: r.DamageMin.Line}
		r.DamageMax = mod.SpellInt{Set: true, Val: int32(b), Line: r.DamageMax.Line}
		return nil
	case mod.OffField:
		switch value {
		case "1":
			r.Off = mod.SpellBool{}
		case "2":
			r.Off, r.BookPosition = mod.SpellBool{Set: true, Val: true, Line: r.Off.Line}, mod.SpellInt{}
		default:
			return fmt.Errorf("%s: %q is not a choice", field, value)
		}
		return nil
	}
	return fmt.Errorf("%s is not a field of the spells page", field)
}

// setSpellGlobal sets global field of g to the typed value: a ratio as
// n or n/d, the book's cells as a whole number; empty leaves it to the
// game.
func setSpellGlobal(g *mod.SpellGlobal, field, value string) error {
	if field == "book_size" {
		if value == "" {
			g.BookSize = mod.SpellInt{}
			return nil
		}
		v, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return fmt.Errorf("book_size: %q is not a whole number", value)
		}
		g.BookSize = mod.SpellInt{Set: true, Val: int32(v), Line: g.BookSize.Line}
		return nil
	}
	for _, r := range spellRatios(g) {
		if r.key != field {
			continue
		}
		if value == "" {
			*r.v = mod.SpellRatio{}
			return nil
		}
		n, d, cut := strings.Cut(value, "/")
		num, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		den := int64(1)
		if err == nil && cut {
			den, err = strconv.ParseInt(strings.TrimSpace(d), 10, 64)
		}
		if err != nil || den <= 0 || num <= 0 || num > mod.SpellMaxRatio || den > mod.SpellMaxRatio {
			return fmt.Errorf("%s: %q is not a ratio n or n/d", field, value)
		}
		*r.v = mod.SpellRatio{Set: true, Num: num, Den: den, Line: r.v.Line}
		return nil
	}
	return fmt.Errorf("%s is not a global of the spells page", field)
}
