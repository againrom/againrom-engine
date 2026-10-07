package ui

import (
	"bytes"
	"image"
	"strconv"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// The debug readout (AC-1, AC-8, AC-9, AC-10, AC-11, AC-12). Every fixture
// is built here; nothing reads a game install.

func readoutViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("readout", grid(60, 60), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetFont(panelFont())
	return v
}

func readoutFixture() readoutSubject {
	return readoutSubject{
		Readout:  Readout{PeriodUS: 62_500, Tick: 1234, Digest: 0xdeadbeefcafef00d},
		Cursor:   image.Pt(12, 34),
		OnMap:    true,
		Speed:    35,
		Crossing: 6,
		HasUnit:  true,
		Entities: 41,
		Frames:   60,
	}
}

// textOf is the readout's text for one field, or a marker for a field it
// declines. Written out rather than asserted inline so a table can read.
func textOf(s readoutSubject, f PanelField) string {
	if v, ok := readoutText(s, f); ok {
		return v
	}
	return "<absent>"
}

// TestReadoutStatesEveryFieldOfTheTable — 0060 SC-1: every field either
// states a value or is omitted, and the value is the one the spec's table
// promises.
//
// The cadence is the row this story exists for and it is checked at a period our
// own rate model produced (62500 -> 16/s) AND at the game's own map-load period
// (62000 -> 16/s), because those are two different clocks that state the same
// rate — which is the whole reason the period is drawn beside it.
func TestReadoutStatesEveryFieldOfTheTable(t *testing.T) {
	s := readoutFixture()
	for _, c := range []struct {
		field PanelField
		want  string
	}{
		{PanelFieldCadence, "16/s"},
		{PanelFieldPeriod, "62500us"},
		// The fixture's own 62500 us is our old rate model's 16/s and is on NO
		// rung of the cadence ladder, so the setting row states the absence marker
		// rather than the nearest speed.
		{PanelFieldSetting, "-"},
		{PanelFieldTick, "1234"},
		{PanelFieldDigest, "deadbeefcafef00d"},
		{PanelFieldCursor, "12, 34"},
		{PanelFieldSpeed, "35"},
		{PanelFieldCrossing, "6 t"},
		{PanelFieldEntities, "41"},
		{PanelFieldFrames, "60"},
	} {
		if got := textOf(s, c.field); got != c.want {
			t.Errorf("field %d states %q, want %q", c.field, got, c.want)
		}
	}

	// The game's own map-load period reads the same rate over a different clock.
	game := s
	game.PeriodUS = terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)
	if got := textOf(game, PanelFieldCadence); got != "16/s" {
		t.Errorf("the map-load clock states %q, want 16/s", got)
	}
	if got := textOf(game, PanelFieldPeriod); got != "62000us" {
		t.Errorf("the map-load period states %q, want 62000us — the game's own 62 ms", got)
	}
	if got := textOf(game, PanelFieldSetting); got != "5/9" {
		t.Errorf("the map-load cadence states setting %q, want 5/9 — speed index 4 of the shipped nine", got)
	}

	// The stop is stated BESIDE the rate and never as a rate of zero: a stop
	// does not change the rate, and the rate is what the world resumes at.
	stopped := s
	stopped.Stopped = true
	if got := textOf(stopped, PanelFieldCadence); got != "STOP 16/s" {
		t.Errorf("a stopped cadence states %q, want the stop beside the unchanged rate", got)
	}

	// A unit-panel field written into a readout layout omits its row rather
	// than drawing an empty one, and vice versa.
	if got := textOf(s, PanelFieldHealth); got != "<absent>" {
		t.Errorf("the readout resolved a unit-panel field as %q", got)
	}
	if _, ok := panelText(PanelSubject{}, PanelFieldDigest); ok {
		t.Error("the unit panel resolved a readout field")
	}
}

