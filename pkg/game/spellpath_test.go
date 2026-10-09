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

// 1004: the bolt path, the smoke trail and the burst's wait.
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
// owner's report, walked: a Lightning object never moves, and the figure it
// draws reaches the target from its first frame rather than growing toward it.
func TestAPathPictureStandsStillAndSpansItsWholeSegment(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spLightning,
		FromX: 2, FromY: 2, ToX: 8, ToY: 5}})
	if len(mw.bolts) != 1 || mw.bolts[0].picture != 34 || mw.bolts[0].life != 13 {
		t.Fatalf("lightning spawned %+v, want one picture-34 object of 13 ticks", mw.bolts)
	}

	// The world holds no caster class, so the launch point is the caster's
	// cell centre (castLaunch); both ends are display pixels.
	from, to := image.Pt(2*32+16, 2*32+16), image.Pt(8*32+16, 5*32+16)
	for age := range 13 {
		spCheckFigure(t, fmt.Sprintf("age %d", age), mw.boltDraws(nil), from, to)
		mw.advanceBolts()
	}
	if got := mw.boltDraws(nil); len(got) != 0 {
		t.Errorf("the object outlived its own 13 ticks: %d stamps", len(got))
	}
}

// TestAPathIsRegeneratedWholeOnEveryTick is `MAGIC-BOLTLIST-071`'s replace: the
// engine rebuilds the point list on every driver tick, so no kink holds still.
func TestAPathIsRegeneratedWholeOnEveryTick(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spLightning,
		FromX: 0, FromY: 0, ToX: 9, ToY: 0}})

	var same int
	first := spPositions(mw.boltDraws(nil))
	for age := 1; age < 13; age++ {
		mw.advanceBolts()
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
		mw.advanceBolts()
	}
	if off == 0 {
		t.Error("every stamp of every tick stood on the straight line — the figure is an interpolation")
	}
}

// TestAPathObjectIsDeterministic keeps the figure out of the determinism story:
// it is drawn from the observation and the age, so one replay draws one picture
// and nothing here reads a clock.
func TestAPathObjectIsDeterministic(t *testing.T) {
	t.Parallel()

	run := func() []image.Point {
		mw := spWorld(t)
		mw.observeCasts([]sim.CastEvent{{Caster: 3, Target: 7, Spell: spPrismatic,
			FromX: 1, FromY: 1, ToX: 6, ToY: 4}})
		mw.advanceBolts()
		mw.advanceBolts()
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
	if len(mw.bolts) != 2 {
		t.Fatalf("two observations spawned %d objects", len(mw.bolts))
	}
	if mw.bolts[0].tag != 0 || mw.bolts[1].tag != 1 {
		t.Fatalf("the tags are %d and %d, want 0 and 1", mw.bolts[0].tag, mw.bolts[1].tag)
	}
	draws := mw.boltDraws(nil)
	byTag := map[int]bool{}
	for _, d := range draws {
		byTag[d.Frame] = true
	}
	// Call 1 of the normal route is phase 4; tag 1 adds five.
	if len(byTag) != 2 || !byTag[4] || !byTag[4+boltChainStride] {
		t.Errorf("the two sets drew frames %v, want 4 and %d", byTag, 4+boltChainStride)
	}
}

// TestATravellingObjectLeavesSixPastPositions is `MAGIC-TRAIL-073`: a queue of
// at most six PAST positions, oldest first, each on the sheet frame its own age
// names.
func TestATravellingObjectLeavesSixPastPositions(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	// Fire Arrow over nine cells: picture 10 divides by 200, so the object is
	// drawn for eleven ticks and the queue fills.
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 1,
		FromX: 0, FromY: 0, ToX: 9, ToY: 0}})
	life := mw.bolts[0].life
	if life < 7 {
		t.Fatalf("the object lives %d ticks, too short to fill a six-entry queue", life)
	}

	for age := range life {
		draws := mw.boltDraws(nil)
		want := min(age+1, 6) + 1 // the trail entries plus the object's own sprite
		if len(draws) != want {
			t.Fatalf("at age %d the object handed over %d drawables, want %d", age, len(draws), want)
		}
		head := draws[0]
		trail := draws[1:]
		for i, e := range trail {
			if e.Frame != i {
				t.Errorf("at age %d trail entry %d is frame %d, want its own age", age, i, e.Frame)
			}
			if e.Pos.X >= head.Pos.X {
				t.Errorf("at age %d trail entry %d stands at %d, not behind the object's %d",
					age, i, e.Pos.X, head.Pos.X)
			}
			if i > 0 && e.Pos.X >= trail[i-1].Pos.X {
				t.Errorf("at age %d the trail is not ordered newest first: %d after %d",
					age, e.Pos.X, trail[i-1].Pos.X)
			}
		}
		mw.advanceBolts()
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
		mw.advanceBolts()
	}
}

