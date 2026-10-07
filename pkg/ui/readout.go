package ui

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// The debug readout: a box of the running game's parameters, drawn over the
// map.
//
// IT IS OURS. Nothing published describes a parameter readout in the original
// and none was looked for: this is a development instrument, and no line of it
// asserts anything about the game. AUTHORED — a verdict, not a hold.
//
// THE RULE THE WHOLE FILE EXISTS FOR: every value stated here is read, at
// the moment it is stated, from the thing that owns it. Nothing here is
// accumulated on the path that CHANGES a value. A number kept beside an
// authority agrees with it until the first clamp, truncation or refusal and
// is silently wrong afterwards — and a wrong debug readout is worse than
// none, because it is believed over the game.
//
// For the cadence that rule has teeth, because three things in this tree are
// rate-shaped and only one is the clock. The front-end's key LADDER is the
// obvious one and is the wrong one: it is on the input path, and every clamp
// below it is invisible to it. This package cannot reach it — the ladder is an
// unexported field of the flow and a viewer holds no pointer to one — so the
// wrong answer is unrepresentable here rather than merely unwritten. The
// viewer's own WATER counter is reachable and is deliberately unused: it agrees
// today, it is re-rated through a second call site, and it keeps running while
// the world is stopped, so it is the drawing's clock and not the game's. What is
// stated is the third: the world's own ticker, arriving through Readout below.

// Readout is what the tier that owns the world hands the viewer: the period
// the world's clock is holding, its stop, and the world's own tick and
// digest.
//
// FOUR SCALARS ARE THE WHOLE PAYLOAD, for the reason every seam out of this
// package keeps: this package may not name a simulation type, and two ints and
// a bool name none.
//
// IT CARRIES A PERIOD AND NOT A RATE. The clock physically holds a period, in
// microseconds; the rate is derived from it here by terrain.RateOf, which is the
// exact inverse of the function that computed it. Carrying a rate instead would
// mean the far side deciding what to call the period, which is precisely the
// second nominal identity the clock type refuses to hold.
//
// THE ZERO VALUE IS "NOT TOLD", and every viewer that is never pushed one holds
// it: the standalone developer viewer, and every caller written before this
// story. Combined with the font gate below, such a viewer draws no readout at
// all rather than a box of zeros.
type Readout struct {
	// PeriodUS is the tick length the world's clock is holding, in
	// microseconds — the very quantity the paced advance divides elapsed time
	// by to decide how many ticks fire.
	PeriodUS int
	// Stopped is that same advance's own stop flag.
	Stopped bool
	// Tick is the world's own tick number and Digest its full-state digest.
	Tick, Digest uint64
}

