package sim

import (
	"slices"
	"testing"
)

func sessionMonster(id EntityID, owner, group uint32, x, y int32) Entity {
	e := outpostActor(id, owner, group, x, y)
	e.Mana, e.MaxMana = 7, 7
	return e
}

type sessionReturnCase struct {
	name  string
	world func(t *testing.T, ents []Entity) *World
}

func sessionReturnCases() []sessionReturnCase {
	return []sessionReturnCase{
		{"first", func(t *testing.T, ents []Entity) *World { return scriptWorld(t, nil, ents) }},
		{"second", func(t *testing.T, ents []Entity) *World {
			return secondWorld(t, 1, Bounds{Width: 32, Height: 32}, nil, ents...)
		}},
	}
}

func sessionReturnEntities() []Entity {
	return []Entity{outpostActor(1, SelfSlot, 3, 9, 20), sessionMonster(2, 2, 7, 10, 10),
		sessionMonster(3, 2, 7, 13, 11), sessionMonster(4, 2, 8, 20, 20)}
}

func felledSteps(w *World, ids ...EntityID) {
	cmds := make([]Command, 0, len(ids))
	for _, id := range ids {
		i := indexOfEntity(w.entities, id)
		cmds = append(cmds, Damage(id, w.entities[i].HP+3))
	}
	Step(w, cmds)
}

func stepsUntilBack(w *World, limit int, ids ...EntityID) int {
	for n := 1; n <= limit; n++ {
		Step(w, nil)
		back := true
		for _, id := range ids {
			if i := indexOfEntity(w.entities, id); i < 0 || !w.entities[i].Alive() {
				back = false
			}
		}
		if back {
			return n
		}
	}
	return limit + 1
}

func TestSessionOutpostsReturnAGroupOnceEveryMemberIsDown(t *testing.T) {
	for _, c := range sessionReturnCases() {
		t.Run(c.name, func(t *testing.T) {
			w := c.world(t, sessionReturnEntities())
			w.DeclareOutposts(OutpostMap{Session: true})
			if o := outpostOf(t, w, 2, 7); o.Reload != sessionReturnPasses || len(o.Members) != 2 || o.Radius != 0 {
				t.Fatalf("Outpost %+v", o)
			}
			felledSteps(w, 2, 3)
			waited := stepsUntilBack(w, (sessionReturnPasses+4)*scriptCycle*4, 2, 3)
			if min := sessionReturnPasses * scriptCycle; waited < min || waited > min*3 {
				t.Fatalf("the group stood again after %d steps, want about %d", waited, min)
			}
			for _, want := range []Entity{sessionMonster(2, 2, 7, 10, 10), sessionMonster(3, 2, 7, 13, 11)} {
				e := w.entities[indexOfEntity(w.entities, want.ID)]
				if e.X != want.X || e.Y != want.Y || e.HP != want.MaxHP || e.Mana != want.MaxMana {
					t.Fatalf("member %d at %d,%d health %d mana %d", e.ID, e.X, e.Y, e.HP, e.Mana)
				}
			}
		})
	}
}

func TestSessionOutpostsKeepAGroupWithOneMemberAlive(t *testing.T) {
	for _, c := range sessionReturnCases() {
		t.Run(c.name, func(t *testing.T) {
			w := c.world(t, sessionReturnEntities())
			w.DeclareOutposts(OutpostMap{Session: true})
			felledSteps(w, 2)
			if n := stepsUntilBack(w, (sessionReturnPasses+4)*scriptCycle*3, 2); n <= (sessionReturnPasses+4)*scriptCycle*3 {
				t.Fatalf("member 2 came back after %d steps while member 3 stood", n)
			}
		})
	}
}

func TestSessionOutpostsReturnOnTheNearestFreeCellWhenTheCellIsTaken(t *testing.T) {
	for _, c := range sessionReturnCases() {
		t.Run(c.name, func(t *testing.T) {
			w := c.world(t, sessionReturnEntities())
			w.DeclareOutposts(OutpostMap{Session: true})
			felledSteps(w, 4)
			for n := 0; n < 4000; n++ {
				if i := indexOfEntity(w.entities, 4); i < 0 || w.entities[i].Decay >= decayDarkStage {
					break
				}
				Step(w, nil)
			}
			held := outpostActor(9, SelfSlot, 3, 20, 20)
			hid, err := w.createActor(held, nil, [12]ItemInstance{}, 20, 20, nearestFreeCell(20, 20), true, actorCreationProfile{})
			if err != nil {
				t.Fatal(err)
			}
			if i := indexOfEntity(w.entities, hid); i < 0 || w.entities[i].X != 20 || w.entities[i].Y != 20 {
				t.Fatalf("the blocker did not stand on 20,20 (id %d)", hid)
			}
			if n := stepsUntilBack(w, (sessionReturnPasses+4)*scriptCycle*4, 4); n > (sessionReturnPasses+4)*scriptCycle*4 {
				t.Fatal("the group did not come back")
			}
			e := w.entities[indexOfEntity(w.entities, 4)]
			if (e.X == 20 && e.Y == 20) || e.X < 19 || e.X > 21 || e.Y < 19 || e.Y > 21 {
				t.Fatalf("member came back at %d,%d, want a cell next to the taken 20,20", e.X, e.Y)
			}
		})
	}
}

func TestWithoutTheSessionRuleADeadGroupStaysDown(t *testing.T) {
	for _, c := range sessionReturnCases() {
		t.Run(c.name, func(t *testing.T) {
			w := c.world(t, sessionReturnEntities())
			w.DeclareOutposts(OutpostMap{})
			felledSteps(w, 2, 3)
			if n := stepsUntilBack(w, (sessionReturnPasses+4)*scriptCycle*3, 2, 3); n <= (sessionReturnPasses+4)*scriptCycle*3 {
				t.Fatalf("a campaign group came back after %d steps", n)
			}
		})
	}
}

func returnedIDs(t *testing.T, m OutpostMap) []EntityID {
	t.Helper()
	w := secondWorld(t, 1, Bounds{Width: 32, Height: 32}, nil, outpostActor(1, SelfSlot, 3, 9, 20),
		outpostActor(0, 2, 7, 10, 10), outpostActor(3, 2, 7, 13, 11))
	w.DeclareOutposts(m)
	felledSteps(w, 0, 3)
	for range 4000 {
		Step(w, nil)
	}
	var ids []EntityID
	for _, e := range w.entities {
		ids = append(ids, e.ID)
	}
	return ids
}

func TestCampaignOutpostGivesAnIDZeroMemberANewID(t *testing.T) {
	campaign := returnedIDs(t, OutpostMap{Runs: true})
	if slices.Contains(campaign, 0) || len(campaign) != 3 {
		t.Fatalf("campaign ids %v, want a new id for member 0", campaign)
	}
	if session := returnedIDs(t, OutpostMap{Session: true}); !slices.Contains(session, 0) {
		t.Fatalf("session ids %v, want member 0 kept", session)
	}
}