// TestReadoutOmitsWhatItCannotState — 0060 SC-9 (AC-8, AC-12).
//
// Three absences, three different answers, and they are deliberately not one:
// the two unit rows are OMITTED because a box describing half a unit describes
// nothing, while the cursor and an unrated crossing state a MARKER because the
// question was asked and has a negative answer.
func TestReadoutOmitsWhatItCannotState(t *testing.T) {
	bare := readoutSubject{}
	for _, f := range []PanelField{PanelFieldSpeed, PanelFieldCrossing} {
		if _, ok := readoutText(bare, f); ok {
			t.Errorf("field %d stated something with nothing selected", f)
		}
	}
	if got := textOf(bare, PanelFieldCursor); got != readoutAbsent {
		t.Errorf("a cursor off the map states %q, want %q", got, readoutAbsent)
	}

	// A unit on no recorded crossing: it has taken no step, or its speed does
	// not rate it. Zero ticks a cell would read as a unit crossing in no time.
	unrated := readoutSubject{HasUnit: true, Speed: 0, Crossing: 0}
	if got := textOf(unrated, PanelFieldCrossing); got != readoutAbsent {
		t.Errorf("an unrated crossing states %q, want %q", got, readoutAbsent)
	}
	if got := textOf(unrated, PanelFieldSpeed); got != "0" {
		t.Errorf("an unrated speed states %q, want 0 — a speed of zero is a value", got)
	}

	// The box is shorter by exactly the omitted unit rows, which is what makes
	// "a dropped row costs no space" observable here rather than asserted.
	//
	// FIVE since 0084: the two step rows are unit rows and come and go with the
	// other three. The count is stated rather than derived on purpose — a
	// requirement that says "exactly these rows" is only worth anything if adding
	// a sixth has to come here and say so.
	l := AuthoredReadoutLayout()
	full := len(readoutItems(l, readoutFixture()))
	none := len(readoutItems(l, readoutSubject{}))
	if full-none != 5 {
		t.Errorf("selecting a unit adds %d rows, want exactly 5", full-none)
	}
}

// TestReadoutIsShownByDefault — 0060 AC-1: shown is the ZERO VALUE, so a
// viewer built any way at all is showing it.
//
// The struct literal is the assertion, not a convenience: a default set in a
// constructor would leave every literal in this suite hidden, and this is what
// says the default does not depend on which constructor ran.
func TestReadoutIsShownByDefault(t *testing.T) {
	if !(&Viewer{}).ReadoutShown() {
		t.Error("a zero viewer hides the readout; shown must be the zero value")
	}
	v := readoutViewer(t)
	if !v.ReadoutShown() {
		t.Error("a constructed viewer hides the readout")
	}
	if v.ToggleReadout() || v.ReadoutShown() {
		t.Error("the first toggle did not hide it")
	}
	if !v.ToggleReadout() || !v.ReadoutShown() {
		t.Error("the second toggle did not show it again")
	}
	v.ShowReadout(false)
	if v.ReadoutShown() {
		t.Error("ShowReadout(false) left it shown")
	}
}

// TestHiddenReadoutComposesNothing — 0060 SC-6: hidden costs nothing, and
// the test is the composition COUNTER rather than the pixels, because a box
// that was composed and then not drawn looks identical to one that was never
// composed.
func TestHiddenReadoutComposesNothing(t *testing.T) {
	v := readoutViewer(t)
	v.SetReadout(Readout{PeriodUS: 62_500, Tick: 7})
	v.ShowReadout(false)

	for i := 0; i < 5; i++ {
		v.SetReadout(Readout{PeriodUS: 62_500, Tick: uint64(i)})
		if _, _, ok := v.readoutPresent(60); ok {
			t.Fatal("a hidden readout presented a picture")
		}
	}

	if v.readoutBuilds != 0 {
		t.Errorf("a hidden readout composed %d pictures, want 0", v.readoutBuilds)
	}
}