// The readout's fields, in the SHARED field space.
//
// They start at 16 and the unit panel's own set ends at 3, so that closed set
// keeps room to grow without either register having to renumber. One space
// serves two boxes because PanelLayout is already "geometry, colours, and which
// field each row states in which order", which is exactly what both need; the
// alternative is a second layout type carrying the same thirteen appearance
// fields, and that duplication is what this avoids.
//
// The two resolvers are TOTAL and each answers false for the other's fields, so
// a field written into the wrong layout omits its row — which is behaviour the
// panel already specifies for a field a build does not define. A mixed layout
// therefore degrades to a shorter box and never to a wrong value.
const (
	// PanelFieldCadence is the rate the world's clock is running at, in whole
	// ticks a second, with the stop stated BESIDE it rather than as a rate of
	// zero: a stop does not change the rate, and the rate is what the world
	// resumes at, so hiding it under the stop would lose the one number the
	// two ladder keys move.
	PanelFieldCadence PanelField = 16
	// PanelFieldPeriod is that same clock's held period in microseconds. Both
	// are drawn because the period is the quantity and the rate is a
	// derivation, and because they are the only way to see a fact that is
	// otherwise invisible: a freshly opened map runs at the game's own
	// truncated 62000 us, where our rate model's own 16/s is 62500. Same
	// stated rate, different clock.
	PanelFieldPeriod PanelField = 17
	// PanelFieldSetting is WHICH OF THE GAME'S OWN NINE SPEED SETTINGS the
	// clock is running at, or how far past the ends of that set our extension
	// has taken it.
	//
	// It earns a line because the two numbers above it cannot answer the question
	// the extension raises. 62000 us at 16/s and 31250 us at 32/s are both
	// perfectly good cadences and nothing in either pair says that the first is a
	// setting the original offers and the second is one we added. A player who
	// cannot see that line will report our extension's behaviour as the game's.
	//
	// IT IS DERIVED FROM THE PERIOD, here, by the exact inverse of the function
	// that computed it — never from the key ladder, which this package cannot
	// reach and must not learn to. terrain.SpeedIndexOf reports absence rather
	// than the nearest speed, so a cadence off the shipped set can only be drawn
	// as off it.
	PanelFieldSetting PanelField = 26
	// PanelFieldTick is the world's own tick number.
	PanelFieldTick PanelField = 18
	// PanelFieldDigest is the world's full-state digest, as 16 hex digits. It
	// is the whole determinism state in one number, and it is what makes a
	// desync visible at the instant it happens rather than at the end of a run.
	PanelFieldDigest PanelField = 19
	// PanelFieldCursor is the map cell under the cursor — the GROUND PICK's own
	// answer, which is the cell a left click would order a unit to.
	//
	// On displaced ground both the lattice and the ground pick use the projected
	// corner mesh. The ground pick evaluates ROM1's exact per-column lerp while
	// the lattice draws the continuous edge, so this field remains the action's
	// answer rather than a second geometric derivation.
	PanelFieldCursor PanelField = 20
	// PanelFieldSpeed is the selected unit's own speed input, absent when
	// nothing is selected.
	PanelFieldSpeed PanelField = 21
	// PanelFieldGroup is that unit's GROUP RATE TERM, on a line of its own
	// beside the speed and never folded into it.
	//
	// While a unit carries one, THAT is the rate it moves at and its own speed
	// is inert. Two lines rather than one composed number, because a composed
	// number would show the right value and destroy the reason it is that value:
	// a unit whose own speed is 40, whose group term is 12, and which is
	// therefore crawling, is the whole diagnosis read straight off the box.
	//
	// It earns the line because the term OUTLIVES what set it. An order sets it
	// and only another order or being felled clears it — not arriving, and not
	// the slow member it came from dying — so a survivor can keep walking at a
	// dead unit's pace with nothing else on screen accounting for it.
	PanelFieldGroup PanelField = 25
	// PanelFieldCrossing is how many ticks that unit's CURRENT cell crossing
	// runs for — the span the simulation recorded when the unit took that step,
	// never a length recomputed here from the speed. A unit that has taken no
	// step, and one whose speed does not rate it, is on no such crossing, and
	// the row states that rather than a number.
	PanelFieldCrossing PanelField = 22
	// PanelFieldEntities is how many entities the frame was given to draw.
	PanelFieldEntities PanelField = 23
	// PanelFieldFrames is the engine's own frame rate, rounded to a whole
	// number. It is rounded BEFORE it reaches the refresh key: the engine's
	// measurement moves continuously, so a raw one in the key would recompose
	// the box every frame at every cadence, including a stopped one.
	PanelFieldFrames PanelField = 24
	// PanelFieldAttack is whether an ATTACK IS ARMED — the map screen's own
	// one-shot mode, waiting for the press that will spend it.
	//
	// It earns a line because the mode is otherwise INVISIBLE and one-shot
	// together, which is the worst pair a control can have: nothing on screen
	// changes when the key is pressed, and the consequence lands on the next
	// press rather than on this one. A player who cannot see it reads a refused
	// arming as a dead key, and an accidental arming as a unit attacking his own
	// side for no reason.
	//
	// It is a DIAGNOSTIC statement of front-end state and not a cursor. What is
	// being reconstructed shows the mode by swapping the pointer for its own
	// attack art, which is a drawing-tier story with real assets behind it; this
	// asserts nothing about that.
	PanelFieldAttack PanelField = 27
	// PanelFieldOrder is WHICH AIMED STANDING ORDER IS ARMED — the map
	// screen's own mode, exactly as PanelFieldAttack above is. It is APPENDED
	// after it, so no existing constant's value moves.
	//
	// It states the ARM and not the order a unit is under. What a group's
	// standing order actually is lives behind the seam and nothing pushes it
	// here; this row answers the question the player has between pressing a
	// key and pressing the button that spends it.
	PanelFieldOrder PanelField = 53

	// PanelFieldStepRate and PanelFieldStepCost are what ONE STEP ONTO THE CELL
	// UNDER THE CURSOR would cost THE SELECTED UNIT: the movement law's rate,
	// and the ticks the transit that rate produces takes.
	//
	// THE PAIR IS THE POINT. The cost plane's own byte is not what these state and
	// is not what was asked for: that number is the same for every unit standing
	// anywhere. The law is a per-step law over an ORDERED PAIR of cells, and every
	// mover-dependent term enters through it — the domain fork, the effective
	// speed, the height difference between the two cells, and the mean of BOTH
	// cells' cost bytes. Half the answer therefore comes from where the mover is
	// standing, and that is exactly what makes it his.
	//
	// BOTH ARE THE SIMULATION'S OWN ANSWER, asked for at the moment they are
	// stated and composed nowhere here — the rule this whole file exists for. This
	// package holds no movement law and must not learn one; what it holds is the
	// question.
	//
	// TWO ROWS AND NOT ONE, because the transit is the QUANTISED one. It is the
	// grid divided by the step and rounded up, so a rate of 16 and a rate of 17
	// are the same transit — the rate is where the terrain is actually visible,
	// and a box showing only the ticks would show cost and slope changes doing
	// nothing.
	//
	// The rate carries NO UNIT SUFFIX. It is a displacement in sub-cell units per
	// tick, and the denominator is a simulation constant this package would have
	// to hard-code to print — which is the one thing a seam of builtins is for
	// avoiding.
	PanelFieldStepRate PanelField = 28
	PanelFieldStepCost PanelField = 29
)

