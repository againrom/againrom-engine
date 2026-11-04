package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// guardStrikeRow is the guard member after one step of runGuardStrike: its
// cell, health, victim and destination, its order's pursuit word cmd +8 and
// blow record, the server word +0 (World tick / 16), its Group's programme
// byte +0x20 and radius +0x2d, and the hero beside it.
type guardStrikeRow struct {
	Phase           string
	Tick            uint64
	Clock           uint16
	X, Y            int32
	HP, MaxHP       int32
	Victim          sim.EntityID
	Chasing         bool
	DestX, DestY    int32
	Walking         bool
	Pursuit         byte
	Strike          sim.SavedStrike
	Mode, Radius    byte
	HeroX, HeroY    int32
	HeroHP, HeroMax int32
	HeroVictim      sim.EntityID
	HeroChasing     bool
	Hostile         bool
}

// guardStrikeRun is the scenario's member, the hero, the rows and why the
// run stopped early, if it did.
type guardStrikeRun struct {
	Member, Hero sim.EntityID
	Rows         []guardStrikeRow
	Refused      string
}

const (
	// strikeGroupSelector is mission 10's Troll.4 Group.
	strikeGroupSelector = 10
	// strikeAfterPasses is the group ticks the run plays after the blow.
	strikeAfterPasses = 40
)

