package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// sameMoverRouteForAudit compares a file's own decoded mover/route state,
// keyed by archive index, against the corresponding live saved motions,
// keyed the same way through Entity.SourceBinding.ArchiveIndex. Mirroring
// sameSpellEffectGraphForAudit's own role (originalspelleffects1132_test.go),
// it is independent of exportOriginalMoverRoutes: it never calls it, so a bug
// shared with the production join would not silently pass both this and the
// byte-level export check the corpus/release tests also run.
func sameMoverRouteForAudit(file map[uint16]sav.ActorRecord, live map[uint16]sim.SavedActorMotion) (string, bool) {
	if len(file) != len(live) {
		return fmt.Sprintf("count %d vs %d", len(file), len(live)), false
	}
	sameWords := func(a, b []uint16) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	for archiveIndex, f := range file {
		l, ok := live[archiveIndex]
		if !ok {
			return fmt.Sprintf("actor %d: file has no live motion", archiveIndex), false
		}
		if !f.HasMover {
			return fmt.Sprintf("actor %d: file record has no mover block", archiveIndex), false
		}
		if f.Mover != l.Mover {
			return fmt.Sprintf("actor %d: mover block differs", archiveIndex), false
		}
		if !sameWords(f.StaticRoute, l.StaticRoute) || !sameWords(f.DynamicRoute, l.DynamicRoute) {
			return fmt.Sprintf("actor %d: route lists differ: file static=%v dynamic=%v live static=%v dynamic=%v",
				archiveIndex, f.StaticRoute, f.DynamicRoute, l.StaticRoute, l.DynamicRoute), false
		}
	}
	return "", true
}

// archiveMotionsForAudit keys a world's live saved motions by archive index
// through Entity.SourceBinding.ArchiveIndex, the same join
// exportOriginalMoverRoutes performs, so a corpus/release test can build its
// comparison value without depending on that function's own internals.
func archiveMotionsForAudit(w *sim.World) map[uint16]sim.SavedActorMotion {
	byEntity := make(map[sim.EntityID]uint16)
	for _, e := range w.Entities() {
		if e.SourceBinding.Class != 0 {
			byEntity[e.ID] = e.SourceBinding.ArchiveIndex
		}
	}
	motions, _, _, _ := w.SavedActorMotions()
	out := make(map[uint16]sim.SavedActorMotion, len(motions))
	for _, m := range motions {
		if archiveIndex, ok := byEntity[m.Entity]; ok {
			out[archiveIndex] = m
		}
	}
	return out
}

// archiveActorsForAudit keys a file's own decoded actor graph by archive
// index, restricted to the actors LOAD constructs as entities
// (ownsOriginalEntity): living, stage-1 dying and current, never terminal.
// Each gets a source binding and a saved motion, so this is the population
// exportOriginalMoverRoutes writes back. A stage-1 dying actor keeps its
// stored state and binds at LOAD (DIV-1003, SAV-DEADLOAD-126).
func archiveActorsForAudit(graph sav.SavedActorGraph) map[uint16]sav.ActorRecord {
	out := make(map[uint16]sav.ActorRecord, len(graph.Actors))
	for _, a := range graph.Actors {
		if ownsOriginalEntity(a) {
			out[a.ArchiveIndex] = a
		}
	}
	return out
}

// moverRoutesCarriedForAudit reports whether a file's mover/route records are
// an oracle for the live saved motions: an original's always are, an
// Againrom-written SAV's only when its WriterCarriers holds the mover/route
// bit. Without it LOAD drops the imported motions and its actors carry no
// source binding (DIV-2885).
func moverRoutesCarriedForAudit(source *sav.File, raw []byte) (bool, error) {
	bits, written, err := milestone2WriterCarriers(milestone2File{raw: raw, f: source})
	if err != nil {
		return false, err
	}
	return !written || bits&currentCarrierMoverRoutes != 0, nil
}

