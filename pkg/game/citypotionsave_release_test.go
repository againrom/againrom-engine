package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func cityPotionLoadApp(t *testing.T, raw []byte) (*FrontEnd, *ui.App, SaveStore) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "potion.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app := openLocalTownSAV(t, f, store.Dir, "potion.sav")
	t.Cleanup(app.StopAudio)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	return f, app, store
}

func cityPotionPointer(t *testing.T, app *ui.App, target string, index int, edges ...string) {
	t.Helper()
	x, y, err := app.HeadlessShopPoint(target, index)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range edges {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func cityPotionMissionEqual(t *testing.T, live, cold *sim.World, tick int) {
	t.Helper()
	if live.Hash() == cold.Hash() {
		return
	}
	wantEntities, gotEntities := live.Entities(), cold.Entities()
	if len(wantEntities) == len(gotEntities) {
		for i, want := range wantEntities {
			wantValue, gotValue := reflect.ValueOf(want), reflect.ValueOf(gotEntities[i])
			for field := 0; field < wantValue.NumField(); field++ {
				if wantValue.Field(field).CanInterface() && !reflect.DeepEqual(wantValue.Field(field).Interface(), gotValue.Field(field).Interface()) {
					t.Logf("actor %d %s: live=%+v cold=%+v", want.ID, wantValue.Type().Field(field).Name, wantValue.Field(field).Interface(), gotValue.Field(field).Interface())
				}
			}
		}
	}
	t.Logf("live effects=%+v cold=%+v", live.ActiveEffects(), cold.ActiveEffects())
	for _, field := range []struct {
		name       string
		live, cold any
	}{
		{"stock", live.Stock(), cold.Stock()},
		{"actions", live.Actions(), cold.Actions()},
		{"policy", live.CurrentPolicy(), cold.CurrentPolicy()},
		{"saved objects", live.SavedObjects(), cold.SavedObjects()},
		{"sacks", live.Sacks(), cold.Sacks()},
		{"item weights", live.ItemWeights(), cold.ItemWeights()},
		{"script", live.Script(), cold.Script()},
		{"cells", live.SavedCellRecords(), cold.SavedCellRecords()},
	} {
		if !reflect.DeepEqual(field.live, field.cold) {
			a, b := fmt.Sprintf("%+v", field.live), fmt.Sprintf("%+v", field.cold)
			at := 0
			for at < len(a) && at < len(b) && a[at] == b[at] {
				at++
			}
			t.Logf("%s differ at %d (lengths %d/%d): live=%s cold=%s", field.name, at, len(a), len(b), a[max(0, at-80):min(len(a), at+400)], b[max(0, at-80):min(len(b), at+400)])
		}
	}
	a, aerr := live.MarshalBinary()
	b, berr := cold.MarshalBinary()
	if aerr != nil || berr != nil {
		t.Fatal("mismatched World encoding", aerr, berr)
	}
	at := 0
	for at < len(a) && at < len(b) && a[at] == b[at] {
		at++
	}
	t.Logf("world bytes differ at %d (lengths %d/%d): live=%x cold=%x", at, len(a), len(b), a[max(0, at-16):min(len(a), at+32)], b[max(0, at-16):min(len(b), at+32)])
	t.Fatalf("uninterrupted and cold next mission differ at tick %d", tick)
}
