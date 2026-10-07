package sim

import "testing"

var dropGoldBounds = Bounds{Width: 64, Height: 64}

func dropGoldWorld(t *testing.T, purse uint32, sacks []Sack, ents ...Entity) *World {
	t.Helper()
	w, err := NewLootWorld(1, dropGoldBounds, ModeCanonical, Terrain{}, ents, nil, Relations{}, sacks)
	if err != nil {
		t.Fatal(err)
	}
	if !w.SetPurse(SelfSlot, purse) {
		t.Fatal("SetPurse refused the primary slot")
	}
	return w
}

func dropGoldSack(t *testing.T, w *World, x, y int32) (Sack, bool) {
	t.Helper()
	for _, s := range w.Sacks() {
		if s.X == x && s.Y == y {
			return s, true
		}
	}
	return Sack{}, false
}

// The command debits the addressed purse and plants the amount as one Sack at
// the requested cell. The wallet crosses 16 bits, so a narrowed amount would
// leave a different balance.
func TestPlayerDropGoldDebitsFullWidthAndPlantsASack(t *testing.T) {
	w := dropGoldWorld(t, 70000, nil, Entity{ID: 1, X: 10, Y: 20, Owner: SelfSlot})
	Step(w, []Command{DropGold(SelfSlot, 65536, CellPoint{X: 12, Y: 22})})
	if got := w.Purse(SelfSlot); got != 4464 {
		t.Fatalf("purse %d, want 4464", got)
	}
	s, ok := dropGoldSack(t, w, 12, 22)
	if !ok || s.Gold != 65536 || len(s.Items) != 0 {
		t.Fatalf("ground at (12,22) = %+v, %v", s, ok)
	}
}

// Distance is two independent per-axis tests from the first actor, and a
// request that fails either falls back to the first actor's own cell.
func TestPlayerDropGoldCellWindowIsPerAxisAroundTheFirstActor(t *testing.T) {
	for _, c := range []struct {
		name     string
		at       CellPoint
		wantX, y int32
	}{
		{"corner of the window", CellPoint{X: 12, Y: 22}, 12, 22},
		{"one axis too far", CellPoint{X: 13, Y: 20}, 10, 20},
		{"off the map", CellPoint{X: 1 << 20, Y: 1 << 20}, 10, 20},
		{"negative", CellPoint{X: -1, Y: 19}, 10, 20},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := dropGoldWorld(t, 500, nil, Entity{ID: 1, X: 10, Y: 20, Owner: SelfSlot})
			Step(w, []Command{DropGold(SelfSlot, 100, c.at)})
			if s, ok := dropGoldSack(t, w, c.wantX, c.y); !ok || s.Gold != 100 {
				t.Fatalf("no 100 gold at (%d,%d): %+v", c.wantX, c.y, w.Sacks())
			}
			if w.Purse(SelfSlot) != 400 {
				t.Fatalf("purse %d, want 400", w.Purse(SelfSlot))
			}
		})
	}
}

// The addressed Player's first actor takes the gold, never the lowest entity
// id or an actor of another slot, and the cell window is read around it.
func TestPlayerDropGoldUsesTheAddressedPlayersFirstActor(t *testing.T) {
	w := dropGoldWorld(t, 500, nil,
		Entity{ID: 1, X: 40, Y: 40, Owner: 2},
		Entity{ID: 2, X: 10, Y: 20, Owner: SelfSlot},
		Entity{ID: 3, X: 30, Y: 40, Owner: SelfSlot})
	Step(w, []Command{DropGold(SelfSlot, 100, CellPoint{X: 30, Y: 40})})
	if _, ok := dropGoldSack(t, w, 10, 20); !ok {
		t.Fatalf("the sack is not at the first actor's cell: %+v", w.Sacks())
	}
	if _, ok := dropGoldSack(t, w, 30, 40); ok {
		t.Fatal("the request cell outside the window was kept")
	}
}

