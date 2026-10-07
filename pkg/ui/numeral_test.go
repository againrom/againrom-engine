package ui

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// The floating damage numeral (0083). Everything below drives the viewer's own
// step with a hand-made timestamp and hand-made entities, so no window, no clock
// and no world is needed to say what a blow leaves behind.

const numeralGrid = 24

// numeralViewer is a viewer over a flat grid with no altitude layer, so the
// placement arithmetic is the flat one and no relief lift enters. The camera is
// left where the constructor put it.
func numeralViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("numerals", grid(numeralGrid, numeralGrid), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	hideBottomPanels(v)
	layoutViewport(v, 640, 480)
	return v
}

// numeralUnit is one entity on the seam: an id, an owner, a cell and a health.
func numeralUnit(id uint32, owner uint32, hp int) MapEntity {
	return MapEntity{ID: id, Owner: owner, Cell: image.Pt(6, 6), HP: hp, MaxHP: 100}
}

// push is one frame: the tick's entities, then the viewer's own step at now.
func push(v *Viewer, now time.Time, ents ...MapEntity) {
	v.SetEntities(ents)
	v.step(Input{}, now)
}

// at0 is a fixed instant, so every elapsed time below is exact.
var at0 = time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)

func TestAStrictDecreaseMakesAFigureOfTheDifference(t *testing.T) {
	v := numeralViewer(t)

	// The FIRST push an entity is ever seen in makes nothing, whatever it holds.
	push(v, at0, numeralUnit(1, 1, 40))
	if _, live := v.DamageNumerals(); live != 0 {
		t.Fatalf("the first sighting of an entity made %d figure(s); arriving is not a wound", live)
	}

	push(v, at0, numeralUnit(1, 1, 31))
	if _, live := v.DamageNumerals(); live != 1 {
		t.Fatalf("a nine-point drop made %d figure(s), want 1", live)
	}
	if got := v.numerals[0].damage; got != 9 {
		t.Errorf("figure = %d, want 9 — the health held less the health told", got)
	}
}

func TestAHealthThatDidNotFallMakesNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		then int
	}{
		{"unchanged", 40},
		{"risen", 55},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := numeralViewer(t)
			push(v, at0, numeralUnit(1, 1, 40))
			push(v, at0, numeralUnit(1, 1, tc.then))
			if _, live := v.DamageNumerals(); live != 0 {
				t.Errorf("a health that %s made %d figure(s)", tc.name, live)
			}
		})
	}
}

// The remembered health follows EVERY push, whatever the gates said about it —
// the ordering rule the ingest rests on. A stale memory would show the next blow
// as the sum of both, which is a merge across a window that has already expired.
func TestTheRememberedHealthFollowsEveryPush(t *testing.T) {
	for _, tc := range []struct {
		name string
		off  bool
		mid  int
		want int
	}{
		{"a blow taken with the display off", true, 30, 5},
		{"a health that rose", false, 60, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := numeralViewer(t)
			if tc.off {
				v.ToggleDamageNumerals()
			}
			push(v, at0, numeralUnit(1, 1, 40))
			push(v, at0, numeralUnit(1, 1, tc.mid))
			if tc.off {
				v.ToggleDamageNumerals()
			}
			push(v, at0, numeralUnit(1, 1, tc.mid-5))
			if _, live := v.DamageNumerals(); live != 1 {
				t.Fatalf("live = %d, want 1", live)
			}
			if got := v.numerals[0].damage; got != tc.want {
				t.Errorf("figure = %d, want %d — the memory did not follow the push", got, tc.want)
			}
		})
	}
}

// A unit that is not alive is remembered like every other, so the blow that
// takes a corpse further below zero still shows what it took.
func TestADeadUnitIsRememberedAndStillShowsItsBlow(t *testing.T) {
	v := numeralViewer(t)
	e := numeralUnit(1, 1, 4)
	e.Life = LifeDead
	push(v, at0, e)
	e.HP = -6
	push(v, at0, e)
	if _, live := v.DamageNumerals(); live != 1 || v.numerals[0].damage != 10 {
		t.Errorf("a blow on a body made %d figure(s) of %v, want 1 of 10", live, v.numerals)
	}
}

