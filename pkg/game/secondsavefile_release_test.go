package game

import (
	"path/filepath"
	"sync"
	"testing"

	"againrom/pkg/base"
)

// secondSaved maps a save directory and a typed label to the file the second
// game's save page wrote for it.
var secondSaved = struct {
	sync.Mutex
	files map[[2]string]string
}{files: map[[2]string]string{}}

// recordSecondSave finds the file the save page just wrote under label in
// dir: the newest second-game save carrying that label.
func recordSecondSave(t *testing.T, dir, label string) string {
	t.Helper()
	for _, e := range ownGameSaves(dir, 0, base.GameROM2) {
		if e.Label == label {
			secondSaved.Lock()
			secondSaved.files[[2]string{filepath.Clean(dir), label}] = e.Name
			secondSaved.Unlock()
			return e.Name
		}
	}
	t.Fatalf("no second-game save labelled %q in %s", label, dir)
	return ""
}

// secondSaveFile is the file recordSecondSave found for label in dir.
func secondSaveFile(t *testing.T, dir, label string) string {
	t.Helper()
	secondSaved.Lock()
	defer secondSaved.Unlock()
	name, ok := secondSaved.files[[2]string{filepath.Clean(dir), label}]
	if !ok {
		t.Fatalf("no save labelled %q was written in %s", label, dir)
	}
	return name
}
