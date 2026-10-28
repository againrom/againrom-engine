package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// unitResidualMapUnitsForAudit groups the records LOAD builds as entities by
// map unit ID. Several records can share one ID: LOAD keeps duplicate source
// map IDs as distinct actors (planOriginalActors).
func unitResidualMapUnitsForAudit(want map[uint16]sav.ActorRecord) map[uint16][]sav.ActorRecord {
	out := make(map[uint16][]sav.ActorRecord)
	for _, a := range want {
		if a.MapUnitID != 0 {
			out[a.MapUnitID] = append(out[a.MapUnitID], a)
		}
	}
	return out
}

// unitResidualRecordForAudit finds one live entity's own saved record. An
// imported entity is its archive record. An entity without an archive binding,
// as LOAD keeps an Againrom-written SAV's actors (DIV-2885), is the one record
// carrying its map unit ID; a shared ID names no record.
func unitResidualRecordForAudit(e sim.Entity, byArchive map[uint16]sav.ActorRecord, byMapUnit map[uint16][]sav.ActorRecord) (sav.ActorRecord, bool) {
	if e.SourceBinding.Class != 0 && !e.SourceBinding.Generated() {
		a, ok := byArchive[e.SourceBinding.ArchiveIndex]
		return a, ok
	}
	if got := byMapUnit[e.MapUnitID]; e.MapUnitID != 0 && len(got) == 1 {
		return got[0], true
	}
	return sav.ActorRecord{}, false
}

// unitResidualWantForAudit is the own weight, load and whole-cell sight LOAD
// keeps for one record: the stored words (SAV-794). A Unit record holding
// capacity 0 and own weight 0 was written by this engine before every Unit
// record carried capacity 300 and its own weight (DIV-1675; SAV-UNITFLD-049:
// the constructor's 300 is never recomputed). LOAD gives it the own weight the
// load law leaves beside its container (ITEM-LOAD-005), which its SAVE writes.
func unitResidualWantForAudit(c sav.Character) (own, load int16, sight uint8) {
	s := c.LoadState
	own = s.OwnWeight
	if c.Basis.Class == "Unit" && s.Capacity == 0 && own == 0 {
		carried := sim.ActorLoad{Present: true, ContainerPresent: s.ContainerPresent, Accumulator: s.Accumulator}.CurrentLoad()
		if int32(s.Load) > carried {
			own = s.Load - int16(carried)
		}
	}
	return own, s.Load, uint8(c.Basis.Human.Fields.Sight >> 8)
}

func unitResidualRecordForTest(archive, mapUnit uint16, class string, load sav.ActorLoadState, sight uint16) sav.ActorRecord {
	basis := &sav.ActorBasis{Class: class}
	basis.Human.Fields.Sight = sight
	load.Present = true
	return sav.ActorRecord{Actor: sav.Actor{RuntimeID: uint32(archive), MapUnitID: mapUnit, HP: 10}, ArchiveIndex: archive,
		Character: sav.Character{Basis: basis, Class: class, LoadState: load}}
}

// Three records sharing map unit 128 each pair with their own entity. A join
// keyed by map unit ID alone would give all three one record (loss control).
func TestUnitResidualRecordForAuditJoinsEachActorToItsOwnRecord(t *testing.T) {
	graph := sav.SavedActorGraph{Actors: []sav.ActorRecord{
		unitResidualRecordForTest(642, 128, "Unit", sav.ActorLoadState{OwnWeight: 180, Load: 180, Capacity: 300}, 0x0700),
		unitResidualRecordForTest(837, 128, "Unit", sav.ActorLoadState{OwnWeight: 180, Load: 180, Capacity: 300}, 0x0700),
		unitResidualRecordForTest(1018, 128, "Unit", sav.ActorLoadState{OwnWeight: 29, Load: 29, Capacity: 300}, 0x0800),
		unitResidualRecordForTest(9, 3, "Unit", sav.ActorLoadState{OwnWeight: 5, Load: 5, Capacity: 300}, 0x0700),
	}}
	byArchive := archiveActorsForAudit(graph)
	byMapUnit := unitResidualMapUnitsForAudit(byArchive)
	for _, archive := range []uint16{642, 837, 1018} {
		e := sim.Entity{MapUnitID: 128, SourceBinding: sim.SourceBinding{Class: 1, ArchiveIndex: archive}}
		got, ok := unitResidualRecordForAudit(e, byArchive, byMapUnit)
		if !ok || got.ArchiveIndex != archive {
			t.Fatalf("entity of archive %d paired with archive %d (found %v)", archive, got.ArchiveIndex, ok)
		}
	}
	if got, ok := unitResidualRecordForAudit(sim.Entity{MapUnitID: 3}, byArchive, byMapUnit); !ok || got.ArchiveIndex != 9 {
		t.Fatalf("unbound entity of map unit 3 paired with archive %d (found %v)", got.ArchiveIndex, ok)
	}
	if got, ok := unitResidualRecordForAudit(sim.Entity{MapUnitID: 128}, byArchive, byMapUnit); ok {
		t.Fatalf("unbound entity of shared map unit 128 paired with archive %d", got.ArchiveIndex)
	}
	if n := len(byMapUnit[128]); n != 3 {
		t.Fatalf("map unit 128 groups %d records, want 3", n)
	}
}

// Only a Unit record holding capacity 0 gains an own weight; an original Unit
// record (capacity 300) and a Human keep the stored 0 (loss control).
func TestUnitResidualWantForAuditKeepsStoredWordsButRepairsZeroUnits(t *testing.T) {
	for _, c := range []struct {
		name  string
		class string
		load  sav.ActorLoadState
		own   int16
	}{
		{"zero-capacity Unit, empty container", "Unit", sav.ActorLoadState{Load: 20, ContainerPresent: true}, 20},
		{"zero-capacity Unit, carried 10", "Unit", sav.ActorLoadState{Load: 25, ContainerPresent: true, Accumulator: 20}, 15},
		{"zero-capacity Unit at its container's load", "Unit", sav.ActorLoadState{Load: 10, ContainerPresent: true, Accumulator: 20}, 0},
		{"original Unit", "Unit", sav.ActorLoadState{Load: 20, Capacity: 300, ContainerPresent: true}, 0},
		{"Human", "Human", sav.ActorLoadState{Load: 267, ContainerPresent: true}, 0},
		{"stored own weight", "Unit", sav.ActorLoadState{OwnWeight: 29, Load: 29, ContainerPresent: true}, 29},
	} {
		r := unitResidualRecordForTest(1, 1, c.class, c.load, 0x0800)
		own, load, sight := unitResidualWantForAudit(r.Character)
		if own != c.own || load != c.load.Load || sight != 8 {
			t.Errorf("%s: want %d/%d/8, got %d/%d/%d", c.name, c.own, c.load.Load, own, load, sight)
		}
	}
}
