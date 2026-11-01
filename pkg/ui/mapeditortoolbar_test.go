package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"os"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// toolAreaHost is the panels map's host with box areas: the draw-area tool
// arms on it.
type toolAreaHost struct{ *groupsProbe }

func (toolAreaHost) AddBoxArea(image.Rectangle, int) (*InspectionDocument, error) { return nil, nil }

// toolEditor is the panels map at 1280 x 800 on a host that takes every
// tool of the toolbar's left group.
func toolEditor(t *testing.T, lang string) (*MapEditor, *groupsProbe) {
	t.Helper()
	e, p, _ := panelsEditor(t, 1280, 800, lang)
	e.doc.Edits = toolAreaHost{p}
	return e, p
}

// toolState is what a toolbar button, a menu row or the key its row names
// changes: the view switches, the tool, the panels, the report, the open
// prompt and Add window, the camera, the message and the host's calls.
func toolState(e *MapEditor, calls *modesCalls) string {
	cam := ""
	if e.doc != nil {
		cam = fmt.Sprintf("%+v", e.doc.Viewer.cam)
	}
	return fmt.Sprintf("view=%+v tool=%s brush=%d pick=%v panels=%v report=%v list=%q prompt=%d/%q add=%q menu=%q cam=%s message=%q calls=%v",
		e.view, e.currentTool(), e.brush.mode, e.pick, e.panelsShown(), e.report.open, e.listKind(), e.prompt.kind, e.prompt.path,
		addKindNow(e), e.menu, cam, e.message, calls.lines)
}

// openMenuRow presses the bar entry or toolbar button that opens menu, then
// the row whose key is key.
func openMenuRow(t *testing.T, e *MapEditor, menu, key string) {
	t.Helper()
	if _, ok := e.ToolbarRect(menu); ok {
		modesClick(e, toolPoint(t, e, menu))
	} else {
		modesClick(e, barPoint(t, e, menu))
	}
	if e.menu != menu {
		t.Fatalf("pressing %s opened menu %q", menu, e.menu)
	}
	modesClick(e, menuPoint(t, e, menuKeyRow(t, e, menu, key)))
}

// scriptsAddTrigger presses the Scenario bar entry, its palette's Add and
// chooses Trigger in the kind step.
func scriptsAddTrigger(t *testing.T, e *MapEditor) {
	t.Helper()
	modesClick(e, barPoint(t, e, "scripts"))
	e.places[popObject].at = image.Pt(0, e.window.Y) // the selected script's popup stands clear of the list
	r, ok := e.ListButtonKeyRect("add")
	if !ok {
		t.Fatal("the Scenario palette has no Add button")
	}
	modesClick(e, center(r))
	if addKindNow(e) != "Script" {
		t.Fatalf("Add opened %q", addKindNow(e))
	}
	e.prompt.path = "2"
	_ = e.Key("enter")
	if addKindNow(e) != "Trigger" {
		t.Fatalf("choosing Trigger opened %q", addKindNow(e))
	}
}

// baseViewToolsRows are the row keys the old View and Tools menus held, with
// the places each has now: the toolbar and the bar menu that lists the
// toolbar's row, or the Map menu, or the Scenario palette's Add.
var baseViewToolsRows = []struct{ key, place, menu string }{
	{"drops", "toolbar", "view"}, {"footprints", "toolbar", "view"}, {"issues", "toolbar", "view"}, {"grid", "toolbar", "view"},
	{"passview", "toolbar", "view"}, {"heightlabels", "toolbar", "view"}, {"border", "toolbar", "view"}, {"areas", "toolbar", "view"},
	{"playertint", "toolbar", "view"}, {"report", "toolbar", "view"}, {"fit", "toolbar", "view"}, {"drawarea", "toolbar", "tools"},
	{"records", "map", ""}, {"objectspanel", "toolbar", "window"}, {"propertiespanel", "toolbar", "window"},
	{"newtrigger", "scripts-add", ""}, {"brush.terrain", "toolbar", "tools"}, {"brush.height", "toolbar", "tools"},
	{"brush.objects", "toolbar", "tools"}, {"brush.passability", "toolbar", "tools"},
	{"randomterrain", "map", ""}, {"randomheights", "map", ""},
}

