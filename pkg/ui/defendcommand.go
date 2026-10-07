package ui

// MapDefend queues one selected actor's membership in a player Defend order.
// The game tier joins the selected members into one fresh command group.
type MapDefend func(entity, subject uint32)

func (v *Viewer) SetDefendSink(fn MapDefend) { v.defendSink = fn }

// D and panel cell 3 reassert mode 4, including repeated presses. They share
// the owned, non-structure selection gate (AI-PANEL-123, AI-KEY-125).
func (v *Viewer) armDefend() {
	if !commandPanelActive(v.sel, v.entities, v.localOwner) {
		return
	}
	v.cmdOverlayHidden = false
	v.aimed = commandDefend
	v.armed, v.attackHeld, v.selectedSpell = false, false, 0
	v.spellArmed = false
	v.itemCast = nil
}
