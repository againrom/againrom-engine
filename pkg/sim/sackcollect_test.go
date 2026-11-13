package sim

import (
	"slices"
	"testing"
)

func collectWorld(t *testing.T, sacks []Sack) *World {
	t.Helper()
	w := secondWorld(t, 1, Bounds{Width: 24, Height: 24}, nil,
		Entity{ID: 7, X: 10, Y: 10, TypeID: HumanTypeID, Humanoid: true, Owner: SelfSlot, HP: 100, MaxHP: 100, ScanRange: 3})
	if err := w.ReplaceGroundSacks(sacks); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestCollectChoosesTheClaimsLastCellOfTheFirstSuccessfulStep(t *testing.T) {
	for _, tc := range []struct {
		name  string
		sacks []Sack
		want  CellPoint
	}{
		{"centre first", []Sack{{X: 10, Y: 10, Gold: 1}, {X: 9, Y: 11, Gold: 2}}, CellPoint{10, 10}},
		{"corner step last", []Sack{{X: 9, Y: 11, Gold: 1}, {X: 9, Y: 9, Gold: 2}}, CellPoint{9, 9}},
		{"four sides last", []Sack{{X: 10, Y: 11, Gold: 1}, {X: 10, Y: 9, Gold: 2}, {X: 11, Y: 10, Gold: 3}, {X: 9, Y: 10, Gold: 4}}, CellPoint{9, 10}},
		{"first step before later step", []Sack{{X: 9, Y: 11, Gold: 1}, {X: 9, Y: 10, Gold: 2}}, CellPoint{9, 11}},
		{"inclusive radius", []Sack{{X: 7, Y: 13, Gold: 1}}, CellPoint{7, 13}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := collectWorld(t, tc.sacks)
			before := w.Hash()
			order, ok := w.OrderSackCollect(7)
			if !ok || order.Kind != KindPickUp || order.X != tc.want.X || order.Y != tc.want.Y {
				t.Fatalf("collect %+v/%v, want pickup %+v", order, ok, tc.want)
			}
			if w.Hash() != before {
				t.Fatal("planning changed the World before command admission")
			}
		})
	}
}

func TestCollectRefusesAMemberWhoseSearchFindsNoSack(t *testing.T) {
	w := collectWorld(t, []Sack{{X: 14, Y: 10, Gold: 17}})
	before := w.Hash()
	if order, ok := w.OrderSackCollect(7); ok {
		t.Fatalf("outside radius answered %+v", order)
	}
	if w.Hash() != before {
		t.Fatal("an empty search changed the World")
	}
	w.entities[0].ScanRange = 0
	if err := w.ReplaceGroundSacks([]Sack{{X: 11, Y: 10, Gold: 1}}); err != nil {
		t.Fatal(err)
	}
	if c, ok := w.OrderSackCollect(7); ok {
		t.Fatalf("zero radius sought %+v", c)
	}
}

func TestCollectExcludesNonHeroForeignDeadAndOffMapMembers(t *testing.T) {
	for _, which := range []string{"non-hero", "foreign", "dead", "off-map", "missing", "first game"} {
		t.Run(which, func(t *testing.T) {
			w := collectWorld(t, []Sack{{X: 10, Y: 10, Gold: 17}})
			id := EntityID(7)
			switch which {
			case "non-hero":
				w.entities[0].TypeID = 1
			case "foreign":
				w.entities[0].Owner = 2
			case "dead":
				w.entities[0].HP = 0
			case "off-map":
				w.entities[0].OffMap = true
			case "missing":
				id = 99
			case "first game":
				w.game = newGameProfile(ScriptROM1)
			}
			if c, ok := w.OrderSackCollect(id); ok {
				t.Fatalf("excluded %s received %+v", which, c)
			}
		})
	}
}

// collectDrive steps w as the map driver does, queueing each next collect
// order; at may replace tick k's commands. It answers the cells taken.
func collectDrive(w *World, first Command, n int, docs SackDocuments, at func(k int, cmds []Command) []Command) []CellPoint {
	var taken []CellPoint
	cmds := []Command{first}
	for k := 0; k < n; k++ {
		if at != nil {
			cmds = at(k, cmds)
		}
		Step(w, cmds)
		var got []CellPoint
		got, cmds = settleOnce(w, docs)
		taken = append(taken, got...)
	}
	return taken
}

