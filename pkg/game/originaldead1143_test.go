package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestOriginalDead1143MintsDistinctUnboundIdentities(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 7, X: 1, Y: 1, HP: 20, MaxHP: 20}})
	if err != nil {
		t.Fatal(err)
	}
	before := w.Entities()
	ms := &Mission{Map: &alm.Map{}, World: w}
	source := []sav.DeadActor{
		{Identity: 100, ArchiveIndex: 2, Class: "Human", MapUnitID: 0, RuntimeID: 39, Cell: 0x0304, FineX: 128, FineY: 128, Stage: 2, HP: -14},
		{Identity: 200, ArchiveIndex: 3, Class: "Human", MapUnitID: 0, RuntimeID: 40, Cell: 0x0304, FineX: 128, FineY: 128, Stage: 4, HP: -58},
	}
	var report OriginalSaveResume
	if err := applyOriginalDead(ms, source, &report); err != nil {
		t.Fatal(err)
	}
	got := w.OriginalDeadActors()
	if len(got) != 2 || got[0].ID != 8 || got[1].ID != 9 || got[0].Source.Identity != 100 || got[1].Source.Identity != 200 {
		t.Fatalf("fresh identities do not preserve both records: %+v", got)
	}
	if report.UnboundRestored != 2 || report.CorpsesRestored != 0 || report.TerminalRestored != 0 || !reflect.DeepEqual(before, w.Entities()) {
		t.Fatalf("unbound import changed the live population or counters: %+v", report)
	}
	if next, ok := w.NextEntityID(); !ok || next != 10 {
		t.Fatalf("fresh identity reservation lost: %d %t", next, ok)
	}
}
