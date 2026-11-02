package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// menuEditor is the panels map at 1920x1080 in lang on a host that takes
// every tool of the toolbar's left group.
func menuEditor(t *testing.T, lang string) (*MapEditor, *groupsProbe, *[][]byte) {
	t.Helper()
	e, p, saved := panelsEditor(t, 1920, 1080, lang)
	e.doc.Edits = toolAreaHost{p}
	return e, p, saved
}

// tableMenus are the bar menus whose rows the toolbar's table gives.
var tableMenus = []string{"map", "tools", "view", "window"}

// menuRowKeys are each table menu's row keys in order, "" for a heading.
var menuRowKeys = map[string][]string{
	"map":    mapRowKeys,
	"tools":  {"select", "", "brush.terrain", "brush.height", "brush.objects", "brush.passability", "drawarea"},
	"view":   {"border", "areas", "drops", "footprints", "issues", "heightlabels", "passview", "playertint", "grid", "fit", "report"},
	"window": {"toolbar", "objectspanel", "propertiespanel", "reportpanel"},
}

// menuLabel is the label of the row key of menu, and whether it has one.
func menuLabel(e *MapEditor, menu, key string) (string, bool) {
	for _, it := range e.menuItems(menu) {
		if it.key == key {
			return it.label, true
		}
	}
	return "", false
}

// assertMenuMarks fails unless every row of the three table menus carries
// its toolbar state: the mark while its button shows on, none on an action,
// and disabled while its button is.
func assertMenuMarks(t *testing.T, e *MapEditor, how string) {
	t.Helper()
	for _, menu := range tableMenus {
		for _, it := range e.menuItems(menu) {
			if it.key == "" {
				continue
			}
			on, off, ok := e.ToolbarState(it.key)
			if !ok {
				t.Errorf("%s: %s row %s has no toolbar row", how, menu, it.key)
				continue
			}
			mark := ""
			b, _ := toolRow(it.key)
			if b.kind != toolAction {
				mark = markWord(on, "")
			}
			if !strings.HasPrefix(it.label, mark) || b.kind == toolAction && strings.HasPrefix(it.label, "[") || it.off != off {
				t.Errorf("%s: %s row %s reads %q disabled %v, the toolbar shows on %v disabled %v", how, menu, it.key, it.label, it.off, on, off)
			}
		}
	}
}