// TestFireBallsBurstIsNotAPresentationObject: the burst is a World record built
// when the area blasts (ANIM-111), drawn from the record. The cast spawns its
// flying object alone, and no burst stamp is drawn from the cast's own list.
func TestFireBallsBurstIsNotAPresentationObject(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spFireBall,
		FromX: 0, FromY: 0, ToX: 9, ToY: 0}})
	if len(mw.bolts) != 1 {
		t.Fatalf("fire ball spawned %d objects, want the cast alone", len(mw.bolts))
	}
	flight := mw.bolts[0].life
	for age := range flight + 22 {
		for _, d := range mw.boltDraws(nil) {
			if d.Sheet == mw.projectiles.Sheet(13) {
				t.Fatalf("at age %d the cast's list drew a burst stamp", age)
			}
		}
		mw.advanceBolts()
	}
}

// TestAWeaponBorneFireBallSpawnsNoBurstObject: a weapon-borne cast is drawn off
// the swing, and its burst is the World record built at the blast like any
// other Fire_Ball burst, so the release spawns no object.
func TestAWeaponBorneFireBallSpawnsNoBurstObject(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spFireBall, Weapon: true,
		FromX: 0, FromY: 0, ToX: 9, ToY: 0}})
	if len(mw.bolts) != 0 {
		t.Fatalf("a weapon-borne fire ball spawned %d objects, want none", len(mw.bolts))
	}
}

// spStaff is an actor releasing spell through a weapon, mid wind-up, with its
// victim four cells east.
func spStaff(spell uint16) []sim.Entity {
	return []sim.Entity{
		{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Owner: 1, WeaponSpell: spell,
			HasAttackTarget: true, AttackTarget: 2, AttackPhase: sim.AttackCasting, AttackCharge: 4},
		{ID: 2, X: 6, Y: 2, HP: 100, MaxHP: 100, Owner: 2},
	}
}

// TestAWeaponBorneLightningDrawsTheSameFigureABookCastDoes is the review's own
// counterexample. A staff-borne release is not a spellBolt in this tier's list —
// it is rebuilt from the attack cycle every frame — so before this it took a
// second draw path with no figure on it, and a staff Lightning still drew one
// travelling sprite. Thirteen shipped staves cast Lightning.
func TestAWeaponBorneLightningDrawsTheSameFigureABookCastDoes(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	ents := spStaff(spLightning)
	// The world holds no caster class, so the launch point is the cell centre.
	from, to := image.Pt(2*32+16, 2*32+16), image.Pt(6*32+16, 2*32+16)
	for swing := range 5 {
		mw.swing[1] = swing
		spCheckFigure(t, fmt.Sprintf("swing %d", swing), mw.weaponBoltDraws(ents), from, to)
	}
}

// TestAWeaponBorneFigureIsRegeneratedAcrossTheSwing is the swing clock standing
// in for the object's own age: the figure has no life of its own here, so the
// clock is what has to move it.
func TestAWeaponBorneFigureIsRegeneratedAcrossTheSwing(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	ents := spStaff(spLightning)
	var same int
	mw.swing[1] = 0
	prev := spInterior(mw.weaponBoltDraws(ents))
	for swing := 1; swing < 5; swing++ {
		mw.swing[1] = swing
		next := spInterior(mw.weaponBoltDraws(ents))
		if spEqual(prev, next) {
			same++
		}
		prev = next
	}
	if same > 1 {
		t.Errorf("%d of 4 swing ticks redrew the previous figure unchanged", same)
	}
}

