package ui

import (
	"errors"
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

func TestLoadDoubleClickUsesTheSameSaveAndResets(t *testing.T) {
	for _, scenario := range []string{"load", "different row", "late", "focus", "wheel", "reopen", "refused"} {
		t.Run(scenario, func(t *testing.T) {
			a := newTestApp(t, nil, nil)
			var loaded []string
			a.SetSaveSeams(nil, func() []SaveEntry {
				return []SaveEntry{{Name: "first.ags", Label: "First"}, {Name: "second.sav", Label: "Second"}}
			}, func(name string) (MapOpener, bool, error) {
				loaded = append(loaded, name)
				if scenario == "refused" {
					return nil, false, errors.New("bad save")
				}
				return func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
					return okLoader(t)(0)
				}, false, nil
			})
			a.flow.openLoad(ScreenMenu)
			now := time.Unix(500, 0)
			click := func(row int) {
				x, y := 140, 158+row*19
				a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
				a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now.Add(time.Millisecond))
			}
			click(0)
			if len(loaded) != 0 {
				t.Fatal("single click loaded a save")
			}
			now = now.Add(200 * time.Millisecond)
			row := 0
			switch scenario {
			case "different row":
				row = 1
			case "late":
				now = now.Add(time.Second)
			case "focus":
				a.step(appInput{Unfocused: true}, now)
			case "wheel":
				a.step(appInput{WheelY: 1}, now)
			case "reopen":
				a.flow.closeLoad()
				a.flow.openLoad(ScreenMenu)
			}
			click(row)
			if scenario == "load" || scenario == "refused" {
				if !reflect.DeepEqual(loaded, []string{"first.ags"}) {
					t.Fatal("wrong activation", loaded)
				}
				if scenario == "load" && a.Screen() != ScreenMap {
					t.Fatal("loader did not enter mission")
				}
				if scenario == "refused" {
					if a.Screen() != ScreenLoad || a.HeadlessMessage() == "" {
						t.Fatal("refusal lost LOAD or diagnostic")
					}
					now = now.Add(50 * time.Millisecond)
					click(0)
					if len(loaded) != 1 {
						t.Fatal("failed load retained the prior double click")
					}
				}
			} else if len(loaded) != 0 {
				t.Fatal("stale or unrelated click loaded", loaded)
			}
		})
	}
}

func TestRestoredSpellIsArmedOnlyForAvailableSelection(t *testing.T) {
	for _, test := range []struct {
		name      string
		spell     uint32
		selection bool
		want      bool
	}{
		{"selected", 17, true, true}, {"no spell", 0, true, false}, {"unavailable", 3, true, false},
		{"unknown", 31, true, false}, {"no selection", 17, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			a, v, _ := quickSpellApp(t)
			s := v.SaveApplication()
			s.PressedSpell = test.spell
			s.SpellBookOpen = false
			if !test.selection {
				s.Selection = nil
			}
			if err := v.RestoreSaveApplication(s); err != nil {
				t.Fatal(err)
			}
			if _, id, armed := v.QuickSpellState(); id != test.spell || armed != test.want {
				t.Fatal(id, armed)
			}
			if !test.want {
				return
			}
			v.Layout(1024, 768)
			if v.missionMode() != modeCast {
				t.Fatal("first layout lost the cast cursor")
			}
			x, y, err := a.HeadlessGroundPoint()
			if err != nil {
				t.Fatal(err)
			}
			if got := v.gestureCursorAt(x, y); got != "cast" {
				t.Fatal("next click has cursor", got)
			}
		})
	}
}

func TestPathfindingPreferenceOnlyFiltersDrawing(t *testing.T) {
	v := commandViewer(t)
	v.SetEntities([]MapEntity{pthEntity(7, 2, 2, image.Pt(3, 2), image.Pt(4, 3))})
	v.sel = selection{7}
	before := v.SaveApplication()
	if v.PathfindingShown() || len(v.displayedPathSegments()) != 0 {
		t.Fatal("paths visible by default")
	}
	v.SetPathfinding(true)
	if len(v.displayedPathSegments()) != 2 {
		t.Fatal("enabled route absent")
	}
	v.SetPathfinding(false)
	if len(v.pathScreenSegments()) != 2 || !reflect.DeepEqual(before, v.SaveApplication()) {
		t.Fatal("display setting changed route or saved application")
	}
}

