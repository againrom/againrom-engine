package sim

import (
	"bytes"
	"testing"
)

// acsBuff is a synthetic point buff row with the given id and a reach of 5.
func acsBuff(id uint16) SpellRule {
	return SpellRule{ID: id, ManaCost: 3, School: 4, MaxRange: 5, TargetsUnit: true,
		Defensive: true, SpellDuration: 30, EffectKind: EffectProtectionFire,
		EffectMode: EffectDuration, EffectMagnitude: 5}
}

// acsSpells is a table holding every row the three levels list.
func acsSpells() []SpellRule {
	shield := acbShield()
	return []SpellRule{acsBuff(5), hlHeal(), acsBuff(10), acsBuff(15), acsBuff(16), shield, acsBuff(22), acsBuff(23), acsBuff(24), hlArrow()}
}

func acsMask(ids ...uint16) uint32 {
	var m uint32
	for _, id := range ids {
		m |= 1 << id
	}
	return m
}

// acsWorld is the human player's mage at slot 0 knowing ids, plus targets.
func acsWorld(t *testing.T, rel Relations, setting AutoCastSetting, known uint32, armed uint16, targets ...Entity) *World {
	t.Helper()
	caster := spMage(1, 2, 2, 30, 60, 60, known)
	caster.Owner, caster.AutoSpell = SelfSlot, armed
	w := hlWorld(t, 23, rel, acsSpells(), append([]Entity{caster}, targets...)...)
	if setting != 0 && !w.SetAutoCast(SelfSlot, setting) {
		t.Fatalf("setting %#x refused", setting)
	}
	return w
}

// acsFirst runs one automatic attempt through the production route and
// reports the cast it began.
func acsFirst(w *World) (uint16, EntityID, bool) {
	return acsAttempt(w, w.autoCast)
}

// acsLevel runs the level policy alone, after the armed row and the idle
// heal have had their turn.
func acsLevel(w *World) (uint16, EntityID, bool) {
	return acsAttempt(w, w.supportCast)
}

func acsAttempt(w *World, attempt func(int)) (uint16, EntityID, bool) {
	before := len(w.bookCasts)
	attempt(0)
	if len(w.bookCasts) == before {
		return 0, 0, false
	}
	b := w.bookCasts[len(w.bookCasts)-1]
	return b.Spell, b.Target, true
}

func acsUnit(id EntityID, owner uint32, x int32, hp int32) Entity {
	e := spEnt(id, x, 2)
	e.Owner, e.HP = owner, hp
	return e
}

// TestAutoCastLevelsCastTheirLists is the level policy on the human player's
// mage: which row each level casts first, on whom. The idle heal, which runs
// before the policy on the production route, is not part of it.
func TestAutoCastLevelsCastTheirLists(t *testing.T) {
	all := acsMask(5, 6, 10, 15, 16, 18, 22, 23, 24)
	for _, tc := range []struct {
		name    string
		setting AutoCastSetting
		known   uint32
		hurt    bool
		spell   uint16
		target  EntityID
		cast    bool
	}{
		{"minimum heals the wounded companion", AutoCastMinimum | AutoCastOwn, acsMask(6, 23), true, 6, 2, true},
		{"minimum shields itself first", AutoCastMinimum | AutoCastOwn, all, true, 18, 1, true},
		{"minimum leaves Bless alone", AutoCastMinimum | AutoCastOwn, acsMask(23, 24, 5), false, 0, 0, false},
		{"average blesses itself first", AutoCastAverage | AutoCastOwn, acsMask(23, 24, 5), false, 23, 1, true},
		{"average leaves the protections alone", AutoCastAverage | AutoCastOwn, acsMask(5, 15), false, 0, 0, false},
		{"maximum protects from air first", AutoCastMaximum | AutoCastOwn, acsMask(5, 16, 22), false, 16, 1, true},
		{"maximum hides the companion, not itself, first", AutoCastMaximum | AutoCastOwn, acsMask(15), false, 15, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hp := int32(100)
			if tc.hurt {
				hp = 40
			}
			w := acsWorld(t, Relations{}, tc.setting, tc.known, 0, acsUnit(2, SelfSlot, 4, hp))
			spell, target, cast := acsLevel(w)
			if cast != tc.cast || spell != tc.spell || target != tc.target {
				t.Fatalf("cast %v spell %d at %d, want %v spell %d at %d", cast, spell, target, tc.cast, tc.spell, tc.target)
			}
		})
	}
}

