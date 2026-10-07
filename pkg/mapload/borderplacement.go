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

func inBorderRing(x, y, w, h int) bool {
	return x < borderDepth || y < borderDepth || x >= w-borderDepth || y >= h-borderDepth
}
