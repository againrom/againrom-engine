package game

import (
	"testing"
	"time"

	"againrom/pkg/ui"
)

// Historical native-format continuation instruments compare the entire World
// byte form, including source keys which a fresh SAV must remint. Select their
// AGS codec explicitly now that an ordinary admitted world SAVE produces SAV.
// This keeps the native proof; it is not an ordinary SAVE-format witness.
func nativeCheckpoint1170(f *FrontEnd, store SaveStore, clock ...func() time.Time) ui.SaveGame {
	now := time.Now
	if len(clock) != 0 && clock[0] != nil {
		now = clock[0]
	}
	return func(onMap bool) (string, error) {
		s, label, err := f.Snapshot(onMap)
		if err != nil {
			return "", err
		}
		raw, err := EncodeSave(s, label)
		if err != nil {
			return "", err
		}
		return store.Write(now(), raw)
	}
}

func nativeContinuationSeams1170(t *testing.T, f *FrontEnd, store SaveStore, orig OriginalStore, now func() time.Time) (ui.SaveGame, ui.SaveList, ui.LoadGame) {
	t.Helper()
	_, list, load := agsSaveSeams(f, store, orig, now)
	t.Log("native continuation instrument selects the AGS codec explicitly")
	return nativeCheckpoint1170(f, store, now), list, load
}
