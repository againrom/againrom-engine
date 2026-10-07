package game

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

func releaseTargetIDNodes(t *testing.T, f *FrontEnd) []string {
	t.Helper()
	var out []string
	for _, name := range ArchiveMaps(f.Archives.Containers).Names() {
		mission, err := strconv.Atoi(strings.TrimSuffix(strings.ToLower(name), ".alm"))
		if err != nil {
			t.Fatalf("campaign map %q has no numeric mission identity", name)
		}
		p, _, err := StartScenarioMission(f.Archives, HeadlessScenario{Version: 6, Stage: StageMission, Mission: mission})
		if err != nil {
			t.Fatalf("mission %d: %v", mission, err)
		}
		for i, c := range p.World.Script().Checks() {
			if c.Op == sim.ScriptCheckTargetID {
				out = append(out, fmt.Sprintf("%d:%d:%d", mission, i, c.Unit))
			}
		}
	}
	return out
}

func TestReleaseCheckTargetIDReadsOrderFiveOnly(t *testing.T) {
	f := releaseFront(t)
	nodes := releaseTargetIDNodes(t, f)
	if got, want := strings.Join(nodes, " "), "111:10:42 140:9:1 140:12:123"; got != want {
		t.Fatalf("check opcode 9 nodes are %s, want %s", got, want)
	}
	type subject struct {
		mission, node int
		unit          sim.EntityID
	}
	byMission := map[int][]subject{}
	for _, n := range nodes {
		var s subject
		if _, err := fmt.Sscanf(n, "%d:%d:%d", &s.mission, &s.node, &s.unit); err != nil {
			t.Fatal(err)
		}
		byMission[s.mission] = append(byMission[s.mission], s)
	}
	for _, mission := range []int{111, 140} {
		p, _, err := StartScenarioMission(f.Archives, HeadlessScenario{Version: 6, Stage: StageMission, Mission: mission})
		if err != nil {
			t.Fatal(err)
		}
		script := p.World.Script()
		checks := script.Checks()
		triggers := script.Triggers()
		for _, s := range byMission[mission] {
			reg := checks[s.node].Register
			readers := 0
			for _, tr := range triggers {
				for _, pr := range tr.Pairs {
					if pr.Used && (pr.Left == reg || pr.Right == reg) {
						readers++
					}
				}
			}
			t.Logf("mission %d check %d: subject %d, register %d, read by %d trigger pair(s)", mission, s.node, s.unit, reg, readers)
		}
		var ticks, acquisition, idle, order5 int
		for k := 0; k < 4000; k++ {
			p.step(nil)
			for _, s := range byMission[mission] {
				e, ok := p.entity(s.unit)
				if !ok {
					continue
				}
				ticks++
				victimField := int32(0)
				if e.HasAttackTarget && e.AttackTargetKind == sim.AttackTargetUnit {
					if v, ok := p.entity(e.AttackTarget); ok {
						victimField = int32(v.MapUnitID)
					}
				}
				switch {
				case e.AcquirePursuit:
					acquisition++
				case e.PursuitIdle:
					idle++
				case e.HasAttackTarget:
					order5++
				}
				reg := p.World.ScriptRegister(checks[s.node].Register)
				if (e.AcquirePursuit || e.PursuitIdle) && victimField != 0 && reg != 0 {
					t.Fatalf("mission %d tick %d check %d: subject %d holds acquisition %v idle %v and the register reads %d",
						mission, k, s.node, s.unit, e.AcquirePursuit, e.PursuitIdle, reg)
				}
			}
		}
		t.Logf("mission %d: %d subject-ticks; acquisition %d, idle %d, order 5 %d", mission, ticks, acquisition, idle, order5)
	}
}
