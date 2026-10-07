package main

import (
	"fmt"
	"io"
	"os"

	"againrom/pkg/formats/sav"
)

// lead measures the two rules that can put a character at the head of a
// restored party, over any number of save files, and reports where they
// disagree.
//
// IT EXISTS BECAUSE THE STORY THAT CHANGED THE RULE CLAIMED THEY AGREE. A
// corpus figure that no committed command reproduces is a figure a later reader
// has to take on trust, and the population itself grows: the preserved corpus
// was 23 files when this was written and every new owner save is another row.
//
// The two rules:
//
//   - the FILE'S OWN FIELD (SAV-HERO-059) — Player+0x34 is an identity key
//     naming the participant's own starting character, and it equals the key of
//     exactly one actor in that participant's own groups.
//   - the RUNTIME ID (SAV-ID-015) — the actor carrying creation-order id 1,
//     which is this tree's own inference from the allocator and is now the
//     fallback.
//
// A file where either rule matches no character, or more than one, is reported
// with its counts rather than dropped: those are the two cases the fallback
// exists for, and a corpus that produced one would be the news.
func lead(out io.Writer, paths []string) error {
	files, agree, disagree, unresolved := 0, 0, 0, 0
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files++
		f, err := sav.Open(b)
		if err != nil {
			fmt.Fprintf(out, "%-40s OPEN FAILED: %v\n", path, err)
			unresolved++
			continue
		}
		chars, rec, werr := f.PartyWalk()
		named, namedN := onlyOne(chars, func(c sav.Character) bool { return c.Hero })
		byID, byIDN := onlyOne(chars, func(c sav.Character) bool { return c.RuntimeID == 1 })
		var key uint32
		if rec != nil {
			key = rec.Value["Hero"]
		}
		state := "AGREE"
		switch {
		case namedN != 1 || byIDN != 1:
			state = "UNRESOLVED"
			unresolved++
		case named != byID:
			state = "DISAGREE"
			disagree++
		default:
			agree++
		}
		fmt.Fprintf(out, "%-40s %-10s %d character(s), Player names %#08x: "+
			"by field %d (%d match), by runtime id %d (%d match)\n",
			path, state, len(chars), key, named, namedN, byID, byIDN)
		if state != "AGREE" {
			for i, c := range chars {
				fmt.Fprintf(out, "    %2d %-16q key %#08x runtime id %d\n",
					i, c.Name, c.Key, c.RuntimeID)
			}
		}
		if werr != nil {
			fmt.Fprintf(out, "    WALK STOPPED: %v\n", werr)
		}
	}
	fmt.Fprintf(out, "%d file(s): %d agree, %d disagree, %d unresolved\n",
		files, agree, disagree, unresolved)
	if disagree > 0 {
		return fmt.Errorf("the two rules select different characters on %d file(s)", disagree)
	}
	return nil
}

// onlyOne is the index of the single character satisfying want and how many
// satisfied it. The count is returned rather than a boolean because "no
// character" and "several" are different facts about a save and the report
// prints both.
func onlyOne(chars []sav.Character, want func(sav.Character) bool) (int, int) {
	at, n := -1, 0
	for i, c := range chars {
		if !want(c) {
			continue
		}
		n++
		if n == 1 {
			at = i
		}
	}
	if n != 1 {
		return -1, n
	}
	return at, n
}
