package sim

import (
	"reflect"
	"testing"
)

// secondCurse is the shipped second-game Stone Curse row: its Effects string
// parses to no kind (R2-ASSET-107, R2-ENGINE-453).
func secondCurse() SpellRule {
	return secondRow(SpellRule{ID: 18, School: 4, ManaCost: 40, MaxRange: 5, TargetsUnit: true, SpellDuration: 10})
}

// curseWorld holds a second-game caster (1, owner 2) and a target (2, owner
// 1) that can move, attack and cast.
func curseWorld(t *testing.T, seed uint64, earth int32) *World {
	t.Helper()
	caster := effectMage(1, 2, 2, 1<<18)
	caster.Owner = 2
	target := effectMage(2, 3, 2, 1<<18)
	target.Protection[3] = earth
	target.Speed, target.Reach, target.DamageBase, target.AlwaysHits = 4, 1, 5, true
	return secondWorld(t, seed, Bounds{Width: 16, Height: 16}, []SpellRule{secondCurse()}, caster, target)
}

// secondWorld is a World a second-game script runs, so its byte form restores
// the second game's arms.
func secondWorld(t *testing.T, seed uint64, b Bounds, spells []SpellRule, ents ...Entity) *World {
	t.Helper()
	s, err := newScript(ScriptROM2, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewSpelledWorld(seed, b, ModeCanonical, nil, ents, s, spells)
	if err != nil {
		t.Fatal(err)
	}
	if !w.SecondGame() {
		t.Fatal("not a second-game World")
	}
	return w
}

func TestSecondStoneCurseCeilingFollowsTheClaim(t *testing.T) {
	// D = ((P*240*m)/100) * (100 - Earth)/100, m 2 for a spell, 1 for an
	// item (R2-ENGINE-025), worked by hand.
	for _, tc := range []struct {
		power, earth int32
		item         bool
		want         uint16
	}{
		{50, 0, false, 240}, {50, 40, false, 144}, {120, 0, false, 576}, {120, 40, false, 345},
		{50, 0, true, 120}, {50, 40, true, 72}, {50, 150, false, 0}, {50, -20, false, 240},
	} {
		if got := secondCurseCeiling(tc.power, tc.earth, tc.item); got != tc.want {
			t.Errorf("ceiling P %d Earth %d item %v = %d, want %d", tc.power, tc.earth, tc.item, got, tc.want)
		}
	}
}

func TestSecondStoneCurseDrawsItsDurationFromTheWorldStream(t *testing.T) {
	for _, tc := range []struct {
		power, earth int32
		ceiling      int32
	}{{50, 0, 240}, {50, 40, 144}, {120, 0, 576}, {120, 40, 345}} {
		w := curseWorld(t, 0x18, tc.earth)
		probe := w.rng
		want := uint16(probe.uniform(tc.ceiling))
		if want == 0 {
			t.Fatalf("seed draws 0 at P %d Earth %d", tc.power, tc.earth)
		}
		if r := w.pointEffectRefusal(0, 1, w.spells[0], tc.power); r != "" {
			t.Fatalf("refused: %s", r)
		}
		if !w.ordinaryEffectPayload(0, 1, w.spells[0], tc.power) {
			t.Fatalf("P %d Earth %d did not attach", tc.power, tc.earth)
		}
		got := w.ActiveEffects()
		if len(got) != 1 || got[0].Spell != 18 || got[0].Kind != EffectNone || got[0].Magnitude != 0 || got[0].Mode != EffectDuration || got[0].Remaining != want {
			t.Fatalf("P %d Earth %d attached %+v, want one blank record of %d", tc.power, tc.earth, got, want)
		}
		if w.rng != probe {
			t.Fatalf("P %d Earth %d drew other than one 0..%d draw", tc.power, tc.earth, tc.ceiling)
		}
		t.Logf("P %d Earth %d: ceiling %d, drawn %d", tc.power, tc.earth, tc.ceiling, want)
		if !w.stoneCursed(1) || !w.HasEffectArm(2, 20) {
			t.Fatal("the blank record does not hold its target")
		}
	}
	// An item cast halves the ceiling.
	w := curseWorld(t, 0x18, 40)
	item := w.spells[0]
	item.itemCast = true
	probe := w.rng
	want := uint16(probe.uniform(72))
	if !w.ordinaryEffectPayload(0, 1, item, 50) || w.ActiveEffects()[0].Remaining != want {
		t.Fatalf("item cast attached %+v, want %d", w.ActiveEffects(), want)
	}
}

func TestSecondStoneCurseDrawOfZeroAttachesNothing(t *testing.T) {
	seed := uint64(1)
	for ; seed < 1<<20; seed++ {
		if probe := (rng{state: seed}); probe.uniform(240) == 0 {
			break
		}
	}
	w := curseWorld(t, 1, 0)
	w.rng.state = seed
	if w.ordinaryEffectPayload(0, 1, w.spells[0], 50) || len(w.attached) != 0 {
		t.Fatalf("seed %d drew 0 and still attached %+v", seed, w.ActiveEffects())
	}
}

func TestSecondStoneCurseHoldsMoveAttackAndCastUntilItExpires(t *testing.T) {
	w := curseWorld(t, 0x18, 0)
	before := w.entities[1]
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 50) {
		t.Fatal("did not attach")
	}
	if got := w.entities[1]; got.HP != before.HP || got.Protection != before.Protection || got.Speed != before.Speed || got.Mana != before.Mana {
		t.Fatalf("the blank record changed its target: %+v", got)
	}
	w.attached[0].Remaining = 3
	Step(w, []Command{MoveTo(2, CellPoint{X: 9, Y: 9}), Attack(2, 1), Cast(2, 1, 18)})
	e := w.entities[1]
	if e.HasTarget || e.HasAttackTarget || len(w.bookCasts) != 0 || e.Mana != before.Mana || e.X != 3 || e.Y != 2 {
		t.Fatalf("cursed actor acted: target=%v attack=%v casts=%d mana=%d at %d,%d", e.HasTarget, e.HasAttackTarget, len(w.bookCasts), e.Mana, e.X, e.Y)
	}
	if got := w.attached[0].Remaining; got != 2 {
		t.Fatalf("timer after one tick = %d, want 2", got)
	}
	Step(w, nil)
	Step(w, nil)
	if len(w.attached) != 0 || w.stoneCursed(1) {
		t.Fatalf("hold outlived its timer: %+v", w.ActiveEffects())
	}
	Step(w, []Command{MoveTo(2, CellPoint{X: 9, Y: 9})})
	if !w.entities[1].HasTarget {
		t.Fatal("released actor cannot move again")
	}
}