// The two markers the readout states in place of a value, and the word it puts
// in front of a stopped cadence. All three are ours; nothing decodes them.
const (
	readoutAbsent  = "-"
	readoutStopped = "STOP"

	// readoutExtended prefixes a cadence that is PAST the ends of the game's
	// shipped speed set, and the signed number after it is how many rungs past.
	// A word rather than a bare sign, because the row otherwise reads as a
	// setting number and "-1" would look like one.
	readoutExtended = "EXT"

	// The two positions of the attack mode. Both are ours; nothing decodes
	// either, and neither is the marker for an absent value — the mode is never
	// absent, it is up or down.
	readoutArmed    = "ARMED"
	readoutDisarmed = "-"
	// The aimed orders' own words. All three are the ORDERS' names and not the
	// keys', because a binding can move and this row is about what was armed.
	//
	// SWARM REPLACES "MARCH". The order this build called March was never this
	// project's own invention: `mapWorld.march(patrol=false)` queues
	// `sim.KindGroupSwarmTo`, which is the simulation's own Swarm 2 order, and
	// `AI-PANEL-123` gives Swarm mode 6 opcode `0x1a`. The word on this row was
	// the only place the invented name was visible to a player.
	readoutPatrol = "PATROL"
	readoutSwarm  = "SWARM"
	readoutMove   = "MOVE"
	readoutDefend = "DEFEND"

	// readoutFar marks a step figure whose two cells are NOT NEIGHBOURS. The
	// law is total on any ordered pair, so a distant cell has a perfectly good
	// answer and refusing one would empty the row for almost every place a
	// reader actually points; but that answer is a single HYPOTHETICAL step and
	// never the cost of walking there, and no search sits behind it. Unmarked,
	// it would be read as the mover's next step.
	//
	// A word rather than a symbol, for readoutExtended's own reason: the value is
	// a number and a bare sigil beside one reads as part of it.
	readoutFar = "FAR"
)

// StepCost is what the simulation answers about one step: the movement law's
// rate, and how many ticks the transit at that rate takes.
//
// Adjacent is whether the two cells are a single step apart AT ALL, and it is
// answered by the side that computed the figure rather than derived here — only
// that side knows which cell the law was actually evaluated from, so a marker
// derived from this package's own copy of the mover's position could mark a
// number computed about a different pair.
//
// Three ints and a bool: this package still names no simulation type.
type StepCost struct {
	Rate     int
	Transit  int
	Adjacent bool
}

// StepCostFunc answers what one step onto the cell at (col, row) would cost
// the entity id names, and whether there is an answer at all.
//
// THE QUESTION CROSSES, NOT THE ANSWER, and that is forced. The two inputs exist
// only in this package — the cell comes from the ground pick at the draw, and the
// unit from the selection — so the tier that owns a world cannot push a value the
// way it pushes the clock's period. It cannot be computed here either, because
// that would put a second copy of a movement law in the drawing tier, which is
// what the entity seam's own speed field refuses in writing.
//
// FALSE IS A REAL ANSWER AND NOT AN ERROR. The far side declines for an id it
// does not hold, for a mover that is not alive, for one carrying no effective
// speed, and for a destination equal to the source — every input on which its own
// advance would decline to rate a mover. Nothing here interprets which of them it
// was: this package states an absence, and the rule that decides one lives beside
// the law.
type StepCostFunc func(id uint32, col, row int) (StepCost, bool)

