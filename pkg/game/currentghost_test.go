package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentSaveKeepsCapturedGhostAcrossColdLoads(t *testing.T) {
	f := currentArchiveFixture(t)
	ghost := sim.GhostTemplate{Class: 69, TypeID: 69, Domain: sim.DomainGhost, Speed: 10, RotationSpeed: 16, ScanRange: 6, Reach: 1, TokenSize: 1, DyingTime: 12, XPValue: 20}
	policy := f.live.world.CurrentPolicy()
	policy.Ghost = &ghost
	if err := f.live.world.RestoreCurrentContinuation(&policy, nil, f.live.world.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.ghost == nil || *snapshot.ghost != ghost {
			t.Fatal("snapshot lost current Ghost")
		}
		// The producer must use the captured template even if its caller has
		// since opened another mission with a different constructor input.
		other := ghost
		other.Speed += 5
		policy = f.live.world.CurrentPolicy()
		policy.Ghost = &other
		if err := f.live.world.RestoreCurrentContinuation(&policy, nil, f.live.world.Actions(), nil); err != nil {
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
		a, err := readCurrentActions(&doc)
		if err != nil || a == nil || a.Policy == nil || a.Policy.Ghost == nil || *a.Policy.Ghost != ghost {
			t.Fatal("SAV lost captured Ghost", err)
		}
		if f.live.world.Ghost() != other {
			t.Fatal("SAVE changed live constructor")
		}
		f = openCurrentArchive(t, raw)
		if f.live.world.Ghost() != ghost {
			t.Fatal("cold LOAD lost captured Ghost", cycle)
		}
	}
}

func TestCurrentSaveLegacySnapshotUsesInstalledGhost(t *testing.T) {
	f := currentArchiveFixture(t)
	ghost := sim.GhostTemplate{Class: 69, TypeID: 69, Domain: sim.DomainGhost, Speed: 7, TokenSize: 1}
	policy := f.live.world.CurrentPolicy()
	policy.Ghost = &ghost
	if err := f.live.world.RestoreCurrentContinuation(&policy, nil, f.live.world.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.ghost = nil
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	cold := openCurrentArchive(t, raw)
	if cold.live.world.Ghost() != ghost {
		t.Fatal("legacy capture did not use current installed constructor")
	}
}