func TestSecondStoneCurseLeavesDamageTakenUnchanged(t *testing.T) {
	plain := curseWorld(t, 0x18, 0)
	plain.entities[0].DamageBase, plain.entities[0].DamageSpread, plain.entities[0].AlwaysHits, plain.entities[0].Reach = 10, 20, true, 1
	cursed := cloneWorld(t, plain)
	if !cursed.ordinaryEffectPayload(0, 1, cursed.spells[0], 50) {
		t.Fatal("did not attach")
	}
	plain.rng.state, cursed.rng.state = 0x2a, 0x2a
	plain.resolveBlow(0, 1, nil)
	cursed.resolveBlow(0, 1, nil)
	if plain.entities[1].HP != cursed.entities[1].HP {
		t.Fatalf("cursed target took %d, uncursed %d", 100-cursed.entities[1].HP, 100-plain.entities[1].HP)
	}
}

func TestSecondStoneCurseRepeatKeepsTheLargerDuration(t *testing.T) {
	w := curseWorld(t, 0x18, 0)
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 50) {
		t.Fatal("did not attach")
	}
	w.attached[0].Remaining = 1
	probe := w.rng
	draw := uint16(probe.uniform(240))
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 50) || len(w.attached) != 1 || w.attached[0].Remaining != max(draw, 1) {
		t.Fatalf("shorter record: %+v after a draw of %d", w.ActiveEffects(), draw)
	}
	w.attached[0].Remaining = 9000
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 50) || len(w.attached) != 1 || w.attached[0].Remaining != 9000 {
		t.Fatalf("longer record: %+v, want 9000 kept", w.ActiveEffects())
	}
}

func TestSecondStoneCurseSurvivesAColdLoad(t *testing.T) {
	w := curseWorld(t, 0x18, 0)
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 50) {
		t.Fatal("did not attach")
	}
	cold := cloneWorld(t, w)
	if !cold.stoneCursed(1) {
		t.Fatal("cold world lost the hold")
	}
	Step(w, nil)
	Step(cold, nil)
	if w.Hash() != cold.Hash() {
		t.Fatal("next tick differs after a cold load")
	}
}

