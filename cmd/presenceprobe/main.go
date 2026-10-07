package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func main() {
	assets := flag.String("assets", "", "asset root")
	mission := flag.Int("mission", 60, "mission number")
	flag.Parse()

	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	archives, err := game.OpenArchives(root)
	if err != nil {
		fmt.Println("archives:", err)
		return
	}
	defs, err := game.LoadDefinitions(archives.Containers)
	if err != nil {
		fmt.Println("definitions:", err)
		return
	}
	table := defs.Table
	party := game.MissionParty(defs.StartWeapon, defs.Bodies, table)
	ms, err := game.StartMission(archives.Containers, *mission, table, mapload.DifficultyNormal, party)
	if err != nil {
		fmt.Println("mission:", err)
		return
	}
	w := ms.World

	held := map[sim.EntityID]bool{}
	groups := map[uint32]int{}
	for _, e := range w.Entities() {
		held[e.ID] = true
		groups[e.Group]++
	}

	type row struct{ authored, resolved int }
	out := map[int32]*row{}
	for _, in := range w.Script().Instants() {
		var r *row
		switch in.Op {
		case sim.ScriptInstantTakeOffMap, sim.ScriptInstantReturnToMap, sim.ScriptInstantSwapOnMap,
			sim.ScriptInstantGroupOffMap, sim.ScriptInstantGroupOnMap:
			if out[in.Op] == nil {
				out[in.Op] = &row{}
			}
			r = out[in.Op]
		default:
			continue
		}
		r.authored++
		switch in.Op {
		case sim.ScriptInstantGroupOffMap, sim.ScriptInstantGroupOnMap:
			if in.HasGroup && groups[in.Group] > 0 {
				r.resolved++
			}
		case sim.ScriptInstantSwapOnMap:
			if in.HasUnit && in.HasUnit2 && held[in.Unit] && held[in.Unit2] {
				r.resolved++
			}
		default:
			if in.HasUnit && held[in.Unit] {
				r.resolved++
			}
		}
	}
	var ops []int
	for op := range out {
		ops = append(ops, int(op))
	}
	sort.Ints(ops)
	for _, op := range ops {
		r := out[int32(op)]
		fmt.Printf("mission %d  op %2d  %2d authored, %2d with every reference resolving in this world\n",
			*mission, op, r.authored, r.resolved)
	}
}
