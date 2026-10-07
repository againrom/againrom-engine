package game

import (
	"fmt"
	"image"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Five campaign placements name a person by definition id and give him a
// secondary key: mission 71's entity 37, mission 111's entities 1, 36 and 46 and
// mission 120's entity 17. The spawner writes the key into his face byte and bit
// 2 of the placement's flags into its bit 7 (ALM-FLAGPATH-109), and the client
// draws a person below type id 0x1a from that byte (UNIT-PICT-035, PAL-FACE-005),
// so each is drawn by a figure his row's own face and gender columns do not
// state. The controls are a type-key person whose key equals his row's face and
// two definition-id persons with no secondary key, whose byte the original's
// saves record as the row's face with its gender in bit 7.
//
// Each person is selected through the pointer and his pane is compared with the
// figure composed from his raw placement words and worn set, fresh and after
// SAVE and a cold LOAD. SAVE is the ordinary producer, written before the first
// frame, and the face byte it stores for each person is read back from the file.
// Only the presentation fog is opened and the world must not move.
func TestReleasePlacedPersonsDrawTheFaceByteTheirSpawnerWrites(t *testing.T) {
	for _, mission := range []struct {
		number int
		people []placedFacePerson
	}{
		{71, []placedFacePerson{
			{entity: 0, unit: 41, control: true, dir: data.FigureDirManMage, face: 1},
			{entity: 37, unit: 78, dir: data.FigureDirManFighter, face: 1},
		}},
		{111, []placedFacePerson{
			{entity: 1, unit: 21, dir: data.FigureDirManMage, face: 1},
			{entity: 30, unit: 79, control: true, dir: data.FigureDirWomanFighter, face: 10},
			{entity: 36, unit: 85, dir: data.FigureDirManMage, face: 1},
			{entity: 46, unit: 95, dir: data.FigureDirManFighter, face: 1},
		}},
		{120, []placedFacePerson{
			{entity: 13, unit: 88, control: true, dir: data.FigureDirWomanFighter, face: 10},
			{entity: 17, unit: 92, dir: data.FigureDirManFighter, face: 2},
		}},
	} {
		f := releaseFront(t)
		f.Options = OptionsStore{}
		f.SetDeterministicFrames(true)
		a := f.App("placed face byte")
		t.Cleanup(a.StopAudio)
		a.Layout(1024, 768)
		if err := a.OpenMission(f.MissionOpener(mission.number)); err != nil {
			t.Fatal(err)
		}
		path, raw := writeOrdinarySAV(t, f, "placed-face.sav")
		fresh := f.live

		people := placedFacePeople(t, f, fresh, mission.people)
		placedFaceRoutes(t, fresh, fmt.Sprintf("mission %d fresh", mission.number), people)
		placedFacePanes(t, f, a, fresh, fmt.Sprintf("mission %d fresh", mission.number), people)
		placedFaceSaved(t, raw, mission.number, people)

		g, b := lancerLoad(t, filepath.Dir(path), filepath.Base(path))
		placedFaceRoutes(t, g.live, fmt.Sprintf("mission %d loaded", mission.number), people)
		placedFacePanes(t, g, b, g.live, fmt.Sprintf("mission %d loaded", mission.number), people)
	}
}

// placedFacePerson is one placement of the witness. dir and face are the figure
// the claims give him, written out; the rest is read from the install.
type placedFacePerson struct {
	entity  sim.EntityID
	unit    uint16
	control bool
	dir     data.FigureDir
	face    int

	name     string
	row      data.HumanDef
	faceByte uint8
}

func (p placedFacePerson) String() string {
	return fmt.Sprintf("%s (unit %d, class %d)", p.name, p.unit, p.row.TypeID)
}

// placedFacePeople reads each named placement's raw words and row from the
// mission's map and the installed table, states the figure they give by the
// spawner's own rule with no mapload code, and fails when that disagrees with
// the figure the caller wrote down or when the row's own figure would not have
// differed for a person the fix moves.
func placedFacePeople(t *testing.T, f *FrontEnd, live *mapWorld, want []placedFacePerson) []placedFacePerson {
	t.Helper()
	ms := live.mission.state
	out := make([]placedFacePerson, 0, len(want))
	for _, p := range want {
		if int(p.entity) >= len(ms.Map.Units) {
			t.Fatalf("mission %d has no placement %d", ms.Number, p.entity)
		}
		u := ms.Map.Units[p.entity]
		if u.UnitID != p.unit {
			t.Fatalf("mission %d placement %d is unit %d, want %d", ms.Number, p.entity, u.UnitID, p.unit)
		}
		r := mapload.Resolve(u, f.Table)
		if !r.Found() || (r.Arm != mapload.ArmServerID && r.Arm != mapload.ArmHumansByType) {
			t.Fatalf("mission %d unit %d resolved by arm %s to row %d, want a definition-id or type-key person",
				ms.Number, u.UnitID, r.Arm, r.Index)
		}
		p.name = f.Table.Humans.EntryName(r.Index)
		row, err := data.NewHumanDef(p.name, f.Table.Humans.EntryParams(r.Index))
		if err != nil || row.TypeID >= 0x1a {
			t.Fatalf("mission %d unit %d row %q: class %d, err %v", ms.Number, u.UnitID, p.name, row.TypeID, err)
		}
		p.row = row

		// The spawner's stores: the type-key arm always, the definition-id arm
		// when the secondary key is not zero. Anything else keeps the
		// constructor's byte, which is the row's face with its gender in bit 7.
		stores := r.Arm == mapload.ArmHumansByType || u.ClassSubID != 0
		rowByte := byte(row.Face) | byte(row.Gender&1)<<7
		p.faceByte = rowByte
		if stores {
			p.faceByte = byte(u.ClassSubID)&0x7f | byte(u.Flags>>2&1)<<7
		}
		mage := row.TypeID == 0x17 || row.TypeID == 0x18
		dir := data.FigureDirFor(mage, p.faceByte&0x80 != 0)
		face := int(p.faceByte & 0x7f)
		if dir != p.dir || face != p.face {
			t.Fatalf("mission %d unit %d (%s): placement words give %s/%d, the witness expects %s/%d",
				ms.Number, u.UnitID, p.name, dir, face, p.dir, p.face)
		}
		rowDir, rowFace := data.FigureFor(row.TypeID, row.Face, row.Gender)
		same := rowDir == p.dir && rowFace == p.face && rowByte == p.faceByte
		if same != p.control {
			t.Fatalf("mission %d unit %d (%s): row figure %s/%d byte %#x against expected %s/%d byte %#x; control %t",
				ms.Number, u.UnitID, p.name, rowDir, rowFace, rowByte, p.dir, p.face, p.faceByte, p.control)
		}
		out = append(out, p)
	}
	return out
}

// placedFaceRoutes compares the figure each route that stands for a placed person
// hands the world: his roster template, the figure a roster person's portrait is
// composed from and the dialogue candidate.
func placedFaceRoutes(t *testing.T, live *mapWorld, when string, people []placedFacePerson) {
	t.Helper()
	for _, p := range people {
		what := fmt.Sprintf("%s: %s, entity %d", when, p, p.entity)
		template, ok := live.mission.state.Start.Roster[p.entity]
		if !ok || template.FigureDir != string(p.dir) || template.FigureFace != p.face {
			t.Errorf("%s: roster template is %q/%d present %t, want %s/%d", what, template.FigureDir, template.FigureFace, ok, p.dir, p.face)
		}
		if fig := rosterFigureID(template, live.world, p.entity); fig.Dir != p.dir || fig.Face != p.face {
			t.Errorf("%s: roster figure is %s/%d, want %s/%d", what, fig.Dir, fig.Face, p.dir, p.face)
		}
		found := false
		for _, c := range live.speakerActors {
			if c.id != p.entity {
				continue
			}
			found = true
			if c.fig.Dir != p.dir || c.fig.Face != p.face || int(c.face) != p.face {
				t.Errorf("%s: dialogue candidate is %s/%d with face term %d, want %s/%d", what, c.fig.Dir, c.fig.Face, c.face, p.dir, p.face)
			}
		}
		if !found {
			t.Errorf("%s: no dialogue candidate", what)
		}
	}
}

// placedFacePanes selects each person on the map and compares the character pane
// with the figure his placement words give, and with the figure his row alone
// gives as the loss control. The world must not move.
func placedFacePanes(t *testing.T, f *FrontEnd, a *ui.App, live *mapWorld, when string, people []placedFacePerson) {
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
	before := live.world.Hash()
	for _, p := range people {
		what := fmt.Sprintf("%s: %s, entity %d", when, p, p.entity)
		e, ok := live.entity(p.entity)
		if !ok || e.MapUnitID != p.unit || e.TypeID != p.row.TypeID {
			t.Fatalf("%s: entity present %t, map unit %d, type id %d, want unit %d and the row's type id %d",
				what, ok, e.MapUnitID, e.TypeID, p.unit, p.row.TypeID)
		}
		slots, _ := live.world.Equipped(p.entity)
		eq := equipmentFromSlots(slots)
		want, _ := composeInventorySubject(f.Archives.Containers, uint32(p.entity), eq, p.dir, p.face)
		rowDir, rowFace := data.FigureFor(p.row.TypeID, p.row.Face, p.row.Gender)
		row, _ := composeInventorySubject(f.Archives.Containers, uint32(p.entity), eq, rowDir, rowFace)
		if want.Figure == nil || row.Figure == nil {
			t.Fatalf("%s: figure %s/%d or %s/%d is unreadable", what, p.dir, p.face, rowDir, rowFace)
		}

		inspectionCentre(live, int(e.X), int(e.Y))
		if err := a.HeadlessPointer("hover", -1, -1); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessSelectEntity(uint32(p.entity)); err != nil {
			t.Fatal(what, err)
		}
		if err := a.HeadlessPointer("hover", -1, -1); err != nil {
			t.Fatal(err)
		}
		if got, ok := live.view.SelectedUnit(); !ok || got != uint32(p.entity) {
			t.Fatalf("%s: selection is %d/%t", what, got, ok)
		}
		pic, statistics, err := a.HeadlessCharacterPane()
		if err != nil || statistics {
			t.Fatalf("%s: figure pane unavailable: statistics=%t err=%v", what, statistics, err)
		}
		checkInspectionFigure(t, pic, want.Figure, what+" selected")
		if shown := paneShowsFigure(pic, row.Figure); shown == !p.control {
			t.Fatalf("%s: the pane shows his row's own figure %s/%d: %t, want %t", what, rowDir, rowFace, shown, p.control)
		}
		t.Logf("%s: pane shows %s/%d; his row alone would give %s/%d", what, p.dir, p.face, rowDir, rowFace)
	}
	if live.world.Hash() != before {
		t.Fatalf("%s: selecting and inspecting the people changed the world", when)
	}
}

// paneShowsFigure reports whether every opaque pixel of figure in the pane's
// picture window is the pane's own pixel, on checkInspectionFigure's geometry.
func paneShowsFigure(pane, figure *image.RGBA) bool {
	for y := 40; y < 240; y++ {
		for x := 32; x < 120; x++ {
			if c := figure.RGBAAt(x, y); c.A != 0 && pane.RGBAAt(x+16, y+2) != c {
				return false
			}
		}
	}
	return true
}

// placedFaceSaved reads the face byte SAVE stored for each person back out of the
// file and compares it with the byte the spawner leaves him. The row's own byte
// is the loss control: it differs for every person the fix moves.
func placedFaceSaved(t *testing.T, raw []byte, mission int, people []placedFacePerson) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatalf("mission %d: SAVE does not open: %v", mission, err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatalf("mission %d: SAVE has no actor graph: %v", mission, err)
	}
	for _, p := range people {
		var saved []sav.ActorRecord
		for _, r := range graph.Actors {
			if r.MapUnitID == p.unit {
				saved = append(saved, r)
			}
		}
		if len(saved) != 1 {
			t.Fatalf("mission %d: %s has %d saved actors on unit %d", mission, p, len(saved), p.unit)
		}
		got := saved[0]
		rowByte := byte(p.row.Face) | byte(p.row.Gender&1)<<7
		if got.Face != p.faceByte || (got.Face == rowByte) != p.control {
			t.Errorf("mission %d: %s saved face byte %#x (type id %d), want %#x; his row's byte is %#x",
				mission, p, got.Face, got.TypeID, p.faceByte, rowByte)
		}
		t.Logf("mission %d: %s saved face byte %#x; his row's byte is %#x", mission, p, got.Face, rowByte)
	}
}
