package sim

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func healTargetsWorld(t *testing.T, hostile bool) *World {
	t.Helper()
	caster := spMage(1, 2, 2, 30, 50, 50, 1<<6)
	caster.Owner = 1
	caster.AttackCharge, caster.AttackRelax = 4, 2
	enemy := spEnt(2, 3, 3)
	enemy.Owner, enemy.HP = 2, 40
	neutral := spEnt(3, 4, 4)
	neutral.Owner, neutral.HP = 3, 40
	heal := hlHeal()
	heal.HealHostile = hostile
	return hlWorld(t, 5, acEnemies(t), []SpellRule{heal}, caster, enemy, neutral)
}

func TestHealHostileHealsAnEnemyAndChangesNothingElse(t *testing.T) {
	t.Parallel()

	for _, victim := range []EntityID{2, 3} {
		w := healTargetsWorld(t, true)
		before := w.Relations()
		events := spRunCast(w, spCast(1, victim, 6))

		if got := spAt(t, w, 1).Mana; got != 40 {
			t.Errorf("victim %d: caster holds %d mana, want 40", victim, got)
		}
		got := spAt(t, w, victim)
		if got.HP <= 40 {
			t.Errorf("victim %d holds %d health, want a heal above 40", victim, got.HP)
		}
		if got.HasKillCredit || got.KillCreditSource != 0 {
			t.Errorf("victim %d carries a kill credit toward the healer", victim)
		}
		if len(events) != 1 || events[0].HealthRestored == 0 {
			t.Errorf("victim %d: cast events %+v, want one Heal event restoring health", victim, events)
		}
		after := w.Relations()
		for _, pair := range [][2]uint32{{1, 2}, {2, 1}, {1, 3}, {3, 1}, {2, 3}, {3, 2}} {
			if before.Byte(pair[0], pair[1]) != after.Byte(pair[0], pair[1]) {
				t.Errorf("victim %d: relation %v changed from %d to %d", victim, pair,
					before.Byte(pair[0], pair[1]), after.Byte(pair[0], pair[1]))
			}
		}
		if w.attackNoticeAt(indexOfEntity(w.entities, victim)).Cell != 0 {
			t.Errorf("victim %d was told of an attacker", victim)
		}
	}
}

func TestHealHostileOffKeepsTheRefusedHostileHeal(t *testing.T) {
	t.Parallel()

	w := healTargetsWorld(t, false)
	spRunCast(w, spCast(1, 2, 6))
	if got := spAt(t, w, 2).HP; got != 40 {
		t.Errorf("the enemy holds %d health, want its own untouched 40", got)
	}
	if got := spAt(t, w, 1).Mana; got != 40 {
		t.Errorf("the caster holds %d mana, want the cast paid", got)
	}
}

func TestHealHostileReachesScrollAndWeaponReleases(t *testing.T) {
	t.Parallel()

	w := healTargetsWorld(t, true)
	ci, ti := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)
	w.entities[ci].WeaponSpellLevel = 60
	obs := &castObs{}
	heal := hlHeal()
	heal.HealHostile = true
	w.releaseWeaponSpell(ci, ti, heal, obs)
	if got := spAt(t, w, 2).HP; got <= 40 {
		t.Errorf("a weapon heal left the enemy at %d health", got)
	}

	w = healTargetsWorld(t, true)
	item := ItemInstance{Code: 0xe10, Kind: 4, Price: 50, Effects: []ItemEffect{{Kind: 41, Operand: 6 | 60<<16}}}
	w.carried[0] = []ItemStack{StackItem(item, 1)}
	StepObserved(w, []Command{UseScroll(1, 0, 2)})
	for n := 0; n < 60 && len(w.scrollCasts) > 0; n++ {
		StepObserved(w, nil)
	}
	if got := spAt(t, w, 2).HP; got <= 40 {
		t.Errorf("a scroll heal left the enemy at %d health", got)
	}
}

func unbiddenHealWorld(t *testing.T, hostile bool, ally bool) *World {
	t.Helper()
	caster := acCaster(1, 2, 2, 50, 1<<6, 6)
	caster.Owner = 1
	enemy := spEnt(2, 3, 3)
	enemy.Owner, enemy.HP = 2, 20
	ents := []Entity{caster, enemy}
	if ally {
		friend := spEnt(3, 4, 4)
		friend.Owner, friend.HP = 1, 60
		ents = append(ents, friend)
	}
	heal := hlHeal()
	heal.HealHostile = hostile
	return hlWorld(t, 9, acEnemies(t), []SpellRule{heal}, ents...)
}

