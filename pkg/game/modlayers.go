package game

import (
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// figureLayer is one clothing layer composed into a doll: the equipment slot it
// is placed against, whether it goes under or over what is worn there, and the
// item whose worn picture it is drawn with.
type figureLayer struct {
	Anchor int
	Depth  int8
	Code   data.ItemCode
}

// paths are the pictures the layer paints: the item's figure sheet and, for an
// anchor slot that draws a second sheet, the second.
func (l figureLayer) paths(dir data.FigureDir) []string {
	out := []string{data.ItemFigureLayerPath(dir, l.Code)}
	if data.HasItemFigureSecondaryLayer(l.Anchor) {
		out = append(out, data.ItemFigureSecondaryLayerPath(dir, l.Code))
	}
	return out
}

// figureLayersFor turns worn layer codes into what the doll composes. A code
// no mod item names as a layer draws nothing.
func figureLayersFor(layers []uint16, t *mapload.Table) []figureLayer {
	var out []figureLayer
	for _, code := range layers {
		if m, ok := mapload.LayerItem(t, code); ok {
			out = append(out, figureLayer{Anchor: int(m.Anchor), Depth: m.Layer, Code: data.ItemCode(m.Code)})
		}
	}
	return out
}

// activeLayers is the layers the member of entity id wears that the entity's
// pack still backs. A layer whose unit left the pack is not worn, so a sale, a
// drop or a trade takes the layer off. The member is not changed: a mission
// opens before its pack is filled, and a layer must survive that moment.
func (mw *mapWorld) activeLayers(id sim.EntityID) []uint16 {
	member := mw.missionPartyMember(id)
	if member == nil || len(member.Layers) == 0 {
		return nil
	}
	carried, ok := mw.world.Carried(id)
	if !ok {
		return slices.Clone(member.Layers)
	}
	return mapload.ActiveLayers(member.Layers, carried)
}

// toggleLayer wears the layer item, or takes it off when it is already worn. A
// layer is one unit of the pack item that stays in the pack, so its weight is
// the pack's. Wearing replaces the layer already worn at the same anchor and
// depth. The wear rule of the item's table applies to the subject, and an actor
// whose statistics come from an original save refuses a layer, since its
// derived values are not recomputed from items.
func (mw *mapWorld) toggleLayer(id sim.EntityID, layer mapload.ModItem) {
	member := mw.missionPartyMember(id)
	if member == nil {
		return
	}
	member.Layers = slices.Clone(mw.activeLayers(id))
	if i := slices.Index(member.Layers, layer.Code); i >= 0 {
		member.Layers = slices.Delete(member.Layers, i, i+1)
		return
	}
	if e, ok := mw.entity(id); !ok || e.ActorLoad.Source.Class != 0 {
		mw.refuseLayer()
		return
	}
	if !mw.wearAllows(data.ItemCode(layer.Code)) {
		mw.refuseLayer()
		return
	}
	member.Layers = wearLayer(member.Layers, layer, mw.invParty.table)
}

// refuseLayer posts the town's wear refusal on the map message line, so a
// layer the member cannot wear is not a silent no-op.
func (mw *mapWorld) refuseLayer() {
	if mw.view != nil {
		mw.view.PostMessage("he cannot wear that", ui.MessageWhite, announceLife)
	}
}

// shopToggleLayer is the town's wear gesture for a clothing layer: the pack cell
// dropped on the doll puts the layer on, or takes it off when it is worn. The
// bool reports whether the cell holds a layer at all. Every other route that
// would wear a layer in a slot (the shelf, the table) is refused in
// shopWearInstance.
func (t *townScreen) shopToggleLayer(i int) (ui.TownAction, bool) {
	stacks := t.shopPackStacks()
	k := t.packBase + i - 1
	if k < 0 || k >= len(stacks) {
		return ui.TownAction{}, false
	}
	layer, ok := mapload.LayerItem(t.in.Table, stacks[k].Code)
	if !ok {
		return ui.TownAction{}, false
	}
	member := t.shopPartyMember(t.shopMemberIndex())
	if member == nil {
		return ui.TownAction{}, true
	}
	if j := slices.Index(member.Layers, layer.Code); j >= 0 {
		member.Layers = slices.Delete(slices.Clone(member.Layers), j, j+1)
		t.composeShopFaces()
		return ui.TownInfo("removed"), true
	}
	if mapload.HasSourceActor(*member) {
		return ui.TownAction{}, true
	}
	if !t.shopUsable(data.ItemCode(layer.Code)) {
		return ui.TownAction{}, true
	}
	member.Layers = wearLayer(member.Layers, layer, t.in.Table)
	t.composeShopFaces()
	return ui.TownInfo("worn"), true
}

// wearLayer is the layers with one more worn. A layer already worn at the same
// anchor and depth is replaced.
func wearLayer(worn []uint16, layer mapload.ModItem, t *mapload.Table) []uint16 {
	next := worn[:0:0]
	for _, code := range worn {
		if other, ok := mapload.LayerItem(t, code); ok && other.Anchor == layer.Anchor && other.Layer == layer.Layer {
			continue
		}
		next = append(next, code)
	}
	return append(next, layer.Code)
}