// The decay walk is not a blow. A corpse loses one health every second full
// tick all the way down to -600, and every one of those decreases arrives here
// looking exactly like a wound. The original shows none of them: a dead actor's
// falling health reaches its client as the corpse STAGE byte and never as a
// number (ANIM-DEATH-007, High).
//
// The test that separates them is the health this viewer was ALREADY holding,
// which is why the killing blow above still shows: it lands on a living victim.
func TestTheDecayWalkOnABodyMakesNoFigures(t *testing.T) {
	v := numeralViewer(t)
	e := numeralUnit(1, 1, 3)
	push(v, at0, e)

	// The blow that fells him: 3 -> -2, one figure of 5.
	e.HP, e.Life = -2, LifeDead
	push(v, at0, e)
	if _, live := v.DamageNumerals(); live != 1 || v.numerals[0].damage != 5 {
		t.Fatalf("the killing blow made %d figure(s) of %v, want 1 of 5", live, v.numerals)
	}
	v.numerals = nil

	// Then the ladder, all the way to the bottom rung. Not one of these is a
	// cause, so not one of them is a number.
	for hp := -3; hp >= -600; hp-- {
		e.HP = hp
		push(v, at0, e)
	}
	if _, live := v.DamageNumerals(); live != 0 {
		t.Errorf("the walk from -2 to -600 made %d figure(s); decay is not a cause", live)
	}
}

// The display gates CREATION, not drawing: switching it on afterwards must
// reveal nothing, and the next blow shows that blow's damage alone.
func TestTheDisplayGatesCreationAndNotDrawing(t *testing.T) {
	v := numeralViewer(t)
	if on, _ := v.DamageNumerals(); !on {
		t.Fatalf("a viewer opened with the display off; it defaults to on")
	}
	if on := v.ToggleDamageNumerals(); on {
		t.Fatalf("the toggle reported on after flipping from on")
	}

	push(v, at0, numeralUnit(1, 1, 40))
	push(v, at0, numeralUnit(1, 1, 25))
	if _, live := v.DamageNumerals(); live != 0 {
		t.Fatalf("a blow under a hidden display made %d figure(s)", live)
	}
	if on := v.ToggleDamageNumerals(); !on {
		t.Fatalf("the toggle reported off after flipping back")
	}
	if _, live := v.DamageNumerals(); live != 0 {
		t.Fatalf("turning the display on revealed %d figure(s) that had accrued invisibly", live)
	}
	push(v, at0, numeralUnit(1, 1, 20))
	if _, live := v.DamageNumerals(); live != 1 || v.numerals[0].damage != 5 {
		t.Errorf("the next blow made %d figure(s) of %v, want 1 of 5 — this blow's own damage",
			live, v.numerals)
	}
}

// A second blow inside the window the original would have merged into is its
// OWN figure beside the first (hotfix, docs/hotfix/LEDGER.md): an
// owner-directed divergence from ANIM-NUM-020's decoded merge, so a
// frequently-hit victim shows one number per blow instead of one sum that
// periodically pops.
func TestASecondBlowMakesAnIndependentFigureAtTheSameBirthOffset(t *testing.T) {
	v := numeralViewer(t)
	// A cadence no elapsed time below can cross, so nothing here drifts either
	// figure between the two blows.
	v.SetPeriod(1 << 30)
	push(v, at0, numeralUnit(1, 1, 40))
	push(v, at0, numeralUnit(1, 1, 31))
	firstDamage, firstBorn, firstOff := v.numerals[0].damage, v.numerals[0].born, v.numerals[0].off

	push(v, at0.Add(300*time.Millisecond), numeralUnit(1, 1, 27))
	if _, live := v.DamageNumerals(); live != 2 {
		t.Fatalf("a second blow made %d figure(s), want 2 — no merge", live)
	}
	if v.numerals[0].damage != firstDamage || !v.numerals[0].born.Equal(firstBorn) || v.numerals[0].off != firstOff {
		t.Errorf("the first figure changed to %+v; a later blow must not touch it", v.numerals[0])
	}
	second := v.numerals[1]
	if second.damage != 4 {
		t.Errorf("second figure = %d, want 4 — its own blow's amount, not the sum", second.damage)
	}
	if !second.born.Equal(at0.Add(300 * time.Millisecond)) {
		t.Errorf("second figure born at %v, want its own blow's instant", second.born)
	}
	if second.off != firstOff {
		t.Errorf("second figure starts at %v, want the same unit-relative origin %v", second.off, firstOff)
	}
}

