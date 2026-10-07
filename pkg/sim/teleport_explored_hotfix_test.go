package sim

import "testing"

const (
	teleportTestWidth  = int32(48)
	teleportTestHeight = int32(8)
)

func teleportHiddenWorld(t *testing.T, blockedX int32) *World {
	t.Helper()
	caster := Entity{ID: 1, X: 2, Y: 3, HP: 100, MaxHP: 100, Owner: SelfSlot, TokenSize: 1,
		Mind: 100, Mana: 200, MaxMana: 200, KnownSpells: 1 << 26, ScanRange: 4,
		AttackCharge: 1, AttackRelax: 1}
	caster.Skill[5] = 30
	block := make([]byte, teleportTestWidth*teleportTestHeight)
	if blockedX >= 0 {
		block[3*teleportTestWidth+blockedX] = blockGround
	}
	w, err := NewStockedSpelledWorld(0x7e1e, Bounds{Width: teleportTestWidth, Height: teleportTestHeight},
		ModeCanonical, Terrain{Block: block}, []Entity{caster}, nil, Relations{}, nil, nil,
		[]SpellRule{{ID: 26, ManaCost: 60, School: 5, MaxRange: 8, Defensive: true}})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	return w
}

func teleportRangeWorld(t *testing.T, width int32, maxRange uint8, mind, schoolSkill int32) *World {
	t.Helper()
	caster := Entity{ID: 1, X: 2, Y: 3, HP: 100, MaxHP: 100, Owner: SelfSlot, TokenSize: 1,
		Mind: mind, Mana: 200, MaxMana: 200, KnownSpells: 1 << 26, ScanRange: 4,
		AttackCharge: 1, AttackRelax: 1}
	caster.Skill[5] = schoolSkill
	w, err := NewStockedSpelledWorld(0x7e21, Bounds{Width: width, Height: teleportTestHeight},
		ModeCanonical, Terrain{}, []Entity{caster}, nil, Relations{}, nil, nil,
		[]SpellRule{{ID: 26, ManaCost: 60, School: 5, MaxRange: maxRange, Defensive: true}})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	return w
}

func runCellCast(w *World, x, y int32, spell uint16) {
	Step(w, []Command{{Kind: KindCastAt, Entity: 1, X: x, Y: y, Spell: spell}})
	finishCellCast(w)
}

func finishCellCast(w *World) {
	for i := 0; i < 32; i++ {
		if len(w.bookCasts) == 0 {
			return
		}
		if w.bookCasts[0].Complete {
			if w.bookCasts[0].Phase == bookRelaxing {
				w.entities[0].CastWait = w.bookCasts[0].Remaining
			}
			w.bookCasts = nil
			return
		}
		Step(w, nil)
	}
}

func TestHiddenTeleportSurvivesAdmissionAndReleaseSightChecks(t *testing.T) {
	w := teleportHiddenWorld(t, -1)
	const farX, farY = int32(35), int32(3) // distance 33; decoded Teleport range is 41.
	if r := w.BookSpellCellRefusal(1, farX, farY, 26); r != "" {
		t.Fatalf("hidden Teleport admission refused with %q", r)
	}
	runCellCast(w, farX, farY, 26)
	got := spAt(t, w, 1)
	if got.X != farX || got.Y != farY {
		t.Fatalf("hidden Teleport ended at (%d,%d), want (%d,%d)", got.X, got.Y, farX, farY)
	}
	if got.Mana != 140 {
		t.Fatalf("hidden Teleport left %d mana, want 140 after one 60-mana cast", got.Mana)
	}
}

func TestPointSpellRangeIsIndependentOfPersonalSight(t *testing.T) {
	caster := Entity{ID: 1, X: 2, Y: 3, HP: 100, MaxHP: 100, Owner: SelfSlot, TokenSize: 1,
		Mind: 100, Mana: 200, MaxMana: 200, KnownSpells: 1 << 21, ScanRange: 4}
	w, err := NewSpelledWorld(0x7e1f, Bounds{Width: teleportTestWidth, Height: teleportTestHeight},
		ModeCanonical, nil, []Entity{caster}, nil,
		[]SpellRule{{ID: 21, ManaCost: 20, School: 4, MaxRange: 40, Area: true,
			Distribution: 5, Radius: 2, Damaging: true, DamageMin: 1, DamageMax: 1}})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	if r := w.BookSpellCellRefusal(1, 35, 3, 21); r != "" {
		t.Fatalf("in-range point spell refusal = %q", r)
	}
	runCellCast(w, 35, 3, 21)
	if got := spAt(t, w, 1).Mana; got != 180 {
		t.Fatalf("point spell mana: got %d, want 180", got)
	}
}

