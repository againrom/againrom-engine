package game

import (
	"fmt"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
)

// tokenReferenceCensus counts, per class, the non-zero Token References of
// Item-class and Effect records that name a key of a record at or before
// them in archive order (kept) and those that do not (missed).
func tokenReferenceCensus(d *sav.DocumentData) (kept, missed map[string]int) {
	kept, missed = map[string]int{}, map[string]int{}
	keys := map[uint32]bool{}
	for i := range d.Objects {
		r := &d.Objects[i]
		for _, v := range r.Values {
			if (v.Name == "Identity" || v.Name == "This") && v.Value != 0 {
				keys[v.Value] = true
			}
		}
		if !tokenReferenceClass(r.Class) {
			continue
		}
		if v, err := savedStructureValue(r, "Reference"); err == nil && v != 0 {
			if keys[v] {
				kept[r.Class]++
			} else {
				missed[r.Class]++
			}
		}
	}
	return kept, missed
}

// humanLoadWords is each Human's U90 by Identity, and whether it is a dead
// record.
func humanLoadWords(t *testing.T, d *sav.DocumentData) (map[uint32]uint32, map[uint32]bool) {
	t.Helper()
	dead := map[uint16]bool{}
	for _, root := range d.DeadActors {
		dead[root] = true
	}
	words, deadKeys := map[uint32]uint32{}, map[uint32]bool{}
	for i := range d.Objects {
		r := &d.Objects[i]
		if r.Class != "Human" {
			continue
		}
		key, err1 := savedStructureValue(r, "Identity")
		load, err2 := savedStructureValue(r, "U90")
		if err1 != nil || err2 != nil {
			t.Fatalf("Human #%d without Identity or U90", i+1)
		}
		words[key], deadKeys[key] = load, dead[uint16(i+1)]
	}
	return words, deadKeys
}

// After LOAD of a first-game SAV the original wrote and an immediate SAVE, a
// Token Reference that names a key read earlier in the archive keeps its
// value, one that names no such key is written as 0 (SAV-1376), and every
// Human, a dead record included, keeps its loaded U90 (SAV-1378). The two
// files with Reference misses are the corpus's only ones (SAV-1377).
func TestReleaseOriginalResaveReferenceAndHumanLoad(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, c := range []struct {
		name, hash string
		missed     int
	}{
		{"2026-09-23/exp0397-r2/game0002.sav", "4787b1a38f7680249462ae6908ff84f805eb5b44fb96a26b6d282c23e376e939", 5},
		{"2026-09-24/exp-engine-lineage/game0000-original-resave9250.sav", "a1bc3c65ba5b384dc0802449cba6d45f5a167eecd0ed3a1e707e34a9e0da071d", 1},
		{"2026-08-02/game0007.sav", "a7cb35ea5d87c9c7a9b8cfd70f8a1f61b6927ca089e468a9d6fe983da777156e", 0},
		{"2026-09-24/game0017-victory.sav", "a30de01ec3d96d58a3a244b9a536a3e49c04a4d45098cb828dabb908756488c3", 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, payload := groundCorpusFile(t, c.name, c.hash)
			loaded, err := sav.DecodeDocumentData(payload)
			if err != nil {
				t.Fatal(err)
			}
			f := releaseFront(t)
			a := f.App("resave fields " + c.name)
			a.Layout(1024, 768)
			open, _, err := f.RestoreOriginal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if err := a.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			s, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			written, _, err := f.playerMissionSave(s, label)
			if err != nil {
				t.Fatal(err)
			}
			out, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			keptBefore, missedBefore := tokenReferenceCensus(&loaded)
			keptAfter, missedAfter := tokenReferenceCensus(&out)
			missed := 0
			for _, n := range missedBefore {
				missed += n
			}
			if missed != c.missed {
				t.Fatalf("loaded file has %d Reference misses %v, want %d", missed, missedBefore, c.missed)
			}
			if len(missedAfter) != 0 {
				t.Errorf("SAVE wrote Reference misses %v, want none", missedAfter)
			}
			if fmt.Sprint(keptAfter) != fmt.Sprint(keptBefore) {
				t.Errorf("SAVE kept References %v, loaded %v", keptAfter, keptBefore)
			}
			before, dead := humanLoadWords(t, &loaded)
			after, _ := humanLoadWords(t, &out)
			// A dead Human the World holds as an entity keeps its loaded load
			// only when its record is in no Group (bodyWords); a moved load is
			// counted, not required.
			held := map[uint32]bool{}
			for _, d := range f.live.world.OriginalDeadActors() {
				if _, ok := f.live.world.Entity(d.ID); ok {
					held[d.Source.Identity] = true
				}
			}
			paired, absent, heldMoved := 0, 0, 0
			for key, want := range before {
				got, ok := after[key]
				if !ok {
					continue
				}
				paired++
				if dead[key] && held[key] {
					if got != want {
						heldMoved++
					}
					continue
				}
				if dead[key] {
					absent++
				}
				if got != want {
					t.Errorf("Human %d (dead %v) U90 after SAVE %d, loaded %d", key, dead[key], int16(got), int16(want))
				}
			}
			if paired == 0 {
				t.Fatal("no loaded Human is in the SAVE")
			}
			t.Logf("References kept %v, loaded misses %v; %d of %d Humans paired by Identity, %d dead with no entity; %d dead entity bodies write another U90",
				keptAfter, missedBefore, paired, len(before), absent, heldMoved)
		})
	}
}
