package main

import (
	"fmt"
	"io"

	"againrom/pkg/formats/sav"
)

// party WALKS the object graph and prints the human participant's own
// characters — the state a resume can restore, beside the axes it cannot.
//
// It is the evidence tool for the walk, and it prints the walk's error BESIDE
// what the walk read rather than instead of it: a walk that desynchronised at
// the fourth actor still read three, and the offset the fourth failed at is the
// only thing that says where the programme is wrong.
//
// THE ITEM CODE IS PRINTED WHOLE AND SPLIT. Whole, because that is the value a
// consumer carries; split, because the four fields are what say a code is a real
// item rather than two bytes read at the wrong offset — ITEM-APPEAR-023's slot,
// row, material and shape, cut out of the word by its own builder's masks.
func party(out io.Writer, f *sav.File, n int) error {
	chars, rec, err := f.PartyWalk()
	fmt.Fprintf(out, "%d characters in the human participant's own subtree\n", len(chars))
	if rec != nil {
		// THE BYTES AFTER THE RECORD ARE THE EXTENT PROOF. The top-level
		// Player list continues where this record ends (SAV-DOC-053), so a
		// programme that tiled leaves an object tag naming Player, or the null
		// reference, at exactly that offset.
		fmt.Fprintf(out, "  record 1: %s @%#06x..%#06x (%d bytes), then % x -- "+
			"the Player list's next element (SAV-DOC-053)\n",
			rec.Class, rec.Off, rec.End, rec.End-rec.Off, after(f.Body, rec.End, 4))
		for _, d := range rec.Refs["Diary"] {
			fmt.Fprintf(out, "  its own Diary @%#06x..%#06x: %d dwords, %d words, "+
				"reference %#08x (its own key is %#08x)\n",
				d.Off, d.End, d.Counts["Journal"], d.Counts["JournalWords"],
				d.Value["D2C"], rec.Value["This"])
		}
	}
	for i, c := range chars {
		if n > 0 && i >= n {
			fmt.Fprintf(out, "  ... %d more\n", len(chars)-n)
			break
		}
		own := " "
		if c.Hero {
			own = "*"
		}
		fmt.Fprintf(out, " %s%2d  %-8s @%#06x  %-16q cell (%3d,%3d) fine %#02x,%#02x  id %-5d unit %-5d stage %d\n",
			own, i, c.Class, c.Off, c.Name, c.Col(), c.Row(), c.FineX, c.FineY,
			c.RuntimeID, c.MapUnitID, c.Stage)
		fmt.Fprintf(out, "        definition row %d\n", c.DefRow)
		fmt.Fprintf(out, "        body %d reaction %d mind %d spirit %d speed %d capacity %d\n",
			c.Stat(sav.StatBody), c.Stat(sav.StatReaction), c.Stat(sav.StatMind),
			c.Stat(sav.StatSpirit), c.Stat(sav.StatSpeed), c.Stat(sav.StatCapacity))
		fmt.Fprintf(out, "        health %d/%d regen %d   mana %d/%d regen %d   unknown %d,%d\n",
			c.Stat(sav.StatHealth), c.Stat(sav.StatHealthMax), c.Stat(sav.StatHealthRegen),
			c.Stat(sav.StatMana), c.Stat(sav.StatManaMax), c.Stat(sav.StatManaRegen),
			c.Stat(sav.StatOwnWeight), c.Stat(sav.StatLoad))
		fmt.Fprintf(out, "        skills %v   skill XP %v   total XP %d\n",
			c.SkillLevels, c.SkillXP, c.Experience)
		fmt.Fprintf(out, "        worn %d, carried %d of %d declared, spellbook %v (%d), diary %v (%d/%d)\n",
			len(c.Worn), len(c.Items), c.ItemCount, c.HasSpellbook, c.SpellCount,
			c.HasDiary, c.JournalLen, c.JournalWords)
		for _, w := range c.Worn {
			fmt.Fprintf(out, "          worn    %s\n", pieceLine(w))
		}
		for _, it := range c.Items {
			fmt.Fprintf(out, "          carried %s\n", pieceLine(it))
		}
	}
	if err != nil {
		fmt.Fprintln(out, "  WALK STOPPED:", err)
	}
	return nil
}

// after is up to n bytes of b from off, and nothing at all past the end. It is
// here so the terminator line prints what is there rather than panicking on a
// walk that ran to the last byte.
func after(b []byte, off, n int) []byte {
	if off < 0 || off >= len(b) {
		return nil
	}
	if off+n > len(b) {
		n = len(b) - off
	}
	return b[off : off+n]
}

// pieceLine is one item's code, whole and cut into ITEM-APPEAR-023's four
// fields.
func pieceLine(p sav.Piece) string {
	return fmt.Sprintf("%-7s code %#06x  slot %2d row %2d material %2d shape %d  head-row %2d  x%d",
		p.Class, p.Code, (p.Code>>8)&0xf, p.Code&0x1f, p.Code>>12, (p.Code>>5)&7, p.Row, p.Stack)
}
