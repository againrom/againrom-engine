package game

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/rules"
	"againrom/pkg/ui"
)

// withSpellTable gives the fixture table a spell row for each spell of the
// first game's catalog, named as the game's table names them.
func withSpellTable(x *MapInspector) {
	rows := dbCollection{{}}
	for _, s := range rules.SpellProfile(rules.SpellGameFirst) {
		rows = append(rows, dbEntry{name: strings.ReplaceAll(s.Name, "_", " "), params: []int32{0, 7, 1, 0, 1, 0, 5, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0}})
	}
	t := *x.table
	t.Spells = rows
	x.table = &t
}

// spellsPage is W turned to its spells page over a new mod folder, in an
// editor window of 1920 x 1080 over the fixture install.
func spellsPage(t *testing.T) (*MapInspector, *ui.MapEditor, string) {
	t.Helper()
	f, e := workspaceFixture(t)
	withInstallTable(f.x)
	withSpellTable(f.x)
	e.Layout(1920, 1080)
	dir := filepath.Join(t.TempDir(), "authored-map")
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	for range 12 {
		_, boxes := e.PopupCanvasBoxes()
		if _, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupGridCell, N: 0}); ok && len(boxes) >= rules.SpellBookPositions() {
			return f.x, e, dir
		}
		if err := e.Key("package"); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("W shows no spells page: %q", e.PopupLines())
	return nil, nil, ""
}

// spellsClock is the time of the next pointer gesture, a second after the
// last, so no two presses make a double click.
var spellsClock = time.Unix(100, 0)

// spellsPointer presses at a, moves to b and releases there, through the
// editor's own pointer.
func spellsPointer(e *ui.MapEditor, a, b image.Rectangle) {
	spellsClock = spellsClock.Add(time.Second)
	p, q := a.Min.Add(a.Size().Div(2)), b.Min.Add(b.Size().Div(2))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, spellsClock)
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, spellsClock)
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y}, spellsClock)
}

// spellsBookCell is the window rectangle of book cell n, from 1.
func spellsBookCell(t *testing.T, e *ui.MapEditor, n int) image.Rectangle {
	t.Helper()
	_, rects := e.PopupCanvasBoxes()
	if n < 1 || n > len(rects) {
		t.Fatalf("no book cell %d among %d boxes", n, len(rects))
	}
	return rects[n-1]
}

// catalogCell is the window rectangle of catalog spell name's grid cell.
func catalogCell(t *testing.T, e *ui.MapEditor, name string) image.Rectangle {
	t.Helper()
	i := slices.Index(rules.SpellCatalog(), name)
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupGridCell, N: i})
	if i < 0 || !ok {
		t.Fatalf("the catalog shows no cell of %s", name)
	}
	return r
}

// spellsSource is the spells file the open mod's project holds.
func spellsSource(e *ui.MapEditor) string {
	return string(e.Document().Edits.(*mapInspectionEdits).workspace.Project.Files[mod.SpellsFile])
}

// contextBook is the book the game builds from the mod's rows: the spell
// number at each cell of the editor's mod context.
func contextBook(t *testing.T, x *MapInspector) []uint16 {
	t.Helper()
	if x.mod == nil || x.mod.context == nil {
		t.Fatal("the editor has no mod context")
	}
	return mapload.SpellGameProfileOf(x.mod.context.table).BookRows()
}

// pageBook is the book the spells page shows: each cell's spell number.
func pageBook(t *testing.T, e *ui.MapEditor) []uint16 {
	t.Helper()
	for _, f := range e.Document().Edits.(*mapInspectionEdits).WorkspaceFields(ui.ModSpellsScope) {
		if f.Book != nil {
			var out []uint16
			for _, c := range f.Book.Cells {
				out = append(out, uint16(c.ID))
			}
			return out
		}
	}
	t.Fatal("the spells page shows no book")
	return nil
}

// spellName is the first game's catalog name of spell number id.
func spellName(id uint16) string { return rules.SpellProfile(rules.SpellGameFirst)[id-1].Name }

