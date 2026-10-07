package data

import "testing"

func TestHumanDefaultsAreTheAllEmptyRow(t *testing.T) {
	want, err := NewHumanDef("", sentinelRowValues(lastHumanSlot+1))
	if err != nil {
		t.Fatal(err)
	}
	got := HumanDefaults()
	if got != want {
		t.Fatalf("HumanDefaults %+v differs from the all-empty row %+v", got, want)
	}
	if got.RotationSpeed != unitCtorDefaults.RotationSpeed || got.DyingTime != unitCtorDefaults.DyingTime || got.HealthMax != unitCtorDefaults.HealthMax {
		t.Fatalf("HumanDefaults %+v do not carry the constructor values", got)
	}
}
