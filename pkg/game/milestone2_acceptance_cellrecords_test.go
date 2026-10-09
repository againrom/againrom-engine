//go:build sessioncorpusaudit

package game

import (
	"testing"
)

// The raw count includes every overwritten row; comparison uses each key's
// final SAV-CELLLOAD-109 overwrite. Neither live carrier may silently omit,
// duplicate or introduce a saved key. This compares baselines and layer
// counts; object-reference and trigger consumers have separate instruments.
func TestMilestone2CellRecords(t *testing.T) {
	filesChecked, recordsChecked, uniqueChecked, differences := 0, 0, 0, 0
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		_, wants, err := terrainCellsRaw(mf.body, mf.f.World.BlocksOff)
		if err != nil {
			t.Fatal(err)
		}
		ms, _, err := loadOriginalMission(fe, mf.raw)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		filesChecked++
		recordsChecked += len(wants)
		uniqueChecked += len(cellAcceptanceFinal(wants))
		_, baselines, present := ms.World.SavedStructures()
		for _, diff := range cellAcceptanceDifferences(wants, baselines, ms.World.SavedCellRecords(), present) {
			differences++
			t.Error(diff)
		}
	})
	t.Logf("cell records: %d world-half file(s), %d raw record(s), %d final keys, %d duplicate overwrite(s); %d refused", filesChecked, recordsChecked, uniqueChecked, recordsChecked-uniqueChecked, len(refused))
	milestone2LogRefusals(t, "cell records", refused)
	t.Logf("cell records: %d field or population differences", differences)
	if filesChecked == 0 || recordsChecked == 0 {
		t.Fatal("no cell population compared")
	}
}
