package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The spell in flight and the two clocks.

// sbSheets is a synthetic art bundle shaped like the four shipped rows this
// package's tests reach: the rotating cast sheet at picture 10, the plain cast
// sheet at picture 20, and a burst sheet at picture 13. No archive is opened and
// no pixel is decoded — a frame's size is all any assertion here reads.
func sbSheets() *terrain.EffectSet {
	frames := func(n int) []*terrain.EffectFrame {
		out := make([]*terrain.EffectFrame, n)
		for i := range out {
			out[i] = &terrain.EffectFrame{Width: 8, Height: 8, Pixels: make([]color.RGBA, 64)}
		}
		return out
	}
	return &terrain.EffectSet{Sheets: map[int]*terrain.EffectSheet{
		// firebolt: 4 phases over 9 stored facings, the halving bit set.
		10: {Frames: frames(36), Phases: 4, RotationPhases: 16, Flip: true, CenterX: 32, CenterY: 32},
		// healing and drain: one rotation phase, so the frame is the phase alone.
		20: {Frames: frames(8), Phases: 7, RotationPhases: 1, CenterX: 8, CenterY: 8},
		30: {Frames: frames(9), Phases: 9, RotationPhases: 1, CenterX: 8, CenterY: 8},
		// fireexpl: the burst behind spell 2.
		13: {Frames: frames(11), Phases: 11, RotationPhases: 1, CenterX: 64, CenterY: 64},
	}}
}

// sbWorld is a mapWorld with only the memories a spell object reads: the swing
// clocks, the bolt list and the art bundle. It steps no world and opens no
// install.
func sbWorld(t *testing.T) *mapWorld {
	t.Helper()
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, nil, nil, nil,
		[]sim.SpellRule{{ID: 1, School: 1, MaxRange: 7, DamageMin: 4, DamageMax: 8,
			TargetsUnit: true, Damaging: true}})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	return &mapWorld{world: w, swing: map[sim.EntityID]int{},
		phase: map[sim.EntityID]sim.AttackPhase{}, castRun: map[sim.EntityID]castRun{},
		projectiles: sbSheets()}
}

// TestABookCastAndAStaffReleaseEachBuildOneRecord: a book cast and a staff's
// cast-diverted release each reach the cast producer (SAV-1129, ANIM-147).
func TestABookCastAndAStaffReleaseEachBuildOneRecord(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	mw.observeCasts([]sim.CastEvent{
		{Caster: 1, Target: 2, Spell: 1, School: 1, FromX: 2, FromY: 2, ToX: 5, ToY: 2},
		{Caster: 3, Target: 4, Spell: 1, School: 1, FromX: 1, FromY: 1, ToX: 2, ToY: 1, Weapon: true},
	})
	records := flightRecords(mw)
	if len(records) != 2 {
		t.Fatalf("the tick built %d records, want the book cast's and the staff's", len(records))
	}
	// Spell 1's cast picture is 2*1+8 and its flight over three cells is
	// 3*256/200 calls; the first has run.
	var book sim.SavedProjectile
	for _, p := range records {
		if p.ID == 0 {
			book = p
		}
	}
	if book.Picture != 10 || book.ActionSegments != 2 || book.ActionX != 5*256+128 || book.ActionY != 2*256+128 {
		t.Errorf("the book record is %+v, want picture 10 aimed at (5,2) with 2 of 3 calls left", book)
	}
}

// TestEveryCastTakesACounterIDAndSevenPicturesFly is ANIM-147 and SAV-1204:
// every cast builds a record and takes its counter id (Teleport two); a
// picture outside the segment switch retires on its first call.
func TestEveryCastTakesACounterIDAndSevenPicturesFly(t *testing.T) {
	t.Parallel()

	flying := map[int]bool{1: true, 2: true, 6: true, 11: true, 13: true, 14: true, 26: true}
	for spell := 1; spell <= 28; spell++ {
		mw := sbWorld(t)
		mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: uint16(spell),
			FromX: 0, FromY: 0, ToX: 5, ToY: 0}})
		want, ids := 0, uint16(1)
		if flying[spell] {
			want++
		}
		if spell == 26 {
			want, ids = 2, 2
		}
		// Heal's picture 20 has one segment: the first call spends it.
		if spell == 6 {
			want = 1
		}
		if got := len(flightRecords(mw)); got != want {
			t.Errorf("spell %d left %d records armed, want %d", spell, got, want)
		}
		if got := mw.world.SavedProjectiles().FreeIndex; got != ids {
			t.Errorf("spell %d moved the counter to %d, want %d", spell, got, ids)
		}
	}
}

