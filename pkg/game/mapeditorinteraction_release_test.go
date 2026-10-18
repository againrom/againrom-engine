package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func TestReleaseEditorCancelPreservesInstalledMapBytes(t *testing.T) {
	x := installedInspector(t)
	keys, sources := installedMapBytes(t, x)
	popups := 0
	for n, key := range keys {
		t.Run(key, func(t *testing.T) {
			e := x.Editor()
			e.Layout(1280, 800)
			if err := e.Open(key); err != nil {
				t.Fatal(err)
			}
			closeMapPanel(t, e)
			s := e.Document().Edits.(*mapInspectionEdits)
			check := func(selected, mark int) {
				t.Helper()
				if !bytes.Equal(s.model.Bytes(), sources[n]) || s.model.Mark() != mark || e.Selected() != selected || e.Dirty() {
					t.Fatal("Cancel changed bytes, Undo, selection or dirty state")
				}
			}
			seen := map[string]bool{}
			for i, r := range e.Document().Records {
				if seen[r.Kind] {
					continue
				}
				seen[r.Kind] = true
				e.Select(i, false)
				mark, selected := s.model.Mark(), e.Selected()
				if rect, ok := e.PopupButtonRect(1); ok {
					trigClick(t, e, rect, true, "Cancel")
				} else {
					_ = e.Key("escape")
				}
				check(selected, mark)
				popups++
			}
			if unit := recordOf(e.Document(), "Unit", 0); unit >= 0 {
				e.Select(unit, false)
				_ = e.Key("escape")
			}
			for _, command := range []string{"unit", "structure", "sack", "field", "metadata", "pages", "package", "new", "open", "saveas", "recipe"} {
				selected, mark := e.Selected(), s.model.Mark()
				_ = e.Key(command)
				if _, ok := e.PromptRect(); ok {
					button := 1
					if command == "unit" || command == "structure" || command == "sack" {
						button = 2
					}
					r, ok := e.PromptButtonRect(button)
					if ok {
						trigClick(t, e, r, true, "Cancel")
					} else {
						_ = e.Key("escape")
					}
					popups++
				}
				check(selected, mark)
			}
			for _, bar := range []string{"groups", "players", "scripts"} {
				r, ok := e.BarRect(bar)
				trigClick(t, e, r, ok, bar)
				selected, mark := e.Selected(), s.model.Mark()
				_ = e.Key("player")
				r, ok = e.PromptButtonRect(2)
				trigClick(t, e, r, ok, "Cancel Add "+bar)
				check(selected, mark)
				popups++
				if bar == "scripts" {
					for choice := 1; choice < 4; choice++ {
						_ = e.Key("player")
						for j := 0; j < choice; j++ {
							_ = e.Key("down")
						}
						r, ok = e.PromptButtonRect(2)
						trigClick(t, e, r, ok, "Cancel script kind")
						check(selected, mark)
						popups++
					}
				}
			}
			openMessage := x.inspectionUTF8(x.word("editor.param.open") + " " + x.word("editor.message.parameter"))
			for i, record := range e.Document().Records {
				if record.Kind != "Action" {
					continue
				}
				messageMark := s.model.Mark()
				e.Select(i, false)
				opened := false
				for row, line := range e.PopupLines() {
					if strings.HasPrefix(line, openMessage) {
						r, ok := e.PopupRowRect(row)
						trigClick(t, e, r, ok, "Open message")
						opened = true
						break
					}
				}
				if opened {
					r, ok := e.PopupButtonRect(1)
					trigClick(t, e, r, ok, "Cancel message")
					check(i, messageMark)
					popups++
					break
				}
			}
			_, _, out := areaSaveAs(t, e, "cancel.alm")
			if !bytes.Equal(out, sources[n]) {
				t.Fatal("untouched Save As differs")
			}
			t.Logf("source and Save As SHA256=%x bytes=%d", sha256.Sum256(out), len(out))
		})
	}
	t.Logf("root=%s maps=%d cancelled popups=%d", os.Getenv("AGAINROM_ASSETS"), len(keys), popups)
}

func TestReleaseEditorCharacterCardsMatchGameSample(t *testing.T) {
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

func TestReleaseEditorInheritedMeaningsWithStatOverride(t *testing.T) {
	x := installedInspector(t)
	doc, err := x.Open("scenario/141.alm")
	if err != nil {
		t.Fatal(err)
	}
	s := doc.Edits.(*mapInspectionEdits)
	actual, bare := s.unitViewSheets(false), s.unitViewSheets(true)
	if len(actual) == 0 || len(bare) == 0 || actual[0].Values.HP != 112 || bare[0].Values.HP != 163 {
		t.Fatal("mission unit 0 must have derived health 112 and type health 163")
	}
	format := "Inherited; in game: %d"
	if installLanguage(x) == "ru" {
		format = "Наследуется; в игре: %d"
	}
	count := 0
	for _, f := range s.Fields("Unit", 0) {
		value, ok := overrideTemplate(actual[0].Values, f.Key)
		if !ok || f.Value != overrideAbsent(f.Key) {
			continue
		}
		count++
		if want := fmt.Sprintf(format, value); f.Meaning != want {
			t.Errorf("%s=%d: inherited meaning %q, want %q", f.Key, f.Value, f.Meaning, want)
		}
	}
	if count != len(overrideKeys)-4 {
		t.Fatalf("checked %d inherited rows, want %d", count, len(overrideKeys)-4)
	}
	t.Logf("root=%s scenario/141.alm unit=0 inherited rows=%d; health type=163 in-game=112", os.Getenv("AGAINROM_ASSETS"), count)
}

func TestReleaseEditorInteractionFrames(t *testing.T) {
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
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, pic); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		t.Log(path)
	}
	r, ok := e.BarRect("units")
	trigClick(t, e, r, ok, "Units")
	e.ListRowRect(0)
	_ = e.Key("end")
	write("list-end")
	_ = e.Key("escape")
	_ = e.Key("unit")
	e.ShowCell(image.Pt(20, 20))
	p := image.Pt(1000, 600)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	write("armed-placement")
	if r, ok := e.PromptButtonRect(2); ok {
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
