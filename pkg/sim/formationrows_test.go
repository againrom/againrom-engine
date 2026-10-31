package sim

import (
	"math"
	"testing"
)

func TestFormationRowsKeepEveryAnswer(t *testing.T) {
	forward := map[int32]uint8{0: 0, 1: 2, 2: 1}
	for _, v := range []int32{math.MinInt32, -1, 0, 1, 2, 3, 9, math.MaxInt32} {
		want, ok := forward[v]
		if !ok {
			want = 2
		}
		if got := FormationModeOfSetting(v); got != want {
			t.Errorf("FormationModeOfSetting(%d) = %d, want %d", v, got, want)
		}
	}
	for m := 0; m <= 255; m++ {
		want := int32(2)
		switch m {
		case 0:
			want = 0
		case 2:
			want = 1
		}
		if got := FormationSettingOfMode(uint8(m)); got != want {
			t.Errorf("FormationSettingOfMode(%d) = %d, want %d", m, got, want)
		}
	}
	for s := int32(0); s <= 2; s++ {
		if got := FormationSettingOfMode(FormationModeOfSetting(s)); got != s {
			t.Errorf("setting %d round-trips to %d", s, got)
		}
	}
	if FormationModeOfSetting(1) != 2 {
		t.Error("setting 1 must command mode 2")
	}
}
