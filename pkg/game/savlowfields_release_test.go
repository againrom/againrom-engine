package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

func TestGeneratedMissionSAVFullTickFollowsSubTick(t *testing.T) {
	f := releaseFront(t)
	app := f.App("generated mission clock")
	if err := app.OpenMission(f.MissionOpenerWith(10, MissionParty(nil, nil, nil))); err != nil {
		t.Fatal(err)
	}
	f.LiveAdvance(113)
	if _, present := f.live.world.SessionClock(); present {
		t.Fatal("generated mission unexpectedly has an imported session clock")
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