// (a) Every row key of the old View and Tools menus stands at its places,
// a toolbar button also in the bar menu that lists it, and pressing the
// row's first place does what menuAct does with the key on a fresh editor.
func TestMapEditorViewAndToolsRowsHaveOnePlaceEach(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	places := map[string][]string{}
	for _, b := range toolButtons {
		if b.onToolbar() {
			places[b.key] = append(places[b.key], "toolbar")
		}
	}
	for _, b := range e.barEntries() {
		if !b.menu {
			continue
		}
		for _, it := range e.menuItems(b.key) {
			if it.key != "" {
				places[it.key] = append(places[it.key], b.key)
			}
		}
	}
	if row, ok := addKindFor(editorScriptsFilter); ok {
		for _, st := range row.Steps {
			for _, k := range st.Kinds {
				if k == KindTrigger {
					places["newtrigger"] = append(places["newtrigger"], "scripts-add")
				}
			}
		}
	}
	for _, r := range baseViewToolsRows {
		want := strings.TrimSpace(r.place + " " + r.menu)
		if got := strings.Join(places[r.key], " "); got != want {
			t.Errorf("row %s stands at %q, want %q", r.key, got, want)
		}
	}
	type fixture func(t *testing.T) (*MapEditor, *modesCalls)
	full := func(t *testing.T) (*MapEditor, *modesCalls) { return newModesEditor(t, "full") }
	area := func(t *testing.T) (*MapEditor, *modesCalls) { e, p := toolEditor(t, ""); return e, p.calls }
	land := func(t *testing.T) (*MapEditor, *modesCalls) { e, h, _ := terrainEditor(t); return e, h.calls }
	for _, r := range baseViewToolsRows {
		t.Run(r.key, func(t *testing.T) {
			var open fixture = full
			switch r.key {
			case "drawarea", "objectspanel", "propertiespanel":
				open = area
			case "randomterrain", "randomheights":
				open = land
			}
			want, wantCalls := open(t)
			got, gotCalls := open(t)
			want.menuAct(r.key)
			switch r.place {
			case "toolbar":
				modesClick(got, toolPoint(t, got, r.key))
			case "scripts-add":
				scriptsAddTrigger(t, got)
				if addKindNow(want) != "Trigger" {
					t.Fatalf("menuAct opened %q", addKindNow(want))
				}
				addAdvance(t, want, 6)
				addAdvance(t, got, 6)
				if fmt.Sprint(wantCalls.lines) != fmt.Sprint(gotCalls.lines) || len(gotCalls.lines) == 0 {
					t.Fatalf("Add wrote %v, menuAct %v", gotCalls.lines, wantCalls.lines)
				}
				return
			default:
				openMenuRow(t, got, r.place, r.key)
			}
			if w, g := toolState(want, wantCalls), toolState(got, gotCalls); w != g {
				t.Fatalf("the %s place differs from menuAct\n got  %s\n want %s", r.place, g, w)
			}
		})
	}
}

