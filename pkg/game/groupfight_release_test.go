package game_test

// The Kadagan Post garrison, run on a lawful install with player commands
// after one roaming pack is removed: the hero walks to the post's gate and
// attacks one warrior. The garrison is one placed group of five (two warriors,
// two crossbow bearers and a mage), and every member must start attacking, the
// mage included.
//
// It reads a LAWFUL INSTALL and is skipped unless an asset root is
// configured.

import (
	"os"
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestReleaseAnAttackedGroupAtKadaganPostFightsAsAGroup attacks the warrior
// standing at the gate of the fortified post and watches the five members of
// its group. A member is fighting when it holds an enemy unit as its victim and
// its attack cycle has left the ready phase: a wind-up loaded, a blow or a cast
// in flight. A member that takes a victim and stands beside it never leaves
// the ready phase, which is how the mage stood through the whole fight.
func TestReleaseAnAttackedGroupAtKadaganPostFightsAsAGroup(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the Kadagan Post group fight needs a lawful install")
	}
	archives, err := game.OpenArchives(root)
	if err != nil {
		t.Fatalf("OpenArchives: %v", err)
	}
	defs, err := game.LoadDefinitions(archives.Containers)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	party := game.MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
	ms, err := game.StartMission(archives.Containers, 41, defs.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	w := ms.World
	hero := ms.Start.IDs[0]

	const anchor = sim.EntityID(9)
	first, ok := w.Entity(anchor)
	if !ok {
		t.Fatalf("this install's mission 41 holds no entity %d, the warrior at the gate", anchor)
	}
	var members []sim.EntityID
	for _, e := range w.Entities() {
		if e.Owner == first.Owner && e.Group == first.Group {
			members = append(members, e.ID)
		}
	}
	if len(members) != 5 {
		t.Fatalf("the warrior's group holds %d members %v, want the garrison of five", len(members), members)
	}

	// Without the roaming pack the hero meets on its way, the far members
	// fight or not by the pack's skirmish timing.
	const pack = sim.EntityID(28)
	stray, ok := w.Entity(pack)
	if h, _ := w.Entity(hero); !ok || stray.Owner == first.Owner || stray.Owner == h.Owner {
		t.Fatalf("this install's mission 41 holds no roaming unit %d of a third owner", pack)
	}
	var clear []sim.Command
	for _, e := range w.Entities() {
		if e.Owner == stray.Owner && e.Group == stray.Group {
			clear = append(clear, sim.Kill(e.ID))
		}
	}
	if len(clear) != 4 {
		t.Fatalf("the roaming pack holds %d members, want four", len(clear))
	}
	sim.Step(w, append(clear, sim.MoveTo(hero, sim.CellPoint{X: 52, Y: 61})))
	for i := 0; i < 3000; i++ {
		sim.Step(w, nil)
		if h, _ := w.Entity(hero); !h.HasTarget {
			break
		}
	}
	sim.Step(w, []sim.Command{sim.Attack(hero, anchor)})

	fightingAt := make([]uint64, len(members))
	var firstAt uint64
	remaining := len(members)
	for i := 0; i < 4000 && remaining > 0; i++ {
		sim.Step(w, nil)
		for k, id := range members {
			if fightingAt[k] != 0 {
				continue
			}
			e, _ := w.Entity(id)
			if !e.Alive() || !e.HasAttackTarget || e.AttackPhase == sim.AttackReady {
				continue
			}
			if victim, held := w.Entity(e.AttackTarget); held && w.Relations().Hostile(e.Owner, victim.Owner) {
				fightingAt[k] = w.Tick()
				remaining--
				if firstAt == 0 {
					firstAt = w.Tick()
				}
			}
		}
		if firstAt != 0 && w.Tick() > firstAt+600 {
			break
		}
	}
	for k, id := range members {
		if fightingAt[k] == 0 {
			e, _ := w.Entity(id)
			t.Errorf("member %d never fought within 600 ticks of the first blow at tick %d: "+
				"at %d,%d alive=%v victim=%v/%d phase=%d", id, firstAt, e.X, e.Y, e.Alive(), e.HasAttackTarget, e.AttackTarget, e.AttackPhase)
		}
	}
	if firstAt == 0 {
		t.Fatal("no member of the garrison ever fought")
	}
	t.Logf("first member fought at tick %d; each member fought at ticks %v", firstAt, fightingAt)
}
