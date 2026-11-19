//go:build release

package game

import (
	"testing"

	"againrom/pkg/base"
)

// The open install's tables are its game's: the row count of each is logged
// for the root under test.
func TestReleaseEditorTablesFollowTheInstall(t *testing.T) {
	x := installedInspector(t)
	got := x.editorTables()
	t.Logf("game %v: objectKinds %d, handoutBuildings %d, heroChoices %d, itemSuits %d, addKinds %d",
		x.installGame(), len(got.objectKinds), len(got.handoutBuildings), len(got.heroChoices), len(got.itemSuits), len(got.addKinds))
	if x.installGame() == base.GameROM2 {
		if len(got.objectKinds)+len(got.handoutBuildings)+len(got.heroChoices)+len(got.itemSuits) != 0 || len(got.addKinds) != 3 {
			t.Errorf("second-game install tables %+v", got)
		}
		if f := x.heroField("class", ""); !f.ReadOnly || len(f.Choices) != 0 || f.TextValue != x.word("editor.tables.none") {
			t.Errorf("an empty hero table shows %+v", f)
		}
		return
	}
	if len(got.objectKinds) != 1 || len(got.handoutBuildings) != 3 || len(got.heroChoices) != 2 || len(got.itemSuits) != 3 || len(got.addKinds) != 9 {
		t.Errorf("first-game install tables %+v", got)
	}
}
