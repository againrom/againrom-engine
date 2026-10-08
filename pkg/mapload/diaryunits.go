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
	for _, u := range units {
		r := Resolve(u, t)
		if _, taken := out[u.UnitID]; u.UnitID == 0 || taken || !r.Found() || r.Index > 255 {
			continue
		}
		out[u.UnitID] = sim.DiaryUnit{Row: uint8(r.Index), Face: uint8(u.ClassSubID)}
	}
	return out
}
