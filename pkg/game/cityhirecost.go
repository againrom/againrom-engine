package game

import "againrom/pkg/mapload"

// cityHireCost is the total price of the squads the town holds as hired. A
// town SAV stores it in the document's global dword, the cell after the
// document marker, which is the original tavern's accumulated hire cost
// (MERC-CMD-007: the static tavern sits at L08104 and its +0x9c is
// L07886, the word SAV-TRAIL-026 writes after the marker). The original
// refunds that cost when the tavern opens and charges the hired set again when
// it closes, so a SAV that holds hire flags and a purse after the debit must
// also hold the cost, or the first visit after LOAD charges the hires twice.
func cityHireCost(s Snapshot, table *mapload.Table) uint32 {
	var total uint64
	for typ := 1; typ < len(s.MercenaryHired); typ++ {
		if !s.MercenaryHired[typ] {
			continue
		}
		if price, ok := mercenarySquadPrice(table, s.MainMission, typ, s.MercenaryPool[typ]); ok && price > 0 {
			total += uint64(price)
		}
	}
	if total > 0xffffffff {
		return 0xffffffff
	}
	return uint32(total)
}