// readoutSubject is EVERYTHING the readout states, as one comparable value —
// the pushed world half embedded whole, and the view half this package answers
// for itself.
//
// It is builtins and one point, so it is copied by assignment and compared with
// ==, which is what lets the refresh rule be a comparison of two of these. That
// is the unit panel's own rule and it carries the same consequence: what the
// picture is a function of is exactly what decides when it is redrawn, so a
// value that moved the picture and is not in here would leave a stale box on
// screen until something else happened to change.
type readoutSubject struct {
	Readout

	// Cursor is the cell the ground pick resolved and OnMap whether it
	// resolved to one at all. The pick returns a zero pair outside the extent
	// and declines to promise a value there, so the bool is read first and the
	// point is untouched when it is false.
	Cursor image.Point
	OnMap  bool

	// Speed, Group and Crossing are the selected unit's, and HasUnit whether
	// there is one to state them about. All three rows are omitted together,
	// because they describe the same unit and a box stating some of them would
	// be describing part of a unit.
	Speed, Group, Crossing int
	HasUnit                bool

	// Step is what a step onto the cursor's cell would cost that same unit, and
	// HasStep whether the far side answered at all. They are UNIT FIELDS: their
	// rows come and go with the three above, because there is no such thing as
	// what a step would cost nobody.
	//
	// HasStep is false with no unit, with no cell, and for every input the far
	// side declines — so an absent Step is an absence and never a zero rate.
	Step    StepCost
	HasStep bool

	Entities int
	Frames   int

	// Armed is the map screen's attack mode. It is the view half, like Cursor
	// and unlike Tick: nothing is pushed for it, because nothing behind the
	// seam knows the mode exists.
	Armed bool

	// Aimed is which aimed standing order is armed. It is the view half beside
	// Armed, and for Armed's own reason: nothing is pushed for it, because
	// nothing behind the seam knows the arm exists.
	Aimed uint8
}

