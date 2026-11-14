package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// The record below states what the two screens shown over a running mission
// (the campaign documents panel and the town's Shop page) do today: the
// screen after the raise and the arm, the world and animation ticks during a
// held span, the cadence, the save point answers, the keys that reach the map
// while each shows, the screen after the close, and the arm order cases. The
// expected lines were produced by this test before the screens were put on one
// record, and they stay equal after it. Only the four adapter functions that
// name the raise, the open, the close and the standing request change with the
// code they adapt.

type coverFixKind int

const (
	coverFixDocs coverFixKind = iota
	coverFixShop
)

func (k coverFixKind) String() string { return [...]string{"documents", "shop"}[k] }

// coverShopTown is a town whose Shop page opens over the mission and shows
// until the test closes it.
type coverShopTown struct {
	*stubTown
	open bool
}

func (t *coverShopTown) OpenMissionShop() bool { t.open = true; return true }
func (t *coverShopTown) MissionShopOpen() bool { return t.open }

func coverRaise(h *haltFix, k coverFixKind) {
	switch k {
	case coverFixDocs:
		h.v.RaiseCover(CoverDocuments)
	case coverFixShop:
		h.v.RaiseCover(CoverShop)
	}
}

func coverOpenDirect(f *flow, k coverFixKind, back Screen) bool {
	if k == coverFixDocs {
		return f.openCover(CoverDocuments, back)
	}
	return f.openCover(CoverShop, ScreenMap)
}

func coverCloseDirect(f *flow) { f.closeCover() }

func coverBack(f *flow) Screen { return f.cover.back }

func coverStanding(v *Viewer) string {
	return fmt.Sprintf("docs=%v shop=%v", v.coverRequests&(1<<CoverDocuments) != 0, v.coverRequests&(1<<CoverShop) != 0)
}

type coverFix struct {
	*haltFix
	town *coverShopTown
}

func newCoverFix(t *testing.T, docPages int) *coverFix {
	t.Helper()
	h := newHaltFix(t, haltOpts{})
	pages := make([]DocumentPage, docPages)
	for i := range pages {
		pages[i] = DocumentPage{Text: fmt.Sprintf("page %d", i)}
	}
	h.a.flow.docSrc = &fakeDocumentSource{pages: pages}
	h.a.flow.docArt = documentsTestArt()
	town := &coverShopTown{stubTown: &stubTown{rows: []TownRow{{Text: "town", Choosable: true}}}}
	h.a.SetTown(town)
	return &coverFix{haltFix: h, town: town}
}

// arm raises the screen and drives the one frame that takes the request.
func (c *coverFix) arm(k coverFixKind) {
	coverRaise(c.haltFix, k)
	c.frame(haltNeutral())
}

// shut closes the screen the way the player does: the documents panel through
// its own close, the Shop page by the town ending the visit.
func (c *coverFix) shut(k coverFixKind) {
	if k == coverFixDocs {
		coverCloseDirect(c.a.flow)
		return
	}
	c.town.open = false
}

func (c *coverFix) cadenceText() string {
	cc, ok := c.w.lastCadence()
	return fmt.Sprintf("told=%v stopped=%v calls=%d", ok, cc.stopped, len(c.w.calls))
}