func TestHealHostileLetsAnUnbiddenHealChooseAnEnemyLast(t *testing.T) {
	t.Parallel()

	w := unbiddenHealWorld(t, false, false)
	if events := spRunUnbidden(w); len(events) != 0 || spAt(t, w, 2).HP != 20 {
		t.Errorf("without the flag: events %+v, enemy health %d, want no heal", events, spAt(t, w, 2).HP)
	}

	w = unbiddenHealWorld(t, true, false)
	spRunUnbidden(w)
	if got := spAt(t, w, 2).HP; got <= 20 {
		t.Errorf("with the flag the lone hurt enemy holds %d health, want a heal", got)
	}

	w = unbiddenHealWorld(t, true, true)
	spRunUnbidden(w)
	if got := spAt(t, w, 3).HP; got <= 60 {
		t.Errorf("with the flag the own ally holds %d health, want it chosen first", got)
	}
	if got := spAt(t, w, 2).HP; got != 20 {
		t.Errorf("with the flag the enemy holds %d health, want it passed over for the ally", got)
	}
}

func selfCastWorld(t *testing.T, allowed bool) *World {
	t.Helper()
	caster := spMage(1, 2, 2, 30, 50, 50, 1<<1)
	caster.Owner = 1
	arrow := hlArrow()
	arrow.SelfCast = allowed
	relations := engRel(t, [3]uint32{1, 2, relationHostile}, [3]uint32{2, 1, relationHostile}, [3]uint32{1, 1, relationLocked})
	return hlWorld(t, 5, relations, []SpellRule{arrow}, caster, spEnt(2, 3, 3))
}

func TestSelfCastLetsADamagingSpellNameItsCaster(t *testing.T) {
	t.Parallel()

	w := selfCastWorld(t, false)
	if got := w.BookSpellRefusal(1, 1, 1); got != "damaging self target" {
		t.Errorf("without the flag the refusal is %q", got)
	}
	Step(w, []Command{spCast(1, 1, 1)})
	if spAt(t, w, 1).Mana != 50 || spAt(t, w, 1).HP != 100 {
		t.Error("a refused self cast changed the caster")
	}

	w = selfCastWorld(t, true)
	if got := w.BookSpellRefusal(1, 1, 1); got != "" {
		t.Fatalf("with the flag the refusal is %q", got)
	}
	relations := w.Relations()
	spRunCast(w, spCast(1, 1, 1))
	for n := 0; n < 40; n++ {
		Step(w, nil)
	}
	caster := spAt(t, w, 1)
	if caster.Mana != 47 {
		t.Errorf("the caster holds %d mana, want 47", caster.Mana)
	}
	if caster.HP >= 100 {
		t.Errorf("the caster holds %d health, want damage from its own cast", caster.HP)
	}
	for _, pair := range [][2]uint32{{1, 1}, {1, 2}, {2, 1}} {
		if relations.Byte(pair[0], pair[1]) != w.Relations().Byte(pair[0], pair[1]) {
			t.Errorf("relation %v moved", pair)
		}
	}
}

func areaHitsWorld(t *testing.T, rule SpellRule) (*World, [4]EntityID) {
	t.Helper()
	caster := effectMage(1, 6, 6, 1<<fireBallSpell)
	caster.Owner = 1
	own := spEnt(2, 7, 6)
	own.Owner = 1
	foe := spEnt(3, 5, 5)
	foe.Owner = 2
	bystander := spEnt(4, 6, 7)
	bystander.Owner = 3
	w, err := NewStockedSpelledWorld(7, Bounds{Width: 16, Height: 16}, ModeCanonical,
		Terrain{}, []Entity{caster, own, foe, bystander}, nil, acEnemies(t), nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	w.SetBurstPhases(11)
	return w, [4]EntityID{1, 2, 3, 4}
}

func TestAreaHitsChoosesWhichSidesABlastReaches(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		hits AreaHits
		want [4]bool
	}{
		{AreaHitsAll, [4]bool{true, true, true, true}},
		{AreaHitsNotOwn, [4]bool{false, false, true, true}},
		{AreaHitsHostile, [4]bool{false, false, true, false}},
	} {
		rule := blastRule(1)
		rule.AreaHits = tc.hits
		w, ids := areaHitsWorld(t, rule)
		w.landAreaFacing(rule, 0, 1, true, 6, 6, 6, 6, 0, false, nil)
		for i, id := range ids {
			if hit := spAt(t, w, id).HP < 100; hit != tc.want[i] {
				t.Errorf("area_hits %d: unit %d hit=%v, want %v", tc.hits, id, hit, tc.want[i])
			}
		}
	}
}

