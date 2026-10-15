package game

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// nativeCheckpoint1170 is the production SAVE seam on its own: the current
// state through the SAV producer into the store. It returns the load-window
// token of the file it wrote, so the same name opens it through SaveSeams.
func nativeCheckpoint1170(f *FrontEnd, store SaveStore, clock ...func() time.Time) ui.SaveGame {
	var now func() time.Time
	if len(clock) != 0 {
		now = clock[0]
	}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, now)
	return localTokenSave(save)
}

// localTokenSave reports a written file under its load-window token.
func localTokenSave(save ui.SaveGame) ui.SaveGame {
	return func(onMap bool) (string, error) {
		name, err := save(onMap)
		if err != nil {
			return "", err
		}
		return localOriginalSaveToken(name), nil
	}
}

// nativeContinuationSeams1170 is the production save menu over store and orig;
// its SAVE reports the load-window token of the file it wrote.
func nativeContinuationSeams1170(t *testing.T, f *FrontEnd, store SaveStore, orig OriginalStore, now func() time.Time) (ui.SaveGame, ui.SaveList, ui.LoadGame) {
	t.Helper()
	save, list, load := f.SaveSeams(store, orig, now)
	return localTokenSave(save), list, load
}

// localSAVRows lists the SAV files in a store directory, newest first, under
// the names the load window passes back to SaveSeams.
func localSAVRows(s SaveStore) ([]SaveEntry, error) {
	rows, err := s.List()
	for i := range rows {
		rows[i].Name = localOriginalSaveToken(rows[i].Name)
	}
	return rows, err
}

// sameResave reports whether a SAVE of b writes the same original document as
// a SAVE of a, the front b was loaded from. The retained Document of a SAV
// LOAD holds what the producer wrote, not the source file's residue, so the
// next SAVE is the oracle. The engine's action supplement is left out: a SAV
// LOAD marks its roster spellbooks restored and fills roster weapon names.
func sameResave(t *testing.T, a, b *FrontEnd) bool {
	t.Helper()
	var docs [2]sav.DocumentData
	for i, f := range []*FrontEnd{a, b} {
		s, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal("resave Snapshot", i, err)
		}
		raw, err := f.ExportCurrentSave(s, "resave")
		if err != nil {
			t.Fatal("resave", i, err)
		}
		if docs[i], err = sav.DecodeDocumentData(raw); err != nil {
			t.Fatal("resave decode", i, err)
		}
		docs[i].State.ValueRecords = slices.DeleteFunc(docs[i].State.ValueRecords, func(r sav.CityStateRecordData) bool {
			return r.Path == "/CurrentState/AgainromActions"
		})
	}
	return reflect.DeepEqual(docs[0], docs[1])
}

// arriveInTownGrantTaken arrives with the chapter's companion grant taken: a
// SAV LOAD hands a pending grant out (SAV-CAMPAIGN-084).
func arriveInTownGrantTaken(f *FrontEnd) {
	f.arriveInTown()
	f.Town.takeAddHeroes(f.Town.Chapter())
}
