package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

// A projectile in flight of a picture whose client consumers the engine does
// not run is still written. game0024 holds a picture 13 burst and a picture 2
// bolt; SAVE at the load point must write the source's projectile records
// leaf for leaf and the allocator and ID list with them.
func TestReleaseResaveOfProjectilesInFlightWritesTheSourceRecords(t *testing.T) {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS: projectile resave witness requires owner saves")
	}
	dir := filepath.Join(corpus, "2026-10-02", "projectiles-original-en")
	raw, err := os.ReadFile(filepath.Join(dir, "game0024.sav"))
	if err != nil {
		t.Skip("corpus file missing:", err)
	}
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := projectile1157Read(source.Store)
	if err != nil {
		t.Fatal(err)
	}
	if !want.present || len(want.items) != 2 || want.free != 31 {
		t.Fatalf("source projectiles changed: %+v", want)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("projectile resave")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game0024.sav")
	_, _, written := menuSAVE(t, f, app, OriginalStore{Dir: dir})
	out, err := sav.Open(written)
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectile1157Read(out.Store)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("resaved Projectiles differ from the source: %v | %+v | %+v", err, got, want)
	}
}
