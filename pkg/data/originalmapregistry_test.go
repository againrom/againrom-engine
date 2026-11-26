package data_test

import (
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
	want := sim.PathFinding{StaticScanAhead: 6, DynamicScanAhead: 4, StaticRefreshRate: 17, DynamicRefreshRate: 33, DynamicByStaticLookup: 2, StaticIsntNeeded: 7}
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
