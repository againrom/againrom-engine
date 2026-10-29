package game

import (
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// searchRadiusRecord is one actor record's mover byte +0x08 in a SAV.
type searchRadiusRecord struct {
	identity uint32
	unit     uint32
	stage    uint32
	radius   byte
}

func searchRadiusRecords(t *testing.T, doc sav.DocumentData) []searchRadiusRecord {
	t.Helper()
	var out []searchRadiusRecord
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
			continue
		}
		mover, err := savedActorRaw(r, "U154", 180)
		if err != nil {
			t.Fatal(err)
		}
		identity, _ := savedStructureValue(r, "Identity")
		unit, _ := savedStructureValue(r, "T08")
		stage, _ := savedStructureValue(r, "Stage")
		out = append(out, searchRadiusRecord{identity, unit, stage, mover[8]})
	}
	return out
}

func searchRadiusSave(t *testing.T, f *FrontEnd, label string) ([]byte, sav.DocumentData) {
	t.Helper()
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, doc
}

// TestReleaseSaveWritesEveryActorsSearchRadius starts the Kadagan Post
// mission, kills one AI player's actors and runs until bodies are gone, then
// SAVEs through the save point. Every actor record, the party's, the living
// AI's and the dead's, holds the constructor's 5 at mover byte +0x08 (AI-692).
// A body that is gone is written at its terminal stage. A SAV whose party and
// living AI records hold 0 there LOADs with 0 in the World and SAVEs 0 again
// (AI-693); a dead record holding 0 is written with 5 (DIV-3942), as is every
// other record.
func TestReleaseSaveWritesEveryActorsSearchRadius(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	const mission = 41
	f := releaseFront(t)
	f.SetTipsOff(true)
	a := f.App("search radius SAVE")
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(mission, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	owners := map[uint32]int{}
	for _, e := range w.Entities() {
		if e.Owner != sim.SelfSlot && e.Owner != 0 && e.Alive() {
			owners[e.Owner]++
		}
	}
	victim, most := uint32(0), 0
	for owner, n := range owners {
		if n > most || n == most && owner < victim {
			victim, most = owner, n
		}
	}
	if most == 0 || w.CheatKillPlayer(victim) == 0 {
		t.Fatal("no AI player's actor to kill")
	}
	gone := func() int {
		n := 0
		for _, d := range f.live.world.OriginalDeadActors() {
			if d.Current.Stage == 5 {
				n++
			}
		}
		return n + len(f.live.world.CurrentTerminalActors())
	}
	for i := 0; i < 20000 && gone() == 0; i++ {
		f.live.tick()
	}
	if gone() == 0 {
		t.Fatal("no killed actor's body is gone")
	}
	_, doc := searchRadiusSave(t, f, "Search radius 1")
	records := searchRadiusRecords(t, doc)
	party, dead, alive := 0, 0, 0
	for _, r := range records {
		switch {
		case r.stage != 0:
			dead++
		case r.unit == 0:
			party++
		default:
			alive++
		}
		if r.radius != 5 {
			t.Errorf("record %#x (unit %d, stage %d) holds +0x08 = %d, want 5", r.identity, r.unit, r.stage, r.radius)
		}
	}
	t.Logf("first SAVE: %d records, %d without a map unit, %d placed, %d dead", len(records), party, alive, dead)
	if party == 0 || dead == 0 || alive == 0 {
		t.Fatalf("population lacks a party, dead or placed record: %d/%d/%d", party, alive, dead)
	}

	// Zero one record without a map unit, one placed living record and one
	// dead record, as a SAV the original wrote can hold them.
	zeroed := map[uint32]bool{}
	deadZeroed := map[uint32]bool{}
	var kinds [3]bool
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
			continue
		}
		identity, _ := savedStructureValue(r, "Identity")
		unit, _ := savedStructureValue(r, "T08")
		stage, _ := savedStructureValue(r, "Stage")
		kind := 1
		if stage != 0 {
			kind = 2
		} else if unit == 0 {
			kind = 0
		}
		if kinds[kind] || identity == 0 {
			continue
		}
		kinds[kind] = true
		mover, _ := savedActorRaw(r, "U154", 180)
		mover[8] = 0
		if kind == 2 {
			deadZeroed[identity] = true
		} else {
			zeroed[identity] = true
		}
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := groupCarrierColdLoad(t, raw)
	zeroInWorld := 0
	for _, e := range cold.live.world.Entities() {
		var m [180]byte
		if cold.live.world.ApplyMover(e.ID, &m) && m[8] == 0 {
			zeroInWorld++
		}
	}
	if zeroInWorld != 2 {
		t.Errorf("the LOADed World holds %d actors with search radius 0, want the 2 living zeroed records", zeroInWorld)
	}
	_, again := searchRadiusSave(t, cold, "Search radius 2")
	for _, r := range searchRadiusRecords(t, again) {
		want := byte(5)
		if zeroed[r.identity] {
			want = 0
			delete(zeroed, r.identity)
		}
		delete(deadZeroed, r.identity)
		if r.radius != want {
			t.Errorf("second SAVE: record %#x (unit %d, stage %d) holds +0x08 = %d, want %d", r.identity, r.unit, r.stage, r.radius, want)
		}
	}
	if len(zeroed) != 0 || len(deadZeroed) != 0 {
		t.Errorf("second SAVE lost %d living and %d dead zeroed records by identity", len(zeroed), len(deadZeroed))
	}
}
