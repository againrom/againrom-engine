package sim

import (
	"bytes"
	"slices"
	"testing"
)

func TestCreatureSpellForScalesTheProbability(t *testing.T) {
	if got := CreatureSpellFor(18, 10); got != (CreatureSpell{ID: 18, Threshold: 10 * 0x147}) {
		t.Errorf("slot = %+v, want id 18 threshold %d", got, 10*0x147)
	}
	if got := CreatureSpellFor(100, 100).Threshold; got != 32700 {
		t.Errorf("probability 100 scales to %d, want 32700", got)
	}
	for _, spell := range []int32{0, -1} {
		if got := CreatureSpellFor(spell, 10); got != (CreatureSpell{}) {
			t.Errorf("empty spell cell %d built slot %+v", spell, got)
		}
	}
	if got := CreatureSpellFor(5, -1); int32(got.Threshold) >= 0 {
		t.Errorf("an empty probability cell scaled to %d, want a negative threshold", int32(got.Threshold))
	}
}

func TestCreatureSpellArmsSplitTwentyEightIDsSevenNineEightTwoTwo(t *testing.T) {
	got := map[creatureAim][]uint32{}
	for id := uint32(1); id <= 28; id++ {
		got[creatureAimOf(id)] = append(got[creatureAimOf(id)], id)
	}
	want := map[creatureAim][]uint32{
		creatureAimVictim:     {1, 11, 13, 14, 20, 27, 28},
		creatureAimCaster:     {5, 6, 10, 15, 16, 18, 22, 23, 24},
		creatureAimVictimCell: {2, 3, 7, 8, 12, 17, 19, 21},
		creatureAimStepCell:   {9, 26},
		creatureAimNone:       {4, 25},
	}
	for aim, ids := range want {
		if !slices.Equal(got[aim], ids) {
			t.Errorf("arm %d holds ids %v, want %v", aim, got[aim], ids)
		}
	}
	if creatureAimOf(0) != creatureAimNone || creatureAimOf(29) != creatureAimNone {
		t.Error("an id outside 1..28 reached an order")
	}
}

func TestCreatureSpellDrawIsPerSlotAndLaterSlotsOverwrite(t *testing.T) {
	always := uint32(creatureDrawMax + 1)
	e := Entity{CreatureSpells: [CreatureSpellSlots]CreatureSpell{{ID: 1, Threshold: always}, {ID: 5, Threshold: always}, {}}}
	w := &World{}
	w.rng = rng{state: 7}
	before := w.rng
	if got := w.creatureSpellPick(e); got != 5 {
		t.Errorf("two certain slots picked %d, want the later slot's 5", got)
	}
	// One draw per non-empty slot and none for an empty one.
	want := before
	want.uniform(creatureDrawMax)
	want.uniform(creatureDrawMax)
	if w.rng != want {
		t.Error("the draw did not consume exactly one number per non-empty slot")
	}
	e.CreatureSpells[1].Threshold = 0
	if got := w.creatureSpellPick(e); got != 1 {
		t.Errorf("a certain first slot and a never second slot picked %d, want 1", got)
	}
	e.CreatureSpells[0].Threshold = 0
	if got := w.creatureSpellPick(e); got != 0 {
		t.Errorf("no slot can hit but %d was picked", got)
	}
}

func TestCreatureSpellDrawRateIsTheScaledProbability(t *testing.T) {
	e := Entity{CreatureSpells: [CreatureSpellSlots]CreatureSpell{CreatureSpellFor(1, 10)}}
	w := &World{}
	w.rng = rng{state: 11}
	hits := 0
	const draws = 40000
	for range draws {
		if w.creatureSpellPick(e) == 1 {
			hits++
		}
	}
	// 10 x 0x147 of 0x8000 draws is 9.98 percent.
	if hits < draws*95/1000 || hits > draws*105/1000 {
		t.Errorf("%d hits in %d draws, want about 9.98 percent", hits, draws)
	}
}