// The chosen spell's fields and the globals' button stand above the catalog;
// the catalog lists both games' spells with the whole name as each cell's
// tip, ticks a spell in the book, shuts the other game's spells with a tip
// saying why, and gives each spell its initials for a cell without a picture.
func TestSpellsPageCatalogTicksTheBookAndShutsTheOtherGamesSpells(t *testing.T) {
	_, e, _ := spellsPage(t)
	fields := e.Document().Edits.(*mapInspectionEdits).WorkspaceFields(ui.ModSpellsScope)
	at := func(key string) int {
		return slices.IndexFunc(fields, func(f ui.InspectionField) bool { return f.Key == key })
	}
	grid, globals, chosen := at(spellsGridKey), at(spellsGlobalsKey), at("chosen")
	if grid < 0 || globals < 0 || chosen < 0 || globals > grid || chosen > grid {
		t.Fatalf("chosen %d, globals %d, catalog %d: the fields do not stand above the catalog", chosen, globals, grid)
	}
	f := fields[grid]
	if len(f.Choices) != len(rules.SpellCatalog()) {
		t.Fatalf("the catalog shows %d spells, the catalog of both games has %d", len(f.Choices), len(rules.SpellCatalog()))
	}
	value := func(name string) int64 { return int64(slices.Index(rules.SpellCatalog(), name) + 1) }
	label := func(v int64) string { return f.Choices[v-1].Label }
	if v := value("Ice_Missile"); !slices.Contains(f.Shut, v) || !strings.Contains(label(v), "Ice Missile: ") {
		t.Fatalf("Ice Missile of the second game: shut %v, tip %q", slices.Contains(f.Shut, v), label(v))
	}
	if v := value("Fire_Arrow"); !slices.Contains(f.Ticked, v) || slices.Contains(f.Shut, v) || label(v) != "Fire Arrow" {
		t.Fatalf("Fire Arrow in the book: ticked %v, tip %q", slices.Contains(f.Ticked, v), label(v))
	}
	if v := value("Curse"); slices.Contains(f.Ticked, v) || f.Marks[v] != "C" {
		t.Fatalf("Curse outside the book: ticked %v, initials %q", slices.Contains(f.Ticked, v), f.Marks[v])
	}
	if m := f.Marks[value("Protection_from_Fire")]; m != "PfF" {
		t.Fatalf("Protection from Fire's initials are %q", m)
	}
}

// A book spell dragged onto another book cell swaps with the spell there:
// both rows take the other's position, the note states them, and the book
// the game builds from the mod has the two swapped.
func TestSpellsPageSwapsTwoBookSpellsByDrag(t *testing.T) {
	x, e, _ := spellsPage(t)
	ground := rules.SpellBookRows(rules.SpellGameFirst)
	a, b := spellName(ground[0]), spellName(ground[1])
	spellsPointer(e, spellsBookCell(t, e, 1), spellsBookCell(t, e, 2))
	src := spellsSource(e)
	for _, want := range []string{"target = \"" + a + "\"\nbook_position = 2\n", "target = \"" + b + "\"\nbook_position = 1\n"} {
		if !strings.Contains(src, want) {
			t.Fatalf("the spells file lacks %q:\n%s", want, src)
		}
	}
	book := contextBook(t, x)
	if book[0] != ground[1] || book[1] != ground[0] || !slices.Equal(book[2:], ground[2:]) {
		t.Fatalf("the game's book %v, want the first two of %v swapped", book, ground)
	}
	if !slices.Equal(pageBook(t, e), book) {
		t.Fatalf("the page shows %v, the game builds %v", pageBook(t, e), book)
	}
	if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, a+": book_position = 2; "+b+": book_position = 1") {
		t.Fatalf("the note does not state the rows written:\n%s", lines)
	}
}

// A book spell dragged onto the catalog is switched off: its row writes
// off = true, its cell empties and the shops no longer sell it.
func TestSpellsPageTakesASpellOutByDragOntoTheCatalog(t *testing.T) {
	x, e, _ := spellsPage(t)
	ground := rules.SpellBookRows(rules.SpellGameFirst)
	c := spellName(ground[2])
	spellsPointer(e, spellsBookCell(t, e, 3), catalogCell(t, e, "Curse"))
	if src := spellsSource(e); !strings.Contains(src, "target = \""+c+"\"\noff = true\n") {
		t.Fatalf("the spells file does not switch %s off:\n%s", c, src)
	}
	p := mapload.SpellGameProfileOf(x.mod.context.table)
	if book := p.BookRows(); book[2] != 0 || p.ShopBook(int(ground[2])) || p.ShopScroll(int(ground[2])) {
		t.Fatalf("the game's book %v keeps %s, or a shop sells it", book, c)
	}
}

