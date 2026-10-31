package sim

import (
	"fmt"
	"testing"
)

// secondCreatureWorld is creatureWorld in the second game: the creature (1) at
// (5,5) holds the slots, its victim (2) stands at (9,5).
func secondCreatureWorld(t *testing.T, spells []SpellRule, slots [CreatureSpellSlots]CreatureSpell, known uint32, book Spellbook) *World {
	t.Helper()
	creature := engFighter(1, 2, 5, 5)
	creature.Book, creature.KnownSpells, creature.CreatureSpells = book, known, slots
	victim := engFighter(2, 3, 9, 5)
	victim.HP, victim.MaxHP = 5000, 5000
	victim.DamageBase = 0
	rel := engRel(t, [3]uint32{2, 3, relationHostile}, [3]uint32{3, 2, relationLocked})
	s, err := newScript(ScriptROM2, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewStockedSpelledWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{creature, victim}, s, rel, nil, nil, spells)
	if err != nil {
		t.Fatal(err)
	}
	if !w.SecondGame() {
		t.Fatal("not a second-game World")
	}
	wakeGroups(w)
	return w
}

// secondSlotRows is a plain second-game row for every id 1..29.
func secondSlotRows() []SpellRule {
	rows := make([]SpellRule, 0, 29)
	for id := uint16(1); id <= 29; id++ {
		rows = append(rows, SpellRule{ID: id, ManaCost: 3, School: 1, MaxRange: 7})
	}
	AssignSecondGameArms(rows)
	return rows
}

// secondTeleportCreature is a second-game creature whose one slot always draws
// Teleport (23), with its victim at (vx, vy).
func secondTeleportCreature(t *testing.T, vx, vy int32) *World {
	t.Helper()
	book, known := creatureBookOf(map[uint32]uint8{23: 7})
	teleport := secondRow(SpellRule{ID: 23, ManaCost: 3, School: 5, MaxRange: 7, Defensive: true})
	w := secondCreatureWorld(t, []SpellRule{teleport}, creatureCertainSlot(23), known, book)
	w.entities[1].X, w.entities[1].Y = vx, vy
	return w
}

func blockGroundCell(t *testing.T, w *World, x, y int32) {
	t.Helper()
	i, ok := w.cellIndex(x, y)
	if !ok {
		t.Fatalf("cell (%d,%d) is outside the map", x, y)
	}
	w.grid[i] |= blockGround
}

type aimOperands struct {
	target EntityID
	x, y   int32
	atCell bool
	ok     bool
}

func (a aimOperands) String() string {
	if !a.ok {
		return "no cast"
	}
	if a.atCell {
		return fmt.Sprintf("cell (%d,%d)", a.x, a.y)
	}
	return fmt.Sprintf("actor %d", a.target)
}

func creatureOperandsOf(w *World, id uint32) aimOperands {
	target, x, y, atCell, ok := w.creatureAimOperands(0, 2, id)
	if !ok {
		return aimOperands{}
	}
	if !atCell {
		x, y = 0, 0
	}
	return aimOperands{target, x, y, atCell, ok}
}

// R2-ENGINE-653: the special cast aims by the id alone through 29 cells.
func TestASecondGameCreatureAimsEachSlotSpellByTheOriginalsTable(t *testing.T) {
	victim := aimOperands{target: 2, ok: true}
	caster := aimOperands{target: 1, ok: true}
	victimCell := aimOperands{x: 9, y: 5, atCell: true, ok: true}
	want := map[uint32]aimOperands{}
	for _, id := range []uint32{1, 5, 10, 11, 16, 18, 26, 28, 29} {
		want[id] = victim
	}
	for _, id := range []uint32{4, 8, 12, 13, 19, 20, 21, 24, 25, 27} {
		want[id] = caster
	}
	for _, id := range []uint32{2, 3, 6, 7, 14, 15, 17} {
		want[id] = victimCell
	}
	// Id 9: the caster's cell plus the step of the heading toward the victim
	// (east). Id 23: the first open cell of the 3 by 3 around the victim.
	want[9] = aimOperands{x: 6, y: 5, atCell: true, ok: true}
	want[23] = aimOperands{x: 8, y: 4, atCell: true, ok: true}
	want[22] = aimOperands{}
	want[30] = aimOperands{}
	w := secondCreatureWorld(t, secondSlotRows(), [CreatureSpellSlots]CreatureSpell{}, 0, Spellbook{State: BookPresent})
	for id := uint32(1); id <= 30; id++ {
		if got := creatureOperandsOf(w, id); got != want[id] {
			t.Errorf("spell %d aims at %v, want %v", id, got, want[id])
		}
	}
}