// (b) The toolbar spans the window under the top bar at each size and panel
// state; its groups stand apart in table order, every button inside it, and
// the map is the area the base editor gave it less the toolbar above and
// the bottom panel below.
func TestMapEditorToolbarRectsAtEachSizeAndPanelState(t *testing.T) {
	for _, size := range []image.Point{{1280, 800}, {1024, 768}, {640, 480}} {
		for _, st := range []struct {
			name                string
			left, right, bottom bool
		}{{"both", true, true, false}, {"left", true, false, false}, {"right", false, true, false}, {"none", false, false, false}, {"bottom", true, true, true}} {
			t.Run(fmt.Sprintf("%dx%d/%s", size.X, size.Y, st.name), func(t *testing.T) {
				e, _, _ := panelsEditor(t, size.X, size.Y, "")
				for i, open := range []bool{st.left, st.right, st.bottom} {
					e.panels.state[i].Restored = open
				}
				e.relayout()
				bar := e.toolbarRect()
				high := dockedToolbarHigh(size.X)
				if want := image.Rect(0, e.barHeight(), size.X, e.barHeight()+high); bar != want {
					t.Fatalf("toolbar %v, want %v", bar, want)
				}
				prev := map[toolGroup]image.Rectangle{}
				for _, b := range e.toolbarEntries() {
					if !b.onToolbar() {
						continue
					}
					r := b.rect
					if side := high - 2*toolbarPad; r.Dx() != side || r.Dy() != side || !r.In(bar) || r.Min.Y != bar.Min.Y+toolbarPad {
						t.Errorf("%s at %v outside %v", b.key, r, bar)
					}
					if p, ok := prev[b.group]; ok && r.Min.X != p.Max.X+toolGap {
						t.Errorf("%s at %v does not follow %v", b.key, r, p)
					}
					prev[b.group] = r
				}
				first, _ := e.ToolbarRect("select")
				last := prev[toolRight]
				layers, _ := e.ToolbarRect("border")
				// The left group stands after the grip, 4 + 8 + 2 pixels in.
				if first.Min.X != 14 || last.Max.X != size.X-4 || layers.Min.X < prev[toolLeft].Max.X+toolGroupGap {
					t.Errorf("groups at %v..%v and %v..%v", first, prev[toolLeft], layers, last)
				}
				shown := e.panelsShown()
				base := image.Rect(shown[panelLeft], e.barHeight(), size.X-shown[panelRight], size.Y-e.barRowHeight())
				want := image.Rect(base.Min.X, base.Min.Y+high, base.Max.X, base.Max.Y-shown[panelBottom])
				if got := e.mapArea(); got != want {
					t.Errorf("map %v, want %v", got, want)
				}
				if got := e.sidePanelRect(panelBottom); got != image.Rect(want.Min.X, want.Max.Y, want.Max.X, base.Max.Y) {
					t.Errorf("bottom panel %v under map %v", got, want)
				}
				for i := range sidePanels {
					if r := e.sidePanelRect(i); r.Overlaps(bar) || r.Overlaps(e.mapArea()) || r.Overlaps(e.statusRect()) {
						t.Errorf("panel %d at %v crosses the toolbar, the map or the status strip", i, r)
					}
					if s, ok := e.stripSpec(i); ok {
						if l := layoutPopup(s, e.font, e.bodyFont()).Rect; !l.In(e.sidePanelRect(i)) {
							t.Errorf("panel %d's strip draws at %v outside %v", i, l, e.sidePanelRect(i))
						}
					}
				}
				if e.statusRect().Min.Y != base.Max.Y {
					t.Errorf("status %v, want its top at %d", e.statusRect(), base.Max.Y)
				}
				for _, b := range e.toolbarEntries() {
					if b.onToolbar() && e.chromeHit(center(b.rect)) != popToolbar {
						t.Errorf("%s does not hit the toolbar", b.key)
					}
				}
			})
		}
	}
}

// leftTools are the left group's keys and the key rows that choose them.
var leftTools = []struct{ key, row string }{
	{"brush.terrain", "terrain"}, {"brush.height", "height"}, {"brush.objects", "objects"},
	{"brush.passability", "passability"}, {"drawarea", "drawarea"},
}

// assertTool fails unless key is the one left-group button that shows on.
func assertTool(t *testing.T, e *MapEditor, key, how string) {
	t.Helper()
	for _, b := range e.toolbarEntries() {
		if b.group == toolLeft && b.shown != (b.key == key) {
			t.Errorf("after %s, %s shows %v; the current tool is %s", how, b.key, b.shown, e.currentTool())
		}
	}
}

// (c) The current-tool mark follows the tool chosen by its button, by its
// key and by MCP; Select, Esc and MCP's escape return to selecting.
func TestMapEditorCurrentToolFollowsButtonKeyAndMCP(t *testing.T) {
	e, _ := toolEditor(t, "")
	assertTool(t, e, "select", "opening")
	back := func(how string) {
		t.Helper()
		modesClick(e, toolPoint(t, e, "select"))
		assertTool(t, e, "select", how+" then Select")
	}
	for _, tool := range leftTools {
		modesClick(e, toolPoint(t, e, tool.key))
		assertTool(t, e, tool.key, "its button")
		back("its button")
		if err := e.Key(tool.row); err != nil {
			t.Fatal(err)
		}
		assertTool(t, e, tool.key, "key "+tool.row)
		_ = e.Key("escape")
		assertTool(t, e, "select", "key "+tool.row+" then Esc")
	}
	p := newMCPPipe(t, e)
	for _, tool := range leftTools {
		p.tool(t, "action_"+tool.row, map[string]any{})
		assertTool(t, e, tool.key, "MCP action_"+tool.row)
		p.tool(t, "action_escape", map[string]any{})
		assertTool(t, e, "select", "MCP action_"+tool.row+" then escape")
		p.tool(t, "key", map[string]any{"name": tool.row})
		assertTool(t, e, tool.key, "MCP key "+tool.row)
		back("MCP key " + tool.row)
	}
}

