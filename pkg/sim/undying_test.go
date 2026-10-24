package sim

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
)

func undyingWorld(t *testing.T) *World {
	t.Helper()
	return engWorld(t, engRel(t, [3]uint32{2, 3, 1}), withdrawalFighter(1, 2, 20, 20, 30), withdrawalFighter(2, 3, 30, 20, 100))
}

// A world with no undying unit encodes as it did before the state existed:
// the always-hits byte stays 0 or 1. Setting the state moves only bit 1 of
// that byte, and the byte form carries it back.
func TestUndyingIsBitOneOfTheAlwaysHitsByte(t *testing.T) {
	w := undyingWorld(t)
	plain, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	hash := w.Hash()
	w.entities[0].Undying = true
	marked, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) != len(marked) {
		t.Fatalf("lengths %d and %d", len(plain), len(marked))
	}
	var diffs []int
	for i := range plain {
		if plain[i] != marked[i] {
			diffs = append(diffs, i)
		}
	}
	if len(diffs) != 1 || plain[diffs[0]] != 1 || marked[diffs[0]] != 1|entityUndyingBit {
		t.Fatalf("undying moved bytes %v", diffs)
	}
	if w.Hash() == hash {
		t.Fatal("the undying state is not in the hash")
	}
	var back World
	if err := back.UnmarshalBinary(marked); err != nil {
		t.Fatal(err)
	}
	if !back.entities[0].Undying || !back.entities[0].AlwaysHits || back.entities[1].Undying {
		t.Fatalf("decoded undying %v always-hits %v, other %v", back.entities[0].Undying, back.entities[0].AlwaysHits, back.entities[1].Undying)
	}
	w.entities[0].Undying = false
	if again, _ := w.MarshalBinary(); !bytes.Equal(again, plain) || w.Hash() != hash {
		t.Fatal("clearing the state does not restore the byte form and hash")
	}
}

// The SAV actions leaf writes Undying only when it is set, so a save with no
// undying unit holds the same JSON as before, and an old leaf without the key
// restores the unit not undying.
func TestUndyingInTheActorValuesLeaf(t *testing.T) {
	w := undyingWorld(t)
	e := w.entities[0]
	raw, err := json.Marshal(e.Values())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("Undying")) {
		t.Fatalf("a plain actor writes the key: %s", raw)
	}
	var old ActorValues
	if err := json.Unmarshal(raw, &old); err != nil {
		t.Fatal(err)
	}
	target := e
	target.Undying = true
	if err := target.restoreValues(old); err != nil || target.Undying {
		t.Fatalf("old leaf restored undying %v (%v)", target.Undying, err)
	}
	e.Undying = true
	raw, err = json.Marshal(e.Values())
	if err != nil || !bytes.Contains(raw, []byte(`"Undying":true`)) {
		t.Fatalf("an undying actor's leaf: %s (%v)", raw, err)
	}
	var cur ActorValues
	if err := json.Unmarshal(raw, &cur); err != nil {
		t.Fatal(err)
	}
	target.Undying = false
	if err := target.restoreValues(cur); err != nil || !target.Undying {
		t.Fatalf("leaf restored undying %v (%v)", target.Undying, err)
	}
}

// An undying unit is alive, not dead and not downed at any health down to the
// signed 16-bit floor, and a blow past the floor stops at it.
func TestUndyingHealthStopsAtTheFieldFloor(t *testing.T) {
	w := undyingWorld(t)
	w.setUndying(0, true)
	Step(w, []Command{Damage(1, math.MaxInt32)})
	e := w.entities[indexOfEntity(w.entities, 1)]
	if e.HP != undyingHealthFloor || !e.Alive() || e.Dead() || e.Downed() || e.Decay != DecayNone {
		t.Fatalf("hp %d alive %v dead %v downed %v decay %d", e.HP, e.Alive(), e.Dead(), e.Downed(), e.Decay)
	}
	for _, hp := range []int32{0, -1, -10} {
		e.HP = hp
		if !e.Alive() || e.Dead() || e.Downed() || !e.Living() || !e.OrdinaryTargetable() {
			t.Fatalf("undying at %d is not a living unit", hp)
		}
	}
}

