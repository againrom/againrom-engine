package terrain_test

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// sel is one expected answer: the frame and its mirror bit.
type sel struct {
	frame  int
	mirror bool
}

// flip0Mover is a Flip-0 class (S 16, D 8) with both cycles: MB 1, MV 2,
// AT 1, DY 1, BN 0, ID 1.
//
//	MoveBase   = 16
//	AttackBase = 16 + 8*(1+2)       = 40
//	DyingBase  = 40 + 8*1           = 48
//	TailBase   = 48 + 8*1           = 56
//	Total      = 16 + 8*(1+2+1+1+1) = 64
//
// Move track [0,1] (period 2), idle track [0] (period 1).
func flip0Mover() terrain.UnitAnim {
	return terrain.UnitAnim{
		S: 16, D: 8,
		MoveBase: 16, AttackBase: 40, DyingBase: 48, TailBase: 56,
		MoveSlot: 3, MoveWind: 1, IdleSlot: 1,
		Total:     64,
		MoveTrack: []int{0, 1}, IdleTrack: []int{0},
		MoveOK: true, IdleOK: true,
	}
}

// flip1Mover is the spec's I/O example class (Flip 1 → S 9, D 5; MB 1, MV 2,
// AT 2, DY 2, BN 2, ID 0): bases 9/24/34/44, move slot 3, wind 1, total 54.
// Move Time=[2,1], Frame=[0,1] expands to track [0,0,1], period 3; no idle
// cycle, so its idle selection is the standing one.
func flip1Mover() terrain.UnitAnim {
	return terrain.UnitAnim{
		S: 9, D: 5,
		MoveBase: 9, AttackBase: 24, DyingBase: 34, TailBase: 44,
		MoveSlot: 3, MoveWind: 1, IdleSlot: 0,
		Total:     54,
		MoveTrack: []int{0, 0, 1},
		MoveOK:    true,
	}
}

// flip1Idler is a Flip-1 class with an idle cycle and a FAILED move gate
// (MV positive but an empty pair): MB 1, MV 1, AT 0, DY 0, BN 0, ID 2.
//
//	TailBase = 9 + 5*(1+1+0+0)   = 19
//	Total    = 9 + 5*(1+1+0+0+2) = 29
//
// Idle track [0,1], period 2.
func flip1Idler() terrain.UnitAnim {
	return terrain.UnitAnim{
		S: 9, D: 5,
		MoveBase: 9, AttackBase: 19, DyingBase: 19, TailBase: 19,
		MoveSlot: 2, MoveWind: 1, IdleSlot: 2,
		Total:     29,
		IdleTrack: []int{0, 1},
		IdleOK:    true,
	}
}

func TestSelectUnitFrameMovingFlip0(t *testing.T) {
	a := flip0Mover()
	want := [8]sel{
		{17, false}, {20, false}, {23, false}, {26, false},
		{29, false}, {32, false}, {35, false}, {38, false},
	}
	for oct := 0; oct < 8; oct++ {
		f, m := terrain.SelectUnitFrame(a, 64, true, oct, 0, 0)
		if f != want[oct].frame || m != want[oct].mirror {
			t.Errorf("moving flip0 oct %d = (%d, %t), want (%d, %t)",
				oct, f, m, want[oct].frame, want[oct].mirror)
		}
	}
}

// TestSelectUnitFrameIdleFlip0 covers AC-3's idle-cycle row at (S 16, D 8):
// index = TailBase + slot*IdleSlot + idleTrack[step] = 56 + oct*1 + 0 at
// tick 0, every octant plain.
func TestSelectUnitFrameIdleFlip0(t *testing.T) {
	a := flip0Mover()
	want := [8]sel{
		{56, false}, {57, false}, {58, false}, {59, false},
		{60, false}, {61, false}, {62, false}, {63, false},
	}
	for oct := 0; oct < 8; oct++ {
		f, m := terrain.SelectUnitFrame(a, 64, false, oct, 0, 0)
		if f != want[oct].frame || m != want[oct].mirror {
			t.Errorf("idle flip0 oct %d = (%d, %t), want (%d, %t)",
				oct, f, m, want[oct].frame, want[oct].mirror)
		}
	}
}

func TestSelectUnitFrameMovingFlip1(t *testing.T) {
	a := flip1Mover()
	want := [8]sel{
		{10, false}, {13, false}, {16, false}, {19, false},
		{22, false}, // oct 4 → slot 4, plain
		{19, true},  // oct 5 → slot 3, mirrored
		{16, true},  // oct 6 → slot 2, mirrored
		{13, true},  // oct 7 → slot 1, mirrored
	}
	for oct := 0; oct < 8; oct++ {
		f, m := terrain.SelectUnitFrame(a, 54, true, oct, 0, 0)
		if f != want[oct].frame || m != want[oct].mirror {
			t.Errorf("moving flip1 oct %d = (%d, %t), want (%d, %t)",
				oct, f, m, want[oct].frame, want[oct].mirror)
		}
	}

	// 0024's worked I/O example, re-aimed at the odometer: 112 sub-cell units
	// walked is seven sixteenths of a cell, so timeline step 7, step 7 mod 3 =
	// 1 → sub-frame 0; oct 6 → slot 8-6 = 2, mirrored; index = 9 + 2*3 + 1
	// + 0 = 16 → (16, mirror). The tick is passed as 0 beside it to show that
	// the moving arm does not read it.
	if f, m := terrain.SelectUnitFrame(a, 54, true, 6, 0, 112); f != 16 || !m {
		t.Errorf("the spec's worked example = (%d, %t), want (16, true)", f, m)
	}
}

