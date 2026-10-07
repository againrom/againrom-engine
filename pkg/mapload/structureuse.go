package mapload

import "againrom/pkg/sim"

// StructureUseMetadata resolves the two named original potion identities from
// the installed table. Only their ordinary one-shot pool effects are admitted.
func StructureUseMetadata(s *sim.Structure, kind uint16, table *Table) {
	s.Kind = kind
	if kind != 15 && kind != 16 {
		return
	}
	if table == nil || table.MagicItems == nil {
		return
	}
	name, effect := "Potion Big Healing", uint8(6)
	if kind == 16 {
		name, effect = "Potion Big Mana", 9
	}
	for row := 1; row < table.MagicItems.Len() && row <= 255; row++ {
		if table.MagicItems.EntryName(row) != name {
			continue
		}
		item := ItemInstanceFromCode(uint16(0x0e00|row), table)
		if len(item.Effects) == 1 && item.Effects[0].Kind == effect && (item.Effects[0].Mode == 0 || item.Effects[0].Mode == 8) && item.Effects[0].Operand <= 2147483647 {
			s.UseAmount = int32(item.Effects[0].Operand)
		}
		return
	}
}