// Repeated damage must not push new figures up the screen. Existing figures
// keep their own flight, including while the oldest figures expire.
func TestRepeatedDamageNumeralsKeepTheirBirthPosition(t *testing.T) {
	for _, owner := range []uint32{1, 2} {
		v := litViewer(t)
		v.SetLocalOwner(1)
		v.SetPeriod(20000)
		hp := 1000
		push(v, at0, numeralUnit(1, owner, hp))
		var origin image.Point
		for hit := 0; hit < 50; hit++ {
			hp--
			push(v, at0.Add(time.Duration(hit)*100*time.Millisecond), numeralUnit(1, owner, hp))
			placements := v.numeralPlacements()
			if len(placements) != len(v.numerals) || len(placements) == 0 {
				t.Fatalf("owner %d hit %d: %d placed of %d live figures", owner, hit, len(placements), len(v.numerals))
			}
			newest := placements[len(placements)-1].At
			if hit == 0 {
				origin = newest
			} else if newest != origin {
				t.Fatalf("owner %d hit %d starts at %v, want %v", owner, hit, newest, origin)
			}
			if hit > 0 && placements[0].At.Y >= newest.Y {
				t.Fatalf("older figure did not rise independently: old %v new %v", placements[0].At, newest)
			}
		}
	}
}

// N applications inside what was the merge window each make their own
// figure, with that application's own amount, in application order, and no
// record anywhere holds their sum (hotfix, docs/hotfix/LEDGER.md) — the
// shape of a wall of fire or any other frequently-ticking source, which used
// to read as one number that grew and popped roughly once a second.
func TestNApplicationsInsideTheOldMergeWindowEachMakeTheirOwnFigure(t *testing.T) {
	v := numeralViewer(t)
	v.SetPeriod(1 << 30)
	hp := 100
	push(v, at0, numeralUnit(1, 1, hp))
	amounts := []int{3, 5, 2, 7, 4}
	for i, dmg := range amounts {
		hp -= dmg
		push(v, at0.Add(time.Duration(i)*150*time.Millisecond), numeralUnit(1, 1, hp))
	}
	if _, live := v.DamageNumerals(); live != len(amounts) {
		t.Fatalf("%d applications made %d figure(s), want %d — one each", len(amounts), live, len(amounts))
	}
	total := 0
	for _, want := range amounts {
		total += want
	}
	for i, want := range amounts {
		if got := v.numerals[i].damage; got != want {
			t.Errorf("figure %d = %d, want %d — application order, individual amounts", i, got, want)
		}
		if v.numerals[i].damage == total {
			t.Errorf("figure %d shows %d, the accumulated total; no record may hold a sum", i, total)
		}
	}
}

func TestTheColourIsTheStruckUnitsOwners(t *testing.T) {
	v := numeralViewer(t)
	v.SetLocalOwner(1)
	push(v, at0,
		numeralUnit(1, 1, 40), numeralUnit(2, 2, 40), numeralUnit(3, 2, 40), numeralUnit(4, 0, 40))
	push(v, at0,
		numeralUnit(1, 1, 30), numeralUnit(2, 2, 30), numeralUnit(3, 2, 30), numeralUnit(4, 0, 30))

	by := map[uint32]int{}
	for _, n := range v.numerals {
		by[n.victim] = int(n.colour.R)<<16 | int(n.colour.G)<<8 | int(n.colour.B)
	}
	if len(by) != 4 {
		t.Fatalf("four blows made %d figures", len(by))
	}
	if by[2] != by[3] {
		t.Errorf("two units of one owner took two colours")
	}
	if by[1] == by[2] {
		t.Errorf("two units of different owners took one colour")
	}
	if by[4] == by[1] || by[4] == by[2] {
		t.Errorf("an unowned unit shares a colour with an owner's")
	}
}

// No drawn property of a figure depends on how badly the victim is hurt —
// the severity band the original computes and reads nowhere (ANIM-NUM-021).
func TestNoPropertyOfAFigureDependsOnTheVictimsRemainingHealth(t *testing.T) {
	v := numeralViewer(t)
	// Three victims of one owner at three bands: above half, between a quarter
	// and a half, and below a quarter. Equal damage on each.
	push(v, at0, numeralUnit(1, 1, 90), numeralUnit(2, 1, 45), numeralUnit(3, 1, 20))
	push(v, at0, numeralUnit(1, 1, 80), numeralUnit(2, 1, 35), numeralUnit(3, 1, 10))
	if len(v.numerals) != 3 {
		t.Fatalf("three blows made %d figures", len(v.numerals))
	}
	first := v.numerals[0]
	for _, n := range v.numerals[1:] {
		if n.colour != first.colour || n.damage != first.damage ||
			n.off != first.off || n.drift != first.drift {
			t.Errorf("a figure differs by the victim's remaining health: %+v vs %+v", n, first)
		}
	}
}