// spInterior is a draw list's positions with the two endpoints dropped. The
// endpoints are the caster and the victim and do not move across a swing, so
// comparing them would report a figure where a single travelling sprite also
// changes position. A draw list too short to have an interior yields nothing,
// which compares equal to the next tick's nothing.
func spInterior(d []ui.SpellBolt) []image.Point {
	if len(d) < 3 {
		return nil
	}
	return spPositions(d[1 : len(d)-1])
}

// TestAWeaponBorneTravellingPictureLeavesItsTrail is the other half of the same
// wiring: the two pictures that travel leave a trail from a staff for the same
// reason they leave one from a book.
func TestAWeaponBorneTravellingPictureLeavesItsTrail(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	ents := spStaff(1) // Fire Arrow, picture 10
	for swing := range 5 {
		mw.swing[1] = swing
		var trail int
		for _, d := range mw.weaponBoltDraws(ents) {
			if d.Sheet == mw.projectiles.SmokeSheet(0) {
				trail++
			}
		}
		if want := min(swing+1, 6); trail != want {
			t.Errorf("at swing %d the staff left %d trail puffs, want %d", swing, trail, want)
		}
	}
}

// TestAWeaponBorneReleaseAgreesWithABookCastOnWhatEachPictureDraws is the
// property the fix is really about: the two producers must not disagree about
// whether a picture is a figure, a sprite, or a sprite with a trail.
func TestAWeaponBorneReleaseAgreesWithABookCastOnWhatEachPictureDraws(t *testing.T) {
	t.Parallel()

	for _, spell := range []uint16{1, spFireBall, spLightning, spPrismatic} {
		// Both sides run through their own production producer, and neither
		// restates the other's arithmetic: the staff through weaponBoltDraws
		// and the book through boltDraws, each after one tick of its clock.
		staffWorld := spWorld(t)
		staffWorld.swing[1] = 1
		staff := spSheetKinds(staffWorld, spell, staffWorld.weaponBoltDraws(spStaff(spell)))

		bookWorld := spWorld(t)
		bookWorld.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spell,
			FromX: 2, FromY: 2, ToX: 6, ToY: 2}})
		bookWorld.advanceBolts()
		book := spSheetKinds(bookWorld, spell, bookWorld.boltDraws(nil))

		wantFigure := data.CastDrawsPath(data.CastPicture(int(spell)))
		if got := staff["cast"] > 1; got != wantFigure {
			t.Errorf("spell %d: a staff drew %d cast stamps, figure=%v, want figure=%v",
				spell, staff["cast"], got, wantFigure)
		}
		for _, kind := range []string{"cast", "trail"} {
			if (staff[kind] > 0) != (book[kind] > 0) {
				t.Errorf("spell %d: a staff drew %d %s stamps and a book drew %d",
					spell, staff[kind], kind, book[kind])
			}
		}
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

