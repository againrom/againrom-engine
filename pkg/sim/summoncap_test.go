package sim

import (
	"slices"
	"testing"
)

// summonedWorld is summonWorld with n creatures the caster's Summon built.
func summonedWorld(t *testing.T, seed uint64, n int) *World {
	t.Helper()
	w := summonWorld(t, seed, 0)
	for range n {
		if !w.ordinaryEffectPayload(0, 0, w.spells[0], 50) {
			t.Fatal("Summon did not land")
		}
	}
	if len(w.entities) != n+1 {
		t.Fatalf("%d actors, want the caster and %d creatures", len(w.entities), n)
	}
	return w
}

// livingCreatures counts the living actors other than the caster.
func livingCreatures(w *World) int {
	n := 0
	for i := range w.entities {
		if w.entities[i].ID != 1 && w.entities[i].Alive() {
			n++
		}
	}
	return n
}

// keyedWorld is a 32x32 World of one player: a caster for each key and, after
// them, counts[k] actors keyed to caster k, ids ascending in key order. A
// first-game World is built when second is false.
func keyedWorld(t *testing.T, seed uint64, second bool, keyed bool, counts ...int) *World {
	t.Helper()
	var ents []Entity
	for k := range counts {
		ents = append(ents, effectMage(EntityID(k+1), int32(2+k), 2, 0))
	}
	keys := map[EntityID]EntityID{}
	id := EntityID(len(counts) + 1)
	for k, n := range counts {
		for range n {
			e := spEnt(id, 2+int32(id)%28, 4+int32(id)/28)
			e.Owner, e.TokenSize, e.DyingTime = SelfSlot, 1, 0
			ents = append(ents, e)
			keys[id] = EntityID(k + 1)
			id++
		}
	}
	var w *World
	if second {
		w = secondWorld(t, seed, Bounds{Width: 32, Height: 32}, nil, ents...)
	} else {
		var err error
		if w, err = NewWorld(seed, Bounds{Width: 32, Height: 32}, ModeCanonical, nil, ents); err != nil {
			t.Fatal(err)
		}
	}
	if keyed {
		for i := range w.entities {
			w.entities[i].summonKey = keys[w.entities[i].ID]
		}
	}
	return w
}

// keyedIDs lists the ids that carry key, ascending.
func keyedIDs(w *World, key EntityID) []EntityID {
	var out []EntityID
	for _, e := range w.entities {
		if e.summonKey == key {
			out = append(out, e.ID)
		}
	}
	return out
}

// capWitness is R2-ENGINE-776's draw on a copy of the stream: for each key in
// ascending order with count n, r = 1 + Range(99), marked when r < n.
func capWitness(r rng, counts []int) ([]bool, rng) {
	marks := make([]bool, len(counts))
	for j, n := range counts {
		marks[j] = int(1+r.uniform(99)) < n
	}
	return marks, r
}

// A caster's four summoned creatures leave one at a time, each on a tick whose
// session counter at entry is 12 mod 16 (R2-ENGINE-778), until one is left,
// which the pass never takes: its chance is (1-1)/100 (R2-ENGINE-776).
func TestSecondSummonedCreaturesLeaveOnTheWorldPassUntilOneIsLeft(t *testing.T) {
	for _, seed := range []uint64{0x25, 0x31, 0x47} {
		w := summonedWorld(t, seed, 4)
		var left []uint64
		for step := 0; step < 16*1500; step++ {
			entry, before := w.tick, livingCreatures(w)
			Step(w, nil)
			after := livingCreatures(w)
			if after == before {
				continue
			}
			if before-after != 1 || entry%16 != 12 {
				t.Fatalf("seed %#x: %d creatures left on the step entered at tick %d", seed, before-after, entry)
			}
			left = append(left, entry)
		}
		if len(left) != 3 || livingCreatures(w) != 1 {
			t.Fatalf("seed %#x: creatures left at ticks %v, %d remain; want three to leave and one to stay", seed, left, livingCreatures(w))
		}
		t.Logf("seed %#x: creatures left at ticks %v", seed, left)
	}
}

// The creature a Summon builds carries the caster's key (R2-ENGINE-455), the
// id plus one; the caster carries none.
func TestSecondSummonGivesItsCreatureTheCastersKey(t *testing.T) {
	w := summonedWorld(t, 0x25, 2)
	for _, e := range w.entities {
		if want := map[bool]EntityID{true: 0, false: summonKeyOf(1)}[e.ID == 1]; e.summonKey != want {
			t.Fatalf("actor %d key %d, want %d", e.ID, e.summonKey, want)
		}
	}
}

