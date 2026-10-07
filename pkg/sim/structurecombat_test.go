package sim

import (
	"bytes"
	"testing"
)

func structureCombatWorld(t *testing.T, seed uint64, a Entity, s Structure) *World {
	t.Helper()
	w, err := NewStructuredWorld(seed, Bounds{Width: 16, Height: 16}, ModeCanonical,
		Terrain{}, []Entity{a}, nil, Relations{}, nil, nil, nil, GhostTemplate{}, []Structure{s})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func structureCombatActor() Entity {
	return Entity{ID: 0, X: 2, Y: 3, HP: 100, MaxHP: 100, Reach: 1, Facing: 64,
		AttackCharge: 1, AttackRelax: 2, Speed: 50, DyingTime: 200,
		DamageBase: 999, DamageSpread: 999, ToHit: -999,
		SecondaryDamage: SecondaryDamage{Base: 20, Spread: 1}}
}

func structureCombatTarget() Structure {
	return Structure{ID: 0, Col: 3, Row: 3, Field42: 100, MaxHealth: 100, Width: 1, Height: 1, Attach: 1, Blocking: 1}
}

func TestStructurePhysicalDamageAndDrawBoundaries(t *testing.T) {
	// Seed zero's first SplitMix word is e220a8397b1dcdaf. Multiplication
	// reduction gives 1 at bound 1 and 226 at bound 255. Expected values below
	// are literal arithmetic, not calls to the production damage/RNG functions.
	for _, tc := range []struct {
		name          string
		base, spread  uint8
		max, hp, want uint16
		far           bool
		draws         bool
	}{
		{"ordinary pair ignored", 0, 0, 100, 100, 100, false, false},
		{"base suppressed by zero spread", 20, 0, 100, 100, 100, false, false},
		{"zero maximum", 20, 1, 0, 100, 100, false, false},
		{"below armour constant", 3, 1, 100, 100, 100, false, true},
		{"equal armour constant", 4, 1, 100, 100, 100, false, true},
		{"positive", 20, 1, 100, 100, 84, false, true},
		{"byte maximum", 255, 255, 1000, 1000, 524, false, true},
		{"word subtraction not effect clamp", 20, 1, 10, 10, 65530, false, true},
		{"out of reach", 20, 1, 100, 100, 100, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, s := structureCombatActor(), structureCombatTarget()
			a.SecondaryDamage.Base, a.SecondaryDamage.Spread = tc.base, tc.spread
			s.Field42, s.MaxHealth = tc.hp, tc.max
			if tc.far {
				s.Col = 5
			}
			w := structureCombatWorld(t, 0, a, s)
			w.orderAttack(0, 0, AttackTargetStructure)
			w.resolveStructureBlow(0)
			if w.structures[0].Field42 != tc.want {
				t.Fatalf("HP=%d want %d", w.structures[0].Field42, tc.want)
			}
			wantRNG := uint64(0)
			if tc.draws {
				wantRNG = 0x9e3779b97f4a7c15
			}
			if w.rng.state != wantRNG {
				t.Fatalf("RNG=%x want %x", w.rng.state, wantRNG)
			}
			if w.entities[0].HP != 100 {
				t.Fatal("structure zero damaged unit zero")
			}
		})
	}
}

