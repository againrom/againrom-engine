package game

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type spellSoundPlay struct {
	slot  int
	owner uint32
	cell  image.Point
}

func recordSpellSounds(mw *mapWorld) *[]spellSoundPlay {
	plays := new([]spellSoundPlay)
	mw.spellSound = func(slot int, owner uint32, cell image.Point) {
		*plays = append(*plays, spellSoundPlay{slot: slot, owner: owner, cell: cell})
	}
	return plays
}

func TestSpellSoundSelectorsAreTheDecodedPictureFormulas(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		spell  uint16
		cast   int
		effect int
	}{
		{0, 0, 0},
		{1, 510, 511},
		{2, 512, 513},
		{4, 516, 517},
		// Picture 51 is the fixed phase-8 storm arm, not the ordinary
		// construction tail, although both arithmetic paths name slot 551.
		{21, 550, 551},
		// The terminal effect candidate is deliberately retained as formula
		// slot 565; the shipped bank refuses it because SfxCount ends at 564.
		{28, 564, 565},
		{29, 0, 0},
	} {
		if got := castSpellSoundSlot(tc.spell); got != tc.cast {
			t.Errorf("cast selector for spell %d = %d, want %d", tc.spell, got, tc.cast)
		}
		if got := effectSpellSoundSlot(tc.spell); got != tc.effect {
			t.Errorf("effect selector for spell %d = %d, want %d", tc.spell, got, tc.effect)
		}
	}
}