// The birth offset and its ownership sign (AC-9, AC-10).
func TestTheBirthOffsetTakesTheOwnershipSign(t *testing.T) {
	for _, tc := range []struct {
		name  string
		local uint32
		owner uint32
		want  image.Point
	}{
		{"the local participant's own unit", 1, 1, image.Pt(16, -48)},
		{"someone else's unit", 1, 2, image.Pt(-16, -48)},
		{"no local participant, an unowned victim", 0, 0, image.Pt(16, -48)},
		{"no local participant, an owned victim", 0, 3, image.Pt(-16, -48)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := numeralViewer(t)
			v.SetLocalOwner(tc.local)
			push(v, at0, numeralUnit(1, tc.owner, 40))
			push(v, at0, numeralUnit(1, tc.owner, 33))
			if got := v.numerals[0].off; got != tc.want {
				t.Errorf("birth offset = %v, want %v", got, tc.want)
			}
			// The drift sign is the offset's own, so a figure starts on the side
			// it goes on to travel toward.
			if sign := v.numerals[0].drift; (sign > 0) != (tc.want.X > 0) {
				t.Errorf("drift %d disagrees with the birth offset %v", sign, tc.want)
			}
		})
	}
}

// No owner value can index the palette out of range.
func TestEveryOwnerValueTakesAColour(t *testing.T) {
	v := numeralViewer(t)
	for _, owner := range []uint32{0, 1, 7, 8, 9, 1 << 31, ^uint32(0)} {
		v.numerals, v.numeralHP = nil, nil
		push(v, at0, numeralUnit(1, owner, 40))
		push(v, at0, numeralUnit(1, owner, 30))
		if len(v.numerals) != 1 {
			t.Fatalf("owner %d made %d figures", owner, len(v.numerals))
		}
	}
}

// An entity absent from a push is dropped from the memory on that push, so a
// viewer reused across two maps cannot attribute one map's health to another
// map's entity of the same id.
func TestAnEntityAbsentFromAPushIsForgotten(t *testing.T) {
	v := numeralViewer(t)
	push(v, at0, numeralUnit(1, 1, 90))
	push(v, at0) // the map closed
	push(v, at0, numeralUnit(1, 1, 20))
	if _, live := v.DamageNumerals(); live != 0 {
		t.Errorf("a re-arrival at a lower health made %d figure(s); it is an arrival, not a wound", live)
	}
}

// The two clocks. The life is wall clock and the drift is the tick, which is
// why a slow cadence shortens the journey rather than lengthening the life
// — and why a popup freezes a figure in place while it still vanishes.

// tickPeriodUS is the cadence these clock tests drive: one tick a millisecond,
// so an elapsed duration in milliseconds IS a tick count.
const tickPeriodUS = 1000

// clockViewer is a viewer rated at tickPeriodUS with its animation baseline
// already taken, holding one figure over the entity id and owner given. The
// local participant is established BEFORE the blow, because the drift sign is
// fixed at the record's birth.
func clockViewer(t *testing.T, local, owner uint32) (*Viewer, time.Time) {
	t.Helper()
	v := numeralViewer(t)
	v.SetLocalOwner(local)
	v.SetPeriod(tickPeriodUS)
	push(v, at0, numeralUnit(1, owner, 40)) // baseline: no elapsed yet
	push(v, at0, numeralUnit(1, owner, 33)) // the blow
	if _, live := v.DamageNumerals(); live != 1 {
		t.Fatalf("the fixture made %d figures, want 1", live)
	}
	return v, at0
}

