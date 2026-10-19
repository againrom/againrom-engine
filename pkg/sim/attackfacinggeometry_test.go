package sim

import "testing"

func TestAttackFacingAcquisitionPaths(t *testing.T) {
	for _, acquire := range []bool{false, true} {
		w := pursuitEscapeWorld(t)
		w.entities[0].Reach = 1
		if acquire {
			w.orderAcquire(0, 2)
		} else {
			w.orderAttack(0, 2)
		}
		// The admitted victim leaves reach before the progress-zero gate.
		w.entities[1].X = 24
		scratch := newRouteScratch(w)
		got := w.approach(scratch, 0)
		if acquire {
			if got != -1 || w.entities[0].HasTarget || w.entities[0].Facing != 64 {
				t.Fatalf("standing gate: next=%d facing=%d", got, w.entities[0].Facing)
			}
		} else if got != 1 || w.entities[0].Facing != 192 {
			t.Fatalf("engage gate: next=%d facing=%d", got, w.entities[0].Facing)
		}
	}
}

func TestAttackFacingSuppressionUsesLiveProgressZeroInputs(t *testing.T) {
	for _, progress := range []AttackPhase{AttackReady, AttackCharging} {
		for _, reach := range []uint8{1, 2} {
			w := pursuitEscapeWorld(t)
			w.entities[0].MaxMana = 10
			installAcquireControl(t, w, acquireControlCase{owner: 2, present: true, word: 0})
			w.orderAcquire(0, 2)
			w.entities[0].Reach = reach
			w.entities[0].AttackPhase = progress
			w.entities[0].AttackCountdown = 5
			if progress == AttackReady {
				w.entities[0].AttackCountdown = 0
			}
			scratch := newRouteScratch(w)
			w.approach(scratch, 0)
			w.advanceAttack(0, &castObs{})
			e := w.entities[0]
			if progress == AttackReady && reach == 1 {
				if e.Facing != 192 || e.AttackPhase != AttackReady || !e.PursuitIdle {
					t.Fatalf("suppressed facing=%d phase=%d idle=%v", e.Facing, e.AttackPhase, e.PursuitIdle)
				}
				cold := worldRoundTripForTest(t, w)
				if cold.Hash() != w.Hash() || !cold.entities[0].PursuitIdle || cold.entities[0].Facing != 192 {
					t.Fatal("suppressed state changed on cold decode")
				}

			} else if progress == AttackReady {
				if e.Facing != 64 || e.PursuitIdle {
					t.Fatalf("reach control facing=%d idle=%v", e.Facing, e.PursuitIdle)
				}
			} else if e.Facing != 192 || e.AttackCountdown != 4 {
				t.Fatalf("active facing=%d count=%d", e.Facing, e.AttackCountdown)
			}
		}
	}
}

func TestAttackFacingUnitShotRawOriginAndCentreAim(t *testing.T) {
	for _, size := range []uint8{1, 3} {
		pair := shotPair(4)
		pair[0].TokenSize = size
		pair[1].TokenSize = 3
		w := shotWorld(t, pair...)
		var origin [2]int32
		if !w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 3, Phases: 4, OffsetX: 16, OffsetY: -24, PreMove: func(x, y int32) { origin = [2]int32{x, y} }}) {
			t.Fatal("shot refused")
		}
		p, ok := savedProjectileOf(w, 0)
		if !ok || origin != [2]int32{400, 1128} || p.ActionX != 1664 || p.ActionY != 1408 {
			t.Fatalf("size %d: origin=%v record=%+v", size, origin, p)
		}
		pair[1].TokenSize = 1
		control := shotWorld(t, pair...)
		control.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 3, Phases: 4, OffsetX: 16, OffsetY: -24})
		q, _ := savedProjectileOf(control, 0)
		if q.ActionX == p.ActionX || q.ActionY == p.ActionY {
			t.Fatal("target-size loss control did not move aim")
		}
	}
}

