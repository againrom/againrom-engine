package data

import (
	"againrom/pkg/formats/reg"
	"againrom/pkg/sim"
)

// MapRegistry is what the engine takes from the installed map.reg: the
// Terrain cost table and the [Path Finding] route values.
type MapRegistry struct {
	Costs [11]byte
	Path  sim.PathFinding
}

// maxRegKeyName is the longest key name the registry stores; a longer name is
// cut to it, and the original's lookup compares that many characters.
const maxRegKeyName = 15

// OriginalMapRegistry reads the installed map.reg. ALM-TERR-043 fixes both
// missing-key defaults and byte narrowing for the Terrain costs; a missing or
// non-integer [Path Finding] key holds its shipped value. A missing FILE is
// not a missing key: callers must supply a successfully read and parsed
// registry.
func OriginalMapRegistry(raw []byte) (MapRegistry, error) {
	out := MapRegistry{Costs: [11]byte{255, 8, 8, 9, 14, 6, 12, 11, 16, 8, 6}, Path: sim.ShippedPathFinding()}
	r, err := reg.Parse(raw)
	if err != nil {
		return out, err
	}
	for i, key := range [...]string{"CostLand", "CostGrass", "CostFlowers", "CostSand", "CostCracked", "CostStones", "CostSavanna", "CostMountain", "CostWater", "CostRoad"} {
		if value, found := r.GetInt("Terrain", key); found {
			out.Costs[i+1] = byte(value)
		}
	}
	for key, into := range map[string]*int32{
		"StaticScanAhead":       &out.Path.StaticScanAhead,
		"DynamicScanAhead":      &out.Path.DynamicScanAhead,
		"StaticRefreshRate":     &out.Path.StaticRefreshRate,
		"DynamicRefreshRate":    &out.Path.DynamicRefreshRate,
		"DynamicByStaticLookup": &out.Path.DynamicByStaticLookup,
		"StaticIsntNeeded":      &out.Path.StaticIsntNeeded,
	} {
		if value, found := r.GetInt("Path Finding", key[:min(len(key), maxRegKeyName)]); found {
			*into = value
		}
	}
	return out, nil
}