// An undying unit at negative health is still struck by an attack order and
// still healed, and stays alive through both.
func TestUndyingAtNegativeHealthIsAttackedAndHealed(t *testing.T) {
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}),
		withdrawalFighter(1, 2, 20, 20, 30), withdrawalFighter(2, 3, 21, 20, 100))
	i := indexOfEntity(w.entities, 1)
	w.setUndying(i, true)
	Step(w, []Command{Damage(1, 50)})
	Step(w, []Command{Attack(2, 1)})
	for range 64 {
		Step(w, nil)
	}
	e := w.entities[indexOfEntity(w.entities, 1)]
	if e.HP >= -20 || !e.Alive() || e.Decay != DecayNone {
		t.Fatalf("the attacked undying unit: hp %d alive %v decay %d", e.HP, e.Alive(), e.Decay)
	}
	before := e.HP
	w.applySpellHealing(indexOfEntity(w.entities, 1), SpellRule{DamageMin: 5, DamageMax: 5}, 0)
	if h := w.entities[indexOfEntity(w.entities, 1)]; h.HP <= before || !h.Alive() || h.Decay != DecayNone {
		t.Fatalf("the healed undying unit: hp %d -> %d alive %v decay %d", before, h.HP, h.Alive(), h.Decay)
	}
}

// health-below is strict: health equal to X is not below it.
func TestHealthBelowIsStrict(t *testing.T) {
	for _, c := range []struct {
		hp, x, want int32
	}{{-25, -25, 0}, {-26, -25, 1}, {0, 1, 1}, {1, 1, 0}, {math.MinInt16, math.MinInt16, 0}} {
		if got := healthBelow(Entity{HP: c.hp}, c.x); got != c.want {
			t.Fatalf("health %d below %d = %d, want %d", c.hp, c.x, got, c.want)
		}
	}
}

// Setting the state on a unit that has already taken the death transition
// changes nothing; clearing it on a living unit at negative health runs that
// transition in the same pass, to the stage the same blow gives a plain unit.
func TestSetUndyingOnTheDeadAndClearingAtNegativeHealth(t *testing.T) {
	w := undyingWorld(t)
	Step(w, []Command{Damage(1, 40)})
	i := indexOfEntity(w.entities, 1)
	if w.entities[i].Alive() {
		t.Fatal("the plain unit survived -10")
	}
	plain := w.entities[i].Decay
	w.setUndying(i, true)
	if w.entities[i].Undying {
		t.Fatal("a dead unit was made undying")
	}
	w = undyingWorld(t)
	i = indexOfEntity(w.entities, 1)
	w.setUndying(i, true)
	Step(w, []Command{Damage(1, 40)})
	if e := w.entities[i]; !e.Alive() || e.Decay != DecayNone {
		t.Fatalf("undying unit at %d: alive %v decay %d", e.HP, e.Alive(), e.Decay)
	}
	w.setUndying(i, false)
	if e := w.entities[i]; e.Alive() || e.Decay != plain {
		t.Fatalf("cleared at %d: alive %v decay %d, the plain unit's %d", e.HP, e.Alive(), e.Decay, plain)
	}
}

// undyingKillRun is a hero's kill of a credited creature through Step and the
// script: trigger 0 sets the creature undying when undying is true, trigger
// 1 clears it once its health is below -25. The creature is struck to -35
// after the first script pass. It returns the Diary and the hero's kill
// experience.
func undyingKillRun(t *testing.T, undying bool) (SavedDiary, bool, int32) {
	t.Helper()
	hero := xpAttacker(SelfSlot)
	body := diaryBody(2, 66, 1)
	body.XPValue = 1000
	first := acts()
	if undying {
		first = acts(0)
	}
	program := mustScript(t,
		[]ScriptCheck{constCheck(0, 1), {Op: ScriptCheckHealthBelow, Register: 1, Unit: 2, HasUnit: true, Args: [scriptParams]int32{-25}}},
		[]ScriptInstant{
			{Op: ScriptInstantUndying, Unit: 2, HasUnit: true, Args: [scriptParams]int32{1}},
			{Op: ScriptInstantUndying, Unit: 2, HasUnit: true, Args: [scriptParams]int32{0}},
		},
		[]ScriptTrigger{
			{Pairs: [3]ScriptPair{pair(0, 0, ScriptCmpEQ)}, Instants: first, Once: true},
			{Pairs: [3]ScriptPair{pair(1, 0, ScriptCmpEQ)}, Instants: acts(1), Once: true, Latch: 1},
		})
	w := scriptWorld(t, program, []Entity{hero, body})
	w.SetDiaryUnits(119, map[uint16]DiaryUnit{2: {Row: 70, Face: 2}})
	scriptTicks(w, 16, nil)
	if got := w.entities[indexOfEntity(w.entities, 2)].Undying; got != undying {
		t.Fatalf("after the first script pass the creature is undying %v, want %v", got, undying)
	}
	before := w.entities[indexOfEntity(w.entities, 1)].SkillXP[3]
	scriptTicks(w, 48, []Command{Damage(2, 135)})
	if i := indexOfEntity(w.entities, 2); i >= 0 && (w.entities[i].Alive() || w.entities[i].Undying) {
		t.Fatalf("the creature still stands: hp %d undying %v", w.entities[i].HP, w.entities[i].Undying)
	}
	d, ok := w.playerDiary()
	return d, ok, w.entities[indexOfEntity(w.entities, 1)].SkillXP[3] - before
}