// TestTeleportBuildsStationaryRecordsAtBothEnds is MAGIC-265 and ANIM-149: a
// record at the caster and one at the destination, and the picture-60 arm
// writes no position.
func TestTeleportBuildsStationaryRecordsAtBothEnds(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 26,
		FromX: 1, FromY: 1, ToX: 6, ToY: 1}})
	records := flightRecords(mw)
	if len(records) != 2 {
		t.Fatalf("teleport built %d records, want two", len(records))
	}
	start := map[uint16][2]int32{}
	for _, p := range records {
		start[p.ID] = [2]int32{p.X, p.Y}
		if p.Picture != teleportPicture {
			t.Errorf("record %+v, want picture %d", p, teleportPicture)
		}
	}
	if start[0] != [2]int32{1*256 + 128, 1*256 + 128} || start[1] != [2]int32{6*256 + 128, 1*256 + 128} {
		t.Errorf("the records stand at %v, want the caster's and the destination's cell centres", start)
	}
	for range 19 {
		flightStep(mw)
		for _, p := range flightRecords(mw) {
			if [2]int32{p.X, p.Y} != start[p.ID] {
				t.Fatalf("record %d moved to (%d,%d), want it standing", p.ID, p.X, p.Y)
			}
		}
	}
}

func TestFireBallBurstHoldsItsLastFrameInsteadOfWrapping(t *testing.T) {
	t.Parallel()
	mw := sbWorld(t)
	mw.world.ReleaseAreaBurst(sim.AreaBurst{CellX: 4, CellY: 4, Picture: fireBallBurstPicture, Segments: 22, Phases: 11})
	var frames []int
	for range 30 {
		for _, d := range mw.savedProjectileDraws() {
			frames = append(frames, d.Frame)
		}
		flightStep(mw)
	}
	if len(frames) == 0 || frames[0] != 0 || frames[len(frames)-1] != 10 {
		t.Fatalf("the burst drew frames %v, want frame 0 first and terminal frame 10 last", frames)
	}
}

func TestABurstIsBuiltOnlyWhereTheGameShipsOne(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		spell uint16
		want  int
	}{
		// Fire Arrow's burst picture is 11 and Heal's is 21; the bundle holds
		// neither, exactly as the shipped registry holds neither.
		{"fire arrow", 1, 0},
		{"heal", 6, 0},
		// Fire Ball's is 13, which the game does ship.
		{"fire ball", 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := sbWorld(t)
			mw.releaseAreaBursts(sim.AreaPaint{Spell: tc.spell, Cells: []sim.CellPoint{{X: 4, Y: 4}}})
			records := flightRecords(mw)
			if got := len(records); got != tc.want {
				t.Fatalf("spell %d built %d bursts, want %d", tc.spell, got, tc.want)
			}
			for _, p := range records {
				if p.X != 4*256+128 || p.Y != 4*256+128 || p.ActionTarget != 0 {
					t.Errorf("the burst is %+v, want it at the centre of (4,4) with no target", p)
				}
			}
		})
	}
}

func TestABurstStandsStillForItsWholeLife(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	mw.world.ReleaseAreaBurst(sim.AreaBurst{CellX: 4, CellY: 4, Picture: fireBallBurstPicture, Segments: 22, Phases: 11})
	drawn := 0
	for range 30 {
		draws := mw.boltDraws(nil)
		if len(draws) > 1 {
			t.Fatalf("the burst handed over %d drawables, want one", len(draws))
		}
		if len(draws) == 1 {
			drawn++
			if draws[0].Pos != (image.Point{X: 4*ui.ShotScale + 128, Y: 4*ui.ShotScale + 128}) {
				t.Errorf("the burst stands at %v, want the centre of (4,4)", draws[0].Pos)
			}
		}
		flightStep(mw)
	}
	if drawn == 0 || drawn > 22 {
		t.Errorf("the burst drew on %d calls, want between 1 and its 22 segments", drawn)
	}
}