// One pass on the original stream: one draw r = 1 + Range(99) per key in
// ascending key order, a key marked when r < n, and the lowest id of each
// marked key given the death at health -10000 with its key cleared
// (R2-ENGINE-776). A key of one actor still draws and never loses it; a key
// of 101 loses exactly one actor each pass.
func TestSecondWorldPassDrawsOncePerKeyAndKillsTheFirstActorOfAMarkedKey(t *testing.T) {
	w := keyedWorld(t, 1, true, true, 1, 2, 4, 101)
	w.rng = rng{state: 0x5eed, original: true}
	marked := 0
	for pass := 0; pass < 400; pass++ {
		var counts []int
		var firsts []EntityID
		for key := EntityID(1); key <= 4; key++ {
			ids := keyedIDs(w, key)
			if len(ids) > 0 {
				counts, firsts = append(counts, len(ids)), append(firsts, ids[0])
			}
		}
		marks, after := capWitness(w.rng, counts)
		living := map[EntityID]bool{}
		for _, e := range w.entities {
			living[e.ID] = e.Alive()
		}
		w.summonCapPass()
		if w.rng != after {
			t.Fatalf("pass %d: the stream is %+v, want %+v after one draw per key", pass, w.rng, after)
		}
		for _, e := range w.entities {
			j := slices.Index(firsts, e.ID)
			dies := j >= 0 && marks[j]
			if died := living[e.ID] && !e.Alive(); died != dies {
				t.Fatalf("pass %d: actor %d died %v, want %v (counts %v, marks %v)", pass, e.ID, died, dies, counts, marks)
			}
			if dies {
				marked++
				if e.HP != summonedDeathHP || e.summonKey != 0 || e.Decay != DecayFallen {
					t.Fatalf("pass %d: actor %d health %d key %d stage %d", pass, e.ID, e.HP, e.summonKey, e.Decay)
				}
			}
		}
		if n := counts[len(counts)-1]; n > 100 && !marks[len(marks)-1] {
			t.Fatalf("pass %d: the key of %d actors was not marked", pass, n)
		}
	}
	if ids := keyedIDs(w, 1); len(ids) != 1 {
		t.Fatalf("the caster with one actor has %v keyed", ids)
	}
	t.Logf("%d deaths in 400 passes; keys 2, 3 and 4 keep %d, %d and %d", marked, len(keyedIDs(w, 2)), len(keyedIDs(w, 3)), len(keyedIDs(w, 4)))
}

// The pass runs on the step whose session tick at entry is 12 mod 16 and on no
// other (R2-ENGINE-778): a key of 101 actors, which every pass marks, loses
// one actor on each such step and none between. The death's health is -10000
// and the body, whose dwell is 0, leaves the World on that step.
func TestSecondWorldPassRunsOnlyOnTheTwelfthTickOfSixteen(t *testing.T) {
	w := keyedWorld(t, 5, true, true, 101)
	for step := 0; step < 96; step++ {
		entry, before := w.tick, len(keyedIDs(w, 1))
		ids := keyedIDs(w, 1)
		Step(w, nil)
		gone := before - len(keyedIDs(w, 1))
		if want := map[bool]int{true: 1, false: 0}[entry%16 == 12]; gone != want {
			t.Fatalf("tick %d: %d keyed actors left, want %d", entry, gone, want)
		}
		if gone == 1 && slices.ContainsFunc(w.entities, func(e Entity) bool { return e.ID == ids[0] }) {
			t.Fatalf("tick %d: the first keyed actor %d is still in the World", entry, ids[0])
		}
	}
}

// The death is the engine's one death transition, so a keyed actor felled by
// any route stores -10000 and loses its key (R2-ENGINE-776: the death call's
// two callers). Its body falls, lies for its dwell and leaves the World at
// the corpse ladder's end (DIV-4388).
func TestSecondKeyedDeathStoresTheSummonedHealthAndTheBodyLeavesAfterItsDwell(t *testing.T) {
	w := keyedWorld(t, 7, true, true, 2)
	i := slices.IndexFunc(w.entities, func(e Entity) bool { return e.ID == 2 })
	w.entities[i].DyingTime = 5
	w.entities[i].setCurrentHealth(0)
	w.clearFelled(i)
	if e := w.entities[i]; e.HP != summonedDeathHP || e.summonKey != 0 || e.Decay != DecayFallen || e.Dwell != 5 {
		t.Fatalf("felled keyed actor: health %d key %d stage %d dwell %d", e.HP, e.summonKey, e.Decay, e.Dwell)
	}
	for step := 1; step <= 5; step++ {
		Step(w, nil)
		present := slices.ContainsFunc(w.entities, func(e Entity) bool { return e.ID == 2 })
		if present != (step < 5) {
			t.Fatalf("step %d: the body is present %v", step, present)
		}
	}
	if ids := keyedIDs(w, 1); !slices.Equal(ids, []EntityID{3}) {
		t.Fatalf("keyed actors %v, want the survivor alone", ids)
	}
}

// The first game has no world pass: a first-game World whose actors carry
// keys draws the same stream and loses no actor across six passes' worth of
// ticks, exactly as the same World without keys.
func TestFirstGameKeyedActorsMeetNoWorldPass(t *testing.T) {
	keyed, plain := keyedWorld(t, 9, false, true, 101), keyedWorld(t, 9, false, false, 101)
	if keyed.SecondGame() {
		t.Fatal("a second-game World")
	}
	for step := 0; step < 96; step++ {
		Step(keyed, nil)
		Step(plain, nil)
		if keyed.RandomState() != plain.RandomState() || len(keyedIDs(keyed, 1)) != 101 || len(keyed.entities) != len(plain.entities) {
			t.Fatalf("tick %d: stream %#x / %#x, %d keyed", keyed.tick, keyed.RandomState(), plain.RandomState(), len(keyedIDs(keyed, 1)))
		}
	}
}

