package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"againrom/pkg/base"
	"againrom/pkg/locale"
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
		if got := secondSaveSlot(dir, "", tc.listed); got != tc.want {
			t.Errorf("listed %q: slot %q, want %q", tc.listed, got, tc.want)
		}
	}
	// An unlisted file that holds a number is never the slot.
	if err := os.WriteFile(filepath.Join(dir, "game0000.sav"), []byte("not a save"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := secondSaveSlot(dir, "", nil); got != "game0001.sav" {
		t.Fatalf("slot over an unlisted file: %q", got)
	}
	// 9998 and 9999 are never chosen.
	var all []string
	for n := 0; n < 9998; n++ {
		all = append(all, fmt.Sprintf("game%04d.sav", n))
	}
	if got := secondSaveSlot(dir, "", all); got != "" {
		t.Fatalf("full slot range answered %q", got)
	}
}

func TestSecondSaveSlotNamesTheFileByTheTypedLabel(t *testing.T) {
	dir := t.TempDir()
	if got := secondSaveSlot(dir, "Camp fire", nil); got != "Camp fire.sav" {
		t.Fatalf("slot %q", got)
	}
	if got := secondSaveSlot(dir, "Привал у реки", nil); got != "Привал у реки.sav" {
		t.Fatalf("Cyrillic slot %q", got)
	}
	for _, tc := range []struct{ label, want string }{
		{`a<b>c:d"e/f\g|h?i*j`, "a_b_c_d_e_f_g_h_i_j.sav"},
		{"tab	here", "tab_here_.sav"},
		{"name. . ", "name.sav"},
		{"con", "con_.sav"},
		{"NUL", "NUL_.sav"},
		{"Com1", "Com1_.sav"},
		{"lpt9.x", "lpt9_.x.sav"},
		{"com0", "com0.sav"},
		{"console", "console.sav"},
		{"", "game0000.sav"},
		{"...", "game0000.sav"},
		{"   ", "game0000.sav"},
	} {
		if got := secondSaveSlot(dir, tc.label, nil); got != tc.want {
			t.Errorf("label %q: slot %q, want %q", tc.label, got, tc.want)
		}
	}
	long := secondSaveSlot(dir, strings.Repeat("ж", 300), nil)
	if len(long) > 255 || !utf8.ValidString(long) || !strings.HasSuffix(long, ".sav") || len(long) < 150 {
		t.Fatalf("long label slot has %d bytes, valid %v", len(long), utf8.ValidString(long))
	}
}

func TestSecondSaveSlotNeverTakesAnExistingOrHiddenName(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Camp.sav", "camp 2.SAV", "Camp 3.txt", "game9998.sav", "Quick save"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got := secondSaveSlot(dir, "CAMP", []string{"Camp.sav"}); got != "CAMP 3.sav" {
		t.Fatalf("taken name gave %q", got)
	}
	if got := secondSaveSlot(dir, "game9998", nil); got != "game9998 2.sav" {
		t.Fatalf("hidden name gave %q", got)
	}
	hidden := secondSaveHidden()
	for _, h := range hidden {
		base := strings.TrimSuffix(h, originalSaveExt)
		if got := secondSaveSlot(dir, base, nil); got == h || strings.EqualFold(got, h) {
			t.Fatalf("label %q took hidden name %q", base, got)
		}
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
		if !reflect.DeepEqual(c.Window, secondPageQuestion) || c.Window.ShapeSize != image.Pt(288, 224) || c.Yes != "Yes" || c.No != "No" {
			t.Fatalf("%s page: window %+v, answers %q %q", name, c.Window, c.Yes, c.No)
		}
	}
}

func TestSecondSavePageNameLimitFollowsTheInstallLanguage(t *testing.T) {
	want := map[string]int{"english": 123, "russian": 127, "": 123, "other": 123}
	for _, l := range locale.All() {
		if _, ok := want[l.Entry]; !ok {
			t.Fatalf("no expected limit for the language %q", l.Entry)
		}
	}
	for entry, limit := range want {
		f := &FrontEnd{}
		f.Archives = &Archives{Containers: townTextFS(t, nil), Base: base.Match{Profile: base.Profile{Language: entry}}}
		p := secondSavePages(f)
		if p.Save == nil || p.Save.NameLimit != limit {
			t.Errorf("language %q: name limit %+v, want %d", entry, p.Save, limit)
		}
	}
}

func TestSaveRowDetailFormatsTheLastWriteTime(t *testing.T) {
	at := time.Date(2001, 2, 3, 4, 5, 6, 0, time.Local)
	if got, want := saveRowDetail("Fort", at), "Fort  03.02.2001  04:05:06"; got != want {
		t.Fatalf("detail %q, want %q", got, want)
	}
}

// A save under a typed name is listed by content, and the Load lock's content
// scan finds it with no game*.sav present (DIV-4708).
func TestNamedSecondGameSaveIsListedAndOpensTheLoadLock(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Привал у реки.sav", "not a save.sav"} {
		raw := []byte("not a save")
		if name[0] != 'n' {
			raw = secondFixtureTown("Second town", secondFixtureHero(), 44)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := ownGameSaves(dir, 0, base.GameROM2)
	if len(got) != 1 || got[0].Name != "Привал у реки.sav" {
		t.Fatalf("rows %+v, want the named save alone", got)
	}
	if anyGameSave(dir) {
		t.Fatal("the name pattern alone opens the lock")
	}
	if n := len(scanSaves(saveScan{Dir: dir, Bounded: true, Regular: true, OfGame: true, Game: base.GameROM2, First: true})); n != 1 {
		t.Fatalf("lock scan found %d saves", n)
	}
}