func TestAFireArrowCrossesInItsPicturesOwnLength(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 1,
		FromX: 0, FromY: 0, ToX: 5, ToY: 0}})

	var seen []int
	for len(flightRecords(mw)) > 0 {
		draws := mw.boltDraws(nil)
		if len(draws) != 1 {
			t.Fatalf("at age %d the viewer is handed %d drawables, want one", len(seen), len(draws))
		}
		seen = append(seen, draws[0].Pos.X)
		flightStep(mw)
		if len(seen) > 20 {
			t.Fatal("the record outlived twenty calls")
		}
	}
	// 5 cells of 256 over picture 10's divisor 200 is 6 segments: six calls
	// move it, and the seventh reaps the record (ANIM-PROJ-025).
	if len(seen) != 6 {
		t.Fatalf("the record was drawn %d times, want 6", len(seen))
	}
	if want := 5*ui.ShotScale + 128; seen[len(seen)-1] != want {
		t.Errorf("on its last call the record stands at %d, want the aim's %d", seen[len(seen)-1], want)
	}
	for i := 1; i < len(seen); i++ {
		if seen[i] <= seen[i-1] {
			t.Errorf("the record stands at %d after %d, want it advancing", seen[i], seen[i-1])
		}
	}
}

// TestHealsOneTickCastObjectUsesThePublishedNoBlitArm keeps the record
// distinct from the target-local positive-Heal feedback: picture 20 has one
// segment, and its ordinary draw arm deliberately submits no art.
func TestHealsOneTickCastObjectUsesThePublishedNoBlitArm(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 6,
		FromX: 0, FromY: 0, ToX: 3, ToY: 0}})
	if got := mw.boltDraws(nil); len(got) != 0 {
		t.Fatalf("heal handed over %d ordinary drawables, want the picture-20 no-blit", len(got))
	}
	if got := mw.world.SavedProjectiles().FreeIndex; got != 1 {
		t.Fatalf("Heal took %d counter ids, want one", got)
	}
	flightStep(mw)
	if got := flightRecords(mw); len(got) != 0 {
		t.Errorf("heal outlived its single segment: %+v", got)
	}
}

func TestAnObjectWithNoSheetIsHandedOverAtAll(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	// Spell 14's picture 36 flies, and this bundle holds no sheet for it.
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 14,
		FromX: 0, FromY: 0, ToX: 3, ToY: 0}})
	if got := len(flightRecords(mw)); got != 1 {
		t.Fatalf("prismatic spray built %d records, want one", got)
	}
	if got := mw.boltDraws(nil); len(got) != 0 {
		t.Errorf("a record whose picture names no sheet was handed over: %+v", got)
	}
}

// TestAWindUpDrawsNoFlight: a staff's cast record is built at its release, so
// a caster standing in the casting wind-up has nothing in flight.
func TestAWindUpDrawsNoFlight(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	caster := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Owner: 1,
		WeaponSpell: 1, HasAttackTarget: true, AttackTarget: 2,
		AttackPhase: sim.AttackCasting, AttackCharge: 4}
	victim := sim.Entity{ID: 2, X: 6, Y: 2, HP: 100, MaxHP: 100, Owner: 2}
	for swing := range 5 {
		mw.swing[1] = swing
		if got := mw.boltDraws([]sim.Entity{caster, victim}); len(got) != 0 {
			t.Fatalf("at swing %d the wind-up drew %+v, want nothing in flight", swing, got)
		}
	}
}

func TestBothWindUpPhasesAreASwing(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		phase sim.AttackPhase
		want  bool
	}{
		{sim.AttackCharging, true},
		{sim.AttackCasting, true},
		{sim.AttackReady, false},
		{sim.AttackRelaxing, false},
	} {
		if got := windUp(tc.phase); got != tc.want {
			t.Errorf("windUp(%v) = %v, want %v", tc.phase, got, tc.want)
		}
	}
}

func TestACastingWindUpStartsAFreshSwingRun(t *testing.T) {
	t.Parallel()

	for _, phase := range []sim.AttackPhase{sim.AttackCasting, sim.AttackCharging} {
		attacker := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Owner: 1, Reach: 1,
			HasAttackTarget: true, AttackTarget: 2, AttackPhase: phase, AttackCharge: 4}
		victim := sim.Entity{ID: 2, X: 3, Y: 2, HP: 100, MaxHP: 100, Owner: 2}
		w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, nil,
			[]sim.Entity{attacker, victim}, nil, nil)
		if err != nil {
			t.Fatalf("NewSpelledWorld: %v", err)
		}
		mw := &mapWorld{world: w, swing: map[sim.EntityID]int{},
			phase: map[sim.EntityID]sim.AttackPhase{}}
		// The recorded phase is the zero AttackReady, so this pass is the one
		// that enters the wind-up.
		mw.swing[1] = 99
		mw.advanceSwings()
		if got := mw.swing[1]; got != 0 {
			t.Errorf("entering %v left the swing clock at %d, want a fresh run at 0", phase, got)
		}
		// The phase is unchanged on the next pass, so the run advances.
		mw.advanceSwings()
		if got := mw.swing[1]; got != 1 {
			t.Errorf("the second tick of a %v run left the clock at %d, want 1", phase, got)
		}
	}
}

