package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// panelOut is the notes folder the Map panel witnesses write their text and
// frames into, AGAINROM_EDITOR_FRAMES; empty writes nothing.
func panelOut(t *testing.T, name string, body string) {
	t.Helper()
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	if dir == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// installLanguage is the install's table language, "en" or "ru".
func installLanguage(x *MapInspector) string {
	if x.archives != nil && x.archives.Base.Profile.Language != "" && x.word("editor.panel.title") != "Map" {
		return "ru"
	}
	return "en"
}

// winTriggerNames are the names of the triggers whose actions include an
// instant 4, read from the decoded script without the editor's helpers.
func winTriggerNames(t *testing.T, m *alm.Map) []string {
	t.Helper()
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	win := map[uint32]bool{}
	for i := len(sc.Actions) - 1; i >= 0; i-- {
		win[sc.Actions[i].ID] = sc.Actions[i].Opcode == 4
	}
	var out []string
	for _, tr := range sc.Triggers {
		if slices.ContainsFunc(tr.Acts[:], func(id uint32) bool { return id != 0 && win[id] }) {
			out = append(out, tr.Name)
		}
	}
	return out
}

// editorKeys are the catalogue keys of the installed maps, in list order.
func editorKeys(x *MapInspector) []string {
	var out []string
	for _, e := range x.Maps {
		if e.FromArchive {
			out = append(out, "scenario/"+e.Source)
		} else {
			out = append(out, "loose/"+e.Source)
		}
	}
	return out
}

// mapCensusRow is one installed map as the census reads it.
type mapCensusRow struct {
	key        string
	drops, win int
	mode       uint32
	rules      []string
}

func mapCensus(t *testing.T, x *MapInspector) []mapCensusRow {
	t.Helper()
	var out []mapCensusRow
	for _, key := range editorKeys(x) {
		d, err := x.Open(key)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		m, err := alm.Open(d.SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		r := mapCensusRow{key: key, drops: len(mapload.DropCells(m)), win: len(winTriggerNames(t, m)), mode: m.Meta.Word70}
		for _, f := range d.Edits.(ui.InspectionChecker).Report() {
			if f.Rule == ui.RuleDropCell || f.Rule == ui.RuleVictory {
				r.rules = append(r.rules, f.Rule)
			}
		}
		out = append(out, r)
	}
	return out
}

// TestReleaseMapPanelCensusOfTheInstalledMaps counts, over every installed map
// of the root, the maps without a drop cell, the maps without a trigger that
// runs instant 4 and the maps per +0x70 value, with their names. The two rules
// name exactly those maps. Every campaign map carries 1, so 1 is the
// single-player value a new map is given (ALM-MODE-070); every campaign map is
// won and no loose map is (MISSION-WIN-003).
func TestReleaseMapPanelCensusOfTheInstalledMaps(t *testing.T) {
	x := installedInspector(t)
	rows := mapCensus(t, x)
	var noDrop, noWin []string
	modes := map[uint32][]string{}
	campaignModes := map[uint32]int{}
	for _, r := range rows {
		if r.drops == 0 {
			noDrop = append(noDrop, r.key)
		}
		if r.win == 0 {
			noWin = append(noWin, r.key)
		}
		modes[r.mode] = append(modes[r.mode], r.key)
		campaign := strings.HasPrefix(r.key, "scenario/")
		if campaign {
			campaignModes[r.mode]++
		}
		if got, want := slices.Contains(r.rules, ui.RuleDropCell), r.drops == 0; want && !got {
			t.Errorf("%s has no drop cell and no drop-cell finding", r.key)
		}
		if got, want := slices.Contains(r.rules, ui.RuleVictory), r.win == 0; got != want {
			t.Errorf("%s: %d win triggers, victory finding %v", r.key, r.win, got)
		}
		if campaign && r.win == 0 || !campaign && r.win > 0 {
			t.Errorf("%s: %d win triggers", r.key, r.win)
		}
	}
	if len(campaignModes) != 1 || campaignModes[ui.MapModeSingle] == 0 {
		t.Errorf("campaign +0x70 values %v, want 1 on every campaign map", campaignModes)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "root %s: %d maps (%d campaign)\n", installLanguage(x), len(rows), campaignModes[ui.MapModeSingle])
	fmt.Fprintf(&b, "no drop cell: %d %v\n", len(noDrop), noDrop)
	fmt.Fprintf(&b, "no trigger runs instant 4: %d %v\n", len(noWin), noWin)
	var values []uint32
	for v := range modes {
		values = append(values, v)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	for _, v := range values {
		names := modes[v]
		if v == ui.MapModeSingle {
			names = slices.DeleteFunc(slices.Clone(names), func(k string) bool { return strings.HasPrefix(k, "scenario/") })
			names = append([]string{fmt.Sprintf("%d campaign maps", campaignModes[v])}, names...)
		}
		fmt.Fprintf(&b, "+0x70 = %d: %d %v\n", v, len(modes[v]), names)
	}
	t.Log(b.String())
	panelOut(t, "census-"+installLanguage(x)+".txt", b.String())
}

// TestReleaseMapPanelLinesOfInstalledMaps reads the Map panel of a campaign
// map with a drop cell and a win, of a loose map nothing wins, and of a map
// with neither when the install holds one: each line states the value the
// map's own bytes give, in the install's words.
func TestReleaseMapPanelLinesOfInstalledMaps(t *testing.T) {
	x := installedInspector(t)
	w := func(id string, a ...any) string {
		if len(a) == 0 {
			return x.inspectionUTF8(x.word(id))
		}
		return x.inspectionUTF8(fmt.Sprintf(x.word(id), a...))
	}
	line := func(label, value string) string {
		return strings.Join(strings.Fields(w(label)+" "+value), " ")
	}
	var campaign, loose, bare string
	for _, r := range mapCensus(t, x) {
		switch {
		case campaign == "" && strings.HasPrefix(r.key, "scenario/") && r.drops == 1 && r.win == 1:
			campaign = r.key
		case loose == "" && strings.HasPrefix(r.key, "loose/") && r.drops == 1 && r.win == 0:
			loose = r.key
		case bare == "" && r.drops == 0 && r.win == 0:
			bare = r.key
		}
	}
	if campaign == "" || loose == "" {
		t.Fatalf("no campaign map %q or loose map %q to read", campaign, loose)
	}
	var out strings.Builder
	for _, key := range []string{campaign, loose, bare} {
		if key == "" {
			continue
		}
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		if !e.PanelOpen() {
			t.Fatalf("%s: the Map panel did not open first", key)
		}
		m, err := alm.Open(e.Document().SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		start := w("editor.panel.start.none")
		if cells := mapload.DropCells(m); len(cells) == 1 {
			start = w("editor.panel.start.cell", cells[0].X, cells[0].Y)
		}
		victory := w("editor.panel.victory.none")
		if wins := winTriggerNames(t, m); len(wins) == 1 {
			victory = w("editor.panel.victory.one", wins[0])
		}
		mode := w("editor.mode.single")
		if m.Meta.Word70 != ui.MapModeSingle {
			mode = w("editor.mode.othermulti", m.Meta.Word70)
		}
		got := e.PanelLines()
		for _, want := range []string{
			line("editor.panel.size", fmt.Sprintf("%d x %d", m.Width, m.Height)),
			line("editor.panel.mode", mode),
			line("editor.panel.start", start),
			line("editor.panel.victory", victory),
		} {
			if !slices.Contains(got, want) {
				t.Errorf("%s: no line %q in %q", key, want, got)
			}
		}
		fmt.Fprintf(&out, "%s\n  %s\n", key, strings.Join(got, "\n  "))
	}
	t.Log(out.String())
	panelOut(t, "panel-lines-"+installLanguage(x)+".txt", out.String())
}

// metaWord70 is the type-0 +0x70 word of the map bytes b.
func metaWord70(t *testing.T, b []byte) uint32 {
	t.Helper()
	return binary.LittleEndian.Uint32(b[payloadOffsets(t, b)[0]+0x70:])
}

// TestReleaseMapPanelNewMapAsksTheModeAndWritesItsBytes opens a new map from
// the catalogue through the question in both modes: +0x70 is 1 for
// single-player and 4 for multiplayer, the drop cell is the centre, and the
// Mode line and the Mode field name the mode. Save As writes those bytes.
func TestReleaseMapPanelNewMapAsksTheModeAndWritesItsBytes(t *testing.T) {
	x := installedInspector(t)
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	for _, mode := range []struct {
		button int
		word70 uint32
		word   string
	}{{0, ui.MapModeSingle, "editor.mode.single"}, {1, ui.MapModeMultiplayer, "editor.mode.multi"}} {
		e := x.Editor()
		e.Layout(1280, 800)
		_ = e.Key("new")
		_ = e.Key("clear")
		e.TextInput("64x64")
		_ = e.Key("enter")
		r, ok := e.PromptButtonRect(mode.button)
		c.click(e, r, ok, "the mode button")
		d := e.Document()
		if d == nil || d.Width != 64 || e.Dirty() {
			t.Fatalf("mode %d: the new map did not open clean", mode.word70)
		}
		m, err := alm.Open(d.SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		if got := metaWord70(t, d.SourceBytes); got != mode.word70 || m.Meta.Word70 != mode.word70 {
			t.Fatalf("+0x70 is %d, want %d", got, mode.word70)
		}
		if cells := mapload.DropCells(m); !slices.Equal(cells, []mapload.Cell{{X: 32, Y: 32}}) {
			t.Fatalf("drop cells %v", cells)
		}
		want := strings.Join(strings.Fields(x.inspectionUTF8(x.word("editor.panel.mode"))+" "+x.inspectionUTF8(x.word(mode.word))), " ")
		if !slices.Contains(e.PanelLines(), want) {
			t.Fatalf("no line %q in %q", want, e.PanelLines())
		}
		target := filepath.Join(t.TempDir(), "new.alm")
		_ = e.Key("saveas")
		e.TextInput(target)
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
		saved, err := os.ReadFile(target)
		if err != nil || metaWord70(t, saved) != mode.word70 {
			t.Fatalf("Save As: %v", err)
		}
		sm, _ := alm.Open(saved)
		if cells := mapload.DropCells(sm); !slices.Equal(cells, []mapload.Cell{{X: 32, Y: 32}}) {
			t.Fatalf("saved drop cells %v", cells)
		}
		t.Logf("%s: +0x70 = %d at file offset %#x, drop cell %v", x.word(mode.word), metaWord70(t, saved), payloadOffsets(t, saved)[0]+0x70, mapload.DropCells(sm))
	}
}

// TestReleaseMapPanelSaveIntoModWithBothWarningsWritesTheMap saves a new map
// with neither a start point nor a win into a mod: the notice names both,
// Continue writes the map, and the warnings refuse nothing.
func TestReleaseMapPanelSaveIntoModWithBothWarningsWritesTheMap(t *testing.T) {
	root := newMapMod(t)
	modDir := filepath.Join(root, newMapModID)
	x := installedInspector(t)
	if err := x.SetMod(modDir); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(modDir, "mission", "map.alm")
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.Delete("Action", 0); err != nil {
		t.Fatal(err)
	}
	rules := map[string]bool{}
	for _, f := range s.Report() {
		rules[f.Rule] = !f.Error
	}
	if len(rules) != 2 || !rules[ui.RuleDropCell] || !rules[ui.RuleVictory] {
		t.Fatalf("report rules %v", rules)
	}
	_ = e.Key("saveas")
	if !e.NoticeOpen() {
		t.Fatal("no notice before the save into the mod")
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("the notice wrote the map")
	}
	lines := strings.Join(e.PromptNotes(), "") // a wrapped line keeps its space
	for _, id := range []string{"editor.panel.start.none", "editor.panel.victory.none"} {
		if want := x.inspectionUTF8(x.word(id)); !strings.Contains(lines, want) {
			t.Errorf("the notice %q lacks %q", lines, want)
		}
	}
	// The panel's text under the notice is hidden by it.
	cover, _ := e.PromptRect()
	writeEditorFrame(t, e, filepath.Join(os.Getenv("AGAINROM_EDITOR_FRAMES"), "notice-modsave-"+installLanguage(x)+".png"), os.Getenv("AGAINROM_EDITOR_FRAMES") != "", cover)
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	r, ok := e.PromptButtonRect(0)
	c.click(e, r, ok, "Continue")
	saved, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(saved, s.model.Bytes()) || e.Dirty() {
		t.Fatalf("Continue did not write the map: %v, dirty %v", err, e.Dirty())
	}
	m, _ := alm.Open(saved)
	if len(mapload.DropCells(m)) != 0 || len(winTriggerNames(t, m)) != 0 {
		t.Fatal("the saved map is not the map with both warnings")
	}
}

// TestReleaseMapPanelUntouchedMapsSaveTheirBytes opens every installed map,
// reads its Map panel and its report, and saves it with Save As: the file
// holds the installed bytes, so the panel and its checks write nothing.
func TestReleaseMapPanelUntouchedMapsSaveTheirBytes(t *testing.T) {
	x := installedInspector(t)
	dir := t.TempDir()
	for i, key := range editorKeys(x) {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		before := bytes.Clone(e.Document().SourceBytes)
		if len(e.PanelLines()) == 0 {
			t.Fatalf("%s: no panel", key)
		}
		_ = e.Document().Edits.(ui.InspectionChecker).Report()
		_ = e.ChromeImage()
		if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, before) {
			t.Fatalf("%s: reading the panel changed the map", key)
		}
		target := filepath.Join(dir, fmt.Sprintf("%02d.alm", i))
		_ = e.Key("saveas")
		e.TextInput(target)
		if err := e.Key("enter"); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, before) {
			t.Errorf("%s: Save As wrote other bytes than the installed map's (%v)", key, err)
		}
	}
}

// TestReleaseMapPanelFrames writes the Map panel, the Mode field, the new-map
// question and the notice before test play, rendered headless from the
// install, as PNGs into AGAINROM_EDITOR_FRAMES; without it the same steps
// run. In each frame the installed fonts keep every glyph, except the text an
// open prompt or question covers.
func TestReleaseMapPanelFrames(t *testing.T) {
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	lang := installLanguage(x)
	shot := func(e *ui.MapEditor, name string) {
		t.Helper()
		cover, _ := e.PromptRect()
		writeEditorFrame(t, e, filepath.Join(dir, name+"-"+lang+".png"), dir != "", cover)
	}
	opened := func(key string) *ui.MapEditor {
		t.Helper()
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		return e
	}
	var campaign, loose string
	for _, key := range editorKeys(x) {
		if campaign == "" && strings.HasPrefix(key, "scenario/") {
			campaign = key
		}
		if loose == "" && strings.HasPrefix(key, "loose/") {
			loose = key
		}
	}
	shot(opened(campaign), "panel-campaign")
	e := opened(loose)
	shot(e, "panel-loose")
	_ = e.Key("testplay")
	if e.NoticeOpen() {
		t.Fatal("a notice before test play of a map nothing wins and no other warning")
	}
	w := opened("scenario/50.alm")
	_ = w.Key("testplay")
	if !w.NoticeOpen() {
		t.Fatal("no notice before test play of a map with a start point warning")
	}
	shot(w, "notice-testplay")
	_ = w.Key("escape")
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	n := slices.IndexFunc(e.PanelLines(), func(l string) bool {
		return strings.HasPrefix(l, x.inspectionUTF8(x.word("editor.panel.mode"))+" ")
	})
	r, ok := e.PanelRowRect(n)
	c.click(e, r, ok, "the Mode line")
	shot(e, "mode-field")
	e = opened(newMapSource(80, 80))
	shot(e, "panel-newmap")
	e = x.Editor()
	e.Layout(1280, 800)
	_ = e.Key("new")
	_ = e.Key("enter")
	shot(e, "newmap-question")
}