// The bar holds the Tools, View and Window menus right after Map, in EN and
// RU, each opening the one drop-down on its rows in order: Select, the
// Brushes heading over the four brushes, Draw trigger area; the eight layer
// switches, grid, fit map and report; the toolbar and the three panels. A
// row reads as its toolbar tip with the mark of its state, and the marks
// follow the state. At 1920x1080 the menus stand in the bar's first row.
func TestMapEditorToolsViewAndWindowMenusListTheToolbarRows(t *testing.T) {
	for _, c := range []struct {
		lang     string
		bar      []string
		heading  string
		terrain  string
		selected string
		toolbar  string
		report   string
		panel    string
	}{
		{"", []string{"Map", "Tools", "View", "Window"}, "Brushes", "[ ] Terrain brush (T)", "[x] Select", "[x] Toolbar", "[ ] Report panel (F8)", "[x] Objects and scenarios panel (F6)"},
		{"russian", []string{"Карта", "Инструменты", "Вид", "Окно"}, "Кисти", "[ ] Кисть ландшафта (T)", "[x] Выбор", "[x] Панель инструментов", "[ ] Панель отчета (F8)", "[x] Панель объектов и сценариев (F6)"},
	} {
		t.Run(fmt.Sprintf("%q", c.lang), func(t *testing.T) {
			e, _, _ := menuEditor(t, c.lang)
			var keys, labels []string
			for _, b := range e.barEntries()[:4] {
				keys, labels = append(keys, b.key), append(labels, b.label)
				if !b.menu || b.disabled {
					t.Errorf("bar entry %s menu %v disabled %v", b.key, b.menu, b.disabled)
				}
			}
			if !slices.Equal(keys, []string{"map", "tools", "view", "window"}) || !slices.Equal(labels, c.bar) {
				t.Fatalf("the bar starts %v %q", keys, labels)
			}
			for _, menu := range tableMenus {
				chromeClick(e, barPoint(t, e, menu))
				b, ok := e.openMenuEntry()
				if !ok || b.key != menu {
					t.Fatalf("the %s entry opened %q", menu, e.menu)
				}
				d := e.menuDropdown(b, true)
				if got := menuKeys(e, menu); !slices.Equal(got, menuRowKeys[menu]) || !d.Open || d.List.Len() != len(got) {
					t.Fatalf("the %s menu holds %v in %d drop-down rows, want %v", menu, got, d.List.Len(), menuRowKeys[menu])
				}
				for i, it := range e.menuItems(menu) {
					row := d.List.Rows()[i]
					if row.Text != it.label || row.Heading != (it.key == "") || row.Choosable != (it.key != "" && !it.off) {
						t.Errorf("%s drop-down row %d is %+v for %+v", menu, i, row, it)
					}
					if r, ok := e.ToolbarRect(it.key); ok {
						tip, _, _ := e.toolTip(center(r))
						if !strings.HasSuffix(it.label, tip) {
							t.Errorf("%s row %s reads %q, its tip %q", menu, it.key, it.label, tip)
						}
					}
				}
				chromeClick(e, barPoint(t, e, menu))
				if e.menu != "" {
					t.Fatalf("a second press left %q open", e.menu)
				}
			}
			for _, w := range []struct{ menu, key, want string }{
				{"tools", "brush.terrain", c.terrain}, {"tools", "select", c.selected}, {"window", "toolbar", c.toolbar},
				{"window", "reportpanel", c.report}, {"window", sidePanels[panelLeft].key, c.panel},
			} {
				if got, _ := menuLabel(e, w.menu, w.key); got != w.want {
					t.Errorf("%s row %s reads %q, want %q", w.menu, w.key, got, w.want)
				}
			}
			if got := e.menuItems("tools")[1].label; got != c.heading {
				t.Errorf("the brushes heading reads %q, want %q", got, c.heading)
			}
			assertMenuMarks(t, e, "opening")
			modesClick(e, toolPoint(t, e, "grid"))
			_ = e.Key("terrain")
			_ = e.Key(sidePanels[panelLeft].key)
			_ = e.Key(sidePanels[panelBottom].key)
			assertMenuMarks(t, e, "grid, terrain, F6 and F8")
			for _, w := range []struct{ menu, key, mark string }{
				{"view", "grid", "[x] "}, {"tools", "brush.terrain", "[x] "}, {"tools", "select", "[ ] "},
				{"window", sidePanels[panelLeft].key, "[ ] "}, {"window", "reportpanel", "[x] "},
			} {
				if got, _ := menuLabel(e, w.menu, w.key); !strings.HasPrefix(got, w.mark) {
					t.Errorf("%s row %s reads %q, want the mark %q", w.menu, w.key, got, w.mark)
				}
			}
			font := wordsFont("")
			if c.lang != "" {
				font = wordsFont("ru")
			}
			wide := NewMapEditor(font, nil, nil)
			wide.SetWords(words.For(c.lang))
			wide.Layout(1920, 1080)
			bar := wide.barEntries()
			for _, b := range bar[:4] {
				if b.rect.Min.Y != bar[0].rect.Min.Y {
					t.Errorf("at 1920x1080 the %s entry stands at %v, off the first bar row", b.key, b.rect)
				}
			}
			// The fixture font gives a Cyrillic letter twice a Latin one's
			// width, so only the English bar is held to one row.
			if c.lang == "" && wide.barHeight() != wide.barRowHeight() {
				t.Errorf("at 1920x1080 the bar is %d high, one row is %d", wide.barHeight(), wide.barRowHeight())
			}
		})
	}
}

