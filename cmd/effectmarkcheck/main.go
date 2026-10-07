// Command effectmarkcheck is the marking-spell integration witness: it opens
// a real campaign mission from a lawful install, casts each marking spell on
// a real mission actor through the ordinary simulation paths, and reports the
// mark records the client rebuild builds for it against the install's own
// projectile art and unit registry.
//
// It is a developer tool. The asset root comes from -assets or
// AGAINROM_ASSETS, and no output is written to the repository.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// The ten kinds MAGIC-MARK-061 sends to a builder, with the two MAGIC-ACTOR-066
// sends to the unit draw's own by-kind arms instead, in spell order.
var kinds = []struct {
	Spell int
	Name  string
}{
	{5, "protection_from_fire"}, {6, "heal"}, {8, "poison_cloud"},
	{10, "protection_from_water"}, {11, "drain_life"}, {15, "invisibility"},
	{16, "protection_from_air"}, {18, "shield"}, {20, "stone_curse"},
	{22, "protection_from_earth"}, {23, "bless"}, {27, "curse"},
}

func main() {
	assets := flag.String("assets", "", "game asset root (default $AGAINROM_ASSETS)")
	mission := flag.Int("mission", 91, "real campaign mission whose actors and install back the witness")
	flag.Parse()
	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		die("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	archives, err := game.OpenArchives(root)
	if err != nil {
		die(err)
	}
	table, err := game.LoadTable(archives.Containers)
	if err != nil {
		die(err)
	}
	projectiles, err := game.LoadProjectiles(archives.Containers)
	if err != nil {
		die(err)
	}
	units, err := game.LoadUnits(archives.Containers)
	if err != nil {
		die(err)
	}
	started, err := game.StartMission(archives.Containers, *mission, table, mapload.DifficultyNormal, nil)
	if err != nil {
		die(err)
	}

	// The shipped-content sweep: every kind that reaches a builder must name a
	// defined projectiles.reg row, because the record index IS the picture id
	// (MAGIC-MARK-059, MAGIC-PIC-026).
	fmt.Printf("install %s: projectile sheets=%d unit classes=%d\n",
		root, len(projectiles.Sheets), len(units.Classes))
	for _, k := range kinds {
		kind := terrain.MarkKind(k.Spell)
		sheet := projectiles.Sheet(kind)
		fmt.Printf("  spell %2d %-22s kind=%#04x sheet=%-5v frames=%d\n",
			k.Spell, k.Name, kind, sheet != nil, len(sheetFrames(sheet)))
	}
	for _, picture := range []int{12, 13, 20, 60} {
		sheet := projectiles.Sheet(picture)
		fmt.Printf("  hotfix picture %2d sheet=%-5v frames=%d\n",
			picture, sheet != nil, len(sheetFrames(sheet)))
	}

	// The mission's own actors, and the TileSize each of their classes carries.
	ents := started.World.Entities()
	if len(ents) == 0 {
		die(fmt.Sprintf("mission %d holds no actors", *mission))
	}
	tiles := map[int]int{}
	for _, e := range ents {
		tiles[markTileSize(units.Classes[e.Class])]++
	}
	fmt.Printf("mission %d: actors=%d tilesize histogram=%v\n", *mission, len(ents), sortedCounts(tiles))

	// The rebuild itself, at each footprint scale the mission actually holds.
	for _, tile := range sortedKeys(tiles) {
		for _, k := range kinds {
			kind := terrain.MarkKind(k.Spell)
			records := terrain.EffectMarkRecords(kind, 0xffff, tile)
			back, front := 0, 0
			for _, r := range records {
				if r.Depth > 0 {
					back++
				} else {
					front++
				}
			}
			fmt.Printf("  tilesize=%d spell %2d %-22s records=%3d behind=%3d in-front=%3d\n",
				tile, k.Spell, k.Name, len(records), back, front)
		}
	}

	// And the whole path: each marking spell cast on a controlled actor inside
	// the mission's OWN terrain and installed spell table, through the ordinary
	// command, wind-up and apply paths. The client's mark elements are opened
	// from exactly the effect set this reports.
	marks, err := sim.ControlledEffectMarkWitness(started.World)
	if err != nil {
		die(err)
	}
	for _, m := range marks {
		kind := terrain.MarkKind(int(m.Spell))
		fmt.Printf("  cast spell %2d kind=%#04x attached=%-5v records=%d\n",
			m.Spell, kind, m.Attached, len(terrain.EffectMarkRecords(kind, 0xffff, 1)))
	}
}

func sheetFrames(s *terrain.EffectSheet) []*terrain.EffectFrame {
	if s == nil {
		return nil
	}
	return s.Frames
}

func markTileSize(c *terrain.UnitClass) int {
	if c == nil || c.TileSize < 1 {
		return 1
	}
	return c.TileSize
}

func sortedKeys(in map[int]int) []int {
	out := make([]int, 0, len(in))
	for k := range in {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

type orderedCount struct {
	Key   int
	Count int
}

func sortedCounts(in map[int]int) []orderedCount {
	out := make([]orderedCount, 0, len(in))
	for k, n := range in {
		out = append(out, orderedCount{Key: k, Count: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func die(v any) {
	fmt.Fprintln(os.Stderr, "effectmarkcheck:", v)
	os.Exit(1)
}