func TestArmingAnAutocastQueuesTheCommandWithoutTakingTheUnitOver(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	mw.commanded = map[sim.EntityID]bool{}
	mw.setAutocast(7, 6)

	if len(mw.pending) != 1 {
		t.Fatalf("the toggle queued %d commands, want one", len(mw.pending))
	}
	got := mw.pending[0]
	if got.Kind != sim.KindAutocast || got.Entity != 7 || got.X != 6 {
		t.Errorf("the command is %+v, want KindAutocast on entity 7 with spell 6", got)
	}
	if mw.commanded[7] {
		t.Error("arming an autocast marked the unit commanded, taking it off the mission script")
	}
}

func TestABookCastPlaysTheCastersOwnAttackRun(t *testing.T) {
	t.Parallel()

	caster := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Owner: 1, Class: 3}
	target := sim.Entity{ID: 2, X: 4, Y: 2, HP: 100, MaxHP: 100, Owner: 2}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, nil,
		[]sim.Entity{caster, target}, nil,
		[]sim.SpellRule{{ID: 1, School: 1, MaxRange: 7, DamageMin: 4, DamageMax: 8,
			TargetsUnit: true, Damaging: true}})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	mw := &mapWorld{world: w, swing: map[sim.EntityID]int{},
		phase: map[sim.EntityID]sim.AttackPhase{}, castRun: map[sim.EntityID]castRun{}}

	if mw.casting(1) {
		t.Fatal("the caster is in a run before any cast")
	}
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 1, School: 1, Owner: 1,
		FromX: 2, FromY: 2, ToX: 4, ToY: 2}})

	if !mw.casting(1) {
		t.Fatal("an applied book cast started no run on its caster")
	}
	if got := mw.castRun[1]; got.left != castRunFallbackTicks || got.span != castRunFallbackTicks {
		t.Errorf("the run is %+v, want %d ticks over a span of %d",
			got, castRunFallbackTicks, castRunFallbackTicks)
	}
	// The clock is seeded one below zero, so advanceSwings' own increment lands
	// the run on its first frame.
	if got := mw.swing[1]; got != -1 {
		t.Errorf("the swing clock is %d at the start of a run, want -1", got)
	}

	// The run holds the clock open for a caster holding no victim, and the
	// frames advance one per tick.
	for tick := range castRunFallbackTicks {
		mw.advanceSwings()
		mw.advanceCastRuns()
		if tick < castRunFallbackTicks-1 {
			if !mw.casting(1) {
				t.Fatalf("the run ended after %d ticks, want %d", tick+1, castRunFallbackTicks)
			}
			if got := mw.swing[1]; got != tick {
				t.Errorf("at tick %d the swing clock is %d, want %d", tick, got, tick)
			}
		}
	}
	if mw.casting(1) {
		t.Errorf("the run outlived its own %d ticks", castRunFallbackTicks)
	}
	// And with the run gone the clock is returned to zero by the next pass.
	mw.advanceSwings()
	if got := mw.swing[1]; got != 0 {
		t.Errorf("the swing clock is %d after the run ended, want 0", got)
	}
}

