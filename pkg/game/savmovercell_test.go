package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

// moverCellRecord reads actor A's Position and mover words from a written SAV.
func moverCellRecord(t *testing.T, store SaveStore, name string) (pos, mover []byte) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(store.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if (r.Class == "Unit" || r.Class == "Human") && actorProjectionValue(t, *r, "Identity") == newGroupA {
			return crossingRawField(t, r, "Block12"), crossingRawField(t, r, "U154")
		}
	}
	t.Fatal("written SAV lacks the stepping actor")
	return nil, nil
}

// An actor saved mid-step names the claimed cell at mover +06 as well as at
// the claim +0x80 (MOVE-CLAIM-007). The original creates a non-centred
// actor's drawable at +06 (SAV-1119); a zero there puts it at cell 0,0.
func TestSavedMidStepActorNamesClaimedMoverCell(t *testing.T) {
	check := func(t *testing.T, store SaveStore, name string) {
		t.Helper()
		pos, mover := moverCellRecord(t, store, name)
		if pos[4] == 0x80 && pos[5] == 0x80 {
			t.Fatalf("actor is not mid-step: Position %x", pos)
		}
		route, claim := binary.LittleEndian.Uint16(mover[0x06:]), binary.LittleEndian.Uint16(mover[0x80:])
		if claim != 0x1010 || route != 0x1010 {
			t.Fatalf("mover +06=%04x +80=%04x, want both 1010", route, claim)
		}
		crossingNativeDoor(t, store, name)
	}
	t.Run("native step", func(t *testing.T) {
		f := newGroupFront(t, -1)
		app := f.App("native step mover cell")
		originals, store := t.TempDir(), SaveStore{Dir: t.TempDir()}
		if err := os.WriteFile(filepath.Join(originals, "game1115.sav"), newGroupLiteral(t, f, false), 0600); err != nil {
			t.Fatal(err)
		}
		save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
		app.SetSaveSeams(save, list, load)
		groundAppLoad(t, app, list, "game1115.sav")
		strideStartEast(t, f)
		check(t, store, crossingMenuSave(t, app, store))
	})
	t.Run("loaded step with zero mover cell", func(t *testing.T) {
		f := newGroupFront(t, -1)
		doc, err := sav.DecodeDocumentData(crossingCellLiteral(t, f, true))
		if err != nil {
			t.Fatal(err)
		}
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if (r.Class == "Unit" || r.Class == "Human") && actorProjectionValue(t, *r, "Identity") == newGroupA {
				binary.LittleEndian.PutUint16(crossingRawField(t, r, "U154")[0x06:], 0)
			}
		}
		raw, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		app := f.App("loaded step mover cell")
		originals, store := t.TempDir(), SaveStore{Dir: t.TempDir()}
		if err := os.WriteFile(filepath.Join(originals, "game1115.sav"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
		app.SetSaveSeams(save, list, load)
		groundAppLoad(t, app, list, "game1115.sav")
		check(t, store, crossingMenuSave(t, app, store))
	})
}
