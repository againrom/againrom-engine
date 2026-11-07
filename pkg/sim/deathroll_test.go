package sim

import (
	"slices"
	"testing"
)

// deathRollGold is the gold one death roll of entity i pays.
func deathRollGold(w *World, i int) uint32 {
	gold, _, _ := w.deathRoll(i)
	return gold
}

// drDragonRoll is a Dragon row's death item columns (R2-ASSET-224).
var drDragonRoll = DeathItemRoll{Chance: 100, Min: 1500, Max: 4500, Mask: 0x2c0fffff}

// drDragon is a dragon-like Units actor at (x,5): gold and an item always.
func drDragon(id EntityID, x int32) Entity {
	return Entity{ID: id, X: x, Y: 5, HP: 10, MaxHP: 10, TypeID: 71,
		GoldChance: 100, TreasureMin: 3000, TreasureMax: 6000, DeathItem: drDragonRoll}
}

// drDraws is a death item draw that takes three rand() values and answers
// three items priced by them, keeping each roll and value it was given.
type drDraws struct {
	rolls []DeathItemRoll
	seen  []int
}

func (d *drDraws) draw(roll DeathItemRoll, rnd func() int) []ItemInstance {
	d.rolls = append(d.rolls, roll)
	var out []ItemInstance
	for k := range 3 {
		v := rnd()
		d.seen = append(d.seen, v)
		out = append(out, ItemInstance{Code: uint16(0x0e05 + k), Price: int32(v)})
	}
	return out
}

// drWorld is a second-game World over ents on the original's stream or the
// seeded one, stocked with stock, with the recording draw bound.
func drWorld(t *testing.T, original bool, ents []Entity, stock []Stock) (*World, *drDraws) {
	t.Helper()
	w, err := NewStockedWorld(7, ddBounds, ModeCanonical, Terrain{}, ents, nil, Relations{}, nil, stock)
	if err != nil {
		t.Fatal(err)
	}
	w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	w.rng = rng{original: original, state: 0x2468ace1}
	d := &drDraws{}
	w.SetDeathItemDraw(d.draw)
	return w, d
}

var drModes = []bool{false, true}

// The second game's death roll draws in the original's order on the World's
// one generator (R2-ENGINE-940): rand(100) for gold, the amount, rand(100)
// for the item, the shop draw's own values, then entry rand(n-1). A replica
// of the generator taken before the roll gives every value; any reordering
// moves a value the draw sees, the gold or the entry.
func TestSecondGameDeathRollDrawsInTheOriginalOrder(t *testing.T) {
	for _, mode := range drModes {
		w, d := drWorld(t, mode, []Entity{drDragon(1, 4)}, nil)
		r := w.rng
		r.uniform(100)
		wantGold := uint32(3000 + r.uniform(6000))
		r.uniform(100)
		wantSeen := []int{int(r.raw()), int(r.raw()), int(r.raw())}
		pick := r.uniform(2)
		gold, item, rolled := w.deathRoll(0)
		if !rolled || gold != wantGold || !slices.Equal(d.seen, wantSeen) || w.rng != r {
			t.Fatalf("original %v: gold %d rolled %v, draw saw %v; want gold %d, draw %v and the replica's state", mode, gold, rolled, d.seen, wantGold, wantSeen)
		}
		if item.Code != uint16(0x0e05+pick) || item.Price != int32(wantSeen[pick]) {
			t.Fatalf("original %v: item %+v, want entry %d of %v", mode, item, pick, wantSeen)
		}
		if !slices.Equal(d.rolls, []DeathItemRoll{drDragonRoll}) {
			t.Fatalf("original %v: the draw took %v", mode, d.rolls)
		}
	}
}

// The original stream at state 0x2468ace1 pays these to drDragon's roll.
var (
	drPinnedGold = uint32(4619)
	drPinnedSeen = []int{13769, 14391, 23297}
	drPinnedCode = uint16(0x0e07)
)

// One fixed seed gives one fixed sequence: the pinned gold, values and
// entry of the original stream at state 0x2468ace1.
func TestSecondGameDeathRollKeepsItsSeededSequence(t *testing.T) {
	w, d := drWorld(t, true, []Entity{drDragon(1, 4)}, nil)
	gold, item, rolled := w.deathRoll(0)
	t.Logf("gold %d item %#04x seen %v", gold, item.Code, d.seen)
	if !rolled || gold != drPinnedGold || !slices.Equal(d.seen, drPinnedSeen) || item.Code != drPinnedCode {
		t.Fatalf("gold %d item %#04x seen %v, want %d, %#04x and %v", gold, item.Code, d.seen, drPinnedGold, drPinnedCode, drPinnedSeen)
	}
}

// A kill leaves the gold and the rolled item in one ground sack at the
// cell, the item after the bag's own holdings.
func TestSecondGameDeathLeavesTheRolledItemInTheSack(t *testing.T) {
	stock := []Stock{{ID: 1, ItemInstances: []ItemInstance{{Code: 0x0e01, Price: 20}}}}
	w, d := drWorld(t, true, []Entity{drDragon(1, 4)}, stock)
	Step(w, []Command{TerminalKill(1)})
	sacks := w.Sacks()
	if len(sacks) != 1 || sacks[0].X != 4 || sacks[0].Y != 5 || sacks[0].Gold < 3000 || sacks[0].Gold > 9000 {
		t.Fatalf("sacks %+v, want one at (4,5) with 3000..9000 gold", sacks)
	}
	items := sacks[0].ItemInstances
	if len(items) != 2 || items[0].Code != 0x0e01 || items[1].Code < 0x0e05 || items[1].Code > 0x0e07 ||
		items[1].Price != int32(d.seen[items[1].Code-0x0e05]) {
		t.Fatalf("sack items %+v, want the carried potion then one drawn entry of %v", items, d.seen)
	}
}

