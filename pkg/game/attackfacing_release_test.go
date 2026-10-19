package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseAttackFacingMissionTrace(t *testing.T) {
	f := releaseFront(t)
	for _, mission := range []int{10, 20, 40} {
		p, _, err := StartScenarioMission(f.Archives, HeadlessScenario{Version: 6, Stage: StageMission, Mission: mission})
		if err != nil {
			t.Fatal(err)
		}
		w := p.World
		shots := 0
		var commands []sim.Command
		for _, e := range w.Entities() {
			if c, ok := f.Table.FlightInputs.Classes[e.Class]; ok && c.Projectile > 0 && e.Reach > 1 {
				t.Logf("candidate m%d id%d class%d cell%d,%d reach%d facing%d rate%d", mission, e.ID, e.Class, e.X, e.Y, e.Reach, e.Facing, e.RotationSpeed)
				victim := p.Party[0]
				best := int32(1 << 30)
				for _, v := range w.Entities() {
					if v.ID == e.ID || v.Owner == e.Owner || !v.Alive() || v.OffMap {
						continue
					}
					dx, dy := v.X-e.X, v.Y-e.Y
					if dx < 0 {
						dx = -dx
					}
					if dy < 0 {
						dy = -dy
					}
					if d := max(dx, dy); d < best {
						best = d
						victim = v.ID
					}
				}
				commands = append(commands, sim.Attack(e.ID, victim))
			}
		}
		for n := 0; n < 1200 && w.Outcome() == sim.OutcomeUndecided; n++ {
			report := sim.StepReported(w, commands)
			commands = nil
			if len(report.FlightStarts) == 0 {
				continue
			}
			for _, e := range w.Entities() {
				c, ok := f.Table.FlightInputs.Classes[e.Class]
				if !ok || c.Projectile <= 0 || !e.HasAttackTarget || !e.FlightRun.Active || e.FlightRun.Kind != sim.AttackCharging || int64(e.FlightRun.Elapsed) != max(int64(c.ShootDelay)-1, 0) {
					continue
				}
				target, ok := w.Entity(e.AttackTarget)
				if !ok {
					continue
				}
				x, y, _ := w.ProjectilePoint(e.ID)
				k := (int(e.Facing>>4) - 8) & 14
				if len(c.ShootOffset) < k+2 {
					continue
				}
				x += 8 * (c.ShootOffset[k] - c.CenterX)
				y += 8 * (c.ShootOffset[k+1] - c.CenterY)
				released := false
				for _, start := range report.FlightStarts {
					if start.Picture == c.Projectile && start.X == x && start.Y == y {
						released = true
					}
				}
				if !released {
					continue
				}
				shots++
				ax, ay, _ := w.ProjectilePoint(e.ID)
				bx, by, _ := w.ProjectilePoint(target.ID)
				ax += int32(max(e.TokenSize, 1)-1) * 128
				ay += int32(max(e.TokenSize, 1)-1) * 128
				bx += int32(max(target.TokenSize, 1)-1) * 128
				by += int32(max(target.TokenSize, 1)-1) * 128
				ax, ay, bx, by = int32(uint16(ax)), int32(uint16(ay)), int32(uint16(bx)), int32(uint16(by))
				direction := attackTraceHeading(bx-ax, by-ay)
				t.Logf("mission=%d tick=%d unit=%d class=%d target=%d facing=%d drawn=%d direction=%d centres=%d,%d:%d,%d delta=%d,%d size=%d/%d phase=%d elapsed=%d", mission, w.Tick(), e.ID, e.Class, target.ID, e.Facing, e.DrawnFacing(), direction, ax, ay, bx, by, target.X-e.X, target.Y-e.Y, e.TokenSize, target.TokenSize, e.AttackPhase, e.FlightRun.Elapsed)
			}
		}
		t.Logf("mission=%d shots=%d hash=%016x", mission, shots, w.Hash())
		if mission == 20 && shots == 0 {
			t.Fatal("directed installed ranged witness released no shot")
		}
	}
}

func attackTraceHeading(dx, dy int32) uint8 {
	x, y := dx, dy
	if x < 0 {
		x = -x
	}
	if y < 0 {
		y = -y
	}
	if x > 2*y {
		if dx < 0 {
			return 192
		}
		return 64
	}
	if y > 2*x {
		if dy < 0 {
			return 0
		}
		return 128
	}
	if dx <= 0 {
		if dy <= 0 {
			return 224
		}
		return 160
	}
	if dy <= 0 {
		return 32
	}
	return 96
}