func TestCollectRepeatsTheSearchFromEachTakenSackUntilItFails(t *testing.T) {
	// Range 3: each sack is three cells past the last take, the fourth four.
	w := collectWorld(t, []Sack{{X: 12, Y: 10, Gold: 1}, {X: 15, Y: 10, Gold: 2}, {X: 18, Y: 10, Gold: 4}, {X: 22, Y: 10, Gold: 8}})
	order, ok := w.OrderSackCollect(7)
	if !ok || order.Kind != KindPickUp || order.X != 12 {
		t.Fatalf("first search %+v/%v", order, ok)
	}
	taken := collectDrive(w, order, 240, SackDocuments{}, nil)
	want := []CellPoint{{12, 10}, {15, 10}, {18, 10}}
	if !slices.Equal(taken, want) {
		t.Fatalf("collect took %v, want %v", taken, want)
	}
	e, _ := w.Entity(7)
	sacks := w.Sacks()
	if len(sacks) != 1 || sacks[0].X != 22 || w.Purse(SelfSlot) != 7 || e.X != 18 || e.HasTarget || e.PendingOrder.Kind != PendingNone || e.ActorState != actorStateAcquire {
		t.Fatalf("after the failed search: %+v sacks %v purse %d, want Stand Ground at (18,10)", e, sacks, w.Purse(SelfSlot))
	}
}

func TestCollectEndsOnARefusedTakeAndSearchesAgainPastAVanishedSack(t *testing.T) {
	t.Run("refused", func(t *testing.T) {
		w := collectWorld(t, []Sack{{X: 12, Y: 10, Items: []uint16{5}}, {X: 14, Y: 10, Gold: 2}})
		order, _ := w.OrderSackCollect(7)
		var refusals int
		collectDrive(w, order, 240, SackDocuments{Code: 5}, func(k int, cmds []Command) []Command {
			if e, _ := w.Entity(7); e.X == 12 && e.PendingOrder.Kind == PendingNone && len(w.Sacks()) == 2 {
				refusals++
			}
			return cmds
		})
		e, _ := w.Entity(7)
		if len(w.Sacks()) != 2 || e.X != 12 || e.HasTarget || e.PendingOrder.Kind != PendingNone || refusals == 0 {
			t.Fatalf("refused take: %+v sacks %v", e, w.Sacks())
		}
	})
	t.Run("vanished", func(t *testing.T) {
		w := collectWorld(t, []Sack{{X: 12, Y: 10, Gold: 1}, {X: 14, Y: 10, Gold: 2}})
		order, _ := w.OrderSackCollect(7)
		taken := collectDrive(w, order, 240, SackDocuments{}, func(k int, cmds []Command) []Command {
			if k == 1 {
				if err := w.ReplaceGroundSacks([]Sack{{X: 14, Y: 10, Gold: 2}}); err != nil {
					t.Fatal(err)
				}
			}
			return cmds
		})
		e, _ := w.Entity(7)
		if !slices.Equal(taken, []CellPoint{{14, 10}}) || len(w.Sacks()) != 0 || w.Purse(SelfSlot) != 2 || e.X != 14 {
			t.Fatalf("vanished sack: took %v, %+v sacks %v", taken, e, w.Sacks())
		}
	})
}

func TestCollectStopsAtAnyNewOrder(t *testing.T) {
	w := collectWorld(t, []Sack{{X: 12, Y: 10, Gold: 1}, {X: 15, Y: 10, Gold: 2}})
	order, _ := w.OrderSackCollect(7)
	stopped := false
	taken := collectDrive(w, order, 240, SackDocuments{}, func(k int, cmds []Command) []Command {
		if e, _ := w.Entity(7); !stopped && e.PendingOrder.Collect && e.PendingOrder.X == 15 {
			stopped = true
			return []Command{MoveTo(7, CellPoint{X: 12, Y: 12})}
		}
		return cmds
	})
	e, _ := w.Entity(7)
	if !stopped || !slices.Equal(taken, []CellPoint{{12, 10}}) || len(w.Sacks()) != 1 || e.X != 12 || e.Y != 12 || e.PendingOrder.Kind != PendingNone {
		t.Fatalf("move mid-collect: stopped %v took %v, %+v sacks %v", stopped, taken, e, w.Sacks())
	}
}

func TestCollectMarkSurvivesTheByteForm(t *testing.T) {
	w := collectWorld(t, []Sack{{X: 13, Y: 10, Gold: 1}})
	plain := collectWorld(t, []Sack{{X: 13, Y: 10, Gold: 1}})
	order, _ := w.OrderSackCollect(7)
	Step(w, []Command{order})
	Step(plain, []Command{PickUp(7, CellPoint{X: 13, Y: 10})})
	if e, _ := w.Entity(7); !e.PendingOrder.Collect || w.Hash() == plain.Hash() {
		t.Fatalf("collect pickup %+v, hash equal to a plain pickup's: %v", e.PendingOrder, w.Hash() == plain.Hash())
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if e, _ := back.Entity(7); !e.PendingOrder.Collect || back.Hash() != w.Hash() {
		t.Fatalf("byte form lost the collect mark: %+v", e.PendingOrder)
	}
}