// TestReadoutNeedsAFont — 0060 AC-11: the unit panel's own gate, which is
// what leaves the standalone developer viewer — handed no font and owning
// no world — drawing exactly the frame it drew before this story.
func TestReadoutNeedsAFont(t *testing.T) {
	v, err := NewViewer("nofont", grid(60, 60), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetReadout(Readout{PeriodUS: 62_500, Tick: 3})
	if !v.ReadoutShown() {
		t.Fatal("the readout is hidden; this test would pass for the wrong reason")
	}
	if _, _, ok := v.readoutPresent(60); ok {
		t.Error("a viewer with no font presented a readout")
	}
	if v.readoutBuilds != 0 {
		t.Errorf("a viewer with no font composed %d pictures, want 0", v.readoutBuilds)
	}
}

func TestReadoutRecomposesOnlyWhenTheStatedValuesChange(t *testing.T) {
	v := readoutViewer(t)
	v.SetReadout(Readout{PeriodUS: 62_500, Tick: 10, Digest: 99})

	first, at, ok := v.readoutPresent(60)
	if !ok {
		t.Fatal("no readout on the first frame")
	}
	builds := v.readoutBuilds
	if builds != 1 {
		t.Fatalf("the first frame composed %d pictures, want 1", builds)
	}

	again, at2, ok := v.readoutPresent(60)
	if !ok {
		t.Fatal("no readout on the second frame")
	}
	if v.readoutBuilds != builds {
		t.Errorf("an unchanged frame recomposed: %d builds, want %d", v.readoutBuilds, builds)
	}
	if again != first || at2 != at {
		t.Error("an unchanged frame presented a different picture or a different origin")
	}

	// A new tick is a new picture. This is the honest cost: two of the stated
	// values move on every logic tick.
	v.SetReadout(Readout{PeriodUS: 62_500, Tick: 11, Digest: 99})
	v.readoutPresent(60)
	if v.readoutBuilds != builds+1 {
		t.Errorf("a new tick composed %d pictures, want %d", v.readoutBuilds, builds+1)
	}
	v.readoutPresent(59)
	if v.readoutBuilds != builds+2 {
		t.Errorf("a new frame rate composed %d pictures, want %d", v.readoutBuilds, builds+2)
	}
}

// TestReadoutAndPanelDoNotMeet — 0060 SC-8 (AC-10): the two boxes share no
// pixel, and the unit panel's own picture is identical whether the readout
// is shown or hidden.
func TestReadoutAndPanelDoNotMeet(t *testing.T) {
	v := readoutViewer(t)
	v.cam.ViewW, v.cam.ViewH = 1024, 768
	v.SetReadout(Readout{PeriodUS: 62_500, Tick: 5, Digest: 0x0123456789abcdef})
	v.entities = []MapEntity{{ID: 1, Name: "Warrior", HP: 63, MaxHP: 100,
		Cell: image.Pt(12, 34), Speed: 35, TransitSpan: 6}}
	v.sel = selection{1}

	panelPic, panelAt, ok := v.panelPresent()
	if !ok {
		t.Fatal("no unit panel to compare against")
	}
	panelPix := append([]uint8(nil), panelPic.Pix...)
	panelBox := image.Rectangle{Min: panelAt, Max: panelAt.Add(panelPic.Bounds().Size())}

	readPic, readAt, ok := v.readoutPresent(60)
	if !ok {
		t.Fatal("no readout")
	}
	readBox := image.Rectangle{Min: readAt, Max: readAt.Add(readPic.Bounds().Size())}

	if readBox.Overlaps(panelBox) {
		t.Errorf("the readout at %v overlaps the unit panel at %v", readBox, panelBox)
	}

	// The panel is untouched by the readout being drawn: same picture, same
	// place, and no recomposition provoked.
	builds := v.panelBuilds
	v.ShowReadout(false)
	v.readoutPresent(60)
	again, againAt, ok := v.panelPresent()
	if !ok || againAt != panelAt || !bytes.Equal(again.Pix, panelPix) {
		t.Error("hiding the readout changed the unit panel's picture or its place")
	}
	if v.panelBuilds != builds {
		t.Errorf("the readout provoked %d panel recompositions", v.panelBuilds-builds)
	}
}

// TestReadoutCursorIsTheGroundPicksOwnAnswer — 0060 SC-9 (AC-12).
//
// The cell stated is the cell a map click would order a unit to, taken through
// the displaced ground picker. The fixture's literal corner mesh selects row 0
// at (1,80), while the flat camera lattice finds no cell there and the replaced
// mean-of-four picker selects row 1. The expectation below is therefore
// independent of all three production helpers and kills either old route.
func TestReadoutCursorIsTheGroundPicksOwnAnswer(t *testing.T) {
	v := groundPickSlopeViewer(t)

	// Never stepped: nobody has pointed anywhere, and (0, 0) is a real cell on
	// every map, so the readout must not claim it.
	if got := textOf(v.readoutSubjectOf(60), PanelFieldCursor); got != readoutAbsent {
		t.Errorf("an unstepped viewer states the cursor cell %q, want %q", got, readoutAbsent)
	}

	ask := &scAsk{give: StepCost{Rate: 8, Transit: 32, Adjacent: true}, ok: true}
	v.entities = []MapEntity{{ID: 7, Cell: image.Pt(1, 1), Speed: 16, TransitSpan: 16}}
	v.sel = selection{7}
	v.SetStepCost(ask.fn())
	v.step(Input{CursorX: 1, CursorY: 80}, time.Now())

	s := v.readoutSubjectOf(60)
	if !s.OnMap || s.Cursor != image.Pt(0, 0) {
		t.Errorf("readout cursor = (%v,%v), want ((0,0),true) from the literal corner mesh", s.Cursor, s.OnMap)
	}
	if got := textOf(s, PanelFieldCursor); got != "0, 0" {
		t.Errorf("cursor row states %q, want %q", got, "0, 0")
	}
	if ask.calls != 1 || ask.id != 7 || ask.col != 0 || ask.row != 0 {
		t.Errorf("step-cost question = (calls %d, id %d, cell %d,%d), want (1,7,0,0)",
			ask.calls, ask.id, ask.col, ask.row)
	}
	if !s.HasStep || s.Step != ask.give {
		t.Errorf("step-cost answer = (%+v,%v), want (%+v,true)", s.Step, s.HasStep, ask.give)
	}
}

func TestReadoutUnitLinesComeFromTheSelectedEntry(t *testing.T) {
	v := readoutViewer(t)
	v.entities = []MapEntity{
		{ID: 1, Speed: 11, TransitSpan: 2},
		{ID: 2, Speed: 35, TransitSpan: 6},
		{ID: 3, Speed: 99, TransitSpan: 9, Life: LifeDead},
	}

	if s := v.readoutSubjectOf(60); s.HasUnit {
		t.Error("nothing is selected and yet a unit was stated")
	}

	v.sel = selection{2}
	s := v.readoutSubjectOf(60)
	if !s.HasUnit || s.Speed != 35 || s.Crossing != 6 {
		t.Errorf("selected unit 2 states speed %d crossing %d, want 35 and 6", s.Speed, s.Crossing)
	}

	// A dead id is not present, exactly as it is not present to the marks, the
	// orders or the panel.
	v.sel = selection{3}
	if s := v.readoutSubjectOf(60); s.HasUnit {
		t.Error("a dead selected id was stated")
	}

	// The first of several, in the selection's own order — the unit the panel
	// describes, so the two boxes cannot describe different units.
	v.sel = selection{1, 2}
	s = v.readoutSubjectOf(60)
	ps, ok := v.panelSubject()
	if !ok || !s.HasUnit || ps.ID != 1 || s.Speed != 11 {
		t.Errorf("the readout describes speed %d while the panel describes id %d", s.Speed, ps.ID)
	}

	// The entity count is what the frame was GIVEN to draw, corpses included:
	// it answers "how much is on this map", not "how much is alive".
	if s.Entities != 3 {
		t.Errorf("the readout counts %d entities, want the 3 it was given", s.Entities)
	}
}

// TestReadoutWritesNothing — 0060 SC-8: a frame that drew the readout
// leaves world, selection, camera and clock where a frame that hid it leaves
// them.
func TestReadoutWritesNothing(t *testing.T) {
	v := readoutViewer(t)
	v.cam.ViewW, v.cam.ViewH = 800, 600
	v.step(Input{CursorX: 100, CursorY: 100}, time.Now())
	v.entities = []MapEntity{{ID: 1, Speed: 35, TransitSpan: 6}}
	v.sel = selection{1}
	pushed := Readout{PeriodUS: 62_500, Tick: 42, Digest: 7}
	v.SetReadout(pushed)

	before := struct {
		cam      camView
		sel      selection
		readout  Readout
		anim     uint32
		period   int
		cursorAt image.Point
	}{camOf(v), append(selection(nil), v.sel...), v.ReadoutState(),
		v.anim.Count(), v.anim.Period(), image.Pt(v.cursorX, v.cursorY)}

	for i := 0; i < 3; i++ {
		if _, _, ok := v.readoutPresent(60); !ok {
			t.Fatal("no readout")
		}
	}

	if camOf(v) != before.cam {
		t.Error("drawing the readout moved the camera")
	}
	if len(v.sel) != len(before.sel) || (len(v.sel) > 0 && v.sel[0] != before.sel[0]) {
		t.Error("drawing the readout changed the selection")
	}
	if v.ReadoutState() != before.readout {
		t.Error("drawing the readout wrote back to what it was pushed")
	}
	if v.anim.Count() != before.anim || v.anim.Period() != before.period {
		t.Error("drawing the readout moved a clock")
	}
	if got := image.Pt(v.cursorX, v.cursorY); got != before.cursorAt {
		t.Error("drawing the readout moved the cursor it read")
	}
}

// camView is the comparable part of the camera's state, for the assertion above.
type camView struct {
	X, Y, Zoom          float64
	ViewW, ViewH        int
	Cols, Rows          int
	WorldW, WorldHeight float64
}

func camOf(v *Viewer) camView {
	return camView{X: v.cam.X, Y: v.cam.Y, Zoom: v.cam.Zoom,
		ViewW: v.cam.ViewW, ViewH: v.cam.ViewH, Cols: v.cam.Cols, Rows: v.cam.Rows}
}

// TestReadoutStatesTheGroupRateBesideTheSpeed — 0060 SC-10 (AC-13).
//
// The fixture's two numbers DIFFER on purpose, and they differ the way the live
// defect makes them differ: a fast unit dragged down to a group's pace. A
// readout that folded them into one effective number, or that stated one for the
// other, fails here — and so does one that leaves the group row out.
func TestReadoutStatesTheGroupRateBesideTheSpeed(t *testing.T) {
	s := readoutFixture()
	s.Speed, s.Group = 40, 12

	if got := textOf(s, PanelFieldSpeed); got != "40" {
		t.Errorf("the speed row states %q, want the unit's own 40", got)
	}
	if got := textOf(s, PanelFieldGroup); got != "12" {
		t.Errorf("the group row states %q, want the group's 12", got)
	}

	// A unit carrying no term: its own speed moves it, and the row says the
	// term is absent rather than stating a rate of nothing.
	alone := s
	alone.Group = 0
	if got := textOf(alone, PanelFieldGroup); got != readoutAbsent {
		t.Errorf("a unit with no group term states %q, want %q", got, readoutAbsent)
	}
	if got := textOf(alone, PanelFieldSpeed); got != "40" {
		t.Errorf("dropping the group term changed the speed row to %q", got)
	}

	// Every unit row follows the selection together — five of them since 0084.
	none := readoutSubject{}
	for _, f := range []PanelField{PanelFieldSpeed, PanelFieldGroup, PanelFieldCrossing,
		PanelFieldStepRate, PanelFieldStepCost} {
		if _, ok := readoutText(none, f); ok {
			t.Errorf("field %d stated something with nothing selected", f)
		}
	}
	l := AuthoredReadoutLayout()
	if full, bare := len(readoutItems(l, s)), len(readoutItems(l, none)); full-bare != 5 {
		t.Errorf("selecting a unit adds %d rows, want exactly 5", full-bare)
	}
}

// TestReadoutGroupRateComesFromTheEntry — 0060 SC-10: the term crosses the
// seam whole and is composed with the speed nowhere on this side.
func TestReadoutGroupRateComesFromTheEntry(t *testing.T) {
	v := readoutViewer(t)
	v.entities = []MapEntity{{ID: 1, Speed: 40, GroupSpeed: 12, TransitSpan: 5}}
	v.sel = selection{1}

	s := v.readoutSubjectOf(60)
	if s.Speed != 40 || s.Group != 12 {
		t.Errorf("the readout states speed %d group %d, want the entry's 40 and 12", s.Speed, s.Group)
	}
	if s.Speed == s.Group {
		t.Fatal("the fixture's two numbers must differ for this to discriminate")
	}
}

// TestTheReadoutSaysWhichSideOfTheShippedSetItIsOn — 0061 SC-5 (AC-7).
//
// Every rung of the ladder, and three periods that are on none of it. The
// expected text is written out by hand: the nine shipped settings state which of
// the nine they are, the eight rungs of our extension state how far past an end
// they are, and a period no rung produced states the absence marker rather than
// being snapped to the nearest setting.
//
// A row that read the KEY LADDER instead of the period could not answer the last
// group at all, which is why the off-ladder cases are here rather than only the
// reachable ones.
func TestTheReadoutSaysWhichSideOfTheShippedSetItIsOn(t *testing.T) {
	for _, c := range []struct {
		periodUS int
		want     string
	}{
		{1_000_000, "EXT -3"}, {500_000, "EXT -2"}, {250_000, "EXT -1"},
		{125_000, "1/9"}, {100_000, "2/9"}, {83_000, "3/9"}, {71_000, "4/9"},
		{62_000, "5/9"}, {50_000, "6/9"}, {41_000, "7/9"}, {35_000, "8/9"},
		{31_000, "9/9"},
		{15_625, "EXT +1"}, {7_812, "EXT +2"}, {3_906, "EXT +3"},
		{1_953, "EXT +4"}, {976, "EXT +5"},

		// On no rung: our old rate model's 16/s, a period between two shipped
		// settings, and one faster than the whole ladder.
		{62_500, "-"}, {40_000, "-"}, {500, "-"},
	} {
		s := readoutSubject{Readout: Readout{PeriodUS: c.periodUS}}
		if got := textOf(s, PanelFieldSetting); got != c.want {
			t.Errorf("%d us states setting %q, want %q", c.periodUS, got, c.want)
		}
	}

	// The nine shipped rows really are the nine shipped speeds, compared against
	// the speed table's own periods rather than against the literals above.
	for i := terrain.SpeedIndexMin; i <= terrain.SpeedIndexMax; i++ {
		s := readoutSubject{Readout: Readout{PeriodUS: terrain.SpeedIndexPeriod(i)}}
		want := strconv.Itoa(i+1) + "/9"
		if got := textOf(s, PanelFieldSetting); got != want {
			t.Errorf("speed index %d (%d us) states %q, want %q", i, terrain.SpeedIndexPeriod(i), got, want)
		}
	}

	// The row is in the shipped layout, and it is drawn whatever the cadence —
	// there is no arm on which the box simply omits the question.
	l := AuthoredReadoutLayout()
	found := false
	for _, r := range l.Rows {
		if r.Field == PanelFieldSetting {
			found = true
		}
	}
	if !found {
		t.Error("the shipped readout layout has no setting row")
	}
	for _, p := range []int{62_000, 62_500, 976} {
		if _, ok := readoutText(readoutSubject{Readout: Readout{PeriodUS: p}}, PanelFieldSetting); !ok {
			t.Errorf("%d us omits the setting row; every cadence has an answer", p)
		}
	}
}
