package game

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// In the town shop a staff whose Spell comes from its kind-41 effect owns
// that Spell only while worn. Taking it off drops the Spell child; putting it
// back mints a new one. Before this fix the shop's transaction check treated
// that child as an Item the operation forgot to mint and rolled the equip
// back, so the doll stayed empty. The first 27 steps of the doll-and-shop
// scenario take the staff off, put it back and take it off again.
func TestReleaseShopStaffTakeOffAndPutBack(t *testing.T) {
	source := os.Getenv("AGAINROM_SAVE_666")
	if source == "" {
		t.Skip("AGAINROM_SAVE_666 is not set")
	}
	f := releaseFront(t)
	scenario, err := ReadHeadlessScenario(filepath.Join("..", "..", "scenarios", "1005-doll-and-shop.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(scenario.Steps) < 27 || scenario.Steps[22].Command != "assert_shop" || scenario.Steps[26].Command != "assert_shop" {
		t.Fatal("doll-and-shop scenario no longer puts the staff back at step 23")
	}
	scenario.Steps = scenario.Steps[:27]
	scenario.OriginalSaves, scenario.Saves = filepath.Dir(source), t.TempDir()
	f.SetDeterministicFrames(true)
	app := f.App("shop staff take off and put back")
	app.SetSaveSeams(f.SaveSeams(SaveStore{Dir: scenario.Saves}, OriginalStore{Dir: scenario.OriginalSaves}, nil))
	if err := RunHeadlessScenario(f, app, scenario, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
}