func TestAFigureDriftsOnePerTickSidewaysAndTwoUp(t *testing.T) {
	for _, tc := range []struct {
		name  string
		owner uint32
		signX int
	}{
		{"the local participant's own unit", 1, +1},
		{"someone else's unit", 2, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, now := clockViewer(t, 1, tc.owner)
			born := v.numerals[0].off

			// A frame that crossed no tick drifts nothing.
			push(v, now, numeralUnit(1, tc.owner, 33))
			if got := v.numerals[0].off; got != born {
				t.Fatalf("a frame crossing no tick moved the figure to %v from %v", got, born)
			}

			// Three ticks in one frame move it three steps, not one.
			now = now.Add(3 * time.Millisecond)
			push(v, now, numeralUnit(1, tc.owner, 33))
			want := born.Add(image.Pt(3*tc.signX, -6))
			if got := v.numerals[0].off; got != want {
				t.Errorf("after three ticks the offset is %v, want %v", got, want)
			}
		})
	}
}

// The sign is fixed at birth, so a change of local participant mid-flight does
// not turn a figure round.
func TestAFigureInFlightKeepsTheDirectionItSetOffIn(t *testing.T) {
	v, now := clockViewer(t, 1, 1)
	push(v, now.Add(time.Millisecond), numeralUnit(1, 1, 33))
	after := v.numerals[0].off

	v.SetLocalOwner(2) // the same unit is now somebody else's
	push(v, now.Add(2*time.Millisecond), numeralUnit(1, 1, 33))
	if got := v.numerals[0].off.X - after.X; got != +1 {
		t.Errorf("the figure stepped %+d after the local participant changed, want +1", got)
	}
}

// The life is WALL CLOCK: present at 999 and at exactly 1000, gone at 1001 —
// and the same with the ambient counter never advanced at all, which is what
// says the life is not the tick.
func TestTheLifeIsAThousandMillisecondsOfWallClock(t *testing.T) {
	for _, tc := range []struct {
		name   string
		period int
	}{
		{"with the ambient clock running", tickPeriodUS},
		{"with the ambient clock never advanced", 1 << 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, at := range []struct {
				ms   int
				live int
			}{{999, 1}, {1000, 1}, {1001, 0}} {
				v := numeralViewer(t)
				v.SetPeriod(tc.period)
				push(v, at0, numeralUnit(1, 1, 40))
				push(v, at0, numeralUnit(1, 1, 33))
				push(v, at0.Add(time.Duration(at.ms)*time.Millisecond), numeralUnit(1, 1, 33))
				if _, live := v.DamageNumerals(); live != at.live {
					t.Errorf("at %d ms: %d figure(s) live, want %d", at.ms, live, at.live)
				}
			}
		})
	}
}

// Two figures over one victim each expire on their OWN clock (hotfix,
// docs/hotfix/LEDGER.md): the first blow's figure is gone once its own 1000
// ms has passed even while the second blow's figure, born later, still has
// life left — the opposite of the merge this replaces, which tied both to
// the first blow's clock alone.
func TestASecondFigureExpiresOnItsOwnClockNotTheFirsts(t *testing.T) {
	v := numeralViewer(t)
	push(v, at0, numeralUnit(1, 1, 40))
	push(v, at0, numeralUnit(1, 1, 33)) // figure 1: born at0, damage 7
	push(v, at0.Add(900*time.Millisecond), numeralUnit(1, 1, 30))
	if _, live := v.DamageNumerals(); live != 2 { // figure 2: born +900ms, damage 3
		t.Fatalf("two blows produced %d figure(s), want 2", live)
	}

	// 1001 ms after the FIRST blow: it is gone, but the second, only 101 ms
	// into its own life, still stands.
	push(v, at0.Add(1001*time.Millisecond), numeralUnit(1, 1, 30))
	if _, live := v.DamageNumerals(); live != 1 {
		t.Fatalf("at 1001 ms: %d figure(s) live, want 1 — the second's own clock", live)
	}
	if v.numerals[0].damage != 3 {
		t.Errorf("the survivor shows %d, want 3 — the second blow's own amount", v.numerals[0].damage)
	}

	// 1901 ms after the first is 1001 ms after the second: gone too.
	push(v, at0.Add(1901*time.Millisecond), numeralUnit(1, 1, 30))
	if _, live := v.DamageNumerals(); live != 0 {
		t.Errorf("%d figure(s) survived past the second blow's own window", live)
	}
}

// A clock that went backwards must not reap a figure it cannot have aged.
func TestAClockThatWentBackwardsReapsNothing(t *testing.T) {
	v := numeralViewer(t)
	push(v, at0, numeralUnit(1, 1, 40))
	push(v, at0, numeralUnit(1, 1, 33))
	push(v, at0.Add(-time.Hour), numeralUnit(1, 1, 33))
	if _, live := v.DamageNumerals(); live != 1 {
		t.Errorf("a backwards clock left %d figure(s), want 1", live)
	}
}