func TestStructureAttackApproachesRepeatsStopsAndRestores(t *testing.T) {
	a, s := structureCombatActor(), structureCombatTarget()
	a.AttackCharge, a.Owner = 4, SelfSlot
	s.Col, s.Width, s.Height = 8, 3, 2 // rectangle must not enlarge combat reach
	w := structureCombatWorld(t, 0, a, s)
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 0, X: 0}})
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTargetKind != AttackTargetStructure || !e.HasTarget {
		t.Fatalf("no structure pursuit: %+v", e)
	}
	approachSaved, strikeSaved, hits := false, false, 0
	previous := s.Field42
	for n := 0; n < 1000; n++ {
		e := w.entities[0]
		if (!approachSaved && e.HasTarget) || (!strikeSaved && e.AttackPhase == AttackCharging && e.AttackCountdown > 0) {
			b, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var restored World
			if err := restored.UnmarshalBinary(b); err != nil {
				t.Fatal(err)
			}
			for k := 0; k < 35; k++ {
				Step(w, nil)
				Step(&restored, nil)
				if w.Hash() != restored.Hash() {
					t.Fatalf("resume diverged at %d/%d", n, k)
				}
			}
			if e.HasTarget {
				approachSaved = true
			} else {
				strikeSaved = true
			}
		}
		Step(w, nil)
		if hp := w.structures[0].Field42; hp != previous {
			hits++
			previous = hp
		}
		if int16(w.structures[0].Field42) <= 0 {
			break
		}
	}
	if !approachSaved || !strikeSaved || hits < 2 {
		t.Fatalf("approach save=%v strike save=%v hits=%d", approachSaved, strikeSaved, hits)
	}
	if int16(w.structures[0].Field42) > 0 || w.entities[0].HasAttackTarget || w.entities[0].AttackTargetKind != AttackTargetUnit || w.entities[0].HasTarget {
		t.Fatalf("not cleanly stopped: HP=%d actor=%+v", int16(w.structures[0].Field42), w.entities[0])
	}
	hp, rng := w.structures[0].Field42, w.rng.state
	for n := 0; n < 100; n++ {
		Step(w, nil)
	}
	if w.structures[0].Field42 != hp || w.rng.state != rng {
		t.Fatal("ruin attacked again")
	}
}

func TestStructureAttackTaggedIdentityAndReplacement(t *testing.T) {
	a, s := structureCombatActor(), structureCombatTarget()
	a.AttackCharge = 5
	w := structureCombatWorld(t, 0, a, s)
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 0, X: 0}})
	b, _ := w.MarshalBinary()
	// 34-byte header, 3*16*16 terrain bytes, attack tag at record+48.
	if b[0] != formatVersion || b[34+3*16*16+48] != 2 {
		t.Fatal("wrong version/structure tag")
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	out, _ := back.MarshalBinary()
	if !bytes.Equal(out, b) || back.entities[0].AttackTargetKind != AttackTargetStructure {
		t.Fatal("target identity lost")
	}
	for _, bad := range []byte{3, 255} {
		corrupt := append([]byte(nil), b...)
		corrupt[34+3*16*16+48] = bad
		if err := back.UnmarshalBinary(corrupt); err == nil {
			t.Fatalf("accepted tag %d", bad)
		}
	}
	missing := append([]byte(nil), b...)
	missing[34+3*16*16+44] = 99
	if err := back.UnmarshalBinary(missing); err == nil {
		t.Fatal("accepted absent structure target")
	}
	remaining := w.entities[0].AttackCountdown
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 0, X: 0}})
	if w.entities[0].AttackCountdown != remaining-1 {
		t.Fatal("repeat order reset wind-up")
	}
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 0, X: 99}})
	if w.entities[0].AttackTarget != 0 || !w.entities[0].HasAttackTarget {
		t.Fatal("missing structure replaced order")
	}
	Step(w, []Command{{Kind: KindMoveTo, Entity: 0, X: 1, Y: 1}})
	// The charge already loaded on the structure resolves first: the move waits
	// behind it (DIV-1563) and replaces the order once the cycle has ended.
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTargetKind != AttackTargetStructure || !e.HasTarget {
		t.Fatal("move discarded a loaded structure attack")
	}
	for n := 0; n < 200 && w.entities[0].HasAttackTarget; n++ {
		Step(w, nil)
	}
	if w.entities[0].HasAttackTarget || w.entities[0].AttackTargetKind != AttackTargetUnit {
		t.Fatal("move retained structure order")
	}
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 0, X: 0}, {Kind: KindKill, Entity: 0}})
	if w.entities[0].HasAttackTarget || w.entities[0].AttackTargetKind != AttackTargetUnit {
		t.Fatal("dead attacker retained order")
	}
}

