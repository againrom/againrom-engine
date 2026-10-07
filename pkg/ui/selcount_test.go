package ui

import (
	"image"
	"testing"
)

// The selection count as a field of the panel (AC-1, AC-2, AC-3, AC-6).
// Every fixture is built in test code and nothing here reads a game install;
// the font and the blacked-out ink layout are panel_test.go's.

// layoutWithoutCount is l with every count row removed — which over the
// authored layout is the layout THIS STORY INHERITED, derived from the shipped
// one rather than hand-copied beside it. A copy would drift: the day someone
// changes a colour or the padding, a hand-written twin stops being the
// pre-story layout while still passing as one, and the byte-identity below
// would then be asserting nothing.
func layoutWithoutCount(l PanelLayout) PanelLayout {
	rows := make([]PanelRow, 0, len(l.Rows))
	for _, r := range l.Rows {
		if r.Field != PanelFieldCount {
			rows = append(rows, r)
		}
	}
	l.Rows = rows
	return l
}

// samePixels reports whether two pictures are byte-identical, and where they
// first differ if they are not.
func samePixels(a, b *image.RGBA) (int, bool) {
	if a == nil || b == nil {
		return -1, a == nil && b == nil
	}
	if a.Bounds() != b.Bounds() || len(a.Pix) != len(b.Pix) {
		return -1, false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return i, false
		}
	}
	return -1, true
}

// AC-1 — the count states itself from two up and is absent below it.
func TestPanelCountFieldText(t *testing.T) {
	for _, tc := range []struct {
		selected int
		want     string
		ok       bool
	}{
		{0, "", false},
		{1, "", false},
		{2, "2", true},
		{17, "17", true},
	} {
		got, ok := panelText(PanelSubject{Selected: tc.selected}, PanelFieldCount)
		if ok != tc.ok {
			t.Errorf("%d selected: has-a-value = %v, want %v", tc.selected, ok, tc.ok)
			continue
		}
		if ok && got != tc.want {
			t.Errorf("%d selected: text = %q, want %q", tc.selected, got, tc.want)
		}
	}
}

func TestSingleSelectionPanelIsUnchanged(t *testing.T) {
	f := panelFont()
	before := layoutWithoutCount(AuthoredPanelLayout())

	for _, selected := range []int{0, 1} {
		s := panelSubjectFixture()
		s.Selected = selected

		// Four since 0140: the name, the health heading, the pair and the
		// cell. The property this asserts is unchanged — a selection of one or
		// none states no count row — and the number moved only because the
		// health block became two rows.
		if n := len(panelLines(AuthoredPanelLayout(), f, s)); n != 4 {
			t.Errorf("%d selected: the panel states %d row(s), want the countless 4", selected, n)
		}
		now := composePanel(AuthoredPanelLayout(), f, s)
		was := composePanel(before, f, panelSubjectFixture())
		if now == nil || was == nil {
			t.Fatalf("%d selected: composePanel returned nil", selected)
		}
		if i, same := samePixels(now, was); !same {
			t.Errorf("%d selected: the panel differs from the pre-story one at byte %d (%v vs %v)",
				selected, i, now.Bounds(), was.Bounds())
		}
	}
}

func TestPanelStatesTheCount(t *testing.T) {
	f := panelFont()
	l := AuthoredPanelLayout()
	s := panelSubjectFixture()
	s.Selected = 4

	lines := panelLines(l, f, s)
	want := []struct{ label, value string }{
		{"SELECTED", "4"},
		{"", "Warrior"},
		{"HEALTH", ""},
		{"", "63/100"},
		{"CELL", "12, 34"},
	}
	if len(lines) != len(want) {
		t.Fatalf("the panel states %d row(s), want %d", len(lines), len(want))
	}
	for i, w := range want {
		if lines[i].label != w.label || lines[i].value != w.value {
			t.Errorf("row %d = (%q, %q), want (%q, %q)",
				i, lines[i].label, lines[i].value, w.label, w.value)
		}
	}

	// The count reaches PIXELS and not only the line list: the four-unit panel
	// differs from the one-unit panel for the same unit.
	one := panelSubjectFixture()
	one.Selected = 1
	if _, same := samePixels(composePanel(l, f, s), composePanel(l, f, one)); same {
		t.Error("the four-unit and one-unit panels are byte-identical: the count paints nothing")
	}
}

func TestPanelCountIsAnOrdinaryRow(t *testing.T) {
	f := panelFont()
	s := panelSubjectFixture()
	s.Selected = 4

	t.Run("a layout with no count row states none", func(t *testing.T) {
		for _, ln := range panelLines(layoutWithoutCount(AuthoredPanelLayout()), f, s) {
			if ln.value == "4" || ln.label == "SELECTED" {
				t.Fatalf("a layout carrying no count row stated (%q, %q)", ln.label, ln.value)
			}
		}
		// And its picture is the one the same unit gets with nothing else
		// selected, which is the same statement made in pixels.
		one := panelSubjectFixture()
		one.Selected = 1
		if _, same := samePixels(
			composePanel(layoutWithoutCount(AuthoredPanelLayout()), f, s),
			composePanel(AuthoredPanelLayout(), f, one)); !same {
			t.Error("a count-less layout over four units differs from the authored one over one")
		}
	})

	t.Run("a placed count row lands at its own offset", func(t *testing.T) {
		at := image.Pt(31, 19)
		l := PanelLayout{LabelGap: 5, Size: image.Pt(120, 60),
			Rows: []PanelRow{{Field: PanelFieldCount, Label: "PICKED", At: at}}}
		lines := panelLines(l, f, s)
		if len(lines) != 1 {
			t.Fatalf("a one-row layout stated %d row(s)", len(lines))
		}
		if lines[0].at != at {
			t.Errorf("the placed count row is at %v, want %v", lines[0].at, at)
		}
		if lines[0].label != "PICKED" || lines[0].value != "4" {
			t.Errorf("the placed row = (%q, %q), want (%q, %q)",
				lines[0].label, lines[0].value, "PICKED", "4")
		}
		if wantX := f.Advance("PICKED") + l.LabelGap; lines[0].valueX != wantX {
			t.Errorf("the number starts at %d, want %d (the label's pen plus the gap)",
				lines[0].valueX, wantX)
		}
	})
}

