package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// sackTokenAtCell returns the RuntimeID and Identity of the one Sack root a
// document holds at cell.
func sackTokenAtCell(t *testing.T, raw []byte, x, y uint8) (uint32, uint32) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	var runtime, identity uint32
	for _, index := range doc.World.Sacks {
		token, _, _, err := savedSackRecord(&doc.Objects[index-1])
		if err != nil {
			t.Fatal(err)
		}
		if token.Position[2] == x && token.Position[3] == y {
			found++
			runtime, identity = token.RuntimeID, token.Identity
		}
	}
	if found != 1 {
		t.Fatalf("document holds %d Sack roots at %d,%d, want 1", found, x, y)
	}
	return runtime, identity
}

// An original Sack whose cell names no Sack keeps its RuntimeID and Identity
// across LOAD and SAVE. The file's Sack at 71,116 has no cell record naming
// it; LOAD binds it by its unique cell and gold (DIV-2506). Before, each SAVE
// wrote it with a new RuntimeID and Identity.
func TestReleaseSackWithoutCellKeyKeepsItsToken(t *testing.T) {
	for _, tc := range []struct {
		rel, hash string
		cells     [][2]uint8
		bound     bool
	}{
		{"2026-09-27/oldsaves7/game0006.sav", "b652cb4c5745b6dfa1a4cec12ea3be1dbb5991f44bbde97716755d6a5b8197ba", [][2]uint8{{71, 116}}, true},
	} {
		t.Run(tc.rel, func(t *testing.T) {
			_, raw := groundCorpusFile(t, tc.rel, tc.hash)
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			open, town, err := f.RestoreOriginal(raw)
			if err != nil || town {
				t.Fatalf("restore: town %v, %v", town, err)
			}
			app := f.App("sack cell key")
			defer app.StopAudio()
			app.Layout(1024, 768)
			if err := app.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			for _, cell := range tc.cells {
				sack := groundAt(f.live.world.Sacks(), int32(cell[0]), int32(cell[1]))
				if sack == nil || (sack.ObjectID != 0) != tc.bound {
					t.Fatalf("Sack at %v bound %v, want %v: %+v", cell, sack != nil && sack.ObjectID != 0, tc.bound, sack)
				}
			}
			dir := t.TempDir()
			prepared, err := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{OnMap: true, Directory: dir, Name: "sackkey", Format: ui.SaveSAV})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := prepared.Commit(true); err != nil {
				t.Fatal(err)
			}
			written, err := ReadSaveFile(filepath.Join(dir, "sackkey.sav"))
			if err != nil {
				t.Fatal(err)
			}
			for _, cell := range tc.cells {
				wantRuntime, wantIdentity := sackTokenAtCell(t, raw, cell[0], cell[1])
				gotRuntime, gotIdentity := sackTokenAtCell(t, written, cell[0], cell[1])
				if gotRuntime != wantRuntime || gotIdentity != wantIdentity {
					t.Errorf("Sack at %v written as RuntimeID %d Identity %#x, want %d %#x", cell, gotRuntime, gotIdentity, wantRuntime, wantIdentity)
				}
			}
		})
	}
}