// The byte form carries each key, so a cold copy of a World with keyed actors
// is equal and runs on equal; a World with no key writes no key section, so
// every keyless form is the one the base wrote.
func TestSummonKeysRoundTripTheByteFormAndAKeylessFormHasNoSection(t *testing.T) {
	w := summonedWorld(t, 0x31, 3)
	form, err := w.MarshalBinary()
	if err != nil || form[0] != summonKeyFormVersion {
		t.Fatalf("keyed form version %d: %v", form[0], err)
	}
	cold := cloneWorld(t, w)
	if !slices.Equal(keyedIDs(cold, 1), keyedIDs(w, 1)) {
		t.Fatalf("cold keys %v, want %v", keyedIDs(cold, 1), keyedIDs(w, 1))
	}
	for range 16 * 200 {
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("tick %d: the cold copy diverged", w.tick)
		}
	}
	plain := keyedWorld(t, 3, true, false, 4)
	if b, err := plain.MarshalBinary(); err != nil || b[0] == summonKeyFormVersion {
		t.Fatalf("keyless form version %d: %v", b[0], err)
	}
}

// A form whose key section names an index past the actors, a zero key or an
// index out of order is refused.
func TestSummonKeySectionRefusesABadPair(t *testing.T) {
	w := keyedWorld(t, 3, true, true, 2)
	good, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	// The section holds two pairs (index, key) ahead of its 5-byte trailer
	// and 4-byte tag; the first pair's index is the third actor.
	at := len(good) - 9 - 16
	for _, bad := range []func(b []byte){
		func(b []byte) { b[at] = 99 },
		func(b []byte) { b[at+4] = 0 },
		func(b []byte) { b[at], b[at+8] = b[at+8], b[at] },
	} {
		b := slices.Clone(good)
		bad(b)
		if err := (&World{}).UnmarshalBinary(b); err == nil {
			t.Fatal("a bad key pair decoded")
		}
	}
}

// An actor every SAV LOAD builds, through NewActor from its record, carries no
// key (R2-SESSION-326: the Unit programme stores no +0x148; R2-ENGINE-535:
// init stores 0), so it is outside the cap: four creatures loaded beside
// their caster are never taken by the pass, and the pass draws once per key
// only for the keys still carried.
// A caster with entity id 0, as the first map-placed unit is, gives its
// creatures a key like any other caster: they carry a nonzero key and the
// pass takes them down to one, as it does for caster 1.
func TestSecondSummonCasterWithEntityIDZeroIsCapped(t *testing.T) {
	living := map[EntityID]int{}
	for _, casterID := range []EntityID{0, 1} {
		caster := effectMage(casterID, 16, 16, 1<<25)
		caster.Skill[summonSkill], caster.Group, caster.MaxMana, caster.Mana = 50, 5, 200, 200
		w := secondWorld(t, 0x25, Bounds{Width: 32, Height: 32}, []SpellRule{summonRule()}, caster)
		w.DeclareSpellCreatures(summonCreatures())
		for range 4 {
			if !w.ordinaryEffectPayload(0, 0, w.spells[0], 50) {
				t.Fatal("Summon did not land")
			}
		}
		for _, e := range w.entities {
			if want := map[bool]EntityID{true: 0, false: summonKeyOf(casterID)}[e.ID == casterID]; e.summonKey != want {
				t.Fatalf("caster %d: actor %d key %d, want %d", casterID, e.ID, e.summonKey, want)
			}
		}
		for range 16 * 1500 {
			Step(w, nil)
		}
		for _, e := range w.entities {
			if e.ID != casterID && e.Alive() {
				living[casterID]++
			}
		}
	}
	if living[0] != 1 || living[1] != 1 {
		t.Fatalf("living creatures after the passes: caster 0 %d, caster 1 %d, want 1 and 1", living[0], living[1])
	}
}

func TestALoadedActorIsOutsideTheCap(t *testing.T) {
	w := summonedWorld(t, 0x47, 4)
	var ents []Entity
	for _, e := range w.Entities() {
		loaded := NewActor(ActorDefinition{TypeID: e.TypeID, HP: e.HP, MaxHP: e.MaxHP, TokenSize: e.TokenSize},
			ActorPlacement{ID: e.ID, X: e.X, Y: e.Y, Owner: e.Owner})
		ents = append(ents, loaded)
	}
	cold := secondWorld(t, 0x47, Bounds{Width: 32, Height: 32}, nil, ents...)
	if ids := keyedIDs(cold, 1); len(ids) != 0 {
		t.Fatalf("loaded actors %v carry the caster's key", ids)
	}
	before := cold.RandomState()
	cold.summonCapPass()
	if cold.RandomState() != before {
		t.Fatal("the pass drew for a World with no key")
	}
	for range 16 * 1500 {
		Step(cold, nil)
	}
	if n := livingCreatures(cold); n != 4 {
		t.Fatalf("%d loaded creatures remain, want 4", n)
	}
}
