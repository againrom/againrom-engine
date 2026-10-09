//go:build sessioncorpusaudit

package game

import "testing"

func TestMilestone2DeadRoots(t *testing.T) {
	worlds, cities, occurrences, unique, mismatches, materialized, virtual, terminal := 0, 0, 0, 0, 0, 0, 0, 0
	classes := map[uint8]int{}
	stages := map[uint8]int{}
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		want, err := dead1163Expected(mf.f, mf.raw)
		if err != nil {
			mismatches++
			t.Error("independent dead roots", err)
			return
		}
		occurrences += len(want.roots)
		unique += len(want.actors)
		for _, a := range want.actors {
			classes[a.Class]++
			stages[a.State.Stage]++
		}
		decoded, _ := decodeSavedDocument(mf.raw)
		for _, diff := range want.documentDifferences(decoded) {
			mismatches++
			t.Error("decoded Document: " + diff)
		}
		if !mf.present {
			cities++
			return
		}
		ms, _, err := loadOriginalMission(fe, mf.raw)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		worlds++
		for _, diff := range want.documentDifferences(ms.savedDocument) {
			mismatches++
			t.Error("imported Document BEFORE Snapshot: " + diff)
		}
		for _, diff := range want.liveDifferences(ms.World.OriginalDeadActors(), ms.World.Entities()) {
			mismatches++
			t.Error("initial World: " + diff)
		}
		for _, a := range want.actors {
			if a.MapUnitID == 0 {
				virtual++
			} else if a.State.Stage == 5 {
				terminal++
			} else {
				materialized++
			}
		}
	})
	milestone2LogRefusals(t, "dead roots", refused)
	t.Logf("dead roots: %d world files resumed,%d city files; %d raw root occurrences,%d unique archives,%d repeated roots; classes%v stages%v; %d materialized,%d virtual,%d terminal; %d mismatches,%d refused", worlds, cities, occurrences, unique, occurrences-unique, classes, stages, materialized, virtual, terminal, mismatches, len(refused))
	t.Log("dead roots: source state/identity/references/held weapon and initial current tuple checked; full root order/multiplicity stays in Document; World deduplicates archive references; original occupancy/first callbacks remain Unknown")
	if worlds == 0 || unique == 0 {
		t.Fatal("no dead-root acceptance population")
	}
}
