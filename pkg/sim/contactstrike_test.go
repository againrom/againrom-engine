package sim

import "testing"

func TestTouchingBodiesUseTheirTokenSizesForPhysicalStrike(t *testing.T) {
	for _, tc := range []struct {
		name           string
		as, ts         uint8
		ax, ay, tx, ty int32
		face           uint8
	}{
		{"ogre south edge", 2, 1, 6, 4, 6, 6, 128},
		{"large target north edge", 1, 2, 6, 6, 6, 4, 0},
		{"ordinary unit", 1, 1, 6, 4, 6, 5, 128},
		{"legacy zero size", 0, 0, 6, 4, 6, 5, 128},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, target := cbEnt(1, tc.ax, tc.ay), cbEnt(2, tc.tx, tc.ty)
			a.TokenSize, target.TokenSize = tc.as, tc.ts
			a.Owner, target.Owner = SelfSlot, 3
			a.Facing, a.Reach = tc.face, 1
			a.AttackCharge, a.AttackRelax = 3, 4
			a.AlwaysHits, a.DamageBase = true, 12
			a.HasAttackTarget, a.AttackTarget, a.ActorState = true, 2, 3
			w := cbWorld(t, 17, a, target)
			before := cbAt(t, w, 2).HP
			Step(w, nil)
			if e := cbAt(t, w, 1); e.AttackPhase != AttackCharging {
				t.Fatalf("touching retained target did not load a charge: %+v", e)
			}
			hurt := false
			for range 12 {
				Step(w, nil)
				hurt = hurt || cbAt(t, w, 2).HP < before
			}
			if !hurt {
				t.Fatal("retained physical strike never changed target HP")
			}
			if e := cbAt(t, w, 1); e.X != tc.ax || e.Y != tc.ay {
				t.Fatal("touching attacker tried to enter the target footprint")
			}
		})
	}
}

func TestPhysicalPursuitWalksToContactWithoutOverlappingTheTarget(t *testing.T) {
	for _, size := range []uint8{1, 2} {
		t.Run(string(rune('0'+size)), func(t *testing.T) {
			a, target := cbEnt(1, 6, 2), cbEnt(2, 6, 6)
			a.TokenSize, target.TokenSize = size, 1
			a.Owner, target.Owner = SelfSlot, 3
			a.Facing, a.Reach, a.Speed = 128, 1, 12
			a.AttackCharge, a.AttackRelax = 3, 4
			a.AlwaysHits, a.DamageBase = true, 12
			w := cbWorld(t, 17, a, target)
			moved, charged, hurt := false, false, false
			for tick := range 512 {
				var commands []Command
				if tick == 0 {
					commands = []Command{Attack(a.ID, target.ID)}
				}
				Step(w, commands)
				e := cbAt(t, w, 1)
				if tick == 0 && e.AttackPhase == AttackCharging {
					t.Fatal("distant retained target charged before approach")
				}
				moved = moved || e.X != a.X || e.Y != a.Y
				charged = charged || e.AttackPhase == AttackCharging
				hurt = hurt || cbAt(t, w, 2).HP < target.HP
				if e.X < target.X+1 && target.X < e.X+int32(size) &&
					e.Y < target.Y+1 && target.Y < e.Y+int32(size) {
					t.Fatal("pursuit entered the target footprint", e.X, e.Y)
				}
				if moved && charged && hurt {
					return
				}
			}
			t.Fatalf("pursuit failed: moved=%v charged=%v hurt=%v actor=%+v", moved, charged, hurt, cbAt(t, w, 1))
		})
	}
}

func TestLargeTouchingAttackerFinishesItsTurnBeforeCharging(t *testing.T) {
	a, target := cbFighter(1, 6, 4, 3, 4), cbEnt(2, 6, 6)
	a.TokenSize, target.TokenSize = 2, 1
	a.Owner, target.Owner = SelfSlot, 3
	a.Facing, a.Reach, a.RotationSpeed = 0, 1, 32
	w := cbWorld(t, 17, a, target)
	sawTurn := false
	for tick := range 64 {
		var commands []Command
		if tick == 0 {
			commands = []Command{Attack(a.ID, target.ID)}
		}
		Step(w, commands)
		e := cbAt(t, w, 1)
		if e.X != a.X || e.Y != a.Y || cbAt(t, w, 2).HP != target.HP {
			t.Fatal("touching turn moved or damaged a body before charging")
		}
		if e.TurnRemaining > 0 {
			sawTurn = true
			if e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
				t.Fatal("turning large attacker loaded a charge", e)
			}
		}
		if e.AttackPhase == AttackCharging {
			if !sawTurn || e.Facing != 128 {
				t.Fatal("large attacker skipped its required turn", e)
			}
			return
		}
	}
	t.Fatal("large touching attacker never completed its turn and charged")
}

func TestLargeBodiesKeepThePhysicalFlightCountdown(t *testing.T) {
	for _, tc := range []struct {
		as, ts uint8
		ty     int32
		blow   int
	}{
		{1, 1, 6, 6},
		{2, 1, 6, 5},
		{1, 2, 5, 5},
	} {
		a, target := cbFighter(1, 6, 2, 1, 4), cbEnt(2, 6, tc.ty)
		a.TokenSize, target.TokenSize = tc.as, tc.ts
		a.Owner, target.Owner = SelfSlot, 3
		a.Facing, a.Reach = 128, 4
		w := cbWorld(t, 17, a, target)
		for tick := 1; tick <= tc.blow; tick++ {
			var commands []Command
			if tick == 1 {
				commands = []Command{Attack(a.ID, target.ID)}
			}
			Step(w, commands)
			hp := cbAt(t, w, 2).HP
			if tick < tc.blow && hp != target.HP || tick == tc.blow && hp >= target.HP {
				t.Fatalf("sizes %d/%d: physical HP=%d at tick=%d, want first blow on %d", tc.as, tc.ts, hp, tick, tc.blow)
			}
			if e := cbAt(t, w, 1); e.X != a.X || e.Y != a.Y {
				t.Fatal("ranged attacker approached a victim already in reach")
			}
		}
	}
}
