package ui

import (
	"image"
	"testing"
)

// WHAT THE PANEL SAYS, READ WITHOUT DRAWING IT (AC-8).
//
// PanelStatement is the reading a measurement can be checked against: "the box
// got smaller" is pixels and needs the game's own font, "nothing it stated was
// lost" is this and needs no font at all. Keeping them apart is what stops a
// compaction from being judged by the code that performed it.

// AC-8 — one entry per drawn row, in row order, `LABEL value` for a labelled
// cell and the value alone for an unlabelled one.
func TestPanelStatementIsOneEntryPerDrawnRow(t *testing.T) {
	l := PanelLayout{Rows: []PanelRow{
		{Field: PanelFieldName},
		{Field: PanelFieldHealth, Label: "HP"},
		{Field: PanelFieldCell, Label: "CELL"},
	}}
	s := PanelSubject{Name: "Warrior", HP: 63, MaxHP: 100, Cell: image.Pt(12, 34)}

	got := PanelStatement(l, s)
	want := []string{"Warrior", "HP 63/100", "CELL 12, 34"}
	if len(got) != len(want) {
		t.Fatalf("%d entries %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d is %q, want %q", i, got[i], want[i])
		}
	}
}

// AC-8 — A DROPPED ROW CONTRIBUTES NO ENTRY, which is the same rule that makes
// it cost no space. The name is absent for a unit whose id resolved to no
// class, and the count says nothing below two.
func TestPanelStatementSkipsARowWithNothingToState(t *testing.T) {
	l := PanelLayout{Rows: []PanelRow{
		{Field: PanelFieldCount, Label: "SELECTED"},
		{Field: PanelFieldName},
		{Field: PanelFieldHealth, Label: "HP"},
	}}
	s := PanelSubject{HP: 5, MaxHP: 5, Selected: 1}

	got := PanelStatement(l, s)
	if len(got) != 1 || got[0] != "HP 5/5" {
		t.Fatalf("got %q, want exactly [\"HP 5/5\"]", got)
	}

	// And the two rows come back the moment they have something to say.
	s.Name, s.Selected = "Warrior", 4
	if got, want := len(PanelStatement(l, s)), 3; got != want {
		t.Fatalf("%d entries, want %d", got, want)
	}
}

// AC-8 over the SHIPPED layout and the two subject kinds the window has to
// serve, so the reading the measuring tool prints is the reading a test has
// seen. The counts are the ones verification.md records against the install.
func TestPanelStatementReadsBothSubjectKinds(t *testing.T) {
	l := AuthoredPanelLayout()

	full := PanelStatement(l, sheetSubject())
	if len(full) == 0 {
		t.Fatal("a fully described unit states nothing")
	}

	bare := sheetSubject()
	bare.Char = UnitCharacter{}
	plain := PanelStatement(l, bare)
	if len(plain) >= len(full) {
		t.Errorf("a unit with no character states %d rows and one with a character %d;"+
			" the character rows are not being dropped", len(plain), len(full))
	}
	for _, r := range plain {
		if r == "" {
			t.Error("a stated row is empty, so an absent value is being drawn as a blank")
		}
	}
}
