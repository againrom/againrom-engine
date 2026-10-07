// Command dlgtool censuses the campaign's event-text corpus against a lawful
// install (developer tool).
//
//	dlgtool census [-assets DIR]
//
// It answers one question: do the dialogue window's TWO speaker tests agree on
// the shipped data? The window's shape is decided once per file by whether the
// payload contains the three letters `npc` anywhere in it — prose included — and
// which face it shows is decided per part by whether that part's own tag
// contains `npc=`. The two differ by one character, and a build that used one
// needle for both would pass every playthrough. This measures both, separately,
// over every event file of a root.
//
// It prints FIGURES ONLY — counts. No path out of an archive, no tag, no line of
// game text and no byte of one reaches its output, so what it prints can be
// pasted into a work item's evidence. It writes no file at all.
//
// The asset root comes from -assets or AGAINROM_ASSETS and is never compiled in.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"againrom/pkg/game"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "dlgtool:", err)
		os.Exit(1)
	}
}

// eventPrefix and eventStem are what an event text's address looks like. They
// are spelt here rather than reached for, because the reader that composes one
// address from a mission and an event number cannot enumerate: this tool asks
// the archive what it holds instead of asking whether a name it invented is
// there, so a mission or an event number the corpus uses and nothing in this
// tree predicts is still counted.
const (
	eventPrefix = "main/text/battle/"
	eventStem   = "/event"
)

// partCeiling bounds the walk over one file's parts. The parts of a file are
// numbered from 1 and the walk stops at the first number the file does not hold,
// so this only has to be past any authored part number; it exists so a payload
// crafted to hold every number cannot spin.
const partCeiling = 1000

func run(args []string, out io.Writer) error {
	// THE VERB IS TAKEN BEFORE THE FLAGS ARE PARSED, because the flag package
	// stops at the first argument that is not a flag: handing it the verb first
	// would leave every flag after it unparsed and the asset root empty, with
	// nothing said about why.
	if len(args) == 0 || args[0] != "census" {
		return fmt.Errorf("usage: dlgtool census [-assets DIR]")
	}
	fs := flag.NewFlagSet("dlgtool", flag.ContinueOnError)
	fs.SetOutput(out)
	assets := fs.String("assets", os.Getenv("AGAINROM_ASSETS"), "path to a lawful game install")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: dlgtool census [-assets DIR]")
	}
	if *assets == "" {
		return fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	a, err := game.OpenArchives(*assets)
	if err != nil {
		return err
	}
	return census(a, out)
}

func census(a *game.Archives, out io.Writer) error {
	var files, paneByFile, spokenByTag, disagree, parts, speakingParts int
	for _, e := range a.Containers.Entries() {
		if !isEventText(e.Address) {
			continue
		}
		payload, err := a.Containers.ReadFile(e.Address)
		if err != nil {
			// A listed entry that will not read is a fact about the archive and
			// not about the corpus. It is counted nowhere rather than counted
			// as a file with no speaker, which would move the very figure this
			// tool exists to report.
			return fmt.Errorf("read a listed entry: %w", err)
		}
		files++

		pane := game.EventHasSpeaker(payload)
		if pane {
			paneByFile++
		}
		named := false
		// THE CENSUS IS TAKEN FOR ONE AUDIENCE, and states which. Part selection
		// now applies the eight conditional markup arms, so "how many parts does
		// this file have" has no audience-free answer. The zero audience is a male
		// fighter with no speaker resolvable, which is the audience every file's
		// unconditional tags satisfy.
		aud := game.EventAudience{}
		for n := 1; n <= partCeiling; n++ {
			if _, ok := game.EventPart(payload, n, aud); !ok {
				break
			}
			parts++
			if _, speaks := game.EventPartSpeaker(payload, n, aud); speaks {
				speakingParts++
				named = true
			}
		}
		if named {
			spokenByTag++
		}
		if pane != named {
			disagree++
		}
	}

	fmt.Fprintf(out, "event files                       %6d\n", files)
	fmt.Fprintf(out, "file test says a pane             %6d\n", paneByFile)
	fmt.Fprintf(out, "some part names a speaker         %6d\n", spokenByTag)
	fmt.Fprintf(out, "the two tests disagree            %6d\n", disagree)
	fmt.Fprintf(out, "parts                             %6d\n", parts)
	fmt.Fprintf(out, "parts naming a speaker            %6d\n", speakingParts)
	return nil
}

// isEventText reports whether an address names a campaign mission's event text.
//
// The match is on the ADDRESS SHAPE and not on a composed name: the prefix, a
// mission segment, the event stem and the extension. The case fold is the
// archive's own convention rather than a guess — the two preserved roots spell
// their container names differently — and it costs nothing to be right about.
func isEventText(addr string) bool {
	a := strings.ToLower(addr)
	if !strings.HasPrefix(a, eventPrefix) || !strings.HasSuffix(a, ".txt") {
		return false
	}
	rest := a[len(eventPrefix):]
	i := strings.Index(rest, eventStem)
	if i <= 0 {
		return false
	}
	// One segment between the prefix and the stem — the mission — and nothing
	// after the file name.
	return !strings.Contains(rest[:i], "/") && !strings.Contains(rest[i+len(eventStem):], "/")
}