// A refused request costs nothing: zero, an unaffordable amount, an absent
// first actor, a slot past the roster and an actor off the map all leave the
// purse and the ground untouched.
func TestPlayerDropGoldRefusalsChangeNothing(t *testing.T) {
	for _, c := range []struct {
		name   string
		player uint32
		amount uint32
		ents   []Entity
	}{
		{"zero amount", SelfSlot, 0, []Entity{{ID: 1, X: 5, Y: 5, Owner: SelfSlot}}},
		{"more than the purse", SelfSlot, 501, []Entity{{ID: 1, X: 5, Y: 5, Owner: SelfSlot}}},
		{"no actor", SelfSlot, 10, []Entity{{ID: 1, X: 5, Y: 5, Owner: 2}}},
		{"slot past the roster", 50, 10, []Entity{{ID: 1, X: 5, Y: 5, Owner: SelfSlot}}},
		{"dead actor", SelfSlot, 10, []Entity{{ID: 1, X: 5, Y: 5, Owner: SelfSlot, HP: -5, MaxHP: 10}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := dropGoldWorld(t, 500, nil, c.ents...)
			Step(w, []Command{DropGold(c.player, c.amount, CellPoint{X: 5, Y: 5})})
			if w.Purse(SelfSlot) != 500 || len(w.Sacks()) != 0 {
				t.Fatalf("purse %d, ground %+v", w.Purse(SelfSlot), w.Sacks())
			}
		})
	}
}

// Gold merges into the Sack already on the cell and leaves its items alone,
// and the same request applied twice debits twice only while the purse covers
// it.
func TestPlayerDropGoldMergesIntoAnExistingSackAndRechecksAffordability(t *testing.T) {
	sack := Sack{X: 12, Y: 22, Gold: 23, Items: []uint16{0x101}}
	w := dropGoldWorld(t, 1500, []Sack{sack}, Entity{ID: 1, X: 10, Y: 20, Owner: SelfSlot})
	drop := DropGold(SelfSlot, 700, CellPoint{X: 12, Y: 22})
	Step(w, []Command{drop})
	s, ok := dropGoldSack(t, w, 12, 22)
	if !ok || s.Gold != 723 || !equalCodes(s.Items, []uint16{0x101}) || w.Purse(SelfSlot) != 800 {
		t.Fatalf("after one request: sack %+v purse %d", s, w.Purse(SelfSlot))
	}
	Step(w, []Command{drop})
	Step(w, []Command{drop})
	s, _ = dropGoldSack(t, w, 12, 22)
	if s.Gold != 1423 || w.Purse(SelfSlot) != 100 {
		t.Fatalf("after three requests: sack gold %d purse %d, want 1423 and 100", s.Gold, w.Purse(SelfSlot))
	}
}

// The first actor of a slot named by two Player containers is ambiguous and
// the request is refused rather than guessed.
func TestPlayerFirstActorFollowsPlayerContainerOrderAndRefusesAmbiguity(t *testing.T) {
	w := dropGoldWorld(t, 500, nil,
		Entity{ID: 1, X: 10, Y: 20, Owner: SelfSlot},
		Entity{ID: 2, X: 30, Y: 40, Owner: SelfSlot})
	w.savedGroups = &savedGroupState{
		PlayersPresent: true,
		Players:        []SavedGroupPlayer{{ID: 7, Slot: SelfSlot}},
		Groups: []SavedGroup{
			{ID: 1, ContainerID: 7, Members: []SavedGroupMember{{Entity: 2, Bound: true}, {Entity: 1, Bound: true}}},
		},
		HighWater: 1,
	}
	if i := w.playerFirstActor(SelfSlot); i < 0 || w.entities[i].ID != 2 {
		t.Fatalf("first actor index %d, want the container's first member (entity 2)", i)
	}
	w.savedGroups.Players = append(w.savedGroups.Players, SavedGroupPlayer{ID: 8, Slot: SelfSlot})
	if i := w.playerFirstActor(SelfSlot); i >= 0 {
		t.Fatalf("two containers for one slot resolved to entity %d", w.entities[i].ID)
	}
	Step(w, []Command{DropGold(SelfSlot, 100, CellPoint{X: 10, Y: 20})})
	if w.Purse(SelfSlot) != 500 || len(w.Sacks()) != 0 {
		t.Fatalf("ambiguous binding changed state: purse %d ground %+v", w.Purse(SelfSlot), w.Sacks())
	}
}