// A map that takes no brush, and the catalog, show the brush tools disabled
// and a press on one chooses nothing; draw-area needs an editable map.
func TestMapEditorToolbarDisablesWhatTheMapRefuses(t *testing.T) {
	e, _ := newModesEditor(t, "plain")
	for _, tool := range leftTools[:4] {
		if _, off, ok := e.ToolbarState(tool.key); !ok || !off {
			t.Errorf("%s is not disabled on a map that takes no brush", tool.key)
		}
		modesClick(e, toolPoint(t, e, tool.key))
		assertTool(t, e, "select", "a disabled "+tool.key)
	}
	full, _ := newModesEditor(t, "full")
	for _, tool := range leftTools[:4] {
		if _, off, _ := full.ToolbarState(tool.key); off {
			t.Errorf("%s is disabled on a map that takes brushes", tool.key)
		}
	}
	full.catalog = true
	for _, tool := range leftTools[:4] {
		if _, off, _ := full.ToolbarState(tool.key); !off {
			t.Errorf("%s stays enabled in the catalog", tool.key)
		}
	}
	none := NewMapEditor(panelFont(), nil, nil)
	none.Layout(1280, 800)
	for _, b := range toolButtons {
		if _, off, _ := none.ToolbarState(b.key); off != (b.off != nil) {
			t.Errorf("with no map, %s disabled=%v", b.key, off)
		}
	}
}

// Every toolbar button names its word in both tables and the key that does
// the same in its hover tip; its icon is a 16 by 16 bitmap of palette
// letters.
func TestMapEditorToolbarTipsAndIcons(t *testing.T) {
	ru, err := words.Load(words.Engine(), "ru")
	if err != nil {
		t.Fatal(err)
	}
	en := words.For("")
	for _, lang := range []string{"", "russian"} {
		e, _ := toolEditor(t, lang)
		table := en
		if lang != "" {
			table = ru
		}
		for _, b := range toolButtons {
			if en.Text(b.word) == b.word || ru.Text(b.word) == b.word || en.Text(b.word) == ru.Text(b.word) {
				t.Errorf("word %s is not in both tables", b.word)
			}
			want := table.Text(b.word)
			if r := editorKeyByName[b.keyRow]; r != nil && r.label != "" {
				want += " (" + r.label + ")"
			} else if b.keyRow != "" {
				t.Errorf("%s names key row %q with no label", b.key, b.keyRow)
			}
			if !b.onToolbar() {
				continue
			}
			rect, _ := e.ToolbarRect(b.key)
			if tip, at, ok := e.toolTip(center(rect)); !ok || tip != want || at != rect.Min {
				t.Errorf("%s tip %q at %v, want %q", b.key, tip, at, want)
			}
		}
	}
	for _, b := range toolButtons {
		if !b.onToolbar() {
			continue
		}
		if len(b.icon.rows) != 16 {
			t.Errorf("%s icon has %d rows", b.key, len(b.icon.rows))
		}
		for y, row := range b.icon.rows {
			if len(row) != 16 {
				t.Errorf("%s icon row %d is %d wide", b.key, y, len(row))
			}
			for x := 0; x < len(row); x++ {
				if _, ok := iconInk[row[x]]; !ok && row[x] != '.' {
					t.Errorf("%s icon (%d,%d) is %q, neither clear nor a palette letter", b.key, x, y, row[x])
				}
			}
		}
	}
}

