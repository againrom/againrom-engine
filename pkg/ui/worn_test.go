package ui

import "testing"

// The worn-set row on the unit panel. Every fixture here is built in this
// file; nothing reads a game install.

// AC-8, first half — a subject carrying names states them in one row, the
// names in slot order and joined by ", ".
func TestPanelWornRowStatesNamesInSlotOrder(t *testing.T) {
	s := PanelSubject{Worn: [PanelWornSlots]string{
		"Chain Mail", "", "Round Shield", "", "", "", "", "", "", "", "", "Iron Helm",
	}}
	got, ok := panelText(s, PanelFieldWorn)
	if !ok {
		t.Fatal("a subject wearing three pieces stated no worn row")
	}
	if want := "Chain Mail, Round Shield, Iron Helm"; got != want {
		t.Errorf("worn row = %q, want %q", got, want)
	}
}

// AC-8 — a subject wearing nothing states no worn row at all: the empty slots
// are not three empty names joined by two commas, they are nothing to say,
// exactly like every other field this build treats as absent.
func TestPanelWornRowIsAbsentWhenNothingIsWorn(t *testing.T) {
	if _, ok := panelText(PanelSubject{}, PanelFieldWorn); ok {
		t.Error("a subject wearing nothing stated a worn row")
	}
}

// AC-8, first half, through PanelStatement — WEAPON and WORN answer the same
// question about the same subject and are read the same way, one entry each,
// `LABEL value`.
func TestPanelStatementStatesTheWornRow(t *testing.T) {
	l := PanelLayout{Rows: []PanelRow{
		{Field: PanelFieldWeapon, Label: "WEAPON"},
		{Field: PanelFieldWorn, Label: "WORN"},
	}}
	s := PanelSubject{
		Char: UnitCharacter{Known: true, Weapon: "Bronze Pike"},
		Worn: [PanelWornSlots]string{"Bronze Pike", "Round Shield", "Chain Mail"},
	}
	got := PanelStatement(l, s)
	want := []string{"WEAPON Bronze Pike", "WORN Bronze Pike, Round Shield, Chain Mail"}
	if len(got) != len(want) {
		t.Fatalf("%d entries %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWornRowSitsDirectlyUnderWeaponWithNoRightCell(t *testing.T) {
	rows := AuthoredPanelLayout().Rows
	weaponAt := -1
	for i, r := range rows {
		if r.Field == PanelFieldWeapon {
			weaponAt = i
		}
	}
	if weaponAt < 0 {
		t.Fatal("the authored layout is missing WEAPON")
	}
	if weaponAt+1 >= len(rows) || rows[weaponAt+1].Field != PanelFieldWorn {
		t.Fatalf("the row after WEAPON is %+v, want the WORN row directly beneath it",
			rows[weaponAt+1])
	}
	if rows[weaponAt+1].Right != nil {
		t.Errorf("the WORN row carries a right cell %+v, want none", rows[weaponAt+1].Right)
	}
}

// AC-8, second half — A SUBJECT WEARING NOTHING COMPOSES BYTE-FOR-BYTE THE
// PICTURE IT COMPOSED BEFORE THIS STORY. "Before this story" is derived from
// the shipped layout by pruning the one row this story added, never a
// hand-copied golden string that could drift from what AuthoredPanelLayout
// actually draws today.
func TestASubjectWearingNothingComposesThePanelItComposedBefore(t *testing.T) {
	s := sheetSubject() // Worn left at its zero value: every slot empty.
	s.Selected = 1

	before := AuthoredPanelLayout()
	pruned := make([]PanelRow, 0, len(before.Rows))
	for _, r := range before.Rows {
		if r.Field != PanelFieldWorn {
			pruned = append(pruned, r)
		}
	}
	before.Rows = pruned

	f := panelFont()
	now, was := composePanel(AuthoredPanelLayout(), f, s), composePanel(before, f, s)
	if now == nil || was == nil {
		t.Fatal("composePanel returned nil for a subject with a font")
	}
	if now.Bounds() != was.Bounds() {
		t.Fatalf("the box is %v where it was %v — an untold value took space",
			now.Bounds(), was.Bounds())
	}
	for i := range was.Pix {
		if now.Pix[i] != was.Pix[i] {
			t.Fatalf("byte %d of the picture differs: %#02x where it was %#02x",
				i, now.Pix[i], was.Pix[i])
		}
	}

	// The same identity read without a font, over PanelStatement.
	nowStmt, wasStmt := PanelStatement(AuthoredPanelLayout(), s), PanelStatement(before, s)
	if len(nowStmt) != len(wasStmt) {
		t.Fatalf("%d statement(s), want %d (the pre-story count)", len(nowStmt), len(wasStmt))
	}
	for i := range wasStmt {
		if nowStmt[i] != wasStmt[i] {
			t.Errorf("statement %d = %q, want %q", i, nowStmt[i], wasStmt[i])
		}
	}
}