// sameMoverRouteByIdentityForAudit compares a file's living actors with the
// records a SAVE wrote, joined by Identity: a SAVE renumbers archive indices.
// Each living actor must be written once with its mover block and both route
// lists unchanged.
func sameMoverRouteByIdentityForAudit(file sav.SavedActorGraph, saved sav.SavedActorGraph) (int, string, bool) {
	written := make(map[uint32][]sav.ActorRecord, len(saved.Actors))
	for _, a := range saved.Actors {
		written[a.Identity] = append(written[a.Identity], a)
	}
	living := 0
	for _, f := range file.Actors {
		if f.Dead() {
			continue
		}
		living++
		got := written[f.Identity]
		if f.Identity == 0 || len(got) != 1 {
			return living, fmt.Sprintf("actor %d (identity %d): SAVE wrote %d record(s)", f.ArchiveIndex, f.Identity, len(got)), false
		}
		l := sim.SavedActorMotion{Mover: got[0].Mover, StaticRoute: got[0].StaticRoute, DynamicRoute: got[0].DynamicRoute}
		if !got[0].HasMover {
			return living, fmt.Sprintf("actor %d (identity %d): SAVE wrote no mover block", f.ArchiveIndex, f.Identity), false
		}
		if msg, ok := sameMoverRouteForAudit(map[uint16]sav.ActorRecord{0: f}, map[uint16]sim.SavedActorMotion{0: l}); !ok {
			return living, fmt.Sprintf("identity %d, file archive %d: %s", f.Identity, f.ArchiveIndex, msg), false
		}
	}
	return living, "", true
}

func TestArchiveActorsForAuditKeepsTheEntitiesLoadBuilds(t *testing.T) {
	graph := sav.SavedActorGraph{Actors: []sav.ActorRecord{
		{Actor: sav.Actor{RuntimeID: 1, HP: 10}, ArchiveIndex: 1},
		{Actor: sav.Actor{RuntimeID: 2, Stage: 1, HP: 0}, ArchiveIndex: 2},
		{Actor: sav.Actor{RuntimeID: 3, Stage: 1, HP: -11}, ArchiveIndex: 3},
		{Actor: sav.Actor{RuntimeID: 4, Stage: 3, HP: -20}, ArchiveIndex: 4},
		{Actor: sav.Actor{RuntimeID: 5, Stage: 1, HP: 0}, TerminalActor: true, ArchiveIndex: 5},
		{Actor: sav.Actor{RuntimeID: 6, Stage: 4, HP: -30}, CurrentEntity: true, ArchiveIndex: 6},
		{Actor: sav.Actor{HP: 10}, ArchiveIndex: 7},
	}}
	var got []uint16
	for archiveIndex := range archiveActorsForAudit(graph) {
		got = append(got, archiveIndex)
	}
	slices.Sort(got)
	// Living, both stage-1 dying records and the current one; the stage-3
	// corpse, the terminal record and the record without a runtime ID stay out.
	if want := []uint16{1, 2, 3, 6}; !slices.Equal(got, want) {
		t.Fatalf("archiveActorsForAudit kept %v, want %v", got, want)
	}
}

func TestSameMoverRouteByIdentityJoinsSavedRecords(t *testing.T) {
	var mover [180]byte
	mover[5] = 0x41
	record := func(identity uint32, archiveIndex uint16, hp int16, stage uint8, dynamic ...uint16) sav.ActorRecord {
		return sav.ActorRecord{Actor: sav.Actor{RuntimeID: 1, HP: hp, Stage: stage}, Identity: identity, ArchiveIndex: archiveIndex,
			HasMover: true, Mover: mover, StaticRoute: []uint16{0x0101}, DynamicRoute: dynamic}
	}
	file := sav.SavedActorGraph{Actors: []sav.ActorRecord{record(10, 1, 5, 0, 0x0202), record(20, 2, 0, 1)}}
	// A SAVE renumbers archive indices and may leave a dying record out.
	if n, msg, ok := sameMoverRouteByIdentityForAudit(file, sav.SavedActorGraph{Actors: []sav.ActorRecord{record(10, 9, 5, 0, 0x0202)}}); !ok || n != 1 {
		t.Fatalf("renumbered record: n=%d ok=%v %s", n, ok, msg)
	}
	noMover := record(10, 9, 5, 0, 0x0202)
	noMover.HasMover = false
	otherMover := record(10, 9, 5, 0, 0x0202)
	otherMover.Mover[1] = 3
	for name, saved := range map[string][]sav.ActorRecord{
		"route":     {record(10, 9, 5, 0, 0x0203)},
		"mover":     {otherMover},
		"no mover":  {noMover},
		"missing":   {record(11, 9, 5, 0, 0x0202)},
		"duplicate": {record(10, 9, 5, 0, 0x0202), record(10, 8, 5, 0, 0x0202)},
	} {
		if _, msg, ok := sameMoverRouteByIdentityForAudit(file, sav.SavedActorGraph{Actors: saved}); ok {
			t.Errorf("%s: a differing SAVE compared equal", name)
		} else if msg == "" {
			t.Errorf("%s: no message", name)
		}
	}
}
