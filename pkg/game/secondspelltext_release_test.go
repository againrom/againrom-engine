package game

import (
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// The second game names each spell row r by its own spell.txt line r-1, all
// 29 rows, and each spellbook row by the spells.txt line of its slot through
// the slot table (R2-ENGINE-444, R2-ASSET-103); the first game's Drain Life
// alias names no second-game row. Loss control: the first game's slot table
// joins a different line to some row.
func TestReleaseSecondGameSpellTextIsItsOwnLines(t *testing.T) {
	f := secondGameFront(t)
	selector := LanguageSelector(f.Archives.Containers)
	names := secondGameShippedRows(t, f, SpellNamesTextPath)
	book := secondGameShippedRows(t, f, SpellBookNamesTextPath)
	if names.Lines() != 29 || book.Lines() != 24 {
		t.Fatalf("spell.txt %d lines, spells.txt %d lines; want 29 and 24", names.Lines(), book.Lines())
	}
	for r := 1; r <= 29; r++ {
		shipped, _ := names.At(r - 1)
		if got := f.Words.ItemSpellNames[r]; got == "" || !secondGameWordReads(selector, shipped, got) {
			t.Errorf("row %d item name %q, spell.txt line %d %q", r, got, r-1, shipped)
		}
	}
	defs, err := LoadDefinitionsFor(f.Archives.Containers, f.Archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	byRow := spellNamesFrom(defs.Table, f.Words.SpellBookNames)
	if n := len(mapload.SpellRules(defs.Table)); n != 29 {
		t.Fatalf("SpellRules = %d rows, want 29", n)
	}
	join := func(rows []uint16) (missed int) {
		for s, r := range rows {
			shipped, _ := book.At(s)
			shipped, _, _ = strings.Cut(shipped, "#")
			if !secondGameWordReads(selector, shipped, byRow[r]) {
				missed++
			}
		}
		return missed
	}
	if missed := join(data.SpellPresentationNamed(f.Archives.Game().Edition().Spells).BookRows); missed != 0 {
		t.Errorf("%d spellbook rows do not read their slot's spells.txt line", missed)
	}
	if join(originalBookIDs) == 0 {
		t.Error("the first game's slot table joined every line too: the join is not exercised")
	}
}