func TestBurningSceneryPrecedesDeadArtWithoutColdReplay(t *testing.T) {
	dead := &terrain.StaticClass{Width: 32, Height: 32, Frame: staticsFrame(4, 8, 9)}
	live := &terrain.StaticClass{Width: 32, Height: 64, Frame: staticsFrame(20, 40, 7), Dead: dead}
	set := &terrain.StaticSet{}
	set.Classes[1] = live
	g := grid(32, 32)
	g.Overlay = make([]byte, 32*32)
	g.Overlay[16*32+16] = 1
	makeViewer := func() *Viewer {
		v, err := NewViewerWithStatics("fire", g, &terrain.Tileset{}, set, true, false, true, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	v := makeViewer()
	v.SetScorchedCellsAt(nil, 0)
	v.SetScorchedCellsAt([]uint16{0x1010}, 10)
	if len(v.BurningScenery()) != 1 || v.staticsFlat[0].Class != live {
		t.Fatal("tree did not enter visible fire phase")
	}
	v.SetScorchedCellsAt([]uint16{0x1010}, 33)
	if len(v.BurningScenery()) != 1 {
		t.Fatal("fire ended early")
	}
	v.SetScorchedCellsAt([]uint16{0x1010}, 34)
	if len(v.BurningScenery()) != 0 || v.staticsFlat[0].Class != dead {
		t.Fatal("fire did not become a dead tree")
	}
	cold := makeViewer()
	cold.SetScorchedCellsAt([]uint16{0x1010}, 11)
	if len(cold.BurningScenery()) != 0 || cold.staticsFlat[0].Class != dead {
		t.Fatal("cold load replayed an old fire")
	}
	if g.Tiles[16*32+16] != 0 {
		t.Fatal("presentation changed the source map")
	}
}

func TestCrystalSeamIsDrawnAndCapturedWithCentredContent(t *testing.T) {
	v := newStaticsViewer(t, staticsBundle(), true, true)
	art := &DialogFrame{Minimap: image.NewRGBA(image.Rect(0, 0, 160, 158))}
	v.SetDialogFrame(art)
	before, _ := v.minimapGeometry()
	art.MinimapSeam = image.NewRGBA(image.Rect(0, 0, 16, 158))
	art.MinimapSeam.SetRGBA(2, 60, color.RGBA{20, 240, 60, 255})
	pic, at, ok := v.minimapPresent()
	after, _ := v.minimapGeometry()
	if !ok || pic.Bounds().Dx() != 176 || at.X != before.Box.Min.X-16 || after.Content.Min.X != before.Content.Min.X-8 || after.Content.Dy() != before.Content.Dy() || pic.RGBAAt(2, 60) != art.MinimapSeam.RGBAAt(2, 60) {
		t.Fatal("seam absent or content not centred in the complete frame")
	}
	if got, want := after.Content.Min.X-at.X, (pic.Bounds().Dx()-after.Content.Dx())/2; got != want {
		t.Fatalf("content offset in crystal frame = %d, want centred %d", got, want)
	}
	if cell, ok := v.minimapCellAt(after.Content.Min.X, after.Content.Min.Y); !ok || cell != (image.Point{}) {
		t.Fatalf("left edge of shifted content names cell %v (ok=%v), want origin", cell, ok)
	}
	if !v.minimapCaptures(at.X+2, 60) {
		t.Fatal("crystal seam leaks clicks into the world")
	}
	if _, ok := v.minimapCellAt(at.X+2, 60); ok {
		t.Fatal("bevel moves the minimap camera")
	}
}

// The Load list's double click leaves the loaded map untouched.
func TestLoadDoubleClickReachesNothingOfTheLoadedMap(t *testing.T) {
	a := newTestApp(t, nil, nil)
	var ticks, orders int
	var loadedApp SaveApplicationState
	var viewer *Viewer
	a.SetSaveSeams(nil, func() []SaveEntry {
		return []SaveEntry{{Name: "first.sav", Label: "First"}}
	}, func(string) (MapOpener, bool, error) {
		return func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
			v, err := NewViewer("m", grid(60, 60), &terrain.Tileset{})
			if err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			v.SetEntities([]MapEntity{{ID: 7, Cell: image.Pt(1, 1)}})
			v.sel = selection{7}
			viewer, loadedApp = v, v.SaveApplication()
			return v, func() { ticks++ }, func(uint32, int, int) { orders++ }, nil, nil, nil, nil, nil, nil, nil, nil
		}, false, nil
	})
	a.flow.openLoad(ScreenMenu)
	now := time.Unix(500, 0)
	press := func(edge string, at time.Time) {
		in := appInput{CursorX: 140, CursorY: 158}
		in.PrimaryPressed, in.PrimaryReleased = edge == "press", edge == "release"
		a.step(in, at)
	}
	press("press", now)
	press("release", now.Add(time.Millisecond))
	press("press", now.Add(200*time.Millisecond))
	press("release", now.Add(201*time.Millisecond))
	if a.Screen() != ScreenMap || viewer == nil {
		t.Fatalf("double click did not load: screen %s", a.Screen())
	}
	if ticks != 0 || orders != 0 || !reflect.DeepEqual(loadedApp, viewer.SaveApplication()) {
		t.Fatalf("the double click reached the loaded map: %d ticks, %d orders, selection %v", ticks, orders, viewer.sel)
	}
}
