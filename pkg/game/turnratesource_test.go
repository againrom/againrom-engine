package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestNativeRearmTurnRateUsesDerivedSpeed(t *testing.T) {
	const id sim.EntityID = 7
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: id, X: 3, Y: 3, HP: 30, MaxHP: 30, Humanoid: true, RotationSpeed: 99}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := Rearm(w, id, data.Hero{Body: 20, Reaction: 45, Mind: 20, Spirit: 20}, data.Profile{}, nil, false, nil, 99); !ok {
		t.Fatal("Rearm refused")
	}
	e, _ := w.Entity(id)
	if e.Speed != 21 || e.RotationSpeed != 21 {
		t.Fatalf("rearmed speed/turn rate = %d/%d, want 21/21", e.Speed, e.RotationSpeed)
	}
}
