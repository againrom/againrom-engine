package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseTownSAVWritesCurrentHumanSight(t *testing.T) {
	f := releaseFront(t)
	sourcePath, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	_, _, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(sourcePath)}, nil)
	if _, town, err := load(filepath.Base(sourcePath)); err != nil || !town {
		t.Fatal("original town LOAD", town, err)
	}
	member := trainingPartyMember(t, f, "hero")
	retained, ok := member.OriginalHumanState()
	if !ok {
		t.Fatal("original town hero has no retained Human")
	}
	want := retained.Sight ^ 0x0101
	old := setCurrentHumanSight(t, f, want)
	raw := currentTownSave(t, f)
	assertSavedHumanSight(t, raw, want)
	assertReloadedHumanSight(t, currentTownReload(t, raw), want)
	if unchanged, err := os.ReadFile(sourcePath); err != nil || !bytes.Equal(unchanged, source) {
		t.Fatal("original town input changed", err)
	}
	t.Logf("original town Human Sight: loaded=%d current=%d; ordinary SAVE wire and cold LOAD retain current value", old, want)
}
