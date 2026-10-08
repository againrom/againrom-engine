//go:build sessioncorpusaudit

package game

import "testing"

func TestMilestone2Sacks(t *testing.T) {
	files, roots, documentRecords, liveRecords, retiredRecords, mismatches, gaps, weightDifferences := 0, 0, 0, 0, 0, 0, 0, 0
	classes := map[string]int{"Sack": 0, "Item": 0, "Weapon": 0, "Armor": 0, "Shield": 0, "Effect": 0, "Spell": 0}
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		source, err := sackByteWalk(mf.f)
		if err != nil {
			t.Fatalf("Sack independent source walk refused: %v", err)
		}
		roots += int(source.count)
		for _, index := range source.indices() {
			r := source.rows[index]
			classes[r.class]++
			if r.class == "Sack" {
				var sum int32
				for _, ref := range r.refs["Contents"] {
					if item := source.rows[ref]; item != nil {
						sum += int32(int16(item.values["F4A"])) * int32(item.values["F42"])
					}
				}
				if sum != int32(r.values["Contents20"]) {
					weightDifferences++
					t.Logf("sacks: source archive %d stored load %d differs from diagnostic item sum %d; preserved verbatim", index, int32(r.values["Contents20"]), sum)
				}
			}
		}
		ms, _, err := ResumeOriginalSave(fe.Archives.Containers, mf.raw, fe.Table, fe.Difficulty, nil, fe.Bodies)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		files++
		if ms.savedDocument == nil {
			t.Fatal("complete retained Document absent")
		}
		origins, err := sackCurrentOrigins(source, ms.savedDocument.Document)
		if err != nil {
			t.Fatalf("Sack raw-to-current identity correspondence refused: %v", err)
		}
		for _, d := range sackDocumentDifferences(source, origins, ms.savedDocument.Document) {
			mismatches++
			t.Error(d)
		}
		documentRecords += len(source.rows)
		differences, unavailable := sackLiveDifferences(source, origins, ms.savedDocument.Objects, ms.World.SavedObjects(), ms.World.Sacks())
		for _, d := range differences {
			mismatches++
			t.Error(d)
		}
		for _, gap := range unavailable {
			gaps++
			t.Log("sacks: consumer gap:", gap)
		}
		live, retired := sackLifecycleCounts(ms.World.SavedObjects())
		liveRecords += live
		retiredRecords += retired
	})
	milestone2LogRefusals(t, "sacks", refused)
	t.Logf("sacks: %d world files resumed, %d raw root slots, %d source records checked in Document, %d live Sacks, %d retired Sack rows, %d mismatches, %d consumer gaps, %d refused, %d diagnostic weight differences", files, roots, documentRecords, liveRecords, retiredRecords, mismatches, gaps, len(refused), weightDifferences)
	t.Logf("sacks: raw class population %v; zero means synthetic-only coverage; installed Definition binding and original-runtime consumers of opaque Token/scalar fields are outside this source-value audit", classes)
	if files == 0 || roots == 0 {
		t.Fatal("no source Sack population compared")
	}
}
