package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestReleaseDistantSpellDamageAcquiresCaster(t *testing.T) {
	f := releaseFront(t)
	caster := sim.Entity{ID: 1, Owner: sim.SelfSlot, X: 16, Y: 24, HP: 1000, MaxHP: 1000,
		Mind: 60, Mana: 200, MaxMana: 200, KnownSpells: 1 << 1, ScanRange: 19}
	target := sim.Entity{ID: 2, Owner: 2, X: 24, Y: 24, HP: 10000, MaxHP: 10000, ScanRange: 1}
	var relations sim.Relations
	relations.Set(1, 1, 2)
	relations.Set(2, 2, 2)
	w, err := sim.NewStructuredWorld(1, sim.Bounds{Width: 48, Height: 48}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{caster, target}, nil, relations, nil, nil, mapload.SpellRules(f.Table), sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Actual installed arrow rule, ordinary cast command and tick processing.
	sim.Step(w, []sim.Command{{Kind: sim.KindCast, Entity: 1, X: 2, Y: 1}})
	hit := false
	for range 128 {
		sim.Step(w, nil)
		if w.Entities()[1].HP < target.HP {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatal("installed spell did not hit distant target")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold sim.World
	if err := cold.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	acquired := false
	for range 32 {
		sim.Step(w, nil)
		sim.Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("spell reaction changed across native resume")
		}
		e := w.Entities()[1]
		acquired = acquired || e.HasAttackTarget && e.AttackTarget == caster.ID
	}
	if !acquired {
		t.Fatal("hit victim never acquired the distant caster")
	}
	t.Log("installed arrow hit beyond target sight; target acquired caster; native resume matched for 32 ticks")
}
