// Command campaigncensus prints the second game's campaign support census:
// one tab-separated row per campaign map of a lawful ROM2 install, then the
// totals. It reads the install and writes nothing but its output.
//
//	go run ./cmd/campaigncensus -assets <root>
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"againrom/pkg/game"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "campaigncensus:", err)
		os.Exit(2)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("campaigncensus", flag.ContinueOnError)
	assets := fs.String("assets", "", "ROM2 install root (default $AGAINROM_ASSETS)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root := *assets
	if root == "" {
		root = os.Getenv("AGAINROM_ASSETS")
	}
	if root == "" {
		return fmt.Errorf("pass -assets or set AGAINROM_ASSETS to a ROM2 install root")
	}
	archives, err := game.OpenArchives(root)
	if err != nil {
		return err
	}
	census, err := game.SecondGameCensus(archives)
	if err != nil {
		return err
	}
	return game.WriteSecondCensus(out, census)
}
