package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// orderedVictimFight is the arena a warrior's ordered victim is measured in:
// mission 20's own ground and its own first hostile group, the three creatures
// the map places for owner 2. The entering warrior stands beside one of them
// and the other two are placed one after the other further out, so they walk in
// and strike him in turn. Placement, the warrior's Body and the first
// creature's health are fixtures; the terrain, the creatures' statistics, the
// relations and every order the fight runs on are the install's and the
// production input path's.
type orderedVictimFight struct {
	live      *mapWorld
	app       *ui.App
	hero      sim.EntityID
	first     sim.EntityID
	newcomers []sim.EntityID
}

// openGroundSquare returns the top-left cell of the first size-by-size square
// of the block plane with no closed cell.
func openGroundSquare(block []byte, width, height, size int) (int, int, bool) {
	for y := 0; y+size <= height; y++ {
		for x := 0; x+size <= width; x++ {
			open := true
			for dy := 0; dy < size && open; dy++ {
				for dx := 0; dx < size; dx++ {
					if block[(y+dy)*width+x+dx] != 0 {
						open = false
						break
					}
				}
			}
			if open {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// beside reports whether a stands on a cell b's reach of one covers.
func beside(a, b sim.Entity) bool {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx >= -1 && dx <= 1 && dy >= -1 && dy <= 1
}

// openOrderedVictimFight opens mission 20 with a warrior and stands the arena
// on it. The first creature has the highest id of the three, so an ordinary
// decision prefers each newcomer when it arrives beside the warrior.
func openOrderedVictimFight(t *testing.T) *orderedVictimFight {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	// Body 40: the warrior's health is derived from Body again at his first
	// order, so a health value set on the entity would not last.
	party := f.ChargenParty(ui.ChargenResult{Name: "Ordered victim", Choices: []int{0, 0, 0}, Stats: []int{40, 30, 30, 30}})
	a := f.App("ordered victim")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	hero, ok := live.entity(live.mission.ids[0])
	if !ok {
		t.Fatal("no party actor")
	}
	var trio []sim.Entity
	for _, e := range live.world.Entities() {
		if e.Owner == 2 && e.Group == 1 {
			trio = append(trio, e)
		}
	}
	if len(trio) != 3 || trio[0].ID != 0 || trio[1].ID != 1 || trio[2].ID != 2 {
		t.Fatalf("mission 20's first hostile group changed: %d entities", len(trio))
	}
	rel := live.world.Relations()
	if !rel.Hostile(sim.SelfSlot, 2) || !rel.Hostile(2, sim.SelfSlot) {
		t.Fatal("owner 2 is not hostile to the participant in mission 20")
	}

	m := live.mission.state.Map
	terrain := sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)}
	bounds := live.world.Bounds()
	px, py, found := openGroundSquare(terrain.Block, int(bounds.Width), int(bounds.Height), 15)
	if !found {
		t.Fatal("mission 20 has no open ground square of 15 cells")
	}
	cx, cy := int32(px+7), int32(py+7)
	place := func(e sim.Entity, x, y int32) sim.Entity {
		e.X, e.Y, e.PostX, e.PostY = x, y, x, y
		e.TargetX, e.TargetY, e.HasTarget = 0, 0, false
		e.Transit, e.TransitTotal = 0, 0
		return e
	}
	hero = place(hero, cx, cy)
	trio[2] = place(trio[2], cx-1, cy)
	trio[2].HP, trio[2].MaxHP = 60, 60
	trio[0] = place(trio[0], cx, cy+4)
	trio[1] = place(trio[1], cx+2, cy+7)

	worn, _ := live.world.EquippedItems(hero.ID)
	for slot := range worn {
		worn[slot].ObjectID = 0
	}
	arena, err := sim.NewStructuredWorld(2020, bounds, sim.ModeCanonical, terrain,
		[]sim.Entity{trio[0], trio[1], trio[2], hero}, nil, rel, nil,
		[]sim.Stock{{ID: hero.ID, EquippedItems: worn, ItemInstances: party[0].CarriedItems}},
		mapload.SpellRules(f.Table), sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	codes := make([]uint16, 0)
	for _, item := range worn {
		if item.Code != 0 {
			codes = append(codes, item.Code)
		}
	}
	for _, item := range party[0].CarriedItems {
		codes = append(codes, item.Code)
	}
	mapload.DeclareCodeWeights(arena, f.Table, codes)
	mapload.BindSourceDerive(arena)
	live.world = arena
	candidate := *live.mission.state
	candidate.World, candidate.savedDocument = arena, nil
	candidate.ActorManifest = &SnapshotActorManifest{Version: actorManifestVersion}
	if live.mission.state.ActorManifest != nil {
		for _, row := range live.mission.state.ActorManifest.Actors {
			if row.ID == hero.ID || row.ID <= 2 {
				candidate.ActorManifest.Actors = append(candidate.ActorManifest.Actors, row)
			}
		}
	}
	live.mission.state = &candidate
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	inspectionCentre(live, int(cx), int(cy))
	live.push()
	return &orderedVictimFight{live: live, app: a, hero: hero.ID, first: trio[2].ID,
		newcomers: []sim.EntityID{trio[0].ID, trio[1].ID}}
}

// strikers counts the creatures beside the warrior that hold him as their
// victim.
func (fight *orderedVictimFight) strikers() int {
	hero, _ := fight.live.entity(fight.hero)
	n := 0
	for _, id := range append([]sim.EntityID{fight.first}, fight.newcomers...) {
		e, _ := fight.live.entity(id)
		if e.Alive() && e.HasAttackTarget && e.AttackTarget == fight.hero && beside(e, hero) {
			n++
		}
	}
	return n
}

// Tester item: a warrior the player ordered onto one creature turned on each
// new attacker before he finished it. The original gives an attack order a
// fresh group at group order 0 with the named target (AI-CMD-033, AI-CMD-054),
// and being struck issues no order and produces no target (AI-RETAL-056).
//
// The warrior is selected, ordered with the attack key and a click on the first
// creature, and the world advances by ordinary ticks while the other two
// creatures walk in and strike him. The unordered warrior in the same arena is
// the control: it changes to a newcomer, so the fight is the tester's.
//
// The arena is installed-data ground, not a claim about where mission 20
// places its creatures. The first creature's health is raised so it outlasts
// the newcomers' walk, and the warrior's Body so that three strikers cannot
// bring him down before it falls.
func TestReleaseAWarriorKeepsItsOrderedVictim(t *testing.T) {
	t.Run("control: without the order the warrior changes to a newcomer", func(t *testing.T) {
		fight := openOrderedVictimFight(t)
		live := fight.live
		heldFirst, changed := false, false
		for n := 0; n < 600 && !changed; n++ {
			live.tick()
			h, _ := live.entity(fight.hero)
			v, _ := live.entity(fight.first)
			if !v.OrdinaryTargetable() {
				break
			}
			switch {
			case h.HasAttackTarget && h.AttackTarget == fight.first:
				heldFirst = true
			case heldFirst && h.HasAttackTarget && h.AttackTarget != fight.first:
				changed = true
				t.Logf("the unordered warrior left the first creature for %d at tick %d", h.AttackTarget, live.world.Tick())
			}
		}
		if !heldFirst || !changed {
			t.Fatalf("held the first creature %v, changed to a newcomer %v: the arena does not produce the tester's fight",
				heldFirst, changed)
		}
	})

	t.Run("ordered: the warrior stays on the first creature until it falls", func(t *testing.T) {
		fight := openOrderedVictimFight(t)
		live, a := fight.live, fight.app
		if err := a.HeadlessSelectEntity(uint32(fight.hero)); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessKey("attack"); err != nil {
			t.Fatal(err)
		}
		x, y, err := a.HeadlessEntityPoint(uint32(fight.first))
		if err != nil {
			t.Fatal(err)
		}
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		if len(live.pending) != 1 || live.pending[0].Kind != sim.KindAttack || live.pending[0].Entity != fight.hero ||
			live.pending[0].X != int32(fight.first) {
			t.Fatalf("the click queued %+v, want the warrior's attack on creature %d", live.pending, fight.first)
		}
		live.tick()
		crowded, fell := 0, false
		for n := 0; n < 600; n++ {
			h, _ := live.entity(fight.hero)
			v, _ := live.entity(fight.first)
			if !h.Alive() {
				t.Fatalf("tick %d: the warrior fell before the first creature", live.world.Tick())
			}
			if !v.OrdinaryTargetable() {
				fell = true
				break
			}
			if !h.HasAttackTarget || h.AttackTargetKind != sim.AttackTargetUnit || h.AttackTarget != fight.first {
				t.Fatalf("tick %d: the warrior ordered onto creature %d holds victim %v/%d with %d strikers",
					live.world.Tick(), fight.first, h.HasAttackTarget, h.AttackTarget, fight.strikers())
			}
			crowded = max(crowded, fight.strikers())
			live.tick()
		}
		if !fell {
			t.Fatal("the warrior never brought the first creature down in 600 ticks")
		}
		if crowded < 3 {
			t.Fatalf("at most %d creatures struck the warrior at once, want all 3 before the first fell", crowded)
		}
		t.Logf("the first creature fell at tick %d with %d creatures striking the warrior", live.world.Tick(), crowded)

		// The order ends with its victim: the warrior takes a creature still
		// beside him.
		took := false
		for n := 0; n < 200 && !took; n++ {
			live.tick()
			h, _ := live.entity(fight.hero)
			for _, id := range fight.newcomers {
				if e, _ := live.entity(id); h.HasAttackTarget && h.AttackTarget == id && e.OrdinaryTargetable() {
					took = true
				}
			}
		}
		if !took {
			h, _ := live.entity(fight.hero)
			t.Fatalf("after the first creature fell the warrior holds victim %v/%d, want a creature beside him",
				h.HasAttackTarget, h.AttackTarget)
		}
	})
}
