package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A saved Human whose definition row is 0 holds no Humans definition: the
// installed collection carries no parameters at row 0. LOAD constructs the
// actor from the save, with the constructor's defaults for what a definition
// would add, and never refuses the file.
func TestOriginalActorAdmitsHumanWithoutDefinitionRow(t *testing.T) {
	fixture := actorBookFixture(91)
	fixture.human = true
	file, err := sav.Open(poolFixtureSave(fixture))
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil || len(graph.Actors) != 1 {
		t.Fatal("actor fixture", graph, err)
	}
	a := graph.Actors[0]
	a.Class, a.DefRow, a.TypeID, a.Face = "Human", 0, 0x1b, 1
	world, err := sim.NewWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil, []sim.Entity{{ID: 1, X: 5, Y: 6, HP: 31, MaxHP: 31}})
	if err != nil {
		t.Fatal(err)
	}
	ms := &Mission{World: world, Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{}}}
	registry := &originalActorRegistry{actors: []originalActorBinding{{Source: a, ID: 2, New: true}}, groups: graph.Groups}
	if err := admitOriginalActorRegistry(ms, registry, actorRegistryTable()); err != nil {
		t.Fatalf("a Human with definition row 0 refused LOAD: %v", err)
	}
	var found bool
	for _, e := range world.Entities() {
		if e.ID == 2 {
			found = true
			if e.TypeID != 0x1b || !e.Humanoid || e.SourceBinding.DefinitionRow() != 0 {
				t.Fatalf("constructed actor is type %#x humanoid %v row %d", e.TypeID, e.Humanoid, e.SourceBinding.DefinitionRow())
			}
			if e.RotationSpeed == 0 || e.DyingTime == 0 {
				t.Fatalf("constructor defaults missing: rotation %d dying %d", e.RotationSpeed, e.DyingTime)
			}
		}
	}
	if !found {
		t.Fatal("the constructed actor is absent from the world")
	}
}