// TestARiderReleaseSpawnsAnObjectThatFlies is the review's second
// counterexample. A rider carrier never reaches AttackCasting, so
// weaponBoltDraws never draws it; before this it was skipped by observeCasts
// with the caster's release too, so nothing flew and the burst appeared at the
// target on its own. Three shipped Boulder Throwers carry Fire Ball on this arm.
func TestARiderReleaseSpawnsAnObjectThatFlies(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	mw.observeCasts(spRiderWorld(t, spFireBall, false))

	if len(mw.bolts) == 0 {
		t.Fatal("a rider release put no object on the map")
	}
	var flew bool
	var prev image.Point
	for tick := 0; len(mw.bolts) > 0 && tick < 32; tick++ {
		for _, d := range mw.boltDraws(nil) {
			if d.Sheet == mw.projectiles.Sheet(data.CastPicture(spFireBall)) {
				if tick > 0 && d.Pos != prev {
					flew = true
				}
				prev = d.Pos
			}
		}
		mw.advanceBolts()
	}
	if !flew {
		t.Error("the rider's object never moved — it is a travelling picture and must cross")
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

// TestARiderBurstWaitsForItsProjectile is the second half of the counterexample.
// A caster's replacement release is already at the victim on the release tick
// and waits nothing; a rider's object has to cross first, so its burst waits
// exactly as a book cast's does.
func TestARiderBurstWaitsForItsProjectile(t *testing.T) {
	t.Parallel()

	rider := spRiderWorld(t, spFireBall, false)[0]
	caster := spRiderWorld(t, spFireBall, true)[0]
	mw := spWorld(t)
	from := image.Point{X: int(rider.FromX), Y: int(rider.FromY)}
	to := image.Point{X: int(rider.ToX), Y: int(rider.ToY)}

	if got := mw.burstDelay(rider, from, to); got <= 0 {
		t.Errorf("a rider burst waits %d ticks, want its projectile's flight", got)
	}
	if got := mw.burstDelay(caster, from, to); got != 0 {
		t.Errorf("a caster's replacement release waits %d ticks, want 0", got)
	}
}

// TestAWeaponBornePrismaticSprayVariesItsPhaseBlockByCarrier is the tag the
// constructed object never set. Picture 36's sheet holds seven blocks of five
// phases and the block is chosen by the record tag; with the field left unset
// every staff on the map drew block 0.
func TestAWeaponBornePrismaticSprayVariesItsPhaseBlockByCarrier(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	seen := map[int]bool{}
	for id := sim.EntityID(1); id <= 3; id++ {
		ents := spStaff(spPrismatic)
		ents[0].ID = id
		mw.swing[id] = 1
		for _, d := range mw.weaponBoltDraws(ents) {
			seen[d.Frame] = true
		}
	}
	if len(seen) < 2 {
		t.Errorf("three carriers drew frames %v — the phase block does not vary by carrier", seen)
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

func TestAPrismaticSprayDrawsOneFigurePerVictimInItsOwnColour(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	victims := []sim.CellPoint{{X: 8, Y: 5}, {X: 9, Y: 6}, {X: 7, Y: 3}}
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spPrismatic,
		FromX: 2, FromY: 2, ToX: 8, ToY: 5, Victims: victims}})

	if len(mw.bolts) != len(victims) {
		t.Fatalf("a spray over %d victims spawned %d objects, want one each", len(victims), len(mw.bolts))
	}
	for k, v := range victims {
		if got, want := mw.bolts[k].to, image.Pt(int(v.X), int(v.Y)); got != want {
			t.Errorf("figure %d runs to %v, want its own victim's %v", k, got, want)
		}
		if mw.bolts[k].tag != k {
			t.Errorf("figure %d carries tag %d, want the victim's own loop index", k, mw.bolts[k].tag)
		}
	}

	// The drawables are read, not the objects: a stamp carries the far cell of
	// the figure it belongs to, so the whole spray can be sorted into figures
	// without the producer being asked which stamp it just emitted.
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
		mw.advanceBolts()
	}
	// Each figure holds ONE colour block for its whole life — the ramp walks the
	// five phases inside the block and the block does not move — and no two
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

// spVictimIndex is which victim a stamp's far cell names, or -1.
func spVictimIndex(victims []sim.CellPoint, to image.Point) int {
	for k, v := range victims {
		if int(v.X) == to.X && int(v.Y) == to.Y {
			return k
		}
	}
	return -1
}

// TestAWeaponBornePrismaticReleaseDrawsOneFigurePerVictim is the owner's
// staff spray: the release reported three victims and only the aimed one was
// drawn. A caster's release that reports a list now spawns the same set a
// book cast of the same list spawns — same count, cells, tags and life.
func TestAWeaponBornePrismaticReleaseDrawsOneFigurePerVictim(t *testing.T) {
	t.Parallel()

	victims := []sim.CellPoint{{X: 8, Y: 5}, {X: 9, Y: 6}, {X: 7, Y: 3}}
	ev := sim.CastEvent{Caster: 1, Target: 2, Spell: spPrismatic, Weapon: true,
		FromX: 2, FromY: 2, ToX: 8, ToY: 5, Victims: victims}
	staff, book := spWorld(t), spWorld(t)
	staff.observeCasts([]sim.CastEvent{ev})
	ev.Weapon = false
	book.observeCasts([]sim.CastEvent{ev})

	got, want := spPictureBolts(staff, 36), spPictureBolts(book, 36)
	if len(got) != len(victims) {
		t.Fatalf("a staff spray over %d victims spawned %d figures, want one each", len(victims), len(got))
	}
	for k, v := range victims {
		if got[k].to != image.Pt(int(v.X), int(v.Y)) || got[k].tag != k {
			t.Errorf("figure %d runs to %v with tag %d, want %v and tag %d", k, got[k].to, got[k].tag, v, k)
		}
		if got[k].life != want[k].life {
			t.Errorf("figure %d lives %d ticks, want the book cast's %d", k, got[k].life, want[k].life)
		}
	}
}

// TestAWeaponBorneReleaseWithNoVictimListSpawnsNothing keeps every other
// caster release on the live wind-up draw alone.
func TestAWeaponBorneReleaseWithNoVictimListSpawnsNothing(t *testing.T) {
	t.Parallel()

	for _, spell := range []uint16{spPrismatic, spLightning, spFireBall} {
		mw := spWorld(t)
		mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spell, Weapon: true,
			FromX: 2, FromY: 2, ToX: 8, ToY: 5}})
		if got := spPictureBolts(mw, data.CastPicture(int(spell))); len(got) != 0 {
			t.Errorf("spell %d: a listless caster release spawned %d cast objects, want none", spell, len(got))
		}
	}
}