// The composed figure and its placement.

// numeralFont is a 224-record font whose records are distinguishable: record k
// paints one pixel at (row k%6, column (k/6)%5). It is panelFont's arrangement,
// built here so these tests do not depend on that file's constants.
func numeralFont() *text.Font {
	f := &text.Font{Spacing: 1, Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		g := text.Glyph{Width: 5, Height: 6, Pixels: make([]text.Pixel, 30), Advance: 3}
		if k == 0 {
			g.Advance = 2
		} else {
			g.Pixels[(k%6)*5+(k/6)%5] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[k] = g
	}
	return f
}

// litViewer is numeralViewer with a font, so figures compose.
func litViewer(t *testing.T) *Viewer {
	t.Helper()
	v := numeralViewer(t)
	v.SetPeriod(1 << 30) // a cadence no test below crosses, so nothing drifts
	v.SetFont(numeralFont())
	return v
}

// A figure is drawn TWICE: every painted face pixel has a shadow counterpart one
// pixel down and right, in a different colour (AC-12).
func TestAFigureIsDrawnTwiceAShadowUnderAFace(t *testing.T) {
	v := litViewer(t)
	push(v, at0, numeralUnit(1, 1, 40))
	push(v, at0, numeralUnit(1, 1, 33))
	pic := v.numerals[0].pic
	if pic == nil {
		t.Fatalf("a viewer holding a font composed nothing")
	}

	face := v.numerals[0].colour
	faces, shadows := 0, 0
	b := pic.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			switch pic.RGBAAt(x, y) {
			case face:
				faces++
				// The shadow is UNDER the face, so where the two overlap the face
				// wins and the counterpart may itself be a face pixel.
				if c := pic.RGBAAt(x+DamageNumeralShadow, y+DamageNumeralShadow); c.A == 0 {
					t.Errorf("the face pixel at (%d,%d) has no issue behind it", x, y)
				}
			case DamageNumeralShadowColor:
				shadows++
			}
		}
	}
	if faces == 0 || shadows == 0 {
		t.Errorf("the picture holds %d face and %d shadow pixel(s); both issues must be present",
			faces, shadows)
	}
	if face == DamageNumeralShadowColor {
		t.Errorf("the two issues are the same colour, so there is only one")
	}
}

// A viewer with no font makes the record anyway — it lives, drifts and expires —
// and simply draws nothing.
func TestAViewerWithNoFontStillKeepsTheRecord(t *testing.T) {
	v := numeralViewer(t)
	push(v, at0, numeralUnit(1, 1, 40))
	push(v, at0, numeralUnit(1, 1, 33))
	if _, live := v.DamageNumerals(); live != 1 {
		t.Fatalf("a fontless viewer made %d record(s), want 1", live)
	}
	if got := v.numeralPlacements(); got != nil {
		t.Errorf("a fontless viewer placed %v, want nothing", got)
	}
}

// The placement is the struck unit's own placed position plus the record's
// offset, through the transform every other mark on that unit takes (AC-13).
func TestAFigureIsPlacedOffTheStruckUnitsOwnPosition(t *testing.T) {
	v := litViewer(t)
	v.SetLocalOwner(1)
	push(v, at0, numeralUnit(1, 1, 40))
	push(v, at0, numeralUnit(1, 1, 33))

	base := v.numeralPlacements()
	if len(base) != 1 {
		t.Fatalf("one figure placed %d times", len(base))
	}

	// Panning the camera moves it exactly as it moves every other glyph.
	before := base[0].At
	v.cam.Pan(30, 20)
	moved := v.numeralPlacements()
	if len(moved) != 1 || moved[0].At == before {
		t.Fatalf("a pan left the figure at %v", moved)
	}
	v.cam.Pan(-30, -20)

	// The record's offset is what separates it from the unit's own anchor: a
	// record at zero offset places at a different point from one at the birth
	// offset, and the difference is the offset itself under unit zoom.
	off := v.numerals[0].off
	v.numerals[0].off = image.Point{}
	at0Off := v.numeralPlacements()[0].At
	v.numerals[0].off = off
	atBirth := v.numeralPlacements()[0].At
	if got := atBirth.Sub(at0Off); got != off {
		t.Errorf("the offset moved the figure by %v, want %v", got, off)
	}

	// A zoom scales the offset, because the figure hangs off a unit that scaled.
	v.cam.SetZoom(2)
	zoomed := v.numeralPlacements()
	if len(zoomed) != 1 {
		t.Fatalf("a zoom placed %d figures", len(zoomed))
	}
	v.numerals[0].off = image.Point{}
	if zoomed[0].At == v.numeralPlacements()[0].At {
		t.Errorf("the offset did not scale with the zoom")
	}
}

