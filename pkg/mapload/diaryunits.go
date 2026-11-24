package mapload

import (
	"againrom/pkg/data"
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

// firstLookRow is the first Units row that shares row's type and face, the row
// one look is counted under; zero when row is its own first or the table
// holds no pair for it. Rows of one look repeat only where a table lists a look
// twice.
func firstLookRow(t *Table, row int) uint16 {
	if t == nil || t.Units == nil || row <= 0 || row >= t.Units.Len() {
		return 0
	}
	typeID, face, ok := data.UnitKeyPair(t.Units.EntryParams(row))
	if !ok {
		return 0
	}
	if first := data.FindUnit(t.Units, typeID, face); first > 0 && first < row {
		return uint16(first)
	}
	return 0
}

// DiaryFold maps every named Units row that repeats an earlier row's type and
// face to the first row of that look. The first game's table has none.
func DiaryFold(t *Table) map[uint16]uint16 {
	if t == nil || t.Units == nil {
		return nil
	}
	var out map[uint16]uint16
	for row := 1; row < t.Units.Len() && row <= maxTokenRow; row++ {
		if t.Units.EntryName(row) == "" {
			continue
		}
		if first := firstLookRow(t, row); first != 0 {
			if out == nil {
				out = map[uint16]uint16{}
			}
			out[uint16(row)] = first
		}
	}
	return out
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
		first := firstLookRow(t, r.Index)
		key := r.Index
		if first != 0 {
			key = int(first)
		}
		out[u.UnitID] = sim.DiaryUnit{Row: uint16(r.Index), Face: uint8(u.ClassSubID), First: first, NoServerKey: !serverKeyed(t, key)}
	}
	return out
}

// serverKeyed reports whether Units row row states a server id in 1..0x9ff,
// the range a server-keyed Diary counts.
func serverKeyed(t *Table, row int) bool {
	if t == nil || t.Units == nil || row < 0 || row >= t.Units.Len() {
		return false
	}
	id, ok := data.UnitServerID(t.Units.EntryParams(row))
	return ok && uint16(id) >= 1 && uint16(id) <= 0x9ff
}