// R2-ENGINE-654: id 9 adds the offset of one of eight headings to the
// caster's cell; the engine's offsets are one step along the heading.
func TestASecondGameAcidStreamAimsOneStepAlongEachOfEightHeadings(t *testing.T) {
	w := secondCreatureWorld(t, secondSlotRows(), [CreatureSpellSlots]CreatureSpell{}, 0, Spellbook{State: BookPresent})
	for _, tc := range []struct {
		name   string
		dx, dy int32
		x, y   int32
	}{
		{"0: dy below 0 dominates", 0, -4, 5, 4},
		{"1: dx above 0, dy at most 0", 3, -2, 6, 4},
		{"2: dx above 0 dominates", 4, 1, 6, 5},
		{"3: dx and dy above 0", 2, 1, 6, 6},
		{"4: dy above 0 dominates", -1, 4, 5, 6},
		{"5: dx at most 0, dy above 0", -2, 3, 4, 6},
		{"6: dx below 0 dominates", -4, 0, 4, 5},
		{"7: dx and dy at most 0", -3, -3, 4, 4},
		{"7: dx = dy = 0", 0, 0, 4, 4},
	} {
		w.entities[1].X, w.entities[1].Y = 5+tc.dx, 5+tc.dy
		if got, want := creatureOperandsOf(w, 9), (aimOperands{x: tc.x, y: tc.y, atCell: true, ok: true}); got != want {
			t.Errorf("heading %s: Acid Stream aims at %v, want %v", tc.name, got, want)
		}
	}
}

// R2-ENGINE-654: id 23 scans the 3 by 3 around the victim, rows outer and
// columns inner, for the first cell where the caster's whole footprint is open.
func TestASecondGameCreatureTeleportScansTheThreeByThreeAroundItsVictim(t *testing.T) {
	t.Run("rows outer, columns inner", func(t *testing.T) {
		w := secondTeleportCreature(t, 9, 5)
		blockGroundCell(t, w, 8, 4)
		if got, want := creatureOperandsOf(w, 23), (aimOperands{x: 9, y: 4, atCell: true, ok: true}); got != want {
			t.Errorf("Teleport aims at %v, want %v", got, want)
		}
		blockGroundCell(t, w, 9, 4)
		blockGroundCell(t, w, 10, 4)
		if got, want := creatureOperandsOf(w, 23), (aimOperands{x: 8, y: 5, atCell: true, ok: true}); got != want {
			t.Errorf("with the top row blocked Teleport aims at %v, want %v", got, want)
		}
	})
	t.Run("the caster's footprint", func(t *testing.T) {
		w := secondTeleportCreature(t, 9, 5)
		w.entities[0].TokenSize = 2
		blockGroundCell(t, w, 9, 4)
		if got, want := creatureOperandsOf(w, 23), (aimOperands{x: 10, y: 4, atCell: true, ok: true}); got != want {
			t.Errorf("a two-cell caster's Teleport aims at %v, want %v", got, want)
		}
	})
	t.Run("no open cell", func(t *testing.T) {
		w := secondTeleportCreature(t, 9, 5)
		for y := int32(4); y <= 6; y++ {
			for x := int32(8); x <= 10; x++ {
				blockGroundCell(t, w, x, y)
			}
		}
		if got := creatureOperandsOf(w, 23); got.ok {
			t.Errorf("with every cell blocked Teleport aims at %v", got)
		}
	})
	t.Run("the packed cell 0", func(t *testing.T) {
		w := secondTeleportCreature(t, 1, 1)
		if got := creatureOperandsOf(w, 23); got.ok {
			t.Errorf("a first open cell (0,0) aims at %v, want no cast", got)
		}
	})
	t.Run("at distance 2 or less", func(t *testing.T) {
		w := secondTeleportCreature(t, 7, 5)
		if got := creatureOperandsOf(w, 23); got.ok {
			t.Errorf("at distance 2 Teleport aims at %v, want no cast", got)
		}
	})
}