// readoutText is the ONE place a readout subject becomes a field's text, and the
// one place that says which readout fields exist at all.
//
// It reports whether the field has a value. The two unit fields have none with
// nothing selected; every other field always has one, because every other value
// exists whether or not it is interesting — a tick of zero is a world that has
// not been advanced, not an absence, and stating it is how the readout shows
// that a stopped world is stopped rather than empty.
//
// A field this resolver does not define answers false, which is what makes a
// unit-panel field written into a readout layout omit its row.
func readoutText(s readoutSubject, f PanelField) (string, bool) {
	switch f {
	case PanelFieldCadence:
		// The rate is DERIVED FROM THE PERIOD, here, at the moment it is
		// stated — never carried across the seam and never remembered.
		rate := strconv.Itoa(terrain.RateOf(s.PeriodUS)) + "/s"
		if s.Stopped {
			return readoutStopped + " " + rate, true
		}
		return rate, true
	case PanelFieldPeriod:
		return strconv.Itoa(s.PeriodUS) + "us", true
	case PanelFieldSetting:
		if i, ok := terrain.SpeedIndexOf(s.PeriodUS); ok {
			return strconv.Itoa(i+1) + "/" + strconv.Itoa(terrain.SpeedIndexMax+1), true
		}
		// Past an end of the shipped set — or on no rung at all, which is what a
		// clock re-rated by something other than the keys can be. The second
		// states the marker rather than a distance, because there is no rung to
		// measure a distance from and inventing the nearest is the substitution
		// this row exists to make visible.
		rung := terrain.CadenceRung(s.PeriodUS)
		if terrain.CadencePeriod(rung) != s.PeriodUS {
			return readoutAbsent, true
		}
		if rung < terrain.CadenceShippedLo {
			return readoutExtended + " " + strconv.Itoa(rung-terrain.CadenceShippedLo), true
		}
		return readoutExtended + " +" + strconv.Itoa(rung-terrain.CadenceShippedHi), true
	case PanelFieldTick:
		return strconv.FormatUint(s.Tick, 10), true
	case PanelFieldDigest:
		return fmt.Sprintf("%016x", s.Digest), true
	case PanelFieldCursor:
		if !s.OnMap {
			return readoutAbsent, true
		}
		return fmt.Sprintf("%d, %d", s.Cursor.X, s.Cursor.Y), true
	case PanelFieldSpeed:
		if !s.HasUnit {
			return "", false
		}
		return strconv.Itoa(s.Speed), true
	case PanelFieldGroup:
		if !s.HasUnit {
			return "", false
		}
		// Zero is a unit carrying no group term, which is an absence and not a
		// rate of nothing: its own speed is what moves it. A "0" here would
		// read as a group that had stopped it.
		if s.Group == 0 {
			return readoutAbsent, true
		}
		return strconv.Itoa(s.Group), true
	case PanelFieldCrossing:
		if !s.HasUnit {
			return "", false
		}
		// A span of zero is a unit on no recorded crossing — one that has
		// taken no step, and one whose speed does not rate it. It is an
		// absence and is stated as one; a "0 t" would read as a unit crossing
		// a cell in no time at all.
		if s.Crossing <= 0 {
			return readoutAbsent, true
		}
		return strconv.Itoa(s.Crossing) + " t", true
	case PanelFieldStepRate, PanelFieldStepCost:
		// BOTH ROWS GO WITH THE UNIT, like the three above: what a step costs is
		// a fact about a mover, and a box stating it with nothing selected would
		// be stating it about nobody.
		if !s.HasUnit {
			return "", false
		}
		// Present but unanswerable — no cell under the cursor, the unit's own
		// cell, an unrated mover, or a viewer that was handed no question. The
		// row stays and states the absence, because a row that vanished would
		// make "there is nothing to say" and "this build has no such row" the
		// same picture.
		if !s.HasStep {
			return readoutAbsent, true
		}
		val := strconv.Itoa(s.Step.Rate)
		if f == PanelFieldStepCost {
			val = strconv.Itoa(s.Step.Transit) + " t"
		}
		if !s.Step.Adjacent {
			val += " " + readoutFar
		}
		return val, true
	case PanelFieldEntities:
		return strconv.Itoa(s.Entities), true
	case PanelFieldFrames:
		return strconv.Itoa(s.Frames), true
	case PanelFieldAttack:
		// BOTH STATES ARE STATED, and neither is an absence. "Armed" and "not
		// armed" are two positions of one control, so a row that vanished when
		// the mode was down would make "the key did nothing" and "the readout
		// has no such row" the same picture — which is exactly the confusion
		// the row exists to remove.
		if s.Armed {
			return readoutArmed, true
		}
		return readoutDisarmed, true
	case PanelFieldOrder:
		// EVERY STATE IS STATED AND NONE IS AN ABSENCE, on the attack row's own
		// grounds: a row that vanished when nothing was armed would make "the
		// key did nothing" and "the readout has no such row" the same picture,
		// which is exactly the confusion this row exists to remove.
		switch s.Aimed {
		case commandPatrol:
			return readoutPatrol, true
		case commandSwarm:
			return readoutSwarm, true
		case commandMove:
			return readoutMove, true
		case commandDefend:
			return readoutDefend, true
		}
		return readoutDisarmed, true
	}
	return "", false
}

