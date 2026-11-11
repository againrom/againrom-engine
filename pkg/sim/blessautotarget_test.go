package sim

import "testing"

// An unbidden Bless never chooses a mage, the caster included; a warrior
// stays its target, and a player's order still casts it on any mage (owner,
// DIV-4830, DIV-4831). A mage is a human with a mana pool or a creature of
// the game's spared types; any other creature with mana is a warrior.

// The Units type ids these tests place: the first game's Dragon, and the
// second game's Orc Shaman, Necromancer and Druid.
const (
	blessDragon      int32 = 71
	blessOrcShaman   int32 = 81
	blessNecromancer int32 = 99
	blessDruid       int32 = 100
)

func blessMageCompanion(id EntityID, x int32) Entity {
	e := spMage(id, x, 2, 30, 40, 40, 0)
	e.Owner, e.Humanoid = SelfSlot, true
	return e
}

// blessCreatureCompanion is the player's creature of typeID with a mana pool.
func blessCreatureCompanion(id EntityID, x int32, typeID int32) Entity {
	e := spMage(id, x, 2, 30, 40, 40, 0)
	e.Owner, e.TypeID = SelfSlot, typeID
	return e
}

// blessWorld is acsWorld under the first game's profile or the second's.
func blessWorld(t *testing.T, second bool, setting AutoCastSetting, known uint32, armed uint16, targets ...Entity) *World {
	t.Helper()
	w := acsWorld(t, Relations{}, setting, known, armed, targets...)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

// The level policy of the human player's mage: Average lists Bless then Haste
// per candidate, the caster first.
func TestAnAutomaticBlessSparesEveryMageAndTakesAWarrior(t *testing.T) {
	for _, tc := range []struct {
		name      string
		setting   AutoCastSetting
		known     uint32
		companion Entity
		spell     uint16
		target    EntityID
		cast      bool
	}{
		{"Bless passes over its caster to the warrior", AutoCastAverage | AutoCastOwn, acsMask(23), acsUnit(2, SelfSlot, 4, 100), 23, 2, true},
		{"Bless with only mages in sight casts nothing", AutoCastAverage | AutoCastOwn, acsMask(23), blessMageCompanion(2, 4), 0, 0, false},
		{"the caster takes Haste where Bless spares it", AutoCastAverage | AutoCastOwn, acsMask(23, 24), acsUnit(2, SelfSlot, 4, 100), 24, 1, true},
		{"Shield still lands on its caster", AutoCastMinimum | AutoCastOwn, acsMask(18, 23), acsUnit(2, SelfSlot, 4, 100), 18, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := acsWorld(t, Relations{}, tc.setting, tc.known, 0, tc.companion)
			spell, target, cast := acsLevel(w)
			if cast != tc.cast || spell != tc.spell || target != tc.target {
				t.Fatalf("cast %v spell %d at %d, want %v spell %d at %d", cast, spell, target, tc.cast, tc.spell, tc.target)
			}
		})
	}
}

// The spared kinds against a creature with mana outside them, by the level
// policy and by the armed row. The first game spares no creature type, so a
// type the second game spares is a warrior there.
func TestAnAutomaticBlessSparesHumanMagesNecromancersAndDruidsOnly(t *testing.T) {
	for _, tc := range []struct {
		name      string
		second    bool
		companion Entity
		blessed   bool
	}{
		{"first game human mage", false, blessMageCompanion(2, 4), false},
		{"first game Dragon", false, blessCreatureCompanion(2, 4, blessDragon), true},
		{"first game creature of the Necromancer type", false, blessCreatureCompanion(2, 4, blessNecromancer), true},
		{"second game human mage", true, blessMageCompanion(2, 4), false},
		{"second game Necromancer", true, blessCreatureCompanion(2, 4, blessNecromancer), false},
		{"second game Druid", true, blessCreatureCompanion(2, 4, blessDruid), false},
		{"second game Orc Shaman", true, blessCreatureCompanion(2, 4, blessOrcShaman), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, route := range []struct {
				name    string
				setting AutoCastSetting
				armed   uint16
				attempt func(*World) (uint16, EntityID, bool)
			}{{"level", AutoCastAverage | AutoCastOwn, 0, acsLevel}, {"armed", 0, 23, acsFirst}} {
				w := blessWorld(t, tc.second, route.setting, acsMask(23), route.armed, tc.companion)
				spell, target, cast := route.attempt(w)
				if cast != tc.blessed || cast && (spell != 23 || target != 2) {
					t.Errorf("%s: cast %v spell %d at %d, want Bless at 2 %v", route.name, cast, spell, target, tc.blessed)
				}
			}
		})
	}
}

