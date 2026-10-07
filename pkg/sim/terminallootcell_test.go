package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func terminalLootFixture(t *testing.T, route string) *World {
	t.Helper()
	item := ItemInstance{Code: 0xe01, WeightPresent: true, Weight: 3}
	var w *World
	if route == "source" || route == "source objects" {
		w = sourceMutationWorld(t, item)
	} else {
		w = bookWorld(t, Entity{ID: 1, HP: 50, MaxHP: 100}, item)
	}
	w.entities[0].X, w.entities[0].Y = 3, 3
	w.entities[0].TypeID, w.entities[0].GoldChance = 0x41, 101
	w.entities[0].TreasureMin = 23
	if route == "native objects" || route == "source objects" {
		bindOperationsPack(t, w, 0)
	}
	return w
}

func TestTerminalLootRelocatesBlockedOriginAcrossProducers(t *testing.T) {
	for _, route := range []string{"native", "source", "native objects", "source objects"} {
		t.Run(route, func(t *testing.T) {
			w := terminalLootFixture(t, route)
			w.grid[3*8+3] = blockGround
			if !w.dropTerminalLoot(0) {
				t.Fatal("terminal transfer refused")
			}
			if len(w.sacks) != 1 || w.sacks[0].X != 2 || w.sacks[0].Y != 2 || w.sacks[0].Gold != 23 || !reflect.DeepEqual(w.sacks[0].Items, []uint16{0xe01}) {
				t.Fatalf("loot must move to the first free ground neighbour: %+v", w.sacks)
			}
			if len(w.carried[0]) != 0 || w.entities[0].X != 3 || w.entities[0].Y != 3 {
				t.Fatal("transfer retained carried items or moved the body")
			}
			if w.savedObjects != nil {
				s := w.sacks[0]
				container := w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: s.ObjectID})
				if s.ObjectID == 0 || len(s.ItemInstances) != 1 || s.ItemInstances[0].ObjectID != 10 || w.savedObjects.item(10).Retired || container == nil || !reflect.DeepEqual(container.Items, []SavedObjectID{10}) {
					t.Fatal("relocation lost the original item identity or owner")
				}
				if binary.LittleEndian.Uint16(w.savedObjects.sack(s.ObjectID).Token.Position[2:]) != 0x0202 {
					t.Fatal("sack token stayed at the corpse cell")
				}
			}
		})
	}
}

func TestTerminalLootSearchReadsSavedMasksAndFreeFootprints(t *testing.T) {
	w, err := NewStockedWorld(1, Bounds{8, 8}, ModeCanonical, Terrain{}, []Entity{
		{ID: 1, X: 3, Y: 3, HP: 50, MaxHP: 100, TypeID: 0x41, GoldChance: 101, TreasureMin: 23},
		{ID: 2, X: 2, Y: 3, TokenSize: 2, HP: 10, MaxHP: 10},
	}, nil, Relations{}, nil, []Stock{{ID: 1, Items: []uint16{0xe01}}})
	if err != nil {
		t.Fatal(err)
	}
	p := &SavedCellPlanes{}
	p.Static[0x0303] = 0x21
	p.Static[0x0202] = 1
	p.Dynamic[0x0203] = 1
	w.grid[2*8+4] = blockMagicWall
	w.savedCellPlanes = p
	w.pourSack(4, 3, 7, nil)
	if !w.dropTerminalLoot(0) {
		t.Fatal("terminal transfer refused")
	}
	if len(w.sacks) != 2 || w.sacks[0].X != 4 || w.sacks[0].Y != 3 || w.sacks[0].Gold != 7 || w.sacks[1].X != 4 || w.sacks[1].Y != 4 || w.sacks[1].Gold != 23 {
		t.Fatalf("used static/dynamic/wall, occupied footprint or existing sack: %+v", w.sacks)
	}
}

func TestTerminalLootFlyingDeathLeavesAccessibleSackAfterBodyRemoval(t *testing.T) {
	w := terminalLootFixture(t, "native objects")
	w.entities[0].Domain = DomainAir
	w.entities[0].DyingTime = 1
	w.grid[3*8+3] = blockGround
	Step(w, []Command{TerminalKill(1)})
	for range 4 {
		Step(w, nil)
	}
	if len(w.entities) != 0 || len(w.sacks) != 1 || w.sacks[0].X != 2 || w.sacks[0].Y != 2 || w.sacks[0].Gold != 23 || w.savedObjects.item(10).Retired {
		t.Fatalf("flying body removal lost accessible loot: %+v", w.sacks)
	}
	reloadOperations(t, w)
}

