package game

import (
	"encoding/binary"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// nativeCityGraftableMember is exactly the shape the fully native branch of
// missionCityProvenance already admits with no live entity: no retained Carry,
// and the one companion identity a town-chapter grant can add
// (addChapterCompanions). Any other unbound member is refused by name rather
// than silently guessed at.
func nativeCityGraftableMember(member mapload.PartyMember) bool {
	return member.Carry == nil && member.CompanionNPC == 22
}

// graftNativeCityMember appends one natively constructed Human object, and its
// fresh starting equipment, to a document-derived city DTO for a member the
// source document never described. It mints every new identity above every
// identity already present in data, so the duplicate check
// (*CityProvenance).Marshal's own remintCityIdentities runs is a hard failure
// on a real collision, never a silent one -- Marshal remints every identity
// before the bytes are final regardless of what is minted here.
func graftNativeCityMember(data sav.CityData, member, hero mapload.PartyMember, table *mapload.Table) (sav.CityData, error) {
	playerIndex := -1
	for i, obj := range data.Objects {
		if obj.Class == "Player" && obj.Player != nil {
			playerIndex = i
			break
		}
	}
	if playerIndex < 0 || len(data.Objects[playerIndex].Player.Groups) == 0 {
		return data, originalCityUnsupportedf("mission city has no Player group to add a companion to")
	}
	playerFixed := data.Objects[playerIndex].Player.Fixed
	if len(playerFixed) < 51 {
		return data, originalCityUnsupportedf("mission city Player object has no identity to graft a companion onto")
	}
	// The document's real Player identity, not the native constructor's own
	// from-scratch placeholder (nativeCityPlayerIdentity): a document parsed
	// from a real save mints its own identities, and the grafted unit's owner
	// reference has to name the object that is actually in this graph.
	playerIdentity := binary.LittleEndian.Uint32(playerFixed[47:51])
	identity := nativeCityNextIdentity(data)
	unit := nativeCityUnitData(identity, playerIdentity, member, hero, table,
		[sav.UnitStatWords]uint16{}, [sav.CharacterSkillSlots]uint16{})
	binary.LittleEndian.PutUint32(unit.Token[12:16], nextSavedRuntimeID(cityRuntimeIDs(data.Objects)))
	if err := nativeCityInitializeHumanMovement(&unit, table); err != nil {
		return data, err
	}
	itemSeq := 0
	objects, err := nativeCityAttachItems(data.Objects, &unit, member, table, identity, &itemSeq)
	if err != nil {
		return data, err
	}
	if err := nativeCityApplyHuman(&unit, member, table); err != nil {
		return data, err
	}
	objects = append(objects, sav.CityObjectData{Class: "Human", Unit: &unit})
	data.Objects = objects
	group := &data.Objects[playerIndex].Player.Groups[0]
	group.Actors = append(append([]uint16(nil), group.Actors...), uint16(len(data.Objects)))
	return data, nil
}

// nativeCityNextIdentity returns one identity above every identity already
// present in data, so a freshly minted object can never collide with a real
// decoded one. The field offsets match cityObjectIdentity's own layout.
func nativeCityNextIdentity(data sav.CityData) uint32 {
	next := uint32(0x10)
	consider := func(v uint32) {
		if v >= next {
			next = v + 0x10
		}
	}
	for _, obj := range data.Objects {
		switch {
		case obj.Player != nil && len(obj.Player.Fixed) >= 51:
			consider(binary.LittleEndian.Uint32(obj.Player.Fixed[47:51]))
		case obj.Unit != nil && len(obj.Unit.Token) >= 37:
			consider(binary.LittleEndian.Uint32(obj.Unit.Token[29:33]))
		case obj.Item != nil && len(obj.Item.Token) >= 37:
			consider(binary.LittleEndian.Uint32(obj.Item.Token[29:33]))
		case obj.Effect != nil && len(obj.Effect.Token) >= 37:
			consider(binary.LittleEndian.Uint32(obj.Effect.Token[29:33]))
		case obj.Spell != nil && len(obj.Spell.Fields) >= 9:
			consider(binary.LittleEndian.Uint32(obj.Spell.Fields[5:9]))
		}
	}
	return next
}