// runGuardStrike plays mission 10 on a second-game root: the hero, placed
// beside the Troll.4 member by the script's own placement, attacks it until
// one blow lands, then stops and stands three cells off it, inside its
// circle and beyond its reach, for strikeAfterPasses group ticks. The hero
// is healed after every tick. A row is written after the first group tick,
// after each tick of the attack, and after each group tick that follows.
func runGuardStrike(t *testing.T, archives *Archives) guardStrikeRun {
	t.Helper()
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
	ms, err := StartMission(archives.Containers, 10, defs.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	w, hero := ms.World, ms.Start.IDs[0]
	h, _ := w.Entity(hero)
	var member sim.EntityID
	found := 0
	groups, _, _ := w.SavedGroups()
	for _, g := range groups {
		if g.Selector != strikeGroupSelector || g.Owner.Owner == h.Owner {
			continue
		}
		for _, m := range g.Members {
			if m.Bound {
				member, found = m.Entity, found+1
			}
		}
	}
	if found != 1 {
		t.Fatalf("mission 10 holds %d bound members of Group %d, want one", found, strikeGroupSelector)
	}
	run := guardStrikeRun{Member: member, Hero: hero}
	record := func(phase string) guardStrikeRow {
		e, _ := w.Entity(member)
		hh, _ := w.Entity(hero)
		row := guardStrikeRow{Phase: phase, Tick: w.Tick(), Clock: uint16(w.Tick() / 16), X: e.X, Y: e.Y, HP: e.HP, MaxHP: e.MaxHP,
			Victim: e.AttackTarget, Chasing: e.HasAttackTarget, DestX: e.TargetX, DestY: e.TargetY, Walking: e.HasTarget,
			HeroX: hh.X, HeroY: hh.Y, HeroHP: hh.HP, HeroMax: hh.MaxHP, HeroVictim: hh.AttackTarget, HeroChasing: hh.HasAttackTarget,
			Hostile: w.Relations().Hostile(e.Owner, hh.Owner)}
		gs, orders, _ := w.SavedGroups()
		for _, g := range gs {
			for _, m := range g.Members {
				if m.Entity == member {
					row.Mode, row.Radius = g.AI[0x20], g.AI[0x2d]
				}
			}
		}
		for _, o := range orders {
			if o.Entity == member {
				row.Pursuit, row.Strike = o.Raw[8], o.Strike
			}
		}
		run.Rows = append(run.Rows, row)
		return row
	}
	step := func(cmds []sim.Command) {
		sim.Step(w, cmds)
		if hh, _ := w.Entity(hero); hh.Alive() && hh.HP < hh.MaxHP {
			if err := w.HeadlessHeal(hero); err != nil {
				t.Fatal(err)
			}
		}
	}
	pass := func() {
		step(nil)
		for !w.ScriptPassJustRan() {
			step(nil)
		}
	}
	place := func(dx int32) {
		m, _ := w.Entity(member)
		b := w.Bounds()
		x := m.X + dx
		if x < 0 || x >= b.Width {
			x = m.X - dx
		}
		if err := w.HeadlessPlace(hero, x, m.Y); err != nil {
			t.Fatal(err)
		}
	}
	pass()
	record("start")
	place(1)
	struck := false
	cmds := []sim.Command{sim.Attack(hero, member)}
	for n := 0; n < 600 && !struck; n++ {
		step(cmds)
		cmds = nil
		row := record("attack")
		struck = row.Strike.Held && row.Strike.Attacker == hero
		if hh, _ := w.Entity(hero); !hh.Alive() {
			run.Refused = "the hero fell before his blow landed"
			return run
		}
	}
	if !struck {
		run.Refused = "the hero's blow did not land within 600 ticks"
		return run
	}
	run.Rows[len(run.Rows)-1].Phase = "blow"
	hh, _ := w.Entity(hero)
	step([]sim.Command{sim.MoveTo(hero, sim.CellPoint{X: hh.X, Y: hh.Y})})
	for range strikeAfterPasses {
		place(3)
		pass()
		record("after")
		if hh, _ := w.Entity(hero); !hh.Alive() {
			run.Refused = "the hero fell after his blow"
			return run
		}
	}
	return run
}

// The owner's case on the installed mission 10: Troll.4, a guard B member
// struck once by the hero, holds the blow's record; on the group tick it
// pursues the hero beyond its reach it stamps cmd +0x16 with server word +0,
// and while the stamp lies below 10 group ticks back (30 while hurt) it holds
// no victim and walks to cmd +0x9c, or to the nearest cell it reaches; on the
// first group tick past the window it takes the striker again
// (R2-ENGINE-844, R2-ENGINE-845, R2-ENGINE-846, DIV-4503).
func TestReleaseSecondGuardStruckWalksAwayInsideTheStampWindow(t *testing.T) {
	run := runGuardStrike(t, secondGameRoot(t))
	if run.Refused != "" {
		t.Fatal(run.Refused)
	}
	var after []guardStrikeRow
	for _, r := range run.Rows {
		switch r.Phase {
		case "start":
			if r.Mode != 18 || r.Chasing {
				t.Fatalf("Troll.4's Group runs programme %d with victim %v/%v at the mission start, want 18 and none", r.Mode, r.Victim, r.Chasing)
			}
		case "blow":
			// The row follows the step whose World tick wrote the blow.
			if r.Strike.Attacker != run.Hero || r.Strike.Clock != uint32(r.Tick-1) || r.Strike.Stamped {
				t.Fatalf("the blow's record is %+v after tick %d", r.Strike, r.Tick-1)
			}
		case "after":
			after = append(after, r)
		}
	}
	s := -1
	for i, r := range after {
		if r.Strike.Stamped {
			s = i
			break
		}
	}
	if s < 1 || s > 3 {
		t.Fatalf("the first stamp falls on group tick %d after the blow, want the second or third", s+1)
	}
	if p := after[s-1]; !p.Chasing || p.Victim != run.Hero || p.Pursuit != 5 {
		t.Fatalf("before the stamp the member chases %v/%v with cmd +8 = %d, want the hero and 5", p.Victim, p.Chasing, p.Pursuit)
	}
	stamp := after[s].Strike.Stamp
	if stamp != after[s].Clock {
		t.Fatalf("the stamp is %d on server word %d, want the word", stamp, after[s].Clock)
	}
	took := false
	for _, r := range after[s:] {
		d := r.Clock - stamp
		if d >= 10 && r.HP == r.MaxHP || d >= 30 {
			took = r.Chasing && r.Victim == run.Hero
			break
		}
		away := guardStrikeCell(r.Strike.Away)
		goal := sim.CellPoint{X: r.X, Y: r.Y}
		if r.Walking {
			goal = sim.CellPoint{X: r.DestX, Y: r.DestY}
		}
		if r.Chasing || guardStrikeSpan(goal, away) > 2 || r.Strike.Stamp != stamp {
			t.Fatalf("group tick %d after the stamp: the member at (%d,%d), health %d/%d, chases %v/%v and walks to (%d,%d)/%v with stamp %d, want +0x9c %v and stamp %d",
				d, r.X, r.Y, r.HP, r.MaxHP, r.Victim, r.Chasing, r.DestX, r.DestY, r.Walking, r.Strike.Stamp, away, stamp)
		}
	}
	if !took {
		t.Fatal("past the stamp's window the member does not take the striker")
	}
}

// guardStrikeSpan is the Chebyshev distance of a and b.
func guardStrikeSpan(a, b sim.CellPoint) int32 {
	return max(a.X-b.X, b.X-a.X, a.Y-b.Y, b.Y-a.Y)
}

// guardStrikeCell is a cmd +0x9c word's cell.
func guardStrikeCell(v uint16) sim.CellPoint {
	return sim.CellPoint{X: int32(v & 0xff), Y: int32(v >> 8)}
}
