package rules

import "testing"

// firstDriveOracle is the first game's driver switch as written before the
// table, kept as an independent statement of the arms.
func firstDriveOracle(id int32) EffectDrive {
	switch id {
	case 34, 36:
		return EffectDrive{EffectPath, FrameRamp}
	case 60:
		return EffectDrive{EffectStill, FrameCounterLess1}
	case 51:
		return EffectDrive{EffectTravel, FrameCounter}
	case 20, 30:
		return EffectDrive{EffectFollow, FrameResolvedOne}
	case 18, 24, 28, 40, 44, 48, 52, 54, 56, 62, 64:
		return EffectDrive{EffectFollow, FrameHalf}
	}
	return EffectDrive{}
}

func TestTheFirstGameKeepsItsDriverArms(t *testing.T) {
	for id := int32(-2); id <= 80; id++ {
		if got, want := EffectDriveOf(SpellGameFirst, id), firstDriveOracle(id); got != want {
			t.Errorf("first game id %d: %+v, want %+v", id, got, want)
		}
	}
}

// secondDriveClaims is R2-ENGINE-447's update switch; 56 and 60 take the
// half clock (DIV-3101).
var secondDriveClaims = map[int32]EffectDrive{
	23: {EffectStill, FrameCounter},
	28: {EffectPath, FrameRamp}, 30: {EffectPath, FrameRamp},
	54: {EffectStill, FrameCounterLess1},
	16: {EffectFollow, FrameCounter}, 20: {EffectFollow, FrameCounter}, 24: {EffectFollow, FrameCounter},
	34: {EffectFollow, FrameCounter}, 44: {EffectFollow, FrameCounter}, 46: {EffectFollow, FrameCounter},
	48: {EffectFollow, FrameCounter}, 50: {EffectFollow, FrameCounter}, 62: {EffectFollow, FrameCounter},
	64: {EffectFollow, FrameCounter}, 66: {EffectFollow, FrameCounter},
	56: {EffectFollow, FrameHalf}, 60: {EffectFollow, FrameHalf},
	10: {}, 12: {}, 13: {}, 18: {}, 40: {},
}

func secondDriveMismatches(shift int32) int {
	n := 0
	for id, want := range secondDriveClaims {
		if EffectDriveOf(SpellGameSecond, id+shift) != want {
			n++
		}
	}
	return n
}

func TestTheSecondGameTakesItsOwnDriverArms(t *testing.T) {
	if n := secondDriveMismatches(0); n != 0 {
		t.Fatalf("%d identifiers miss R2-ENGINE-447", n)
	}
	// Loss control: the arms read one spell row off (identifier + 2) fail.
	if n := secondDriveMismatches(2); n == 0 {
		t.Fatal("a shifted identifier still matches every arm")
	}
}

func TestRampAndCounterFrames(t *testing.T) {
	want := []int32{4, 3, 2, 1, 0, 1, 2, 1, 0, 1, 2, 3, 4}
	for i, w := range want {
		if got, ok := EffectFrameAt(FrameRamp, int32(i+1), 35, false); !ok || got != w {
			t.Errorf("ramp counter %d: %d %v, want %d", i+1, got, ok, w)
		}
	}
	for _, c := range []int32{0, 14} {
		if _, ok := EffectFrameAt(FrameRamp, c, 35, false); ok {
			t.Errorf("ramp counter %d writes a frame", c)
		}
	}
	for _, tc := range []struct {
		frame    EffectFrame
		counter  int32
		phases   uint16
		resolved bool
		want     int32
	}{
		{FrameHalf, 9, 4, false, 0}, {FrameHalf, 7, 4, false, 3}, {FrameHalf, 7, 0, false, 0},
		{FrameCounter, 9, 9, false, 9}, {FrameCounterLess1, 9, 21, false, 8},
		{FrameResolvedOne, 7, 4, true, 1}, {FrameResolvedOne, 7, 4, false, 3},
	} {
		if got, _ := EffectFrameAt(tc.frame, tc.counter, tc.phases, tc.resolved); got != tc.want {
			t.Errorf("%+v: %d", tc, got)
		}
	}
}
