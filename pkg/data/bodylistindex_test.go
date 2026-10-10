package data

import "testing"

// The shipped list has 26 entries with a blank at entry 22 (ANIM-108): a
// Weapons row D reads entry D-1, row 23 meets the blank, rows 24 and 25 their
// own names, row 26 the last entry, and row 27 is past the end.
func TestBodyListCountsTheBlankEntryAndReadsPastTheEnd(t *testing.T) {
	names := make([]string, 26)
	for i := range names {
		names[i] = "n"
	}
	names[0], names[1] = "unarmed", "swordsman"
	names[22], names[23], names[24], names[25] = "", "Sonic Beam", "Flame Thrower", "swordsman"
	text := ""
	for _, n := range names {
		text += n + "\r\n"
	}
	l := ParseBodyList([]byte(text))
	if l.Len() != 26 {
		t.Fatalf("list holds %d entries, want 26", l.Len())
	}
	for _, tc := range []struct {
		row  int
		want HeroBody
		ok   bool
	}{
		{2, "swordsman", true},
		{23, "", false},
		{24, "Sonic Beam", true},
		{25, "Flame Thrower", true},
		{26, "swordsman", true},
		{27, "", false},
	} {
		var e Equipment
		e.SetCode(1, ComposeItemCode(0, 1, 0, tc.row))
		got, ok := HeroBodyFor(l, e)
		if got != tc.want || ok != tc.ok {
			t.Errorf("row %d: %q ok=%t, want %q ok=%t", tc.row, got, ok, tc.want, tc.ok)
		}
	}
	// A dagger (row 2) derives class 3, class 4 with a shield; a name no arm
	// matches leaves class 1.
	var dagger Equipment
	dagger.SetCode(1, ComposeItemCode(0, 1, 0, 2))
	if _, _, class, _ := HeroAppearance(l, dagger, false, false); class != 3 {
		t.Errorf("dagger class %d, want 3", class)
	}
	dagger.SetCode(2, ComposeItemCode(0, 2, 0, 1))
	if _, _, class, _ := HeroAppearance(l, dagger, false, false); class != 4 {
		t.Errorf("dagger and shield class %d, want 4", class)
	}
	var odd Equipment
	odd.SetCode(1, ComposeItemCode(0, 1, 0, 24))
	if _, _, class, _ := HeroAppearance(l, odd, false, false); class != 1 {
		t.Errorf("a name no arm matches gives class %d, want 1", class)
	}
}
