package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// Outposts is what the map gives the type-10 pass: the gate, type-0 dword
// 0x84 above 1 (R2-ENGINE-509); whether the build keeps the members' spread
// (t.OutpostSpread, R2-ENGINE-511); and the type-10 entries in file order.
func Outposts(m *alm.Map, t *Table) sim.OutpostMap {
	if m == nil {
		return sim.OutpostMap{}
	}
	out := sim.OutpostMap{Runs: m.Meta.Word70 > 1, Spread: t != nil && t.OutpostSpread}
	entries, _ := m.OutpostEntries()
	for _, e := range entries {
		out.Entries = append(out.Entries, sim.OutpostEntry{Group: e.Group, Percent: e.Percent, Flags: e.Flags})
	}
	return out
}
