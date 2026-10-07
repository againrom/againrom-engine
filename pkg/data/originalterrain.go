package data

import "againrom/pkg/formats/reg"

// OriginalTerrainCostTable reads the installed Terrain parameters. ALM-TERR-043
// fixes both missing-key defaults and byte narrowing. A missing FILE is not a
// missing key: callers must supply a successfully read and parsed registry.
func OriginalTerrainCostTable(raw []byte) ([11]byte, error) {
	costs := [11]byte{255, 8, 8, 9, 14, 6, 12, 11, 16, 8, 6}
	r, err := reg.Parse(raw)
	if err != nil {
		return costs, err
	}
	for i, key := range [...]string{"CostLand", "CostGrass", "CostFlowers", "CostSand", "CostCracked", "CostStones", "CostSavanna", "CostMountain", "CostWater", "CostRoad"} {
		if value, found := r.GetInt("Terrain", key); found {
			costs[i+1] = byte(value)
		}
	}
	return costs, nil
}
