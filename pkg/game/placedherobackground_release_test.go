package game

import (
	"fmt"
	"image"
	"os"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A placed person who is not a Hero NPC shows its own body and equipment in
// the information portrait and no warrior hero background; a dressed hero
// warrior keeps it (UNIT-PICT-035, HERO-FIGURE-144). The mission-10 peasants
// wear nothing, so their expected picture is the bare figure sheet, decoded
// independently of the production compositor. Where that sheet is transparent
// the pane must show exactly its own art, the installed pane bitmaps, after
// the ordinary map right-click cancel has cleared the selection. The pane is
// reached through the ordinary pointer hover and selection, in the started
// mission and again after an ordinary SAVE and a cold LOAD. Only the
// presentation fog is opened, as TestReleaseInspectionLivePanels does, and the
// world hash must not move.
func TestReleasePlacedPersonPortraitDrawsNoHeroBackground(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	open := func(g *FrontEnd, opener ui.MapOpener) (*ui.App, *mapWorld) {
		t.Helper()
		a := g.App("placed person portrait")
		t.Cleanup(a.StopAudio)
		a.Layout(1024, 768)
		if err := a.OpenMission(opener); err != nil {
			t.Fatal(err)
		}
		releasePauseMission(t, g, a)
		live := g.live
		for i := range live.fog.visible {
			live.fog.visible[i], live.fog.explored[i] = 1, 1
		}
		live.push()
		return a, live
	}
	back := map[bool]*image.RGBA{
		false: inspectionSprite(t, f, "graphics/interface/heroback/backm.256"),
		true:  inspectionSprite(t, f, "graphics/interface/heroback/backf.256"),
	}
	pane := func(a *ui.App, what string) *image.RGBA {
		t.Helper()
		pic, stats, err := a.HeadlessCharacterPane()
		if err != nil || stats {
			t.Fatalf("%s: figure pane unavailable: stats=%v err=%v", what, stats, err)
		}
		return pic
	}
	hover := func(a *ui.App, live *mapWorld, id sim.EntityID, what string) *image.RGBA {
		t.Helper()
		e, _ := live.entity(id)
		inspectionCentre(live, int(e.X), int(e.Y))
		releaseHoverInspection(t, a, live, ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(id)})
		return pane(a, what+" hovered")
	}
	leave := func(a *ui.App) {
		t.Helper()
		if err := a.HeadlessPointer("hover", -1, -1); err != nil {
			t.Fatal(err)
		}
	}
	// selected reaches the pane's selection arm: the pointer leaves the map
	// and the unit is picked by its own screen rectangle.
	selected := func(a *ui.App, live *mapWorld, id sim.EntityID, what string) *image.RGBA {
		t.Helper()
		leave(a)
		if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
			t.Fatal(what, err)
		}
		leave(a)
		if got, ok := live.view.SelectedUnit(); !ok || got != uint32(id) {
			t.Fatalf("%s: selection = %d/%v", what, got, ok)
		}
		return pane(a, what+" selected")
	}

	// Two dressed hero warriors: the entering party's own hero and, on the
	// placement route the peasants take, mission 40's Hero NPC Brian.
	control := func(a *ui.App, live *mapWorld, id sim.EntityID, face int, what string) {
		t.Helper()
		e, ok := live.entity(id)
		slots, _ := live.world.Equipped(id)
		worn := 0
		for _, code := range slots {
			if code != 0 {
				worn++
			}
		}
		fig := live.figures[id]
		if !ok || e.TypeID != sim.HumanTypeID || worn < 3 || fig.Dir != data.FigureDirManFighter || fig.Face != face {
			t.Fatalf("%s discriminator changed: %+v slots=%x figure=%+v", what, e, slots, fig)
		}
		bare, _ := composeUnitFigure(f.Archives.Containers, equipmentFromSlots(slots), figureID{Dir: fig.Dir, Face: fig.Face})
		if bare == nil {
			t.Fatal(what, "has no composed figure")
		}
		pic := hover(a, live, id, what)
		checkInspectionFigure(t, pic, bare, what)
		shown, candidates := 0, 0
		for y := 40; y < 240; y++ {
			for x := 32; x < 120; x++ {
				c := back[false].RGBAAt(x, y)
				if bare.RGBAAt(x, y).A != 0 || c.A == 0 {
					continue
				}
				candidates++
				if pic.RGBAAt(x+16, y+2) == c {
					shown++
				}
			}
		}
		if candidates < 100 || shown != candidates {
			t.Errorf("%s: %d of %d background pixels drawn", what, shown, candidates)
			return
		}
		t.Logf("%s: %d worn slots, all %d background pixels drawn", what, worn, candidates)
	}

	peasants := []struct {
		unit   uint16
		female bool
		sheet  string
	}{
		{0, false, "graphics/equipment/mfighter/17.256"},
		{0, true, "graphics/equipment/ffighter/9.256"},
	}
	// mission10 checks both peasants on both routes and the entering hero. A
	// peasant is found by its own map unit id, the identity SAVE carries.
	mission10 := func(a *ui.App, live *mapWorld, when string) {
		t.Helper()
		before := live.world.Hash()
		// The reference pane: a right-click on the map with no armed mode
		// clears the selection (AI-CURSOR-177), and the pane then draws its own
		// art under the zero-selection lines. The reference is the art alone,
		// drawn from the installed pane bitmaps.
		leave(a)
		if err := a.HeadlessPointer("right-press", 200, 200); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("right-release", 200, 200); err != nil {
			t.Fatal(err)
		}
		leave(a)
		if id, ok := live.view.SelectedUnit(); ok {
			t.Fatalf("%s: right-click cancel kept unit %d selected", when, id)
		}
		pane(a, when+" no subject")
		art := releaseMissionFigurePane(f, nil)
		for _, p := range peasants {
			var e sim.Entity
			found := 0
			for _, c := range live.world.Entities() {
				if c.MapUnitID == p.unit {
					e, found = c, found+1
				}
			}
			what := fmt.Sprintf("%s peasant %d", when, p.unit)
			eq, _ := live.world.Equipped(e.ID)
			fig := live.figures[e.ID]
			if found != 1 || e.Owner == sim.SelfSlot || e.TypeID != 1 || eq != ([sim.EquipSlots]uint16{}) || fig.Dir.Female() != p.female {
				t.Fatalf("%s discriminator changed: %d found, %+v eq=%x figure=%+v", what, found, e, eq, fig)
			}
			want := inspectionSprite(t, f, p.sheet)
			for _, route := range []string{"hovered", "selected"} {
				var pic *image.RGBA
				if route == "hovered" {
					pic = hover(a, live, e.ID, what)
				} else {
					pic = selected(a, live, e.ID, what)
				}
				checkInspectionFigure(t, pic, want, what+" "+route)
				// Behind the body the pane shows its art; a background pixel
				// whose colour differs from that art is where a drawn
				// background shows.
				extra, visible := 0, 0
				for y := 40; y < 240; y++ {
					for x := 32; x < 120; x++ {
						if want.RGBAAt(x, y).A != 0 {
							continue
						}
						behind := art.RGBAAt(x+16, y+2)
						if pic.RGBAAt(x+16, y+2) != behind {
							extra++
						}
						if c := back[p.female].RGBAAt(x, y); c.A != 0 && c != behind {
							visible++
						}
					}
				}
				if visible < 500 || extra != 0 {
					t.Errorf("%s %s: %d pixels behind the body differ from the pane art; %d background pixels would show",
						what, route, extra, visible)
					continue
				}
				t.Logf("%s %s: body drawn over the pane art alone; none of %d background pixels drawn", what, route, visible)
			}
		}
		control(a, live, live.mission.ids[0], 5, when+" entering hero")
		if live.world.Hash() != before {
			t.Fatalf("%s: portrait inspection changed the world", when)
		}
	}

	a, live := open(f, f.MissionOpener(10))
	for i, id := range []sim.EntityID{33, 34} {
		if e, ok := live.entity(id); ok {
			peasants[i].unit = e.MapUnitID
		}
		if peasants[i].unit == 0 {
			t.Fatalf("mission-10 peasant %d has no map unit id", id)
		}
	}
	mission10(a, live, "mission 10")

	path, _ := writeOrdinarySAV(t, f, "placed-person.sav")
	g := releaseFront(t)
	g.Options = OptionsStore{}
	g.SetDeterministicFrames(true)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	opener, town, err := g.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("mission-10 SAV LOAD town=%t: %v", town, err)
	}
	a, live = open(g, opener)
	mission10(a, live, "mission 10 after LOAD")

	a, live = open(f, f.MissionOpener(40))
	before := live.world.Hash()
	if live.mission.state.Start.Roster[0].CompanionNPC != 25 {
		t.Fatal("mission-40 placement 0 is no longer npc25")
	}
	control(a, live, 0, 1, "mission-40 Brian")
	if live.world.Hash() != before {
		t.Fatal("portrait inspection changed the mission-40 world")
	}
}