// TestSelectUnitFrameIdleFlip1 covers AC-3's idle-cycle row at (S 9, D 5) —
// the D 5 slot rule through the idle formula, index = 19 + slot*2 + 0 at
// tick 0 — and the mover-failing-its-gate fallback: with MoveOK false, moving
// true takes the SAME idle selection.
func TestSelectUnitFrameIdleFlip1(t *testing.T) {
	a := flip1Idler()
	want := [8]sel{
		{19, false}, {21, false}, {23, false}, {25, false},
		{27, false}, // oct 4 → slot 4, plain
		{25, true},  // oct 5 → slot 3, mirrored
		{23, true},  // oct 6 → slot 2, mirrored
		{21, true},  // oct 7 → slot 1, mirrored
	}
	for oct := 0; oct < 8; oct++ {
		f, m := terrain.SelectUnitFrame(a, 29, false, oct, 0, 0)
		if f != want[oct].frame || m != want[oct].mirror {
			t.Errorf("idle flip1 oct %d = (%d, %t), want (%d, %t)",
				oct, f, m, want[oct].frame, want[oct].mirror)
		}
		// A mover whose gate fails takes its idle selection, unchanged — at the
		// TICK, however far it has walked.
		if f, m := terrain.SelectUnitFrame(a, 29, true, oct, 0, 4096); f != want[oct].frame || m != want[oct].mirror {
			t.Errorf("gate-failed mover oct %d = (%d, %t), want the idle (%d, %t)",
				oct, f, m, want[oct].frame, want[oct].mirror)
		}
	}
}

func TestSelectUnitFrameStanding(t *testing.T) {
	flip0 := terrain.UnitAnim{S: 16, D: 8}
	want0 := [8]sel{
		{0, false}, {2, false}, {4, false}, {6, false},
		{8, false}, {10, false}, {12, false}, {14, false},
	}
	for oct := 0; oct < 8; oct++ {
		f, m := terrain.SelectUnitFrame(flip0, 16, false, oct, 0, 0)
		if f != want0[oct].frame || m != want0[oct].mirror {
			t.Errorf("standing S16 oct %d = (%d, %t), want (%d, %t)",
				oct, f, m, want0[oct].frame, want0[oct].mirror)
		}
	}

	flip1 := terrain.UnitAnim{S: 9, D: 5}
	want1 := [8]sel{
		{0, false}, {2, false}, {4, false}, {6, false},
		{8, false}, // oct 4 → g = 8, plain
		{6, true},  // oct 5 → g = 10 → 16-10 = 6, mirrored
		{4, true},  // oct 6 → g = 12 → 4, mirrored
		{2, true},  // oct 7 → g = 14 → 2, mirrored
	}
	for oct := 0; oct < 8; oct++ {
		f, m := terrain.SelectUnitFrame(flip1, 9, false, oct, 0, 0)
		if f != want1[oct].frame || m != want1[oct].mirror {
			t.Errorf("standing S9 oct %d = (%d, %t), want (%d, %t)",
				oct, f, m, want1[oct].frame, want1[oct].mirror)
		}
	}

	// A mover with no working cycle at all stands: moving true, both gates
	// false, oct 5 at S 16 → the standing (10, plain).
	if f, m := terrain.SelectUnitFrame(flip0, 16, true, 5, 3, 48); f != 10 || m {
		t.Errorf("gateless mover = (%d, %t), want the standing (10, false)", f, m)
	}
}

// walker builds a Flip-0 mover whose whole sheet is one direction's Move
// block, so that at oct 0 the selected index IS the track's own value:
// MoveBase 0, MoveWind 0, slot 0. track is the run-length expansion a
// class's registry keys produce, and its LENGTH is the walk cycle.
func walker(track []int) terrain.UnitAnim {
	return terrain.UnitAnim{
		S: 16, D: 8, MoveSlot: len(track),
		MoveTrack: track, MoveOK: true,
	}
}

