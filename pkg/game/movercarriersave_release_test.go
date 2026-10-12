package game

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// TestReleaseMoverRecordsSurviveSAVEAndColdLOAD starts the Kadagan Post
// mission, orders the hero to walk, and SAVEs twice: on a tick the hero turns
// and on a tick it is between two cells. A placed AI actor stands throughout.
// Each SAV writes the standing actor's Mover from its record: its facing, the
// constructor's bytes and the occupy's step cost. Each cold LOAD restores the
// World byte for byte and the next 256 ticks agree with the live run. Loss
// control: a SAV whose standing actor holds the constructor's zero step cost
// restores a different World.
func TestReleaseMoverRecordsSurviveSAVEAndColdLOAD(t *testing.T) {
	const mission = 41
	f := releaseFront(t)
	f.SetTipsOff(true)
	a := f.App("mover record SAVE")
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(mission, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	hero := f.live.mission.state.Start.IDs[0]
	for n, phase := range []string{"turning", "stepping"} {
		h, _ := f.live.world.Entity(hero)
		dx := int32(6 - 12*n)
		f.live.pending = []sim.Command{sim.MoveTo(hero, sim.CellPoint{X: h.X + dx, Y: h.Y + 3})}
		for i := 0; i < 600; i++ {
			e, _ := f.live.world.Entity(hero)
			if phase == "turning" && e.Turning() || phase == "stepping" && e.Transit > 0 && e.Transit < e.TransitTotal {
				break
			}
			f.live.tick()
		}
		e, _ := f.live.world.Entity(hero)
		if phase == "turning" && !e.Turning() || phase == "stepping" && e.Transit == 0 {
			t.Fatalf("the hero never reached the %s state", phase)
		}
		moverSaveRoundTrip(t, f, phase, hero)
	}
}

func moverSaveRoundTrip(t *testing.T, f *FrontEnd, phase string, hero sim.EntityID) {
	t.Helper()
	w := f.live.world
	var standing sim.Entity
	h, _ := w.Entity(hero)
	far := int32(-1)
	for _, e := range w.Entities() {
		d := (e.X-h.X)*(e.X-h.X) + (e.Y-h.Y)*(e.Y-h.Y)
		if e.Alive() && e.Owner != sim.SelfSlot && e.MapUnitID != 0 && e.Transit == 0 && !e.Turning() && d > far {
			standing, far = e, d
		}
	}
	if far < 0 {
		t.Fatal("no placed AI actor stands")
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "Mover record "+phase)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	record := patrolRecordByMapUnit(t, doc, standing.MapUnitID)
	mover, err := savedActorRaw(record, "U154", 180)
	if err != nil {
		t.Fatal(err)
	}
	var want [180]byte
	if !w.ApplyMover(standing.ID, &want) {
		t.Fatal("the standing actor holds no Mover record")
	}
	cost := binary.LittleEndian.Uint16(mover[0x72:])
	if mover[0] != standing.Facing || mover[1] != standing.DesiredFacing || mover[0] < 0x40 || mover[0] > 0xc0 ||
		mover[5] != want[5] || mover[8] != 5 || mover[9] != 0xff || cost == 0 || cost != binary.LittleEndian.Uint16(want[0x72:]) {
		t.Fatalf("%s: standing actor %d Mover % x, step cost %#x", phase, standing.ID, mover[:11], cost)
	}
	cold := groupCarrierColdLoad(t, raw)
	liveBytes, _ := f.live.world.MarshalBinary()
	coldBytes, _ := cold.live.world.MarshalBinary()
	if !bytes.Equal(liveBytes, coldBytes) {
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "Mover record cold LOAD "+phase)
		t.Fatalf("%s: the restored World bytes differ from the live ones", phase)
	}
	coldHash := cold.live.world.Hash()
	binary.LittleEndian.PutUint16(mover[0x72:], 0)
	lossRaw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if loss := groupCarrierColdLoad(t, lossRaw); loss.live.world.Hash() == coldHash {
		t.Fatalf("%s: loss control: the constructor's step cost restored the same World", phase)
	}
	for i := 0; i < 256; i++ {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("%s: successor tick %d differs after the cold LOAD", phase, i+1)
		}
	}
}
