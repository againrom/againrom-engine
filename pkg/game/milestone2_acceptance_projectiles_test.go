//go:build sessioncorpusaudit

package game

import (
	"fmt"
	"testing"
)

func TestMilestone2Projectiles(t *testing.T) {
	sources, resumed, empty, ids, items, leaves, valueBytes, mismatches, nonemptyResumed := 0, 0, 0, 0, 0, 0, 0, 0, 0
	var refused []milestone2ResumeRefusal
	var nonempty []string
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		sources++
		want, err := projectile1157Read(mf.f.Store)
		if err != nil {
			mismatches++
			t.Error(err)
			return
		}
		ids, items, leaves = ids+len(want.ids), items+len(want.items), leaves+len(want.values)
		for _, value := range want.values {
			if value.kind&14 == 2 {
				valueBytes += 4
			} else {
				valueBytes += len(value.data)
			}
		}
		if len(want.items) == 0 {
			empty++
		} else {
			nonempty = append(nonempty, fmt.Sprintf("%s: allocator=%d IDs=%v items=%d", mf.rel, want.free, want.ids, len(want.items)))
		}
		// Source counts precede LOAD, so a refusal cannot shrink the oracle.
		ms, report, err := ResumeOriginalSave(fe.Archives.Containers, mf.raw, fe.Table, fe.Difficulty, nil, fe.Bodies)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		resumed++
		if len(want.items) > 0 {
			nonemptyResumed++
		}
		if report.ProjectilesApplied != want.present {
			mismatches++
			t.Error("raw leaf presence differs from LOAD report")
		}
		for _, difference := range want.worldDifferences(ms.World.SavedProjectiles()) {
			mismatches++
			t.Error("World: " + difference)
		}
		// Initial complete state, before a Snapshot can project over it.
		for _, difference := range want.documentDifferences(ms.savedDocument) {
			mismatches++
			t.Error("retained Document: " + difference)
		}
	})
	milestone2LogRefusals(t, "projectiles", refused)
	for _, source := range nonempty {
		t.Log("projectiles: nonempty source " + source)
	}
	t.Logf("projectiles: %d source worlds, %d resumed (%d nonempty), %d empty; %d ordered IDs, %d distinct items, %d raw leaves/%d value bytes; %d mismatches, %d refused; counts describe discovered paths, not distinct original-runtime documents", sources, resumed, nonemptyResumed, empty, ids, items, leaves, valueBytes, mismatches, len(refused))
	if resumed == 0 || nonemptyResumed == 0 {
		t.Fatal("no nonempty Projectile source population compared")
	}
}