func TestSelectUnitFrameWalkRun(t *testing.T) {
	// Two steps per art frame, the shipped shape at both lengths.
	aligned := walker([]int{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7})
	unaligned := walker([]int{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6})

	// One cell at four ticks: shares 64, 64, 64, 64 → 64, 128, 192, 256 →
	// steps 4, 8, 12, 16. The tick is held at 0 throughout, so anything that
	// moved would be reading the wrong clock.
	wantAligned := [4]int{2, 4, 6, 0}   // track[4], track[8], track[12], track[16 mod 16]
	wantUnaligned := [4]int{2, 4, 6, 1} // track[4], track[8], track[12], track[16 mod 14]
	for i, odo := range [4]int{64, 128, 192, 256} {
		if f, m := terrain.SelectUnitFrame(aligned, 16, true, 0, 0, odo); f != wantAligned[i] || m {
			t.Errorf("cycle 16 at %d units = (%d, %t), want (%d, false)", odo, f, m, wantAligned[i])
		}
		if f, m := terrain.SelectUnitFrame(unaligned, 14, true, 0, 0, odo); f != wantUnaligned[i] || m {
			t.Errorf("cycle 14 at %d units = (%d, %t), want (%d, false)", odo, f, m, wantUnaligned[i])
		}
	}

	// Three cell boundaries, walked without stopping.
	for i, odo := range [3]int{256, 512, 768} {
		if f, m := terrain.SelectUnitFrame(aligned, 16, true, 0, 0, odo); f != 0 || m {
			t.Errorf("cycle 16, boundary %d = (%d, %t), want (0, false)", i+1, f, m)
		}
		want := [3]int{1, 2, 3}[i] // steps 2, 4, 6 → track values 1, 2, 3
		if f, m := terrain.SelectUnitFrame(unaligned, 14, true, 0, 0, odo); f != want || m {
			t.Errorf("cycle 14, boundary %d = (%d, %t), want (%d, false)", i+1, f, m, want)
		}
	}
}

// TestSelectUnitFrameGuard covers AC-4's selector half: the guard is the last
// act, so an over-reaching index, a frameCount of zero or below, and a
// negative tick all answer without panicking, and out of range is always
// (0, false) — the mirror bit reset with the frame.
func TestSelectUnitFrameGuard(t *testing.T) {
	// The worked example's raw selection is (16, mirrored); a 16-frame sheet
	// puts index 16 exactly one past its last frame, so the guard answers
	// (0, false), the mirror bit gone with the frame.
	if f, m := terrain.SelectUnitFrame(flip1Mover(), 16, true, 6, 0, 112); f != 0 || m {
		t.Errorf("over-reaching index = (%d, %t), want (0, false)", f, m)
	}

	// frameCount 0 and below: every index is outside [0, frameCount), so the
	// function is total there — (0, false), no panic, the seam's square case.
	if f, m := terrain.SelectUnitFrame(flip0Mover(), 0, true, 3, 5, 80); f != 0 || m {
		t.Errorf("frameCount 0 = (%d, %t), want (0, false)", f, m)
	}
	if f, m := terrain.SelectUnitFrame(terrain.UnitAnim{S: 16, D: 8}, -5, false, 0, 0, 0); f != 0 || m {
		t.Errorf("frameCount -5 = (%d, %t), want (0, false)", f, m)
	}

	// A negative count on the moving arm and a negative tick on the idle one
	// both reduce euclideanly, never panic and never divide by zero. The count
	// rounds DOWN to a step first: -1 unit is step -1, and -1 mod 3 = 2 →
	// sub-frame 1 → 11; -48 units is step -3, and -3 mod 3 = 0 → sub-frame
	// 0 → 10. Over the idler's period-2 idle track [0,1]: -7 mod 2 = 1 →
	// sub-frame 1 → 19 + 1 = 20.
	if f, m := terrain.SelectUnitFrame(flip1Mover(), 54, true, 0, 0, -1); f != 11 || m {
		t.Errorf("count -1 = (%d, %t), want (11, false)", f, m)
	}
	if f, m := terrain.SelectUnitFrame(flip1Mover(), 54, true, 0, 0, -48); f != 10 || m {
		t.Errorf("count -48 = (%d, %t), want (10, false)", f, m)
	}
	if f, m := terrain.SelectUnitFrame(flip1Idler(), 29, false, 0, -7, 0); f != 20 || m {
		t.Errorf("idle tick -7 = (%d, %t), want (20, false)", f, m)
	}
}

func TestSelectUnitFrameEqualInputsEqualAnswers(t *testing.T) {
	cases := []struct {
		name   string
		a      terrain.UnitAnim
		fc     int
		moving bool
		oct    int
		tick   int
		odo    int
	}{
		{"move", flip0Mover(), 64, true, 5, 3, 48},
		{"move mirrored", flip1Mover(), 54, true, 6, 0, 112},
		{"idle", flip1Idler(), 29, false, 7, 11, 0},
		{"standing", terrain.UnitAnim{S: 9, D: 5}, 9, false, 5, -2, -2},
		{"guarded", flip1Mover(), 16, true, 6, 0, 112},
		{"negative count", flip1Mover(), 54, true, 0, 0, -1},
	}
	for _, c := range cases {
		f1, m1 := terrain.SelectUnitFrame(c.a, c.fc, c.moving, c.oct, c.tick, c.odo)
		f2, m2 := terrain.SelectUnitFrame(c.a, c.fc, c.moving, c.oct, c.tick, c.odo)
		if f1 != f2 || m1 != m2 {
			t.Errorf("%s: two identical calls answered (%d, %t) then (%d, %t)", c.name, f1, m1, f2, m2)
		}
	}
}