// creatureWorld is a hostile creature with a class book at (5,5) and a victim
// in sight and spell range. The creature has no mana pool.
func creatureWorld(t *testing.T, spells []SpellRule, slots [CreatureSpellSlots]CreatureSpell, known uint32, book Spellbook) *World {
	t.Helper()
	creature := engFighter(1, 2, 5, 5)
	creature.Book, creature.KnownSpells, creature.CreatureSpells = book, known, slots
	victim := engFighter(2, 3, 9, 5)
	victim.HP, victim.MaxHP = 5000, 5000
	victim.DamageBase = 0
	rel := engRel(t, [3]uint32{2, 3, relationHostile}, [3]uint32{3, 2, relationLocked})
	w, err := NewStockedSpelledWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{creature, victim}, nil, rel, nil, nil, spells)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func creatureArrowBook() Spellbook {
	book := Spellbook{State: BookPresent}
	book.Slots[0] = BookSpell{Range: 7, ManaCost: 3}
	return book
}

func creatureEntity(w *World) Entity { return w.entities[indexOfEntity(w.entities, 1)] }

var creatureCertain = [CreatureSpellSlots]CreatureSpell{{ID: 1, Threshold: creatureDrawMax + 1}}

func TestACreatureCastsItsClassSpellInsteadOfEngaging(t *testing.T) {
	w := creatureWorld(t, []SpellRule{hlArrow()}, creatureCertain, 1<<1, creatureArrowBook())
	engRun(w, 1)
	if _, pending := w.bookCastIndex(1); !pending {
		t.Fatal("the creature began no cast on its first decision")
	}
	if _, held := engVictim(w, 1); held {
		t.Error("the creature engaged instead of casting")
	}
	for range 200 {
		Step(w, nil)
	}
	if hp := w.entities[indexOfEntity(w.entities, 2)].HP; hp >= 5000 {
		t.Error("the cast never reached its victim")
	}
	if e := creatureEntity(w); e.Mana != 0 || e.MaxMana != 0 {
		t.Errorf("a creature cast changed its mana to %d of %d", e.Mana, e.MaxMana)
	}
}

func TestACreatureWithoutSlotsABookOrAnOrderingIDDoesNotCast(t *testing.T) {
	noOrderBook := Spellbook{State: BookPresent}
	noOrderBook.Slots[3] = BookSpell{ManaCost: 30}
	for _, tc := range []struct {
		name   string
		slots  [CreatureSpellSlots]CreatureSpell
		known  uint32
		book   Spellbook
		engage bool
	}{
		{"no slots", [CreatureSpellSlots]CreatureSpell{}, 1 << 1, creatureArrowBook(), true},
		{"no book", creatureCertain, 0, Spellbook{State: BookAbsent}, false},
		{"id the book lacks", [CreatureSpellSlots]CreatureSpell{{ID: 2, Threshold: creatureDrawMax + 1}}, 1 << 1, creatureArrowBook(), false},
		{"id with no order", [CreatureSpellSlots]CreatureSpell{{ID: 4, Threshold: creatureDrawMax + 1}}, 1 << 4, noOrderBook, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := creatureWorld(t, []SpellRule{hlArrow(), {ID: 4, ManaCost: 30, MaxRange: 5, TargetsUnit: true}}, tc.slots, tc.known, tc.book)
			engRun(w, 1)
			if _, pending := w.bookCastIndex(1); pending {
				t.Error("a cast began")
			}
			if _, held := engVictim(w, 1); held != tc.engage {
				t.Errorf("creature holds a victim = %v, want %v", held, tc.engage)
			}
		})
	}
}

