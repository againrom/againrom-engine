package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The cast records' path figures and smoke trails.
//
// Every fixture here is synthetic. The frame counts are the shipped ones —
// picture 34 holds five, picture 36 thirty-five and each trail sheet six — so a
// selection that refuses an index refuses it here for the same reason it would
// on an install.

func spFrames(n, size int) []*terrain.EffectFrame {
	out := make([]*terrain.EffectFrame, n)
	for i := range out {
		out[i] = &terrain.EffectFrame{Width: size, Height: size,
			Pixels: make([]color.RGBA, size*size)}
	}
	return out
}

// spWorld is sbWorld with the art a path picture and a trail need.
func spWorld(t *testing.T) *mapWorld {
	t.Helper()
	mw := sbWorld(t)
	set := &terrain.EffectSet{Sheets: map[int]*terrain.EffectSheet{
		// firebolt and fireball, the two that travel and leave a trail.
		10: {Frames: spFrames(36, 8), Phases: 4, RotationPhases: 16, Flip: true, CenterX: 32, CenterY: 32},
		12: {Frames: spFrames(36, 8), Phases: 4, RotationPhases: 16, Flip: true, CenterX: 64, CenterY: 64},
		13: {Frames: spFrames(11, 8), Phases: 11, RotationPhases: 1, CenterX: 64, CenterY: 64},
		// lightnin and chain, the two that draw a path.
		34: {Frames: spFrames(5, 16), Phases: 5, RotationPhases: 1, CenterX: 8, CenterY: 8},
		36: {Frames: spFrames(35, 16), Phases: 35, RotationPhases: 1, CenterX: 8, CenterY: 8},
	}}
	for i := range set.Smoke {
		set.Smoke[i] = &terrain.EffectSheet{Frames: spFrames(6, 12), Phases: 6,
			RotationPhases: 1, CenterX: 6, CenterY: 6}
	}
	mw.projectiles = set
	return mw
}

const spLightning, spPrismatic, spFireBall = 13, 14, 2

// TestAPathPictureStandsStillAndSpansItsWholeSegment is the whole of the
// owner's report, walked: a Lightning record never moves, and the figure it
// draws reaches the target from its first frame rather than growing toward it.
func TestAPathPictureStandsStillAndSpansItsWholeSegment(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spLightning,
		FromX: 2, FromY: 2, ToX: 8, ToY: 5}})
	records := flightRecords(mw)
	if len(records) != 1 || records[0].Picture != 34 || records[0].ActionSegments != 12 {
		t.Fatalf("lightning built %+v, want one picture-34 record after its first of 13 calls", records)
	}
	x, y := records[0].X, records[0].Y

	// The world holds no caster class, so the launch point is the caster's
	// cell centre (castLaunch); both ends are display pixels.
	from, to := image.Pt(2*32+16, 2*32+16), image.Pt(8*32+16, 5*32+16)
	for age := range 13 {
		spCheckFigure(t, fmt.Sprintf("age %d", age), mw.boltDraws(nil), from, to)
		if r := flightRecords(mw); len(r) != 1 || r[0].X != x || r[0].Y != y {
			t.Fatalf("at age %d the record is %+v, want it standing at (%d,%d)", age, r, x, y)
		}
		flightStep(mw)
	}
	if got := mw.boltDraws(nil); len(got) != 0 {
		t.Errorf("the record outlived its own 13 calls: %d stamps", len(got))
	}
}

// TestAPathIsRegeneratedWholeOnEveryTick is `MAGIC-BOLTLIST-071`'s replace: the
// engine rebuilds the point list on every driver call, so no kink holds still.
func TestAPathIsRegeneratedWholeOnEveryTick(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spLightning,
		FromX: 0, FromY: 0, ToX: 9, ToY: 0}})

	var same int
	first := spPositions(mw.boltDraws(nil))
	for age := 1; age < 13; age++ {
		flightStep(mw)
		next := spPositions(mw.boltDraws(nil))
		if spEqual(first, next) {
			same++
		}
		first = next
	}
	if same > 2 {
		t.Errorf("%d of 12 ticks redrew the previous figure unchanged, want a fresh walk each tick", same)
	}
}

