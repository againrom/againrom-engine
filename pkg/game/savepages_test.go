package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/ui"
)

func TestSecondSaveSlotTakesTheSmallestUnusedNumber(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		listed []string
		want   string
	}{
		{nil, "game0000.sav"},
		{[]string{"game0000.sav", "game0001.sav", "game0003.sav"}, "game0002.sav"},
		{[]string{"GAME0000.SAV", "game0002.sav"}, "game0001.sav"},
		// Characters 4..7 that are not four digits use no number.
		{[]string{"gamex000.sav", "game12.sav", "game0000.sav"}, "game0001.sav"},
	} {
		if got := secondSaveSlot(dir, tc.listed); got != tc.want {
			t.Errorf("listed %q: slot %q, want %q", tc.listed, got, tc.want)
		}
	}
	// An unlisted file that holds a number is never the slot.
	if err := os.WriteFile(filepath.Join(dir, "game0000.sav"), []byte("not a save"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := secondSaveSlot(dir, nil); got != "game0001.sav" {
		t.Fatalf("slot over an unlisted file: %q", got)
	}
	// 9998 and 9999 are never chosen.
	var all []string
	for n := 0; n < 9998; n++ {
		all = append(all, fmt.Sprintf("game%04d.sav", n))
	}
	if got := secondSaveSlot(dir, all); got != "" {
		t.Fatalf("full slot range answered %q", got)
	}
}

func TestAnyGameSaveMatchesTheNamePatternWhateverTheContent(t *testing.T) {
	dir := t.TempDir()
	if anyGameSave(dir) || anyGameSave("") || anyGameSave(filepath.Join(dir, "missing")) {
		t.Fatal("an empty or missing directory reported a save")
	}
	for _, name := range []string{"save.sav", "game.txt", "mygame1.sav"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if anyGameSave(dir) {
		t.Fatal("names outside game*.sav reported a save")
	}
	if err := os.WriteFile(filepath.Join(dir, "Game0007.SAV"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !anyGameSave(dir) {
		t.Fatal("game0007.sav with any content is a save file")
	}
}

// TestSecondGamePagesAskDeleteInTheQuestionWindow: the second game's Load and
// Save pages ask their delete question in the measured window with the
// engine's Yes and No. A page with no window shape asks in its body.
func TestSecondGamePagesAskDeleteInTheQuestionWindow(t *testing.T) {
	p := secondSavePages(&FrontEnd{})
	if p.Save == nil {
		t.Fatal("no save page")
	}
	for name, c := range map[string]ui.PageConfirm{"load": p.Load.Confirm, "save": p.Save.Confirm} {
		if c.Window != secondPageQuestion || c.Window.Size != image.Pt(288, 224) || c.Yes != "Yes" || c.No != "No" {
			t.Fatalf("%s page: window %+v, answers %q %q", name, c.Window, c.Yes, c.No)
		}
	}
}

func TestSaveRowDetailFormatsTheLastWriteTime(t *testing.T) {
	at := time.Date(2001, 2, 3, 4, 5, 6, 0, time.Local)
	if got, want := saveRowDetail("Fort", at), "Fort  03.02.2001  04:05:06"; got != want {
		t.Fatalf("detail %q, want %q", got, want)
	}
}