func TestTeleportUsesTheWholeConfiguredAndPowerScaledRange(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		width                 int32
		maxRange              uint8
		mind, skill           int32
		exactX, oneBeyondX    int32
		expectedComputedRange int32
	}{
		{name: "range one", width: 8, maxRange: 1, mind: 30, exactX: 3, oneBeyondX: 4,
			expectedComputedRange: 1},
		{name: "maximum row range plus maximum power", width: 300, maxRange: 255,
			mind: 100, skill: 30, exactX: 290, oneBeyondX: 291, expectedComputedRange: 288},
	} {
		t.Run(tc.name, func(t *testing.T) {
			atBoundary := teleportRangeWorld(t, tc.width, tc.maxRange, tc.mind, tc.skill)
			if got := tc.exactX - 2; got != tc.expectedComputedRange {
				t.Fatalf("fixture distance = %d, want fixed range boundary %d", got, tc.expectedComputedRange)
			}
			if r := atBoundary.BookSpellCellRefusal(1, tc.exactX, 3, 26); r != "" {
				t.Fatalf("exact range boundary refused with %q", r)
			}
			runCellCast(atBoundary, tc.exactX, 3, 26)
			got := spAt(t, atBoundary, 1)
			if got.X != tc.exactX || got.Y != 3 || got.Mana != 140 {
				t.Fatalf("exact range cast ended at (%d,%d), mana %d; want (%d,3), 140",
					got.X, got.Y, got.Mana, tc.exactX)
			}

			beyond := teleportRangeWorld(t, tc.width, tc.maxRange, tc.mind, tc.skill)
			if r := beyond.BookSpellCellRefusal(1, tc.oneBeyondX, 3, 26); r != "cell out of range" {
				t.Fatalf("one cell beyond range refusal = %q, want range refusal", r)
			}
			Step(beyond, []Command{CastAt(1, 26, CellPoint{X: tc.oneBeyondX, Y: 3})})
			for n := 0; n < 600 && len(beyond.bookCasts) != 0; n++ {
				Step(beyond, nil)
			}
			got = spAt(t, beyond, 1)
			if got.X != tc.oneBeyondX || got.Y != 3 || got.Mana != 140 {
				t.Fatalf("out-of-range cast ended at (%d,%d), mana %d; want a walk into range and one cast to (%d,3), 140",
					got.X, got.Y, got.Mana, tc.oneBeyondX)
			}
		})
	}
}

func TestTeleportReleaseDoesNotRequireSightThatWasLostDuringWindup(t *testing.T) {
	w := teleportHiddenWorld(t, -1)
	const targetX, targetY = int32(5), int32(3)
	if !w.actorSees(0, cell{x: targetX, y: targetY}) {
		t.Fatal("target must be visible when the cast is admitted")
	}
	Step(w, []Command{{Kind: KindCastAt, Entity: 1, X: targetX, Y: targetY, Spell: 26}})
	if len(w.bookCasts) != 1 {
		t.Fatalf("admission queued %d casts, want one wind-up", len(w.bookCasts))
	}
	w.entities[0].ScanRange = 0
	if w.actorSees(0, cell{x: targetX, y: targetY}) {
		t.Fatal("target remained visible after the fixture removed current sight")
	}
	finishCellCast(w)
	got := spAt(t, w, 1)
	if got.X != targetX || got.Y != targetY || got.Mana != 140 {
		t.Fatalf("release after sight loss ended at (%d,%d), mana %d; want (%d,%d), 140",
			got.X, got.Y, got.Mana, targetX, targetY)
	}
}

func TestTeleportReleaseRechecksAOnceOpenDestination(t *testing.T) {
	w := teleportHiddenWorld(t, -1)
	const targetX, targetY = int32(35), int32(3)
	Step(w, []Command{{Kind: KindCastAt, Entity: 1, X: targetX, Y: targetY, Spell: 26}})
	if len(w.bookCasts) != 1 {
		t.Fatalf("admission queued %d casts, want one wind-up", len(w.bookCasts))
	}
	i, inside := cellIndexIn(w.bounds, targetX, targetY)
	if !inside {
		t.Fatal("target fixture unexpectedly outside the map")
	}
	w.grid[i] |= blockGround
	for i := 0; i < 32; i++ {
		Step(w, nil)
	}
	got := spAt(t, w, 1)
	if got.X != 2 || got.Y != 3 || got.Mana != 140 || got.CastWait != 0 {
		t.Fatalf("blocked-at-release cast left caster at (%d,%d), mana %d, recovery %d",
			got.X, got.Y, got.Mana, got.CastWait)
	}
}

func TestOutOfRangeTeleportWalksTowardTheCellAndCastsInRange(t *testing.T) {
	w := teleportRangeWorld(t, 120, 8, 30, 0)
	const farX = int32(60) // range 8 plus power/3; the caster stands at x=2
	Step(w, []Command{CastAt(1, 26, CellPoint{X: farX, Y: 3})})
	start := spAt(t, w, 1)
	if start.Mana != 200 || len(w.bookCasts) != 1 {
		t.Fatalf("an out-of-range order paid %d mana or queued %d casts before walking", 200-start.Mana, len(w.bookCasts))
	}
	for n := 0; n < 20; n++ {
		Step(w, nil)
	}
	if got := spAt(t, w, 1); got.X <= start.X || got.Mana != 200 {
		t.Fatalf("after 20 ticks the caster stands at x=%d with %d mana, want a walk toward the cell and no payment yet", got.X, got.Mana)
	}
	for n := 0; n < 3000 && len(w.bookCasts) != 0; n++ {
		Step(w, nil)
	}
	if got := spAt(t, w, 1); got.X != farX || got.Y != 3 || got.Mana != 140 {
		t.Fatalf("the order ended at (%d,%d) with %d mana, want (%d,3) and one cast", got.X, got.Y, got.Mana, farX)
	}
}

func TestOutOfRangeTeleportOrderIsReplacedByAMove(t *testing.T) {
	w := teleportRangeWorld(t, 120, 8, 30, 0)
	Step(w, []Command{CastAt(1, 26, CellPoint{X: 60, Y: 3})})
	Step(w, []Command{MoveTo(1, CellPoint{X: 2, Y: 9})})
	for n := 0; n < 400; n++ {
		Step(w, nil)
	}
	if got := spAt(t, w, 1); got.X > 6 || got.Mana != 200 || len(w.bookCasts) != 0 {
		t.Fatalf("a later move left the caster at (%d,%d), mana %d, %d casts armed", got.X, got.Y, got.Mana, len(w.bookCasts))
	}
}