// A catalog spell the game's book has no cell for, dragged onto an empty
// cell, takes it: its row writes book_position and the game's book holds
// it there.
func TestSpellsPageAddsACatalogSpellIntoAnEmptyCell(t *testing.T) {
	x, e, _ := spellsPage(t)
	curse, _ := rules.SpellNumber(rules.SpellGameFirst, "Curse")
	if p, _ := rules.GameSpellOf(rules.SpellGameFirst, curse); p.Slot != 0 {
		t.Fatalf("Curse has cell %d in the game's book; the test needs a spell without one", p.Slot)
	}
	spellsPointer(e, spellsBookCell(t, e, 4), catalogCell(t, e, "Curse"))
	spellsPointer(e, catalogCell(t, e, "Curse"), spellsBookCell(t, e, 4))
	if src := spellsSource(e); !strings.Contains(src, "target = \"Curse\"\nbook_position = 4\n") {
		t.Fatalf("the spells file does not place Curse:\n%s", src)
	}
	if book := contextBook(t, x); book[3] != uint16(curse) {
		t.Fatalf("the game's book %v holds no Curse at cell 4", book)
	}
	if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, "Curse: book_position = 4") {
		t.Fatalf("the note does not state the row written:\n%s", lines)
	}
}

// A catalog spell dropped on a cell that holds a spell takes the cell: the
// spell there is written off the book as a drop on the catalog writes it and
// is listed in the catalog again, the other cells keep their positions, and
// the page outlines the held cell as it outlines an empty one.
func TestSpellsPageCatalogDropReplacesTheSpellInAHeldCell(t *testing.T) {
	x, e, _ := spellsPage(t)
	ground := rules.SpellBookRows(rules.SpellGameFirst)
	old := spellName(ground[3])
	curse, _ := rules.SpellNumber(rules.SpellGameFirst, "Curse")
	for _, f := range e.Document().Edits.(*mapInspectionEdits).WorkspaceFields(ui.ModSpellsScope) {
		if f.Book == nil {
			continue
		}
		for _, d := range f.Book.Drops {
			if d.Word == "add" && d.Empty {
				t.Fatalf("the catalog drop outlines only empty cells")
			}
		}
	}
	spellsPointer(e, catalogCell(t, e, "Curse"), spellsBookCell(t, e, 4))
	src := spellsSource(e)
	for _, want := range []string{"target = \"Curse\"\nbook_position = 4\n", "target = \"" + old + "\"\noff = true\n"} {
		if !strings.Contains(src, want) {
			t.Fatalf("the spells file lacks %q:\n%s", want, src)
		}
	}
	book := contextBook(t, x)
	want := slices.Clone(ground)
	want[3] = uint16(curse)
	if !slices.Equal(book, want) || !slices.Equal(pageBook(t, e), book) {
		t.Fatalf("the game's book %v, the page %v, want %v", book, pageBook(t, e), want)
	}
	for _, f := range e.Document().Edits.(*mapInspectionEdits).WorkspaceFields(ui.ModSpellsScope) {
		if f.Grid {
			v := int64(slices.Index(rules.SpellCatalog(), old) + 1)
			if slices.Contains(f.Ticked, v) {
				t.Fatalf("%s is still ticked in the catalog", old)
			}
		}
	}
	if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, "Curse: book_position = 4; "+old+": off = true") {
		t.Fatalf("the note does not state both rows:\n%s", lines)
	}
}

// A click on a book cell chooses its spell, and Remove switches the chosen
// spell off; Add map writes nothing on this page.
func TestSpellsPageRemoveButtonTakesTheChosenSpellOut(t *testing.T) {
	x, e, _ := spellsPage(t)
	ground := rules.SpellBookRows(rules.SpellGameFirst)
	c := spellName(ground[4])
	cell := spellsBookCell(t, e, 5)
	spellsPointer(e, cell, cell)
	if s := e.Document().Edits.(*mapInspectionEdits).spell; s != c {
		t.Fatalf("the click chose %q, want %s", s, c)
	}
	for _, n := range []int{1, 2} {
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: n})
		if !ok {
			t.Fatalf("no button %d in %q", n, e.PopupButtons())
		}
		spellsPointer(e, r, r)
		if n == 1 && spellsSource(e) != "" {
			t.Fatalf("Add map wrote:\n%s", spellsSource(e))
		}
	}
	if src := spellsSource(e); !strings.Contains(src, "target = \""+c+"\"\noff = true\n") {
		t.Fatalf("Remove did not switch %s off:\n%s", c, src)
	}
	if book := contextBook(t, x); book[4] != 0 {
		t.Fatalf("the game's book %v keeps %s", book, c)
	}
}

