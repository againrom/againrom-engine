package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
)

func TestGeneratedMissionSAVFullTickFollowsSubTick(t *testing.T) {
	f := releaseFront(t)
	app := f.App("generated mission clock")
	if err := app.OpenMission(f.MissionOpenerWith(10, MissionParty(nil, data.BodyList{}, nil))); err != nil {
		t.Fatal(err)
	}
	f.LiveAdvance(113)
	// A first-game World holds the session clock from construction (SESS-084,
	// SESS-085); its full tick follows the sub-tick (SAV-1210, SESS-086).
	if clock, present := f.live.world.SessionClock(); !present || clock.SubTick != 113 || clock.FullTick != 7 {
		t.Fatalf("generated mission session clock=%+v present=%v, want 113/7 held", clock, present)
	}
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := uint32(f.live.world.Tick()); got != 113 || doc.Head.CounterA != got || doc.Head.CounterB != 7 {
		t.Fatalf("World tick=%d, SAV head A=%d B=%d; want 113/113/7", got, doc.Head.CounterA, doc.Head.CounterB)
	}
}
