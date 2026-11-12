package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

func TestSecondDefendWalkAndNineShortcutsSurviveOrdinarySAV(t *testing.T) {
	f, app, screen, _ := secondBagFront(t, nil)
	secondBagEnter(t, f, app, screen)
	var party []sim.Entity
	for _, e := range f.live.world.EntityView() {
		if e.Owner == sim.SelfSlot && e.Alive() {
			party = append(party, e)
		}
	}
	if len(party) != 2 {
		t.Fatalf("fixture party has %d actors", len(party))
	}
	subject, defender := party[0], party[1]
	f.live.enqueue(uint32(defender.ID), int(subject.X+1), int(subject.Y))
	for n := 0; n < 240; n++ {
		f.live.tick()
		e, _ := f.live.world.Entity(defender.ID)
		if e.X == subject.X+1 && e.Y == subject.Y && e.Transit == 0 && !e.HasTarget {
			break
		}
	}
	defender, _ = f.live.world.Entity(defender.ID)
	if defender.X != subject.X+1 || defender.Y != subject.Y || defender.HasTarget {
		t.Fatal("fixture defender did not reach the adjacent cell")
	}
	f.live.defend(uint32(defender.ID), uint32(subject.ID))
	for n := 0; n < 20; n++ {
		f.live.tick()
		e, _ := f.live.world.Entity(defender.ID)
		if e.HasTarget && e.HasEscortTarget {
			break
		}
	}
	before, _ := f.live.world.Entity(defender.ID)
	if !before.HasTarget || !before.HasEscortTarget || before.EscortTarget != subject.ID || before.ActorState != 8 {
		t.Fatalf("checkpoint has no Defend walk: subject (%d,%d), defender (%d,%d), destination (%d,%d)/%v", subject.X, subject.Y, before.X, before.Y, before.TargetX, before.TargetY, before.HasTarget)
	}
	if before.TargetX != subject.X+4 || before.TargetY != subject.Y+1 {
		t.Fatalf("step-away destination (%d,%d), want (%d,%d)", before.TargetX, before.TargetY, subject.X+4, subject.Y+1)
	}
	// Four ordinary slots and five rows in the existing session record.
	f.quickSpells = [9]uint32{1, 2, 3, 4, 5, 6, 7, 8, 9}
	raw := secondPageSave(t, f, app)
	checkSecondMissionFile(t, raw, 10)
	doc := engineSaveDocument(t, raw)
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if actions == nil || actions.Session == nil {
		t.Fatal("SAV has no current session record")
	}
	last := len(actions.Session.QuickSpells) - 1
	if last < 0 {
		t.Fatal("no extended shortcut rows were written")
	}
	actions.Session.QuickSpells[last].Slot = 9
	if validateCurrentSession(actions.Session) == nil {
		t.Fatal("shortcut slot outside the profile was admitted")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cold, _ := secondMissionLoadOf(t, secondMissionContent{side: 128}, dir, "game0001.sav")
	if cold.live == nil || cold.quickSpells != f.quickSpells {
		t.Fatal("SAV lost mission or nine shortcuts")
	}
	restored, ok := cold.live.world.Entity(defender.ID)
	if !ok || restored.TargetX != before.TargetX || restored.TargetY != before.TargetY || !restored.HasTarget || !restored.HasEscortTarget || restored.EscortTarget != subject.ID || restored.EscortRange != 3 {
		t.Fatal("SAV lost the current Defend destination or binding")
	}
	for _, mw := range []*mapWorld{f.live, cold.live} {
		mw.stopped = false
		arrived := false
		for n := 0; n < 240; n++ {
			mw.tick()
			e, _ := mw.world.Entity(defender.ID)
			if e.X == before.TargetX && e.Y == before.TargetY && e.Transit == 0 {
				arrived = true
				break
			}
		}
		if !arrived {
			e, _ := mw.world.Entity(defender.ID)
			t.Fatalf("defender did not arrive: at (%d,%d), target (%d,%d)/%v, escort %v", e.X, e.Y, e.TargetX, e.TargetY, e.HasTarget, e.HasEscortTarget)
		}
	}
}
