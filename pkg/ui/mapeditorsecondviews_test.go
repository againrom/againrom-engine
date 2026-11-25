package ui

import (
	"image"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/scriptprofile"
)

// A host that notes its view's limit has the view switch on and the status
// line say it; a host with no note switches on without one.
func TestPassViewSaysTheHostsNote(t *testing.T) {
	e := newMapEditorFixture(t)
	e.doc.Passability = make([]PassArm, e.doc.Width*e.doc.Height)
	e.doc.PassNote = "editor.view.secondpass"
	_ = e.Key("passview")
	on, _, _ := e.ToolbarState("passview")
	if !on || !e.passabilityShown() || !strings.Contains(e.StatusLine(), e.word("editor.view.secondpass")) {
		t.Fatalf("view on %v, shown %v, status %q", on, e.passabilityShown(), e.StatusLine())
	}
	_ = e.Key("passview")
	e.doc.PassNote = ""
	_ = e.Key("passview")
	if on, _, _ = e.ToolbarState("passview"); !on || !e.passabilityShown() {
		t.Fatalf("a host with no note: on %v, shown %v", on, e.passabilityShown())
	}
}

// A host that shows stored heights draws each number as the stored height; any
// other map counts from the first game's new-map height.
func TestRawHeightNumbersShowTheStoredHeight(t *testing.T) {
	for _, raw := range []bool{false, true} {
		e, h, _ := terrainEditor(t)
		e.doc.RawHeights = raw
		_ = e.Key("heightlabels")
		e.doc.Viewer.cam.Zoom = 2
		e.doc.Viewer.cam.Clamp()
		rects, texts := e.HeightLabels()
		if len(rects) < 50 {
			t.Fatalf("%d numbers", len(rects))
		}
		zero := HeightLabelZero
		if raw {
			zero = 0
		}
		ox, oy := e.mapOrigin()
		for i, r := range rects {
			found := false
			for row := range terrainSide {
				for col := range terrainSide {
					x, y := e.corner(image.Pt(col, row))
					if image.Pt(int(float64(x)+ox)+1, int(float64(y)+oy)+1) == r.Min {
						found = true
						if want := strconv.Itoa(int(h.alt[row*terrainSide+col]) - zero); texts[i] != want {
							t.Fatalf("raw %v: number %d shows %q, want %q", raw, i, texts[i], want)
						}
					}
				}
			}
			if !found {
				t.Fatalf("number %d stands at no vertex", i)
			}
		}
	}
}

// secondTab is a map of the second game's profile with the records a search
// and the legend read.
func secondTab(t *testing.T, source string, players []string) *InspectionDocument {
	t.Helper()
	v, err := NewViewer(source, terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	d := &InspectionDocument{Viewer: v, Source: source, Width: 50, Height: 50, SourceBytes: []byte(source)}
	d.Records = []InspectionRecord{
		{Kind: KindUnit, Index: 0, Section: 6, Cell: image.Pt(10, 10), Spatial: true, Size: image.Pt(1, 1), Label: "orc " + source, Player: 1},
		{Kind: KindStructure, Index: 0, Section: 4, Cell: image.Pt(20, 20), Spatial: true, Size: image.Pt(2, 2), Label: "keep " + source, Player: 1},
		{Kind: KindSack, Index: 0, Section: 8, Cell: image.Pt(12, 12), Spatial: true, Size: image.Pt(1, 1), Label: "sack " + source},
		{Kind: KindAction, Index: 0, Section: 7},
		{Kind: KindTrigger, Index: 0, Section: 7},
	}
	for i, name := range players {
		d.Records = append(d.Records, InspectionRecord{Kind: KindPlayer, Index: i, Section: 5, Label: name})
	}
	d.Script = &InspectionScriptModel{Limits: scriptprofile.Second, Triggers: []InspectionTrigger{{Record: 4, Name: "gate " + source, Left: make([]uint32, 3), Right: make([]uint32, 3), Cmp: make([]uint32, 3), Acts: []uint32{1, 0, 0, 0}}}}
	return d
}

// On a second-game map the search lists its units, structures, sacks and
// triggers, and with two maps open each answers from the map shown, as does
// the legend of the player colours.
func TestSecondGameMapsSearchAndLegendFollowTheShownTab(t *testing.T) {
	first := secondTab(t, "alpha", []string{"Self", "Wolves"})
	second := secondTab(t, "beta", []string{"Crown"})
	e := NewMapEditor(panelFont(), nil, func(string) (*InspectionDocument, error) { return first, nil })
	e.Layout(1920, 1080)
	if err := e.Open("alpha"); err != nil {
		t.Fatal(err)
	}
	if err := e.addOpenMap(second); err != nil {
		t.Fatal(err)
	}
	for n, c := range []struct {
		name    string
		players []string
	}{{"alpha", []string{"Self", "Wolves"}}, {"beta", []string{"Crown"}}} {
		if n == 0 {
			e.ShowOpenMap(0)
		} else {
			e.ShowOpenMap(1)
		}
		_ = e.Key("search")
		e.TextInput(c.name)
		got := e.SearchLines()
		for _, kind := range []string{"orc ", "keep ", "sack "} {
			if !slices.ContainsFunc(got, func(l string) bool { return strings.HasSuffix(l, kind+c.name) || strings.Contains(l, kind+c.name) }) {
				t.Errorf("tab %s: search for its name lists no %q row in\n%s", c.name, kind, strings.Join(got, "\n"))
			}
		}
		other := "alpha"
		if c.name == "alpha" {
			other = "beta"
		}
		if slices.ContainsFunc(got, func(l string) bool { return strings.Contains(l, other) }) {
			t.Errorf("tab %s lists a row of map %s", c.name, other)
		}
		e.searchLeave(true)
		if !e.view.playerColours {
			_ = e.Key("playertint")
		}
		if lines := e.LegendLines(); !slices.Equal(lines, c.players) {
			t.Errorf("tab %s legend %q, want %q", c.name, lines, c.players)
		}
	}
}

// The scripts of a second-game map are searched by their trigger's name.
func TestSecondGameSearchFindsATriggerByName(t *testing.T) {
	d := secondTab(t, "alpha", []string{"Self"})
	e := NewMapEditor(panelFont(), nil, func(string) (*InspectionDocument, error) { return d, nil })
	e.Layout(1920, 1080)
	if err := e.Open("alpha"); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("search")
	e.TextInput("gate")
	if got := e.SearchLines(); len(got) != 1 || !strings.Contains(got[0], "gate alpha") {
		t.Fatalf("search for a trigger's name lists %q", got)
	}
}