// layerSwitches is each layer switch's view state as the toolbar shows it,
// in toolbar order.
var layerSwitches = []struct {
	key string
	on  func(e *MapEditor) bool
}{
	{"border", func(e *MapEditor) bool { return !e.view.hideBorder }},
	{"areas", func(e *MapEditor) bool { return !e.view.hideAreas }},
	{"drops", func(e *MapEditor) bool { return !e.view.hideDrops }},
	{"footprints", func(e *MapEditor) bool { return !e.view.hideFootprints }},
	{"issues", func(e *MapEditor) bool { return !e.view.hideIssues }},
	{"heightlabels", func(e *MapEditor) bool { return e.view.heights }},
	{"passview", func(e *MapEditor) bool { return e.view.passability }},
	{"playertint", func(e *MapEditor) bool { return e.view.playerColours }},
}

// (d) Each map layer is a toolbar switch, in the old Layers menu's order and
// before grid; it shows on exactly while its layer shows, a press flips the
// layer and the next press flips it back, and no button overlaps another at
// any size in either language. No Layers button or menu remains.
func TestMapEditorToolbarHoldsEachLayerSwitch(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {1024, 768}, {1920, 1080}} {
		for _, lang := range []string{"", "russian"} {
			t.Run(fmt.Sprintf("%dx%d/%q", size.X, size.Y, lang), func(t *testing.T) {
				e, _, _ := panelsEditor(t, size.X, size.Y, lang)
				at := map[string]int{}
				entries := e.toolbarEntries()
				for i, b := range entries {
					at[b.key] = i
					for _, o := range entries[:i] {
						if b.rect.Overlaps(o.rect) {
							t.Errorf("%s at %v overlaps %s at %v", b.key, b.rect, o.key, o.rect)
						}
					}
				}
				if _, ok := at["layers"]; ok || e.menuItems("layers") != nil {
					t.Error("a Layers button or menu remains")
				}
				for i, sw := range layerSwitches {
					n, ok := at[sw.key]
					if !ok {
						t.Fatalf("no %s switch", sw.key)
					}
					if b := entries[n]; b.kind != toolSwitch || b.group != toolRight || b.icon == nil || b.word == "" {
						t.Errorf("%s is %+v", sw.key, b)
					}
					if i > 0 && n != at[layerSwitches[i-1].key]+1 {
						t.Errorf("%s stands at %d, not after %s", sw.key, n, layerSwitches[i-1].key)
					}
					if n >= at["grid"] {
						t.Errorf("%s stands at %d, not before grid", sw.key, n)
					}
					if tip, _, ok := e.toolTip(center(entries[n].rect)); !ok || tip == "" || tip == entries[n].word {
						t.Errorf("%s tip %q", sw.key, tip)
					}
				}
				for _, sw := range layerSwitches {
					for press := 0; press < 2; press++ {
						was := sw.on(e)
						if shown, closed, _ := e.ToolbarState(sw.key); shown != was || closed {
							t.Fatalf("%s shows %v disabled %v while the layer is %v", sw.key, shown, closed, was)
						}
						modesClick(e, toolPoint(t, e, sw.key))
						if sw.on(e) == was {
							t.Fatalf("%s press %d left the layer %v", sw.key, press, was)
						}
						if shown, _, _ := e.ToolbarState(sw.key); shown != sw.on(e) {
							t.Fatalf("%s shows %v after the press, the layer is %v", sw.key, shown, sw.on(e))
						}
					}
				}
				e.doc = nil
				for _, sw := range layerSwitches {
					if _, closed, _ := e.ToolbarState(sw.key); !closed {
						t.Errorf("%s is enabled with no map", sw.key)
					}
				}
			})
		}
	}
}

// The layer switches' icons are 16 by 16 and no two read alike; the tips of
// the keyed layers name the key in both languages.
func TestMapEditorLayerIconsDifferAndTipsNameKeys(t *testing.T) {
	seen := map[string]string{}
	for _, b := range toolButtons {
		if !b.onToolbar() {
			continue
		}
		if sz := b.icon.size(); sz != image.Pt(16, 16) {
			t.Errorf("%s icon is %v", b.key, sz)
		}
		sig := strings.Join(b.icon.rows, "/")
		if o, dup := seen[sig]; dup {
			t.Errorf("%s and %s share an icon", b.key, o)
		}
		seen[sig] = b.key
	}
	for _, lang := range []string{"", "russian"} {
		e, _, _ := panelsEditor(t, 1280, 800, lang)
		for key, label := range map[string]string{"heightlabels": "V", "passview": "Z", "playertint": "Y"} {
			if tip, _, _ := e.toolTip(center(toolRect(t, e, key))); !strings.HasSuffix(tip, "("+label+")") {
				t.Errorf("%q: %s tip %q lacks its key %s", lang, key, tip, label)
			}
		}
	}
}