// AuthoredReadoutLayout is what this project ships: a box in the top left, in
// the unit panel's own colours, whose unit rows come and go with the selection.
//
// EVERY VALUE IN IT IS OURS — the corner, the order, the wording of every label,
// the gap and the width. No source ranks these fields or names them.
//
// THE CORNER IS THE OTHER ONE. The unit panel is anchored bottom left; two
// boxes anchored to different corners of one window cannot overlap unless
// their combined heights exceed it, which for these two at this font is not
// reachable at any window this game opens. Top left is also simply free: the
// map screen draws nothing else in window coordinates.
//
// It is handed out as a fresh value rather than held in a package variable, for
// the unit panel's own reason: a variable carrying a row slice is writable from
// anywhere, and "two viewers show the same readout" would then be true by
// mutation rather than by design.
func AuthoredReadoutLayout() PanelLayout {
	return PanelLayout{
		Corner:   PanelTopLeft,
		Margin:   image.Pt(12, 12),
		MinWidth: 168,
		Pad:      image.Pt(10, 8),
		Gap:      2,
		LabelGap: 8,
		Flow:     true,

		Fill:       color.RGBA{R: 0x10, G: 0x12, B: 0x18, A: 0xff},
		Border:     color.RGBA{R: 0x8a, G: 0x74, B: 0x46, A: 0xff},
		LabelColor: color.RGBA{R: 0x8f, G: 0x9a, B: 0xa8, A: 0xff},
		ValueColor: color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff},

		Rows: []PanelRow{
			{Field: PanelFieldCadence, Label: "RATE"},
			{Field: PanelFieldPeriod, Label: "PERIOD"},
			{Field: PanelFieldSetting, Label: "SETTING"},
			{Field: PanelFieldTick, Label: "TICK"},
			{Field: PanelFieldDigest, Label: "HASH"},
			{Field: PanelFieldCursor, Label: "CURSOR"},
			{Field: PanelFieldSpeed, Label: "SPEED"},
			{Field: PanelFieldGroup, Label: "GROUP"},
			{Field: PanelFieldCrossing, Label: "CROSS"},
			{Field: PanelFieldStepRate, Label: "MOVE"},
			{Field: PanelFieldStepCost, Label: "STEP"},
			{Field: PanelFieldEntities, Label: "ENTS"},
			{Field: PanelFieldFrames, Label: "FPS"},
			{Field: PanelFieldAttack, Label: "ATTACK"},
			{Field: PanelFieldOrder, Label: "ORDER"},
		},
	}
}

// readoutKey is everything the presented picture is a function of: what is
// stated, which layout and font, and how big the area it is placed in is.
//
// The area is here because a box anchored to a corner moves when the window
// resizes, and the serial because a layout is not comparable.
type readoutKey struct {
	subject readoutSubject
	serial  int
	area    image.Point
}

// SetReadout adopts the world half of what the readout states.
//
// It is called by the tier that owns the world, from the very fields that
// tier's advance reads, and it REPLACES rather than merges. A push that
// stopped happening would freeze the readout visibly rather than let it
// drift plausibly.
//
// It writes NOTHING ELSE — no camera, no selection, no clock.
func (v *Viewer) SetReadout(r Readout) { v.readout = r }

// ReadoutState reports the world half last pushed, mirroring Phase's shape and
// reason: the tier that pushes needs to see that its value arrived without
// opening a window.
func (v *Viewer) ReadoutState() Readout { return v.readout }

// SetStepCost adopts the question the readout asks about a step.
//
// It is INSTALLED ONCE and not pushed per frame, because it is a capability and
// not a value: what it answers changes every tick, but that it can answer does
// not, and a closure allocated per frame would be garbage for a fact that never
// moves. It REPLACES, like SetReadout, and nil takes the question away again.
//
// It writes NOTHING ELSE — no camera, no selection, no clock.
func (v *Viewer) SetStepCost(f StepCostFunc) { v.stepCost = f }

// StepCostAt asks the installed question, and reports none for a viewer holding
// none — ReadoutState's twin and for exactly its reason: the tier that installs
// the question needs to see that it arrived, and what it answers, without
// opening a window.
//
// It is a pass-through and interprets nothing. In particular it does NOT supply
// the readout's own two conditions — a selected unit and a resolved cell — so it
// is not a second route to what the box states; it is a view of the seam.
func (v *Viewer) StepCostAt(id uint32, col, row int) (StepCost, bool) {
	if v.stepCost == nil {
		return StepCost{}, false
	}
	return v.stepCost(id, col, row)
}

// ShowReadout shows or hides the readout.
//
// THE FLAG IS STORED INVERTED so that SHOWN is the zero value. A default set
// in a constructor is a default every struct literal in a suite gets wrong;
// a default that IS the zero value is one nothing has to know about. What
// actually gates the drawing is the font, unchanged from the unit panel —
// a viewer holding none draws no readout — which is why every existing
// viewer, the standalone one included, is byte-identical without a single
// call being added to it.
func (v *Viewer) ShowReadout(show bool) { v.readoutHidden = !show }

// ToggleReadout flips the readout and reports where it landed, mirroring
// ToggleGrid: the caller is a key press that has no state of its own to consult.
func (v *Viewer) ToggleReadout() bool {
	v.readoutHidden = !v.readoutHidden
	return !v.readoutHidden
}

// ReadoutShown reports whether the readout is being drawn. The tier that
// pushes reads it to decide whether the world's digest is worth computing at
// all.
func (v *Viewer) ReadoutShown() bool { return !v.readoutHidden }

