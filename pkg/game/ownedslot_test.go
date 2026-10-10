package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Each kind's files and error texts, pinned as literals.
func TestOwnedSlotWriterBytesPerKind(t *testing.T) {
	raw := quickFixtureRaw(t)
	sum := fmt.Sprintf("%x", sha256.Sum256(raw))
	for _, kind := range []struct {
		name, base, ext, tag string
		write                func(SaveStore, int, uint64) error
	}{
		{"quick", "quick-save", ".quick-owner", "againrom-quick-sav", func(s SaveStore, slot int, seq uint64) error {
			return writeQuickSave(s, nil, slot, seq, raw, nil)
		}},
		{"timed", "timed-autosave", ".timed-owner", "againrom-timed-sav", func(s SaveStore, slot int, seq uint64) error {
			return writeTimedSave(s, nil, slot, seq, raw, nil)
		}},
	} {
		t.Run(kind.name, func(t *testing.T) {
			store := SaveStore{Dir: t.TempDir()}
			if err := kind.write(store, 1, 7); err != nil {
				t.Fatal(err)
			}
			if err := kind.write(store, 1, 9); err != nil {
				t.Fatal("rotation over an owned pair:", err)
			}
			entries, err := os.ReadDir(store.Dir)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			wantNames := []string{kind.base + "-2" + kind.ext, kind.base + "-2.sav"}
			slices.Sort(wantNames)
			if !slices.Equal(names, wantNames) {
				t.Fatalf("files %q, want %q", names, wantNames)
			}
			if got, err := os.ReadFile(filepath.Join(store.Dir, kind.base+"-2.sav")); err != nil || !bytes.Equal(got, raw) {
				t.Fatal("SAV bytes differ from the input", err)
			}
			wantOwner := `{"Kind":"` + kind.tag + `","Slot":1,"Sequence":9,"SHA256":"` + sum + `"}`
			if got, err := os.ReadFile(filepath.Join(store.Dir, kind.base+"-2"+kind.ext)); err != nil || string(got) != wantOwner {
				t.Fatalf("owner %q, want %q (%v)", got, wantOwner, err)
			}

			if err := kind.write(store, 1, 9); err == nil || err.Error() != kind.name+" slot changed after rotation selection" {
				t.Fatalf("stale sequence: %v", err)
			}
			if err := os.WriteFile(filepath.Join(store.Dir, kind.base+"-3.sav"), []byte("foreign"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := kind.write(store, 2, 10); err == nil || err.Error() != kind.name+" slot 3 is occupied by an unowned file" {
				t.Fatalf("unowned slot: %v", err)
			}
			if err := os.WriteFile(filepath.Join(store.Dir, kind.base+"-3"+kind.ext), []byte("{}"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := kind.write(store, 2, 10); err == nil || err.Error() != kind.name+" slot 3 has no matching ownership record" {
				t.Fatalf("foreign owner: %v", err)
			}
		})
	}
}

type (
	quickSaveOwner = ownedSlotRecord
	timedSaveOwner = ownedSlotRecord
)

func quickSaveBase(slot int) validatedSaveName { return quickSlots.slotBase(slot) }
func timedSaveBase(slot int) validatedSaveName { return timedSlots.slotBase(slot) }

func validateQuickSavePair(slot int, raw, owner []byte) (uint64, error) {
	return quickSlots.validatePair(slot, raw, owner)
}

func validateTimedSavePair(slot int, raw, owner []byte) (uint64, error) {
	return timedSlots.validatePair(slot, raw, owner)
}

func writeQuickSave(store SaveStore, fences []string, slot int, sequence uint64, raw []byte, files namedSaveFileOps) error {
	return quickSlots.write(store, fences, slot, sequence, raw, files)
}

func writeTimedSave(store SaveStore, fences []string, slot int, sequence uint64, raw []byte, files namedSaveFileOps) error {
	return timedSlots.write(store, fences, slot, sequence, raw, files)
}
