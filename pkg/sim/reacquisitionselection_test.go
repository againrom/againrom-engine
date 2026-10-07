package sim

import "testing"

func selectionWorld(t *testing.T, foes ...Entity) *World {
	t.Helper()
	self := retreatFighter(1, 20, 20)
	self.Reach, self.ScanRange, self.Facing = 4, 8, 64
	return engWorld(t, retreatRelations(t), append([]Entity{self}, foes...)...)
}

func TestReacquisitionAdmissionPrecedesDiplomacyAndHealth(t *testing.T) {
	for _, dead := range []bool{false, true} {
		for _, order := range [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
			actors := []Entity{retreatIntruder(0, 24, 20), retreatIntruder(0, 20, 17), retreatIntruder(0, 21, 20)}
			if dead {
				actors[2].HP = 0
			} else {
				actors[2].Owner = SelfSlot
			}
			var input []Entity
			for n, kind := range order {
				e := actors[kind]
				e.ID = EntityID(n + 2)
				input = append(input, e)
			}
			w := selectionWorld(t, input...)
			want := -1
			nearest := int64(5)
			var survivors []int
			for n, kind := range order {
				distance := []int64{4, 3, 1}[kind]
				if distance <= nearest {
					nearest = distance
					if kind != 2 {
						survivors = append(survivors, n+1)
					}
				}
			}
			if len(survivors) != 0 {
				want = survivors[0]
				for _, n := range survivors {
					if w.entities[n].X == 24 {
						want = n
					}
				}
			} else if dead {
				for n, kind := range order {
					if kind == 2 {
						want = n + 1
					}
				}
			}
			if got := w.reacquisitionVictim(0); got != want {
				t.Errorf("dead=%t order=%v: victim index %d, want %d", dead, order, got, want)
			}
		}
	}
}

func TestReacquisitionUsesTurnScoreAndLastEqualSurvivor(t *testing.T) {
	w := selectionWorld(t, retreatIntruder(2, 24, 20), retreatIntruder(3, 20, 17))
	if got := w.reacquisitionVictim(0); got != 1 {
		t.Fatalf("farther aligned survivor lost to distance: %d", got)
	}
	w = selectionWorld(t, retreatIntruder(2, 20, 18), retreatIntruder(3, 20, 22))
	if got := w.reacquisitionVictim(0); got != 2 {
		t.Fatalf("equal turn scores chose %d, want later survivor 2", got)
	}
}

func TestReacquisitionReturnAppendsAfterSurvivors(t *testing.T) {
	w := selectionWorld(t, retreatIntruder(2, 20, 18), retreatIntruder(3, 20, 22))
	w.takeOffMap(1)
	if !w.returnToMap(1) {
		t.Fatal("return fixture refused its retained cell")
	}
	if got := w.reacquisitionVictim(0); got != 1 {
		t.Fatalf("returned actor did not follow its survivor: %d", got)
	}
	if w.entities[1].ID != 2 || w.entities[2].ID != 3 {
		t.Fatal("identity storage was reordered")
	}
}

func TestRetreatPolicyWritesFleeBehindLoadedCycle(t *testing.T) {
	w := selectionWorld(t, retreatIntruder(2, 21, 20))
	w.orderAttack(0, 2)
	w.entities[0].ActorState = actorStateRetreat
	w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCharging, 1
	w.armRetreat(0)
	if e := w.entities[0]; !e.Retreat.Pending || e.Retreat.X != 17 || e.Retreat.Y != 20 {
		t.Fatalf("loaded policy omitted pending flee: %+v", e.Retreat)
	}
	if e := w.entities[0]; e.AttackPhase != AttackCharging || e.AttackTarget != 2 {
		t.Fatal("policy changed loaded cycle")
	}
}

func TestReacquisitionDirectionMatchesControlledCentres(t *testing.T) {
	want := [9][9]uint8{
		{224, 224, 224, 0, 0, 0, 32, 32, 32},
		{224, 224, 224, 0, 0, 0, 32, 32, 32},
		{224, 224, 224, 224, 0, 32, 32, 32, 32},
		{192, 192, 224, 224, 0, 32, 32, 64, 64},
		{192, 192, 192, 192, 224, 64, 64, 64, 64},
		{192, 192, 160, 160, 128, 96, 96, 64, 64},
		{160, 160, 160, 160, 128, 96, 96, 96, 96},
		{160, 160, 160, 128, 128, 128, 96, 96, 96},
		{160, 160, 160, 128, 128, 128, 96, 96, 96},
	}
	for dy := int32(-4); dy <= 4; dy++ {
		for dx := int32(-4); dx <= 4; dx++ {
			if got := headingOf(int64(dx)*256, int64(dy)*256); got != want[dy+4][dx+4] {
				t.Errorf("delta %d/%d: %d, want %d", dx, dy, got, want[dy+4][dx+4])
			}
		}
	}
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			d := a - b
			if d < 0 {
				d = -d
			}
			if d > 128 {
				d = 256 - d
			}
			if got := facingArc(uint8(a), uint8(b)); got != int32(d) {
				t.Fatalf("turn %d/%d: %d, want %d", a, b, got, d)
			}
		}
	}
}
