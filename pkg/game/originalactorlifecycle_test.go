package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestOriginalActorAdmissionUsesSavedLifecycle(t *testing.T) {
	for _, dying := range []bool{false, true} {
		name := "saved alive over dead placement"
		if dying {
			name = "saved dying over live placement"
		}
		t.Run(name, func(t *testing.T) {
			file, err := sav.Open(poolFixtureSave(actorBookFixture(91)))
			if err != nil {
				t.Fatal(err)
			}
			graph, err := file.ActorGraph()
			if err != nil || len(graph.Actors) != 1 {
				t.Fatal("actor fixture", graph, err)
			}
			a := graph.Actors[0]
			template := sim.Entity{ID: 1, X: 5, Y: 6, HP: 0, MaxHP: 31, Decay: sim.DecayFallen, Dwell: 12}
			wantHP, wantDecay, wantDwell := int32(31), sim.DecayNone, uint16(0)
			if dying {
				template.HP, template.Decay, template.Dwell = 31, sim.DecayNone, 0
				a.Stage, a.HP, a.DyingTimer = 1, -3, 7
				a.Character.Basis.Stats[8] = uint16(a.HP)
				wantHP, wantDecay, wantDwell = -3, sim.DecayFallen, 7
			}
			world, err := sim.NewWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil, []sim.Entity{template})
			if err != nil {
				t.Fatal(err)
			}
			ms := &Mission{World: world, Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{}}}
			registry := &originalActorRegistry{actors: []originalActorBinding{{Source: a, ID: 1}}, groups: graph.Groups}
			if err := admitOriginalActorRegistry(ms, registry, actorRegistryTable()); err != nil {
				t.Fatal(err)
			}
			e := world.Entities()[0]
			if e.HP != wantHP || e.Decay != wantDecay || e.Dwell != wantDwell {
				t.Fatalf("saved lifecycle became HP %d stage %d dwell %d", e.HP, e.Decay, e.Dwell)
			}
		})
	}
}