func TestTerminalLootSearchExpandsBeyondAdjacentRing(t *testing.T) {
	w := terminalLootFixture(t, "native")
	for y := 2; y <= 4; y++ {
		for x := 2; x <= 4; x++ {
			w.grid[y*8+x] = blockGround
		}
	}
	if !w.dropTerminalLoot(0) || len(w.sacks) != 1 || w.sacks[0].X != 1 || w.sacks[0].Y != 1 {
		t.Fatalf("search did not expand in stable distance/Y/X order: %+v", w.sacks)
	}
}

func TestTerminalLootSearchClipsRectangleEdges(t *testing.T) {
	for _, test := range []struct {
		bounds         Bounds
		origin, target CellPoint
	}{
		{Bounds{1, 9}, CellPoint{X: 0, Y: 4}, CellPoint{X: 0, Y: 0}},
		{Bounds{9, 1}, CellPoint{X: 4, Y: 0}, CellPoint{X: 0, Y: 0}},
		{Bounds{3, 9}, CellPoint{X: 1, Y: 8}, CellPoint{X: 2, Y: 0}},
		{Bounds{9, 3}, CellPoint{X: 8, Y: 1}, CellPoint{X: 0, Y: 2}},
	} {
		w, err := NewStockedWorld(1, test.bounds, ModeCanonical, Terrain{},
			[]Entity{{ID: 1, X: test.origin.X, Y: test.origin.Y}}, nil, Relations{}, nil,
			[]Stock{{ID: 1, Items: []uint16{0xe01}}})
		if err != nil {
			t.Fatal(err)
		}
		for i := range w.grid {
			w.grid[i] = blockGround
		}
		w.grid[test.target.Y*test.bounds.Width+test.target.X] = 0
		if !w.dropTerminalLoot(0) || len(w.sacks) != 1 || w.sacks[0].X != test.target.X || w.sacks[0].Y != test.target.Y {
			t.Fatalf("search missed only ground cell in %v: %+v", test.bounds, w.sacks)
		}
	}
}

func TestTerminalLootNoGroundRetainsEveryOwnerAndRandomState(t *testing.T) {
	for _, route := range []string{"native", "source", "native objects", "source objects"} {
		t.Run(route, func(t *testing.T) {
			w := terminalLootFixture(t, route)
			for i := range w.grid {
				w.grid[i] = blockGround
			}
			before := w.Hash()
			if w.dropTerminalLoot(0) || w.Hash() != before {
				t.Fatal("no free ground mutated items, gold draw, object owners or body")
			}
			w.grid[6*8+7] = 0
			if !w.dropTerminalLoot(0) || len(w.sacks) != 1 || w.sacks[0].X != 7 || w.sacks[0].Y != 6 || w.sacks[0].Gold != 23 {
				t.Fatalf("retry lost loot or failed to search the remaining map: %+v", w.sacks)
			}
		})
	}
}

func TestTerminalLootBlockedDeathRetriesAfterColdContinuation(t *testing.T) {
	w := terminalLootFixture(t, "native objects")
	for i := range w.grid {
		w.grid[i] = blockGround
	}
	Step(w, []Command{TerminalKill(1)})
	if len(w.sacks) != 0 || len(w.carried[0]) != 1 || w.entities[0].Decay != DecayFallen {
		t.Fatal("failed placement passed the once-only death boundary")
	}
	w = reloadOperations(t, w)
	w.grid[2*8+2] = 0
	Step(w, nil)
	if len(w.sacks) != 1 || w.sacks[0].X != 2 || w.sacks[0].Y != 2 || w.sacks[0].Gold != 23 || w.sacks[0].ItemInstances[0].ObjectID != 10 {
		t.Fatalf("resumed terminal drop lost placement, gold or object identity: %+v", w.sacks)
	}
	before, rng := w.Sacks(), w.rng.state
	cold := reloadOperations(t, w)
	for range 4 {
		Step(w, []Command{TerminalKill(1)})
		Step(cold, []Command{TerminalKill(1)})
	}
	if w.Hash() != cold.Hash() || !reflect.DeepEqual(w.Sacks(), before) || w.rng.state != rng {
		t.Fatal("repeat death or cold next ticks duplicated loot or rerolled gold")
	}
}
