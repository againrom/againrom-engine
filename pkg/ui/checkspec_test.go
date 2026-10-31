package ui

import "testing"

func TestCheckSpecMarksAreTheOnesEachProgramDrew(t *testing.T) {
	for _, tc := range []struct {
		style   CheckStyle
		off, on string
	}{
		{CheckText, "[ ] ", "[x] "},
		{CheckRadioText, "( ) ", "(*) "},
		{CheckArt, "", ""},
	} {
		if got := (CheckSpec{Style: tc.style}).Mark(); got != tc.off {
			t.Errorf("style %d off mark = %q, want %q", tc.style, got, tc.off)
		}
		if got := (CheckSpec{On: true, Style: tc.style}).Mark(); got != tc.on {
			t.Errorf("style %d on mark = %q, want %q", tc.style, got, tc.on)
		}
	}
	if got := markWord(true, "Grid"); got != "[x] Grid" {
		t.Errorf("markWord on = %q", got)
	}
	if got := markWord(false, "Grid"); got != "[ ] Grid" {
		t.Errorf("markWord off = %q", got)
	}
}

func TestChoiceGroupRowsAreArtCheckSpecs(t *testing.T) {
	radio := choiceGroup{Kind: choiceRadio, Labels: []string{"a", "b"}, Selected: 1}
	check := choiceGroup{Kind: choiceCheck, Labels: []string{"a", "b"}, Mask: 1}
	for i, want := range []bool{false, true} {
		if got := radio.check(i); got != (CheckSpec{On: want, Style: CheckArt}) {
			t.Errorf("radio row %d = %+v", i, got)
		}
	}
	for i, want := range []bool{true, false} {
		if got := check.check(i); got != (CheckSpec{On: want, Style: CheckArt}) {
			t.Errorf("check row %d = %+v", i, got)
		}
	}
	if got := (choiceGroup{Kind: choiceCheck, Mask: 1 << 31}).check(32); got.On {
		t.Error("row 32 of a check group reads as on")
	}
}
