package game

import (
	"testing"

	"againrom/internal/firstgame"
)

func TestReleaseMapEditorFieldChoicesKeepTheirLists(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	maps, asked := 0, 0
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		s := e.Document().Edits.(*mapInspectionEdits)
		asked += checkFieldChoices(t, key, s, e.Document(), 2)
		maps++
	}
	if maps == 0 || asked == 0 {
		t.Fatalf("compared %d fields on %d maps", asked, maps)
	}
	t.Logf("compared %d fields on %d maps", asked, maps)
}
