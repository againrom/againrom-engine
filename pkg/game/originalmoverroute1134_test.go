package game

import (
	"fmt"

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
// index, restricted to living actors: exportOriginalMoverRoutes' own
// population (originalmoverroute.go), matching state.Actors'
// SourceBinding.Class != 0 join (savdocument.go).
func archiveActorsForAudit(graph sav.SavedActorGraph) map[uint16]sav.ActorRecord {
	out := make(map[uint16]sav.ActorRecord, len(graph.Actors))
	for _, a := range graph.Actors {
		if !a.Dead() {
			out[a.ArchiveIndex] = a
		}
	}
	return out
}
