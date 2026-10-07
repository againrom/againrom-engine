package ui

import (
	"image"
	"testing"
)

// A row may state two fields. AC-4 — the picture a layout with no right
// cell composes is unchanged — is witnessed by every existing panel and
// readout test passing unmodified, exactly as the plan's own risk note says:
// neither file's layout ever sets PanelRow.Right, so both new passes over it
// are inert by construction.

// panelTwoCellFixtureLayout is a layout with one two-cell row over two fields
// that can each be independently absent — NAME (empty Name) and MANA (a zero
// MaxMana) — followed by a one-cell row over a field that is never absent, so
// a subject that drops the first row still states something to have moved up
// into its place.
func panelTwoCellFixtureLayout() PanelLayout {
	l := AuthoredPanelLayout()
	l.ColumnGap = 4
	l.Rows = []PanelRow{
		{Field: PanelFieldName, Label: "NAME", Right: &PanelCell{Field: PanelFieldMana, Label: "MANA"}},
		{Field: PanelFieldCell, Label: "CELL"},
	}
	return l
}

// AC-1 — the four shapes a two-cell row's subject can take.
func TestPanelTwoCellRowFourSubjectShapes(t *testing.T) {
	f, l := panelFont(), panelTwoCellFixtureLayout()

	t.Run("both cells state something: both draw, the right one at the shared x", func(t *testing.T) {
		s := PanelSubject{Name: "Warrior", Mana: 12, MaxMana: 40, Cell: image.Pt(1, 2)}
		lines := panelLines(l, f, s)
		if len(lines) != 2 {
			t.Fatalf("got %d row(s), want 2", len(lines))
		}
		row := lines[0]
		if row.label != "NAME" || row.value != "Warrior" {
			t.Fatalf("row 0 = (%q, %q), want (NAME, Warrior)", row.label, row.value)
		}
		if !row.right || row.rightLabel != "MANA" || row.rightValue != "12/40" {
			t.Fatalf("row 0's right cell = (drawn=%v, %q, %q), want (true, MANA, 12/40)",
				row.right, row.rightLabel, row.rightValue)
		}
		// The shared origin is the WIDEST drawn left cell in the box, not
		// this row's own — computed over both rows rather than assumed, so a
		// change to either fixture string cannot make this pass vacuously.
		_, nameW := panelCellMetrics(f, l.LabelGap, "NAME", "Warrior")
		_, cellW := panelCellMetrics(f, l.LabelGap, "CELL", "1, 2")
		wantX := nameW
		if cellW > wantX {
			wantX = cellW
		}
		wantX += l.ColumnGap
		if row.rightX != wantX {
			t.Errorf("right column x = %d, want %d (the left cell's own width plus the gap)", row.rightX, wantX)
		}
		if lines[1].label != "CELL" || lines[1].value != "1, 2" || lines[1].right {
			t.Errorf("row 1 = %+v, want the plain CELL row", lines[1])
		}
	})

	t.Run("only the left cell states something: only it draws", func(t *testing.T) {
		s := PanelSubject{Name: "Warrior", Cell: image.Pt(1, 2)} // MaxMana == 0
		lines := panelLines(l, f, s)
		if len(lines) != 2 {
			t.Fatalf("got %d row(s), want 2", len(lines))
		}
		if lines[0].label != "NAME" || lines[0].value != "Warrior" || lines[0].right {
			t.Errorf("row 0 = %+v, want an undrawn right cell", lines[0])
		}
	})

	t.Run("only the right cell states something: it draws at the left cell's own origin", func(t *testing.T) {
		s := PanelSubject{Mana: 12, MaxMana: 40, Cell: image.Pt(1, 2)} // Name == ""
		lines := panelLines(l, f, s)
		if len(lines) != 2 {
			t.Fatalf("got %d row(s), want 2", len(lines))
		}
		if lines[0].label != "MANA" || lines[0].value != "12/40" || lines[0].right {
			t.Errorf("row 0 = %+v, want the slid MANA cell alone, left-positioned and undrawn on the right",
				lines[0])
		}
		if lines[0].at.X != l.Pad.X {
			t.Errorf("the slid row starts at x=%d, want the row's own origin %d", lines[0].at.X, l.Pad.X)
		}
	})

	t.Run("neither cell states something: no row at all, and the row after it moves up", func(t *testing.T) {
		s := PanelSubject{Cell: image.Pt(1, 2)} // Name == "" and MaxMana == 0
		lines := panelLines(l, f, s)
		if len(lines) != 1 {
			t.Fatalf("got %d row(s), want 1 — the two-cell row dropped entirely", len(lines))
		}
		if lines[0].label != "CELL" || lines[0].value != "1, 2" {
			t.Fatalf("the surviving row is %+v, want CELL", lines[0])
		}
		if lines[0].at.Y != l.Pad.Y {
			t.Errorf("the surviving row sits at y=%d, want %d — the dropped row left a gap behind it",
				lines[0].at.Y, l.Pad.Y)
		}
	})
}

