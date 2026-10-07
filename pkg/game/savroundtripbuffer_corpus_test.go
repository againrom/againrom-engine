//go:build sessioncorpusaudit

package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestSAVRoundTripGateComparisonStorage(t *testing.T) {
	world := func(count int) *sim.World {
		var actors []sim.Entity
		for i := range count {
			actors = append(actors, sim.Entity{ID: sim.EntityID(i + 1), X: int32(i), HP: 10, MaxHP: 10})
		}
		w, err := sim.NewWorld(17, sim.Bounds{Width: 4, Height: 4}, sim.ModeCanonical, make([]byte, 16), actors)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	var comparison savGateWorldComparison
	a, b := world(2), world(2)
	for tick := 0; tick < savGateAdvance; tick++ {
		sim.Step(a, nil)
		sim.Step(b, nil)
		if tick == savGateAdvance-1 {
			if err := b.HeadlessDamage(1, 1); err != nil {
				t.Fatal(err)
			}
		}
		if got := comparison.equal(a, b); got != (tick != savGateAdvance-1) {
			t.Fatalf("comparison at tick %d concealed a late mutation: %t", tick, got)
		}
	}
	a, b = world(2), world(1)
	if comparison.equal(a, b) {
		t.Fatal("actor removal concealed by reused buffers")
	}
	a, b = world(1), world(1)
	if !a.ImportAutoHealing(0, 50) || !b.ImportAutoHealing(0, 50) || !comparison.equal(a, b) {
		t.Fatal("equal populated suffixes differ")
	}
	b = world(1)
	if comparison.equal(a, b) {
		t.Fatal("absent suffix reused its previous bytes")
	}
	if !b.ImportAutoHealing(0, 100) || comparison.equal(a, b) {
		t.Fatal("changed suffix concealed by reused buffers")
	}
	if !b.ImportAutoHealing(0, 50) {
		t.Fatal("cannot restore matching suffix")
	}
	for i := range comparison.a {
		comparison.a[i] = 0xa5
	}
	for i := range comparison.b {
		comparison.b[i] = 0x5a
	}
	if !comparison.equal(a, b) {
		t.Fatal("poisoned capacity changed equal canonical bytes")
	}
}