// TestAPathLeavesTheStraightLine is the figure's own shape: a bounded random
// walk, not an interpolation. Some point of some tick's figure stands off the
// caster-to-target line.
func TestAPathLeavesTheStraightLine(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spLightning,
		FromX: 0, FromY: 0, ToX: 9, ToY: 0}})

	var off int
	for range 13 {
		for _, d := range mw.boltDraws(nil) {
			if d.Pos.Y != 16 {
				off++
			}
		}
		flightStep(mw)
	}
	if off == 0 {
		t.Error("every stamp of every tick stood on the straight line — the figure is an interpolation")
	}
}

// TestAPathObjectIsDeterministic keeps the figure out of the determinism story:
// it is drawn from the observation and the driver calls, so one replay draws
// one picture and nothing here reads a clock.
func TestAPathObjectIsDeterministic(t *testing.T) {
	t.Parallel()

	run := func() []image.Point {
		mw := spWorld(t)
		mw.observeCasts([]sim.CastEvent{{Caster: 3, Target: 7, Spell: spPrismatic,
			FromX: 1, FromY: 1, ToX: 6, ToY: 4}})
		flightStep(mw)
		flightStep(mw)
		return spPositions(mw.boltDraws(nil))
	}
	if !spEqual(run(), run()) {
		t.Error("two runs of one observation drew two figures")
	}
}

// TestTheSecondPathPictureOffsetsItsPhaseByTheRecordTag is what picture 36's
// thirty-five frames are for: seven tags of five phases.
func TestTheSecondPathPictureOffsetsItsPhaseByTheRecordTag(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{
		{Caster: 1, Target: 2, Spell: spPrismatic, FromX: 0, FromY: 0, ToX: 5, ToY: 0},
		{Caster: 3, Target: 4, Spell: spPrismatic, FromX: 0, FromY: 0, ToX: 5, ToY: 0},
	})
	if got := len(flightRecords(mw)); got != 2 {
		t.Fatalf("two observations built %d records", got)
	}
	byTag := map[int]bool{}
	for _, d := range mw.boltDraws(nil) {
		byTag[d.Frame] = true
	}
	// Call 1 of the normal route is phase 4; tag 1 adds five.
	if len(byTag) != 2 || !byTag[4] || !byTag[4+boltChainStride] {
		t.Errorf("the two sets drew frames %v, want 4 and %d", byTag, 4+boltChainStride)
	}
}

// TestATravellingObjectLeavesSixPastPositions is `ANIM-140` and `ANIM-142`: a
// queue of at most six points each driver call started from, oldest first,
// point i drawing frame i, so the newest of six draws frame 5.
func TestATravellingObjectLeavesSixPastPositions(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	// Fire Arrow over nine cells: picture 10 divides by 200, so the record
	// flies eleven calls and the queue fills.
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 1,
		FromX: 0, FromY: 0, ToX: 9, ToY: 0}})
	records := flightRecords(mw)
	if len(records) != 1 || records[0].ActionSegments < 6 {
		t.Fatalf("the cast built %+v, want one record that flies long enough to fill a six-entry queue", records)
	}

	for age := 0; len(flightRecords(mw)) > 0; age++ {
		draws := mw.boltDraws(nil)
		want := min(age+1, 6) + 1 // the trail points plus the record's own sprite
		if len(draws) != want {
			t.Fatalf("at age %d the record handed over %d drawables, want %d", age, len(draws), want)
		}
		head := draws[0]
		trail := draws[1:]
		for i, e := range trail {
			if e.Frame != i {
				t.Errorf("at age %d trail point %d is frame %d, want %d", age, i, e.Frame, i)
			}
			if e.Pos.X >= head.Pos.X {
				t.Errorf("at age %d trail point %d stands at %d, not behind the record's %d",
					age, i, e.Pos.X, head.Pos.X)
			}
			if i > 0 && e.Pos.X <= trail[i-1].Pos.X {
				t.Errorf("at age %d the trail is not ordered oldest first: %d after %d",
					age, e.Pos.X, trail[i-1].Pos.X)
			}
		}
		flightStep(mw)
	}
}

