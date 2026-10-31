package ui

import (
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/words"
)

// menuKeys are the row keys of menu, "" for a heading.
func menuKeys(e *MapEditor, menu string) []string {
	var out []string
	for _, it := range e.menuItems(menu) {
		out = append(out, it.key)
	}
	return out
}

// The Map menu reads Create new map, Open... (the file dialog), Save, Save
// as..., then Built-in maps... (the installed maps' list) below them, in EN
// and RU.
func TestMapEditorMapMenuNamesItsRowsAndPutsTheBuiltInMapsBelow(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	if got, want := menuKeys(e, "map"), []string{"new", "open", "saveas", "saveto", "catalog", "testplay", "panel", "close"}; !slices.Equal(got, want) {
		t.Fatalf("the Map menu's rows are %v, want %v", got, want)
	}
	for _, c := range []struct {
		lang string
		want []string
	}{
		{"", []string{"Create new map", "Open...", "Save", "Save as...", "Built-in maps..."}},
		{"russian", []string{"Создать новую карту", "Открыть...", "Сохранить", "Сохранить как...", "Встроенные карты..."}},
	} {
		e.SetWords(words.For(c.lang))
		var got []string
		for _, it := range e.menuItems("map")[:5] {
			got = append(got, it.label)
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%q: the Map menu reads %q, want %q", c.lang, got, c.want)
		}
	}
	// The Open row opens the file dialog's route, the Built-in maps row the list.
	chromeClick(e, barPoint(t, e, "map"))
	chromeClick(e, menuPoint(t, e, menuKeyRow(t, e, "map", "catalog")))
	if !e.catalog {
		t.Fatal("Built-in maps... did not show the map list")
	}
}

// The map list shows the new-map sizes, then the single-player maps, then
// the multiplayer maps, each under its own heading.
func TestMapEditorMapListSplitsSingleAndMultiplayerMaps(t *testing.T) {
	e := NewMapEditor(panelFont(), []EditorMap{
		{Key: "new/32x32", Label: "New map 32 x 32"},
		{Key: "loose/Cross.ALM", Label: "Crossroads of Mystery (Cross.ALM)", Multiplayer: true},
		{Key: "scenario/10.alm", Label: "The First Contact (10.alm)"},
		{Key: "scenario/20.alm", Label: "The Way to the Capital (20.alm)"},
	}, nil)
	rows, maps := e.catalogRows()
	var got []string
	for _, r := range rows {
		got = append(got, r.Text)
	}
	want := []string{"Create new map", "  32 x 32", "Single-player maps", "  The First Contact (10.alm)", "  The Way to the Capital (20.alm)",
		"Multiplayer maps", "  Crossroads of Mystery (Cross.ALM)"}
	if !slices.Equal(got, want) {
		t.Fatalf("the map list reads %q, want %q", got, want)
	}
	if !slices.Equal(maps, []int{-1, 0, -1, 2, 3, -1, 1}) {
		t.Fatalf("the rows open maps %v", maps)
	}
	e.SetWords(words.For("russian"))
	rows, _ = e.catalogRows()
	if rows[0].Text != "Создать новую карту" || rows[2].Text != "Карты одиночной игры" || rows[5].Text != "Карты сетевой игры" {
		t.Fatalf("the Russian headings read %q, %q, %q", rows[0].Text, rows[2].Text, rows[5].Text)
	}
}

// The bar lists Map, then the panels Scripts (Scenario), Players, Groups,
// Units, Sacks, Structures, Objects and Trees, then View and Tools; there is
// no Brushes entry.
func TestMapEditorBarListsThePanelsInTheirOrder(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	var got []string
	for _, b := range e.barEntries() {
		got = append(got, b.key)
	}
	want := []string{"map", "scripts", "players", "groups", "units", "sacks", "structures", "objects", "trees", "view", "tools", "undo", "redo", "save"}
	if !slices.Equal(got, want) {
		t.Fatalf("the bar reads %v, want %v", got, want)
	}
	e.SetWords(words.For("russian"))
	var labels []string
	for _, b := range e.barEntries()[1:9] {
		labels = append(labels, b.label)
	}
	if want := []string{"Сценарий", "Игроки", "Группы", "Отряды", "Мешки", "Постройки", "Объекты", "Деревья"}; !slices.Equal(labels, want) {
		t.Fatalf("the Russian panels read %q, want %q", labels, want)
	}
}

