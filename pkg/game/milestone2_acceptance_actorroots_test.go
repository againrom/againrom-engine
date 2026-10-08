//go:build sessioncorpusaudit

package game

import "testing"

func TestMilestone2ActorRoots(t *testing.T) {
	worlds, cities, actors, children, mismatches := 0, 0, 0, 0, 0
	var total actor1161Population
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		roots, r, origins, err := readActorRoots(mf.f, mf.raw)
		if err != nil {
			mismatches++
			t.Error("independent actor roots", err)
			return
		}
		actors += len(roots)
		children += len(r.source.rows)
		decoded, _ := decodeSavedDocument(mf.raw)
		for _, diff := range actor1161DocumentDifferences(roots, r, origins, decoded.Document) {
			mismatches++
			t.Error("decoded Document: " + diff)
		}
		if !mf.present {
			cities++
			return
		}
		ms, _, err := ResumeOriginalSave(fe.Archives.Containers, mf.raw, fe.Table, fe.Difficulty, nil, fe.Bodies)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		worlds++
		for _, diff := range actor1161DocumentDifferences(roots, r, origins, ms.savedDocument.Document) {
			mismatches++
			t.Error("original BEFORE Snapshot: " + diff)
		}
		diffs, p := actor1161WorldDifferences(roots, r, origins, ms.savedDocument, ms.World, ms)
		for _, diff := range diffs {
			mismatches++
			t.Error("initial World: " + diff)
		}
		total.live += p.live
		total.rawOnly += p.rawOnly
		total.books += p.books
		total.spellRefs += p.spellRefs
		total.effects += p.effects
		total.u68 += p.u68
	})
	milestone2LogRefusals(t, "actor roots", refused)
	t.Logf("actor roots: %d world files resumed, %d city files decode-only; %d raw actors, %d reachable item/effect/spell rows; %d live actors, %d raw-only late-dead actors; %d live books, %d non-null spell slots, %d timed actor effects, %d U68 refs; %d mismatches, %d refused", worlds, cities, actors, children, total.live, total.rawOnly, total.books, total.spellRefs, total.effects, total.u68, mismatches, len(refused))
	t.Log("actor roots: complete held/worn/pack/book edges and child fields, inventory headers, sparse/absent book, U68/Diary refs and 17-byte tail compared from Body; live attachment timers restored without replaying modifiers; late-dead roots remain Document-only")
	if worlds == 0 || actors == 0 || total.live == 0 {
		t.Fatal("no actor-root acceptance population")
	}
}