// A map that takes no brush shows the brush rows disabled, as the toolbar
// shows their buttons; a press on one chooses nothing. With no map open
// every row is disabled where its button is.
func TestMapEditorMenuRowsAreDisabledWhereTheirButtonsAre(t *testing.T) {
	e, _ := newModesEditor(t, "plain")
	e.Layout(1920, 1080)
	assertMenuMarks(t, e, "a map that takes no brush")
	chromeClick(e, barPoint(t, e, "tools"))
	b, _ := e.openMenuEntry()
	rows := e.menuDropdown(b, true).List.Rows()
	for _, key := range []string{"brush.terrain", "brush.height", "brush.objects", "brush.passability"} {
		n := menuKeyRow(t, e, "tools", key)
		if _, off, _ := e.ToolbarState(key); !off || rows[n].Choosable || rows[n].Ink != popupHintInk {
			t.Errorf("%s row is %+v, its button disabled %v", key, rows[n], off)
		}
	}
	chromeClick(e, menuPoint(t, e, menuKeyRow(t, e, "tools", "brush.terrain")))
	if e.currentTool() != "select" || e.brush.mode != brushOff {
		t.Fatalf("a disabled row chose %s", e.currentTool())
	}
	none := NewMapEditor(panelFont(), nil, nil)
	none.Layout(1920, 1080)
	assertMenuMarks(t, none, "no map")
}

// A press on each menu row changes what a press on its toolbar button
// changes, or its key for a row with no button; the toolbar row hides the
// toolbar. A second press puts it back. Every row of the table is pressed,
// in EN and RU.
func TestMapEditorMenuRowPressDoesWhatItsButtonDoes(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, b := range toolButtons {
			if b.menu == "map" {
				continue
			}
			t.Run(fmt.Sprintf("%q/%s", lang, b.key), func(t *testing.T) {
				want, wp, _ := menuEditor(t, lang)
				got, gp, _ := menuEditor(t, lang)
				for press := 1; press <= 2; press++ {
					shown := got.toolbarOn()
					switch {
					case b.onToolbar():
						modesClick(want, toolPoint(t, want, b.key))
					case b.keyRow != "":
						_ = want.Key(b.keyRow)
					}
					openMenuRow(t, got, b.menu, b.key)
					if b.key == "toolbar" {
						if got.toolbarOn() == shown {
							t.Fatalf("press %d on the toolbar row left it shown %v", press, shown)
						}
						continue
					}
					if w, g := toolState(want, wp.calls), toolState(got, gp.calls); w != g {
						t.Fatalf("press %d on the %s row differs from its button\n got  %s\n want %s", press, b.key, g, w)
					}
				}
			})
		}
	}
}

// Every toolbar button, and every row with no button, is a row of the
// Tools, View or Window menu, and those menus hold no row the table does
// not: a button added to the table fails here until it names its menu.
func TestMapEditorEveryToolbarButtonIsAMenuRow(t *testing.T) {
	e, _, _ := menuEditor(t, "")
	bar := map[string]bool{}
	for _, b := range e.barEntries() {
		bar[b.key] = b.menu
	}
	table := map[string]bool{}
	for _, b := range toolButtons {
		table[b.key] = true
		if !slices.Contains(tableMenus, b.menu) || !bar[b.menu] {
			t.Errorf("row %s names the menu %q, not a bar menu the table fills", b.key, b.menu)
			continue
		}
		if !slices.Contains(menuKeys(e, b.menu), b.key) {
			t.Errorf("row %s is not in the %s menu", b.key, b.menu)
		}
	}
	ru, en := words.For("russian"), words.For("")
	for _, menu := range tableMenus {
		for _, k := range menuKeys(e, menu) {
			if k != "" && !table[k] {
				t.Errorf("the %s menu holds %s, which the table does not", menu, k)
			}
		}
		word := "editor.bar." + menu
		if en.Text(word) == word || ru.Text(word) == word || en.Text(word) == ru.Text(word) {
			t.Errorf("word %s is not in both tables", word)
		}
	}
	for _, b := range toolButtons {
		if b.headingWord != "" && (en.Text(b.headingWord) == b.headingWord || ru.Text(b.headingWord) == b.headingWord || en.Text(b.headingWord) == ru.Text(b.headingWord)) {
			t.Errorf("heading word %s is not in both tables", b.headingWord)
		}
	}
}