// TestOnlyTheTwoTravellingPicturesLeaveATrail is the other half of the contract:
// a trail is drawn only where the original has one. A path picture leaves none.
func TestOnlyTheTwoTravellingPicturesLeaveATrail(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spLightning,
		FromX: 0, FromY: 0, ToX: 9, ToY: 0}})
	for range 13 {
		for _, d := range mw.boltDraws(nil) {
			if d.Sheet == mw.projectiles.SmokeSheet(0) || d.Sheet == mw.projectiles.SmokeSheet(1) {
				t.Fatal("a path picture left a smoke trail")
			}
		}
		flightStep(mw)
	}
}

// TestFireBallsCastBuildsItsFlightAlone: the burst is a World record built
// when the area blasts (ANIM-111), not at the cast. The cast builds its flying
// record alone.
func TestFireBallsCastBuildsItsFlightAlone(t *testing.T) {
	t.Parallel()

	for _, weapon := range []bool{false, true} {
		mw := spWorld(t)
		mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spFireBall, Weapon: weapon,
			FromX: 0, FromY: 0, ToX: 9, ToY: 0}})
		records := flightRecords(mw)
		if len(records) != 1 || records[0].Picture != 12 {
			t.Fatalf("weapon %v: fire ball built %+v, want the picture-12 flight alone", weapon, records)
		}
		// Not homing (MAGIC-288): aimed at the target cell's centre, no target.
		if r := records[0]; r.ActionTarget != 0 || r.ActionX != 9*256+128 || r.ActionY != 128 {
			t.Errorf("weapon %v: the flight aims at %d (%d,%d), want the cell centre and no target", weapon, r.ActionTarget, r.ActionX, r.ActionY)
		}
	}
}

// TestAStaffReleaseLeavesTheRecordABookCastLeaves is SAV-1129's cast divert:
// a staff's release reaches the cast producer, so each picture is built, flown
// and drawn the same from a staff as from a book.
func TestAStaffReleaseLeavesTheRecordABookCastLeaves(t *testing.T) {
	t.Parallel()

	for _, spell := range []uint16{1, spFireBall, spLightning, spPrismatic} {
		var runs [2][][16]int32
		var kinds [2]map[string]int
		for k, weapon := range []bool{false, true} {
			mw := spWorld(t)
			mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spell, Weapon: weapon,
				FromX: 2, FromY: 2, ToX: 6, ToY: 2}})
			kinds[k] = spSheetKinds(mw, spell, mw.boltDraws(nil))
			for range 16 {
				for _, p := range flightRecords(mw) {
					runs[k] = append(runs[k], shotLeaves(p))
				}
				flightStep(mw)
			}
		}
		if len(runs[0]) == 0 || !reflect.DeepEqual(runs[0], runs[1]) {
			t.Errorf("spell %d: a book flew %v and a staff %v", spell, runs[0], runs[1])
		}
		if !reflect.DeepEqual(kinds[0], kinds[1]) {
			t.Errorf("spell %d: a book drew %v and a staff %v", spell, kinds[0], kinds[1])
		}
	}
}

// TestARiderReleaseLeavesNoCastRecord is ANIM-115: a siege or weapon-spell
// rider builds its area effect and transport only, so its release builds no
// cast record; the shot the carrier throws is its own unit-shot record.
func TestARiderReleaseLeavesNoCastRecord(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts(spRiderWorld(t, spFireBall, false))
	if got := flightRecords(mw); len(got) != 0 {
		t.Fatalf("a rider release built %+v, want no cast record", got)
	}
	if free := mw.world.SavedProjectiles().FreeIndex; free != 0 {
		t.Fatalf("a rider release took counter %d, want none taken", free)
	}
}

// TestARiderCarrierPlaysNoCastRun is the other half of the same wiring. A rider
// carrier is swinging a weapon, not casting: it is already playing its attack
// run, so replacing that with a cast run would animate a blow as a spell.
func TestARiderCarrierPlaysNoCastRun(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts(spRiderWorld(t, spFireBall, false))
	if _, ok := mw.castRun[1]; ok {
		t.Error("a rider carrier started a cast run, want none — it is mid swing")
	}
}