// A creature killed by clearing its undying state below zero gives its
// killer the same Diary entry and kill experience as the same creature killed
// outright by the blow.
func TestClearedUndyingKillPaysAsAPlainKill(t *testing.T) {
	plainDiary, plainOK, plainXP := undyingKillRun(t, false)
	if !plainOK || len(plainDiary.Entries) != 1 || plainXP <= 0 {
		t.Fatalf("the plain kill: diary %+v (%v), experience %d", plainDiary, plainOK, plainXP)
	}
	diary, ok, xp := undyingKillRun(t, true)
	if !ok || len(diary.Entries) != 1 || diary.Entries[0] != plainDiary.Entries[0] {
		t.Fatalf("the cleared undying kill: diary %+v (%v), the plain kill's %+v", diary, ok, plainDiary)
	}
	if xp != plainXP {
		t.Fatalf("the cleared undying kill paid %d kill experience, the plain kill %d", xp, plainXP)
	}
}

// Prismatic Spray refuses a body as its primary, and an undying unit at
// health 0 or below is no body.
func TestPrismaticTakesAnUndyingUnitAsPrimary(t *testing.T) {
	rule := SpellRule{ID: prismaticSpellID}
	for _, c := range []struct {
		e    Entity
		body bool
	}{
		{Entity{HP: 1, MaxHP: 10}, false},
		{Entity{HP: 0, MaxHP: 10}, true},
		{Entity{HP: -5, MaxHP: 10}, true},
		{Entity{HP: 0, MaxHP: 10, Undying: true}, false},
		{Entity{HP: -5, MaxHP: 10, Undying: true}, false},
	} {
		if got := prismaticBodyPrimary(c.e, rule); got != c.body {
			t.Errorf("health %d undying %v: body primary %v, want %v", c.e.HP, c.e.Undying, got, c.body)
		}
	}
}

// The session wrapper's live decay leaves an undying unit's negative health
// alone and walks a plain unit's.
func TestLiveDecaySkipsAnUndyingUnit(t *testing.T) {
	w := undyingWorld(t)
	w.entities[0].HP, w.entities[0].Undying = -20, true
	w.entities[1].HP = -20
	w.decaySessionActors(SessionClock{SubTick: 12, FullTick: 4})
	if hp := w.entities[0].HP; hp != -20 {
		t.Errorf("the undying unit decayed to %d", hp)
	}
	if hp := w.entities[1].HP; hp != -21 {
		t.Errorf("the plain unit at -20 decayed to %d, want -21", hp)
	}
}

// AgainromNodeValue is the one domain check of both Againrom nodes.
func TestAgainromNodeValueDomains(t *testing.T) {
	for _, c := range []struct {
		check bool
		op    int32
		value int64
		ok    bool
	}{
		{true, ScriptCheckHealthBelow, math.MinInt16, true},
		{true, ScriptCheckHealthBelow, math.MaxInt16, true},
		{true, ScriptCheckHealthBelow, math.MinInt16 - 1, false},
		{true, ScriptCheckHealthBelow, math.MaxInt16 + 1, false},
		{false, ScriptInstantUndying, 0, true},
		{false, ScriptInstantUndying, 1, true},
		{false, ScriptInstantUndying, 2, false},
		{false, ScriptInstantUndying, -1, false},
		{true, ScriptCheckConstant, 1 << 40, true},
		{false, ScriptCheckHealthBelow, 1 << 40, true},
		{true, ScriptInstantUndying, 2, true},
	} {
		if err := AgainromNodeValue(c.check, c.op, c.value); (err == nil) != c.ok {
			t.Errorf("check %v op %#x value %d: %v, want ok %v", c.check, c.op, c.value, err, c.ok)
		}
	}
}
