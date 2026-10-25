package game

import (
	"reflect"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

func inventoryPairIdle(t *testing.T, app *ui.App, frames int) {
	t.Helper()
	for range frames {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
}

func inventoryPairPointer(t *testing.T, app *ui.App, x, y int, edges ...string) {
	t.Helper()
	for _, edge := range edges {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReleaseMissionPackAndWornBoxDoubleClicksSaveAndColdLoad(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, app := shieldWitnessOpen(t, shieldWitnessCases[0])
	id := shieldWitnessSelect(t, f, app)
	start := shieldWitnessRead(f.live.world, id)
	shieldWitnessEquip(t, f, app, id)
	worn := shieldWitnessRead(f.live.world, id)
	shieldWitnessRequire(t, "pack double click", shieldWitnessCases[0], start, worn)
	store, name := shieldWitnessSave(t, f, app)
	loaded, loadedApp := loadSAVWindow(t, store, name)
	loadedID := shieldWitnessSelect(t, loaded, loadedApp)
	if got := shieldWitnessRead(loaded.live.world, loadedID); !reflect.DeepEqual(got, worn) {
		t.Fatalf("cold LOAD after pack double click: %+v, want %+v", got, worn)
	}
	if _, _, err := loadedApp.HeadlessWornCellPoint(1); err == nil {
		t.Fatal("shipped frame unexpectedly exposes the worn box")
	}
	if err := loadedApp.HeadlessWornBoxFixture(); err != nil {
		t.Fatal(err)
	}
	x, y, err := loadedApp.HeadlessWornCellPoint(1)
	if err != nil {
		t.Fatal(err)
	}
	inventoryPairPointer(t, loadedApp, x, y, "press", "release")
	inventoryPairIdle(t, loadedApp, 32)
	inventoryPairPointer(t, loadedApp, x, y, "press", "release")
	inventoryPairIdle(t, loadedApp, 4)
	if got := shieldWitnessRead(loaded.live.world, loadedID); !reflect.DeepEqual(got, worn) {
		t.Fatalf("worn presses outside the window changed equipment: %+v, want %+v", got, worn)
	}
	inventoryPairIdle(t, loadedApp, 32)
	inventoryPairPointer(t, loadedApp, x, y, "press", "release", "press", "release")
	inventoryPairIdle(t, loadedApp, 4)
	if got := shieldWitnessRead(loaded.live.world, loadedID); !reflect.DeepEqual(got, start) {
		t.Fatalf("worn double click: %+v, want original inventory %+v", got, start)
	}
	store, name = shieldWitnessSave(t, loaded, loadedApp)
	cold, _ := loadSAVWindow(t, store, name)
	if got := shieldWitnessRead(cold.live.world, cold.live.mission.ids[0]); !reflect.DeepEqual(got, start) {
		t.Fatalf("cold LOAD after worn double click: %+v, want %+v", got, start)
	}
}
