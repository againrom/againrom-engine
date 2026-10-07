package game

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestNativeSaveFreshLoadRestoresScheduledLight(t *testing.T) {
	front := func(t *testing.T) *FrontEnd { return currentPoolFixtureFront(t, 91, 92) }
	nativeLightSaveFresh(t, front(t), front)
}

// This uses the installed mission and renderer with a controlled native clock,
// not a naturally played mission or an original SAV. The observation is the
// renderer's Light cache; no screen pixels or ROM1 load policy are asserted.
func TestReleaseNativeSaveFreshLoadRestoresScheduledLight(t *testing.T) {
	nativeLightSaveFresh(t, releaseFront(t), releaseFront)
}

func nativeLightSaveFresh(t *testing.T, f *FrontEnd, front func(*testing.T) *FrontEnd) {
	t.Helper()
	app := f.App("native-light-save")
	store := SaveStore{Dir: t.TempDir()}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}

	// Construct a non-relight clock through the documented native World header
	// (tick at bytes 1..8), retaining this mission's other canonical fields.
	// This fixture does not spend 9343 busy-map ticks or claim their gameplay.
	raw, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint64(raw[1:9], 9343)
	if err := f.live.world.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(f.live.world)
	// Warm standard play last relit at 9280, then held that cache through 9343.
	// Neither the forced clock diagnostic nor the new restore seam builds the
	// expectation. The independent bracket excludes seed 0.7854 and a forced
	// current-minute 583 result 0.4865; the positive sign fixes its direction.
	view := f.live.view
	if !view.TimeFlow() {
		t.Fatal("fixture must use the standard enabled day/night cycle")
	}
	view.SetLightClock(9280)
	f.live.push()
	warm := view.Sun()
	if !(warm.Theta > 0.4799 && warm.Theta < 0.4801) ||
		warm.Ambient != 14 || warm.Range != 32 || warm.SkyTint != [3]uint8{} ||
		warm.ShroudObject != 4 || warm.ShroudUnit != 2 {
		t.Fatalf("warm tick9343 cache = %+v, want positive minute580 daylight", warm)
	}
	hash := f.live.world.Hash()
	before, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(store.Dir, "Scheduled light", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	var saved SaveEntry
	for _, entry := range entries {
		if entry.Name == "Scheduled light.sav" {
			saved = entry
		}
	}
	if err != nil || len(entries) != 2 || filepath.Ext(saved.Name) != ".sav" {
		t.Fatalf("explicit SAV menu SAVE: %+v %v", entries, err)
	}
	fresh := front(t)
	freshApp := fresh.App("native-light-fresh-load")
	save, list, load := fresh.SaveSeams(store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(save, list, load)
	groundAppLoad(t, freshApp, list, saved.Name)

	// Observe before HeadlessStep or any other simulation advance. A native
	// hash alone cannot detect the former fresh-viewer daytime seed.
	after, err := fresh.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.live.world.Tick() != 9343 || fresh.live.world.Hash() != hash || !bytes.Equal(before, after) {
		t.Fatal("explicit SAV SAVE/fresh LOAD changed canonical World before its first step")
	}
	if fresh.live.view == view || !fresh.live.view.TimeFlow() {
		t.Fatal("LOAD did not construct a fresh standard viewer")
	}
	if cold := fresh.live.view.Sun(); cold != warm {
		t.Fatalf("before first step: cold Sun=%+v, warm=%+v", cold, warm)
	}
	t.Logf("explicit SAV App SAVE/fresh LOAD: tick9343, unchanged native World, theta%.17g at last cadence9280/minute580", warm.Theta)
}
