package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// deadPatrolTroll is the mission-141 troll the script's opening trigger sends
// on patrol.
const deadPatrolTroll = 57

func deadPatrolEntity(t *testing.T, f *FrontEnd, mapUnit uint16) sim.Entity {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.MapUnitID == mapUnit {
			return e
		}
	}
	t.Fatalf("map unit %d is not in the world", mapUnit)
	return sim.Entity{}
}

func deadPatrolSave(t *testing.T, f *FrontEnd, store SaveStore) (string, sav.DocumentData) {
	t.Helper()
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE = %q: %v", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return name, doc
}

func deadPatrolLoad(t *testing.T, store SaveStore, name string) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	_, _, load := f.SaveSeams(store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("LOAD of %s: town=%t %v", name, town, err)
	}
	if err := f.App("dead patrol reload").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func deadPatrolOrder(t *testing.T, doc sav.DocumentData, mapUnit uint16) (uint32, []uint16) {
	t.Helper()
	r := patrolRecordByMapUnit(t, doc, mapUnit)
	state, err := savedActorRaw(r, "U50", 4)
	if err != nil {
		t.Fatal(err)
	}
	return binary.LittleEndian.Uint32(state), patrolWordList(t, r, "U158_90")
}

// A patroller whose order was loaded from this engine's own SAV (U50 0x0a and
// its ring) dies through ordinary damage. The next SAVE writes its body with
// terminal state and no ring, a living patroller in the same file keeps
// 0x0a with its ring, and LOAD of that SAVE continues for 64 ticks. A file
// that still carries the retained 0x0a and ring on the body loads to the same
// World: the state is residue and is dropped.
func TestReleaseDeadPatrollerSAVReloads(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("dead patrol").OpenMission(f.MissionOpener(141)); err != nil {
		t.Fatal(err)
	}
	for f.live.world.Tick() < 113 {
		f.live.tick()
	}
	troll := deadPatrolEntity(t, f, deadPatrolTroll)
	if !troll.Alive() || troll.ActorState != 0x0a {
		t.Fatalf("troll %d at tick 113: HP %d state %#x, want a living patroller", deadPatrolTroll, troll.HP, troll.ActorState)
	}
	store := SaveStore{Dir: t.TempDir()}
	first, doc := deadPatrolSave(t, f, store)
	state, ring := deadPatrolOrder(t, doc, deadPatrolTroll)
	if state != 0x0a || len(ring) != 2 {
		t.Fatalf("first SAVE wrote the troll U50 %#x ring %v, want 0xa and a two-node ring", state, ring)
	}
	loadedRing := ring

	f = deadPatrolLoad(t, store, first)
	var patroller uint16
	for _, e := range f.live.world.Entities() {
		if e.Alive() && e.ActorState == 0x0a && e.MapUnitID != 0 && e.MapUnitID != deadPatrolTroll && e.Owner != sim.SelfSlot {
			patroller = e.MapUnitID
			break
		}
	}
	if patroller == 0 {
		t.Fatal("no second living patroller after LOAD")
	}
	troll = deadPatrolEntity(t, f, deadPatrolTroll)
	headlessDamage(t, f.live.world, troll.ID, troll.HP+16)
	for i := 0; i < 300 && deadPatrolEntity(t, f, deadPatrolTroll).Decay < sim.DecayBones; i++ {
		f.live.tick()
	}
	body := deadPatrolEntity(t, f, deadPatrolTroll)
	if body.Alive() || body.HP > -10 || body.Decay < sim.DecayBones {
		t.Fatalf("troll did not die: HP %d stage %d", body.HP, body.Decay)
	}
	second, doc := deadPatrolSave(t, f, store)
	state, ring = deadPatrolOrder(t, doc, deadPatrolTroll)
	if state != 0x10 || len(ring) != 0 {
		t.Fatalf("SAVE wrote the terminal body U50 %#x ring %v, want 0x10 and no ring", state, ring)
	}
	action, err := savedActorRaw(patrolRecordByMapUnit(t, doc, deadPatrolTroll), "U54", 4)
	if err != nil || binary.LittleEndian.Uint32(action) != 0x10 {
		t.Fatalf("SAVE wrote the terminal body U54 %x: %v, want 10000000", action, err)
	}
	live := deadPatrolEntity(t, f, patroller)
	state, ring = deadPatrolOrder(t, doc, patroller)
	want := []uint16{patrolCell(live.PatrolHeadX, live.PatrolHeadY), patrolCell(live.PatrolTailX, live.PatrolTailY)}
	if !live.Alive() || state != 0x0a || !slices.Equal(ring, want) {
		t.Fatalf("living patroller %d wrote U50 %#x ring %v, want 0xa and %v", patroller, state, ring, want)
	}

	clean := deadPatrolLoad(t, store, second)
	back := deadPatrolEntity(t, clean, deadPatrolTroll)
	if back.Alive() || back.HP != body.HP || back.Decay != body.Decay || back.ActorState != body.ActorState {
		t.Fatalf("reloaded body HP %d stage %d state %#x, want HP %d stage %d state %#x",
			back.HP, back.Decay, back.ActorState, body.HP, body.Decay, body.ActorState)
	}

	t.Run("retained patrol on the body", func(t *testing.T) {
		r := patrolRecordByMapUnit(t, doc, deadPatrolTroll)
		u50, err := savedActorRaw(r, "U50", 4)
		if err != nil {
			t.Fatal(err)
		}
		binary.LittleEndian.PutUint32(u50, 0x0a)
		next, sites, err := cloneSavedGroupFieldRecord(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := sites.wordList(&next, "U158_90", loadedRing); err != nil {
			t.Fatal(err)
		}
		*r = next
		raw, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		residue := SaveStore{Dir: t.TempDir()}
		if err := os.WriteFile(filepath.Join(residue.Dir, "game0000.sav"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		g := deadPatrolLoad(t, residue, "game0000.sav")
		e := deadPatrolEntity(t, g, deadPatrolTroll)
		if e.ActorState != 0x0b || e.PatrolHeadX|e.PatrolHeadY|e.PatrolTailX|e.PatrolTailY != 0 {
			t.Fatalf("body restored with state %#x ring (%d,%d)-(%d,%d), want guard and no ring",
				e.ActorState, e.PatrolHeadX, e.PatrolHeadY, e.PatrolTailX, e.PatrolTailY)
		}
		if g.live.world.Hash() != clean.live.world.Hash() {
			t.Fatalf("residue LOAD hash %016x, clean LOAD hash %016x", g.live.world.Hash(), clean.live.world.Hash())
		}
	})

	for range 64 {
		clean.live.tick()
	}
	if e := deadPatrolEntity(t, clean, patroller); !e.Alive() || e.ActorState != 0x0a {
		t.Fatalf("living patroller %d after 64 ticks: HP %d state %#x", patroller, e.HP, e.ActorState)
	}
}
