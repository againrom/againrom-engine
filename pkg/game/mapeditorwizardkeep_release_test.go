package game

import (
	"bytes"
	"testing"

	"againrom/internal/firstgame"
)

// TestReleaseMapEditorWizardApplyWithoutEditKeepsEveryTrigger opens every
// trigger of every installed map, presses Apply without an edit and finds the
// map's bytes as they were.
func TestReleaseMapEditorWizardApplyWithoutEditKeepsEveryTrigger(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	total, changed := 0, 0
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		before := append([]byte(nil), e.Document().SourceBytes...)
		sc := wizScript(t, before)
		for i := range sc.Triggers {
			e.Select(trigRecord(t, e, "Trigger", i), true)
			wizApply(t, e)
			total++
			if !bytes.Equal(before, e.Document().SourceBytes) {
				changed++
				t.Errorf("%s trigger %d %q: Apply without an edit changed the map", key, i, sc.Triggers[i].Name)
				if err := e.Key("undo"); err != nil || !bytes.Equal(before, e.Document().SourceBytes) {
					t.Fatalf("%s trigger %d: Undo did not restore the map: %v", key, i, err)
				}
			}
		}
	}
	t.Logf("%d triggers, %d changed", total, changed)
}
