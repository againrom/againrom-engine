package game

import (
	"encoding/binary"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The original's tavern spawn writes a hired actor with these constructor
// values, in a Unit and in a Human alike: the actor state word 0x0b (U50), a
// container insert index of 10000, U136 = 1 and the mover pair 5, 255 at
// U154+8. Two original city resaves of the kit hold them on the hired
// Ballista, and the hired Human squads of the same files hold the same four.
const (
	siegeHireStateWord   = 0x0b
	siegeHireInsertIndex = 10000
	siegeHireU136        = 1
)

// nativeCitySiegeUnit builds the Unit object a hired Catapult or Ballista is
// written as. The Units row supplies every combat, footprint and load field
// through the constructor a map creature takes; the hire-spawn constants above
// replace the fields the original sets differently at a tavern hire. The
// record holds no name, no spellbook and no map unit id, as the original's
// hired Unit holds none. The worn weapon is attached by the holdings writer.
func nativeCitySiegeUnit(identity, owner uint32, member mapload.PartyMember, table *mapload.Table) (sav.CityUnitData, error) {
	hire, err := mapload.SiegeHireActor(member, table, identity)
	if err != nil {
		return sav.CityUnitData{}, originalCityUnsupportedf("native city %s has no Units row to bind its Unit: %v", member.Name, err)
	}
	b := hire.Basis
	s := b.ActorLoad.Source
	binding := b.SourceBinding
	token := nativeCityToken(identity, owner, binding.TokenRow, binding.TypeID)

	scalar1 := make([]byte, 19)
	domain := uint32(b.Domain) + 1
	scalar1[0], scalar1[1], scalar1[2], scalar1[3] = b.TokenSize, byte(domain), binding.Face, binding.ClassFlags
	binary.LittleEndian.PutUint32(scalar1[4:8], siegeHireStateWord)

	scalar2 := make([]byte, 55)
	for i, v := range s.Stats {
		binary.LittleEndian.PutUint16(scalar2[2*i:], v)
	}
	binary.LittleEndian.PutUint16(scalar2[30:], s.ManaFloor)
	binary.LittleEndian.PutUint16(scalar2[32:], s.Sight)
	scalar2[34], scalar2[39], scalar2[40], scalar2[41] = s.Reach, s.AttackCharge, s.AttackRelax, siegeHireU136
	binary.LittleEndian.PutUint32(scalar2[35:], s.Experience)
	binary.LittleEndian.PutUint32(scalar2[47:51], uint32(member.MercenaryType))

	mover := make([]byte, 180)
	mover[5], mover[8], mover[9], mover[10] = sim.MoverPassabilityMask(domain), 5, 255, s.MoverSpeed

	return sav.CityUnitData{
		Token:          token,
		RawA6:          s.Attack[:],
		RawBE:          s.Defence[:],
		Raw114:         s.Base[:],
		RawD4:          s.Modifier[:],
		Raw154:         mover,
		Raw158:         make([]byte, 148),
		Scalar1:        scalar1,
		Scalar2:        scalar2,
		ContainerFlag:  1,
		ContainerTails: [2]uint32{siegeHireInsertIndex, 0},
		ScalarTail:     make([]byte, 17),
	}, nil
}

// cityMemberEquipment is the worn set a city SAVE writes for member. A hired
// siege squad's weapon comes from its Units row, as it does at mission entry,
// because the party member holds no worn items of its own.
func cityMemberEquipment(member mapload.PartyMember, table *mapload.Table) [sim.EquipSlots]sim.ItemInstance {
	items := mapload.MemberItemEquipment(member, table)
	if !nativeCitySiegeMember(member) {
		return items
	}
	for _, item := range items {
		if !item.Empty() {
			return items
		}
	}
	if hire, err := mapload.SiegeHireActor(member, table, 0); err == nil {
		return hire.Worn
	}
	return items
}
