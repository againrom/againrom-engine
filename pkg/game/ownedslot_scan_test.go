package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOwnedSlotScanSeesOnlyItsOwnPair(t *testing.T) {
	raw := quickFixtureRaw(t)
	kinds := allOwnedSlotKinds()
	for i, kind := range kinds {
		dir := t.TempDir()
		if err := kind.write(SaveStore{Dir: dir}, nil, 1, 7, raw, nil); err != nil {
			t.Fatalf("%s %q: %v", kind.word, kind.game, err)
		}
		for j, other := range kinds {
			slots := other.scan(dir)
			for slot, got := range slots {
				wantSeq := uint64(0)
				if i == j && slot == 1 {
					wantSeq = 7
				}
				if got.sequence != wantSeq || got.free != (wantSeq == 0) {
					t.Errorf("%s %q written, %s %q slot %d: free %v sequence %d", kind.word, kind.game, other.word, other.game, slot+1, got.free, got.sequence)
				}
			}
		}
		foreign := filepath.Join(dir, string(kind.slotBase(2))+".sav")
		if err := os.WriteFile(foreign, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if got := kind.scan(dir)[2]; got.free || got.sequence != 0 {
			t.Errorf("%s %q: an unowned sav reads free %v sequence %d", kind.word, kind.game, got.free, got.sequence)
		}
	}
}