// dragBottomEdge drags the restored bottom panel's top edge by dy.
func dragBottomEdge(t *testing.T, e *MapEditor, dy int) {
	t.Helper()
	id := popReport
	if e.reportDocked() {
		id = popList
	}
	r := e.sidePanelRect(panelBottom)
	x := (r.Min.X + r.Max.X) / 2
	for y := r.Min.Y; y < r.Min.Y+4; y++ {
		if at := image.Pt(x, y); e.panelEdgeAt(id, at) != 0 {
			modesPress(e, at, true)
			modesPress(e, at.Add(image.Pt(0, dy)), true)
			modesPress(e, at.Add(image.Pt(0, dy)), false)
			return
		}
	}
	t.Fatal("no point of the bottom panel's top edge grabs")
}

// keptBottom is the bottom panel as the last profile write keeps it.
func keptBottom(t *testing.T, saved *[][]byte) (minimized bool, height int) {
	t.Helper()
	if len(*saved) == 0 {
		t.Fatal("the panels wrote no profile")
	}
	var kept map[string]struct {
		Minimized bool
		Width     int
	}
	if err := json.Unmarshal((*saved)[len(*saved)-1], &kept); err != nil {
		t.Fatal(err)
	}
	k, ok := kept["bottom"]
	if !ok {
		t.Fatalf("the profile keeps no bottom panel: %s", (*saved)[len(*saved)-1])
	}
	return k.Minimized, k.Width
}

// (e) The bottom panel opens and closes by F8, by its strip's button and by
// the toolbar's Report button, which docks the report in it; its top edge
// resizes it between its least height and the map's least height; the
// profile keeps it, and a missing or broken profile gives its default.
func TestMapEditorBottomPanelOpensResizesAndKeepsItsState(t *testing.T) {
	e, _, saved := panelsEditor(t, 1280, 800, "")
	strip := e.panelStrip(panelBottom)
	check := func(how string, open bool, height int) {
		t.Helper()
		r := e.sidePanelRect(panelBottom)
		if e.panelOpen(panelBottom) != open || r.Dy() != height || r.Max.Y != e.statusRect().Min.Y || e.mapArea().Max.Y != r.Min.Y {
			t.Fatalf("after %s: open=%v rect %v map %v, want open=%v height %d", how, e.panelOpen(panelBottom), r, e.mapArea(), open, height)
		}
	}
	check("opening", false, strip)
	_ = e.Key("reportpanel")
	check("F8", true, 180)
	if m, h := keptBottom(t, saved); m || h != 180 {
		t.Fatalf("F8 kept minimized=%v height %d", m, h)
	}
	if _, ok := e.reportHintSpec(); !ok {
		t.Error("the open panel without a report shows no hint")
	}
	_ = e.Key("reportpanel")
	check("F8 again", false, strip)
	for _, want := range []bool{true, false} {
		r, ok := e.PanelButtonRect(panelBottom)
		if !ok {
			t.Fatal("the bottom panel has no button")
		}
		modesClick(e, center(r))
		h := strip
		if want {
			h = 180
		}
		check("the panel button", want, h)
	}
	modesClick(e, toolPoint(t, e, "report"))
	check("the Report button", true, 180)
	if on, _, _ := e.ToolbarState("report"); !on || !e.reportDocked() {
		t.Fatal("the Report button did not dock the report in the bottom panel")
	}
	modesClick(e, toolPoint(t, e, "report"))
	check("the Report button again", false, strip)
	if on, _, _ := e.ToolbarState("report"); on || e.report.open {
		t.Fatal("the second press left the report open")
	}
	_ = e.Key("reportpanel")
	for _, c := range []struct {
		dy, height int
	}{{-40, 220}, {500, 96}, {-1000, 480}} {
		dragBottomEdge(t, e, c.dy)
		check(fmt.Sprintf("a drag by %d", c.dy), true, c.height)
		if m, h := keptBottom(t, saved); m || h != c.height {
			t.Fatalf("the drag by %d kept minimized=%v height %d", c.dy, m, h)
		}
	}
	kept := (*saved)[len(*saved)-1]
	again, _, _ := panelsEditor(t, 1280, 800, "")
	again.SetPanelProfile(kept, nil, nil)
	if again.panels.state != e.panels.state || again.sidePanelRect(panelBottom) != e.sidePanelRect(panelBottom) {
		t.Fatalf("the profile read back %v, want %v", again.panels.state, e.panels.state)
	}
	for name, file := range map[string]struct {
		data []byte
		err  error
	}{"missing": {nil, os.ErrNotExist}, "broken": {[]byte("{\"bottom\":"), nil}, "unreadable": {[]byte(kept), errors.New("denied")}} {
		again.SetPanelProfile(file.data, file.err, nil)
		if s := again.panels.state[panelBottom]; s != (panelState{Restored: false, Width: 180}) || again.panelOpen(panelBottom) {
			t.Errorf("a %s profile gives the bottom panel %+v", name, s)
		}
	}
	// At its least height the panel still holds the docked report.
	dragBottomEdge(t, e, 1000)
	check("a drag to the least height", true, 96)
	_ = e.Key("report")
	s, ok := e.listSpec()
	if !ok || !e.reportDocked() {
		t.Fatal("R did not dock the report")
	}
	if l := layoutPopup(s, e.font, e.bodyFont()).Rect; !l.In(e.sidePanelRect(panelBottom)) {
		t.Fatalf("the docked report draws at %v outside the panel %v", l, e.sidePanelRect(panelBottom))
	}
}

