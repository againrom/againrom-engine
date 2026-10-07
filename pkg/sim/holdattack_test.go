package sim

import "testing"

func holdReachFixture(t *testing.T, reach uint8, dist int32, hostileAttacks, saved bool) *World {
	t.Helper()
	hero := withdrawalFighter(1, SelfSlot, 20, 20, 100)
	hero.Reach = reach
	foe := withdrawalFighter(2, 3, 20+dist, 20, 100)
	foe.Reach = reach
	cells := [][3]uint32{{SelfSlot, 3, 1}, {3, SelfSlot, 1}}
	if !hostileAttacks {
		cells[1] = [3]uint32{3, SelfSlot, 2}
	}
	w := engWorld(t, engRel(t, cells...), hero, foe)
	if saved {
		savedTacticalRegistry(t, w, true)
	}
	Step(w, []Command{GroupStance(1, OrderStandGround, 2)})
	return w
}

func TestHoldPositionAttacksWhatStandsInReach(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		reach uint8
		dist  int32
		foes  bool
		saved bool
	}{
		{"melee, passive hostile", 1, 1, false, false},
		{"melee, attacking hostile", 1, 1, true, false},
		{"ranged, passive hostile", 4, 4, false, false},
		{"ranged, attacking hostile", 4, 4, true, false},
		{"saved group melee, passive", 1, 1, false, true},
		{"saved group melee, attacking", 1, 1, true, true},
		{"saved group ranged, attacking", 4, 4, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := holdReachFixture(t, tc.reach, tc.dist, tc.foes, tc.saved)
			for range 400 {
				Step(w, nil)
				if e := entityAt(t, w, 1); e.X != 20 || e.Y != 20 {
					t.Fatalf("the member left its cell: (%d,%d)", e.X, e.Y)
				}
			}
			foe := entityAt(t, w, 2)
			if foe.HP >= 100 {
				t.Errorf("the hostile at distance %d took no damage (HP %d): the member never struck", tc.dist, foe.HP)
			}
		})
	}
}

func TestHoldPositionDoesNotApproachAHostilePastReach(t *testing.T) {
	t.Parallel()
	w := holdReachFixture(t, 1, 3, false, false)
	for range 400 {
		Step(w, nil)
	}
	if e := entityAt(t, w, 1); e.X != 20 || e.Y != 20 || e.HasAttackTarget {
		t.Fatalf("the member moved or took a victim: (%d,%d) victim %v", e.X, e.Y, e.HasAttackTarget)
	}
	if foe := entityAt(t, w, 2); foe.HP != 100 {
		t.Fatalf("the hostile past reach took damage: HP %d", foe.HP)
	}
}

func TestHoldPositionAttacksAWideBodyThatTouchesIt(t *testing.T) {
	t.Parallel()
	for _, size := range []uint8{2, 3} {
		for dx := int32(-4); dx <= 4; dx++ {
			for dy := int32(-4); dy <= 4; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				hero := withdrawalFighter(1, SelfSlot, 30, 30, 1000)
				foe := withdrawalFighter(2, 3, 30+dx, 30+dy, 1000)
				foe.TokenSize = size
				w := engWorld(t, engRel(t, [3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 2}), hero, foe)
				touches := strikeDistance(w.entities[0], w.entities[1]) <= 1
				Step(w, []Command{GroupStance(1, OrderStandGround, 2)})
				for range 100 {
					Step(w, nil)
				}
				struck := entityAt(t, w, 2).HP < 1000
				if struck != touches {
					t.Errorf("size %d foe at offset (%d,%d): body touches %v, member struck %v", size, dx, dy, touches, struck)
				}
				if e := entityAt(t, w, 1); e.X != 30 || e.Y != 30 {
					t.Errorf("size %d foe at offset (%d,%d): the member left its cell for (%d,%d)", size, dx, dy, e.X, e.Y)
				}
			}
		}
	}
}
