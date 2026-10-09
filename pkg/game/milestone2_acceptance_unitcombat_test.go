//go:build sessioncorpusaudit

package game

import "testing"

func TestMilestone2UnitCombat(t *testing.T) {
	worlds, cities, rawRecords, worldRecords, cityRecords, live, rawOnly, bytesRead, mismatches := 0, 0, 0, 0, 0, 0, 0, 0, 0
	classes := map[string]int{"Unit": 0, "Human": 0, "Humanoid": 0}
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		want, err := readUnitCombatExpected(mf.f, mf.raw)
		if err != nil {
			mismatches++
			t.Errorf("independent Unit combat read: %v", err)
			return
		}
		// Count the complete raw population BEFORE any resume/projection gate.
		rawRecords += len(want.records)
		for _, r := range want.records {
			classes[r.class]++
			bytesRead += 134
			if r.class != "Unit" {
				bytesRead += 24
			}
		}
		if !mf.present {
			cities++
			cityRecords += len(want.records)
			state, _ := decodeSavedDocument(mf.raw)
			for _, difference := range want.documentDifferences(state) {
				mismatches++
				t.Error("city decode-only Document: " + difference)
			}
			for _, r := range want.records {
				t.Logf("unit combat: archive %d %s offset %d: city decode-only, complete Document compared; no live combat basis", r.archive, r.class, r.off)
			}
			return
		}
		worldRecords += len(want.records)
		ms, _, err := loadOriginalMission(fe, mf.raw)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		worlds++
		// Complete imported Document is checked before Snapshot can project
		// the current World fields back over a decoder/retention disagreement.
		for _, difference := range want.documentDifferences(ms.savedDocument) {
			mismatches++
			t.Error("retained Document before Snapshot: " + difference)
		}
		differences, excluded, n := want.entityDifferences(ms.World.Entities(), ms.savedDocument, ms)
		live += n
		rawOnly += len(excluded)
		for _, difference := range differences {
			mismatches++
			t.Error("live: " + difference)
		}
		for _, reason := range excluded {
			t.Log("unit combat: excluded from live basis: " + reason)
		}
	})
	milestone2LogRefusals(t, "unit combat", refused)
	t.Logf("unit combat: %d world files resumed, %d city files decode-only; %d raw tagged actors (%d world, %d city), classes %v; %d selected wire bytes; %d live combat bases, %d world raw-only records; %d mismatches, %d refused", worlds, cities, rawRecords, worldRecords, cityRecords, classes, bytesRead, live, rawOnly, mismatches, len(refused))
	t.Log("unit combat: UA6 24, UBE 22, U114 24, UD4 64 and Human/Humanoid H1CC 24 raw bytes; whole source blocks and existing live counterparts compared separately; no inferred modifier/derive/callback semantics; city App and exact Humanoid runtime acceptance remain outside this witness")
	if worlds == 0 || live == 0 || rawRecords == 0 {
		t.Fatal("no Unit combat acceptance population compared")
	}
}