// The Window menu's panel rows toggle their panels as F6, F7 and F8 do. Its
// toolbar row hides the toolbar, docked or floating: while hidden the map
// and the side panels start right under the bar, as while it floats, and no
// press or tip reaches a button; a second press shows it where it stood.
func TestMapEditorWindowMenuTogglesEachPanelAndTheToolbar(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		t.Run(fmt.Sprintf("%q", lang), func(t *testing.T) {
			e, _, _ := menuEditor(t, lang)
			for i, p := range sidePanels {
				was := e.panelOn(i)
				openMenuRow(t, e, "window", p.key)
				if e.panelOn(i) == was {
					t.Errorf("the %s row left panel %d shown %v", p.key, i, was)
				}
				openMenuRow(t, e, "window", p.key)
				if e.panelOn(i) != was {
					t.Errorf("a second press on the %s row left panel %d shown %v", p.key, i, !was)
				}
			}
			bar, mapDocked := e.toolbarRect(), e.mapArea()
			panels := [2]image.Rectangle{e.sidePanelRect(panelLeft), e.sidePanelRect(panelRight)}
			button, _ := e.ToolbarRect("select")
			openMenuRow(t, e, "window", "toolbar")
			top := e.barHeight()
			mapHidden := e.mapArea()
			if e.toolbarOn() || e.toolbarRect().Dy() != 0 || mapHidden.Min.Y != top || e.chromeHit(image.Pt(1000, bar.Max.Y-2)) == popToolbar {
				t.Fatalf("hidden, the toolbar is %v and the map %v, want the map from %d", e.toolbarRect(), mapHidden, top)
			}
			for _, i := range []int{panelLeft, panelRight} {
				if r := e.sidePanelRect(i); r.Min.Y != top {
					t.Errorf("hidden, panel %d stands at %v, want its top at %d", i, r, top)
				}
			}
			if _, ok := e.ToolbarRect("select"); ok {
				t.Error("a hidden toolbar keeps a button")
			}
			if _, _, ok := e.toolTip(center(button)); ok {
				t.Error("a hidden toolbar shows a tip")
			}
			openMenuRow(t, e, "window", "toolbar")
			if again, _ := e.ToolbarRect("select"); !e.toolbarOn() || e.toolbarRect() != bar || e.mapArea() != mapDocked || again != button ||
				e.sidePanelRect(panelLeft) != panels[0] || e.sidePanelRect(panelRight) != panels[1] {
				t.Fatalf("shown again, the toolbar is %v and the map %v, was %v and %v", e.toolbarRect(), e.mapArea(), bar, mapDocked)
			}
			e.undockToolbar(image.Pt(900, 400))
			floating, ok := toolbarFloating(e)
			if !ok {
				t.Fatal("the toolbar did not float")
			}
			if e.mapArea() != mapHidden {
				t.Errorf("floating, the map is %v; hidden it was %v", e.mapArea(), mapHidden)
			}
			openMenuRow(t, e, "window", "toolbar")
			if _, ok := toolbarFloating(e); ok || e.toolbarOn() || e.chromeHit(center(floating.Rect)) == popToolbar || e.mapArea() != mapHidden {
				t.Fatalf("hidden while floating, the window stands %v and the map %v", ok, e.mapArea())
			}
			openMenuRow(t, e, "window", "toolbar")
			if l, ok := toolbarFloating(e); !ok || l.Rect != floating.Rect || e.chromeHit(center(floating.Rect)) != popToolbar {
				t.Fatalf("shown again, the floating toolbar stands at %v, was %v", l.Rect, floating.Rect)
			}
		})
	}
}