func coverKeys() []struct {
	name string
	set  func(in *appInput)
} {
	return []struct {
		name string
		set  func(in *appInput)
	}{
		{"Escape", func(in *appInput) { in.Escape = true }},
		{"Space", func(in *appInput) { in.Panels = true }},
		{"Pause", func(in *appInput) { in.Pause = true }},
		{"PauseText", func(in *appInput) { in.PauseText = true }},
		{"Faster", func(in *appInput) { in.Faster = true }},
		{"Slower", func(in *appInput) { in.Slower = true }},
		{"Unpaced", func(in *appInput) { in.Unpaced = true }},
		{"Paced", func(in *appInput) { in.Paced = true }},
		{"F1", func(in *appInput) { in.Help = true }},
		{"F2", func(in *appInput) { in.QuickSave = true }},
		{"F3", func(in *appInput) { in.QuickLoad = true }},
		{"CtrlF2", func(in *appInput) { in.SaveGame = true }},
		{"CtrlF3", func(in *appInput) { in.LoadGame = true }},
		{"F4", func(in *appInput) { in.QuickSave = true }},
		{"ShiftF4", func(in *appInput) { in.Reveal = true; in.ShiftHeld = true }},
		{"F5", func(in *appInput) { in.QuickSpell[0] = true }},
		{"F6", func(in *appInput) { in.QuickSpell[1] = true }},
		{"F7", func(in *appInput) { in.QuickSpell[2] = true }},
		{"F8", func(in *appInput) { in.QuickSpell[3] = true }},
		{"F9", func(in *appInput) { in.QuickLoad = true }},
		{"F10", func(in *appInput) { in.Grid = true }},
		{"F11", func(in *appInput) { in.Readout = true }},
		{"F12", func(in *appInput) { in.FPS = true }},
	}
}

func coverRecord(t *testing.T) []string {
	var out []string
	add := func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) }
	kinds := []coverFixKind{coverFixDocs, coverFixShop}

	for _, k := range kinds {
		// The screen after the raise and the arm, the held span, the save
		// point answers, and the screen and pace after the close.
		c := newCoverFix(t, 1)
		c.arm(k)
		back := "-"
		if k == coverFixDocs {
			back = coverBack(c.a.flow).String()
		}
		add("%v/armed screen=%v back=%s standing=%s", k, c.a.Screen(), back, coverStanding(c.v))
		add("%v/armed popup=%v docUp=%v msg=%q", k, c.a.flow.popupOpen(), c.v.docUp, c.a.flow.msg)
		anim, worldBefore, ticksBefore := c.v.AnimationCounter(), c.w.world, c.w.ticks
		c.run(haltRun, haltNeutral())
		add("%v/span world=%d anim=%d ticks=%d %s", k, c.w.world-worldBefore, c.v.AnimationCounter()-anim,
			c.w.ticks-ticksBefore, c.cadenceText())
		for kind := saveKind(0); kind < saveKinds; kind++ {
			g := c.a.flow.savePoint(kind)
			add("%v/savepoint kind=%d row=%d shown=%v open=%v", k, kind, g.row, g.shown, g.open())
		}
		worldBefore, anim = c.w.world, c.v.AnimationCounter()
		c.shut(k)
		c.frame(haltNeutral())
		add("%v/closed screen=%v popup=%v docUp=%v msg=%q", k, c.a.Screen(), c.a.flow.popupOpen(), c.v.docUp, c.a.flow.msg)
		add("%v/closed first frame world=%d anim=%d %s", k, c.w.world-worldBefore, c.v.AnimationCounter()-anim, c.cadenceText())
		worldBefore, anim = c.w.world, c.v.AnimationCounter()
		c.frame(haltNeutral())
		add("%v/closed second frame world=%d anim=%d screen=%v", k, c.w.world-worldBefore, c.v.AnimationCounter()-anim, c.a.Screen())

		// Each key, on a fresh armed screen.
		for _, key := range coverKeys() {
			c := newCoverFix(t, 1)
			c.arm(k)
			world, ticks, anim := c.w.world, c.w.ticks, c.v.AnimationCounter()
			in := haltNeutral()
			key.set(&in)
			c.frame(in)
			add("%v/key %s screen=%v world=%d ticks=%d anim=%d %s stopped=%v rung=%d unpaced=%v grid=%v readout=%v fps=%v msg=%q docUp=%v menuUp=%v",
				k, key.name, c.a.Screen(), c.w.world-world, c.w.ticks-ticks, c.v.AnimationCounter()-anim, c.cadenceText(),
				c.a.flow.stopped, c.a.flow.rung, c.a.flow.unpaced, c.v.showGrid, c.v.readoutHidden, c.v.fpsShown,
				c.a.flow.msg, c.v.docUp, c.v.menuUp)
			c.frame(haltNeutral())
			add("%v/key %s then screen=%v", k, key.name, c.a.Screen())
		}
	}

	// The arm order cases.
	c := newCoverFix(t, 1)
	coverRaise(c.haltFix, coverFixDocs)
	coverRaise(c.haltFix, coverFixShop)
	c.frame(haltNeutral())
	add("order both, documents open: screen=%v standing=%s", c.a.Screen(), coverStanding(c.v))
	c.shut(coverFixDocs)
	c.frame(haltNeutral())
	add("order both, documents closed: screen=%v standing=%s shopOpen=%v", c.a.Screen(), coverStanding(c.v), c.town.open)
	c.frame(haltNeutral())
	add("order both, next frame: screen=%v standing=%s shopOpen=%v", c.a.Screen(), coverStanding(c.v), c.town.open)

	c = newCoverFix(t, 0)
	coverRaise(c.haltFix, coverFixDocs)
	coverRaise(c.haltFix, coverFixShop)
	c.frame(haltNeutral())
	add("order both, documents refuse: screen=%v standing=%s shopOpen=%v", c.a.Screen(), coverStanding(c.v), c.town.open)

	for _, k := range kinds {
		c = newCoverFix(t, 1)
		coverRaise(c.haltFix, k)
		coverRaise(c.haltFix, k)
		c.frame(haltNeutral())
		add("order %v twice: screen=%v standing=%s", k, c.a.Screen(), coverStanding(c.v))
		c.shut(k)
		c.frame(haltNeutral())
		c.frame(haltNeutral())
		add("order %v twice, closed: screen=%v standing=%s", k, c.a.Screen(), coverStanding(c.v))
	}

	// A raise with nothing to show leaves the map as it was.
	c = newCoverFix(t, 0)
	c.arm(coverFixDocs)
	add("documents refuse alone: screen=%v standing=%s", c.a.Screen(), coverStanding(c.v))

	// The documents panel armed from the town returns to the town.
	c = newCoverFix(t, 2)
	c.a.flow.screen = ScreenTown
	add("documents from the town: opened=%v screen=%v back=%v", coverOpenDirect(c.a.flow, coverFixDocs, ScreenTown),
		c.a.Screen(), coverBack(c.a.flow))
	coverCloseDirect(c.a.flow)
	add("documents from the town, closed: screen=%v", c.a.Screen())
	return out
}

