// Command modmission assembles a mod mission: it copies an installed map and
// its event and tip texts into mission/<lang>/ of a mod folder, where the
// mod's data/missions.toml names them. It reads the install and writes only
// into the mod folder; both are explicit arguments.
//
//	modmission -install <install root> -map <mission> -out <mod folder>
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"againrom/pkg/game"
	"againrom/pkg/modrt"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "modmission:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("modmission", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	install := fs.String("install", "", "install root to read (required)")
	mission := fs.Int("map", 0, "installed mission whose map and texts are copied (required)")
	dest := fs.String("out", "", "mod folder to write into (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *install == "" || *dest == "" || *mission <= 0 || fs.NArg() != 0 {
		return fmt.Errorf("usage: modmission -install <install root> -map <mission> -out <mod folder>")
	}
	lang := modrt.LanguageFor(game.BaseID(game.InspectInstall(*install)))
	written, err := game.AssembleModMission(*install, lang, *mission, *dest)
	sort.Strings(written)
	for _, w := range written {
		fmt.Fprintf(out, "modmission: wrote %s\n", w)
	}
	return err
}