// The Objects entry turns the object brush on in its one-kind mode and the
// Trees entry in its Forest mode; a second press turns it off.
func TestMapEditorObjectsAndTreesEntriesOpenTheObjectBrush(t *testing.T) {
	e, _ := passEditor(t)
	chromeClick(e, barPoint(t, e, "objects"))
	if e.brush.mode != brushObjects || e.forestOn() {
		t.Fatalf("Objects: mode %d forest %v", e.brush.mode, e.forestOn())
	}
	chromeClick(e, barPoint(t, e, "trees"))
	if e.brush.mode != brushObjects || !e.forestOn() {
		t.Fatalf("Trees: mode %d forest %v", e.brush.mode, e.forestOn())
	}
	chromeClick(e, barPoint(t, e, "trees"))
	if e.brush.mode != brushOff {
		t.Fatalf("Trees twice: mode %d", e.brush.mode)
	}
}

// The Scenario list's bottom row holds the Drop zones toggle left of Add; it
// turns the drop tool on and off and shows its state. Tools holds neither
// Drop zones nor Draw trigger area; View and L still reach the area tool,
// which on this map without a script says so.
func TestMapEditorScenarioListHoldsTheDropZoneToggle(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	e.showRecords(editorScriptsFilter)
	e.objectAt = image.Pt(0, e.window.Y) // a selected record's popup stands clear of the list
	s, ok := e.listSpec()
	if !ok || !slices.Equal(s.Buttons, []string{"[ ] Drop zones (D)", "Add"}) {
		t.Fatalf("the Scenario list's buttons are %q", s.Buttons)
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	if len(l.Buttons) != 2 || l.Buttons[0].Max.X > l.Buttons[1].Min.X || e.font.Advance(s.Buttons[0]) > l.Buttons[0].Dx() {
		t.Fatalf("the toggle does not stand whole left of Add: %v", l.Buttons)
	}
	modesClick(e, center(l.Buttons[0]))
	if e.pick != pickDrop {
		t.Fatalf("the toggle armed pick %v", e.pick)
	}
	if s, _ := e.listSpec(); e.listKind() != "records" || s.Buttons[0] != "[x] Drop zones (D)" {
		t.Fatalf("the list after the toggle: %q %q", e.listKind(), s.Buttons)
	}
	s, _ = e.listSpec()
	modesClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[0]))
	if e.pick == pickDrop {
		t.Fatal("a second press left the drop tool on")
	}
	e.showRecords(1)
	if s, _ := e.listSpec(); !slices.Equal(s.Buttons, []string{"Add"}) {
		t.Fatalf("the Units list's buttons are %q", s.Buttons)
	}
	for _, key := range []string{"drop", "dropzones", "drawarea"} {
		if slices.Contains(menuKeys(e, "tools"), key) {
			t.Errorf("Tools still lists %s", key)
		}
	}
	if !slices.Contains(menuKeys(e, "view"), "drawarea") {
		t.Error("View no longer lists Draw trigger area")
	}
	_ = e.Key("escape")
	if err := e.Key("drawarea"); err != nil || e.message != e.word("editor.area.noscript") {
		t.Fatalf("L no longer reaches the trigger area tool: %v %q", err, e.message)
	}
}

// Tools holds New trigger, then the brushes under a Brushes heading, then
// random terrain and random heights under a Whole map heading. There is no
// No brush row and no Brushes menu; RU names the terrain brush Ландшафты. A
// press on a heading does nothing.
func TestMapEditorToolsHoldTheBrushesAndTheWholeMapGroup(t *testing.T) {
	e, _ := passEditor(t)
	want := []string{"newtrigger", "", "brush.terrain", "brush.height", "brush.objects", "brush.passability", "", "randomterrain", "randomheights"}
	if got := menuKeys(e, "tools"); !slices.Equal(got, want) {
		t.Fatalf("Tools rows %v, want %v", got, want)
	}
	if e.menuItems("brushes") != nil {
		t.Fatal("a Brushes menu remains")
	}
	for _, c := range []struct{ lang, brushes, terrain, whole, off string }{
		{"", "Brushes", "Terrain", "Whole map", "No brush"},
		{"russian", "Кисти", "Ландшафты", "Вся карта", "Без кисти"},
	} {
		e.SetWords(words.For(c.lang))
		items := e.menuItems("tools")
		if items[1].label != c.brushes || items[2].label != "[ ] "+c.terrain || items[6].label != c.whole {
			t.Errorf("%q: Tools reads %+v", c.lang, items)
		}
		for _, it := range items {
			if strings.Contains(it.label, c.off) {
				t.Errorf("%q: Tools lists %q", c.lang, it.label)
			}
		}
		if e.brushModeWord(brushOff) != "" {
			t.Errorf("%q: the brush off reads %q", c.lang, e.brushModeWord(brushOff))
		}
	}
	chromeClick(e, barPoint(t, e, "tools"))
	b, _ := e.openMenuEntry()
	rows := e.menuDropdown(b, true).List.Rows()
	if rows[1].Choosable || rows[6].Choosable || !rows[2].Choosable {
		t.Fatalf("heading rows are choosable: %+v", rows)
	}
	chromeClick(e, menuPoint(t, e, 1))
	if e.brush.mode != brushOff {
		t.Fatalf("a press on the Brushes heading turned on brush %d", e.brush.mode)
	}
}

