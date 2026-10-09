//go:build sessioncorpusaudit

package game

import "testing"

func TestMilestone2Buildings(t *testing.T) {
	files, records, cells, nonzero, normalized, mismatches := 0, 0, 0, 0, 0, 0
	classes := map[string]int{"Building": 0, "Outpost": 0, "Tavern": 0, "Shop": 0}
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		want, links, err := buildings1145Expected(mf.f)
		if err != nil {
			t.Fatal(err)
		}
		ms, _, err := loadOriginalMission(fe, mf.raw)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		files++
		records += len(want)
		cells += len(links)
		for _, key := range links {
			if key != 0 {
				nonzero++
			}
		}
		for _, w := range want {
			classes[[]string{"", "Building", "Outpost", "Tavern", "Shop"}[w.source.Class]]++
			if w.normalized {
				normalized++
			}
		}
		sources, liveCells, present := ms.World.SavedStructures()
		for _, difference := range buildings1145Differences(want, links, ms.World.Structures(), sources, liveCells, present) {
			mismatches++
			t.Error(difference)
		}
	})
	milestone2LogRefusals(t, "buildings", refused)
	t.Logf("buildings: %d world files resumed, %d records, %d cell links (%d nonzero), %d normalized raw/scalar overlaps, %d mismatches, %d refused", files, records, cells, nonzero, normalized, mismatches, len(refused))
	t.Logf("buildings: class population %v; zero means no original-corpus witness for that subclass", classes)
	if files == 0 || records == 0 {
		t.Fatal("no Building population compared")
	}
}
