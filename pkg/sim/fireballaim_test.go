package sim

import "testing"

// aimedFireBall answers the transport countdown of a Fire_Ball from cell (1,1)
// at a target in cell (5,1) whose stride puts it 68 units west of the centre.
// By SAV-1155: caster point 384, target point 1084, distance 700 over speed
// 384 is 1; the cell centre is 1024 away, 2.
func aimedFireBall(t *testing.T, size uint8, aimed bool, speed int32) uint16 {
	t.Helper()
	w, _ := burstWorld(t)
	w.spells[0].EffectSpeed = speed
	w.spells[0].Area, w.spells[0].TargetsUnit, w.spells[0].Distribution, w.spells[0].Radius = true, false, distributionDiamond, 1
	w.entities[1].TokenSize = size
	w.entities[1].Transit, w.entities[1].TransitTotal = 4, 8
	w.entities[1].Stride = NativeStride{Present: true, FromX: 4, FromY: 1, ToX: 5, ToY: 1, StepX: -17}
	if aimed {
		if !w.castSpell(0, 2, fireBallSpell, 5, 1, nil) {
			t.Fatal("cast refused")
		}
	} else if !w.landArea(w.spells[0], 0, 1, true, 1, 1, 5, 1, nil) {
		t.Fatal("area preparation")
	}
	if len(w.deliveries) != 1 {
		t.Fatalf("%d deliveries", len(w.deliveries))
	}
	return w.deliveries[0].Remaining
}

// A Fire_Ball aimed at a one-cell actor times its transport to the actor's
// point, fine bytes included (SAV-1155). An actor wider than one cell is aimed
// at its footprint centre less one sub-unit (MAGIC-245): at speed 300 that
// point, 836 away, gives 2 where the aimed cell's centre, 1024 away, gives 3.
// A cast at a cell keeps the cell centre.
func TestFireBallTransportAimsAtASingleCellActorsFinePosition(t *testing.T) {
	for _, tc := range []struct {
		name  string
		size  uint8
		aimed bool
		speed int32
		want  uint16
	}{
		{"one-cell actor", 1, true, 384, 1},
		{"size zero is one cell", 0, true, 384, 1},
		{"two-cell actor, centre less one", 2, true, 300, 2},
		{"two-cell actor at the cell centre's speed", 2, true, 384, 2},
		{"cast at a cell", 1, false, 384, 2},
		{"cast at a cell, slow", 1, false, 300, 3},
	} {
		if got := aimedFireBall(t, tc.size, tc.aimed, tc.speed); got != tc.want {
			t.Errorf("%s: countdown %d, want %d", tc.name, got, tc.want)
		}
	}
}

// The countdown a two-cell aim wrote (3 at speed 380; the cell centre would give 2) is the countdown a cold LOAD continues.
func TestFireBallTransportCountdownOfAWideAimSurvivesSaveAndLoad(t *testing.T) {
	w, _ := burstWorld(t)
	w.spells[0].EffectSpeed = 380
	w.spells[0].Area, w.spells[0].TargetsUnit, w.spells[0].Distribution, w.spells[0].Radius = true, false, distributionDiamond, 1
	w.entities[1].TokenSize = 2
	if !w.castSpell(0, 2, fireBallSpell, 5, 1, nil) || len(w.deliveries) != 1 {
		t.Fatal("cast refused")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if len(cold.deliveries) != 1 || cold.deliveries[0].Remaining != w.deliveries[0].Remaining || w.deliveries[0].Remaining != 3 {
		t.Errorf("source countdown %d, loaded deliveries %+v, want 3 for both", w.deliveries[0].Remaining, cold.deliveries)
	}
}

// The weapon-borne route (ANIM-115) times its transport the same way; at
// speed 300 the two-cell actor's centre point gives 2 and its cell centre 3.
func TestWeaponBorneFireBallAimsAtASingleCellActorsFinePosition(t *testing.T) {
	for size, want := range map[uint8]uint16{1: 2, 2: 2} {
		w, _ := burstWorld(t)
		w.spells[0].EffectSpeed = 300
		w.spells[0].Area, w.spells[0].TargetsUnit, w.spells[0].Distribution, w.spells[0].Radius = true, false, distributionDiamond, 1
		w.entities[1].TokenSize = size
		w.entities[1].Transit, w.entities[1].TransitTotal = 4, 8
		w.entities[1].Stride = NativeStride{Present: true, FromX: 4, FromY: 1, ToX: 5, ToY: 1, StepX: -17}
		if !w.weaponSpellApply(0, 1, w.spells[0], 0, nil) || len(w.deliveries) != 1 {
			t.Fatal("rider refused")
		}
		if got := w.deliveries[0].Remaining; got != want {
			t.Errorf("size %d: countdown %d, want %d", size, got, want)
		}
	}
}

// A restored actor's fine position times the transport too: cell (4,1), fine
// x 100 is 740 from the caster over speed 384, 1; the centre is 768 away, 2.
func TestFireBallTransportAimsAtARestoredActorsFinePosition(t *testing.T) {
	for _, tc := range []struct {
		name  string
		fineX uint8
		want  uint16
	}{
		{"fine position", 100, 1},
		{"centred", 128, 2},
	} {
		w, _ := burstWorld(t)
		w.spells[0].EffectSpeed = 384
		w.spells[0].Area, w.spells[0].TargetsUnit, w.spells[0].Distribution, w.spells[0].Radius = true, false, distributionDiamond, 1
		w.entities[1].X = 4
		w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 2, Current: true,
			Position: SavedActorPosition{Cell: 1<<8 | 4, PackedCell: 1<<8 | 4, FineX: tc.fineX, FineY: 128}}}}
		if !w.castSpell(0, 2, fireBallSpell, 4, 1, nil) || len(w.deliveries) != 1 {
			t.Fatal("cast refused")
		}
		if got := w.deliveries[0].Remaining; got != tc.want {
			t.Errorf("%s: countdown %d, want %d", tc.name, got, tc.want)
		}
	}
}
