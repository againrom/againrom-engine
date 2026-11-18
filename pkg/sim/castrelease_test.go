package sim

import "testing"

// releaseWorld holds the player's mage at (1,8) and a standing hostile at
// (4,8), inside the arrow's reach. second selects the second game.
func releaseWorld(t *testing.T, second bool, rule SpellRule) *World {
	t.Helper()
	caster := spMage(1, 1, 8, 30, 100, 100, 1<<1)
	caster.Owner = SelfSlot
	enemy := spEnt(2, 4, 8)
	enemy.Owner, enemy.HP, enemy.MaxHP = 2, 500, 500
	w := hlWorld(t, 7, acEnemies(t), []SpellRule{rule}, caster, enemy)
	holdGroups(w)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

func holdGroups(w *World) {
	for i := range w.groups {
		w.groups[i].order = orderStandGround
	}
	for i := range w.savedGroups.Groups {
		w.savedGroups.Groups[i].AI[0x20] = orderStandGround
	}
}

// relocate puts actor id at (x, y) and keeps its post there.
func relocate(t *testing.T, w *World, id EntityID, x, y int32) {
	t.Helper()
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		t.Fatalf("world holds no entity %d", id)
	}
	w.entities[i].X, w.entities[i].Y = x, y
	w.entities[i].PostX, w.entities[i].PostY = x, y
	w.syncSavedPost(i)
}

// admitCharging admits cmd and requires a charging record for caster.
func admitCharging(t *testing.T, w *World, caster EntityID, cmd Command) {
	t.Helper()
	Step(w, []Command{cmd})
	k, ok := w.bookCastIndex(caster)
	if !ok || w.bookCasts[k].Phase != bookCharging {
		t.Fatalf("the cast is not winding up: %+v", w.bookCasts)
	}
}

// releaseEvents steps the wind-up out and returns the casts caster released.
func releaseEvents(w *World, caster EntityID) []CastEvent {
	var out []CastEvent
	for range 2 * castPeriod {
		for _, ev := range StepObserved(w, nil) {
			if ev.Caster == caster {
				out = append(out, ev)
			}
		}
	}
	return out
}

// TestACastDecidedInRangeLandsOnATargetThatFledOutOfRange: a book cast whose
// range test passed at the order lands on its target at the release frame,
// whatever the distance then, for a missile, an instant effect and an area at
// the target, in both games (MAGIC-403 for the first game; no second-game
// claim states a range test at release).
func TestACastDecidedInRangeLandsOnATargetThatFledOutOfRange(t *testing.T) {
	missile := hlArrow()
	missile.Delivery, missile.EffectSpeed = 2, 200
	instant := hlArrow()
	instant.Delivery = 1
	area := SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 7, Area: true, Distribution: 2, Radius: 1,
		Damaging: true, DamageMin: 10, DamageMax: 10, TargetsUnit: true}
	for _, second := range []bool{false, true} {
		for name, rule := range map[string]SpellRule{"missile": missile, "instant": instant, "area": area} {
			w := releaseWorld(t, second, rule)
			admitCharging(t, w, 1, Cast(1, 2, 1))
			relocate(t, w, 2, 15, 8)
			if r := w.BookSpellRefusal(1, 2, 1); r != refusalTargetOutOfRange {
				t.Fatalf("%s second=%v fixture: refusal %q, want %q", name, second, r, refusalTargetOutOfRange)
			}
			events := releaseEvents(w, 1)
			if len(events) != 1 || events[0].Target != 2 || (spAt(t, w, 2).HP >= 500 && name != "missile") || spAt(t, w, 1).Mana != 97 {
				t.Fatalf("%s second=%v: events %v, target %d, mana %d, want one release paid once",
					name, second, events, spAt(t, w, 2).HP, spAt(t, w, 1).Mana)
			}
		}
	}
}

// TestACastDecidedOutOfRangeStillApproachesFirst: a cast ordered at a target
// beyond the range is refused for range alone and walks the caster in before
// the wind-up, in both games.
func TestACastDecidedOutOfRangeStillApproachesFirst(t *testing.T) {
	for _, second := range []bool{false, true} {
		w := releaseWorld(t, second, hlArrow())
		relocate(t, w, 2, 15, 8)
		if r := w.BookSpellRefusal(1, 2, 1); r != refusalTargetOutOfRange {
			t.Fatalf("second=%v fixture: refusal %q, want %q", second, r, refusalTargetOutOfRange)
		}
		Step(w, []Command{Cast(1, 2, 1)})
		x0 := spAt(t, w, 1).X
		events := releaseEvents(w, 1)
		if spAt(t, w, 1).X <= x0 || len(events) != 1 {
			t.Fatalf("second=%v: x %d to %d, events %v, want an approach then one release",
				second, x0, spAt(t, w, 1).X, events)
		}
	}
}

