package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/internal/witness"
	"againrom/pkg/formats/alm"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func TestReleaseEditorCharacterCardsMatchGameSample(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	total := 0
	for _, key := range []string{"scenario/141.alm", "scenario/151.alm"} {
		t.Run(key, func(t *testing.T) {
			e := x.Editor()
			e.Layout(1280, 800)
			if err := e.Open(key); err != nil {
				t.Fatal(err)
			}
			closeMapPanel(t, e)
			m, err := alm.Open(e.Document().SourceBytes)
			if err != nil {
				t.Fatal(err)
			}
			v, err := ui.NewViewer("", terrain.Grid{Width: m.Width, Height: m.Height, Tiles: terrain.RenderTileWords(m.Tiles), Altitudes: m.Altitudes}, x.tiles)
			if err != nil {
				t.Fatal(err)
			}
			if x.panelWords != nil {
				v.SetWords(*x.panelWords)
			}
			mw, err := openMapWorld(m, x.table, x.units, v)
			if err != nil {
				t.Fatal(err)
			}
			entities := mw.entityDraws()
			for i, want := range entities {
				e.Select(recordOf(e.Document(), "Unit", i), false)
				got, ok := e.UnitCharacterSubject()
				if !ok || got.Name != want.Name || got.UnitNameIndex != want.UnitNameIndex || got.Char != want.Char || got.Combat != want.Combat || got.HP != want.HP || got.MaxHP != want.MaxHP || got.Mana != want.Mana || got.MaxMana != want.MaxMana || got.Speed != want.Speed || got.KnownSpells != want.KnownSpells {
					t.Fatalf("unit %d card differs from game", i)
				}
				total++
			}
			t.Logf("map=%s placed cards=%d", key, len(entities))
		})
	}
	t.Logf("root=%s sample maps=2 cards=%d; Normal difficulty, tick zero, full card detail, no campaign party substitution", os.Getenv("AGAINROM_ASSETS"), total)
}

func TestReleaseEditorInheritedHintsWithStatOverride(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	doc, err := x.Open("scenario/141.alm")
	if err != nil {
		t.Fatal(err)
	}
	s := doc.Edits.(*mapInspectionEdits)
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	actual := s.unitViewSheets()
	bare, err := templateSheets(m, x.table)
	if err != nil || len(actual) == 0 || len(bare) == 0 || actual[0].Values.HP != 112 || bare[0].Values.HP != 163 {
		t.Fatal("mission unit 0 must have derived health 112 and type health 163", err)
	}
	count := 0
	for _, f := range s.Fields("Unit", 0) {
		value, ok := overrideTemplate(actual[0].Values, f.Key)
		if !ok || f.Value != overrideAbsent(f.Key) {
			continue
		}
		count++
		if !f.Inherit || f.Absent != f.Value || f.Inherited != strconv.FormatInt(value, 10) || f.Meaning != "" || f.Group == "" {
			t.Errorf("%s=%d: inherited hint %q meaning %q group %q, want %d and no meaning", f.Key, f.Value, f.Inherited, f.Meaning, f.Group, value)
		}
	}
	if count != len(overrideKeys)-4 {
		t.Fatalf("checked %d inherited rows, want %d", count, len(overrideKeys)-4)
	}
	t.Logf("root=%s scenario/141.alm unit=0 inherited rows=%d; health type=163 in-game=112", os.Getenv("AGAINROM_ASSETS"), count)
}

func TestReleaseEditorInteractionFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_EDITOR_INTERACTION_FRAMES")
	if dir == "" {
		dir = t.TempDir()
	}
	lang := installLanguage(x)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/141.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	write := func(name string) {
		t.Helper()
		pic, _, err := e.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, name+"-"+lang+".png")
		witness.PNG(t, path, pic)
		t.Log(path)
	}
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: "units"})
	trigClick(t, e, r, ok, "Units")
	// The Units branch of the tree scrolled to its last unit.
	if n := treeMemberCount(e, "units"); n > 0 {
		_, _ = treeMemberRect(e, "units", n-1)
	}
	write("list-end")
	_ = e.Key("escape")
	_ = e.Key("unit")
	e.ShowCell(image.Pt(20, 20))
	p := image.Pt(1000, 600)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	write("armed-placement")
	if r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 2}); ok {
		trigClick(t, e, r, true, "Cancel")
	}
	e.Select(recordOf(e.Document(), "Unit", 0), false)
	write("unit-fields-card")
	e.ScrollDetails(6)
	write("unit-overrides-card")
	if e.Dirty() {
		t.Fatal(fmt.Sprintf("frames changed %s", lang))
	}
}