// The count on screen (AC-4, AC-5, AC-7). The viewer and entity fixtures are
// panel_draw_test.go's.

func TestPanelCountsOnlyPresentUnits(t *testing.T) {
	v := panelViewer(t)
	v.SetFont(panelFont())
	// 2 is dead and 4 is not in the snapshot at all; 3, 5 and 7 are alive, and
	// 3 is the lowest of them, so it is the described unit.
	v.SetEntities([]MapEntity{
		panelEntity(2, "dead", -1, 10, 2, 2),
		panelEntity(3, "three", 10, 10, 3, 3),
		panelEntity(5, "five", 10, 10, 5, 5),
		panelEntity(7, "seven", 10, 10, 7, 7),
	})
	v.sel = selection{2, 3, 4, 5, 7}

	s, ok := v.panelSubject()
	if !ok {
		t.Fatal("no subject over a selection holding three live units")
	}
	if s.Selected != 3 {
		t.Errorf("the panel states %d selected over a selection of %d ids, want 3 — "+
			"the raw selection was counted", s.Selected, len(v.sel))
	}
	if s.ID != 3 || s.Name != "three" {
		t.Errorf("the described unit is %d %q, want 3 \"three\"", s.ID, s.Name)
	}
	lines := panelLines(AuthoredPanelLayout(), panelFont(), s)
	if len(lines) != 6 || lines[0].value != "3" {
		t.Errorf("the panel's first row is %+v over %d rows, want the count 3", lines[0], len(lines))
	}
}

func TestPanelRebuildsWhenTheCountChanges(t *testing.T) {
	v := panelViewer(t)
	v.SetFont(panelFont())
	live := []MapEntity{
		panelEntity(1, "one", 10, 10, 1, 1),
		panelEntity(2, "two", 10, 10, 2, 2),
		panelEntity(3, "three", 10, 10, 3, 3),
	}
	v.SetEntities(live)
	v.sel = selection{1, 2, 3}

	if _, _, ok := v.panelPresent(); !ok {
		t.Fatal("no panel over three selected units")
	}
	if v.panelBuilds != 1 {
		t.Fatalf("the first frame composed %d picture(s), want 1", v.panelBuilds)
	}
	if got := v.panelKey.selectionStatus; got != ([3]string{"Units", "selected:", "3"}) {
		t.Fatalf("the plural status is %q, want the three-actor phrase", got)
	}

	// One of the three dies. The described unit is untouched: only the count
	// moves, so nothing but the count can be what rebuilds the picture.
	dead := append([]MapEntity(nil), live...)
	dead[2] = panelEntity(3, "three", -1, 10, 3, 3)
	v.SetEntities(dead)
	if _, _, ok := v.panelPresent(); !ok {
		t.Fatal("no panel after one of the three died")
	}
	if v.panelBuilds != 2 {
		t.Errorf("a death that changed only the count gave %d builds, want 2 — the count is not "+
			"in the key", v.panelBuilds)
	}
	if got := v.panelKey.selectionStatus; got != ([3]string{"Units", "selected:", "2"}) {
		t.Errorf("the plural status after one death is %q, want the two-actor phrase", got)
	}
	if len(v.sel) != 3 {
		t.Errorf("the selection holds %d ids, want the 3 it was given — a death pruned it",
			len(v.sel))
	}

	for i := 0; i < 5; i++ {
		v.panelPresent()
	}
	if v.panelBuilds != 2 {
		t.Errorf("five unchanged frames after the death gave %d builds in total, want 2",
			v.panelBuilds)
	}
}

// AC-7 — nothing selected and no font each draw no panel, with several selected
// as with one.
func TestPanelCountNeedsASubjectAndAFont(t *testing.T) {
	ents := []MapEntity{
		panelEntity(1, "one", 10, 10, 1, 1),
		panelEntity(2, "two", 10, 10, 2, 2),
	}

	bare := panelViewer(t)
	bare.SetEntities(ents)
	bare.sel = selection{1, 2}
	if _, _, ok := bare.panelPresent(); ok {
		t.Error("a fontless viewer with two selected presented a panel")
	}
	if bare.panelBuilds != 0 {
		t.Errorf("a fontless viewer composed %d picture(s)", bare.panelBuilds)
	}

	v := panelViewer(t)
	v.SetFont(panelFont())
	v.SetEntities(ents)
	v.sel = nil
	if s, ok := v.panelSubject(); ok {
		t.Errorf("a viewer with nothing selected named subject %v", s)
	}
	// And a selection none of whose ids the snapshot holds is the same case.
	v.sel = selection{41, 42}
	if s, ok := v.panelSubject(); ok {
		t.Errorf("a selection of absent ids named subject %v", s)
	}
}
