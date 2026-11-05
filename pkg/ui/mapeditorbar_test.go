package ui

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestMapEditorBarLayout(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, _, _ := menuEditor(t, lang)
		var keys []string
		for _, b := range e.barEntries() {
			keys = append(keys, b.key)
		}
		want := []string{"file", "panel", "scripts", "players", "groups", "units", "sacks", "structures", "cellspells", "objects", "trees", "tools", "view", "window"}
		if !slices.Equal(keys, want) {
			t.Fatalf("owner items 1, 4, 5, 6: bar %v, want %v", keys, want)
		}
		entries := e.barEntries()
		if !entries[1].disabled {
			t.Fatal("owner item 1: Map is enabled without a panel host")
		}
		if entries[len(entries)-1].rect.Max.X != e.window.X-4 {
			t.Fatal("owner item 5: the right group does not end at the window edge")
		}
		for i, b := range entries {
			if b.rect.Min.Y != 3 || i > 0 && entries[i-1].rect.Max.X > b.rect.Min.X {
				t.Fatalf("owner item 5: entry %s overlaps or leaves the first row: %v", b.key, b.rect)
			}
		}
		e, _ = mapPanelEditor(t, &mapPanelEdits{facts: playable()}, nil)
		e.Layout(1920, 1080)
		if e.barEntries()[1].disabled {
			t.Fatal("owner item 1: Map is disabled with a panel host")
		}
		chromeClick(e, barPoint(t, e, "panel"))
		if e.listKind() != "" {
			t.Fatal("Map did not close the initially open Map panel")
		}
		chromeClick(e, barPoint(t, e, "panel"))
		if e.listKind() != "panel" {
			t.Fatal("owner item 1: Map did not open the existing Map panel")
		}
	}
}

func TestMapEditorFileCommands(t *testing.T) {
	e, h := popupEditor(t)
	_ = e.Open("popups")
	want := []string{"new", "open", "saveas", "saveto", "catalog", "testplay", "undo", "redo", "search", "records", "", "close"}
	if got := menuKeys(e, "file"); !slices.Equal(got, want) {
		t.Fatalf("owner items 1, 3, 4, 6: File %v, want %v", got, want)
	}
	for _, key := range []string{"undo", "redo"} {
		label, _ := menuLabel(e, "file", key)
		if !strings.Contains(label, "Ctrl+") {
			t.Fatalf("owner item 4: %s omits its keys: %q", key, label)
		}
	}
	label, _ := menuLabel(e, "file", "redo")
	if !strings.Contains(label, "Ctrl+Y") || !strings.Contains(label, "Ctrl+Shift+Z") {
		t.Fatalf("owner item 4: Redo keys %q", label)
	}
	for _, it := range e.menuItems("file") {
		if (it.key == "undo" || it.key == "redo") && !it.off {
			t.Fatalf("owner item 4: empty history enables %s", it.key)
		}
	}
	h.regroup(0, 2)
	openMenuRow(t, e, "file", "undo")
	if h.CanUndo() || !h.CanRedo() {
		t.Fatal("owner item 4: Undo did not take the existing action path")
	}
	openMenuRow(t, e, "file", "redo")
	if !h.CanUndo() || h.CanRedo() {
		t.Fatal("owner item 4: Redo did not take the existing action path")
	}
	keys := menuKeys(e, "tools")
	i := slices.Index(keys, "randomterrain")
	if i < 1 || keys[i-1] != "" || keys[i+1] != "randomheights" {
		t.Fatalf("owner item 3: Tools omits Generate: %v", keys)
	}
}

func TestMapEditorSearchFromFile(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	if _, ok := e.BarRect("search"); ok {
		t.Fatal("owner item 6: Search still occupies the bar")
	}
	openMenuRow(t, e, "file", "search")
	if !e.searchTyping() {
		t.Fatal("owner item 6: File Search did not focus the field")
	}
	e.TextInput("hall")
	if len(e.searchHits()) != 1 {
		t.Fatalf("owner item 6: hall hits %v", searchLabels(e))
	}
	chromeClick(e, menuPoint(t, e, 0))
	if e.selected != 5 || e.searchTyping() {
		t.Fatalf("owner item 6: hit selected %d, typing %v", e.selected, e.searchTyping())
	}
}

