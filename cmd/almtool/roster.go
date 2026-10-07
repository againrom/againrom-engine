package main

import (
	"fmt"

	"againrom/pkg/formats/alm"
)

// cmdRoster prints the map's own player roster and the diplomacy matrix that
// roster authors: for each type-5 record its 1-based slot, its name, how many
// placed units name that slot as owner, its sixteen raw relation words, and the
// effective matrix row.
//
// WHY IT EXISTS. Every engagement decision the simulation takes is indexed by an
// owner slot, and until this verb the only way to see what a map actually says
// about two factions was to build a world and watch what happened. That answers
// "did somebody attack" and never "was the map asking for it" — so a question
// about behaviour could not be told apart from a question about content, which is
// the shape of the report this verb was written for.
//
// THE EFFECTIVE ROW IS RE-DERIVED HERE AND IS NOT THE LOADER'S. It cannot be:
// internal/archtest holds cmd/almtool -> pkg/sim as a forbidden edge, and the
// loader's answer is a sim type, so this tool cannot hold it at all. What is
// printed is therefore a reading of the FILE by the published rule
// (AI-DIPLO-005, ALM-GRP-041) — word k narrowed to its low byte at column k+1,
// then the diagonal forced to 2 — and it witnesses the map rather than the
// loader. Column 0 is not printed because nothing ever writes it: the file's row
// is 0-based over records and the matrix is 1-based, so no slot names it.
func cmdRoster(path string) error {
	m, _, err := loadMap(path)
	if err != nil {
		return err
	}
	// The owner census is over the WHOLE type-6 array and is counted before
	// anything is printed, so a slot with no placements prints a 0 rather than
	// being absent from the table.
	owned := make(map[uint32]int, len(m.Groups))
	for _, u := range m.Units {
		owned[u.Owner]++
	}

	fmt.Printf("roster (type5): %d record(s), %d placed unit(s)\n", len(m.Groups), len(m.Units))
	fmt.Printf("slot  name                              units  relation words (raw u16, k=0..15)\n")
	for i, g := range m.Groups {
		slot := uint32(i) + 1
		fmt.Printf("%4d  %-32q %5d  %v\n", slot, g.Name, owned[slot], g.Relation)
	}
	if unnamed := owned[0]; unnamed > 0 {
		fmt.Printf("      %d placed unit(s) name no slot (owner word 0)\n", unnamed)
	}
	for slot, n := range owned {
		if slot != 0 && int(slot) > len(m.Groups) {
			fmt.Printf("      %d placed unit(s) name slot %d, which this roster does not hold\n", n, slot)
		}
	}

	if len(m.Groups) == 0 {
		return nil
	}
	fmt.Printf("\neffective matrix [from][to], low byte with the diagonal forced to 2;\n")
	fmt.Printf("bit 0 is hostile, so an odd cell is a slot that acquires that slot's units\n")
	fmt.Printf("  from")
	for to := 1; to <= len(m.Groups); to++ {
		fmt.Printf("%4d", to)
	}
	fmt.Printf("   <- to\n")
	for i := range m.Groups {
		from := i + 1
		fmt.Printf("%6d", from)
		for to := 1; to <= len(m.Groups); to++ {
			fmt.Printf("%4d", effectiveCell(m.Groups, from, to))
		}
		fmt.Println()
	}
	return nil
}

// effectiveCell is the byte the engine's map-load store leaves at [from][to],
// with from and to 1-based slots.
//
// It is ONE function so that the narrowing and the forced diagonal are written
// down once and in the order the store applies them: the row is written first
// and the diagonal is forced afterwards and unconditionally, which is why a
// shipped roster spelling its own diagonal as 1 or 0 — five of them do — still
// yields a slot that is not at war with itself.
//
// A column past the sixteen a record carries has no word behind it and reads 0,
// which is what a roster of more than sixteen records would leave; the editor
// caps a roster at sixteen and no shipped map comes near it.
func effectiveCell(groups []alm.Group, from, to int) byte {
	if from == to {
		return 2
	}
	row := groups[from-1].Relation
	if to-1 >= len(row) {
		return 0
	}
	return byte(row[to-1])
}
