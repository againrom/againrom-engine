package data_test

import (
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func pathFindingReg(keys []synth.RegNode) []byte {
	return synth.Reg(0, []synth.RegNode{{Name: "Path Finding", Kind: 1, Children: keys}})
}

// The registry stores key names cut to 15 characters, and the six route keys
// are all asked by longer names but one.
func TestOriginalMapRegistryReadsTheStoredPathFindingNames(t *testing.T) {
	got, err := data.OriginalMapRegistry(pathFindingReg([]synth.RegNode{
		{Name: "StaticScanAhead", Kind: 2, Int: 6},
		{Name: "DynamicScanAhea", Kind: 2, Int: 4},
		{Name: "StaticRefreshRa", Kind: 2, Int: 17},
		{Name: "DynamicRefreshR", Kind: 2, Int: 33},
		{Name: "DynamicByStatic", Kind: 2, Int: 2},
		{Name: "StaticIsntNeede", Kind: 2, Int: 7},
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := sim.PathFinding{StaticScanAhead: 6, DynamicScanAhead: 4, StaticRefreshRate: 17, DynamicRefreshRate: 33, SpeedMultiplier: 8, DynamicByStaticLookup: 2, StaticIsntNeeded: 7}
	if got.Path != want {
		t.Fatalf("path finding %+v, want %+v", got.Path, want)
	}
}

// A key the file lacks, or holds as another type, keeps its shipped value; a
// file with no Path Finding section keeps all six.
func TestOriginalMapRegistryKeepsTheShippedValueOfAMissingPathFindingKey(t *testing.T) {
	got, err := data.OriginalMapRegistry(pathFindingReg([]synth.RegNode{
		{Name: "StaticIsntNeede", Kind: 2, Int: 9},
		{Name: "DynamicByStatic", Kind: 0, Str: "three"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := sim.ShippedPathFinding()
	want.StaticIsntNeeded = 9
	if got.Path != want {
		t.Fatalf("path finding %+v, want %+v", got.Path, want)
	}
	bare, err := data.OriginalMapRegistry(synth.Reg(0, []synth.RegNode{{Name: "Terrain", Kind: 1}}))
	if err != nil || bare.Path != sim.ShippedPathFinding() {
		t.Fatalf("a file with no section: %+v, %v", bare.Path, err)
	}
}

func scanReg(speed, shift int32) []byte {
	return synth.Reg(0, []synth.RegNode{
		{Name: "Path Finding", Kind: 1, Children: []synth.RegNode{{Name: "SpeedMultiplier", Kind: 2, Int: speed}}},
		{Name: "Scanning", Kind: 1, Children: []synth.RegNode{{Name: "ScanShift", Kind: 2, Int: shift}}},
	})
}

func TestOriginalMapRegistryReadsSpeedMultiplierAndScanShift(t *testing.T) {
	got, err := data.OriginalMapRegistry(scanReg(4, 6))
	if err != nil {
		t.Fatal(err)
	}
	if got.Path.SpeedMultiplier != 4 || got.Scan.ScanShift != 6 {
		t.Fatalf("read %d and %d, want 4 and 6", got.Path.SpeedMultiplier, got.Scan.ScanShift)
	}
	bare, err := data.OriginalMapRegistry(synth.Reg(0, []synth.RegNode{{Name: "Terrain", Kind: 1}}))
	if err != nil || bare.Path.SpeedMultiplier != 8 || bare.Scan.ScanShift != 7 {
		t.Fatalf("missing keys give %d and %d, %v; want 8 and 7", bare.Path.SpeedMultiplier, bare.Scan.ScanShift, err)
	}
}

func TestOriginalMapRegistryRefusesValuesThatBreakTheArithmetic(t *testing.T) {
	for _, tc := range []struct {
		speed, shift int32
		want         string
	}{
		{0, 7, "SpeedMultiplier 0"},
		{-3, 7, "SpeedMultiplier -3"},
		{256, 7, "SpeedMultiplier 256"},
		{8, 0, "ScanShift 0"},
		{8, 9, "ScanShift 9"},
		{8, -1, "ScanShift -1"},
	} {
		if _, err := data.OriginalMapRegistry(scanReg(tc.speed, tc.shift)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("speed %d shift %d: %v, want a refusal naming %q", tc.speed, tc.shift, err, tc.want)
		}
	}
	for _, edge := range [][2]int32{{1, 1}, {255, 8}} {
		if _, err := data.OriginalMapRegistry(scanReg(edge[0], edge[1])); err != nil {
			t.Errorf("edge %v refused: %v", edge, err)
		}
	}
}
