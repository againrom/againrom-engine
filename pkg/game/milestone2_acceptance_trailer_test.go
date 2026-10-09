//go:build sessioncorpusaudit

package game

import "testing"

func TestMilestone2Trailer(t *testing.T) {
	files, nonzeroFiles, differences, cities := 0, 0, 0, 0
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			cities++ // This milestone measures mission Document state only.
			return
		}
		want, err := trailerAcceptanceRaw(mf.body, mf.f.TrailerOff)
		if err != nil {
			t.Fatal(err)
		}
		ms, _, err := loadOriginalMission(fe, mf.raw)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		files++
		if want != ([100]uint32{}) {
			nonzeroFiles++
			t.Log("trailer: nonzero source", mf.rel)
		}
		for _, diff := range trailerAcceptanceDifferences(want, ms.savedDocument) {
			differences++
			t.Error(diff)
		}
	})
	t.Logf("trailer: %d resumed world files, %d raw dwords, %d nonzero source files; %d city files outside mission scope; %d refused", files, 100*files, nonzeroFiles, cities, len(refused))
	milestone2LogRefusals(t, "trailer", refused)
	t.Logf("trailer: %d retained Document differences; no live World consumer asserted", differences)
	if files == 0 {
		t.Fatal("no mission trailer population compared")
	}
}