func TestCreatureSpellsSurviveTheByteFormAndMoveTheDigest(t *testing.T) {
	plain := creatureWorld(t, []SpellRule{hlArrow()}, [CreatureSpellSlots]CreatureSpell{}, 1<<1, creatureArrowBook())
	w := creatureWorld(t, []SpellRule{hlArrow()}, [CreatureSpellSlots]CreatureSpell{{ID: 1, Threshold: 3270}, {ID: 28, Threshold: 99}}, 1<<1, creatureArrowBook())
	plainForm, err := plain.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if plainForm[0] != formatVersion {
		t.Errorf("a world with no slots wrote version %d, want %d", plainForm[0], formatVersion)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if form[0] != creatureSpellFormVersion || !bytes.HasSuffix(form, []byte("CSP1")) {
		t.Fatalf("a world with slots wrote version %d", form[0])
	}
	if HasStructureBlockingForm(form) != HasStructureBlockingForm(plainForm) {
		t.Error("the slot trailer changed what the structure blocking probe sees")
	}
	if plain.Hash() == w.Hash() {
		t.Error("two worlds differing only in a creature's slots share a digest")
	}
	back := worldRoundTripForTest(t, w)
	if got := creatureEntity(back).CreatureSpells; got != creatureEntity(w).CreatureSpells {
		t.Errorf("slots reloaded as %+v", got)
	}
	if back.Hash() != w.Hash() {
		t.Error("a reload changed the digest")
	}
	for _, cut := range []int{1, 9, 4 + creatureSpellRecordLen} {
		var bad World
		if err := bad.UnmarshalBinary(form[:len(form)-cut]); err == nil {
			t.Errorf("a form missing %d trailing bytes loaded", cut)
		}
	}
	corrupt := bytes.Clone(form)
	corrupt[len(corrupt)-9-creatureSpellRecordLen] ^= 0x40
	var bad World
	if err := bad.UnmarshalBinary(corrupt); err == nil {
		t.Error("a slot record naming an absent entity loaded")
	}
}

func TestACreatureCastContinuesAcrossAReload(t *testing.T) {
	w := creatureWorld(t, []SpellRule{hlArrow()}, creatureCertain, 1<<1, creatureArrowBook())
	engRun(w, 1)
	back := worldRoundTripForTest(t, w)
	for tick := range 220 {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("the reloaded world diverged at tick %d", tick)
		}
	}
	if hp := back.entities[indexOfEntity(back.entities, 2)].HP; hp >= 5000 {
		t.Error("the reloaded world never completed the cast")
	}
	// The next cast is the reloaded creature's own draw from its stored slots.
	casts := 0
	for range 400 {
		Step(back, nil)
		if _, pending := back.bookCastIndex(1); pending {
			casts++
		}
	}
	if casts == 0 {
		t.Error("the reloaded creature never cast again")
	}
}

func TestAHeldOrderKeepsItsLoadedSlotBytesAcrossAReload(t *testing.T) {
	slots := [CreatureSpellSlots]CreatureSpell{{ID: 1, Threshold: 3270}, {ID: 28, Threshold: 99}}
	for _, tc := range []struct {
		name   string
		window func(Entity) []byte
	}{
		{"window equal to the slots", func(e Entity) []byte { w := e.slotWindow(); return w[:] }},
		{"window holding other bytes", func(Entity) []byte {
			return []byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4}
		}},
		{"window all zero", func(Entity) []byte { return make([]byte, orderSlotWindowEnd-orderSlotWindowStart) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := creatureWorld(t, []SpellRule{hlArrow()}, slots, 1<<1, creatureArrowBook())
			var raw [144]byte
			raw[0x10] = 0x55
			raw[0x70] = 0x66
			copy(raw[orderSlotWindowStart:], tc.window(creatureEntity(w)))
			group := SavedGroup{ID: 1, Members: []SavedGroupMember{{Archive: 1, Entity: 1, Bound: true}}}
			if err := w.ImportSavedGroups([]SavedGroup{group}, []SavedActorOrder{{Entity: 1, Raw: raw}}); err != nil {
				t.Fatal(err)
			}
			_, held, _ := w.SavedGroups()
			if len(held) != 1 || held[0].Raw != raw {
				t.Fatalf("the held order changed its loaded bytes")
			}
			back := worldRoundTripForTest(t, w)
			if back.Hash() != w.Hash() {
				t.Error("a reload changed the digest")
			}
			if got := creatureEntity(back).CreatureSpells; got != slots {
				t.Errorf("slots reloaded as %+v", got)
			}
			_, again, _ := back.SavedGroups()
			if len(again) != 1 || again[0].Raw[0x10] != 0x55 || again[0].Raw[0x70] != 0x66 {
				t.Error("a reload lost order bytes outside the slot window")
			}
		})
	}
}
