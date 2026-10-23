package sim

import "testing"

// castSightWorld holds a SelfSlot mage at 2,2 with no SeeInvisible radius and
// one invisible target of owner at x,2; rel is the relation the world takes.
func castSightWorld(t *testing.T, owner uint32, x int32, rel Relations) *World {
	t.Helper()
	caster := effectMage(1, 2, 2, 1<<1)
	target := spEnt(2, x, 2)
	target.Owner = owner
	rule := SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 5, TargetsUnit: true,
		DamageMin: 10, DamageMax: 10, Damaging: true}
	w := hlWorld(t, 101, rel, []SpellRule{rule}, caster, target)
	if !w.attachEffect(2, 2, SpellRule{ID: 15}, EffectInvisible, 1, 30, EffectDuration) {
		t.Fatal("invisibility did not attach")
	}
	return w
}

func castSightRelation(t *testing.T, v byte) Relations {
	t.Helper()
	var r Relations
	r.Set(SelfSlot, 2, v)
	r.Set(2, SelfSlot, v)
	return r
}

// A book cast sees an invisible target of the caster's own player or of an
// allied player (DIV-3996); an enemy or a neutral stays hidden outside the
// caster's SeeInvisible radius. Attack perception does not change.
func TestABookCastSeesAnInvisibleOwnOrAlliedTargetOnly(t *testing.T) {
	for _, tc := range []struct {
		name  string
		owner uint32
		rel   byte
		seen  bool
	}{
		{name: "own", owner: SelfSlot, seen: true},
		{name: "allied", owner: 2, rel: relationLocked, seen: true},
		{name: "enemy", owner: 2, rel: relationHostile},
		{name: "neutral", owner: 2},
		{name: "allied turned hostile", owner: 2, rel: relationLocked | relationHostile},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := castSightWorld(t, tc.owner, 4, castSightRelation(t, tc.rel))
			refusal := w.BookSpellRefusal(1, 2, 1)
			if got := refusal != refusalTargetNotVisible; got != tc.seen || w.castSeesEntity(0, 1) != tc.seen {
				t.Fatalf("cast sight %v (refusal %q), want %v", w.castSeesEntity(0, 1), refusal, tc.seen)
			}
			if w.actorSeesEntity(0, 1) || !w.invisibleToActor(0, 1) {
				t.Fatal("attack perception saw the invisible target")
			}
			if tc.seen && (refusal != "" || !w.beginBookSpell(0, 2, 1)) {
				t.Fatalf("the seen target was not cast at: %q", refusal)
			}
		})
	}
}

// An enemy inside the caster's SeeInvisible radius is seen, as before.
func TestABookCastSeesAnInvisibleEnemyInsideItsSeeInvisibleRadius(t *testing.T) {
	w := castSightWorld(t, 2, 4, castSightRelation(t, relationHostile))
	w.entities[0].SeeInvisible = 2
	if !w.castSeesEntity(0, 1) || w.BookSpellRefusal(1, 2, 1) != "" {
		t.Fatalf("an enemy on the radius was not seen: %q", w.BookSpellRefusal(1, 2, 1))
	}
	w.entities[0].SeeInvisible = 1
	if w.castSeesEntity(0, 1) || w.BookSpellRefusal(1, 2, 1) != refusalTargetNotVisible {
		t.Fatal("an enemy outside the radius was seen")
	}
}

// A player's book cast at an invisible own unit in reach starts where the
// caster stands; out of reach it walks, as at a visible unit.
func TestAPlayersCastAtAnInvisibleOwnUnitWalksOnlyForReach(t *testing.T) {
	w := castSightWorld(t, SelfSlot, 4, Relations{})
	if w.manualCastApproach(spCast(1, 2, 1)) {
		t.Fatal("a cast in reach at an invisible own unit walks")
	}
	far := castSightWorld(t, SelfSlot, 12, Relations{})
	c := spCast(1, 2, 1)
	if r := far.manualCastRefusal(c); !isRangeRefusal(r) || !far.manualCastApproach(c) {
		t.Fatalf("a cast out of reach at an invisible own unit: refusal %q, approach %v", r, far.manualCastApproach(c))
	}
}