// TestARiderReleaseSpawnsOneObjectAsBefore pins the rider arm's own set: one
// cast object at the observation's tick index, whatever the caster arm does.
func TestARiderReleaseSpawnsOneObjectAsBefore(t *testing.T) {
	t.Parallel()

	mw := spWorld(t)
	ev := spRiderWorld(t, spFireBall, false)[0]
	mw.observeCasts([]sim.CastEvent{ev})
	got := spPictureBolts(mw, data.CastPicture(spFireBall))
	if len(got) != 1 || got[0].tag != 0 || got[0].to != image.Pt(int(ev.ToX), int(ev.ToY)) {
		t.Fatalf("a rider release spawned %+v, want one object with tag 0 to its target", got)
	}
}

// spPictureBolts is the live objects of one picture, in spawn order.
func spPictureBolts(mw *mapWorld, picture int) []spellBolt {
	var out []spellBolt
	for _, b := range mw.bolts {
		if b.picture == picture {
			out = append(out, b)
		}
	}
	return out
}

// TestAStaffSprayDrawsItsVictimsOnceEachAcrossTheRelease drives the real step.
// A mage whose weapon carries Prismatic Spray attacks one foe while two more
// stand in view. Until the release the live wind-up draws one figure to the
// aimed foe; on the release tick the phase leaves casting, the wind-up stops,
// and the release's set draws one figure per victim in victim order. The next
// wind-up starts while that set still lives and is not drawn until it ends,
// so no tick draws a wind-up figure beside a set.
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
	sheet := mw.projectiles.Sheet(36)
	releases, windUp, held := 0, 0, 0
	for tick := 0; tick < 80 && releases < 2; tick++ {
		evs := sim.StepObserved(w, []sim.Command{sim.Attack(1, 2)})
		mw.observeCasts(evs)
		ents := w.Entities()
		cells := spFigureCells(mw.boltDraws(ents), sheet)
		live := spFigureCells(mw.weaponBoltDraws(ents), sheet)
		if len(live) != 0 && len(cells) != len(live) {
			t.Fatalf("tick %d drew the wind-up %v beside a live set: %v", tick, live, cells)
		}
		var victims []sim.CellPoint
		for _, ev := range evs {
			if ev.Spell == spPrismatic && ev.Weapon {
				victims = ev.Victims
			}
		}
		switch {
		case victims != nil:
			releases++
			if len(victims) < 3 {
				t.Fatalf("the release reached %v, want all three foes", victims)
			}
			want := make([]image.Point, len(victims))
			for k, v := range victims {
				want[k] = image.Pt(int(v.X), int(v.Y))
			}
			// An earlier release's set may still live; this one's is appended last.
			if len(cells) < len(want) || !reflect.DeepEqual(cells[len(cells)-len(want):], want) || len(live) != 0 {
				t.Fatalf("the release tick drew figures to %v (wind-up %v), want one per victim in order %v",
					cells, live, want)
			}
		case ents[0].AttackPhase == sim.AttackCasting && len(live) == 0:
			held++
		case ents[0].AttackPhase == sim.AttackCasting:
			windUp++
		}
		mw.advanceBolts()
	}
	if releases < 2 || windUp == 0 || held == 0 {
		t.Errorf("releases %d, wind-up ticks drawn %d, held behind a live set %d: want 2 and both kinds of tick",
			releases, windUp, held)
	}
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
