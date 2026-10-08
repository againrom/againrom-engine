package mapload

import (
	"encoding/binary"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// nativeInitialBase records only the fresh constructor's established prefix.
// UNIT-CTOR-004 joins Unit Base initialization to SAV-1032 and
// SAV-HUMGAPS-449. Human skills 1..5 have separate maintained producers.
// The final two bytes remain unknown; this function never replays on rebuild.
func nativeInitialBase(skills *[data.SkillSlots]int32) sim.NativeActorBasis {
	b := sim.NativeActorBasis{BasePresent: true, BaseKnown: 0x003fffff, DefencePresent: true, DefenceKnown: 1 << 16}
	if skills != nil {
		for i := 1; i < data.SkillSlots; i++ {
			binary.LittleEndian.PutUint16(b.Base[2+2*i:4+2*i], uint16(skills[i]))
		}
	}
	return b
}

func nativeInitialModifier(b sim.NativeActorBasis, worn *[sim.EquipSlots]sim.ItemInstance, t *Table, general int32, humanoid, fighter bool) sim.NativeActorBasis {
	b = b.WithModifier([64]byte{})
	for slot := range worn {
		kind := worn[slot].Kind
		worn[slot] = SourceConstructedItem(worn[slot], t)
		worn[slot].Kind = kind
		b = sim.UpdateNativeEquipmentBasis(b, worn[slot], sim.NativeModifierAttach, general, humanoid, sim.NativeClass{Present: true, Fighter: fighter})
	}
	if humanoid && b.ModifierByteKnown(58) {
		b.Defence[16] = b.Modifier[58]
	}
	return b
}
