package game

import (
	"fmt"
	"image"
	"image/draw"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The campaign places four people whose row's own class lies in the horse band
// (HERO-DOLL-078): mission 71's entities 1, 2 and 62 and mission 131's entity 9,
// none of them a Hero-mode placement. Entities 1 and 2 of mission 71 are placed
// already fallen. A scenario-placed human is drawn with his row's own class
// (UNIT-APPEAR-030), so the information portrait of each shows him on the horse
// layer, in a fresh mission and after SAVE and a cold LOAD. The roster template
// that stands for the placed person states the opposite: the equipment law gives
// it a body name and a body class when the mission opens and a SAV load replaces
// its class with the class the entity was placed with, and neither may decide
// the horse.
//
// Each Lancer is selected through the pointer's own press and release on the
// map, and the pane is composed by the viewer's own present call. The expected
// picture is the installed horse bitmap under the rider composed from the row's
// figure and the entity's worn set. SAVE is the ordinary producer, written
// before the mission's first frame; LOAD is the load dialog of a cold front end.
// Only the presentation fog is opened and the world must not move.
func TestReleasePlacedLancerPortraitDrawsHisHorseFreshAndAfterLoad(t *testing.T) {
	for _, mission := range []struct {
		number  int
		lancers []sim.EntityID
	}{
		{71, []sim.EntityID{1, 2, 62}},
		{131, []sim.EntityID{9}},
	} {
		f := releaseFront(t)
		f.Options = OptionsStore{}
		f.SetDeterministicFrames(true)
		a := f.App("placed lancer portrait")
		t.Cleanup(a.StopAudio)
		a.Layout(1024, 768)
		if err := a.OpenMission(f.MissionOpener(mission.number)); err != nil {
			t.Fatal(err)
		}
		path, _ := writeOrdinarySAV(t, f, "placed-lancer.sav")
		fresh := f.live

		riders := placedRiders(t, f, fresh, mission.lancers)
		lancerPortraits(t, f, a, fresh, fmt.Sprintf("mission %d fresh", mission.number), mission.lancers, riders)

		g, b := lancerLoad(t, filepath.Dir(path), filepath.Base(path))
		lancerPortraits(t, g, b, g.live, fmt.Sprintf("mission %d loaded", mission.number), mission.lancers, riders)
	}
}

// placedRiders scans the mission's map for the person placements whose row's own
// class lies in the horse band and that are no Hero-mode placement, checks that
// they are exactly the entities the caller names, and returns each one's row and
// map unit id.
func placedRiders(t *testing.T, f *FrontEnd, live *mapWorld, want []sim.EntityID) map[sim.EntityID]placedRider {
	t.Helper()
	ms := live.mission.state
	found := map[sim.EntityID]placedRider{}
	for i, u := range ms.Map.Units {
		r := mapload.Resolve(u, f.Table)
		if r.Arm == mapload.ArmUnits || !r.Found() {
			continue
		}
		name := f.Table.Humans.EntryName(r.Index)
		row, err := data.NewHumanDef(name, f.Table.Humans.EntryParams(r.Index))
		if err != nil || !data.FigureHasHorse(row.TypeID) {
			continue
		}
		if r.Arm == mapload.ArmNPC && f.Table.NPC.Hero(int32(u.ClassSubID)) {
			continue
		}
		found[sim.EntityID(i)] = placedRider{name: name, row: row, unit: u.UnitID}
	}
	if len(found) != len(want) {
		t.Fatalf("mission %d places %d riders %v, want %v", ms.Number, len(found), found, want)
	}
	for _, id := range want {
		if _, ok := found[id]; !ok {
			t.Fatalf("mission %d: entity %d is no placed rider; the riders are %v", ms.Number, id, found)
		}
	}
	return found
}

type placedRider struct {
	name string
	row  data.HumanDef
	unit uint16
}

func (r placedRider) String() string {
	return fmt.Sprintf("%s (unit %d, class %d)", r.name, r.unit, r.row.TypeID)
}

// lancerPortraits selects each rider on the map and checks the pane against the
// rider's row: the horse layer under the figure, the world untouched.
func lancerPortraits(t *testing.T, f *FrontEnd, a *ui.App, live *mapWorld, when string,
	ids []sim.EntityID, riders map[sim.EntityID]placedRider) {
	t.Helper()
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 4 && a.HeadlessNoticeOpen(); n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if a.HeadlessNoticeOpen() {
		t.Fatalf("%s: a message stayed open over the map", when)
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	horseBMP, err := f.Archives.Containers.ReadFile("graphics/infowindow/horse.bmp")
	if err != nil {
		t.Fatal(err)
	}
	horse := inspectionPortraitBMP(t, horseBMP)
	art := releaseMissionFigurePane(f, nil)
	before := live.world.Hash()
	for _, id := range ids {
		rider := riders[id]
		what := fmt.Sprintf("%s: %s, entity %d", when, rider, id)
		e, ok := live.entity(id)
		if !ok || e.MapUnitID != rider.unit || e.TypeID != rider.row.TypeID {
			t.Fatalf("%s: entity present %t, map unit %d, type id %d", what, ok, e.MapUnitID, e.TypeID)
		}
		if _, ok := live.mission.state.Start.Roster[id]; !ok {
			t.Fatalf("%s: the roster holds no template", what)
		}
		dir, face := data.FigureFor(rider.row.TypeID, rider.row.Face, rider.row.Gender)
		slots, _ := live.world.Equipped(id)
		composed, _ := composeInventorySubject(f.Archives.Containers, uint32(id), equipmentFromSlots(slots), dir, face)
		if composed.Figure == nil {
			t.Fatalf("%s: figure %s %d is unreadable", what, dir, face)
		}
		mounted := image.NewRGBA(composed.Figure.Bounds())
		draw.Draw(mounted, mounted.Bounds(), horse, horse.Bounds().Min, draw.Src)
		draw.Draw(mounted, mounted.Bounds(), composed.Figure, composed.Figure.Bounds().Min, draw.Over)

		inspectionCentre(live, int(e.X), int(e.Y))
		if err := a.HeadlessPointer("hover", -1, -1); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
			t.Fatal(what, err)
		}
		if err := a.HeadlessPointer("hover", -1, -1); err != nil {
			t.Fatal(err)
		}
		if got, ok := live.view.SelectedUnit(); !ok || got != uint32(id) {
			t.Fatalf("%s: selection is %d/%t", what, got, ok)
		}
		pic, statistics, err := a.HeadlessCharacterPane()
		if err != nil || statistics {
			t.Fatalf("%s: figure pane unavailable: statistics=%t err=%v", what, statistics, err)
		}
		checkInspectionFigure(t, pic, mounted, what+" selected")

		// The horse layer decides pixels the pane's own art does not already
		// show, so a portrait without it cannot pass the comparison above.
		horsePixels, differing := 0, 0
		for y := 40; y < 240; y++ {
			for x := 32; x < 120; x++ {
				h := horse.RGBAAt(x, y)
				if h.A == 0 || composed.Figure.RGBAAt(x, y).A != 0 {
					continue
				}
				horsePixels++
				if art.RGBAAt(x+16, y+2) != h {
					differing++
				}
			}
		}
		if differing < 1000 {
			t.Fatalf("%s: only %d of %d horse pixels in the pane window differ from the pane art", what, differing, horsePixels)
		}
		t.Logf("%s: alive %t, template class %d body %q, type id %d; %d of %d horse pixels differ from the pane art",
			what, e.Alive(), live.mission.state.Start.Roster[id].Class, live.mission.state.Start.Roster[id].Body, e.TypeID, differing, horsePixels)
	}
	if live.world.Hash() != before {
		t.Fatalf("%s: selecting and inspecting the riders changed the world", when)
	}
}
