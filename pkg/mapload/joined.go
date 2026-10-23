package mapload

import (
	"fmt"
	"maps"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
)

// CharacterSpec names one persistent party member to build: a campaign NPC
// record, or a definition row of the Humans collection, with the face and the
// equipment cells a party declaration sets over that row.
type CharacterSpec struct {
	// NPC is the scenario npc record the member is; zero builds Row.
	NPC int32
	// Mission is the mission a composed NPC record is resolved for: its
	// forty-mission band selects the row (HERO-JOIN-120).
	Mission int
	// Row explicitly selects a Humans row, also for an NPC without DataBinID.
	Row int
	// Face, when positive, replaces the row's face column.
	Face int32
	// Cells replace the row's equipment cells by position (data.HumanCell*).
	Cells map[int]string
}

// CampaignNPCMember builds the persistent party template behind one campaign
// NPC record, resolved for mission against the party's primary member.
func CampaignNPCMember(t *Table, npc int32, mission int, party []PartyMember) (PartyMember, bool) {
	return CharacterMember(t, CharacterSpec{NPC: npc, Mission: mission}, party)
}

// CharacterMember builds the persistent party template s names. It uses the
// same Humans row, item-instance resolver and roster constructor as an NPC
// placement, so a town AddHero, a map-script handover and a mod's guest party
// cannot disagree about statistics, spellbook, carried items or worn effects.
// A face or equipment cell s sets is read through an edited copy of the row,
// the install-wide table unchanged.
//
// An NPC member's identity is stable across missions and saves. Its localized
// display name and installed body are presentation facts applied by pkg/game;
// this tier retains the row name and exact numeric payload.
func CharacterMember(t *Table, s CharacterSpec, party []PartyMember) (PartyMember, bool) {
	if t == nil || t.Humans == nil {
		return PartyMember{}, false
	}
	var serverID int32
	row := s.Row
	if s.NPC != 0 && row == 0 {
		if t.NPC == nil || len(party) == 0 {
			return PartyMember{}, false
		}
		primary := party[0]
		for _, member := range party {
			if member.StartingHero {
				primary = member
				break
			}
		}
		id, ok := t.NPC.CampaignServerID(s.NPC, s.Mission, primary.Mage,
			data.FigureDir(primary.FigureDir).Female())
		if !ok {
			return PartyMember{}, false
		}
		serverID, row = id, data.FindHumanByServerID(t.Humans, id)
		if row == data.NotFound {
			return PartyMember{}, false
		}
	} else {
		if row < 1 || row >= t.Humans.Len() {
			return PartyMember{}, false
		}
		id, ok := data.HumanServerID(t.Humans.EntryParams(row))
		if !ok || data.FindHumanByServerID(t.Humans, id) != row {
			return PartyMember{}, false
		}
		serverID = id
	}
	local := t
	if s.Face > 0 || len(s.Cells) > 0 {
		edit := data.RowEdit{Cells: s.Cells}
		if s.Face > 0 {
			edit.Params = map[int]int32{data.HumanSlotFace: s.Face}
		}
		edited := *t
		edited.Humans = data.NewEditedRows(t.Humans, map[int]data.RowEdit{row: edit})
		local = &edited
	}
	// A definition-id placement reaches the same Humans row without depending
	// on the table's mission-local composed lookup. The constructor below then
	// resolves the row's complete item cells through the ordinary placement path.
	u := alm.Unit{DefID: uint32(serverID)}
	member, ok := rosterTemplate(u, local, 0)
	if !ok {
		return PartyMember{}, false
	}
	if s.NPC != 0 {
		member.ID = fmt.Sprintf("npc:%d", s.NPC)
		member.CompanionNPC = int(s.NPC)
	}
	member.StartingHero = false
	member.PlayerCharacter = true
	return member, true
}

// CheckCharacterCells refuses an equipment cell that names no item of its
// cell's kind: the weapon cell a weapon, the shield cell a shield, every
// other cell an armour piece. An empty cell is an empty slot.
func CheckCharacterCells(t *Table, cells map[int]string) error {
	for _, pos := range slices.Sorted(maps.Keys(cells)) {
		name := cells[pos]
		if name == "" {
			continue
		}
		var err error
		switch pos {
		case data.HumanCellWeapon:
			_, _, err = resolveWeaponCell(name, t)
		case data.HumanCellShield:
			_, _, err = resolveShieldCell(name, t)
		default:
			_, _, err = resolveArmorCell(name, t)
		}
		if err != nil {
			return fmt.Errorf("%q: %v", name, err)
		}
	}
	return nil
}