// A heal is not spared: the hurt mage heals itself first.
func TestAnAutomaticHealStillTakesItsMageCaster(t *testing.T) {
	w := acsWorld(t, Relations{}, AutoCastAverage|AutoCastOwn, acsMask(6, 23), 0, acsUnit(2, SelfSlot, 4, 100))
	w.entities[0].HP = 40
	if spell, target, cast := acsLevel(w); !cast || spell != 6 || target != 1 {
		t.Fatalf("cast %v spell %d at %d, want Heal at the caster", cast, spell, target)
	}
}

// A Bless armed in the book is cast on its own: the target is chosen
// automatically, so it spares mages as the level policy does.
func TestAnArmedBlessSparesEveryMage(t *testing.T) {
	w := acsWorld(t, Relations{}, 0, acsMask(23), 23, acsUnit(2, SelfSlot, 4, 100))
	if spell, target, cast := acsFirst(w); !cast || spell != 23 || target != 2 {
		t.Fatalf("armed Bless: cast %v spell %d at %d, want 23 at the warrior 2", cast, spell, target)
	}
	w = acsWorld(t, Relations{}, 0, acsMask(23), 23, blessMageCompanion(2, 4))
	if spell, target, cast := acsFirst(w); cast {
		t.Fatalf("armed Bless with only mages: cast %d at %d", spell, target)
	}
}

// The player's order is not the automatic choice: it casts Bless on the
// caster, on another human mage and on a Necromancer.
func TestAnOrderedBlessStillLandsOnAMage(t *testing.T) {
	for _, tc := range []struct {
		second    bool
		companion Entity
		target    EntityID
	}{
		{false, blessMageCompanion(2, 4), 1}, {false, blessMageCompanion(2, 4), 2},
		{true, blessCreatureCompanion(2, 4, blessNecromancer), 1}, {true, blessCreatureCompanion(2, 4, blessNecromancer), 2},
	} {
		w := blessWorld(t, tc.second, 0, acsMask(23), 0, tc.companion)
		Step(w, []Command{Cast(1, tc.target, SpellID(23))})
		k, ok := w.bookCastIndex(1)
		if !ok || w.bookCasts[k].Spell != 23 || w.bookCasts[k].Target != tc.target {
			t.Fatalf("second game %v: ordered Bless at mage %d began %v %+v", tc.second, tc.target, ok, w.bookCasts)
		}
	}
}

// blessSlotCreature is a computer creature whose one slot always draws Bless,
// its book holding Bless and Fire Arrow; kind shapes the caster.
func blessSlotCreature(t *testing.T, second bool, kind func(*Entity)) *World {
	t.Helper()
	bless, arrow := uint32(23), uint32(1)
	rows := []SpellRule{hlArrow(), acsBuff(23)}
	if second {
		bless = 20
		rows = []SpellRule{secondRow(hlArrow()), secondRow(acsBuff(20))}
	}
	book, known := creatureBookOf(map[uint32]uint8{bless: 7, arrow: 7})
	book.Slots[bless-1].Defensive = 1
	var w *World
	if second {
		w = secondCreatureWorld(t, rows, creatureCertainSlot(bless), known, book)
	} else {
		w = creatureWorld(t, rows, creatureCertainSlot(bless), known, book)
	}
	kind(&w.entities[indexOfEntity(w.entities, 1)])
	return w
}

func blessManaless(*Entity) {}

func blessHumanMage(e *Entity) { e.Humanoid, e.MaxMana, e.Mana = true, 50, 50 }

func blessManaCreature(typeID int32) func(*Entity) {
	return func(e *Entity) { e.TypeID, e.MaxMana, e.Mana = typeID, 50, 50 }
}

// The computer's slot draw aims Bless at the caster. A spared mage drops the
// pick and its engage follows: here its book's Fire Arrow at the victim. A
// creature outside the spared kinds, with mana or without, blesses itself.
func TestAComputerMageDrawingBlessDoesNotBlessItself(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second bool
		kind   func(*Entity)
		spell  uint16
		target EntityID
	}{
		{"first game human mage", false, blessHumanMage, 1, 2},
		{"first game creature", false, blessManaless, 23, 1},
		{"first game Dragon", false, blessManaCreature(blessDragon), 23, 1},
		{"second game human mage", true, blessHumanMage, 1, 2},
		{"second game Necromancer", true, blessManaCreature(blessNecromancer), 1, 2},
		{"second game Druid", true, blessManaCreature(blessDruid), 1, 2},
		{"second game creature", true, blessManaless, 20, 1},
		{"second game Orc Shaman", true, blessManaCreature(blessOrcShaman), 20, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := blessSlotCreature(t, tc.second, tc.kind)
			w.orderAttack(0, 2)
			k, ok := w.bookCastIndex(1)
			if !ok {
				t.Fatal("the creature began no cast")
			}
			if c := w.bookCasts[k]; c.Spell != tc.spell || c.Target != tc.target {
				t.Fatalf("cast spell %d at %d, want %d at %d", c.Spell, c.Target, tc.spell, tc.target)
			}
		})
	}
}