// engageCall is what one planner call left: the cast it armed or began, the
// attack it prepared, and whether it drew.
type engageCall struct {
	took   bool
	cast   bool
	castAt aimOperands
	attack bool
	drew   bool
}

func runEngageCall(w *World) engageCall {
	before := w.rng
	took := w.orderAttack(0, 2)
	got := engageCall{took: took, drew: w.rng != before}
	if k, ok := w.bookCastIndex(1); ok {
		c := w.bookCasts[k]
		got.cast = true
		got.castAt = aimOperands{target: c.Target, atCell: c.AtCell, ok: true}
		if c.AtCell {
			got.castAt.x, got.castAt.y = c.X, c.Y
		}
	}
	e := w.entities[0]
	got.attack = e.HasAttackTarget && e.AttackTarget == 2
	return got
}

// R2-ENGINE-655: each gate stops the slot spell in path order; after a
// special-cast call the planner prepares no attack, after a failed draw or a
// Teleport within 2 it prepares the attack.
func TestASecondGameCreatureSlotSpellStopsAtEachGateInOrder(t *testing.T) {
	arrow := secondRow(SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 7, TargetsUnit: true})
	exit := secondRow(SpellRule{ID: 22, ManaCost: 3, School: 5, MaxRange: 7})
	for _, tc := range []struct {
		name  string
		world func(t *testing.T) *World
		want  engageCall
	}{
		{"1: victim off the map draws nothing and attacks not", func(t *testing.T) *World {
			w := secondTeleportCreature(t, 9, 5)
			w.entities[1].OffMap = true
			return w
		}, engageCall{took: true}},
		{"2: a failed draw attacks", func(t *testing.T) *World {
			book, known := creatureBookOf(map[uint32]uint8{1: 7})
			slots := [CreatureSpellSlots]CreatureSpell{{ID: 1, Threshold: 0}}
			return secondCreatureWorld(t, []SpellRule{arrow}, slots, known, book)
		}, engageCall{drew: true, took: true, attack: true}},
		{"3: Teleport within 2 attacks", func(t *testing.T) *World {
			return secondTeleportCreature(t, 7, 5)
		}, engageCall{drew: true, took: true, attack: true}},
		{"4: a spell the book does not hold", func(t *testing.T) *World {
			book, known := creatureBookOf(map[uint32]uint8{2: 7})
			wave := secondRow(SpellRule{ID: 2, ManaCost: 3, School: 1, MaxRange: 7, Area: true, Distribution: 3, Radius: 1})
			return secondCreatureWorld(t, []SpellRule{arrow, wave}, creatureCertainSlot(1), known, book)
		}, engageCall{drew: true, took: true}},
		{"5: id 22 has no aim", func(t *testing.T) *World {
			book, known := creatureBookOf(map[uint32]uint8{22: 7})
			return secondCreatureWorld(t, []SpellRule{exit}, creatureCertainSlot(22), known, book)
		}, engageCall{drew: true, took: true}},
		{"6: Teleport with no open cell", func(t *testing.T) *World {
			w := secondTeleportCreature(t, 9, 5)
			for y := int32(4); y <= 6; y++ {
				for x := int32(8); x <= 10; x++ {
					blockGroundCell(t, w, x, y)
				}
			}
			return w
		}, engageCall{drew: true, took: true}},
		{"past every gate Teleport casts at the scanned cell", func(t *testing.T) *World {
			return secondTeleportCreature(t, 8, 5)
		}, engageCall{drew: true, took: true, cast: true, castAt: aimOperands{x: 7, y: 4, atCell: true, ok: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := runEngageCall(tc.world(t)); got != tc.want {
				t.Errorf("planner call %+v, want %+v", got, tc.want)
			}
		})
	}
}

