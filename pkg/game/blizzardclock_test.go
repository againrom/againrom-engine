package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestSecondBlizzardWatchTracksTheDeferredFirstPulse(t *testing.T) {
	rows := []sim.SpellRule{{ID: 7, School: 2, MaxRange: 10, Area: true, Distribution: 5, Radius: 4,
		AreaDuration: 8, DamageMin: 10, DamageMax: 30, Damaging: true}}
	sim.AssignSecondGameArms(rows)
	caster := sim.Entity{ID: 1, X: 20, Y: 12, HP: 100, MaxHP: 100, Owner: 3, TokenSize: 1,
		Mind: 30, Mana: 500, MaxMana: 500, KnownSpells: 1 << 7, AttackCharge: 4, AttackRelax: 2}
	w, err := sim.NewSpelledWorld(7, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, []sim.Entity{caster}, nil, rows)
	if err != nil {
		t.Fatal(err)
	}
	watch := &secondBlizzardWatch{}
	commands := []sim.Command{sim.CastAt(1, 7, sim.CellPoint{X: 20, Y: 20})}
	for range 150 {
		report := sim.StepReported(w, commands)
		commands = nil
		watch.observe(t, w, report)
	}
	if len(watch.areas) != 1 || watch.areas[0].Done-watch.areas[0].Landing != 94 || len(watch.pulses) != 32 {
		t.Fatalf("areas %+v, pulses %d", watch.areas, len(watch.pulses))
	}
	if watch.pulses[0].Tick != watch.areas[0].Landing+1 {
		t.Fatal("first pulse was not next tick")
	}
}