func TestAreaHitsHoldsForABlastRestoredInFlight(t *testing.T) {
	t.Parallel()

	rule := blastRule(1)
	rule.AreaHits = AreaHitsHostile
	w, ids := areaHitsWorld(t, rule)
	if !w.landArea(rule, 0, 1, true, 1, 1, 6, 6, nil) {
		t.Fatal("area preparation")
	}
	for range 4 {
		Step(w, nil)
	}
	back := requireSpellGraphBinary(t, w)
	if !slices.Equal(back.Spells(), w.Spells()) {
		t.Fatal("the spell table lost a filter across the byte form")
	}
	for range 60 {
		Step(w, nil)
		Step(back, nil)
	}
	for _, world := range []*World{w, back} {
		for i, id := range ids {
			want := id == 3
			if hit := spAt(t, world, id).HP < 100; hit != want {
				t.Errorf("unit %d (index %d) hit=%v, want %v", id, i, hit, want)
			}
		}
	}
	if w.Hash() != back.Hash() {
		t.Error("a restored blast diverged")
	}
}

func TestTargetFiltersRoundTripTheByteFormAndRefuseRowsWithoutAnArm(t *testing.T) {
	t.Parallel()

	heal, arrow, ball := hlHeal(), hlArrow(), blastRule(2)
	heal.HealHostile, arrow.SelfCast, ball.SelfCast, ball.AreaHits = true, true, true, AreaHitsHostile
	w := hlWorld(t, 3, Relations{}, []SpellRule{arrow, heal, ball}, spEnt(1, 1, 1))
	back := requireSpellGraphBinary(t, w)
	if !slices.Equal(back.Spells(), w.Spells()) {
		t.Fatalf("filters lost: %+v want %+v", back.Spells(), w.Spells())
	}
	plain := hlWorld(t, 3, Relations{}, []SpellRule{hlArrow(), hlHeal(), blastRule(2)}, spEnt(1, 1, 1))
	if slices.Equal(hlBytes(t, plain), hlBytes(t, w)) {
		t.Error("a world with filters encodes as the world without")
	}

	for name, rule := range map[string]SpellRule{
		"heal flag on a damaging row": {ID: 1, Damaging: true, DamageMax: 4, HealHostile: true},
		"self flag on a healing row":  {ID: 6, Restorative: true, DamageMax: 4, SelfCast: true},
		"area filter on a point row":  {ID: 1, Damaging: true, DamageMax: 4, AreaHits: AreaHitsNotOwn},
		"area filter out of range":    {ID: 2, Area: true, AreaHits: 3},
	} {
		if _, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical,
			Terrain{}, []Entity{spEnt(1, 1, 1)}, nil, Relations{}, nil, nil, []SpellRule{rule}); err == nil {
			t.Errorf("%s: the table was accepted", name)
		}
	}
}

func TestQueuedDeliveriesKeepTheirHistoricalJSONWhenNoFilterIsSet(t *testing.T) {
	t.Parallel()

	plain, _ := json.Marshal(SpellRule{ID: 1})
	for _, name := range []string{"HealHostile", "SelfCast", "AreaHits"} {
		if strings.Contains(string(plain), name) {
			t.Errorf("an unfiltered rule writes %s: %s", name, plain)
		}
	}
	set, _ := json.Marshal(SpellRule{ID: 2, SelfCast: true, AreaHits: AreaHitsHostile})
	for _, name := range []string{`"SelfCast":true`, `"AreaHits":2`} {
		if !strings.Contains(string(set), name) {
			t.Errorf("a filtered rule lacks %s: %s", name, set)
		}
	}
	var back SpellRule
	if err := json.Unmarshal(set, &back); err != nil || back.AreaHits != AreaHitsHostile || !back.SelfCast {
		t.Errorf("round trip %+v %v", back, err)
	}
}