// TestAutoCastTargetBitsAdmitAlliesAndNeutrals is the target mask: a locked
// ally and a neutral are healed only when their bit is set.
func TestAutoCastTargetBitsAdmitAlliesAndNeutrals(t *testing.T) {
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationLocked}, [3]uint32{2, SelfSlot, relationLocked})
	for _, tc := range []struct {
		setting AutoCastSetting
		owner   uint32
		cast    bool
	}{
		{AutoCastMinimum | AutoCastOwn, 2, false},
		{AutoCastMinimum | AutoCastAllies, 2, true},
		{AutoCastMinimum | AutoCastOwn, 3, false},
		{AutoCastMinimum | AutoCastNeutral, 3, true},
	} {
		w := acsWorld(t, rel, tc.setting, acsMask(6), 0, acsUnit(2, tc.owner, 4, 40))
		if _, target, cast := acsLevel(w); cast != tc.cast || cast && target != 2 {
			t.Errorf("setting %#x owner %d: cast %v at %d, want %v", tc.setting, tc.owner, cast, target, tc.cast)
		}
	}
}

// TestAutoCastOffCastsNothingByLevel is the first game's default: a Player
// with no stored setting reads Off and its mage casts no unarmed Bless, while
// Average casts it. The loss control is a default of Average: the first case
// fails under it.
func TestAutoCastOffCastsNothingByLevel(t *testing.T) {
	w := acsWorld(t, Relations{}, 0, acsMask(23), 0)
	if s := w.AutoCast(SelfSlot); s != AutoCastOwn {
		t.Fatalf("an unstored first-game setting reads %#x, want Off", s)
	}
	if spell, _, cast := acsFirst(w); cast {
		t.Fatalf("Off cast %d", spell)
	}
	w = acsWorld(t, Relations{}, AutoCastAverage|AutoCastOwn, acsMask(23), 0)
	if spell, _, cast := acsFirst(w); !cast || spell != 23 {
		t.Fatalf("unarmed Bless under Average: cast %v spell %d", cast, spell)
	}
	// Storing Off clears the stored byte: the world reads Off again and its
	// byte form is the base form.
	if !w.SetAutoCast(SelfSlot, AutoCastOwn) || w.AutoCastStored(SelfSlot) || w.AutoCast(SelfSlot) != AutoCastOwn {
		t.Fatalf("Off left %#x stored %v", w.AutoCast(SelfSlot), w.AutoCastStored(SelfSlot))
	}
}

// TestAutoCastKeepsTheArmedAttackingRow is the first game's attacking
// autocast: an armed damaging row still fires at a hostile, ahead of support.
func TestAutoCastKeepsTheArmedAttackingRow(t *testing.T) {
	enemy := acsUnit(3, 2, 5, 100)
	w := acsWorld(t, acEnemies(t), AutoCastMaximum|AutoCastOwn, acsMask(1, 23), 1, enemy)
	if spell, target, cast := acsFirst(w); !cast || spell != 1 || target != 3 {
		t.Fatalf("armed Fire Arrow: cast %v spell %d at %d, want 1 at 3", cast, spell, target)
	}
}

// TestAutoCastAbsentReadsTheGamesDefault is a state that stores no setting:
// a new game, an old save or an old profile. The first game reads Off
// whatever the old armed row and healing percentage hold (DIV-3006); the
// second reads the constructor's Average, Own (DIV-3007).
func TestAutoCastAbsentReadsTheGamesDefault(t *testing.T) {
	for _, tc := range []struct {
		name    string
		armed   uint16
		healing int32
	}{{"nothing", 0, -1}, {"reserve 100", 0, 0}, {"reserve 0", 0, 2}, {"armed protection", 5, 0}, {"armed heal", 6, 1}} {
		w := acsWorld(t, Relations{}, 0, acsMask(1, 5, 6), tc.armed)
		if tc.healing >= 0 && !w.SetAutoHealing(SelfSlot, tc.healing) {
			t.Fatal("healing mode refused")
		}
		if got := w.AutoCast(SelfSlot); got != AutoCastOwn || w.AutoCastStored(SelfSlot) {
			t.Errorf("first game, %s: %#x stored %v, want Off", tc.name, got, w.AutoCastStored(SelfSlot))
		}
		w.rom2 = &rom2ScriptState{}
		if got := w.AutoCast(SelfSlot); got != DefaultAutoCast {
			t.Errorf("second game, %s: %#x, want %#x", tc.name, got, DefaultAutoCast)
		}
	}
}