// AC-2 — the shared right-column x is a function of the DRAWN left cells
// only. Two subjects differing in whether a wide left cell is stated get
// different right-column origins for the very same two-cell row.
func TestPanelRightColumnOriginFollowsTheDrawnLeftCells(t *testing.T) {
	f := panelFont()
	l := AuthoredPanelLayout()
	l.ColumnGap = 5
	l.Rows = []PanelRow{
		// Present only for a known character with a weapon, and DELIBERATELY
		// WIDE so that when it draws it dominates the widest-left-cell walk.
		// It carries a right cell of its own, because only a row that does is
		// in the grid that walk measures — see the test below.
		{Field: PanelFieldWeapon, Label: "WEAPON",
			Right: &PanelCell{Field: PanelFieldMoveSpeed, Label: "SPD"}},
		{Field: PanelFieldHealth, Label: "HP", Right: &PanelCell{Field: PanelFieldCell, Label: "CELL"}},
	}

	// No character AND no combat block, so the wide row drops whole rather
	// than sliding its right cell into the left position.
	narrow := PanelSubject{HP: 63, MaxHP: 100, Cell: image.Pt(1, 2)}
	wide := PanelSubject{HP: 63, MaxHP: 100, Cell: image.Pt(1, 2), Speed: 17,
		Combat: UnitCombat{Known: true},
		Char:   UnitCharacter{Known: true, Weapon: "a-very-long-two-handed-weapon-name"}}

	narrowLines := panelLines(l, f, narrow)
	wideLines := panelLines(l, f, wide)

	if len(narrowLines) != 1 {
		t.Fatalf("the narrow subject drew %d row(s), want 1 (WEAPON dropped)", len(narrowLines))
	}
	if len(wideLines) != 2 {
		t.Fatalf("the wide subject drew %d row(s), want 2 (WEAPON present)", len(wideLines))
	}

	narrowRow := narrowLines[0]
	wideRow := wideLines[1]
	if !narrowRow.right || !wideRow.right {
		t.Fatal("the HP/CELL row did not carry a drawn right cell in one of the two subjects")
	}

	// Computed independently of layoutLines, over each subject's own set of
	// drawn left cells — which is what "a function of the drawn left cells
	// only" means. The narrow subject draws HP alone; the wide one draws HP
	// AND WEAPON, so its shared origin is the wider of the two, not WEAPON's
	// assumed by construction.
	_, hpWidth := panelCellMetrics(f, l.LabelGap, "HP", "63/100")
	_, weaponWidth := panelCellMetrics(f, l.LabelGap, "WEAPON", wide.Char.Weapon)
	wantWide := hpWidth
	if weaponWidth > wantWide {
		wantWide = weaponWidth
	}

	if want := hpWidth + l.ColumnGap; narrowRow.rightX != want {
		t.Errorf("narrow subject: right column x = %d, want %d (HP alone)", narrowRow.rightX, want)
	}
	if want := wantWide + l.ColumnGap; wideRow.rightX != want {
		t.Errorf("wide subject: right column x = %d, want %d (the widest of HP and WEAPON)", wideRow.rightX, want)
	}
	if narrowRow.rightX == wideRow.rightX {
		t.Error("the two subjects produced the same right-column origin: a dropped wide row still counted")
	}
	if narrowRow.rightX >= wideRow.rightX {
		t.Errorf("the wide subject's origin (%d) is not past the narrow one's (%d)",
			wideRow.rightX, narrowRow.rightX)
	}
}

