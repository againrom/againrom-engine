package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestReleaseUnorderedAreaCastSparesItsSide(t *testing.T) {
	f := releaseFront(t)
	spells := mapload.SpellRules(f.Table)
	var radius int32
	for _, row := range spells {
		if row.ID == 2 {
			if !row.Area {
				t.Fatal("installed Fire Ball is not area")
			}
			radius = int32(row.Radius)
		}
	}
	if radius < 1 || radius > 5 {
		t.Fatalf("unexpected installed radius %d", radius)
	}
	mage := sim.Entity{ID: 1, Owner: sim.SelfSlot, X: 16, Y: 24, HP: 10000, MaxHP: 10000, Mind: 60, Mana: 500, MaxMana: 500, KnownSpells: 1 << 2, AutoSpell: 2, ScanRange: 19, AttackCharge: 1, AttackRelax: 1}
	near := sim.Entity{ID: 2, Owner: 2, X: 17, Y: 24, HP: 10000, MaxHP: 10000}
	far := near
	far.ID, far.X = 3, 20+radius
	var rel sim.Relations
	rel.Set(sim.SelfSlot, 2, 1)
	rel.Set(2, sim.SelfSlot, 2)
	w, err := sim.NewStructuredWorld(5, sim.Bounds{Width: 48, Height: 48}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{mage, near, far}, nil, rel, nil, nil, spells, sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	hit := false
	for range 160 {
		sim.Step(w, nil)
		actors := w.Entities()
		if actors[0].HP != mage.HP || actors[1].HP != near.HP {
			t.Fatalf("installed area hit protected side or unsafe target: mage=%d near=%d far=%d hash=%x", actors[0].HP, actors[1].HP, actors[2].HP, w.Hash())
		}
		if actors[2].HP < far.HP {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatal("installed armed Fire Ball never hit safe farther hostile")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold sim.World
	if err = cold.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	for range 32 {
		sim.Step(w, nil)
		sim.Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("area target selection diverged after native resume")
		}
	}
	t.Logf("installed Fire Ball radius=%d; protected HP=%d; farther HP=%d; native resume 32 ticks; World=%x", radius, w.Entities()[0].HP, w.Entities()[2].HP, w.Hash())
}
