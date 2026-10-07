package game

import (
	"encoding/binary"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type patrolWitness struct {
	mapUnit uint16
	ring    []uint16
}

func patrolCell(x, y int32) uint16 { return uint16(x) | uint16(y)<<8 }

// patrolRecordByMapUnit finds the one living Unit-family record whose map
// unit is id.
func patrolRecordByMapUnit(t *testing.T, doc sav.DocumentData, id uint16) *sav.DocumentRecordData {
	t.Helper()
	var found *sav.DocumentRecordData
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
			continue
		}
		if v, err := savedStructureValue(r, "T08"); err != nil || uint16(v) != id {
			continue
		}
		if found != nil {
			t.Fatalf("map unit %d names two records", id)
		}
		found = r
	}
	if found == nil {
		t.Fatalf("map unit %d has no record", id)
	}
	return found
}

func patrolWordList(t *testing.T, r *sav.DocumentRecordData, name string) []uint16 {
	t.Helper()
	for _, field := range r.Raw {
		if field.Name != name {
			continue
		}
		var out []uint16
		for i := 0; i+1 < len(field.Bytes); i += 2 {
			out = append(out, binary.LittleEndian.Uint16(field.Bytes[i:]))
		}
		return out
	}
	t.Fatalf("record has no %s list", name)
	return nil
}

// Monsters the mission-141 script sends on patrol are written as patrollers:
// U50 0x0a, their ring in U158_90 in ring order and a cursor that is a ring
// member. A guard in the same file keeps 0x0b, and our LOAD keeps each
// patroller walking its ring.
func TestReleaseMissionSAVWritesScriptPatrols(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("patrol save").OpenMission(f.MissionOpener(141)); err != nil {
		t.Fatal(err)
	}
	for f.live.world.Tick() < 113 {
		f.live.tick()
	}
	var patrols []patrolWitness
	guard := uint16(0)
	for _, e := range f.live.world.Entities() {
		if !e.Alive() || e.Owner == sim.SelfSlot || e.MapUnitID == 0 {
			continue
		}
		switch e.ActorState {
		case 0x0a:
			patrols = append(patrols, patrolWitness{e.MapUnitID, []uint16{patrolCell(e.PatrolHeadX, e.PatrolHeadY), patrolCell(e.PatrolTailX, e.PatrolTailY)}})
		case 0x0b:
			if guard == 0 && !e.HasAttackTarget {
				guard = e.MapUnitID
			}
		}
	}
	if len(patrols) == 0 || guard == 0 {
		t.Fatalf("fixture has %d patrollers and guard map unit %d at tick 113", len(patrols), guard)
	}
	store := SaveStore{Dir: t.TempDir()}
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
	for _, p := range patrols {
		r := patrolRecordByMapUnit(t, doc, p.mapUnit)
		state, err := savedActorRaw(r, "U50", 4)
		if err != nil {
			t.Fatal(err)
		}
		order, err := savedActorRaw(r, "U158", 148)
		if err != nil {
			t.Fatal(err)
		}
		ring := patrolWordList(t, r, "U158_90")
		cursor := binary.LittleEndian.Uint16(order[2:])
		if binary.LittleEndian.Uint32(state) != 0x0a || !slices.Equal(ring, p.ring) || !slices.Contains(ring, cursor) {
			t.Fatalf("map unit %d wrote U50 %#x ring %v cursor %#x, want 0xa, ring %v and a ring-member cursor", p.mapUnit, binary.LittleEndian.Uint32(state), ring, cursor, p.ring)
		}
	}
	state, err := savedActorRaw(patrolRecordByMapUnit(t, doc, guard), "U50", 4)
	if err != nil || binary.LittleEndian.Uint32(state) != 0x0b {
		t.Fatalf("guard map unit %d wrote U50 %v, want 0xb (%v)", guard, state, err)
	}

	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	_, _, load := cold.SaveSeams(store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("LOAD of our SAV: town=%t %v", town, err)
	}
	if err := cold.App("patrol reload").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	for _, p := range patrols {
		for _, e := range cold.live.world.Entities() {
			if e.MapUnitID == p.mapUnit && e.ActorState != 0x0a {
				t.Fatalf("map unit %d reloaded in state %#x, want 0xa", p.mapUnit, e.ActorState)
			}
		}
	}
	// Each patroller keeps walking its ring: every target it holds is a ring
	// node, and within the window it reaches a node or closes on its target.
	dist := func(e sim.Entity) int32 {
		dx, dy := e.X-e.TargetX, e.Y-e.TargetY
		return max(dx, -dx) + max(dy, -dy)
	}
	firstDist, reached, closed := map[uint16]int32{}, map[uint16]bool{}, map[uint16]bool{}
	for tick := 0; tick < 200; tick++ {
		cold.live.tick()
		for _, e := range cold.live.world.Entities() {
			for _, p := range patrols {
				if e.MapUnitID != p.mapUnit {
					continue
				}
				if e.ActorState != 0x0a || e.HasTarget && !slices.Contains(p.ring, patrolCell(e.TargetX, e.TargetY)) {
					t.Fatalf("map unit %d after reload: state %#x target %d,%d, want patrol toward a ring node", p.mapUnit, e.ActorState, e.TargetX, e.TargetY)
				}
				if slices.Contains(p.ring, patrolCell(e.X, e.Y)) {
					reached[p.mapUnit] = true
				}
				if !e.HasTarget {
					continue
				}
				if d, ok := firstDist[p.mapUnit]; !ok {
					firstDist[p.mapUnit] = dist(e)
				} else if dist(e) < d {
					closed[p.mapUnit] = true
				}
			}
		}
	}
	for _, p := range patrols {
		if !reached[p.mapUnit] && !closed[p.mapUnit] {
			t.Fatalf("map unit %d after reload neither reached a ring node nor closed on its target within 200 ticks", p.mapUnit)
		}
	}
}