// readoutSubjectOf is what the readout states this frame: the pushed world half,
// the cursor's cell through the ground pick, the selected unit's two numbers,
// the entity count and the frame rate handed in.
//
// THE SELECTED UNIT COMES THROUGH presentSelected — CALLED, not copied. The
// marks a frame draws, the orders a left click emits, the blows the two keys
// land, the unit panel and now this are the same units, and a sixth reading of
// "still there to act on" would be a sixth chance to disagree. It takes the
// first element for the panel's own reason: that filter emits in the selection's
// ascending order, so "first" is that order's first and not a property of the
// walk, and the unit the readout describes is the unit the panel describes.
//
// The frame rate arrives as an argument rather than being read here, so this
// whole function is reachable with no engine: the one engine call lives at the
// single call site in Draw.
func (v *Viewer) readoutSubjectOf(frames int) readoutSubject {
	// The armed row reads the PREDICATE, so the box states the mode whichever
	// of its two surfaces raised it. A row that answered the key's field alone
	// would read "not armed" with the modifier held and a sword on screen.
	s := readoutSubject{Readout: v.readout, Entities: len(v.entities), Frames: frames, Armed: v.attackMode(), Aimed: v.aimedOrder()}

	if v.hasCursor {
		// The bool is read FIRST. Outside the extent the pick returns 0, 0 and
		// declines to promise a value there, so the point stays untouched
		// unless it is inside — the same discipline the order path keeps.
		if col, row, inside := v.groundCellAt(float64(v.cursorX), float64(v.cursorY)); inside {
			s.Cursor, s.OnMap = image.Pt(col, row), true
		}
	}

	if present := presentSelected(v.sel, v.entities); len(present) > 0 {
		e := present[0]
		s.Speed, s.Group, s.Crossing, s.HasUnit = e.Speed, e.GroupSpeed, e.TransitSpan, true

		// THE QUESTION IS ASKED HERE, at the moment the value is stated, from the
		// thing that owns the answer — and only when there is both a unit to ask
		// about and a cell to ask about. A viewer holding no question answers
		// none, which is the same absence as the other three causes and is stated
		// the same way.
		if s.OnMap && v.stepCost != nil {
			s.Step, s.HasStep = v.stepCost(e.ID, s.Cursor.X, s.Cursor.Y)
		}
	}
	return s
}

// readoutPresent is the picture to draw this frame and where its top-left corner
// goes, or false for a frame that draws no readout at all.
//
// The rebuild rule is the key comparison and nothing else, exactly as the
// panel's is. Two of the stated values — the tick and the digest —
// change on every logic tick, so this recomposes at min(frame rate, tick
// rate): sixteen times a second at the map-load cadence, at most once a
// frame at the top of the ladder. That is the honest ceiling and it is
// measured, not asserted.
func (v *Viewer) readoutPresent(frames int) (*image.RGBA, image.Point, bool) {
	if v.readoutHidden || v.font == nil {
		return nil, image.Point{}, false
	}
	s := v.readoutSubjectOf(frames)
	area := image.Pt(v.cam.ViewW, v.cam.ViewH)
	key := readoutKey{subject: s, serial: v.readoutSerial, area: area}
	if v.readoutPic == nil || key != v.readoutKey {
		// ONE STATEMENT, so a fresh picture can never be presented under a
		// texture holding the last one: what the upload reads and the flag that
		// says it must are written together or not at all.
		var pic *image.RGBA
		v.readoutText = text.Record(func() {
			pic = composeItems(v.readoutLayout, v.font, readoutItems(v.readoutLayout, s))
		})
		v.readoutPic, v.readoutFresh = pic, true
		v.readoutKey = key
		v.readoutBuilds++
	}
	if v.readoutPic == nil {
		return nil, image.Point{}, false
	}
	text.Append(v.readoutText, 0, 0)
	return v.readoutPic, panelOrigin(v.readoutLayout, area, v.readoutPic.Bounds().Size()), true
}

// readoutItems resolves the layout's rows against the subject in row order,
// dropping every row whose field has no value — panelItems' twin, over the other
// resolver.
func readoutItems(l PanelLayout, s readoutSubject) []panelItem {
	out := make([]panelItem, 0, len(l.Rows))
	for _, r := range l.Rows {
		val, ok := readoutText(s, r.Field)
		if !ok {
			continue
		}
		out = append(out, panelItem{label: r.Label, value: val, at: r.At})
	}
	return out
}
