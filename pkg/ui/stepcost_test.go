package ui

import (
	"image"
	"testing"
)

// What a step onto the cursor's cell costs the selected unit.

// scAsk is a question that records what it was asked and answers what it was
// told to.
type scAsk struct {
	id       uint32
	col, row int
	calls    int
	give     StepCost
	ok       bool
}

func (a *scAsk) fn() StepCostFunc {
	return func(id uint32, col, row int) (StepCost, bool) {
		a.id, a.col, a.row, a.calls = id, col, row, a.calls+1
		return a.give, a.ok
	}
}

// scViewer is a viewer with one entity, that entity selected, and the cursor
// resolved onto a cell of the map.
func scViewer(t *testing.T, ask *scAsk) *Viewer {
	t.Helper()
	v := readoutViewer(t)
	v.entities = []MapEntity{{ID: 7, Cell: image.Pt(4, 5), Speed: 16, TransitSpan: 16}}
	v.sel = selection{7}
	if ask != nil {
		v.SetStepCost(ask.fn())
	}
	// A cursor position the camera resolves to a cell inside the extent, and to
	// an ASYMMETRIC one: at (3,3) a column and a row that reached the far side
	// the wrong way round would be invisible, and the first fixture here did
	// exactly that. The subject is asserted on both counts below rather than
	// assumed.
	v.cursorX, v.cursorY, v.hasCursor = 200, 120, true
	return v
}

func TestTheStepRowsStateWhatTheQuestionAnswered(t *testing.T) {
	ask := &scAsk{give: StepCost{Rate: 21, Transit: 13, Adjacent: true}, ok: true}
	v := scViewer(t, ask)

	s := v.readoutSubjectOf(60)
	if !s.OnMap {
		t.Fatal("the fixture's cursor resolved to no cell — the test asserts nothing")
	}
	if !s.HasStep {
		t.Fatal("the subject holds no step figure for a selected unit on a resolved cell")
	}
	if got := textOf(s, PanelFieldStepRate); got != "21" {
		t.Errorf("the rate row states %q, want 21", got)
	}
	if got := textOf(s, PanelFieldStepCost); got != "13 t" {
		t.Errorf("the transit row states %q, want \"13 t\"", got)
	}
}

func TestTheQuestionIsAskedAboutTheSelectedUnitAndTheCursorsCell(t *testing.T) {
	ask := &scAsk{give: StepCost{Rate: 8, Transit: 32, Adjacent: true}, ok: true}
	v := scViewer(t, ask)

	s := v.readoutSubjectOf(60)
	if ask.calls != 1 {
		t.Fatalf("the question was asked %d time(s), want exactly 1", ask.calls)
	}
	if s.Cursor.X == s.Cursor.Y {
		t.Fatalf("the fixture's cursor resolved to the symmetric cell (%d,%d) — a column and a "+
			"row that crossed the seam the wrong way round would pass this test", s.Cursor.X, s.Cursor.Y)
	}
	if ask.id != 7 {
		t.Errorf("the question named entity %d, want the selected 7", ask.id)
	}
	if ask.col != s.Cursor.X || ask.row != s.Cursor.Y {
		t.Errorf("the question named cell (%d,%d) and the CURSOR row states (%d,%d)",
			ask.col, ask.row, s.Cursor.X, s.Cursor.Y)
	}
}

func TestTheStepRowsMarkAPairThatIsNoSingleStep(t *testing.T) {
	ask := &scAsk{give: StepCost{Rate: 21, Transit: 13, Adjacent: false}, ok: true}
	s := scViewer(t, ask).readoutSubjectOf(60)

	if got, want := textOf(s, PanelFieldStepRate), "21 "+readoutFar; got != want {
		t.Errorf("the rate row states %q, want %q", got, want)
	}
	if got, want := textOf(s, PanelFieldStepCost), "13 t "+readoutFar; got != want {
		t.Errorf("the transit row states %q, want %q", got, want)
	}
}

func TestTheStepRowsStateAnAbsenceRatherThanAZero(t *testing.T) {
	for _, tc := range []struct {
		what  string
		build func(t *testing.T) *Viewer
	}{
		{"the far side declined", func(t *testing.T) *Viewer {
			return scViewer(t, &scAsk{ok: false})
		}},
		{"no question was ever installed", func(t *testing.T) *Viewer {
			return scViewer(t, nil)
		}},
		{"the cursor resolves to no cell", func(t *testing.T) *Viewer {
			v := scViewer(t, &scAsk{give: StepCost{Rate: 9, Transit: 9, Adjacent: true}, ok: true})
			v.hasCursor = false
			return v
		}},
	} {
		t.Run(tc.what, func(t *testing.T) {
			s := tc.build(t).readoutSubjectOf(60)
			if s.HasStep {
				t.Fatalf("the subject holds a step figure: %+v", s.Step)
			}
			for _, f := range []PanelField{PanelFieldStepRate, PanelFieldStepCost} {
				if got := textOf(s, f); got != readoutAbsent {
					t.Errorf("field %d states %q, want the absence marker %q", f, got, readoutAbsent)
				}
			}
		})
	}
}