// TestABoltsExcursionStaysInsideTheBand: an accepted figure keeps every
// sample within 0.15 of the length of the projectile ordinate (MAGIC-278);
// rotation and truncation add at most one pixel on a horizontal segment.
func TestABoltsExcursionStaysInsideTheBand(t *testing.T) {
	t.Parallel()

	const length = 384
	off := 0
	for s := uint32(1); s <= 256; s++ {
		for _, p := range boltFigure(0, 100, length, 100, 34, (&boltRNG{state: s}).next) {
			d := max(int(p.Y)-100, 100-int(p.Y))
			off = max(off, d)
			if float64(d) > 0.15*length+1 {
				t.Fatalf("seed %d drew a point %d px off the line, past the band", s, d)
			}
		}
	}
	if off == 0 {
		t.Fatal("no seed left the straight line")
	}
}

// spCheckFigure checks one drawn link: display stamps on one frame, the first
// within a pixel of the launch point, every stamp inside the band, and the
// last within one sampling step of the target end (MAGIC-277, MAGIC-278).
func spCheckFigure(t *testing.T, at string, draws []ui.SpellBolt, from, to image.Point) {
	t.Helper()
	if len(draws) < 3 {
		t.Fatalf("%s: the figure is %d stamps, want a path", at, len(draws))
	}
	d := to.Sub(from)
	length := math.Hypot(float64(d.X), float64(d.Y))
	for i, s := range draws {
		if !s.Display || s.Frame != draws[0].Frame {
			t.Fatalf("%s: stamp %d is %+v, want a display stamp on one frame", at, i, s)
		}
		r := s.Pos.Sub(from)
		perp := math.Abs(float64(r.X*d.Y-r.Y*d.X)) / length
		if perp > 0.15*length+2 {
			t.Fatalf("%s: stamp %d is %v px off the segment", at, i, perp)
		}
	}
	if p := draws[0].Pos.Sub(from); max(p.X, -p.X, p.Y, -p.Y) > 1 {
		t.Errorf("%s: the figure starts at %v, want the launch point %v", at, draws[0].Pos, from)
	}
	r := draws[len(draws)-1].Pos.Sub(from)
	if along := float64(r.X*d.X+r.Y*d.Y) / length; along < length-8 || along > length {
		t.Errorf("%s: the last stamp is %v along a %v segment, want within one sampling step of the end", at, along, length)
	}
}

// TestAPrismaticSprayDrawsOneFigurePerVictimInItsOwnColour: one record, as the
// cast producer builds one (ANIM-147), drawing one figure per victim the cast
// reached, each in its own colour block.
func TestAPrismaticSprayDrawsOneFigurePerVictimInItsOwnColour(t *testing.T) {
	t.Parallel()

	for _, weapon := range []bool{false, true} {
		mw := spWorld(t)
		victims := []sim.CellPoint{{X: 8, Y: 5}, {X: 9, Y: 6}, {X: 7, Y: 3}}
		mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spPrismatic, Weapon: weapon,
			FromX: 2, FromY: 2, ToX: 8, ToY: 5, Victims: victims}})
		if got := flightRecords(mw); len(got) != 1 || got[0].Picture != 36 {
			t.Fatalf("weapon %v: a spray built %+v, want one picture-36 record", weapon, got)
		}

		// The drawables are read, not the record: a stamp carries the far cell of
		// the figure it belongs to, so the whole spray can be sorted into figures.
		sheet := mw.projectiles.Sheet(36)
		blocksSeen := make([]map[int]bool, len(victims))
		for k := range blocksSeen {
			blocksSeen[k] = map[int]bool{}
		}
		for age := range 13 {
			reached := make([]bool, len(victims))
			for _, d := range mw.boltDraws(nil) {
				if d.Frame < 0 || d.Frame >= len(sheet.Frames) {
					t.Fatalf("at age %d a stamp names frame %d, past the sheet's %d", age, d.Frame, len(sheet.Frames))
				}
				k := spVictimIndex(victims, d.To)
				if k < 0 {
					t.Fatalf("at age %d a stamp runs to %v, which is no victim's cell", age, d.To)
				}
				blocksSeen[k][d.Frame/boltChainStride] = true
				if e := d.Pos.Sub(d.To.Mul(32).Add(image.Pt(16, 16))); max(e.X, -e.X, e.Y, -e.Y) <= 12 {
					reached[k] = true
				}
			}
			for k, v := range victims {
				if !reached[k] {
					t.Errorf("at age %d no stamp of figure %d ends near its own victim's cell %v", age, k, v)
				}
			}
			flightStep(mw)
		}
		// Each figure holds ONE colour block for its whole life, and no two
		// figures hold the same one.
		taken := map[int]int{}
		for k, seen := range blocksSeen {
			if len(seen) != 1 {
				t.Fatalf("figure %d drew blocks %v over its life, want one colour", k, seen)
			}
			for b := range seen {
				if prev, dup := taken[b]; dup {
					t.Errorf("figures %d and %d both drew colour block %d", prev, k, b)
				}
				taken[b] = k
			}
		}
	}
}

