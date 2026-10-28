package game

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// diariesDroppedAtLoad names the Diary owners a LOAD drops as marked absent
// (DIV-2885), running the production decision on a scratch World that carries
// a Diary for every owner the document's current actions bind.
func diariesDroppedAtLoad(doc *sav.DocumentData) (map[sim.SavedDiaryOwner]bool, error) {
	if doc == nil || doc.World == nil {
		return nil, fmt.Errorf("diaries: the document has no world")
	}
	actions, err := readCurrentActions(doc)
	if err != nil || actions == nil {
		return nil, err
	}
	var w sim.World
	records, err := currentDiaryRecords(doc, actions, &w)
	if err != nil {
		return nil, err
	}
	var diaries []sim.SavedDiary
	for owner := range records {
		diaries = append(diaries, sim.SavedDiary{Owner: owner})
	}
	slices.SortFunc(diaries, func(a, b sim.SavedDiary) int {
		if a.Owner.Player != b.Owner.Player {
			if a.Owner.Player {
				return -1
			}
			return 1
		}
		return int(a.Owner.Actor) - int(b.Owner.Actor)
	})
	if err := w.ImportOriginalDiaries(diaries); err != nil {
		return nil, err
	}
	if err := restoreCurrentCarrierAbsence(&w, doc, actions); err != nil {
		return nil, err
	}
	kept := map[sim.SavedDiaryOwner]bool{}
	for _, d := range w.SavedDiaries() {
		kept[d.Owner] = true
	}
	dropped := map[sim.SavedDiaryOwner]bool{}
	for owner := range records {
		if !kept[owner] {
			dropped[owner] = true
		}
	}
	return dropped, nil
}

func TestDiariesDroppedAtLoadFollowTheAbsencePolicy(t *testing.T) {
	f, s := currentCarrierFixture(t)
	raw, err := f.ExportCurrentSave(s, "diary absence")
	if err != nil {
		t.Fatal(err)
	}
	decode := func() *sav.DocumentData {
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		return &doc
	}
	doc := decode()
	a, err := readCurrentActions(doc)
	if err != nil || a == nil || len(a.AbsentDiaries) == 0 {
		t.Fatal("fixture no longer marks its Diaries absent", err)
	}
	records, err := currentDiaryRecords(doc, a, &sim.World{})
	if err != nil {
		t.Fatal(err)
	}
	all := map[sim.SavedDiaryOwner]bool{}
	for owner := range records {
		all[owner] = true
	}
	dropped, err := diariesDroppedAtLoad(doc)
	if err != nil || !all[sim.SavedDiaryOwner{Player: true}] || !reflect.DeepEqual(dropped, all) {
		t.Fatalf("dropped %v, %v; want every bound owner %v, the Player's among them", dropped, err, all)
	}
	// The production LOAD of the same file carries none of them.
	if live := openCurrentCarrierSave(t, raw).live.world.SavedDiaries(); len(live) != 0 {
		t.Fatalf("LOAD carried %+v; the helper said every Diary is dropped", live)
	}

	// Loss control: a Diary edited after its anchor is kept.
	doc = decode()
	edited := records[a.AbsentDiaries[0].Owner].Object
	diary := &doc.Objects[edited-1]
	if inline := records[a.AbsentDiaries[0].Owner].Inline; a.AbsentDiaries[0].Owner.Player {
		diary = &doc.Objects[edited-1].Inline[inline].Record
	}
	mustSetValue(diary, "D2C", 0x12345678)
	dropped, err = diariesDroppedAtLoad(doc)
	if err != nil || dropped[a.AbsentDiaries[0].Owner] || len(dropped) != len(all)-1 {
		t.Fatalf("an edited Diary: dropped %v, %v; want %v kept and the rest dropped", dropped, err, a.AbsentDiaries[0].Owner)
	}

	// A document without current actions drops nothing.
	doc = decode()
	doc.State = sav.DocumentStateData{}
	if dropped, err := diariesDroppedAtLoad(doc); err != nil || len(dropped) != 0 {
		t.Fatalf("no current actions: dropped %v, %v; want none", dropped, err)
	}
}
