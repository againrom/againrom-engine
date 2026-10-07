package game

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func loadLocalLegacySave(t *testing.T, store SaveStore, name string) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("cold explicit legacy save")
	app.Layout(1024, 768)
	save, list, load := agsSaveSeams(f, store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, name)
	return f
}

type corpseContinuationWitness struct {
	State        sim.DeadActorState
	Owner, Purse uint32
	Sacks        int
}

func requireCurrentCorpse(t *testing.T, f *FrontEnd, want corpseContinuationWitness) sim.Entity {
	t.Helper()
	var found []sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.RuntimeID == want.State.RuntimeID {
			found = append(found, e)
		}
	}
	if len(found) != 1 || found[0].Alive() || found[0].Decay != sim.DecayStage(want.State.Stage) || found[0].HP != int32(want.State.HP) || found[0].MapUnitID != 0 ||
		found[0].Owner != want.Owner || found[0].X != int32(want.State.Cell&255) || found[0].Y != int32(want.State.Cell>>8) {
		t.Fatalf("cold corpse = %+v, want one unbound dead entity with %+v", found, want.State)
	}
	corpse := found[0]
	drawn := 0
	for _, draw := range f.live.entityDraws() {
		if draw.ID != uint32(corpse.ID) {
			continue
		}
		drawn++
		if draw.Art == nil || draw.Frame == nil || draw.Life != ui.LifeDead || !draw.Untargetable ||
			draw.HP != int(want.State.HP) || draw.Owner != want.Owner ||
			draw.Cell.X != int(want.State.Cell&255) || draw.Cell.Y != int(want.State.Cell>>8) {
			t.Fatalf("cold corpse draw lost current body/owner/position: %+v", draw)
		}
	}
	if drawn != 1 || f.live.world.Purse(sim.SelfSlot) != want.Purse || len(f.live.world.Sacks()) != want.Sacks {
		t.Fatalf("cold LOAD changed draws/purse/sacks: %d/%d/%d, want 1/%d/%d", drawn, f.live.world.Purse(sim.SelfSlot), len(f.live.world.Sacks()), want.Purse, want.Sacks)
	}
	return corpse
}

func saveCorpseMission(t *testing.T, f *FrontEnd, dir string) string {
	t.Helper()
	save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		snapshot, _, snapshotErr := f.Snapshot(true)
		_, exportErr := f.ExportCurrentWorldSave(snapshot, "SAVE diagnostic")
		t.Fatalf("ordinary mission SAVE = %q: %v (snapshot: %v; SAV: %v)", name, err, snapshotErr, exportErr)
	}
	return filepath.Join(dir, name)
}

func TestReleaseNewCorpseSAVContinuation(t *testing.T) {
	t.Run("fighter", func(t *testing.T) { corpseSAVContinuationWitness(t, false) })
	t.Run("mage", func(t *testing.T) { corpseSAVContinuationWitness(t, true) })
}

func corpseSAVContinuationWitness(t *testing.T, mage bool) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	if input := os.Getenv("AGAINROM_CORPSE_SAV_INPUT"); input != "" {
		proof, err := os.ReadFile(input + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var want corpseContinuationWitness
		if err := json.Unmarshal(proof, &want); err != nil {
			t.Fatal(err)
		}
		f := loadLocalLegacySave(t, SaveStore{Dir: filepath.Dir(input)}, filepath.Base(input))
		initial := requireCurrentCorpse(t, f, want)
		for i := 0; i < 128; i++ {
			f.live.tick()
		}
		want.State.HP -= 4
		want.State.Stage = 3
		want.Purse, want.Sacks = f.live.world.Purse(sim.SelfSlot), len(f.live.world.Sacks())
		current := requireCurrentCorpse(t, f, want)
		if current.SourceBinding != initial.SourceBinding {
			t.Fatal("decay rewrote archive provenance")
		}
		second := saveCorpseMission(t, f, t.TempDir())
		fresh := loadLocalLegacySave(t, SaveStore{Dir: filepath.Dir(second)}, filepath.Base(second))
		requireCurrentCorpse(t, fresh, want)
		t.Logf("cold corpse advanced %d -> %d HP, stage %d -> %d; second SAV retains its body and owner", initial.HP, current.HP, initial.Decay, current.Decay)
		return
	}
	_, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	open, _, err := f.RestoreOriginal(raw)
	if err != nil {
		t.Fatal(err)
	}
	app := f.App("new corpse SAV continuation")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	var victim sim.Entity
	for _, e := range f.live.world.Entities() {
		typeID := uint8(e.SourceBinding.TypeID)
		if e.MapUnitID == 0 && e.SourceBinding.Class != 0 && e.Alive() && e.Domain == sim.DomainGround &&
			typeID >= 0x20 && typeID < 0x40 && ((typeID-0x21)&2 != 0) == mage {
			victim = e
		}
	}
	if victim.SourceBinding.Identity == 0 {
		t.Fatal("fixture has no source-bound actor without a map unit")
	}
	headlessDamage(t, f.live.world, victim.ID, victim.HP+16)
	for i := 0; i < 300; i++ {
		f.live.tick()
		victim = worldEntityByRuntimeID(t, f, victim.SourceBinding.RuntimeID)
		if victim.Decay >= sim.DecayBones {
			break
		}
	}
	if victim.Decay != sim.DecayBones || victim.HP < -19 || victim.HP > -16 {
		t.Fatalf("death did not reach the early corpse witness: %+v", victim)
	}
	want := corpseContinuationWitness{State: sim.DeadActorState{RuntimeID: victim.SourceBinding.RuntimeID,
		Cell: uint16(victim.Y)<<8 | uint16(victim.X), FineX: 128, FineY: 128, Stage: uint8(victim.Decay), HP: int16(victim.HP)},
		Owner: victim.Owner, Purse: f.live.world.Purse(sim.SelfSlot), Sacks: len(f.live.world.Sacks())}
	path := saveCorpseMission(t, f, t.TempDir())
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool {
		return setSavedActorHealth(doc, want.State.RuntimeID, want.State.HP-5)
	})
	if e := worldEntityByRuntimeID(t, lost, want.State.RuntimeID); e.HP != int32(want.State.HP)-5 {
		t.Fatalf("loss control: a SAV with the corpse HP altered loaded HP %d", e.HP)
	}
	proof, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_CORPSE_SAV_INPUT="+path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fresh-process corpse continuation: %v\n%s", err, output)
	}
	t.Logf("%s", output)
}