// AC-2, the half the measurement found and no test had asked for — A ROW WITH
// NO RIGHT CELL DOES NOT PLACE THE GRID.
func TestAFullWidthRowDoesNotPlaceTheSecondColumn(t *testing.T) {
	f := panelFont()
	l := AuthoredPanelLayout()
	// THE WIDTH IS UNPINNED FOR THIS TEST (0140). The shipped layout now names
	// a fixed Size.X — one constant shared with the minimap, so neither box can
	// be resized by a selection — and a fixed width cannot grow to hold
	// anything, which is what the second half below is about. The property
	// under test belongs to panelBox's FIT rule, so the fixture is the authored
	// layout with the fit put back rather than a layout invented here.
	l.Size = image.Point{}
	l.ColumnGap = 5
	l.Rows = []PanelRow{
		{Field: PanelFieldHealth, Label: "HP", Right: &PanelCell{Field: PanelFieldCell, Label: "CELL"}},
		{Field: PanelFieldWeapon, Label: "WEAPON"},
	}

	bare := PanelSubject{HP: 63, MaxHP: 100, Cell: image.Pt(1, 2)}
	armed := bare
	armed.Char = UnitCharacter{Known: true, Weapon: "a-very-long-two-handed-weapon-name"}

	bareLines, armedLines := panelLines(l, f, bare), panelLines(l, f, armed)
	if len(bareLines) != 1 || len(armedLines) != 2 {
		t.Fatalf("drew %d and %d rows, want 1 and 2", len(bareLines), len(armedLines))
	}
	if bareLines[0].rightX != armedLines[0].rightX {
		t.Errorf("the second column moved from x=%d to x=%d when a FULL-WIDTH row appeared;"+
			" a row outside the grid is placing the grid",
			bareLines[0].rightX, armedLines[0].rightX)
	}

	// And it is not that the wide row is being ignored: the BOX still grows to
	// hold it, which is where a long name has always been paid for.
	bareBox, armedBox := panelBox(l, f, bareLines), panelBox(l, f, armedLines)
	if armedBox.X <= bareBox.X {
		t.Errorf("the box is %d wide with the weapon row and %d without it;"+
			" the full-width row is being dropped from the fit as well", armedBox.X, bareBox.X)
	}
}

// AC-3 — every pixel a two-cell row paints lies inside the composed box. The
// right cell's own label ("8") selects the fixture font's one glyph whose art
// overhangs its advance (see TestComposePanelStatesExactlyItsRows), which is
// the only configuration able to catch a box measured by the pen instead of
// the label's own ink.
func TestPanelTwoCellRowPaintStaysInsideTheBox(t *testing.T) {
	f := panelFont()
	l := panelInkLayout()
	l.ColumnGap = 3
	l.Rows = []PanelRow{
		{Field: PanelFieldHealth, Label: "HP", Right: &PanelCell{Field: PanelFieldCell, Label: "8"}},
		{Field: PanelFieldName, Label: "NAME"},
	}
	s := PanelSubject{Name: "Warrior", HP: 63, MaxHP: 100, Cell: image.Pt(12, 34)}

	lines := panelLines(l, f, s)
	box := panelBox(l, f, lines)

	img := composePanel(l, f, s)
	if img == nil {
		t.Fatal("composePanel returned nil")
	}
	right, bottom, found := inkExtent(img)
	if !found {
		t.Fatal("the picture carries no text pixel at all")
	}
	if right >= box.X || bottom >= box.Y {
		t.Errorf("text reaches (%d, %d) in a box of %v", right, bottom, box)
	}

	// Nothing was clipped either: the widest row's own last pixel, over both
	// cells, is present — which a box short of the second column would cut.
	widest := 0
	for _, ln := range lines {
		if r := ln.at.X + ln.width; r > widest {
			widest = r
		}
	}
	if right < widest-panelCellW {
		t.Errorf("the rightmost text pixel is at %d but the widest row runs to %d: content was clipped",
			right, widest)
	}
}

// AC-10 — PanelSubject stays comparable with ==. The refresh key (panelKey)
// holds one directly, and this is that same requirement pinned as a compile
// check of its own: this test will not build at all once PanelSubject stops
// being a valid map key.
func TestPanelSubjectIsUsableAsAMapKey(t *testing.T) {
	m := map[PanelSubject]int{}
	a := panelSubjectFixture()
	b := a
	b.ID = a.ID + 1

	m[a] = 1
	m[b] = 2

	if m[a] != 1 || m[b] != 2 {
		t.Errorf("map over PanelSubject keys = %v, %v, want 1, 2", m[a], m[b])
	}
	if _, ok := m[PanelSubject{}]; ok {
		t.Error("the zero PanelSubject was present in a map that never inserted it")
	}
}
