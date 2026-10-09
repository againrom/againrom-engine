//go:build sessioncorpusaudit

package game

import "testing"

func TestMilestone2MoverRoutes(t *testing.T) {
	worlds, cities, records, worldRecords, cityRecords, compared, excluded, current, active, handedOff, dying, mismatches := 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	classes := map[string]int{"Unit": 0, "Human": 0, "Humanoid": 0}
	unsupported := 0
	var nonempty, elements [3]int
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		want, err := mover1160Expected(mf.f, mf.raw)
		if err != nil {
			mismatches++
			t.Errorf("independent mover read: %v", err)
			return
		}
		records += len(want.records)
		for _, r := range want.records {
			classes[r.class]++
			for i, list := range r.routes {
				elements[i] += len(list)
				if len(list) > 0 {
					nonempty[i]++
				}
			}
		}
		// Decode-only includes every source, even a later resume refusal.
		decoded, _ := decodeSavedDocument(mf.raw)
		for _, diff := range want.documentDifferences(decoded) {
			mismatches++
			t.Error("decoded complete Document: " + diff)
		}
		if !mf.present {
			cities++
			cityRecords += len(want.records)
			t.Logf("mover routes: %d city actors, Document only; no live movement", len(want.records))
			return
		}
		worldRecords += len(want.records)
		ms, _, err := loadOriginalMission(fe, mf.raw)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		worlds++
		for _, diff := range want.documentDifferences(ms.savedDocument) {
			mismatches++
			t.Error("imported complete Document BEFORE Snapshot: " + diff)
		}
		diffs, exclusions, p := want.worldDifferences(ms.World, ms.savedDocument)
		compared += p.compared
		current += p.current
		active += p.active
		handedOff += p.handedOff
		dying += p.dying
		excluded += len(exclusions)
		for _, diff := range diffs {
			mismatches++
			t.Error("initial World: " + diff)
		}
		for _, reason := range exclusions {
			t.Log("mover routes: " + reason)
		}
		motions, _, _, _ := ms.World.SavedActorMotions()
		for _, m := range motions {
			if m.Issue != "" {
				unsupported++
				t.Logf("mover routes: entity %d unsupported continuation: %s; raw carrier compared", m.Entity, m.Issue)
			}
		}
	})
	milestone2LogRefusals(t, "mover routes", refused)
	t.Logf("mover routes: %d world files resumed, %d city files decode-only; %d raw tagged actors (%d world, %d city), classes %v; %d live motion/order carriers (%d current, %d active, %d handed to native route, %d dying), %d world raw-only; %d mismatches, %d refused", worlds, cities, records, worldRecords, cityRecords, classes, compared, current, active, handedOff, dying, excluded, mismatches, len(refused))
	t.Logf("mover routes: static/dynamic/order nonempty=%v elements=%v; Position12/Mover180/Order148 plus independent counts/elements compared in complete Document; live Order owns144 bytes, final4 retained as pointer residue; no invented current producer or full runtime fidelity", nonempty, elements)
	t.Logf("mover routes: %d unsupported continuation issues named without excluding their raw carriers", unsupported)
	if worlds == 0 || records == 0 || compared == 0 {
		t.Fatal("no mover acceptance population")
	}
}