// The new Sack then travels through the ordinary pickup: the taker's purse
// rises by exactly the gold and the ground empties.
func TestPlayerDropGoldIsRecoveredByTakeSack(t *testing.T) {
	w := dropGoldWorld(t, 2500, nil, Entity{ID: 1, X: 10, Y: 20, Owner: SelfSlot})
	Step(w, []Command{DropGold(SelfSlot, 700, CellPoint{X: 10, Y: 20})})
	if w.Purse(SelfSlot) != 1800 {
		t.Fatalf("purse %d, want 1800", w.Purse(SelfSlot))
	}
	if err := w.TakeSack(1, 10, 20); err != nil {
		t.Fatal(err)
	}
	if w.Purse(SelfSlot) != 2500 || len(w.Sacks()) != 0 {
		t.Fatalf("purse %d ground %+v, want 2500 and empty", w.Purse(SelfSlot), w.Sacks())
	}
	if err := w.TakeSack(1, 10, 20); err == nil || w.Purse(SelfSlot) != 2500 {
		t.Fatal("a second pickup changed the purse or succeeded")
	}
}

// Over a world that owns saved objects the gold becomes a generated Sack root
// with its own value slot, merges on a repeated drop without a second root,
// survives a byte-form round trip, and is retired by the ordinary pickup.
func TestPlayerDropGoldOverSavedObjectsKeepsRootValueAndRetirement(t *testing.T) {
	w := goldPickupSavedWorld(t, 31)
	w.SetPurse(1, 2500)
	roots := len(w.SavedObjects().Sacks)
	Step(w, []Command{DropGold(1, 700, CellPoint{X: 2, Y: 2})})
	if w.Purse(1) != 1800 {
		t.Fatalf("purse %d, want 1800", w.Purse(1))
	}
	s, ok := dropGoldSack(t, w, 2, 2)
	if !ok || s.Gold != 700 || s.ObjectID == 0 {
		t.Fatalf("ground at (2,2) = %+v, %v", s, ok)
	}
	reg := w.SavedObjects()
	if len(reg.Sacks) != roots+1 {
		t.Fatalf("saved Sack rows %d, want %d", len(reg.Sacks), roots+1)
	}
	row := reg.Sacks[len(reg.Sacks)-1]
	if row.ID != s.ObjectID || row.Gold != 700 || row.Token.T1C != 700 || row.Retired || row.Origin.Kind != SavedObjectGenerated {
		t.Fatalf("saved Sack row %+v does not carry the dropped gold", row)
	}
	Step(w, []Command{DropGold(1, 700, CellPoint{X: 2, Y: 2})})
	s, _ = dropGoldSack(t, w, 2, 2)
	reg = w.SavedObjects()
	if w.Purse(1) != 1100 || s.Gold != 1400 || len(reg.Sacks) != roots+1 || reg.Sacks[len(reg.Sacks)-1].Gold != 1400 || reg.Sacks[len(reg.Sacks)-1].Token.T1C != 1400 {
		t.Fatalf("repeated drop: purse %d sack %+v rows %d", w.Purse(1), s, len(reg.Sacks))
	}
	cold := copySackCellSavedWorld(t, w)
	if cold.Hash() != w.Hash() || cold.Purse(1) != 1100 {
		t.Fatal("byte-form round trip changed the dropped gold")
	}
	if err := cold.TakeSack(7, 2, 2); err != nil {
		t.Fatal(err)
	}
	reg = cold.SavedObjects()
	if cold.Purse(1) != 2500 || !reg.Sacks[len(reg.Sacks)-1].Retired {
		t.Fatalf("pickup left purse %d, row %+v", cold.Purse(1), reg.Sacks[len(reg.Sacks)-1])
	}
	if err := cold.TakeSack(7, 2, 2); err == nil || cold.Purse(1) != 2500 {
		t.Fatal("repeated pickup changed the purse")
	}
}
