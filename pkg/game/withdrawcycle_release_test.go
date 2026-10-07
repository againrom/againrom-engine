package game

import (
	"fmt"
	"sort"
	"testing"

	"againrom/pkg/sim"
)

// releaseWithdrawalMission is the campaign mission whose shipped placements
// carry ranged withdrawers standing a short walk from the party start on both
// installs.
const releaseWithdrawalMission = 41

// TestReleaseRangedWithdrawalLetsItsLoadedShotResolveBeforeItFlees drives the
// party's first hero at a shipped ranged withdrawer through ordinary play: one
// attack command, then production ticks. The first full tick at which the
// withdrawal tail fires on a creature holding a loaded attack cycle must leave
// that cycle alone, the creature must stand until its shot has resolved, and
// its flee walk must start only after the cycle has ended (AI-WITHDRAW-028,
// AI-RETREAT-272, AI-ORDER-039, HERO-CADENCE-112).
func TestReleaseRangedWithdrawalLetsItsLoadedShotResolveBeforeItFlees(t *testing.T) {
	f := releaseFront(t)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	start := func() *Mission {
		m, err := StartMission(f.Archives.Containers, releaseWithdrawalMission, f.Table, openDifficulty, party)
		if err != nil {
			t.Fatalf("StartMission(%d): %v", releaseWithdrawalMission, err)
		}
		return m
	}

	first := start()
	hero := first.Start.IDs[0]
	var self sim.Entity
	for _, e := range first.World.EntityView() {
		if e.ID == hero {
			self = e
		}
	}
	type candidate struct {
		id   sim.EntityID
		dist int32
	}
	var candidates []candidate
	rel := first.World.Relations()
	for _, e := range first.World.EntityView() {
		if !e.Alive() || e.OffMap || e.Owner == 0 || e.Withdraw <= 0 || e.Reach <= 1 || !rel.Hostile(e.Owner, self.Owner) {
			continue
		}
		d := releaseAbs32(e.X - self.X)
		if dy := releaseAbs32(e.Y - self.Y); dy > d {
			d = dy
		}
		candidates = append(candidates, candidate{e.ID, d})
	}
	if len(candidates) == 0 {
		t.Fatalf("mission %d holds no hostile ranged withdrawer", releaseWithdrawalMission)
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].dist < candidates[j].dist })
	if len(candidates) > 8 {
		candidates = candidates[:8]
	}

	const horizon = 1600
	view := func(w *sim.World, id sim.EntityID) sim.Entity {
		for _, e := range w.EntityView() {
			if e.ID == id {
				return e
			}
		}
		return sim.Entity{}
	}
	for _, c := range candidates {
		m := start()
		w := m.World
		cmds := []sim.Command{sim.Attack(hero, c.id)}
		var (
			decisionTick, blowTick, leftTick uint64
			held                             sim.Entity
			prev                             = view(w, c.id)
			lines                            []string
		)
		for n := 0; n < horizon && leftTick == 0; n++ {
			decisions := sim.StepWithdrawalTraced(w, cmds)
			cmds = nil
			cur := view(w, c.id)
			for _, d := range decisions {
				if d.Before.ID != c.id || decisionTick != 0 || !d.Before.HasAttackTarget || d.Before.AttackPhase == sim.AttackReady {
					continue
				}
				decisionTick, held = w.Tick(), cur
				lines = append(lines, fmt.Sprintf("tick %d decision on a loaded cycle: before phase %d countdown %d at (%d,%d); after phase %d countdown %d attack %t destination (%d,%d,%t)",
					decisionTick, d.Before.AttackPhase, d.Before.AttackCountdown, d.Before.X, d.Before.Y,
					d.After.AttackPhase, d.After.AttackCountdown, d.After.HasAttackTarget, d.After.TargetX, d.After.TargetY, d.After.HasTarget))
				if !d.After.HasAttackTarget || d.After.AttackPhase != d.Before.AttackPhase || d.After.AttackCountdown != d.Before.AttackCountdown || !d.After.HasTarget {
					t.Fatalf("victim %d: the tail discarded the loaded cycle: %s", c.id, lines[len(lines)-1])
				}
			}
			if decisionTick != 0 {
				if prev.AttackPhase == sim.AttackCharging && cur.AttackPhase == sim.AttackRelaxing && blowTick == 0 {
					blowTick = w.Tick()
				}
				if (cur.X != held.X || cur.Y != held.Y) && leftTick == 0 {
					leftTick = w.Tick()
					lines = append(lines, fmt.Sprintf("tick %d flee walk begins to (%d,%d): blow resolved at tick %d, phase %d, attack order %t",
						leftTick, cur.X, cur.Y, blowTick, cur.AttackPhase, cur.HasAttackTarget))
					if blowTick == 0 || cur.AttackPhase != sim.AttackReady || cur.HasAttackTarget {
						t.Fatalf("victim %d left its cell before its loaded shot resolved and its cycle ended: %v", c.id, lines)
					}
				}
			}
			prev = cur
			if !cur.Alive() {
				break
			}
		}
		if decisionTick == 0 {
			t.Logf("victim %d (distance %d): no withdrawal on a loaded cycle within %d ticks", c.id, c.dist, horizon)
			continue
		}
		if leftTick == 0 {
			t.Fatalf("victim %d held its cell for the rest of the drive after tick %d: %v", c.id, decisionTick, lines)
		}
		for _, line := range lines {
			t.Logf("mission %d victim %d: %s", releaseWithdrawalMission, c.id, line)
		}
		return
	}
	t.Fatalf("no ranged withdrawer of mission %d met the hero on a loaded cycle within %d ticks", releaseWithdrawalMission, horizon)
}
