package ui

// MapPlayerRetreat queues the complete selection from one explicit Retreat press.
// MapRetreat remains Ctrl+W's separate automatic-withdrawal preference seam.
type MapPlayerRetreat func(entities []uint32)

func (v *Viewer) SetPlayerRetreatSink(fn MapPlayerRetreat) { v.playerRetreatSink = fn }

func (v *Viewer) issuePlayerRetreat() {
	if v.playerRetreatBlocked || v.dragActive || v.dragCandKind != dragNone ||
		!commandPanelActive(v.sel, v.entities, v.localOwner) {
		return
	}
	v.armed, v.attackHeld, v.selectedSpell = false, false, 0
	v.spellArmed = false
	v.itemCast = nil
	v.aimed = commandNone
	v.cmdOverlayHidden = false
	if v.playerRetreatSink != nil {
		ids := v.marked()
		v.playerRetreatSink(ids[:min(253, len(ids))])
		v.retreatSpoken = ids
	}
}