func TestNoQuestionIsAskedWithoutBothOfItsInputs(t *testing.T) {
	t.Run("no unit selected", func(t *testing.T) {
		ask := &scAsk{ok: true}
		v := scViewer(t, ask)
		v.sel = nil
		if s := v.readoutSubjectOf(60); s.HasUnit {
			t.Fatal("the fixture still holds a unit")
		}
		if ask.calls != 0 {
			t.Errorf("the question was asked %d time(s) with nothing selected", ask.calls)
		}
	})
	t.Run("no cell under the cursor", func(t *testing.T) {
		ask := &scAsk{ok: true}
		v := scViewer(t, ask)
		v.hasCursor = false
		v.readoutSubjectOf(60)
		if ask.calls != 0 {
			t.Errorf("the question was asked %d time(s) with no cell resolved", ask.calls)
		}
	})
}

func TestTheStepRowsGoWithTheOtherUnitRows(t *testing.T) {
	none := readoutSubject{}
	for _, f := range []PanelField{PanelFieldStepRate, PanelFieldStepCost} {
		if _, ok := readoutText(none, f); ok {
			t.Errorf("field %d stated something with nothing selected", f)
		}
	}
}

func TestBothStepValuesAreInTheRebuildKey(t *testing.T) {
	base := readoutFixture()
	base.HasStep, base.Step = true, StepCost{Rate: 16, Transit: 16, Adjacent: true}

	for _, tc := range []struct {
		what string
		with readoutSubject
	}{
		{"the rate alone", func() readoutSubject { s := base; s.Step.Rate = 17; return s }()},
		{"the transit alone", func() readoutSubject { s := base; s.Step.Transit = 15; return s }()},
		{"the marker alone", func() readoutSubject { s := base; s.Step.Adjacent = false; return s }()},
		{"the presence alone", func() readoutSubject { s := base; s.HasStep = false; return s }()},
	} {
		t.Run(tc.what, func(t *testing.T) {
			if tc.with == base {
				t.Error("the subject compares equal after the change — it is not in the key")
			}
		})
	}
}

func TestTheStepRowsRecomposeTheBox(t *testing.T) {
	ask := &scAsk{give: StepCost{Rate: 16, Transit: 16, Adjacent: true}, ok: true}
	v := scViewer(t, ask)

	if _, _, ok := v.readoutPresent(60); !ok {
		t.Fatal("the fixture composed no readout")
	}
	first := v.readoutBuilds
	if _, _, ok := v.readoutPresent(60); !ok {
		t.Fatal("the fixture composed no readout on the second frame")
	}
	if v.readoutBuilds != first {
		t.Errorf("an unchanged frame recomposed the box: %d builds, then %d", first, v.readoutBuilds)
	}

	ask.give.Rate++
	if _, _, ok := v.readoutPresent(60); !ok {
		t.Fatal("the fixture composed no readout after the value moved")
	}
	if v.readoutBuilds != first+1 {
		t.Errorf("a moved step rate did not recompose the box: %d builds, want %d",
			v.readoutBuilds, first+1)
	}
}

// TestTheShippedLayoutCarriesBothStepRows is AC-14. The labels are read off the
// shipped layout, so a row added to the type and forgotten in the layout fails
// here rather than on screen.
func TestTheShippedLayoutCarriesBothStepRows(t *testing.T) {
	want := map[PanelField]string{PanelFieldStepRate: "MOVE", PanelFieldStepCost: "STEP"}
	seen := map[PanelField]string{}
	for _, r := range AuthoredReadoutLayout().Rows {
		if _, ours := want[r.Field]; ours {
			seen[r.Field] = r.Label
		}
	}
	for f, label := range want {
		if seen[f] != label {
			t.Errorf("field %d is labelled %q in the shipped layout, want %q", f, seen[f], label)
		}
	}
}

func TestTheStepQuestionIsNotHeldInTheSubject(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("comparing two readout subjects panicked: %v", r)
		}
	}()
	a, b := readoutFixture(), readoutFixture()
	a.HasStep, a.Step = true, StepCost{Rate: 1, Transit: 2, Adjacent: true}
	b.HasStep, b.Step = true, StepCost{Rate: 1, Transit: 2, Adjacent: true}
	if a != b {
		t.Error("two identically built subjects compare unequal")
	}
}