// A long Fire Arrow flight is the owner's reported failure shape: the actor
// must begin one attack/cast run when the one CastEvent arrives, then become
// idle while the projectile continues. The projectile is deliberately much
// longer than the eight-tick floor so a run restarted from its lifetime would
// be visible here.
func TestOneFireArrowEventStartsOneSwingAndDoesNotRestartDuringFlight(t *testing.T) {
	t.Parallel()
	caster := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Owner: 1, AttackCharge: 3}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 64, Height: 16}, sim.ModeCanonical, nil,
		[]sim.Entity{caster}, nil, []sim.SpellRule{{ID: 1, School: 1, MaxRange: 40,
			DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	mw := &mapWorld{world: w, swing: map[sim.EntityID]int{}, phase: map[sim.EntityID]sim.AttackPhase{},
		castRun: map[sim.EntityID]castRun{}, projectiles: sbSheets()}
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 1, Owner: 1,
		FromX: 2, FromY: 2, ToX: 42, ToY: 2}})
	if r := flightRecords(mw); len(r) != 1 || int(r[0].ActionSegments) < castRunFallbackTicks {
		t.Fatalf("setup record = %+v, want one flight longer than %d ticks", r, castRunFallbackTicks)
	}

	starts := 0
	wasCasting := false
	for len(flightRecords(mw)) > 0 {
		casting := mw.casting(1)
		if casting && !wasCasting {
			starts++
		}
		wasCasting = casting
		mw.advanceSwings()
		if !casting && mw.swing[1] != 0 {
			t.Fatalf("projectile still flies after the run, but swing clock restarted at %d", mw.swing[1])
		}
		flightStep(mw)
		mw.advanceCastRuns()
	}
	if starts != 1 {
		t.Fatalf("one CastEvent began %d swing runs, want exactly one", starts)
	}
}

// TestTheManualSpellbookPathReleasesOneFireArrowAndStartsOneSwing runs through
// the real mapWorld queue and sim wind-up. It guards both owning boundaries:
// one manual order yields one release/application, and the client consumes that
// release once even while the decoded long projectile remains in flight.
func TestTheManualSpellbookPathReleasesOneFireArrowAndStartsOneSwing(t *testing.T) {
	caster := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Owner: 1,
		Mind: 30, Mana: 50, MaxMana: 50, KnownSpells: 1 << 1, AttackCharge: 12, AttackRelax: 4,
		ScanRange: 19}
	target := sim.Entity{ID: 2, X: 15, Y: 2, HP: 100, MaxHP: 100, Owner: 2, DyingTime: 200}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 64, Height: 16}, sim.ModeCanonical, nil,
		[]sim.Entity{caster, target}, nil, []sim.SpellRule{{ID: 1, ManaCost: 3, School: 1,
			MaxRange: 40, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	mw := &mapWorld{world: w, swing: map[sim.EntityID]int{}, phase: map[sim.EntityID]sim.AttackPhase{},
		castRun: map[sim.EntityID]castRun{}, projectiles: sbSheets(), commanded: map[sim.EntityID]bool{},
		prev: map[sim.EntityID]image.Point{}, died: map[sim.EntityID]int{}, walk: map[sim.EntityID]walkClock{},
		fog: newFogPlane(64, 16)}
	mw.attackOrCast(1, 2, 1, 0, 0, false)

	starts := 0
	wasCasting := false
	maxBolts := 0
	for frame := 0; frame < 128; frame++ {
		cmds := mw.commands()
		mw.pending = mw.pending[:0]
		mw.observeCasts(sim.StepObserved(w, cmds))
		casting := mw.casting(1)
		if casting && !wasCasting {
			starts++
		}
		wasCasting = casting
		if n := len(flightRecords(mw)); n > maxBolts {
			maxBolts = n
		}
		mw.advanceSwings()
		mw.advanceCastRuns()
		if frame > 12 && len(flightRecords(mw)) == 0 && !mw.casting(1) {
			break
		}
	}
	ents := w.Entities()
	if starts != 1 || maxBolts != 1 {
		t.Fatalf("manual cast started %d swing run(s), at most %d bolt(s); want one and one", starts, maxBolts)
	}
	if ents[0].Mana != 47 || ents[1].HP >= 100 {
		t.Fatalf("release result mana=%d target hp=%d, want one paid application", ents[0].Mana, ents[1].HP)
	}
}

func TestAWeaponBorneCastStartsNoSecondRun(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 1, Weapon: true}})
	if mw.casting(1) {
		t.Error("a weapon-borne release started a cast run of its own")
	}
}