// Esc and a right click on the map each clear the brush; in Forest mode the
// right button keeps the brush, since it thins there.
func TestMapEditorEscAndRightClickClearTheBrush(t *testing.T) {
	e, _ := passEditor(t)
	e.Fit()
	at := e.CellPoint(image.Pt(25, 25))
	right := func(down bool) { e.Pointer(Input{CursorX: at.X, CursorY: at.Y, SecondaryDown: down}, time.Unix(1, 0)) }
	for _, mode := range []brushMode{brushTerrain, brushHeight, brushObjects, brushPassability} {
		e.chooseBrush(mode)
		_ = e.Key("escape")
		if e.brush.mode != brushOff {
			t.Fatalf("Esc left brush %d on", mode)
		}
		e.chooseBrush(mode)
		right(true)
		right(false)
		if e.brush.mode != brushOff {
			t.Fatalf("a right click left brush %d on", mode)
		}
	}
	e.objectBrush(true)
	right(true)
	right(false)
	if !e.forestOn() {
		t.Fatal("a right click in Forest mode cleared the brush")
	}
}

// The hero band's labels in the editor: with roles, hero N reads "Hero N:
// role" in the language of the words; without, "hero N". The wizard's
// unit choices and the trigger's targets ask the one builder.
func TestMapEditorHeroLabelsNameTheRole(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	e.doc.Script = &InspectionScriptModel{Targets: map[uint32][]InspectionScriptTarget{}}
	for _, c := range []struct {
		lang, role, with, without, far string
	}{
		{"", "Traitor mage", "Hero 2: Traitor mage", "hero 3", "hero 40"},
		{"russian", "Маг-предатель", "Герой 2: Маг-предатель", "герой 3", "герой 40"},
	} {
		e.SetWords(words.For(c.lang))
		e.SetHeroRoles(map[int64]string{2: c.role})
		if got := e.targetText(scriptUnit, 10002); got != c.with {
			t.Errorf("%q: the trigger names hero 2 %q, want %q", c.lang, got, c.with)
		}
		if got := e.targetText(scriptUnit, 10003); got != c.without {
			t.Errorf("%q: the trigger names hero 3 %q, want %q", c.lang, got, c.without)
		}
		choices := e.wizardChoices(wpUnit, 10040)
		if !slices.Contains(choices, InspectionChoice{10002, c.with}) || !slices.Contains(choices, InspectionChoice{10003, c.without}) ||
			choices[len(choices)-1] != (InspectionChoice{10040, c.far}) {
			t.Errorf("%q: the wizard's unit choices are %v", c.lang, choices)
		}
	}
	e.SetHeroRoles(nil)
	e.SetWords(words.For(""))
	if got := e.targetText(scriptUnit, 10002); got != "hero 2" {
		t.Errorf("without roles hero 2 reads %q", got)
	}
}

// The wizard offers "Unit near unit": check 6 over two units, compared with
// a radius by "is at most", with its hint, in EN and RU.
func TestMapEditorWizardOffersUnitNearUnit(t *testing.T) {
	i := slices.IndexFunc(wizConditions, func(it wizItem) bool { return it.labelWord == "editor.wizard.c.unitnearunit" })
	if i < 0 {
		t.Fatal("no Unit near unit condition")
	}
	it := wizConditions[i]
	types, _, _ := it.itemSlots()
	if it.opcode != 6 || it.cmp != 5 || it.rightParam != 2 || types != [10]uint32{scriptUnit, scriptUnit} || it.hintWord == "" {
		t.Fatalf("the condition reads %+v, slots %v", it, types)
	}
	e, _ := newModesEditor(t, "full")
	if got := e.word(it.labelWord); got != "Unit near unit" {
		t.Errorf("the condition reads %q", got)
	}
	e.SetWords(words.For("russian"))
	if got := e.word(it.labelWord); got != "Отряд рядом с отрядом" {
		t.Errorf("the Russian condition reads %q", got)
	}
}