func TestAttackFacingActorSpellCentreOriginAndAim(t *testing.T) {
	for _, homes := range []bool{true, false} {
		pair := shotPair(4)
		pair[0].Class = 14
		pair[0].TokenSize = 3
		pair[1].TokenSize = 3
		w := shotWorld(t, pair...)
		offsets := make([]int32, 16)
		for i := 0; i < 16; i += 2 {
			offsets[i] = 12
			offsets[i+1] = 17
		}
		w.SetFlightInputs(FlightInputs{Pictures: map[int32]FlightPicture{10: {Present: true, Phases: 4, Homes: homes, LifeTicks: 8}}, Classes: map[int32]FlightClass{14: {TileSize: 3, CenterX: 10, CenterY: 20, ShootOffset: offsets}}})
		ev := CastEvent{Caster: 1, Target: 2, Spell: 1, FromX: 1, FromY: 4, ToX: 5, ToY: 4}
		obs := &castObs{}
		w.produceCast(&ev, obs)
		p, ok := savedProjectileOf(w, 0)
		aim := [2]int32{1408, 1152}
		if homes {
			aim = [2]int32{1664, 1408}
		}
		if !ok || len(obs.flightStarts) != 1 || obs.flightStarts[0].X != 656 || obs.flightStarts[0].Y != 1384 || p.ActionX != aim[0] || p.ActionY != aim[1] {
			t.Fatalf("homes %v: starts=%+v record=%+v", homes, obs.flightStarts, p)
		}

		pair[0].TokenSize = 1
		pair[1].TokenSize = 1
		control := shotWorld(t, pair...)
		inputs := *w.flightInputs
		inputs.Classes = map[int32]FlightClass{14: {TileSize: 1, CenterX: 10, CenterY: 20, ShootOffset: offsets}}
		control.SetFlightInputs(inputs)
		lost := &castObs{}
		control.produceCast(&ev, lost)
		q, _ := savedProjectileOf(control, 0)
		if len(lost.flightStarts) != 1 || lost.flightStarts[0].X != 400 || lost.flightStarts[0].Y != 1128 || q.ActionX != 1408 || q.ActionY != 1152 {
			t.Fatalf("footprint loss control: starts=%+v record=%+v", lost.flightStarts, q)
		}
	}
}

func TestAttackFacingStructureUsesAnchorRecord(t *testing.T) {
	w := fineWorld(map[EntityID][2]uint8{1: {128, 128}})
	e := Entity{ID: 1, X: 10, Y: 10, TokenSize: 1}
	s := Structure{Col: 11, Row: 9, Width: 9, Height: 9}
	if h := w.structureHeading(e, s); h != 32 {
		t.Fatalf("anchor heading=%d", h)
	}
	s.Col, s.Row = 15, 13
	if h := w.structureHeading(e, s); h == 32 {
		t.Fatal("rectangle-centre loss control retained anchor heading")
	}
}

func TestAttackFacingSecondGameKeepsAimAndAcquisition(t *testing.T) {
	w := pursuitEscapeWorld(t)
	asSecondGame(t, w)
	w.entities[0].MaxMana = 10
	installAcquireControl(t, w, acquireControlCase{owner: 2, present: true, word: 0})
	w.orderAcquire(0, 2)
	if w.suppressAcquisition(0) || !w.entities[0].AcquirePursuit {
		t.Fatal("second-game acquisition changed")
	}
	pair := shotPair(4)
	pair[1].TokenSize = 3
	w = shotWorld(t, pair...)
	asSecondGame(t, w)
	if x, y := w.projectileAimPoint(w.entities[1]); x != 1408 || y != 1152 {
		t.Fatalf("second-game aim %d,%d", x, y)
	}
	if _, ok := w.ReleaseCast(CastRecord{Caster: 1, Picture: 10, Target: 2, HasTarget: true, X: 384, Y: 1152, Segments: 8}); !ok {
		t.Fatal("second-game cast refused")
	}
	p, _ := savedProjectileOf(w, 0)
	if p.ActionX != 1408 || p.ActionY != 1152 {
		t.Fatalf("second-game cast aim %d,%d", p.ActionX, p.ActionY)
	}
}
