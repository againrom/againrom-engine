package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

// DIV-968
func TestReleaseOriginalTrailerRoundTripsOnLoad1138(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	if source.Trailer.TurnTracing != 0 || source.Trailer.ScriptTracing != 0 {
		t.Fatalf("Trailer = %+v, want both named dwords 0 on this file", source.Trailer)
	}
	reopened, err := sav.Open(source.Marshal())
	if err != nil {
		t.Fatalf("re-open after Marshal: %v", err)
	}
	if reopened.Trailer != source.Trailer {
		t.Fatalf("trailer changed across a Marshal round trip: got %+v, want %+v", reopened.Trailer, source.Trailer)
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1138 trailer")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0021.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game0021.sav")

	t.Logf("trailer TurnTracing=%d ScriptTracing=%d round-trips byte-identical; the ordinary App LOAD path reaches the map with this file",
		source.Trailer.TurnTracing, source.Trailer.ScriptTracing)
}
