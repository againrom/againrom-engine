package game

import (
	"testing"

	"againrom/pkg/sim"
)

func openCampaignMission(t *testing.T, mission int) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	prepareAcceptedCampaignMission(t, f, mission)
	a := f.App("script group owner")
	if err := a.OpenMission(f.MissionOpenerWith(mission, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	return f
}

// A script group id carried by two owners resolves to the later owner's group.
// The installed missions that carry such an id are 70 (ids 40 and 7), 100 (id
// 25) and 140 (id 6).
func TestReleaseScriptGroupIdResolvesToOneOwner(t *testing.T) {
	for _, tc := range []struct {
		mission int
		group   uint32
		owner   uint32
	}{{70, 40, 7}, {70, 7, 3}, {100, 25, 6}, {140, 6, 3}} {
		f := openCampaignMission(t, tc.mission)
		carried := map[uint32]int{}
		for _, e := range f.live.world.Entities() {
			if e.Group == tc.group {
				carried[e.Owner]++
			}
		}
		if len(carried) != 2 {
			t.Fatalf("mission %d group %d: the install no longer carries the id under two owners: %v", tc.mission, tc.group, carried)
		}
		got := sim.ObserveScriptGroups(f.live.world, tc.group)
		if len(got) != 1 || got[0].Owner != tc.owner {
			t.Errorf("mission %d group %d resolves to %+v, want the single group of owner %d", tc.mission, tc.group, got, tc.owner)
		}
	}
}

// Mission 70's two Swarm 2 instants name group 40, carried by owner 5's orcs and
// owner 7's monks. Only the monks receive the command and its destination.
func TestReleaseMonasteryMissionSwarmReachesTheMonksNotTheOrcs(t *testing.T) {
	f := openCampaignMission(t, 70)
	mw := f.live
	type pos struct{ x, y int32 }
	start := map[sim.EntityID]pos{}
	for _, e := range mw.world.Entities() {
		if e.Group == 40 {
			start[e.ID] = pos{e.X, e.Y}
		}
	}
	for i := 0; i < 60; i++ {
		mw.tick()
	}
	var orcs, monks int
	for _, e := range mw.world.Entities() {
		if e.Group != 40 {
			continue
		}
		switch e.Owner {
		case 5:
			orcs++
			if e.HasTarget || (pos{e.X, e.Y}) != start[e.ID] {
				t.Errorf("owner 5 orc %d swarmed: target %v at (%d,%d), started (%d,%d)", e.ID, e.HasTarget, e.X, e.Y, start[e.ID].x, start[e.ID].y)
			}
		case 7:
			monks++
			if !e.HasTarget || e.TargetX != 23 || e.TargetY != 96 {
				t.Errorf("owner 7 monk %d has target %v (%d,%d), want the swarm cell (23,96)", e.ID, e.HasTarget, e.TargetX, e.TargetY)
			}
		}
	}
	if orcs == 0 || monks == 0 {
		t.Fatalf("group 40 holds %d orcs and %d monks", orcs, monks)
	}
}

func monasteryFinish(t *testing.T, killOrcs bool) sim.Outcome {
	t.Helper()
	f := openCampaignMission(t, 70)
	mw := f.live
	for _, e := range mw.world.Entities() {
		spared := e.Group == 40 && e.Owner == 5 && !killOrcs
		if (e.Group == 101 || e.Group == 50 || e.Group == 40) && !spared {
			if err := mw.world.HeadlessKill(e.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := 0; i < 128; i++ {
		mw.tick()
	}
	return mw.world.Outcome()
}

// The mission's completion trigger waits for the count of group 40 to reach
// zero. The count is the monks' alone, so the orcs of the earlier owner do not
// hold the mission open.
func TestReleaseMonasteryMissionCompletionCountsTheMonksAlone(t *testing.T) {
	if got := monasteryFinish(t, true); got != sim.OutcomeWon {
		t.Fatalf("control with every carrier of the three groups dead: outcome %v, want won", got)
	}
	if got := monasteryFinish(t, false); got != sim.OutcomeWon {
		t.Fatalf("with owner 5's orcs left alive: outcome %v, want won", got)
	}
}