// A displacement mid-crossing carries the figure with the unit.
func TestAFigureCarriesTheUnitsOwnDisplacement(t *testing.T) {
	v := litViewer(t)
	e := numeralUnit(1, 1, 40)
	push(v, at0, e)
	e.HP = 33
	push(v, at0, e)
	still := v.numeralPlacements()[0].At

	// The phase pair is what turns the displacement on: a viewer never told
	// where it stands inside a tick draws no displacement at all.
	v.SetPhase(0, tickPeriodUS)
	e.Step, e.Transit, e.TransitSpan = image.Pt(1, 0), 1, 2
	v.SetEntities([]MapEntity{e})
	if got := v.numeralPlacements()[0].At; got == still {
		t.Errorf("a unit mid-crossing left its figure at %v", got)
	}
}

// A record whose unit is not in this frame places nothing and is NOT destroyed.
func TestAFigureOverAnAbsentUnitPlacesNothingAndSurvives(t *testing.T) {
	v := litViewer(t)
	push(v, at0, numeralUnit(1, 1, 40))
	push(v, at0, numeralUnit(1, 1, 33))
	push(v, at0.Add(time.Millisecond)) // the unit is gone from the snapshot
	if got := v.numeralPlacements(); got != nil {
		t.Errorf("a figure over an absent unit placed %v", got)
	}
	if _, live := v.DamageNumerals(); live != 1 {
		t.Errorf("the record was destroyed rather than left to expire")
	}
	push(v, at0.Add(1001*time.Millisecond))
	if _, live := v.DamageNumerals(); live != 0 {
		t.Errorf("the record did not expire on its own clock")
	}
}

// The toggle (AC-11). Ctrl+L is the decoded key; bare L stays the chip.

func TestCtrlLFlipsTheDisplayAndBareLDoesNot(t *testing.T) {
	a, v, seam := affOnMap(t)
	if on, _ := v.DamageNumerals(); !on {
		t.Fatalf("a map screen opened with the display off")
	}

	// Bare L is still the chip and touches the display not at all.
	affSelect(t, a, v, affLoCol, affLoRow, selection{affLoID})
	a.step(appInput{Chip: true}, affAt)
	if on, _ := v.DamageNumerals(); !on {
		t.Errorf("bare L flipped the display")
	}
	if len(seam.blows) == 0 {
		t.Errorf("bare L no longer chips")
	}

	// Ctrl+L flips the display and chips nothing.
	chips := len(seam.blows)
	a.step(appInput{Numerals: true}, affAt)
	if on, _ := v.DamageNumerals(); on {
		t.Errorf("Ctrl+L did not flip the display")
	}
	if len(seam.blows) != chips {
		t.Errorf("Ctrl+L chipped %d unit(s)", len(seam.blows)-chips)
	}
	a.step(appInput{Numerals: true}, affAt)
	if on, _ := v.DamageNumerals(); !on {
		t.Errorf("a second Ctrl+L did not flip the display back")
	}
}

// The key is read on the MAP ARM ALONE, so on every other screen it does
// nothing — a property of where the read stands.
func TestTheNumeralKeyDoesNothingOffTheMapScreen(t *testing.T) {
	a, v, _ := affOnMap(t)
	a.step(appInput{Escape: true}, affAt) //
	leaveViaMenu(a.flow)                  // ...and EXIT is what returns to the picker
	if a.Screen() == ScreenMap {
		t.Fatalf("setup: still on the map screen")
	}
	a.step(appInput{Numerals: true}, affAt)
	a.step(appInput{Escape: true}, affAt) // back to the menu
	a.step(appInput{Numerals: true}, affAt)
	if on, _ := v.DamageNumerals(); !on {
		t.Errorf("the key flipped the display from off the map screen")
	}
}