// A mod the page moved, added and removed spells in, saved and run as the
// game runs a mod, gives the book the page shows.
func TestSpellsPageBookIsTheBookOfTheSavedMod(t *testing.T) {
	x, e, dir := spellsPage(t)
	spellsPointer(e, spellsBookCell(t, e, 1), spellsBookCell(t, e, 2))
	spellsPointer(e, spellsBookCell(t, e, 6), catalogCell(t, e, "Curse"))
	spellsPointer(e, catalogCell(t, e, "Drain_Life"), spellsBookCell(t, e, 6))
	spellsPointer(e, spellsBookCell(t, e, 9), catalogCell(t, e, "Curse"))
	shown := pageBook(t, e)
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 3})
	if !ok {
		t.Fatalf("no Save button in %q", e.PopupButtons())
	}
	spellsPointer(e, r, r)
	saved, err := os.ReadFile(filepath.Join(dir, mod.SpellsFile))
	if err != nil || string(saved) != spellsSource(e) {
		t.Fatalf("the saved spells file (%v):\n%s\nwant:\n%s", err, saved, spellsSource(e))
	}
	result, err := modrt.Load([]mod.Entry{mod.Load(dir)}, BaseID(InspectInstall(x.archives.Root)), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	table := *x.contextBase().table
	f := &FrontEnd{InstallResources: InstallResources{Table: &table}}
	if err := f.SetModSpells(result.Data.Spells); err != nil {
		t.Fatal(err)
	}
	if game := spellBookOrder(f.Table, spellBookRows(f.Table)); !slices.Equal(game, shown) {
		t.Fatalf("the game's book %v, the page showed %v", game, shown)
	}
	drain, _ := rules.SpellNumber(rules.SpellGameFirst, "Drain_Life")
	if shown[5] != uint16(drain) || shown[8] != 0 {
		t.Fatalf("the page's book %v lacks the added or keeps the removed spell", shown)
	}
}

// typeSpellsField types value into the spells page's field key and enters
// it, through the editor's keys.
func typeSpellsField(t *testing.T, e *ui.MapEditor, key, value string) {
	t.Helper()
	if !e.FocusModField(key) {
		t.Fatalf("the spells page has no field %s", key)
	}
	_ = e.Key("clear")
	e.TextInput(value)
	if err := e.Key("enter"); err != nil {
		t.Fatalf("%s = %q: %v", key, value, err)
	}
}

// spellsFieldRect is the window rectangle of the spells page's field key.
func spellsFieldRect(t *testing.T, e *ui.MapEditor, key string) image.Rectangle {
	t.Helper()
	i := slices.IndexFunc(e.ObjectFields(), func(f ui.InspectionField) bool { return f.Key == key })
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: i})
	if i < 0 || !ok {
		t.Fatalf("the spells page shows no field %s", key)
	}
	return r
}

// The chosen spell's numbers and the folded globals stand above the
// catalog: a click on a catalog spell chooses it, a number typed writes
// that field of its row, the globals unfold by their button and a ratio
// typed writes the [global] table.
func TestSpellsPageWritesTheChosenSpellsFieldsAndTheGlobals(t *testing.T) {
	_, e, _ := spellsPage(t)
	cell := catalogCell(t, e, "Fire_Arrow")
	spellsPointer(e, cell, cell)
	typeSpellsField(t, e, "spell.mana", "20")
	if !e.FocusModField("globals") || e.Key("enter") != nil {
		t.Fatal("the globals do not unfold")
	}
	typeSpellsField(t, e, "global.damage_mul", "3/2")
	src := spellsSource(e)
	for _, want := range []string{"[[spell]]\ntarget = \"Fire_Arrow\"\nmana = 20\n", "[global]\ndamage_mul = [3, 2]\n"} {
		if !strings.Contains(src, want) {
			t.Fatalf("the spells file lacks %q:\n%s", want, src)
		}
	}
	top := catalogCell(t, e, "Fire_Arrow").Min.Y
	for _, key := range []string{"spell.mana", "global.damage_mul"} {
		if r := spellsFieldRect(t, e, key); r.Max.Y > top {
			t.Fatalf("field %s at %v stands below the catalog's top %d", key, r, top)
		}
	}
}