// TestALoadedSprayDrawsNoFigure is SAV-1202: LOAD loses the record's word list,
// so a loaded picture-36 record draws no segment, while a loaded picture-34
// record draws one figure to its aim point.
func TestALoadedSprayDrawsNoFigure(t *testing.T) {
	t.Parallel()

	for spell, want := range map[uint16]bool{spPrismatic: false, spLightning: true} {
		mw := spWorld(t)
		mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spell,
			FromX: 2, FromY: 2, ToX: 8, ToY: 5}})
		clear(mw.flights)
		if got := len(mw.boltDraws(nil)) > 0; got != want {
			t.Errorf("spell %d without its look drew %v, want %v", spell, got, want)
		}
	}
}

// TestAStaffSprayDrawsItsVictimsOnceEachAcrossTheRelease drives the real step.
// A mage whose weapon carries Prismatic Spray attacks one foe while two more
// stand in view. Nothing is drawn during the wind-up; each release builds one
// record whose figures run to every victim, in victim order.
func TestAStaffSprayDrawsItsVictimsOnceEachAcrossTheRelease(t *testing.T) {
	t.Parallel()

	rule := sim.SpellRule{ID: spPrismatic, ManaCost: 4, School: 1, MaxRange: 15, TargetsUnit: true,
		Damaging: true, DamageMin: 5, DamageMax: 5, Delivery: 2, EffectSpeed: 128}
	mage := sim.Entity{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100, Owner: 1, MaxMana: 100, Mana: 100,
		WeaponSpell: spPrismatic, WeaponSpellLevel: 40, AttackCharge: 3, Reach: 1,
		DamageBase: 5, AlwaysHits: true, ScanRange: 10, TokenSize: 1}
	foe := func(id sim.EntityID, x, y int32) sim.Entity {
		return sim.Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, Owner: 2, TokenSize: 1}
	}
	var rel sim.Relations
	rel.Set(1, 2, 1) // bit 0: hostile
	rel.Set(2, 1, 1)
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{mage, foe(2, 7, 1), foe(3, 3, 3), foe(4, 2, 5)}, nil, rel, nil, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	mw := spWorld(t)
	mw.world = w
	sheet := mw.projectiles.Sheet(36)
	releases := 0
	for tick := 0; tick < 80 && releases < 2; tick++ {
		mw.noteShotPreMoves()
		evs := sim.StepObserved(w, []sim.Command{sim.Attack(1, 2)})
		mw.observeCasts(evs)
		mw.advanceShotTrails()
		cells := spFigureCells(mw.boltDraws(w.Entities()), sheet)
		var victims []sim.CellPoint
		for _, ev := range evs {
			if ev.Spell == spPrismatic && ev.Weapon {
				victims = ev.Victims
			}
		}
		if victims == nil {
			continue
		}
		releases++
		if len(victims) < 3 {
			t.Fatalf("the release reached %v, want all three foes", victims)
		}
		want := make([]image.Point, len(victims))
		for k, v := range victims {
			want[k] = image.Pt(int(v.X), int(v.Y))
		}
		// An earlier release's record may still fly; this one's is the newest.
		records := flightRecords(mw)
		newest := records[0]
		for _, p := range records {
			if p.ID > newest.ID {
				newest = p
			}
		}
		var got []image.Point
		for _, b := range mw.recordPaths(newest, 0) {
			got = append(got, b.to)
		}
		if !reflect.DeepEqual(got, want) || len(cells) < len(want) {
			t.Fatalf("the release built figures to %v (drawn %v), want one per victim in order %v", got, cells, want)
		}
	}
	if releases < 2 {
		t.Errorf("releases %d, want 2", releases)
	}
}

