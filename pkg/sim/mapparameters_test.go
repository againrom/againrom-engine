package sim

import (
	"bytes"
	"testing"
)

func stepRateUnder(t *testing.T, rules BuildRules) int32 {
	t.Helper()
	w := srWorld(t, srPlane(8), nil, []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}})
	w.SetBuildRules(rules)
	rate, _, _, ok := w.StepRate(1, 2, 1)
	if !ok {
		t.Fatal("the query refused a rated live mover")
	}
	return rate
}

func TestSpeedMultiplierIsTheWorldsOwn(t *testing.T) {
	t.Parallel()

	shipped := stepRateUnder(t, BuildRules{})
	if want := int32(16); shipped != want {
		t.Fatalf("shipped rate %d, want %d (16*8/8)", shipped, want)
	}
	for _, mult := range []int32{8, 4, 2} {
		p := ShippedPathFinding()
		p.SpeedMultiplier = mult
		got := stepRateUnder(t, BuildRules{Path: &p})
		if want := 16 * mult / 8; got != want {
			t.Errorf("multiplier %d: rate %d, want %d", mult, got, want)
		}
	}
	if again := stepRateUnder(t, BuildRules{}); again != shipped {
		t.Errorf("a later world rates %d, want %d", again, shipped)
	}
}

func TestShippedScanShiftIsTheCompiledShift(t *testing.T) {
	t.Parallel()

	if got := ShippedScanning().ScanShift; got != sightShift {
		t.Fatalf("shipped ScanShift %d, want %d", got, sightShift)
	}
	if sightTablesAt[sightShift] != sightTables {
		t.Fatal("the shift-7 tables are not the compiled tables")
	}
}

func TestScanShiftMovesTheAISightOnly(t *testing.T) {
	t.Parallel()

	// The observer stands on height 30: the shift changes how many steps it buys.
	centre := cell{x: 22, y: 22}
	hill := make([]byte, rngBounds.Width*rngBounds.Height)
	hill[centre.y*rngBounds.Width+centre.x] = 30
	newWorld := func() *World {
		w, err := NewTerrainWorld(1, rngBounds, ModeCanonical, Terrain{Height: hill},
			[]Entity{{ID: 1, X: centre.x, Y: centre.y, ScanRange: 2}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	stampAt := func(rules BuildRules, ai bool) []byte {
		w := newWorld()
		w.SetBuildRules(rules)
		if ai {
			return w.groupSight(aiSight, []int{0})
		}
		return w.groupSight(fogSight, []int{0})
	}
	shift := func(s int32) BuildRules { return BuildRules{Scan: &Scanning{ScanShift: s}} }

	if !bytes.Equal(stampAt(BuildRules{}, true), stampAt(shift(7), true)) {
		t.Error("an absent Scan and ScanShift 7 give different AI sight")
	}
	for _, s := range []int32{6, 8} {
		w := newWorld()
		w.SetBuildRules(shift(s))
		reader := sightReader{tables: buildSightTables(s), shift: s, inset: aiInset}
		want := w.groupSight(reader, []int{0})
		if got := stampAt(shift(s), true); !bytes.Equal(got, want) {
			t.Errorf("ScanShift %d: AI sight differs from the march over tables built at %d", s, s)
		}
		if bytes.Equal(stampAt(shift(s), true), stampAt(BuildRules{}, true)) {
			t.Errorf("ScanShift %d changed nothing on the hill", s)
		}
		if !bytes.Equal(stampAt(shift(s), false), stampAt(BuildRules{}, false)) {
			t.Errorf("ScanShift %d moved the fog reader", s)
		}
	}
}

func TestAnOutOfRangeScanShiftRunsTheShippedShift(t *testing.T) {
	t.Parallel()

	for _, s := range []int32{0, -1, 9, 40} {
		w := rngWorld(t, Entity{ID: 1, X: 22, Y: 22, ScanRange: 5})
		w.SetBuildRules(BuildRules{Scan: &Scanning{ScanShift: s}})
		if got := w.scanShift(); got != sightShift {
			t.Errorf("ScanShift %d runs %d, want %d", s, got, sightShift)
		}
	}
}
