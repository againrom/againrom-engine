package terrain_test

import (
	"againrom/pkg/render/terrain"
	"testing"
)

func TestTickerRestoreRetainsCounterAndFractionAtNextBoundary(t *testing.T) {
	source := terrain.NewTicker(62000)
	source.AdvanceMicros(5*62000 + 17000)
	cold := terrain.NewTicker(62000)
	cold.Restore(source.Count(), source.Remainder())
	if cold.Count() != 5 || cold.Remainder() != 17000 || cold.Period() != 62000 {
		t.Fatal("restore changed counter, fraction or selected cadence")
	}
	if cold.AdvanceMicros(44999) != 0 || cold.Count() != 5 || cold.Remainder() != 61999 {
		t.Fatal("restored fraction crossed its boundary early")
	}
	if cold.AdvanceMicros(1) != 1 || cold.Count() != 6 || cold.Remainder() != 0 {
		t.Fatal("restored fraction lost its next boundary")
	}
	cold.ResetPhase()
	if cold.Count() != 6 || cold.Remainder() != 0 {
		t.Fatal("historical reset changed logical counter")
	}
}
