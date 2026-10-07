package mapload

import "againrom/pkg/data"

// Placements of a clothing layer against its anchor slot.
const (
	// LayerUnder is drawn just before the item worn in the anchor slot.
	LayerUnder int8 = -1
	// LayerOver is drawn just after it.
	LayerOver int8 = 1
)

// LayerItem returns the mod item a code names when that item is a clothing
// layer.
func LayerItem(t *Table, code uint16) (ModItem, bool) {
	if t == nil || code == 0 {
		return ModItem{}, false
	}
	for _, m := range t.Mods.Items {
		if m.Code == code && m.Layer != 0 {
			return m, true
		}
	}
	return ModItem{}, false
}

// ActiveLayers returns the layers a member wears that his pack still backs: a
// worn layer is one unit of its code held in the pack, so a layer whose unit was
// sold, dropped or traded away is no longer worn. carried is the pack's item
// codes, one per unit.
func ActiveLayers(layers, carried []uint16) []uint16 {
	if len(layers) == 0 {
		return nil
	}
	held := make(map[uint16]int, len(carried))
	for _, code := range carried {
		held[code]++
	}
	out := make([]uint16, 0, len(layers))
	for _, code := range layers {
		if held[code] > 0 {
			held[code]--
			out = append(out, code)
		}
	}
	return out
}

// AddLayers adds the defence and absorption of worn layers to the loadout, by
// the same sum the twelve slots use.
func AddLayers(l *data.Loadout, layers []uint16, t *Table) {
	if len(layers) == 0 || t == nil || t.Shapes == nil || t.Materials == nil {
		return
	}
	codes := make([]data.ItemCode, len(layers))
	for i, c := range layers {
		codes[i] = data.ItemCode(c)
	}
	data.FoldLayers(&l.Mod, codes, t.Shapes, t.Materials, t.Shields, t.Armors)
}

// MemberLayers is the layers a party member wears, limited to those his pack
// backs.
func MemberLayers(p PartyMember, t *Table) []uint16 {
	if len(p.Layers) == 0 {
		return nil
	}
	items := MemberCarriedItems(p, t)
	carried := make([]uint16, len(items))
	for i, it := range items {
		carried[i] = it.Code
	}
	return ActiveLayers(p.Layers, carried)
}
