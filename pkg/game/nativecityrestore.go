package game

import (
	"fmt"

	"againrom/pkg/mapload"
)

const legacyCityHireRebuildCap = 12

// restoreHiredMercenaries rebuilds a hired squad a loaded town SAV could not
// restore from its own actor record. SAV-616 retracts that finding for
// classKey 58 specifically — a Human-type hire (tavern type 3..15) now has
// its own real record, decoded by RestoreParty/restoredMember through
// mercenaryHireTypeFromName (originalparty.go) and already installed in
// the session's Carried by the time this runs (RestoreOriginal). This function is now
// that path's FALLBACK only: per type, it rebuilds through
// buildMercenarySquad — the same tavern template builder the
// live hire action itself calls — only when no member of that type already
// reached f.Carried some other way. Two shapes still take the fallback: a
// siege engine (typ 1 or 2, Catapult/Ballista) whose file holds no Unit actor
// on a siege row, such as a file with only the legacy Human actor of Units row
// 0 (the Unit actor is read by restoredMember); and a file written before
// native hire records, whose set hire flag has no persisted record at all for
// any type, native or siege. The saved working-pool cell
// stays authoritative while hired; a rebuilt party row must not consume it
// (MERC-POOL-012, SAV-1085).
func (s *CampaignSession) restoreHiredMercenaries(table *mapload.Table) error {
	if s == nil || s.Town == nil {
		return nil
	}
	t := s.Town
	present := make(map[int]bool, len(s.Carried))
	for _, member := range s.Carried {
		if member.MercenaryType != 0 {
			present[int(member.MercenaryType)] = true
		}
	}
	insertAt := len(s.Carried)
	for i, member := range s.Carried {
		if member.MercenaryType != 0 {
			insertAt = i
			break
		}
	}
	for typ := 1; typ < len(t.mercHired); typ++ {
		if !t.mercHired[typ] {
			continue
		}
		count := t.mercPool[typ]
		if present[typ] {
			continue
		}
		if count <= 0 {
			continue
		}
		if count > legacyCityHireRebuildCap {
			return fmt.Errorf("original city save: hired type %d's own MercenaryWorking cell is %d, over the %d-member roster cap", typ, count, legacyCityHireRebuildCap)
		}
		members, ok := buildMercenarySquad(table, t, typ, count)
		if !ok {
			continue
		}
		grown := make([]mapload.PartyMember, 0, len(s.Carried)+len(members))
		grown = append(grown, s.Carried[:insertAt]...)
		grown = append(grown, members...)
		grown = append(grown, s.Carried[insertAt:]...)
		// THE SHIFTED TAIL'S OWN STABLE IDS GO STALE HERE, NOT ONLY THE
		// SQUAD THIS LOOP JUST BUILT. mapload.NameParty (party.go) mints a
		// hired member's id from its own ARRAY POSITION
		// (mercenary:<type>:<index+1>) and never renames a member that
		// already carries one -- correct for RestoreParty's own earlier
		// OwnParty call, which had no siege member in front yet to shift
		// anything. Inserting here moves every already-restored Human hire
		// at or past insertAt down by len(members); clearing their own ID
		// (never anything before insertAt, which does not move) lets
		// OwnParty re-derive them against their TRUE final position -- the
		// same position mapload.NameParty's own single mint already gives
		// the live, never-reloaded party, so the reload keeps the exact ids
		// a fresh hire in this same visit order would carry, not the ones
		// an incomplete intermediate array happened to mint first.
		for i := insertAt + len(members); i < len(grown); i++ {
			grown[i].ID = ""
		}
		// OwnParty always replaces the backing array (CloneParty's own
		// make+copy), never mutates in place: bindOriginalCity's earlier
		// overlay already wrote into the OLD array, and mutating it here
		// would corrupt that binding rather than replace it.
		s.Carried = mapload.OwnParty(grown)
		insertAt += len(members)
	}
	return nil
}