func spellSoundWorld(t *testing.T, caster sim.Entity, rules ...sim.SpellRule) *mapWorld {
	t.Helper()
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical,
		nil, []sim.Entity{caster}, nil, rules)
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	v, err := ui.NewViewer("spell-sound", terrain.Grid{
		Width: 32, Height: 32, Tiles: make([]uint16, 32*32),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	mw := newMapWorld(w, nil, nil, v)
	mw.sounds = map[int32]UnitSound{1: {AttackDelay: 2}}
	mw.projectiles = sbSheets()
	return mw
}

func spellSoundFireBallWorld(t *testing.T, mana int32) *mapWorld {
	t.Helper()
	caster := sim.Entity{ID: 1, X: 8, Y: 8, Class: 1, HP: 100, MaxHP: 100,
		Owner: 7, Mind: 40, Mana: mana, MaxMana: 100, KnownSpells: 1 << 2,
		AttackCharge: 12, AttackRelax: 4, ScanRange: 30}
	rule := sim.SpellRule{ID: 2, ManaCost: 3, School: 1, MaxRange: 20,
		Area: true, Distribution: 3, Radius: 1, DamageMin: 4, DamageMax: 8,
		Damaging: true}
	return spellSoundWorld(t, caster, rule)
}

func spellSoundTeleportWorld(t *testing.T) *mapWorld {
	t.Helper()
	caster := sim.Entity{ID: 1, X: 8, Y: 8, Class: 1, HP: 100, MaxHP: 100,
		Owner: 7, Mind: 40, Mana: 100, MaxMana: 100, KnownSpells: 1 << 26,
		AttackCharge: 12, AttackRelax: 4, ScanRange: 30}
	return spellSoundWorld(t, caster,
		sim.SpellRule{ID: 26, ManaCost: 3, School: 5, MaxRange: 20})
}

// driveOneFireBall enters through the production tick seam. The sole command
// is authored on tick zero; tickWithCastSink only counts the immutable cast
// observations that the same tick hands to the production sound observers.
func driveOneFireBall(mw *mapWorld, ticks int) (casts int) {
	mw.sched = [][]sim.Command{{
		{Kind: sim.KindCastAt, Entity: 1, X: 12, Y: 8, Spell: 2},
	}}
	for range ticks {
		mw.tickWithCastSink(func(events []sim.CastEvent) { casts += len(events) })
	}
	return casts
}

func TestOneFireBallCommandProducesOneSoundLifecycle(t *testing.T) {
	mw := spellSoundFireBallWorld(t, 50)
	plays := recordSpellSounds(mw)

	if casts := driveOneFireBall(mw, 96); casts != 1 {
		t.Fatalf("one Fire Ball command applied %d times, want once", casts)
	}
	want := []spellSoundPlay{
		// Direct even-picture message at the caster when the cast applies.
		{slot: 512, owner: 7, cell: image.Pt(8, 8)},
		// The four-cell projectile reaches its visual impact after two driver
		// ticks; AttackDelay phase two is the cast run's third tick.
		{slot: 513, owner: 7, cell: image.Pt(12, 8)},
		{slot: 512, owner: 7, cell: image.Pt(8, 8)},
	}
	if len(*plays) != len(want) {
		t.Fatalf("one Fire Ball produced %d sound(s), want %d: %+v", len(*plays), len(want), *plays)
	}
	for i := range want {
		if (*plays)[i] != want[i] {
			t.Errorf("sound %d = %+v, want %+v", i, (*plays)[i], want[i])
		}
	}
}

func TestTeleportAnimationSoundFollowsTheLiveCasterToItsDestination(t *testing.T) {
	mw := spellSoundTeleportWorld(t)
	plays := recordSpellSounds(mw)
	mw.sched = [][]sim.Command{{
		{Kind: sim.KindCastAt, Entity: 1, X: 12, Y: 9, Spell: 26},
	}}
	casts := 0
	for tick := 0; tick < 96 && len(*plays) < 2; tick++ {
		mw.tickWithCastSink(func(events []sim.CastEvent) { casts += len(events) })
	}
	if casts != 1 {
		t.Fatalf("one Teleport command applied %d times, want once", casts)
	}
	if e, ok := mw.entity(1); !ok || e.X != 12 || e.Y != 9 {
		t.Fatalf("Teleport caster = %+v, present=%v; want live at (12,9)", e, ok)
	}
	want := []spellSoundPlay{
		// The direct message remains at CastEvent.FromX/Y.
		{slot: 560, owner: 7, cell: image.Pt(8, 8)},
		// The independent AttackDelay hook follows the visible caster run.
		{slot: 560, owner: 7, cell: image.Pt(12, 9)},
	}
	if len(*plays) != len(want) || (*plays)[0] != want[0] || (*plays)[1] != want[1] {
		t.Fatalf("Teleport sound lifecycle = %+v, want %+v", *plays, want)
	}
}

func TestBookAnimationSoundDropsWhenTheCasterDiesBeforeAttackDelay(t *testing.T) {
	mw := spellSoundFireBallWorld(t, 50)
	mw.sounds[1] = UnitSound{AttackDelay: 5}
	plays := recordSpellSounds(mw)
	mw.sched = [][]sim.Command{{
		{Kind: sim.KindCastAt, Entity: 1, X: 12, Y: 8, Spell: 2},
	}}

	released := false
	for tick := 0; tick < 64 && !released; tick++ {
		mw.tickWithCastSink(func(events []sim.CastEvent) { released = len(events) != 0 })
	}
	if !released || len(*plays) != 1 || (*plays)[0].slot != 512 {
		t.Fatalf("release precondition: released=%v plays=%+v, want direct cast sound only", released, *plays)
	}
	// The next production step kills the caster while the class AttackDelay
	// cue is pending. Fire Ball's separately correct impact cue may still land.
	mw.pending = append(mw.pending, sim.Command{Kind: sim.KindTerminalKill, Entity: 1})
	for range 32 {
		mw.tick()
	}
	if e, ok := mw.entity(1); !ok || e.Alive() {
		t.Fatalf("kill precondition: caster = %+v, present=%v", e, ok)
	}
	castSounds, impactSounds := 0, 0
	for _, play := range *plays {
		switch play.slot {
		case 512:
			castSounds++
		case 513:
			impactSounds++
		}
	}
	if castSounds != 1 || impactSounds != 1 {
		t.Fatalf("post-death sounds = %+v; want one direct cast, one impact, no animation", *plays)
	}
}

func TestReplacementBookRunCancelsOnlyTheOldAttackDelayCue(t *testing.T) {
	mw := spellSoundFireBallWorld(t, 50)
	plays := recordSpellSounds(mw)
	event := sim.CastEvent{Caster: 1, Spell: 1, Owner: 7,
		FromX: 8, FromY: 8, ToX: 10, ToY: 8}

	// observeCasts is the production presentation seam. A second applied event
	// for the same caster replaces startCastRun's first live run before either
	// reaches AttackDelay; its direct event-source sound remains independent.
	mw.observeCasts([]sim.CastEvent{event})
	first := mw.castRun[1].identity
	mw.observeCasts([]sim.CastEvent{event})
	second := mw.castRun[1].identity
	if first == nil || second == nil || first == second {
		t.Fatalf("replacement run identities = %p then %p, want distinct non-nil identities", first, second)
	}
	for range 8 {
		mw.tick()
	}
	if len(*plays) != 3 {
		t.Fatalf("two direct events plus newest run produced %+v, want exactly three sounds", *plays)
	}
	for i, play := range *plays {
		if play != (spellSoundPlay{slot: 510, owner: 7, cell: image.Pt(8, 8)}) {
			t.Errorf("replacement sound %d = %+v", i, play)
		}
	}
}

func TestARefusedCastAndAudioDisabledWorldStaySilent(t *testing.T) {
	t.Run("refused", func(t *testing.T) {
		mw := spellSoundFireBallWorld(t, 0)
		plays := recordSpellSounds(mw)
		if casts := driveOneFireBall(mw, 24); casts != 0 {
			t.Fatalf("mana-refused Fire Ball applied %d times, want zero", casts)
		}
		if len(*plays) != 0 {
			t.Fatalf("mana-refused Fire Ball played %+v", *plays)
		}
	})

	t.Run("audio disabled", func(t *testing.T) {
		mw := spellSoundFireBallWorld(t, 50)
		if casts := driveOneFireBall(mw, 24); casts != 1 {
			t.Fatalf("audio-disabled cast count = %d, want one simulation cast", casts)
		}
		if len(mw.spellSoundCues) != 0 {
			t.Fatalf("audio-disabled world retained %d stale cue(s), want none", len(mw.spellSoundCues))
		}
	})
}

func TestResolvedTemporaryCasterUsesItsCarriedCellsAndNoAnimationHook(t *testing.T) {
	mw := spellSoundFireBallWorld(t, 50)
	plays := recordSpellSounds(mw)
	mw.observeScriptCasts([]sim.ScriptCastEvent{{
		Spell: 2, FromX: 4, FromY: 6, ToX: 8, ToY: 6,
	}})
	if len(*plays) != 1 || (*plays)[0] != (spellSoundPlay{slot: 512, cell: image.Pt(4, 6)}) {
		t.Fatalf("temporary-caster direct sound = %+v", *plays)
	}
	for range 2 {
		mw.advanceSpellSoundCues()
	}
	want := []spellSoundPlay{
		{slot: 512, cell: image.Pt(4, 6)},
		{slot: 513, cell: image.Pt(8, 6)},
	}
	if len(*plays) != len(want) || (*plays)[0] != want[0] || (*plays)[1] != want[1] {
		t.Fatalf("temporary-caster lifecycle = %+v, want %+v", *plays, want)
	}
	for range 8 {
		mw.advanceSpellSoundCues()
	}
	if len(*plays) != 2 {
		t.Fatalf("temporary caster gained an animation-hook repeat: %+v", *plays)
	}
}

func TestStagedAndStormEffectsVoiceEachAcceptedCellOnce(t *testing.T) {
	mw := aoWorld(t)
	plays := recordSpellSounds(mw)

	mw.observeAreaPaints([]sim.AreaPaint{
		{Spell: 4, Owner: 3, Cells: []sim.CellPoint{{X: 3, Y: 4}, {X: 5, Y: 4}}},
		{Spell: 21, Owner: 9, Cells: []sim.CellPoint{{X: 8, Y: 7}}},
	})
	if want := []spellSoundPlay{
		{slot: 517, owner: 3, cell: image.Pt(3, 4)},
		{slot: 517, owner: 3, cell: image.Pt(5, 4)},
	}; len(*plays) != len(want) || (*plays)[0] != want[0] || (*plays)[1] != want[1] {
		t.Fatalf("construction sounds = %+v, want %+v", *plays, want)
	}

	// Re-reading the same presentation state is not a world tick and cannot
	// consume or duplicate the delayed storm cue.
	_ = mw.boltDraws(mw.world.Entities())
	_ = mw.boltDraws(mw.world.Entities())
	if len(*plays) != 2 {
		t.Fatalf("two redraws changed sound count to %d, want 2", len(*plays))
	}
	for range stormSoundPhase - 1 {
		mw.advanceSpellSoundCues()
	}
	if len(*plays) != 2 {
		t.Fatalf("storm sounded before phase 8: %+v", *plays)
	}
	mw.advanceSpellSoundCues()
	if got := (*plays)[len(*plays)-1]; got != (spellSoundPlay{slot: 551, owner: 9, cell: image.Pt(8, 7)}) {
		t.Fatalf("storm phase-8 sound = %+v", got)
	}
	for range 12 {
		mw.advanceSpellSoundCues()
	}
	if len(*plays) != 3 {
		t.Fatalf("completed storm cue repeated: %+v", *plays)
	}
}

func TestWeaponCastAnimationUsesAttackDelayOnceAtSpellRange(t *testing.T) {
	caster := swingSoundUnit(1, 2, 4)
	caster.Owner, caster.WeaponSpell = 5, 2
	caster.HasAttackTarget, caster.AttackTarget = true, 2
	caster.AttackPhase = sim.AttackCasting
	target := swingSoundUnit(2, 8, 4) // outside the physical weapon's reach
	mw := swingWorld(t, swingAnimDesc(), caster, target)

	var plays []spellSoundPlay
	mw.setSwingSound(map[int32]UnitSound{1: {Slots: []int32{77}, AttackDelay: 0}},
		func(slot int, owner uint32, cell image.Point) {
			plays = append(plays, spellSoundPlay{slot: slot, owner: owner, cell: cell})
		})
	mw.advanceSwings()
	mw.advanceSwings()
	if want := []spellSoundPlay{{slot: 512, owner: 5, cell: image.Pt(2, 4)}}; len(plays) != 1 || plays[0] != want[0] {
		t.Fatalf("weapon cast animation sounds = %+v, want %+v", plays, want)
	}
}
