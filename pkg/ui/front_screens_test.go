package ui

import (
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// detailedTooltipApp is an application showing d's detailed generator page
// with a one-pixel skill mask at column point (10,10) for the class's first
// skill and the hover words the tests read.
func detailedTooltipApp(t *testing.T, d *GeneratorDescription) (*App, image.Point) {
	t.Helper()
	stats := make([]ChargenStat, 4)
	for i := range stats {
		stats[i] = ChargenStat{Floor: 0, Ceiling: 50, Start: 10 + i}
	}
	art := &ChargenPresentation{Layout: d, Font: messageFont()}
	art.ColumnMask[0] = chargenMask(320, 480)
	art.ColumnMask[0].SetColorIndex(10, 10, uint8(d.Detail.Classes[0].Skills[0].Mask))
	c := NewChargen(ChargenSetup{
		Name:      "Danath",
		PreCreate: &ChargenPreCreate{Art: art},
		Choices:   []ChargenChoice{{Options: []string{"male", "female"}, Parent: -1}},
		Stats:     stats, Cost: triangular(50), Budget: 5000,
	})
	c.Forward()
	if c.Stage() != DetailedStage {
		t.Fatal("fixture did not reach DetailedStage")
	}
	a := NewApp("chargen", nil, nil, nil)
	a.Layout(640, 480)
	a.flow.chargen, a.flow.screen = c, ScreenChargen
	a.flow.words.Hover[171], a.flow.words.Hover[273] = "SKILL TEXT", "FREE POOL"
	return a, d.Detail.ColumnRect.Rectangle().Min.Add(image.Pt(10, 10))
}

// The first game's weapon pictures open no tooltip; the second game's open
// the text of the slot its description names (HERO-ROUTE-182).
func TestGeneratorSkillPicturesTooltipByDescription(t *testing.T) {
	first, at := detailedTooltipApp(t, testGenerator())
	if got := first.chargenTooltip(at); len(got.lines) != 0 {
		t.Errorf("first game: skill picture tooltip %v, want none", got.lines)
	}
	second, at := detailedTooltipApp(t, secondTestGenerator(t))
	if got := second.chargenTooltip(at); !reflect.DeepEqual(got.lines, tooltipLines("SKILL TEXT")) {
		t.Errorf("second game: skill picture tooltip %v, want its text", got.lines)
	}
}

// The free-points readout shows the installed word of its tooltip slot in
// both games (HERO-ROUTE-182).
func TestGeneratorFreePoolReadoutTooltip(t *testing.T) {
	for name, d := range map[string]*GeneratorDescription{"first": testGenerator(), "second": secondTestGenerator(t)} {
		a, _ := detailedTooltipApp(t, d)
		pool := d.Detail.Stats.Pool.Rectangle()
		got := a.chargenTooltip(pool.Min.Add(pool.Size().Div(2)))
		if !reflect.DeepEqual(got.lines, tooltipLines("FREE POOL")) {
			t.Errorf("%s game: pool readout tooltip %v, want FREE POOL", name, got.lines)
		}
	}
}

// The first game's command captions take the original's tan at rest and
// bright yellow under the pointer, measured from the original's frames of the
// detailed page (HERO-ROUTE-182); the second game's keep their own.
func TestGeneratorCommandInkByDescription(t *testing.T) {
	want := map[string][2]color.RGBA{
		"first":  {{189, 158, 74, 255}, {255, 255, 115, 255}},
		"second": {{185, 159, 73, 255}, {255, 255, 116, 255}},
	}
	for name, d := range map[string]*GeneratorDescription{"first": testGenerator(), "second": secondTestGenerator(t)} {
		a, _ := detailedTooltipApp(t, d)
		for _, hover := range []bool{false, true} {
			ctl := chargenReset
			panel := chargenCommandPanel(a.flow.chargen, map[bool]chargenControl{false: chargenNone, true: ctl}[hover], chargenNone)
			var found bool
			for _, b := range panel.buttons {
				if b.Hover != hover {
					continue
				}
				found = true
				if got := b.ink(); got != want[name][map[bool]int{false: 0, true: 1}[hover]] {
					t.Errorf("%s game hover=%v: caption ink %v, want %v", name, hover, got, want[name][map[bool]int{false: 0, true: 1}[hover]])
				}
			}
			if !found {
				t.Errorf("%s game hover=%v: no button in that state", name, hover)
			}
		}
	}
}

// A Hall of Fame row whose score is zero prints 0 (FAME-SCORE-004).
func TestHallScoreZeroPrintsZero(t *testing.T) {
	for score, want := range map[int32]string{0: "0", 7: "7", 18114: "18,114", -1234: "-1,234"} {
		if got := hallScoreText(score); got != want {
			t.Errorf("hallScoreText(%d) = %q, want %q", score, got, want)
		}
	}
}

// screensCutsceneLayout reads a game's View Cutscenes geometry from its screen
// description file.
func screensCutsceneLayout(t *testing.T, game string) CutsceneLibraryLayout {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "game", "screens", game+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Manifests []struct {
			Screen       string                 `json:"screen"`
			CutsceneList *CutsceneLibraryLayout `json:"cutscene-list"`
		} `json:"manifests"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	for _, m := range d.Manifests {
		if m.Screen == "menu-media" && m.CutsceneList != nil {
			return *m.CutsceneList
		}
	}
	t.Fatalf("%s names no cutscene list geometry", game)
	return CutsceneLibraryLayout{}
}

// The View Cutscenes list draws a rule under every row whose rule lies inside
// the list, empty rows included: the first game's eight row slots, the last cut
// short by the list's bottom, and the second game's eight rows; its buttons
// stand where the game's layout puts them (MENU-ROUTE-163, R2-ENGINE-565).
func TestCutsceneLibraryRulesRowsAndButtonsByLayout(t *testing.T) {
	for game, rows := range map[string]int{"rom1": 8, "rom2": 8} {
		layout := screensCutsceneLayout(t, game)
		if layout.Rows != rows {
			t.Fatalf("%s: %d rows, want %d", game, layout.Rows, rows)
		}
		a := NewApp("cutscenes", nil, nil, nil)
		a.Layout(640, 480)
		a.flow.menuFont = shippedShapeFont()
		a.SetCutsceneLibraryLayout(layout)
		a.SetCutsceneLibrary([]CutsceneEntry{{"a", "A"}}, []string{"a"}, nil, CutsceneLibraryWords{})
		a.openCutsceneLibrary()
		box := a.movieListBox()
		if box.Rows != rows || !box.Ruled || box.Pitch != 19 {
			t.Fatalf("%s: list box %d rows ruled %v", game, box.Rows, box.Ruled)
		}
		pic := a.composeCutsceneLibrary()
		x := box.Rect.Min.X + 3*box.Rect.Dx()/4
		for i := 0; i < rows; i++ {
			r := box.Row(i)
			light := pic.RGBAAt(x, r.Max.Y-3) == listRuleLight
			dark := pic.RGBAAt(x, r.Max.Y-1) == listRuleDark
			want := i < rows-1 || game == "rom2"
			if light != want || dark != want {
				t.Errorf("%s: row %d rule light=%v dark=%v, want %v", game, i, light, dark, want)
			}
		}
		for button, r := range map[int]image.Rectangle{movieOKButton: layout.OK.Rectangle(), movieCancelButton: layout.Cancel.Rectangle()} {
			if got, ok := a.movieButtonAt(r.Min); !ok || got != button {
				t.Errorf("%s: button %d not at its rectangle %v", game, button, r)
			}
		}
	}
	first := screensCutsceneLayout(t, "rom1")
	if first.Panel.Rectangle() != image.Rect(120, 62, 520, 424) || first.OK.Rectangle() != image.Rect(180, 360, 292, 384) || first.Cancel.Rectangle() != image.Rect(348, 360, 460, 384) ||
		first.ListAt.Pt() != image.Pt(165, 148) || first.ListHeight != 149 || first.ListWidth != 283 {
		t.Errorf("first game layout %+v does not match the original's page", first)
	}
}
