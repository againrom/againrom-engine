// Command areaoverlaycheck is the 1003 integration witness: it drives the
// installed spell rows of a real campaign mission through the same simulation
// and the same art bundle the game uses, and reports what each area row would
// put on screen.
//
// It is a developer tool. The asset root comes from -assets or AGAINROM_ASSETS
// and nothing is written to the repository.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func main() {
	assets := flag.String("assets", "", "game asset root (default $AGAINROM_ASSETS)")
	mission := flag.Int("mission", 91, "campaign mission whose terrain, spell table and art back the witness")
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
	sheets, err := game.LoadProjectiles(archives.Containers)
	if err != nil {
		die(err)
	}
	started, err := game.StartMission(archives.Containers, *mission, table, mapload.DifficultyNormal, nil)
	if err != nil {
		die(err)
	}
	rules := mapload.SpellRules(table)
	fmt.Printf("mission %d  bounds=%dx%d  installed rows=%d\n",
		*mission, started.World.Bounds().Width, started.World.Bounds().Height, len(rules))

	// The four overlay ids against the installed art, and the frames each one
	// can actually reach on the shipped sheet.
	for _, id := range data.OverlaySpells {
		picture, _ := data.OverlayPicture(id)
		sheet := sheets.Sheet(picture)
		if sheet == nil {
			fmt.Printf("overlay spell %2d  picture %2d  NO SHEET\n", id, picture)
			continue
		}
		reached := map[int]bool{}
		for counter := range 400 {
			for x := range 8 {
				for y := range 8 {
					if f, ok := terrain.OverlayFrame(id, sheet.Phases, counter, x, y); ok {
						reached[f] = true
					}
				}
			}
		}
		fmt.Printf("overlay spell %2d  picture %2d  sheet frames=%d phases=%d  drawn frames=%v\n",
			id, picture, len(sheet.Frames), sheet.Phases, sortedKeys(reached))
	}

	// Every installed area row, cast for real in this mission's own world, with
	// the stages it reports and the objects a client would build from them.
	for _, rule := range rules {
		if !rule.Area {
			continue
		}
		stages, cells := castAndCollect(started, rule)
		mode := "cloud"
		switch {
		case rule.Distribution == 5:
			mode = "staged"
		case rule.AreaDuration == 0:
			mode = "blast"
		}
		picture := data.BurstPicture(int(rule.ID))
		life := 0
		if sheets.Sheet(picture) != nil {
			life = data.BurstLife(picture)
		}
		_, overlay := data.OverlayPicture(int(rule.ID))
		fmt.Printf("row %2d mode=%-6s stages=%2d painted-cells=%3d burst-picture=%2d life=%2d overlay-art=%v\n",
			rule.ID, mode, stages, cells, picture, life, overlay)
	}
}

// castAndCollect starts a fresh copy of the mission, drops a caster with the
// mana and the book for this row, casts it, and counts the stages the advance
// reports. A row that is not staged reports nothing, which is the point.
func castAndCollect(started *game.Mission, rule sim.SpellRule) (stages, cells int) {
	fresh, err := sim.NewSpelledWorld(1, started.World.Bounds(), sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 1, X: 40, Y: 40, HP: 200, MaxHP: 200, Owner: 3, TokenSize: 1,
			Mind: 60, Mana: 900, MaxMana: 900, KnownSpells: 1 << rule.ID, ScanRange: 19,
			AttackCharge: 4, AttackRelax: 2}}, nil, []sim.SpellRule{rule})
	if err != nil {
		die(err)
	}
	aimX, aimY := int32(40), int32(40)
	if rule.MaxRange > 0 {
		aimX = 40 + int32(rule.MaxRange)/2
	}
	report := sim.StepReported(fresh, []sim.Command{
		sim.CastAt(1, sim.SpellID(rule.ID), sim.CellPoint{X: aimX, Y: aimY})})
	count := func(r sim.Report) {
		for _, p := range r.AreaPaints {
			stages++
			cells += len(p.Cells)
		}
	}
	count(report)
	for range 220 {
		count(sim.StepReported(fresh, nil))
	}
	return stages, cells
}

func sortedKeys(in map[int]bool) []int {
	out := make([]int, 0, len(in))
	for k := range in {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func die(v any) {
	fmt.Fprintln(os.Stderr, "areaoverlaycheck:", v)
	os.Exit(1)
}
