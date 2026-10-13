package mapload

import "againrom/pkg/formats/alm"

// WithdrawBorderPlacements removes, in place, every placed unit whose cell is in
// the border ring and returns how many it removed. The original loader refuses
// a ring cell and deletes the actor (TERR-PLACE-208), so such a placement is
// no entity and takes no id. Interior blocked cells keep their placements.
func WithdrawBorderPlacements(m *alm.Map) int {
	if m == nil || len(m.Units) == 0 {
		return 0
	}
	w, h := int(m.Width), int(m.Height)
	kept := m.Units[:0]
	for _, u := range m.Units {
		x, y := unitCell(u)
		if inBorderRing(int(x), int(y), w, h) {
			continue
		}
		kept = append(kept, u)
	}
	n := len(m.Units) - len(kept)
	m.Units = kept
	return n
}

// InBorderRing reports whether cell x,y is in the border ring.
func InBorderRing(m *alm.Map, x, y int) bool {
	return inBorderRing(x, y, int(m.Width), int(m.Height))
}

func inBorderRing(x, y, w, h int) bool {
	return x < borderDepth || y < borderDepth || x >= w-borderDepth || y >= h-borderDepth
}

// withdrawnBefore lists, for each ring placement WithdrawBorderPlacements took
// out of m.Units, how many kept units precede it in record order. The
// original constructs such an actor, drawing its facing, before the cell
// refusal deletes it (MOVE-120, MOVE-141). It is empty when nothing was
// withdrawn, or when the authored records cannot be reconciled with Units.
func withdrawnBefore(m *alm.Map) []int {
	if m == nil || len(m.AuthoredUnits) <= len(m.Units) {
		return nil
	}
	w, h := int(m.Width), int(m.Height)
	var out []int
	kept := 0
	for _, u := range m.AuthoredUnits {
		x, y := unitCell(u)
		if inBorderRing(int(x), int(y), w, h) {
			out = append(out, kept)
			continue
		}
		kept++
	}
	if kept != len(m.Units) {
		return nil
	}
	return out
}
