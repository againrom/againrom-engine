package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseSourceBoundDeadVIPSurvivesSAV(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	path, payload := groundCorpusFile(t, "2026-08-02/game0002.sav", "b1ce079cc2b3f1bc101862c1dcf2afd8237421458b3e2474c4447efe5df2e761")
	open, town, err := f.RestoreOriginal(payload)
	if err != nil || town {
		t.Fatal("original mission LOAD", path, town, err)
	}
	app := f.App("source-bound dead VIP")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	var victim sim.Entity
	for _, e := range w.Entities() {
		if e.MapUnitID == 21 {
			victim = e
			break
		}
	}
	if victim.ID != 2 || victim.SourceBinding.Class != 2 || victim.SourceBinding.Identity != 0x4c52b98 || victim.HP != 31 || victim.Domain != sim.DomainGround {
		t.Fatalf("owner SAV no longer binds the checked actor: %+v", victim)
	}
	checked := false
	for _, check := range w.Script().Checks() {
		if check.Op == sim.ScriptCheckVIP && check.HasUnit && check.Unit == victim.ID {
			checked = true
		}
	}
	if !checked {
		t.Fatal("the installed script does not check this actor as a VIP")
	}
	_, lostBefore := w.ScriptCounters()
	if err := w.HeadlessDamage(victim.ID, victim.HP+1002); err != nil {
		t.Fatal(err)
	}
	var lastStage sim.DecayStage
	removedAt := -1
	for tick := 0; tick < 64; tick++ {
		found := false
		for _, e := range w.Entities() {
			if e.ID == victim.ID {
				lastStage, found = e.Decay, true
				break
			}
		}
		if !found {
			removedAt = tick
			break
		}
		f.live.tick()
	}
	if removedAt < 0 || lastStage >= sim.DecayBones {
		t.Fatalf("the checked actor did not leave before bones: tick %d stage %d", removedAt, lastStage)
	}
	var row sim.OriginalDeadRecord
	for _, d := range w.OriginalDeadActors() {
		if d.ID == victim.ID {
			row = d
			break
		}
	}
	if row.Source.Identity != victim.SourceBinding.Identity || row.Current.Stage != 5 || row.Current.HP != -10001 ||
		!row.Source.ContainerPresent || row.Source.ContainerTail != [2]uint32{10000, 0} {
		t.Fatalf("removed source-bound actor has no current retained row: %+v", row)
	}
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	store, name, _ := menuSAVE(t, f, app, OriginalStore{})
	cold := loadLocalLegacySave(t, store, name)
	after, _, err := cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.World, after.World) {
		t.Fatal("cold SAV LOAD changed the World")
	}
	var loaded sim.OriginalDeadRecord
	for _, d := range cold.live.world.OriginalDeadActors() {
		if d.ID == victim.ID {
			loaded = d
			break
		}
	}
	if !reflect.DeepEqual(row, loaded) {
		t.Fatalf("cold SAV LOAD changed the checked dead row: %+v -> %+v", row, loaded)
	}
	for range 32 {
		f.live.tick()
		cold.live.tick()
	}
	_, liveLost := w.ScriptCounters()
	_, coldLost := cold.live.world.ScriptCounters()
	if liveLost <= lostBefore || coldLost != liveLost || w.Outcome() != sim.OutcomeLost || cold.live.world.Outcome() != w.Outcome() || cold.live.world.Hash() != w.Hash() {
		t.Fatalf("the VIP check diverged after cold LOAD: lost %d -> %d/%d, outcome %d/%d, World %x/%x",
			lostBefore, liveLost, coldLost, w.Outcome(), cold.live.world.Outcome(), w.Hash(), cold.live.world.Hash())
	}
	t.Logf("%s map unit 21 left at tick %d in stage %d; menu SAV, cold LOAD and VIP loss %d -> %d matched", path, removedAt, lastStage, lostBefore, liveLost)
}