// The first game's creature aim is the arm table for every id, read through
// the World as the engage reads it (MAGIC-237, MAGIC-238).
func TestTheFirstGameCreatureAimIsTheArmTableForEveryID(t *testing.T) {
	rows := make([]SpellRule, 0, 28)
	for id := uint16(1); id <= 28; id++ {
		rows = append(rows, SpellRule{ID: id, ManaCost: 3, School: 1, MaxRange: 7})
	}
	w := creatureWorld(t, rows, [CreatureSpellSlots]CreatureSpell{}, 0, Spellbook{State: BookPresent})
	if w.SecondGame() {
		t.Fatal("a first-game World reads as the second game")
	}
	want := map[uint32]aimOperands{}
	for _, id := range []uint32{1, 11, 13, 14, 20, 27, 28} {
		want[id] = aimOperands{target: 2, ok: true}
	}
	for _, id := range []uint32{5, 6, 10, 15, 16, 18, 22, 23, 24} {
		want[id] = aimOperands{target: 1, ok: true}
	}
	for _, id := range []uint32{2, 3, 7, 8, 12, 17, 19, 21} {
		want[id] = aimOperands{x: 9, y: 5, atCell: true, ok: true}
	}
	for _, id := range []uint32{9, 26} {
		want[id] = aimOperands{x: 6, y: 5, atCell: true, ok: true}
	}
	for id := uint32(0); id <= 30; id++ {
		if got := creatureOperandsOf(w, id); got != want[id] {
			t.Errorf("first-game spell %d aims at %v, want %v", id, got, want[id])
		}
	}
	// Teleport keeps the step cell next to the caster at any distance, and an
	// off-map victim still draws.
	w.entities[1].X = 7
	if got, want := creatureOperandsOf(w, 26), (aimOperands{x: 6, y: 5, atCell: true, ok: true}); got != want {
		t.Errorf("first-game Teleport within 2 aims at %v, want %v", got, want)
	}
}

func TestTheFirstGameCreatureDrawsForAnOffMapVictim(t *testing.T) {
	book, known := creatureBookOf(map[uint32]uint8{1: 7})
	w := creatureWorld(t, creatureAimRules(), creatureCertainSlot(1), known, book)
	w.entities[1].OffMap = true
	before := w.rng
	w.creatureEngageCast(0, 2)
	if w.rng == before {
		t.Error("the first game's creature did not draw for an off-map victim")
	}
}

// The first game's creature casts keep their draws, aims, orders and hashed
// state: each run's World hash is the one main 934caab9d gives.
func TestTheFirstGameCreatureCastsKeepTheirHashedState(t *testing.T) {
	want := map[uint32]uint64{1: 0x46b6f3a6ba8f571e, 2: 0x57dddf22712a4e6, 9: 0xa1120ebb55eb18bb,
		15: 0x840528414f23c26b, 26: 0xc5e80daf0982e7c4}
	for _, id := range []uint32{1, 2, 9, 15, 26} {
		book, known := creatureBookOf(map[uint32]uint8{id: 7})
		slots := [CreatureSpellSlots]CreatureSpell{{ID: id, Threshold: 50 * CreatureSpellScale}}
		w := creatureWorld(t, creatureAimRules(), slots, known, book)
		w.entities[1].X, w.entities[1].Y = 12, 7
		engRun(w, 8)
		if got := w.Hash(); got != want[id] {
			t.Errorf("first-game creature with spell %d: World hash %#x, want %#x", id, got, want[id])
		}
	}
}
