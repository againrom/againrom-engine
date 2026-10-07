package sim

import (
	"bytes"
	"testing"
)

func TestHeadlessMutationSeamKeepsCanonicalLifecycleAndPlacement(t *testing.T) {
	down := Entity{ID: 1, X: 2, Y: 2, PostX: 2, PostY: 2, HP: 0, MaxHP: 30,
		Defence: 40, DyingTime: 5}
	PrepareAuthoredBody(&down)
	live := Entity{ID: 2, X: 4, Y: 4, PostX: 4, PostY: 4, HP: 20, MaxHP: 20,
		HasTarget: true, TargetX: 7, TargetY: 7}
	w, err := NewWorld(1060, Bounds{Width: 16, Height: 16}, ModeCanonical, nil,
		[]Entity{down, live})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	if err := w.HeadlessHeal(1); err != nil {
		t.Fatalf("HeadlessHeal: %v", err)
	}
	healed := w.Entities()[0]
	if healed.HP != 30 || healed.Decay != DecayNone || healed.Dwell != 0 || healed.Defence != 40 {
		t.Fatalf("healed entity = HP %d decay %d dwell %d defence %d", healed.HP,
			healed.Decay, healed.Dwell, healed.Defence)
	}

	if err := w.HeadlessKill(2); err != nil {
		t.Fatalf("HeadlessKill: %v", err)
	}
	killed := w.Entities()[1]
	if killed.HP != decayBonesHP || killed.OrdinaryTargetable() || killed.Alive() ||
		killed.Dying() || killed.Decay != DecayBones || killed.Dwell != 0 {
		t.Fatalf("killed entity = HP %d targetable=%v alive=%v dying=%v decay=%d dwell=%d",
			killed.HP, killed.OrdinaryTargetable(), killed.Alive(), killed.Dying(),
			killed.Decay, killed.Dwell)
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Fatalf("headless-mutated world hash changed across round trip: %016x -> %016x",
			w.Hash(), back.Hash())
	}
}

func TestHeadlessMutationSeamRefusesInvalidTargets(t *testing.T) {
	w, err := NewWorld(1060, Bounds{Width: 8, Height: 8}, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 2, Y: 2, HP: 10, MaxHP: 10}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	for name, act := range map[string]func() error{
		"missing kill":  func() error { return w.HeadlessKill(99) },
		"missing heal":  func() error { return w.HeadlessHeal(99) },
		"missing place": func() error { return w.HeadlessPlace(99, 1, 1) },
		"outside place": func() error { return w.HeadlessPlace(1, 8, 1) },
	} {
		if err := act(); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if _, err := w.HeadlessKillPlayer(99); err == nil {
		t.Error("missing player was accepted")
	}
}

func TestHeadlessKillPlayerUsesOrdinaryTerminalDeathForEveryOwnedActor(t *testing.T) {
	w, err := NewWorld(1060, Bounds{Width: 8, Height: 8}, ModeCanonical, nil,
		[]Entity{
			{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10, Owner: 3},
			{ID: 2, X: 2, Y: 2, HP: 20, MaxHP: 20, Owner: 3},
			{ID: 3, X: 3, Y: 3, HP: 30, MaxHP: 30, Owner: 4},
		})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	killed, err := w.HeadlessKillPlayer(3)
	if err != nil {
		t.Fatalf("HeadlessKillPlayer: %v", err)
	}
	if killed != 2 {
		t.Fatalf("HeadlessKillPlayer killed %d actors, want 2", killed)
	}
	got := w.Entities()
	if got[0].HP != decayBonesHP || got[1].HP != decayBonesHP || got[2].HP != 30 {
		t.Fatalf("player kill crossed ownership: HPs %d, %d, %d", got[0].HP, got[1].HP, got[2].HP)
	}
	if again, err := w.HeadlessKillPlayer(3); err != nil || again != 0 {
		t.Fatalf("repeated HeadlessKillPlayer = %d, %v; want idempotent zero", again, err)
	}
}

func TestHeadlessKillTerminalizesCanonicalFallenBodyExactlyOnce(t *testing.T) {
	for _, tc := range []struct {
		name string
		act  func(*World) (int, error)
	}{
		{"kill", func(w *World) (int, error) { return 0, w.HeadlessKill(1) }},
		{"kill_player", func(w *World) (int, error) { return w.HeadlessKillPlayer(3) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fallen := Entity{ID: 1, X: 2, Y: 2, PostX: 2, PostY: 2, HP: -1,
				MaxHP: 10, Owner: 3, Decay: DecayFallen, Dwell: 0, Defence: 20}
			w, err := NewStockedWorld(1060, Bounds{Width: 8, Height: 8}, ModeCanonical,
				Terrain{}, []Entity{fallen}, nil, Relations{}, nil,
				[]Stock{{ID: 1, Items: []uint16{7}}})
			if err != nil {
				t.Fatalf("NewStockedWorld: %v", err)
			}

			before := w.Hash()
			count, err := tc.act(w)
			if err != nil {
				t.Fatalf("first action: %v", err)
			}
			if tc.name == "kill_player" && count != 1 {
				t.Fatalf("first kill_player count = %d, want 1", count)
			}
			got := w.Entities()[0]
			if got.HP != decayBonesHP || got.Decay != DecayBones || got.Dwell != 0 ||
				got.OrdinaryTargetable() {
				t.Fatalf("terminal body = HP %d decay %d dwell %d targetable=%v",
					got.HP, got.Decay, got.Dwell, got.OrdinaryTargetable())
			}
			if w.Hash() == before {
				t.Fatal("terminalization left the canonical hash unchanged")
			}
			if sacks := w.Sacks(); len(sacks) != 1 || sacks[0].X != 2 || sacks[0].Y != 2 ||
				!equalCodes(sacks[0].Items, []uint16{7}) {
				t.Fatalf("terminal loot = %+v, want one sack at (2,2) holding [7]", sacks)
			}
			if carried, ok := w.Carried(1); !ok || len(carried) != 0 {
				t.Fatalf("terminal body still carries %v, ok=%v", carried, ok)
			}

			terminalHash := w.Hash()
			terminalForm, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary after first action: %v", err)
			}
			again, err := tc.act(w)
			if err != nil {
				t.Fatalf("repeated action: %v", err)
			}
			if tc.name == "kill_player" && again != 0 {
				t.Fatalf("repeated kill_player count = %d, want 0", again)
			}
			againForm, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary after repeated action: %v", err)
			}
			if w.Hash() != terminalHash || !bytes.Equal(againForm, terminalForm) {
				t.Fatalf("repeated action changed terminal state: hash %016x -> %016x",
					terminalHash, w.Hash())
			}
			if sacks := w.Sacks(); len(sacks) != 1 || !equalCodes(sacks[0].Items, []uint16{7}) {
				t.Fatalf("repeated action duplicated terminal loot: %+v", sacks)
			}
		})
	}
}