// spSheetKinds counts a draw list by which sheet each stamp came off: the
// spell's own cast picture, or one of the two trail sheets. A burst sheet is
// neither and is not counted, because a burst is not what this compares.
func spSheetKinds(mw *mapWorld, spell uint16, d []ui.SpellBolt) map[string]int {
	cast := mw.projectiles.Sheet(data.CastPicture(int(spell)))
	out := map[string]int{"cast": 0, "trail": 0}
	for _, b := range d {
		switch {
		case b.Sheet == mw.projectiles.SmokeSheet(0) || b.Sheet == mw.projectiles.SmokeSheet(1):
			out["trail"]++
		case b.Sheet == cast:
			out["cast"]++
		}
	}
	return out
}

func spPositions(d []ui.SpellBolt) []image.Point {
	out := make([]image.Point, len(d))
	for i, b := range d {
		out[i] = b.Pos
	}
	return out
}

func spEqual(a, b []image.Point) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// spRiderWorld is a real simulation holding one carrier and one victim a cell
// apart, the carrier's weapon carrying spell. mage decides which weapon-borne
// arm the strike takes: a mana pool makes it the caster's replacement release,
// no mana pool makes it the fighter's rider.
//
// It is a real world and a real Step rather than a hand-written CastEvent,
// because the whole of this behaviour is which arm the simulation chose and a
// hand-written event would assert the choice this test is here to observe.
func spRiderWorld(t *testing.T, spell uint16, mage bool) []sim.CastEvent {
	t.Helper()

	// THE CARRIER IS RANGED, and that is the shipped case rather than a
	// convenience. A rider fires at the blow, so the two cells its object runs
	// between are the carrier's own weapon reach: a melee carrier's release
	// crosses one cell and a Boulder Thrower's crosses its firing range.
	rule := sim.SpellRule{ID: spell, MaxRange: 9, DamageMin: 6, DamageMax: 6,
		TargetsUnit: true, Damaging: true}
	carrier := sim.Entity{ID: 1, X: 0, Y: 0, HP: 100, MaxHP: 100, Owner: 1,
		WeaponSpell: spell, WeaponSpellLevel: 30, AttackCharge: 1, Reach: 6,
		DamageBase: 5, AlwaysHits: true, ScanRange: 8}
	if mage {
		carrier.MaxMana, carrier.Mana = 100, 100
	}
	victim := sim.Entity{ID: 2, X: 6, Y: 0, HP: 100, MaxHP: 100, Owner: 2}

	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical,
		nil, []sim.Entity{carrier, victim}, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	for range 16 {
		if ev := sim.StepObserved(w, []sim.Command{{Kind: sim.KindAttack, Entity: 1, X: 2}}); len(ev) > 0 {
			return ev
		}
	}
	t.Fatalf("no cast landed in 16 ticks for spell %d, mage=%v", spell, mage)
	return nil
}

func TestTheSimulationSeparatesTheTwoWeaponArms(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		name      string
		mage      bool
		wantRider bool
	}{
		{"a mage replaces its attack with the release", true, false},
		{"a fighter takes the weapon's rider", false, true},
	} {
		ev := spRiderWorld(t, spFireBall, c.mage)[0]
		if !ev.Weapon {
			t.Errorf("%s: the event is not marked Weapon", c.name)
		}
		if ev.Rider != c.wantRider {
			t.Errorf("%s: Rider = %v, want %v", c.name, ev.Rider, c.wantRider)
		}
	}
}

// spVictimIndex is which victim a stamp's far cell names, or -1.
func spVictimIndex(victims []sim.CellPoint, to image.Point) int {
	for k, v := range victims {
		if int(v.X) == to.X && int(v.Y) == to.Y {
			return k
		}
	}
	return -1
}

// spFigureCells is the far cell of every figure among draws on sheet, in first
// stamp order, and it fails a figure drawn twice by returning its cell twice.
func spFigureCells(draws []ui.SpellBolt, sheet *terrain.EffectSheet) []image.Point {
	var out []image.Point
	for i, d := range draws {
		if d.Sheet != sheet || (i > 0 && draws[i-1].Sheet == sheet && draws[i-1].To == d.To) {
			continue
		}
		out = append(out, d.To)
	}
	return out
}
