package game

import (
	"againrom/pkg/sim"
	"fmt"
)

// Equipment exposes instance values but no count. Pack quantity remains
// authoritative whenever that node also has a Pack occurrence.
func (g *cityObjectTopology) wornQuantity(id sim.SavedObjectID) uint32 {
	for _, row := range g.Items {
		if row.ID == id && row.WornCount != 0 {
			return row.WornCount
		}
	}
	return 1
}

func (g *cityObjectTopology) trimWornCounts() {
	pack, worn := map[sim.SavedObjectID]bool{}, map[sim.SavedObjectID]bool{}
	for _, root := range g.Roots {
		for _, id := range root.Pack {
			pack[id] = true
		}
		for _, id := range root.Worn {
			worn[id] = true
		}
	}
	for i := range g.Items {
		row := &g.Items[i]
		if pack[row.ID] || !worn[row.ID] || row.WornCount == 1 {
			row.WornCount = 0
		}
	}
}

func (t *townScreen) cityShopNodeCount(id sim.SavedObjectID) (uint32, error) {
	p, err := newCityObjectProjection(t.sess.Town.cityObjects, t.sess.Carried, t.in.Table)
	if err != nil {
		return 0, err
	}
	if value, found := p.items[id]; found {
		return value.Count, nil
	}
	return 0, fmt.Errorf("city quantity names absent Item %d", id)
}