// TestAutoCastCommandAndFormKeepTheSetting is the command seam and the byte
// form: the selector stores the byte, the form carries it, and a world with
// no stored setting keeps its old bytes.
func TestAutoCastCommandAndFormKeepTheSetting(t *testing.T) {
	w := acsWorld(t, Relations{}, 0, acsMask(6), 0)
	plain, err := w.MarshalBinary()
	if err != nil || plain[0] == autoCastFormVersion {
		t.Fatalf("a world without a setting wrote form %d: %v", plain[0], err)
	}
	Step(w, []Command{SetPlayerParameter(SelfSlot, PlayerParameterAutoCast, int32(AutoCastMaximum|AutoCastAllies))})
	if got := w.AutoCast(SelfSlot); got != AutoCastMaximum|AutoCastAllies {
		t.Fatalf("command stored %#x", got)
	}
	Step(w, []Command{SetPlayerParameter(SelfSlot, PlayerParameterAutoCast, 0x40)})
	if got := w.AutoCast(SelfSlot); got != AutoCastMaximum|AutoCastAllies {
		t.Fatalf("an invalid byte replaced the setting: %#x", got)
	}
	raw, err := w.MarshalBinary()
	if err != nil || raw[0] != autoCastFormVersion {
		t.Fatalf("form %d: %v", raw[0], err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if cold.AutoCast(SelfSlot) != w.AutoCast(SelfSlot) || cold.Hash() != w.Hash() {
		t.Fatal("cold load lost the setting")
	}
	again, _ := cold.MarshalBinary()
	if !bytes.Equal(again, raw) {
		t.Fatal("re-save changed the bytes")
	}
	a, b := cold, *w
	Step(&a, nil)
	Step(&b, nil)
	if a.Hash() != b.Hash() {
		t.Fatal("next tick differs after cold load")
	}
	bad := bytes.Clone(raw)
	bad[len(bad)-6-1] = 0x40
	if err := cold.UnmarshalBinary(bad); err == nil {
		t.Fatal("an invalid stored byte was admitted")
	}
}

// TestAutoCastLevelNumbersAndArms numbers Off and the three levels as the
// option row does, and lists the lowest level per arm.
func TestAutoCastLevelNumbersAndArms(t *testing.T) {
	s := AutoCastAllies | AutoCastNeutral
	for n, want := range []AutoCastSetting{0x06, 0x0e, 0x1e, 0x3e} {
		if got := s.WithLevel(AutoCastLevel(n)); got != want || got.LevelNumber() != n {
			t.Fatalf("number %d gave %#x (number %d), want %#x", n, got, got.LevelNumber(), want)
		}
	}
	for arm, want := range map[uint16]AutoCastSetting{6: AutoCastMinimum, 18: AutoCastMinimum, 23: AutoCastAverage, 24: AutoCastAverage,
		16: AutoCastMaximum, 22: AutoCastMaximum, 5: AutoCastMaximum, 10: AutoCastMaximum, 15: AutoCastMaximum, 1: 0} {
		if got := AutoCastLevelOfArm(arm); got != want {
			t.Errorf("arm %d level %#x, want %#x", arm, got, want)
		}
	}
}

// TestAutoCastSupportOnTheAttackRouteFollowsTheCommandRange is the attack
// route's support call: an attacking mage whose attack range is below 2
// still heals its wounded companion; one attacking from range 2 or more, by
// its reach or by an armed attacking spell it can pay for, does not
// (R2-ENGINE-397, DIV-3009). The loss control is the melee case: a route that
// stopped support for every attacking mage fails it.
func TestAutoCastSupportOnTheAttackRouteFollowsTheCommandRange(t *testing.T) {
	for _, tc := range []struct {
		reach uint8
		armed uint16
		cast  bool
	}{{1, 0, true}, {4, 0, false}, {1, 1, false}} {
		enemy := acsUnit(3, 2, 3, 100)
		w := acsWorld(t, acEnemies(t), AutoCastMinimum|AutoCastOwn, acsMask(1, 6), tc.armed, acsUnit(2, SelfSlot, 4, 40), enemy)
		e := &w.entities[0]
		e.Reach, e.AttackTarget, e.HasAttackTarget = tc.reach, 3, true
		if tc.armed != 0 && w.attackRange(0) < 2 {
			t.Fatalf("armed Fire Arrow range %d", w.attackRange(0))
		}
		if w.supportWhileAttacking(0) != tc.cast {
			t.Errorf("reach %d armed %d: support gate %v", tc.reach, tc.armed, !tc.cast)
		}
		if tc.armed != 0 {
			continue
		}
		spell, target, cast := acsFirst(w)
		if cast != tc.cast || cast && (spell != 6 || target != 2) {
			t.Errorf("reach %d armed %d: cast %v spell %d at %d, want %v", tc.reach, tc.armed, cast, spell, target, tc.cast)
		}
	}
}