func TestFirstStoneCurseStillRefusesARowWithNoKind(t *testing.T) {
	first := SpellRule{ID: 20, School: 4, TargetsUnit: true, SpellDuration: 10}
	w := spWorld(t, 3, []SpellRule{first}, spEnt(1, 0, 0), spEnt(2, 1, 0))
	if r := w.pointEffectRefusal(0, 1, first, 50); r == "" {
		t.Fatal("a first-game Stone Curse row with no effect kind was admitted")
	}
	if w.ordinaryEffectPayload(0, 1, first, 50) || len(w.attached) != 0 {
		t.Fatal("a first-game Stone Curse row with no effect kind attached")
	}
}

func cloneWorld(t *testing.T, w *World) *World {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	cold := &World{spellCreatures: w.spellCreatures}
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if cold.Hash() != w.Hash() {
		t.Fatal("cold load changed canonical state")
	}
	return cold
}

// summonCreatures is one template per Summon name, its TypeID its index + 1.
func summonCreatures() []SpellCreature {
	var out []SpellCreature
	for k := int32(1); k <= 3; k++ {
		for level := int32(0); level < 4; level++ {
			name := SummonCreatureName(level*25, k)
			out = append(out, SpellCreature{Name: name, Actor: Entity{TypeID: int32(len(out) + 1), HP: 20, MaxHP: 20, Reach: 1, TokenSize: 1, Owner: 7}})
		}
	}
	return out
}

func summonRule() SpellRule { return secondRow(SpellRule{ID: 25, School: 5, ManaCost: 100}) }

// summonWorld is a 32x32 second-game World: a caster (1, owner 1, group 5)
// at 16,16 and ents.
func summonWorld(t *testing.T, seed uint64, astral int32, ents ...Entity) *World {
	t.Helper()
	caster := effectMage(1, 16, 16, 1<<25)
	caster.Skill[summonSkill], caster.Group, caster.MaxMana, caster.Mana = astral, 5, 200, 200
	w := secondWorld(t, seed, Bounds{Width: 32, Height: 32}, []SpellRule{summonRule()}, append([]Entity{caster}, ents...)...)
	w.DeclareSpellCreatures(summonCreatures())
	return w
}

// summonWitness rolls the kind and walks the placement of R2-ENGINE-455 and
// R2-ENGINE-531 on a copy of the stream: kind 1..3, then y and x at -3..3
// around the caster until a cell other than the caster's own.
func summonWitness(r rng, x, y int32) (k, px, py int32) {
	k = 1 + r.uniform(2)
	for try := 0; try < 20; try++ {
		cy := y - 3 + r.uniform(6)
		cx := x - 3 + r.uniform(6)
		if cx != x || cy != y {
			return k, cx, cy
		}
	}
	return k, -1, -1
}

func TestSecondSummonCreatesTheRolledKindAndLevelBesideTheCaster(t *testing.T) {
	for _, tc := range []struct {
		seed   uint64
		astral int32
		level  string
	}{{0x25, 0, ""}, {0x26, 30, ".2"}, {0x27, 60, ".3"}, {0x28, 99, ".4"}, {0x29, 200, ".4"}} {
		w := summonWorld(t, tc.seed, tc.astral)
		k, px, py := summonWitness(w.rng, 16, 16)
		name := SummonCreatureNames[k-1] + tc.level
		var template SpellCreature
		for _, c := range summonCreatures() {
			if c.Name == name {
				template = c
			}
		}
		if r := w.pointEffectRefusal(0, 0, w.spells[0], 50); r != "" {
			t.Fatalf("refused: %s", r)
		}
		if !w.ordinaryEffectPayload(0, 0, w.spells[0], 50) {
			t.Fatalf("seed %#x: Summon did not land", tc.seed)
		}
		if len(w.entities) != 2 {
			t.Fatalf("seed %#x: %d actors, want 2", tc.seed, len(w.entities))
		}
		got, caster := w.entities[1], w.entities[0]
		if got.TypeID != template.Actor.TypeID || got.X != px || got.Y != py {
			t.Fatalf("seed %#x Astral %d: built type %d at %d,%d, want %s (type %d) at %d,%d", tc.seed, tc.astral, got.TypeID, got.X, got.Y, name, template.Actor.TypeID, px, py)
		}
		if got.Owner != caster.Owner || got.CommandGroup == 0 || got.Group == caster.Group || got.ID != 2 {
			t.Fatalf("seed %#x: owner %d group %d/%d, caster owner %d group %d", tc.seed, got.Owner, got.Group, got.CommandGroup, caster.Owner, caster.Group)
		}
		t.Logf("seed %#x Astral %d: %s at %d,%d, group %d", tc.seed, tc.astral, name, got.X, got.Y, got.Group)
		Step(w, nil)
		cold := cloneWorld(t, w)
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("seed %#x: next tick differs after a cold load", tc.seed)
		}
	}
}