// testdata/coverrecord.txt is the record taken at the base, one line per fact.
func TestCoverScreensRecord(t *testing.T) {
	got := coverRecord(t)
	raw, err := os.ReadFile("testdata/coverrecord.txt")
	if err != nil {
		t.Fatalf("read the record: %v\nfull record:\n%s", err, strings.Join(got, "\n"))
	}
	var want []string
	for _, l := range strings.Split(string(raw), "\n") {
		if l = strings.TrimRight(l, "\r"); l != "" {
			want = append(want, l)
		}
	}
	gotSet := map[string]bool{}
	for _, l := range got {
		gotSet[l] = true
	}
	wantSet := map[string]bool{}
	for _, l := range want {
		wantSet[l] = true
	}
	var diff []string
	for _, l := range got {
		if !wantSet[l] {
			diff = append(diff, "+ "+l)
		}
	}
	for _, l := range want {
		if !gotSet[l] {
			diff = append(diff, "- "+l)
		}
	}
	if len(got) != len(want) {
		diff = append(diff, fmt.Sprintf("line count got %d want %d", len(got), len(want)))
	}
	if len(diff) > 0 {
		sort.Strings(diff)
		t.Errorf("record differs:\n%s\nfull record:\n%s", strings.Join(diff, "\n"), strings.Join(got, "\n"))
	}
}