// (f) The Map menu's Generate group runs random landscape and random
// heights as their keys do, and the Scenario palette's Add, choosing
// Trigger, adds the trigger New trigger added.
func TestMapEditorGenerateAndAddDoWhatTheirRowsDid(t *testing.T) {
	e, _, _ := terrainEditor(t)
	items := e.menuItems("map")
	at := -1
	for i, it := range items {
		if it.key == "" && it.label == e.word("editor.menu.generate") {
			at = i
		}
	}
	if at < 0 || at+2 >= len(items) || items[at+1].key != "randomterrain" || items[at+2].key != "randomheights" {
		t.Fatalf("the Map menu holds no Generate group: %v", items)
	}
	for _, key := range []string{"randomterrain", "randomheights"} {
		byRow, rowHost, _ := terrainEditor(t)
		byKey, keyHost, _ := terrainEditor(t)
		openMenuRow(t, byRow, "map", key)
		if err := byKey.Key(key); err != nil {
			t.Fatal(err)
		}
		if byRow.prompt.kind != promptSeed || byRow.prompt.path != byKey.prompt.path {
			t.Fatalf("the %s row opened prompt %d %q", key, byRow.prompt.kind, byRow.prompt.path)
		}
		_ = byRow.Key("enter")
		_ = byKey.Key("enter")
		if w, g := toolState(byKey, keyHost.calls), toolState(byRow, rowHost.calls); w != g || len(rowHost.calls.lines) != 1 {
			t.Fatalf("the %s row differs from its key\n got  %s\n want %s", key, g, w)
		}
	}
	byAdd, addCalls := newModesEditor(t, "full")
	byRow, rowCalls := newModesEditor(t, "full")
	scriptsAddTrigger(t, byAdd)
	byRow.menuAct("newtrigger")
	for i := 0; i < 6 && addKindNow(byRow) != ""; i++ {
		addAdvance(t, byRow, 1)
		addAdvance(t, byAdd, 1)
	}
	if addKindNow(byAdd) != "" || fmt.Sprint(addCalls.lines) != fmt.Sprint(rowCalls.lines) || len(addCalls.lines) != 1 || !strings.HasPrefix(addCalls.lines[0], "AddRecord Trigger") {
		t.Fatalf("Add wrote %v, New trigger %v", addCalls.lines, rowCalls.lines)
	}
}
