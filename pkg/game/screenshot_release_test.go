package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseScreenshotAltSSelectedProfileSuppliedFrame(t *testing.T) {
	f, a, _ := cheatReleaseApp(t, false)
	store := SaveStore{Dir: t.TempDir()}
	launchDir := store.Dir
	f.ConfigureSaveSeams(a, store, OriginalStore{Dir: f.Archives.Root}, nil)
	store.Dir = t.TempDir()
	before := f.live.world.Hash()
	if err := a.HeadlessKey("alt-s"); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(launchDir); err != nil || len(entries) != 0 {
		t.Fatal("Alt+S wrote before receiving a completed frame", entries, err)
	}
	frame := screenshotFixture()
	if err := a.HeadlessScreenshotFrame(frame); err != nil {
		t.Fatal(err)
	}
	screenshotWantPNG(t, filepath.Join(launchDir, "screen0000.png"), frame)
	if err := a.HeadlessScreenshotFrame(frame); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(launchDir); err != nil || len(entries) != 1 {
		t.Fatal("capture request was delivered more than once", entries, err)
	}
	if entries, err := os.ReadDir(store.Dir); err != nil || len(entries) != 0 {
		t.Fatal("capture escaped fixed launch output", entries, err)
	}
	if f.live.world.Hash() != before || f.live.cheats.privilege[1] != 0 {
		t.Fatal("screenshot changed simulation or required privilege")
	}
}
