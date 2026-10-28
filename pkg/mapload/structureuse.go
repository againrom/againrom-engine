package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// PointerSlots is the action slot of each map object, by object index, that
// a type-11c record with a nonzero gate names; the last record wins
// (R2-ENGINE-512). The slot counts the first Position action nodes that
// runtimeAction admits, the compiler's own source slot (R2-ENGINE-520). It is
// nil for a map without such a record.
func PointerSlots(m *alm.Map) []int32 {
	if m == nil {
		return nil
	}
	recs, ok := m.StructureActions()
	if !ok {
		return nil
	}
	src, err := m.Script()
	if err != nil {
		return nil
	}
	return pointerSlots(m.Objects, recs, src.Actions)
}

func pointerSlots(objects []alm.Object, recs []alm.StructureAction, actions []alm.ScriptNode) []int32 {
	var out []int32
	for _, r := range recs {
		if r.Gate == 0 || uint64(r.Position) >= uint64(len(actions)) {
			continue
		}
		slot := int32(0)
		for _, n := range actions[:r.Position] {
			if runtimeAction(n) {
				slot++
			}
		}
		for i, o := range objects {
			if uint32(o.Field12) != r.ID || !sim.PointerKind(uint16(byte(o.Kind))) {
				continue
			}
			if out == nil {
				out = make([]int32, len(objects))
			}
			out[i] = slot
		}
	}
	return out
}

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