func TestARunIsScaledToTheIntervalItsProjectileCrossesIn(t *testing.T) {
	t.Parallel()

	// A span longer than the track holds each frame for several ticks.
	for _, tc := range []struct{ elapsed, span, track, want int }{
		{0, 16, 4, 0}, {3, 16, 4, 0}, {4, 16, 4, 1}, {12, 16, 4, 3},
		// A span shorter than the track skips frames so the whole run plays.
		{0, 4, 16, 0}, {1, 4, 16, 4}, {3, 4, 16, 12},
		// A clock past its own span shows the last frame, never the first.
		{99, 4, 16, 15}, {16, 16, 4, 3},
		// Degenerate inputs answer the first frame rather than panicking.
		{-1, 8, 8, 0}, {4, 0, 8, 8 - 1}, {4, 8, 0, 0},
	} {
		if got := scaleRun(tc.elapsed, tc.span, tc.track); got != tc.want {
			t.Errorf("scaleRun(%d, %d, %d) = %d, want %d",
				tc.elapsed, tc.span, tc.track, got, tc.want)
		}
	}
}

func TestOnlyACastScalesItsRun(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	casting := sim.Entity{ID: 1, HP: 100, MaxHP: 100, HasAttackTarget: true,
		AttackPhase: sim.AttackCasting, AttackCharge: 12}
	if span, ok := mw.castSwingSpan(casting); !ok || span != 12 {
		t.Errorf("a casting attacker answers (%d, %v), want (12, true)", span, ok)
	}

	blow := casting
	blow.AttackPhase = sim.AttackCharging
	if _, ok := mw.castSwingSpan(blow); ok {
		t.Error("a melee blow scales its run; the owner's rule is about magic projectiles")
	}

	// A book caster answers its run's own span, which is floored at eight.
	mw.castRun[2] = castRun{left: 3, span: 20}
	if span, ok := mw.castSwingSpan(sim.Entity{ID: 2, HP: 100, MaxHP: 100}); !ok || span != 20 {
		t.Errorf("a book caster answers (%d, %v), want (20, true)", span, ok)
	}
	if _, ok := mw.castSwingSpan(sim.Entity{ID: 3, HP: 100, MaxHP: 100}); ok {
		t.Error("an entity in no run and no wind-up scales a run")
	}
}

// TestABookCastRunIsNeverShorterThanTheFloor is the owner's eight, measured
// where it is applied: a caster whose own charge is faster still swings for
// eight, so a projectile never outruns its swing.
func TestABookCastRunIsNeverShorterThanTheFloor(t *testing.T) {
	t.Parallel()

	for _, charge := range []int32{0, 1, 7, 8, 20} {
		caster := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Owner: 1, AttackCharge: charge}
		w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, nil,
			[]sim.Entity{caster}, nil, nil)
		if err != nil {
			t.Fatalf("NewSpelledWorld: %v", err)
		}
		mw := &mapWorld{world: w, swing: map[sim.EntityID]int{},
			phase: map[sim.EntityID]sim.AttackPhase{}, castRun: map[sim.EntityID]castRun{}}
		want := int(charge)
		if want < castRunFallbackTicks {
			want = castRunFallbackTicks
		}
		if got := mw.startCastRun(1); got != want {
			t.Errorf("a charge of %d gave a run span of %d, want %d", charge, got, want)
		}
	}
}

func TestShowerUsesTheDecodedCohortCountsAndEllipse(t *testing.T) {
	t.Parallel()
	if healBurstLife != 32 || healSpawnTicks != 25 || healParticleLife != 8 {
		t.Fatalf("shower lifetimes=(%d,%d,%d), want (32,25,8)", healBurstLife, healSpawnTicks, healParticleLife)
	}
	b := healBurst{seed: 12345, tile: 1}
	got := showerCohorts(b, 24)
	wantCounts := [25]int{3, 5, 4, 3, 4, 5, 4, 5, 5, 4, 3, 5, 4, 5, 3, 3, 3, 4, 4, 5, 4, 4, 4, 3, 5}
	total := 0
	for i, want := range wantCounts {
		if len(got[i]) != want {
			t.Fatalf("cohort %d count=%d, want %d", i, len(got[i]), want)
		}
		total += len(got[i])
	}
	if total != 101 {
		t.Fatalf("25 fixed cohorts contain %d particles, want 101", total)
	}
	wantFirst := []showerParticle{{dx: 1, depth: 7}, {dx: -9, depth: -6}, {dx: -15, depth: 2}}
	for i, want := range wantFirst {
		if got[0][i] != want {
			t.Errorf("first cohort particle %d=%+v, want %+v", i, got[0][i], want)
		}
	}
	wide := showerCohorts(healBurst{seed: 12345, tile: 3}, 0)
	wantWide := []showerParticle{{dx: 5, depth: 23}, {dx: -27, depth: -19}, {dx: -46, depth: 6}}
	for i, want := range wantWide {
		if wide[0][i] != want {
			t.Errorf("T3 first cohort particle %d=%+v, want %+v", i, wide[0][i], want)
		}
	}
}

