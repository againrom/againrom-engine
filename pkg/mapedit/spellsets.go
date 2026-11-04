package mapedit

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"
)

type spellSpan struct{ start, end int }
type spellSets [][]spellSpan

// GroupSpellRecords remembers one stroke's traps in this session's history.
func (e *Editor) GroupSpellRecords(indices []int) error {
	w, err := e.walkSection(KindEnchantment, "group traps")
	if err != nil {
		return err
	}
	group := make([]spellSpan, 0, len(indices))
	for _, i := range indices {
		if i < 0 || i >= w.n() {
			return fmt.Errorf("mapedit: trap index outside section")
		}
		at, n := w.at(i)
		rec := e.data[at : at+n]
		if !spellRecordKnown(rec) || binary.LittleEndian.Uint16(rec[12:]) >= 4 || binary.LittleEndian.Uint32(rec[4:]) == 0 && binary.LittleEndian.Uint32(rec[8:]) == 0 {
			return fmt.Errorf("mapedit: a trap set contains a non-trap record")
		}
		span := spellSpan{at, at + n}
		grouped := false
		for _, existing := range e.setsAt[e.cursor] {
			grouped = grouped || slices.Contains(existing, span)
		}
		if slices.Contains(group, span) || grouped {
			return fmt.Errorf("mapedit: trap is already grouped")
		}
		group = append(group, span)
	}
	if len(group) == 0 {
		return fmt.Errorf("mapedit: empty trap set")
	}
	e.setsAt[e.cursor] = append(slices.Clone(e.setsAt[e.cursor]), group)
	return nil
}

// SpellSet returns the surviving records of index's stroke, or none for an
// ungrouped record. Opening saved bytes starts a session with no sets.
func (e *Editor) SpellSet(index int) []int {
	for _, set := range e.SpellSets() {
		if slices.Contains(set, index) {
			return set
		}
	}
	return nil
}

func (e *Editor) SpellSets() [][]int {
	groups := e.setsAt[e.cursor]
	if len(groups) == 0 {
		return nil
	}
	w, err := e.walkSection(KindEnchantment, "find trap sets")
	if err != nil {
		return nil
	}
	membership := make(map[spellSpan]int)
	for g, group := range groups {
		for _, span := range group {
			membership[span] = g
		}
	}
	out := make([][]int, len(groups))
	for i := 0; i < w.n(); i++ {
		at, n := w.at(i)
		if g, ok := membership[spellSpan{at, at + n}]; ok {
			out[g] = append(out[g], i)
		}
	}
	return out
}

func (e *Editor) shiftSpellSets(ed edit) spellSets {
	sets := e.setsAt[e.cursor]
	if len(sets) == 0 {
		return nil
	}
	ordered := slices.Clone(ed)
	slices.SortFunc(ordered, func(a, b part) int { return cmp.Compare(b.off, a.off) })
	out := make(spellSets, len(sets))
	for i, group := range sets {
		for _, span := range group {
			alive := true
			for _, p := range ordered {
				end, delta := p.off+len(p.removed), len(p.inserted)-len(p.removed)
				switch {
				case p.off <= span.start && end >= span.end:
					alive = false
				case end <= span.start:
					span.start, span.end = span.start+delta, span.end+delta
				case p.off < span.end:
					span.end += delta
				}
				if !alive {
					break
				}
			}
			if alive {
				out[i] = append(out[i], span)
			}
		}
	}
	return out
}
