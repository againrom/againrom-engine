package sim

import "testing"

// The river separates two open banks. Ranged actors can shoot across it, then
// keep pursuing a victim that withdraws beyond their reach on the other bank.
// This measures repeated unsuccessful routes, without an installed map or save.
func riverPursuitWorld(t testing.TB) *World {
	t.Helper()
	bounds := Bounds{Width: 64, Height: 48}
	grid := make([]byte, bounds.Width*bounds.Height)
	for y := int32(0); y < bounds.Height; y++ {
		grid[y*bounds.Width+30] = blockGround
		grid[y*bounds.Width+31] = blockGround
	}
	const victim = EntityID(17)
	var actors []Entity
	var commands []Command
	for id := EntityID(1); id < victim; id++ {
		actors = append(actors, Entity{ID: id, X: 26 + int32((id-1)%4), Y: 20 + int32((id-1)/4),
			HP: 100, MaxHP: 100, Reach: 8, AttackCharge: 1, DamageBase: 1, AlwaysHits: true})
		commands = append(commands, Command{Kind: KindAttack, Entity: id, X: int32(victim)})
	}
	actors = append(actors, Entity{ID: victim, X: 33, Y: 22, HP: 100000, MaxHP: 100000})
	w, err := NewWorld(7, bounds, ModeCanonical, grid, actors)
	if err != nil {
		t.Fatal(err)
	}
	Step(w, commands)
	for i := 0; i < 8; i++ {
		Step(w, nil)
	}
	if w.entities[len(w.entities)-1].HP == 100000 {
		t.Fatal("fixture never fired across the river")
	}
	Step(w, []Command{{Kind: KindMoveTo, Entity: victim, X: 44, Y: 22}})
	for i := 0; i < 64; i++ {
		Step(w, nil)
	}
	return w
}

func TestRiverPursuitRetainsBlockedAttackers(t *testing.T) {
	w := riverPursuitWorld(t)
	for _, e := range w.Entities() {
		if e.ID == 17 {
			if e.X != 44 || e.Y != 22 || e.HP <= 0 {
				t.Fatalf("victim did not survive its retreat: %+v", e)
			}
		} else if e.X >= 30 || !e.HasAttackTarget || e.AttackTarget != 17 {
			t.Fatalf("attacker crossed the river or lost its retained order: %+v", e)
		}
	}
}

func BenchmarkRiverPursuit(b *testing.B) {
	w := riverPursuitWorld(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Step(w, nil)
	}
}
