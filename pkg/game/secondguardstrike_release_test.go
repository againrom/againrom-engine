package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// guardStrikeRow is the guard member after one step of runGuardStrike: its
// cell, health, victim and destination, its order's pursuit word cmd +8 and
// blow record, the server word +0 (World tick / 16), its Group's programme
// byte +0x20 and radius +0x2d, and the hero beside it. SideArea reports an
// area effect of a caster on the member's own side painting a cell of the
// member's footprint on a step since the previous row.
type guardStrikeRow struct {
	SideArea        bool
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
	// The subject is the group's pursuit after the blow. Mission 10's idle
	// Player 8 mage (Units row 24, start cell 64,36) shields itself, and an
	// unrelated mage's own casts shift the shared draw stream the hero's
	// to-hit roll reads; whether the original's Shield cast draws from it is
	// Unknown. The run removes that mage before its first pass.
	all := w.Entities()
	mage := slices.IndexFunc(all, func(e sim.Entity) bool {
		return e.Owner == 8 && e.TypeID == 24 && e.X == 64 && e.Y == 36
	})
	if mage < 0 {
		t.Fatal("mission 10 holds no Player 8 mage of Units row 24 on cell 64,36")
	}
	if err := w.HeadlessKill(all[mage].ID); err != nil {
		t.Fatal(err)
	}
	run := guardStrikeRun{Member: member, Hero: hero}
	sideArea := false
	paintSide := func() {
		e, _ := w.Entity(member)
		side := int32(max(e.TokenSize, 1))
		for _, c := range w.CellEffects() {
			caster, ok := w.Entity(c.Caster)
			if !c.HasCaster || !ok || c.Caster == member || caster.Owner != e.Owner {
				continue
			}
			for _, k := range c.Cells {
				if k[0] >= e.X && k[0] < e.X+side && k[1] >= e.Y && k[1] < e.Y+side {
					sideArea = true
				}
			}
		}
	}
	record := func(phase string) guardStrikeRow {
		e, _ := w.Entity(member)
		hh, _ := w.Entity(hero)
		row := guardStrikeRow{SideArea: sideArea, Phase: phase, Tick: w.Tick(), Clock: uint16(w.Tick() / 16), X: e.X, Y: e.Y, HP: e.HP, MaxHP: e.MaxHP,
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
		sideArea = false
		return row
	}
	step := func(cmds []sim.Command) {
		sim.Step(w, cmds)
		paintSide()
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

// The installed Troll.4 takes its striker on the first group tick after the
// blow, as guard B's group takes the enemy inside +0x2d (R2-ENGINE-847,
// DIV-4503), and no group tick stamps cmd +0x16 while the hero stands three
// cells off (R2-ENGINE-873). A group tick without the striker follows a
// same-side Bat_Sonic's Darkness painting the troll; what the original's
// member does under that area blow is not established, so the test admits
// the loss there and asks the striker back by the last group tick.
func TestReleaseSecondGuardStruckWalksAwayInsideTheStampWindow(t *testing.T) {
	run := runGuardStrike(t, secondGameRoot(t))
	if run.Refused != "" {
		t.Fatal(run.Refused)
	}
	after, lost := 0, 0
	var last guardStrikeRow
	for _, r := range run.Rows {
		switch r.Phase {
		case "start":
			if r.Mode != 18 || r.Chasing {
				t.Fatalf("Troll.4's Group runs programme %d with victim %v/%v at the mission start, want 18 and none", r.Mode, r.Victim, r.Chasing)
			}
		case "blow":
			if r.Strike.Attacker != run.Hero || r.Strike.Clock != uint32(r.Tick-1) || r.Strike.Stamped {
				t.Fatalf("the blow's record is %+v after tick %d", r.Strike, r.Tick-1)
			}
		case "after":
			after++
			last = r
			if r.Strike.Stamped {
				t.Fatalf("group tick %d after the blow stamps %+v, want no stamp", after, r.Strike)
			}
			if r.Chasing && r.Victim == run.Hero && r.Pursuit == 5 {
				continue
			}
			if after == 1 || !r.SideArea {
				t.Fatalf("group tick %d after the blow: victim %d/%v, cmd +8 %d, same-side area %v; want the striker in pursuit", after, r.Victim, r.Chasing, r.Pursuit, r.SideArea)
			}
			lost++
		}
	}
	if after != strikeAfterPasses {
		t.Fatalf("checked %d group ticks after the blow, want %d", after, strikeAfterPasses)
	}
	if !last.Chasing || last.Victim != run.Hero || last.Pursuit != 5 {
		t.Fatalf("the last group tick holds victim %d/%v with cmd +8 %d, want the striker in pursuit", last.Victim, last.Chasing, last.Pursuit)
	}
	t.Logf("%d of %d group ticks without the striker under a same-side area effect", lost, after)
}
