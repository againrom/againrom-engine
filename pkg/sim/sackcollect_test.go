package sim

import "testing"

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

func TestCollectRadiusAndEmptySearchUseTheExistingStandGroundOrder(t *testing.T) {
	w := collectWorld(t, []Sack{{X: 14, Y: 10, Gold: 17}})
	order, ok := w.OrderSackCollect(7)
	if !ok || order.Kind != KindGroupStance || order.X != OrderStandGround {
		t.Fatalf("outside radius: %+v/%v, want Stand Ground", order, ok)
	}
	Step(w, []Command{order})
	e, _ := w.Entity(7)
	if e.HasTarget || e.ActorState != actorStateAcquire || len(w.Sacks()) != 1 {
		t.Fatalf("no-cell answer: %+v sacks %v", e, w.Sacks())
	}
	w.entities[0].ScanRange = 0
	if err := w.ReplaceGroundSacks([]Sack{{X: 11, Y: 10, Gold: 1}}); err != nil {
		t.Fatal(err)
	}
	if c, _ := w.OrderSackCollect(7); c.Kind != KindGroupStance {
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

func TestCollectUsesOnePendingPickupAndDoesNotRepeatTheSearch(t *testing.T) {
	for _, at := range []CellPoint{{10, 10}, {12, 10}} {
		w := collectWorld(t, []Sack{{X: at.X, Y: at.Y, Gold: 17}, {X: 13, Y: 10, Gold: 3}})
		order, ok := w.OrderSackCollect(7)
		if !ok {
			t.Fatal("collect refused")
		}
		for n := 0; n < 120; n++ {
			var cmds []Command
			if n == 0 {
				cmds = []Command{order}
			}
			Step(w, cmds)
			w.SettleSackPickup(SackDocuments{})
		}
		e, _ := w.Entity(7)
		if len(w.Sacks()) != 1 || w.Purse(SelfSlot) != 17 || e.X != at.X || e.Y != at.Y || e.PendingOrder.Kind != PendingNone {
			t.Fatalf("take at %+v: entity %+v sacks %v purse %d", at, e, w.Sacks(), w.Purse(SelfSlot))
		}
	}
}