// headlessPlacedNear fails unless entity id stands on the map inside the
// placement window around (x, y): x-1 through x+2 on each axis.
func headlessPlacedNear(t *testing.T, w *World, id EntityID, x, y int32) Entity {
	t.Helper()
	e, ok := w.Entity(id)
	if !ok || e.OffMap || e.X < x-1 || e.X > x+2 || e.Y < y-1 || e.Y > y+2 {
		t.Fatalf("entity %d at (%d,%d) off-map=%v, want on the map near (%d,%d)", id, e.X, e.Y, e.OffMap, x, y)
	}
	return e
}

func TestHeadlessPlaceIsTheScriptPlacement(t *testing.T) {
	grid := make([]byte, 16*16)
	grid[8*16+8] = 1
	build := func() *World {
		mover := Entity{ID: 1, X: 2, Y: 2, HP: 20, MaxHP: 20, HasTarget: true, TargetX: 3, TargetY: 2}
		blocker := Entity{ID: 2, X: 6, Y: 6, HP: 20, MaxHP: 20}
		w, err := NewWorld(1060, Bounds{Width: 16, Height: 16}, ModeCanonical, grid, []Entity{mover, blocker})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		return w
	}
	for _, at := range [][2]int32{{6, 6}, {8, 8}} {
		w, twin := build(), build()
		if err := w.HeadlessPlace(1, at[0], at[1]); err != nil {
			t.Fatalf("HeadlessPlace(%v): %v", at, err)
		}
		twin.takeOffMap(0)
		if !twin.placeNear(0, at[0], at[1]) {
			t.Fatalf("script placement near %v failed", at)
		}
		e := headlessPlacedNear(t, w, 1, at[0], at[1])
		if e.X == at[0] && e.Y == at[1] {
			t.Fatalf("entity placed on the occupied or impassable anchor %v", at)
		}
		if !e.HasTarget || e.TargetX != 3 || e.TargetY != 2 {
			t.Fatalf("placement wrote the order: %+v", e)
		}
		if w.Hash() != twin.Hash() {
			t.Fatalf("HeadlessPlace near %v differs from the script's removal and placement", at)
		}
	}
}