// lastToolbarEntry is the toolbar entry of the last profile written.
func lastToolbarEntry(t *testing.T, saved *[][]byte) map[string]json.RawMessage {
	t.Helper()
	if len(*saved) == 0 {
		t.Fatal("nothing was written")
	}
	var all map[string]map[string]json.RawMessage
	if err := json.Unmarshal((*saved)[len(*saved)-1], &all); err != nil {
		t.Fatal(err)
	}
	return all["toolbar"]
}

// The profile keeps a hidden toolbar, docked or floating, and a fresh
// editor reads it back hidden, the floating one at its corner; a file
// without it, from an earlier build or none, reads as shown.
func TestMapEditorHiddenToolbarProfileRoundTrip(t *testing.T) {
	e, _, saved := menuEditor(t, "")
	openMenuRow(t, e, "window", "toolbar")
	entry := lastToolbarEntry(t, saved)
	if string(entry[hiddenEntry]) != "true" || entry[floatEntry] != nil {
		t.Fatalf("the hidden docked toolbar wrote %q", entry)
	}
	f, _, _ := menuEditor(t, "")
	f.SetPanelProfile((*saved)[len(*saved)-1], nil, nil)
	if f.toolbarOn() || f.toolbarFloats() || f.toolbarRect().Dy() != 0 {
		t.Fatalf("read back, the toolbar shows %v floats %v", f.toolbarOn(), f.toolbarFloats())
	}
	openMenuRow(t, e, "window", "toolbar")
	if entry := lastToolbarEntry(t, saved); entry != nil {
		t.Fatalf("the shown toolbar wrote %q", entry)
	}
	e.undockToolbar(image.Pt(900, 400))
	at := e.floatRect(popToolbar)
	openMenuRow(t, e, "window", "toolbar")
	entry = lastToolbarEntry(t, saved)
	if string(entry[hiddenEntry]) != "true" || entry[floatEntry] == nil {
		t.Fatalf("the hidden floating toolbar wrote %q", entry)
	}
	g, _, _ := menuEditor(t, "")
	g.SetPanelProfile((*saved)[len(*saved)-1], nil, nil)
	if g.toolbarOn() || !g.toolbarFloats() || g.floatRect(popToolbar) != at {
		t.Fatalf("read back, the toolbar shows %v floats %v at %v, want %v", g.toolbarOn(), g.toolbarFloats(), g.floatRect(popToolbar), at)
	}
	for name, file := range map[string][]byte{
		"earlier build": []byte(`{"bottom":{"minimized":true,"width":180},"left":{"minimized":false,"width":260},"right":{"minimized":false,"view":"plain","width":520}}`),
		"floating":      []byte(`{"toolbar":{"float":{"x":40,"y":80,"width":90,"height":500}}}`),
		"broken":        []byte(`{"toolbar":{"hidden":"yes"}}`),
		"none":          nil,
	} {
		h, _, _ := menuEditor(t, "")
		h.panels.toolbarHidden = true
		h.SetPanelProfile(file, map[bool]error{true: os.ErrNotExist}[file == nil], nil)
		if !h.toolbarOn() || (name == "floating") != h.toolbarFloats() {
			t.Errorf("%s: the toolbar reads shown %v floats %v", name, h.toolbarOn(), h.toolbarFloats())
		}
	}
}

// toolRow is the table row of key.
func toolRow(key string) (toolButton, bool) {
	for _, b := range toolButtons {
		if b.key == key {
			return b, true
		}
	}
	return toolButton{}, false
}
