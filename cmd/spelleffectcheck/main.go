// Command spelleffectcheck reports the installed spell rows that feed the
// simulation. It is a developer tool: the asset root comes from -assets or
// AGAINROM_ASSETS, and no output is written to the repository.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func main() {
	assets := flag.String("assets", "", "game asset root (default $AGAINROM_ASSETS)")
	mission := flag.Int("mission", 91, "real campaign mission whose terrain and spell table back the controlled wiring witness")
	flag.Parse()
	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		fmt.Fprintln(os.Stderr, "spelleffectcheck: no asset root: pass -assets or set AGAINROM_ASSETS")
		os.Exit(1)
	}
	archives, err := game.OpenArchives(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spelleffectcheck:", err)
		os.Exit(1)
	}
	table, err := game.LoadTable(archives.Containers)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spelleffectcheck:", err)
		os.Exit(1)
	}
	spells, err := data.LoadSpells(table.Spells)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spelleffectcheck:", err)
		os.Exit(1)
	}
	for i, spell := range spells {
		params := table.Spells.EntryParams(i + 1)
		fmt.Printf("%2d %-24s params=%v effects=%q\n", i+1, spell.Name, params,
			table.Spells.EntryStrings(i+1))
	}
	rules := mapload.SpellRules(table)
	point, area := 0, 0
	distributions := make(map[uint8]int)
	kinds := make(map[sim.EffectKind]int)
	modes := make(map[sim.EffectMode]int)
	for _, rule := range rules {
		if rule.Area {
			area++
		} else {
			point++
		}
		distributions[rule.Distribution]++
		kinds[rule.EffectKind]++
		modes[rule.EffectMode]++
	}
	fmt.Printf("spell table: rows=%d point=%d area=%d distributions=%v effect-kinds=%v effect-modes=%v\n",
		len(rules), point, area, sortedCounts(distributions), sortedCounts(kinds), sortedCounts(modes))

	// Walk every campaign map through the same ALM and script compiler the game
	// uses. This is the shipped-content half of the report: it says which table
	// rows the authored script-cast producers can actually reach, independently
	// from the 28-row installed vocabulary above.
	maps := game.ArchiveMaps(archives.Containers)
	castCells := make(map[int32]int)
	castUnits := make(map[int32]int)
	ages := make(map[int32]int)
	compiled := 0
	for _, name := range maps.Names() {
		stream, err := maps.Read(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, "spelleffectcheck:", err)
			os.Exit(1)
		}
		m, err := alm.Open(stream)
		if err != nil {
			fmt.Fprintln(os.Stderr, "spelleffectcheck:", name, err)
			os.Exit(1)
		}
		program, _, err := mapload.CompileScript(m, mapload.ScriptRefs{})
		if err != nil {
			fmt.Fprintln(os.Stderr, "spelleffectcheck:", name, err)
			os.Exit(1)
		}
		compiled++
		for _, instant := range program.Instants() {
			switch instant.Op {
			case sim.ScriptInstantCastAtCell:
				castCells[instant.Args[4]]++
			case sim.ScriptInstantCastAtUnit:
				castUnits[instant.Args[2]]++
			case sim.ScriptInstantCellEffectAge:
				ages[instant.Args[2]]++
			}
		}
	}
	fmt.Printf("campaign sweep: maps=%d cast-at-cell=%v cast-at-unit=%v cell-effect-age=%v\n",
		compiled, sortedCounts(castCells), sortedCounts(castUnits), sortedCounts(ages))
	started, err := game.StartMission(archives.Containers, *mission, table, mapload.DifficultyNormal, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spelleffectcheck:", err)
		os.Exit(1)
	}
	witness, err := sim.ControlledSpellEffectWitness(started.World)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spelleffectcheck:", err)
		os.Exit(1)
	}
	fmt.Printf("mission %d controlled wiring: bounds=%dx%d patch=(%d,%d) point-speed=%d->%d attached=%d "+
		"area-hp friend=%d/%d/%d enemy=%d/%d/%d outside-wall=%d/%d wall-route ground=%v/%v/%v ghost=%v/%v/%v air=%v/%v/%v\n",
		*mission, witness.Bounds.Width, witness.Bounds.Height, witness.X, witness.Y,
		witness.PointBefore, witness.PointAfter, witness.PointEffects,
		witness.AreaFriendBefore, witness.AreaFriendAt15, witness.AreaFriendAt16,
		witness.AreaEnemyBefore, witness.AreaEnemyAt15, witness.AreaEnemyAt16,
		witness.AreaOutsideBefore, witness.AreaOutsideAfter,
		witness.WallBefore[sim.DomainGround], witness.WallDuring[sim.DomainGround], witness.WallAfter[sim.DomainGround],
		witness.WallBefore[sim.DomainGhost], witness.WallDuring[sim.DomainGhost], witness.WallAfter[sim.DomainGhost],
		witness.WallBefore[sim.DomainAir], witness.WallDuring[sim.DomainAir], witness.WallAfter[sim.DomainAir])
	fmt.Printf("mission %d hotfix wiring: fire-wall-overlap=%d light-cells=%d darkness-cells=%d\n",
		*mission, witness.FireWallOverlap, witness.LightCells, witness.DarknessCells)
}

type orderedCount struct {
	Key   int
	Count int
}

func sortedCounts[K ~uint8 | ~int32](in map[K]int) []orderedCount {
	out := make([]orderedCount, 0, len(in))
	for key, count := range in {
		out = append(out, orderedCount{Key: int(key), Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