func TestHeadlessPlaceWithNoFreeCellReturnsTheActor(t *testing.T) {
	grid := make([]byte, 16*16)
	for y := 5; y <= 8; y++ {
		for x := 5; x <= 8; x++ {
			grid[y*16+x] = 1
		}
	}
	w, err := NewWorld(1060, Bounds{Width: 16, Height: 16}, ModeCanonical, grid,
		[]Entity{{ID: 1, X: 2, Y: 2, HP: 20, MaxHP: 20}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	if err := w.HeadlessPlace(1, 6, 6); err == nil {
		t.Fatal("placement into a closed window was accepted")
	}
	if e, _ := w.Entity(1); e.OffMap || e.X != 2 || e.Y != 2 {
		t.Fatalf("refused placement left the actor at (%d,%d) off-map=%v", e.X, e.Y, e.OffMap)
	}
}
func headlessDeathWorld(t *testing.T, entities ...Entity) *World {
	t.Helper()
	var stock []Stock
	for _, e := range entities {
		stock = append(stock, Stock{ID: e.ID, Items: []uint16{7}})
	}
	w, err := NewStockedWorld(1060, Bounds{Width: 8, Height: 8}, ModeCanonical,
		Terrain{}, entities, nil, Relations{}, nil, stock)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return w
}

func TestHeadlessKillLeavesDwellLootAndRemovalToOrdinarySteps(t *testing.T) {
	ground := Entity{ID: 1, X: 2, Y: 2, PostX: 2, PostY: 2, HP: 10, MaxHP: 10, Owner: 3, DyingTime: 3}
	flier := Entity{ID: 2, X: 4, Y: 4, PostX: 4, PostY: 4, HP: 10, MaxHP: 10, Owner: 3, DyingTime: 3, Domain: DomainAir}
	other := Entity{ID: 3, X: 6, Y: 6, PostX: 6, PostY: 6, HP: 10, MaxHP: 10, Owner: 4, DyingTime: 3}
	for _, tc := range []struct {
		name   string
		killed []EntityID
		kill   func(*World) error
	}{
		{"kill ground", []EntityID{1}, func(w *World) error { return w.HeadlessKill(1) }},
		{"kill flier", []EntityID{2}, func(w *World) error { return w.HeadlessKill(2) }},
		{"kill_player", []EntityID{1, 2}, func(w *World) error {
			if n, err := w.HeadlessKillPlayer(3); err != nil || n != 2 {
				t.Fatalf("HeadlessKillPlayer = %d, %v; want 2", n, err)
			}
			return nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := headlessDeathWorld(t, ground, flier, other)
			if err := tc.kill(w); err != nil {
				t.Fatal(err)
			}
			for _, id := range tc.killed {
				e, _ := w.Entity(id)
				if e.Decay != DecayFallen || e.Dwell != 3 || !e.Dying() || scriptDead(e) {
					t.Fatalf("kill tick: entity %d decay %d dwell %d dying=%v", id, e.Decay, e.Dwell, e.Dying())
				}
			}
			for n := 0; n < 2; n++ {
				Step(w, nil)
			}
			if len(w.Entities()) != 3 || len(w.Sacks()) != 0 {
				t.Fatalf("dwell ended early: %d entities, sacks %+v", len(w.Entities()), w.Sacks())
			}
			Step(w, nil)
			for _, id := range tc.killed {
				e, ok := w.Entity(id)
				if id == flier.ID {
					if ok {
						t.Fatalf("flier kept its body after the dwell: %+v", e)
					}
					continue
				}
				if !ok || e.Decay != DecayBones || e.Dwell != 0 || !scriptDead(e) {
					t.Fatalf("entity %d after the dwell: held=%v decay %d dwell %d", id, ok, e.Decay, e.Dwell)
				}
			}
			if e, _ := w.Entity(other.ID); !e.Alive() || e.HP != 10 {
				t.Fatalf("kill crossed ownership: %+v", e)
			}
			if len(w.Sacks()) != len(tc.killed) {
				t.Fatalf("terminal loot = %+v; want one sack per killed body", w.Sacks())
			}
		})
	}
}

func TestHeadlessHealIsTheScriptHealthWrite(t *testing.T) {
	down := Entity{ID: 1, X: 2, Y: 2, PostX: 2, PostY: 2, HP: -4, MaxHP: 30,
		Defence: 40, DyingTime: 5}
	PrepareAuthoredBody(&down)
	build := func() *World {
		w, err := NewWorld(1060, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, []Entity{down})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		return w
	}
	seam, script := build(), build()
	if err := seam.HeadlessHeal(1); err != nil {
		t.Fatalf("HeadlessHeal: %v", err)
	}
	script.setUnitProperty(0, propertyHealth, 30)
	if seam.Hash() != script.Hash() || seam.Entities()[0] != script.Entities()[0] {
		t.Fatalf("HeadlessHeal %+v differs from the script write %+v",
			seam.Entities()[0], script.Entities()[0])
	}
}