func TestMapEditorHeldButtonLook(t *testing.T) {
	for _, kind := range []string{"bar", "file", "toolbar", "popup", "dialog", "menu"} {
		t.Run(kind, func(t *testing.T) {
			e, _, _ := menuEditor(t, "en")
			var r image.Rectangle
			switch kind {
			case "bar":
				r, _ = e.BarRect("units")
			case "file":
				r, _ = e.BarRect("file")
			case "toolbar":
				r, _ = e.ToolbarRect("fit")
			case "popup":
				e.showRecords(editorScriptsFilter)
				r = listLayout(t, e).Buttons[0]
			case "dialog":
				_ = e.Key("new")
				_, l := promptLayout(t, e)
				r = l.Close
			case "menu":
				chromeClick(e, barPoint(t, e, "view"))
				r, _ = e.MenuRowRect(menuKeyRow(t, e, "view", "grid"))
			}
			at := center(r)
			var edge color.RGBA
			look := func(down bool, p image.Point) bool {
				e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: down}, time.Time{})
				sunk := false
				widgetRecord = func(c widgetCall) {
					if b, ok := c.state.(pushButton); ok && c.rect == r && b.sunk() {
						sunk = true
					}
				}
				pic := e.ChromeImage()
				edge = pic.RGBAAt(r.Min.X+2, r.Min.Y)
				widgetRecord = nil
				return sunk
			}
			look(false, at)
			rest := edge
			if !look(true, at) || !look(true, at) {
				t.Fatal("owner item 2: held press is not drawn through the shared painter")
			}
			if edge == rest {
				t.Fatal("owner item 2: the pressed bevel has unchanged pixels")
			}
			if kind == "menu" && !e.view.grid {
				t.Fatal("the menu action did not run on press")
			}
			if kind == "bar" {
				if !look(true, image.Pt(-1, -1)) || !look(true, at) || !look(false, at) {
					t.Fatal("open bar entry lost its persistent pressed look")
				}
			} else if look(true, image.Pt(-1, -1)) || !look(true, at) || look(false, at) {
				t.Fatal("owner item 2: pressed look does not follow inside and release")
			}
		})
	}
}

func TestMapEditorPressLatchDoesNotReturnWithCancelledState(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	openMenuRow(t, e, "file", "new")
	_ = e.Key("escape")
	if e.buttonPress.latch.Holds() {
		t.Fatal("cancelling a dialog restored the press that opened it")
	}
}

func TestMapEditorCoveredButtonDoesNotTakeThePress(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	s := popupSpec{At: image.Pt(100, 100), Width: 300, Size: image.Pt(300, 200), Bounds: image.Rect(0, 0, 1920, 1080)}
	l := layoutPopup(s, e.font, e.bodyFont())
	at := center(l.Rect)
	if !at.In(l.Rect) || at.In(l.Close) {
		t.Fatal("the fixture does not cover a blank popup area")
	}
	for _, overlay := range []string{"popup", "dropdown"} {
		t.Run(overlay, func(t *testing.T) {
			press := buttonPress{capturing: true, pointer: at}
			pic := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
			r := image.Rectangle{Min: at.Sub(image.Pt(10, 10)), Max: at.Add(image.Pt(10, 10))}
			drawPushButton(pic, e.font, pushButton{Rect: r, Press: &press})
			if overlay == "popup" {
				drawPopup(pic, e.font, e.bodyFont(), nil, s, l, editorPopupInk, at, true, &press)
			} else {
				drawDropdown(pic, e.font, nil, dropdown{Rect: r, Edit: &popupRow{Kind: popupEdit}, Press: &press}, nil, at, true)
			}
			press.begin(pic)
			if press.latch.Holds() {
				t.Fatal("a covered button took the press")
			}
		})
	}
}
