package game

import (
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

// The installed maps the drop-cell rule warns on, by catalogue key.
func TestReleaseDropCellRuleWarnsOnTheNamedInstalledMaps(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	var got []string
	keys := installedMapKeys(x)
	var want []string
	switch len(keys) {
	case 38: // the EN install
		want = []string{"scenario/50.alm"}
	case 34: // the RU install
		want = []string{"loose/Horror.alm", "scenario/50.alm"}
	default:
		t.Fatalf("%d installed maps: neither the EN install (38) nor the RU install (34)", len(keys))
	}
	for _, key := range keys {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range d.Edits.(ui.InspectionChecker).Report() {
			if f.Rule == "drop-cell" {
				got = append(got, key)
			}
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("drop-cell warns on %q, want %q", got, want)
	}
}