func TestACellCastReleasesWithoutAReachTest(t *testing.T) {
	ball := SpellRule{ID: 2, ManaCost: 5, School: 1, MaxRange: 5, Area: true, Distribution: 2, Radius: 1,
		Damaging: true, DamageMin: 10, DamageMax: 10}
	for _, second := range []bool{false, true} {
		w := releaseWorld(t, second, ball)
		w.entities[0].KnownSpells = 1 << 2
		admitCharging(t, w, 1, CastAt(1, 2, CellPoint{X: 4, Y: 8}))
		// The caster cannot walk during a wind-up; moving it is the only way
		// to put the cell out of reach before the release.
		relocate(t, w, 1, 1, 15)
		if r := w.BookSpellCellRefusal(1, 4, 8, 2); r != refusalCellOutOfRange {
			t.Fatalf("second=%v fixture: refusal %q, want %q", second, r, refusalCellOutOfRange)
		}
		if events := releaseEvents(w, 1); len(events) == 0 {
			t.Fatalf("second=%v: the cell cast released no event", second)
		}
	}
}

// TestACastReleasesAtATargetThatLeftSight: neither game's release asks the
// caster's sight (MAGIC-403; R2-ENGINE-1136, R2-ENGINE-1138). The second
// game's admission asks none either, so only the first game refuses anew.
func TestACastReleasesAtATargetThatLeftSight(t *testing.T) {
	for _, second := range []bool{false, true} {
		w := releaseWorld(t, second, hlArrow())
		w.entities[0].ScanRange = 1
		relocate(t, w, 2, 2, 8)
		admitCharging(t, w, 1, Cast(1, 2, 1))
		relocate(t, w, 2, 4, 8)
		want := refusalTargetNotVisible
		if second {
			want = ""
		}
		if r := w.BookSpellRefusal(1, 2, 1); r != want {
			t.Fatalf("second=%v fixture: refusal %q, want %q", second, r, want)
		}
		if events := releaseEvents(w, 1); len(events) == 0 {
			t.Fatalf("second=%v: the cast at a hidden target released no event", second)
		}
		if spAt(t, w, 2).HP >= 500 {
			t.Fatalf("second=%v: the release at the hidden target did not hit", second)
		}
	}
}

func TestAFirstGameCastPaidAtAdmissionIsNotChargedAgain(t *testing.T) {
	paid := hlArrow()
	paid.Delivery, paid.EffectSpeed = 2, 128
	for _, second := range []bool{false, true} {
		w := releaseWorld(t, second, paid)
		admitCharging(t, w, 1, Cast(1, 2, 1))
		if got := spAt(t, w, 1).Mana; got != 97 {
			t.Fatalf("second=%v: mana %d after admission, want 97", second, got)
		}
		w.entities[0].Mana = 1
		events := releaseEvents(w, 1)
		if len(events) != 1 || spAt(t, w, 1).Mana != 1 {
			t.Fatalf("second=%v: events %v, mana %d, want one release with no second charge and no refund",
				second, events, spAt(t, w, 1).Mana)
		}
	}
}

func TestAFirstGameCreatureCastReleasesWithoutArmingAnApproach(t *testing.T) {
	for _, second := range []bool{false, true} {
		w := engageMageWorld(t, 1<<1, 100, hlArrow())
		holdGroups(w)
		if second {
			w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
		}
		if !w.orderAttack(0, 2) {
			t.Fatalf("second=%v: the engage refused", second)
		}
		k, ok := w.bookCastIndex(1)
		if !ok || w.bookCasts[k].Phase != bookCharging {
			t.Fatalf("second=%v: the mage's cast is not winding up: %+v", second, w.bookCasts)
		}
		relocate(t, w, 2, 15, 8)
		x, releasedAt := spAt(t, w, 1).X, int32(-1)
		var events []CastEvent
		approached := false
		for range 2 * castPeriod {
			for _, ev := range StepObserved(w, nil) {
				if ev.Caster == 1 {
					events = append(events, ev)
					releasedAt = spAt(t, w, 1).X
				}
			}
			if k, ok := w.bookCastIndex(1); ok && w.bookCasts[k].Phase == bookApproach && len(events) == 0 {
				approached = true
				break
			}
		}
		if len(events) != 1 || events[0].Target != 2 || approached || spAt(t, w, 2).HP >= 100 {
			t.Fatalf("second=%v: events %v, approach %v, target %d, want one release and no approach",
				second, events, approached, spAt(t, w, 2).HP)
		}
		if releasedAt != x {
			t.Fatalf("first game: the mage moved from %d to %d before its release", x, releasedAt)
		}
	}
}
