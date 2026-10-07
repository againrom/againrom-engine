package ui

// Item casts deliberately bypass the learned spellbook. Arming is display-only;
// the simulation reserves one item only after a real map release.
type itemCastSelection struct {
	owner uint32
	index int
	spell uint32
	key   string
}
type MapItemCast func(owner uint32, index int, key string, victim uint32, x, y int, atCell bool)

func (v *Viewer) SetItemCastSink(sink MapItemCast) { v.itemCastSink = sink }

func (v *Viewer) ArmItemCast(owner uint32, index int, spell uint32, key string) bool {
	present := presentSelected(v.sel, v.entities)
	if v.itemCastSink == nil || len(present) != 1 || present[0].ID != owner ||
		v.localOwner != 0 && present[0].Owner != v.localOwner || index < 0 || spell == 0 {
		return false
	}
	v.armed, v.selectedSpell, v.aimed = false, 0, commandNone
	v.spellArmed = false
	v.itemCast = &itemCastSelection{owner: owner, index: index, spell: spell, key: key}
	return true
}

func (v *Viewer) releaseItemCast(x, y int) {
	c := v.itemCast
	v.itemCast = nil
	if c == nil || v.itemCastSink == nil {
		return
	}
	present := presentSelected(v.sel, v.entities)
	if len(present) != 1 || present[0].ID != c.owner {
		return
	}
	col, row, inside := v.groundCellAt(float64(x), float64(y))
	if !inside || !v.fogGateSack(col, row) {
		return
	}
	_, hit, hasHit := v.hoverMask(x, y)
	v.itemCastSink(c.owner, c.index, c.key, hit, col, row, !hasHit)
}
