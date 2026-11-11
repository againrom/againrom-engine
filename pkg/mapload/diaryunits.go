package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// diaryRows is the length of the Units collection, which sizes the Player's
// Diary arrays.
func diaryRows(t *Table) int {
	if t == nil || t.Units == nil {
		return 0
	}
	return t.Units.Len()
}

// diaryUnits names the definition row and face each placement resolved to, keyed
// by the map unit id the placement carries.
func diaryUnits(units []alm.Unit, t *Table) map[uint16]sim.DiaryUnit {
	out := make(map[uint16]sim.DiaryUnit, len(units))
	// The Diary reads the first placement of an id whose row fits its byte; a
	// wider row stands for the id until such a placement follows.
	for _, u := range units {
		r := Resolve(u, t)
		if u.UnitID == 0 || !r.Found() || r.Index > maxTokenRow {
			continue
		}
		if held, taken := out[u.UnitID]; taken && (held.Row <= 255 || r.Index > 255) {
			continue
		}
		out[u.UnitID] = sim.DiaryUnit{Row: uint16(r.Index), Face: uint8(u.ClassSubID)}
	}
	return out
}