func TestSecondSummonJoinsTheCastersPlayerInANewSavedGroup(t *testing.T) {
	w := summonWorld(t, 0x25, 0)
	g := SavedGroup{ID: 1, Owner: SavedGroupReference{Key: 0x12345, Archive: 7, Class: 1, Owner: SelfSlot}, Members: []SavedGroupMember{{Entity: 1, Bound: true}}}
	if err := w.ImportSavedGroups([]SavedGroup{g}, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{ID: 10, Slot: SelfSlot}, {ID: 20, Slot: SelfSlot}}, []SavedGroupContainer{{GroupID: 1, PlayerID: 20}}); err != nil {
		t.Fatal(err)
	}
	if !w.ordinaryEffectPayload(0, 0, w.spells[0], 50) {
		t.Fatal("Summon did not land")
	}
	spawned := w.savedGroupFor(2)
	if spawned == nil || spawned.ID == 1 || spawned.ContainerID != 20 || len(spawned.Members) != 1 {
		t.Fatalf("summoned group %+v, want a new group under Player 20", spawned)
	}
	cloneWorld(t, w)
}

func TestSecondSummonAbortsWhenNoCellIsFree(t *testing.T) {
	var ring []Entity
	id := EntityID(2)
	for y := int32(13); y <= 19; y++ {
		for x := int32(13); x <= 19; x++ {
			if x != 16 || y != 16 {
				ring = append(ring, spEnt(id, x, y))
				id++
			}
		}
	}
	w := summonWorld(t, 0x25, 0, ring...)
	probe := w.rng
	probe.uniform(2)
	for try := 0; try < 20; try++ {
		probe.uniform(6)
		probe.uniform(6)
	}
	entities := w.Entities()
	if r := w.pointEffectRefusal(0, 0, w.spells[0], 50); r != "" {
		t.Fatalf("a full square is a refusal before the cost: %s", r)
	}
	if w.ordinaryEffectPayload(0, 0, w.spells[0], 50) {
		t.Fatal("Summon landed with no free cell")
	}
	if !reflect.DeepEqual(w.Entities(), entities) {
		t.Fatal("aborted Summon changed the actors")
	}
	if w.rng != probe {
		t.Fatal("aborted Summon did not keep exactly its kind roll and 20 placement tries")
	}
	// Through a book cast the cost stays paid (R2-ENGINE-544).
	w = summonWorld(t, 0x25, 0, ring...)
	mana := w.entities[0].Mana
	spRunCast(w, Cast(1, 1, 25))
	if len(w.entities) != len(ring)+1 || w.entities[0].Mana >= mana {
		t.Fatalf("book cast: %d actors, mana %d of %d", len(w.entities), w.entities[0].Mana, mana)
	}
}

func TestSecondSummonRefusesWithNoCreatureDeclared(t *testing.T) {
	w := summonWorld(t, 0x25, 0)
	w.DeclareSpellCreatures(nil)
	if r := w.pointEffectRefusal(0, 0, w.spells[0], 50); r == "" {
		t.Fatal("Summon admitted with no creature to build")
	}
}

func TestIceMissileAndDiamondDustLandAsFireArrowExceptSchool(t *testing.T) {
	// R2-ENGINE-024, R2-ASSET-107: rows 5 and 16 equal row 1 except Sphere.
	row := func(id uint16, school uint8) SpellRule {
		return secondRow(SpellRule{ID: id, School: school, ManaCost: 3, TargetsUnit: true, Delivery: 2, MaxRange: 7, Damaging: true, DamageMin: 5, DamageMax: 10, Defensive: false})
	}
	hp := map[uint16]int32{}
	for _, r := range []SpellRule{row(1, 1), row(5, 2), row(16, 4)} {
		if !SpellRuleLands(r) || !castsAtActor(r) {
			t.Fatalf("row %d does not land", r.ID)
		}
		caster := effectMage(1, 2, 2, 1<<r.ID)
		w := secondWorld(t, 0x05, Bounds{Width: 16, Height: 16}, []SpellRule{r}, caster, spEnt(2, 5, 2))
		spRunCast(w, Cast(1, 2, SpellID(r.ID)))
		for n := 0; n < 64 && w.PendingSpellDeliveries() != 0; n++ {
			Step(w, nil)
		}
		hp[r.ID] = w.entities[1].HP
	}
	if hp[1] == 100 || hp[5] != hp[1] || hp[16] != hp[1] {
		t.Fatalf("victim health after Fire Arrow %d, Ice Missile %d, Diamond Dust %d", hp[1], hp[5], hp[16])
	}
}
