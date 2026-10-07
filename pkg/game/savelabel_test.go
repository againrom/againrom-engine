package game

import (
	"bytes"
	"testing"
	"time"
)

// TestDrawableLabelSubstitutesForRunesTheLoadFontCannotDraw pins the
// substitution itself. Every expectation here is written out rather than
// derived from drawableLabel's own tables, so a table edited the wrong way
// fails instead of agreeing with itself.
func TestDrawableLabelSubstitutesForRunesTheLoadFontCannotDraw(t *testing.T) {
	for _, c := range []struct {
		name string
		in   string
		want string
	}{
		{"ascii is returned unchanged", "mission 30 - tick 176 - gold 70", "mission 30 - tick 176 - gold 70"},
		{"em dash, the one attested in the corpus", "mission 30 \u2014 tick 176 \u2014 gold 70", "mission 30 - tick 176 - gold 70"},
		{"town arm", "town \u2014 gold 0", "town - gold 0"},
		{"en dash", "a \u2013 b", "a - b"},
		{"ellipsis spells out", "more\u2026", "more..."},
		{"anything else keeps one mark per rune", "\u0430\u0431\u0432", "???"},
		{"empty", "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := drawableLabel(c.in)
			if got != c.want {
				t.Fatalf("drawableLabel(%q) = %q, want %q", c.in, got, c.want)
			}
			for i, r := range got {
				if r < 0x20 || r >= 0x7f {
					t.Fatalf("drawableLabel(%q) = %q, which still carries U+%04X at %d", c.in, got, r, i)
				}
			}
		})
	}
}

// TestTheLoadRowOfALegacyEmDashSaveIsDrawableAndTheFileIsUntouched carries
// the substitution through the seam pkg/ui is actually handed, which is the
// list closure SaveSeams returns and not SaveStore.List.
//
// THE TWO OTHER ASSERTIONS ARE THE BOUNDARY. SaveStore.List still reports the
// bytes in the header, because it is the store and not the screen; and the file
// on disk still carries them, because a save is the player's own and this build
// does not rewrite one to make it render.
func TestTheLoadRowOfALegacyEmDashSaveIsDrawableAndTheFileIsUntouched(t *testing.T) {
	const legacy = "mission 30 \u2014 tick 176 \u2014 gold 70"
	const want = "mission 30 - tick 176 - gold 70"

	data, err := EncodeSave(Snapshot{Gold: 70}, legacy)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	name, err := store.Write(time.Date(2026, 8, 14, 23, 52, 33, 0, time.UTC), data)
	if err != nil {
		t.Fatal(err)
	}

	_, list, _ := agsSaveSeams(&FrontEnd{}, store, OriginalStore{Dir: t.TempDir()}, nil)
	rows := list()
	if len(rows) != 1 {
		t.Fatalf("load rows = %v, want one", rows)
	}
	if rows[0].Label != want {
		t.Fatalf("drawn row label = %q, want %q", rows[0].Label, want)
	}

	listed, err := listAGS(store)
	if err != nil || len(listed) != 1 {
		t.Fatalf("List = (%v, %v), want one row", listed, err)
	}
	if listed[0].Label != legacy {
		t.Fatalf("stored label reported as %q; the store reports the header, not the row", listed[0].Label)
	}

	body, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(legacy)) {
		t.Fatal("the stored label was rewritten; the substitution must be for display only")
	}
}