func TestHealAndDrainShowersUseAllEightVisiblePhases(t *testing.T) {
	t.Parallel()
	mw := sbWorld(t)
	base := image.Pt(4*ui.ShotScale, 4*ui.ShotScale)
	mw.healBursts = []healBurst{{at: image.Pt(4, 4), picture: healingPicture, owner: 1, seed: 12345, tile: 1}}
	if got := mw.healSpriteDraws(); len(got) != 3 || got[0].Frame != 0 || got[0].Pos != base.Add(image.Pt(1, -7).Mul(boltUnitsPerPixel)) {
		t.Fatalf("fresh Heal=%+v (count %d), want three frame-zero records and first offset (1,-7)", got[0], len(got))
	}
	mw.healBursts[0].age = 7
	heal := mw.healSpriteDraws()
	if len(heal) != 33 {
		t.Fatalf("Heal age 7 drew %d particles, want the fixed eight cohorts' 33", len(heal))
	}
	seen := [8]bool{}
	for _, p := range heal {
		if p.Frame < 0 || p.Frame > 7 {
			t.Fatalf("Heal selected frame %d, want 0..7", p.Frame)
		}
		seen[p.Frame] = true
	}
	for phase, ok := range seen {
		if !ok {
			t.Errorf("Heal age 7 has no visible frame %d", phase)
		}
	}
	if heal[0].Pos != base.Add(image.Pt(1, -42).Mul(boltUnitsPerPixel)) {
		t.Errorf("oldest Heal particle=%v, want fixed ellipse plus seven 5px rises", heal[0].Pos)
	}

	mw.healBursts[0] = healBurst{at: image.Pt(4, 4), picture: 30, owner: 2, seed: 12345, tile: 1, drain: true, age: 7}
	drain := mw.healSpriteDraws()
	if len(drain) != 33 || drain[0].Frame != 7 || drain[0].Pos != base.Add(image.Pt(1, -4).Mul(boltUnitsPerPixel)) {
		t.Fatalf("Drain age 7 first=%+v (count %d), want frame 7 descending to offset (1,-4)", drain[0], len(drain))
	}
	mw.healBursts[0].age = 31
	if got := mw.healSpriteDraws(); len(got) != 5 || got[0].Frame != 7 {
		t.Fatalf("last visible tick drew %d particles, first frame %d; want final cohort's five at frame 7", len(got), got[0].Frame)
	}
	mw.advanceHealBursts()
	if got := mw.healSpriteDraws(); len(got) != 0 {
		t.Fatalf("expired shower still drew %d particles", len(got))
	}
}

func TestHealAndDrainCastEventsBothStartShowers(t *testing.T) {
	t.Parallel()
	mw := sbWorld(t)
	mw.observeCasts([]sim.CastEvent{
		{Caster: 1, Target: 9, Spell: 6, ToX: 4, ToY: 4, TargetOwner: 1},
		{Caster: 2, Target: 8, Spell: 11, ToX: 5, ToY: 5, TargetOwner: 2},
	})
	if len(mw.healBursts) != 2 {
		t.Fatalf("Heal+Drain started %d showers, want two", len(mw.healBursts))
	}
	if mw.healBursts[0].drain || mw.healBursts[0].picture != 20 {
		t.Errorf("Heal shower=%+v, want picture 20 rising", mw.healBursts[0])
	}
	if !mw.healBursts[1].drain || mw.healBursts[1].picture != 30 {
		t.Errorf("Drain shower=%+v, want picture 30 descending", mw.healBursts[1])
	}
}

func TestHealSheetIsNotForcedThroughTheTravellingDrawArm(t *testing.T) {
	t.Parallel()

	mw := sbWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 6,
		FromX: 1, FromY: 1, ToX: 2, ToY: 1, HealthRestored: 1}})
	if got := mw.world.SavedProjectiles().FreeIndex; got != 1 {
		t.Fatalf("Heal took %d counter ids, want one", got)
	}
	if got := mw.boltDraws(nil); len(got) != 0 {
		t.Fatalf("the ordinary picture-20 arm drew %d objects, want its published no-blit", len(got))
	}
}
