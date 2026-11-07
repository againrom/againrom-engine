package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/sim"
)

// A second-game mission SAVE writes a moving hero's walk goal at command
// +0x0a: the Move programme, 4, walks each member to its own goal
// (AI-MOVE-023), so a zero there sends the hero toward cell 0
// (R2-ENGINE-883). Fails while the second game's command block is written
// with +0x0a zero.
func TestSecondMissionSaveWritesTheHeroWalkGoal(t *testing.T) {
	c := secondGuardMission()
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnter(t, f, app, screen)
	w := f.live.world
	var hero sim.Entity
	found := false
	for _, e := range w.Entities() {
		if e.Owner == sim.SelfSlot {
			hero, found = e, true
			break
		}
	}
	if !found {
		t.Fatal("the mission holds no hero")
	}
	to := sim.CellPoint{X: hero.X + 4, Y: hero.Y}
	sim.Step(w, []sim.Command{sim.GroupMoveTo(hero.ID, to, 1)})
	doc := checkSecondMissionFile(t, secondPageSave(t, f, app), 10)
	want := uint16(to.X) | uint16(to.Y)<<8
	moving := 0
	for _, p := range doc.Players {
		r := &doc.Objects[p-1]
		if slot, _ := savedStructureValue(r, "Slot"); slot != sim.SelfSlot {
			continue
		}
		for i := range r.Groups {
			g := &r.Groups[i]
			if secondRecordRaw(g, "G3C")[0x20] != 4 {
				continue
			}
			for _, refs := range g.RefSlots {
				for _, m := range refs.Objects {
					moving++
					command := secondRecordRaw(&doc.Objects[m-1], "URaw184")
					if goal := binary.LittleEndian.Uint16(command[0x0a:]); len(command) != 184 || goal != want {
						t.Fatalf("the moving hero writes walk goal %#04x, want %#04x", goal, want)
					}
				}
			}
		}
	}
	if moving != 1 {
		t.Fatalf("%d members of the player's Move Groups, want the hero", moving)
	}
}
