package game

import (
	"errors"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
)

type failingSource struct{ err error }

func (s failingSource) ReadFile(string) ([]byte, error) { return nil, s.err }

func TestTerrainCostTable(t *testing.T) {
	good := synth.Reg(0, []synth.RegNode{{Name: "Terrain", Kind: 0x01, Children: []synth.RegNode{
		{Name: "CostLand", Kind: 0x02, Int: 3}, {Name: "CostRoad", Kind: 0x02, Int: 5},
	}}})
	costs, ok, err := terrainCostTable(memorySource{terrainCostsAddress: good}, terrainCostsAddress)
	if err != nil || !ok || costs[1] != 3 || costs[10] != 5 || costs[2] != 8 {
		t.Fatalf("installed table: %v %t %v", costs, ok, err)
	}
	for name, tc := range map[string]struct {
		src     entrySource
		addr    string
		ok      bool
		wantErr string
	}{
		"no address":  {memorySource{terrainCostsAddress: good}, "", false, ""},
		"absent file": {memorySource{}, terrainCostsAddress, false, ""},
		"read error":  {failingSource{errors.New("disk")}, terrainCostsAddress, false, "original terrain parameters: disk"},
		"malformed":   {memorySource{terrainCostsAddress: []byte("not a registry")}, terrainCostsAddress, false, "original terrain parameters: "},
	} {
		_, ok, err := terrainCostTable(tc.src, tc.addr)
		if ok != tc.ok || (err == nil) != (tc.wantErr == "") || (err != nil && !strings.HasPrefix(err.Error(), tc.wantErr)) {
			t.Errorf("%s: ok=%t err=%v", name, ok, err)
		}
	}
}

// Both games' rows name one cell-plane and checker address, the first
// game's sim address is the same file, and the second game still names no
// sim address.
func TestTerrainCostAddressesOfTheGameRows(t *testing.T) {
	first, second := filesOf(base.GameROM1), filesOf(base.GameROM2)
	if first.terrainCosts != "world/data/map.reg" || first.planeCosts != first.terrainCosts || second.planeCosts != first.terrainCosts || second.terrainCosts != "" {
		t.Fatalf("rows: %q %q / %q %q", first.terrainCosts, first.planeCosts, second.terrainCosts, second.planeCosts)
	}
}

// A root without map.reg gives no planes and the checker no table.
func TestRootWithoutTerrainParametersGivesNoPlanes(t *testing.T) {
	planes, err := readOriginalCellPlanes(&alm.Map{}, memorySource{})
	if planes != nil || err != nil {
		t.Fatalf("planes %v, err %v", planes, err)
	}
	x := checkInspector()
	x.archives = &Archives{Containers: worldFS(t, synth.File{Path: "data/other.reg", Data: synth.Reg(0, nil)})}
	if _, ok, err := x.terrainCosts(); ok || err != nil {
		t.Fatalf("checker table: %t %v", ok, err)
	}
}
