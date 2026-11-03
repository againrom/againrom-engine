// Command rewardmarkwitness prints, for each campaign map of a lawful second
// game install, the count of type-6 rows with bit 8 of dword 0x10 and the
// count of units the started mission places with the reward mark. It reads the
// install and writes nothing but its output; it prints counts and mission
// numbers only.
//
//	go run ./cmd/rewardmarkwitness -assets <root>
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "rewardmarkwitness:", err)
		os.Exit(2)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("rewardmarkwitness", flag.ContinueOnError)
	assets := fs.String("assets", "", "second-game install root (default $AGAINROM_ASSETS)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root := *assets
	if root == "" {
		root = os.Getenv("AGAINROM_ASSETS")
	}
	if root == "" {
		return fmt.Errorf("pass -assets or set AGAINROM_ASSETS to a second-game install root")
	}
	archives, err := game.OpenArchives(root)
	if err != nil {
		return err
	}
	defs, err := game.LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "mission\trows\tplaced_marked\tunits")
	var maps, rows, marked int
	for n := 1; n <= 300; n++ {
		address, _ := game.MissionMap(n)
		raw, err := archives.Containers.ReadFile(address)
		if err != nil {
			continue
		}
		m, err := alm.OpenROM2(raw)
		if err != nil {
			fmt.Fprintf(out, "%d\tdecode error\n", n)
			continue
		}
		r := 0
		for _, u := range m.Units {
			if u.RewardMarkROM2() {
				r++
			}
		}
		ms, err := game.StartMissionFrom(m, "", n, defs.Table, mapload.DifficultyNormal, nil)
		if err != nil {
			fmt.Fprintf(out, "%d\t%d\tstart error\t%d\n", n, r, len(m.Units))
			continue
		}
		placed := 0
		for _, e := range ms.World.Entities() {
			if e.SuppressRewards {
				placed++
			}
		}
		fmt.Fprintf(out, "%d\t%d\t%d\t%d\n", n, r, placed, len(m.Units))
		maps, rows, marked = maps+1, rows+r, marked+placed
	}
	fmt.Fprintf(out, "totals maps=%d rows=%d placed_marked=%d\n", maps, rows, marked)
	return nil
}