func TestStructureTargetSurvivesSameIDUnitDeathAndRemoval(t *testing.T) {
	a, s := structureCombatActor(), structureCombatTarget()
	a.ID, a.AttackCharge = 1, 10
	other := structureCombatActor()
	other.X, other.Y = 10, 10
	w, err := NewStructuredWorld(0, Bounds{Width: 16, Height: 16}, ModeCanonical,
		Terrain{}, []Entity{a, other}, nil, Relations{}, nil, nil, nil, GhostTemplate{}, []Structure{s})
	if err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 1, X: 0}, {Kind: KindKill, Entity: 0}})
	if e := w.entities[indexOfEntity(w.entities, 1)]; !e.HasAttackTarget || e.AttackTargetKind != AttackTargetStructure {
		t.Fatal("unit zero death cleared structure zero target")
	}
	w.remove([]EntityID{0})
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTargetKind != AttackTargetStructure {
		t.Fatal("unit zero removal cleared structure zero target")
	}
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 0}})
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTargetKind != AttackTargetStructure {
		t.Fatal("absent unit zero order aliased structure zero")
	}
}

func TestStructureAttackUsesSharedRangedCadence(t *testing.T) {
	a, s := structureCombatActor(), structureCombatTarget()
	a.Reach, a.AttackCharge, a.AttackRelax = 5, 2, 1
	s.Col, s.Field42, s.MaxHealth = 5, 1000, 1000
	w := structureCombatWorld(t, 0, a, s)
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 0, X: 0}})
	// Distance 3: (3*256+128)/200 = 4 flight ticks, plus charge 2.
	if e := w.entities[0]; e.AttackPhase != AttackCharging || e.AttackCountdown != 5 {
		t.Fatalf("charge=%+v", e)
	}
	for n := 0; n < 4; n++ {
		Step(w, nil)
	}
	if w.structures[0].Field42 != 1000 {
		t.Fatal("early ranged damage")
	}
	Step(w, nil)
	if w.structures[0].Field42 != 984 || w.rng.state != 0x3c6ef372fe94f82a {
		t.Fatalf("HP=%d RNG=%x", w.structures[0].Field42, w.rng.state)
	}
	// Recovery=1+second draw 1, two scheduler boundaries, next charge 6.
	for n := 0; n < 9; n++ {
		Step(w, nil)
	}
	if w.structures[0].Field42 != 984 {
		t.Fatal("repeat fired before cadence")
	}
	Step(w, nil)
	if w.structures[0].Field42 >= 984 {
		t.Fatal("retained attack did not repeat")
	}
}

// An attack given while the unit still crosses into its move destination
// outlives the arrival. With saved groups present, the move leaves the unit
// in a saved group held at Move, whose arrival arm used to end the attack.
func TestStructureAttackDuringMoveOutlivesArrival(t *testing.T) {
	for _, saved := range []bool{false, true} {
		a, s := structureCombatActor(), structureCombatTarget()
		a.Owner, a.AttackCharge, a.X, a.Y = SelfSlot, 4, 2, 8
		s.Col, s.Row = 6, 3
		w := structureCombatWorld(t, 0, a, s)
		if saved {
			w.savedGroups = &savedGroupState{}
		}
		Step(w, []Command{MoveTo(0, CellPoint{X: 6, Y: 4})})
		for n := 0; n < 100 && !(w.entities[0].Y == 4 && w.entities[0].Transit > 0); n++ {
			Step(w, nil)
		}
		if e := w.entities[0]; e.X != 6 || e.Y != 4 || e.Transit == 0 {
			t.Fatalf("saved=%v: not crossing into the destination: %+v", saved, e)
		}
		Step(w, []Command{AttackStructure(0, 0)})
		for range 60 {
			Step(w, nil)
		}
		if e := w.entities[0]; !e.HasAttackTarget || w.structures[0].Field42 > 54 {
			t.Fatalf("saved=%v: attack held=%v HP=%d", saved, e.HasAttackTarget, w.structures[0].Field42)
		}
		if g := w.savedGroupFor(0); saved && (g == nil || g.AI[0x20] != orderNone) {
			t.Fatalf("saved group order not ended: %+v", g)
		}
	}
}
