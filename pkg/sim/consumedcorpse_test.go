package sim

import (
	"reflect"
	"testing"
)

func TestConsumedNativeCorpseZeroIDKeepsStageThroughCompactionAndScript(t *testing.T) {
	corpse := spEnt(0, 2, 1)
	corpse.HP, corpse.Decay, corpse.MapUnitID = -12, DecayBones, 7
	survivor := spEnt(2, 10, 11)
	s := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckHealth, Register: 0, Unit: 0, HasUnit: true, Args: [scriptParams]int32{6}},
		{Op: ScriptCheckAlive, Register: 1, Unit: 0, HasUnit: true},
	}, nil, nil)
	w, err := NewSummoningWorld(51, Bounds{16, 16}, ModeCanonical, Terrain{}, []Entity{corpse, survivor, effectMage(41, 1, 1, 1<<25)}, s,
		Relations{}, nil, nil, []SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}}, hlGhostTemplate())
	if err != nil {
		t.Fatal(err)
	}
	spRunCast(w, Cast(41, 0, 25))
	want := CurrentTerminalActor{ID: 0, Cell: 0x0102, HP: -10001, Stage: 5, MapUnitID: 7}
	if got := w.CurrentTerminalActors(); !reflect.DeepEqual(got, []CurrentTerminalActor{want}) {
		t.Fatalf("consumed current tuple %+v, want %+v", got, want)
	}
	if got := cbAt(t, w, survivor.ID); got.X != survivor.X || got.Y != survivor.Y || got.HP != survivor.HP {
		t.Fatal("compaction changed the survivor", got)
	}
	ghosts := 0
	for _, e := range w.Entities() {
		if e.TypeID == w.Ghost().TypeID && e.Alive() {
			ghosts++
		}
	}
	if ghosts != 1 {
		t.Fatal("successful placement did not create exactly one ghost", ghosts)
	}
	cold := dpReload(t, w)
	dpBoth(t, w, cold, scriptCycle)
	if cold.ScriptRegister(0) != -10001 || cold.ScriptRegister(1) != 0 || !reflect.DeepEqual(cold.CurrentTerminalActors(), []CurrentTerminalActor{want}) {
		t.Fatal("cold consumed zero-ID corpse lost tuple or script answer")
	}
	refs := cold.ActorIdentityReferences()
	ids := map[EntityID]EntityID{}
	for _, id := range refs {
		ids[id] = id + 100
	}
	if err := cold.RestoreActorIdentities(ids, nil); err != nil {
		t.Fatal(err)
	}
	want.ID = 100
	if got := cold.CurrentTerminalActors(); !reflect.DeepEqual(got, []CurrentTerminalActor{want}) || cold.Script().Checks()[0].Unit != 100 {
		t.Fatal("compacted terminal/script references did not remap together", got)
	}
}

func TestCurrentTerminalStageDomainKeepsOldStagesAndRejectsCorruptionAtomically(t *testing.T) {
	for stage := uint8(0); stage <= 5; stage++ {
		w := spWorld(t, 9, nil, spEnt(1, 1, 1))
		row := CurrentTerminalActor{ID: 0, Cell: 0x0203, HP: -10007, Stage: stage}
		if err := w.ImportCurrentTerminalActors([]CurrentTerminalActor{row}); err != nil {
			t.Fatalf("valid stage %d with independently chosen HP: %v", stage, err)
		}
		cold := dpReload(t, w)
		if got := cold.CurrentTerminalActors(); !reflect.DeepEqual(got, []CurrentTerminalActor{row}) {
			t.Fatal("valid tuple changed", got, row)
		}
		before := cold.Hash()
		bad := row
		bad.ID, bad.Stage = 2, 6
		if err := cold.ImportCurrentTerminalActors([]CurrentTerminalActor{bad}); err == nil || cold.Hash() != before {
			t.Fatal("stage >5 was accepted or changed World")
		}
	}
}

func TestConsumedSourceBoundCorpseKeepsTerminalProvenance(t *testing.T) {
	ents := dpBoundEnts(DomainGround, 17)
	ents[1].HP, ents[1].Decay = -12, DecayBones
	ents = append(ents, effectMage(3, 1, 2, 1<<25))
	w, err := NewSummoningWorld(51, Bounds{16, 16}, ModeCanonical, Terrain{}, ents, nil, Relations{}, nil, nil,
		[]SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}}, hlGhostTemplate())
	if err != nil {
		t.Fatal(err)
	}
	spRunCast(w, Cast(3, dpVictim, 25))
	rows := w.OriginalDeadActors()
	if len(rows) != 1 || rows[0].Source.Identity != 0x10000002 || rows[0].Current.Stage != 5 || rows[0].Current.HP != -10001 || len(w.CurrentTerminalActors()) != 0 {
		t.Fatal("consumed source-bound tuple/provenance changed", rows)
	}
	cold := dpReload(t, w)
	for range 256 {
		Step(cold, nil)
	}
	if !reflect.DeepEqual(rows, cold.OriginalDeadActors()) {
		t.Fatal("source-bound terminal decayed or lost provenance")
	}
}