// A template whose bag is discarded still drops the roll (R2-ENGINE-943).
func TestSecondGameDiscardedBagStillDropsTheRoll(t *testing.T) {
	e := drDragon(1, 4)
	e.SuppressCorpseLoot = true
	stock := []Stock{{ID: 1, ItemInstances: []ItemInstance{{Code: 0x0e01, Price: 20}}}}
	w, _ := drWorld(t, true, []Entity{e}, stock)
	Step(w, []Command{TerminalKill(1)})
	sacks := w.Sacks()
	if len(sacks) != 1 || len(sacks[0].ItemInstances) != 1 || sacks[0].ItemInstances[0].Code < 0x0e05 || sacks[0].Gold == 0 {
		t.Fatalf("sacks %+v, want gold and the drawn entry only", sacks)
	}
}

// No item drops for chance 0 (its rand(100) is still drawn), for a type id
// below 64 or a reward mark (neither draws), nor under the first game's
// profile, whose roll draws the gold alone and admits type 64 none.
func TestDeathRollDropsNoItemOutsideTheRule(t *testing.T) {
	for _, tc := range []struct {
		name   string
		edit   func(*Entity)
		first  bool
		replay func(r *rng)
	}{
		{"chance 0", func(e *Entity) { e.DeathItem.Chance = 0 }, false, func(r *rng) { r.uniform(100); r.uniform(6000); r.uniform(100) }},
		{"chance -1", func(e *Entity) { e.DeathItem.Chance = -1 }, false, func(r *rng) { r.uniform(100); r.uniform(6000); r.uniform(100) }},
		{"type 63", func(e *Entity) { e.TypeID = 63 }, false, func(*rng) {}},
		{"reward mark", func(e *Entity) { e.SuppressRewards = true }, false, func(*rng) {}},
		{"first game", func(*Entity) {}, true, func(r *rng) { r.uniform(100); r.uniform(6000) }},
		{"first game type 64", func(e *Entity) { e.TypeID = 64 }, true, func(*rng) {}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := drDragon(1, 4)
			tc.edit(&e)
			w, d := drWorld(t, true, []Entity{e}, nil)
			if tc.first {
				w.game = newGameProfile(ScriptROM1)
			}
			r := w.rng
			tc.replay(&r)
			_, item, rolled := w.deathRoll(0)
			if rolled || !item.Empty() || len(d.rolls) != 0 || w.rng != r {
				t.Fatalf("rolled %v item %+v draws %d; want none and the replica's state", rolled, item, len(d.rolls))
			}
		})
	}
}

// The second game's roll admits type 64: a goblin's gold drops.
func TestSecondGameDeathRollAdmitsTypeSixtyFour(t *testing.T) {
	e := drDragon(1, 4)
	e.TypeID, e.DeathItem = 64, DeathItemRoll{}
	w, _ := drWorld(t, true, []Entity{e}, nil)
	if gold := deathRollGold(w, 0); gold < 3000 {
		t.Fatalf("type 64 paid %d", gold)
	}
}

// An actor whose only reward is its item roll holds terminal loot, so its
// body waits for a cell rather than losing the roll.
func TestAnItemRollIsTerminalLoot(t *testing.T) {
	e := drDragon(1, 4)
	e.GoldChance = 0
	w, _ := drWorld(t, true, []Entity{e}, nil)
	if !w.hasTerminalLoot(0) {
		t.Fatal("the item roll is not terminal loot")
	}
	w.game = newGameProfile(ScriptROM1)
	if w.hasTerminalLoot(0) {
		t.Fatal("the first game counts an item roll")
	}
}

// The byte form carries the roll in its own wrapper only when an actor holds
// one, and a decode restores it; a World without one keeps its form. The
// wrapper does not read the game, so the plain profile carries it here.
func TestTheDeathItemRollCrossesTheByteForm(t *testing.T) {
	w, d := drWorld(t, true, []Entity{drDragon(1, 4)}, nil)
	w.rom2, w.game = nil, newGameProfile(ScriptROM1)
	b, err := w.MarshalBinary()
	if err != nil || b[0] != deathItemFormVersion {
		t.Fatalf("form %d bytes, err %v; want version %d", len(b), err, deathItemFormVersion)
	}
	var back World
	back.SetDeathItemDraw(d.draw)
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if back.entities[0].DeathItem != drDragonRoll || back.Hash() != w.Hash() || back.DeathItemDraw() == nil {
		t.Fatalf("decoded roll %+v, hash equal %v", back.entities[0].DeathItem, back.Hash() == w.Hash())
	}
	plain := drDragon(1, 4)
	plain.DeathItem = DeathItemRoll{}
	p, _ := drWorld(t, true, []Entity{plain}, nil)
	p.rom2, p.game = nil, newGameProfile(ScriptROM1)
	if b, err := p.MarshalBinary(); err != nil || b[0] == deathItemFormVersion {
		t.Fatal("a World without a roll wrote the wrapper")
	}
}
