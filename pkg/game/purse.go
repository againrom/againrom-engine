package game

import (
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The purse the inventory shows is a preview: the simulation's purse less the
// gold requests still queued for it. The simulation debits only when a queued
// request is applied, so a paused game shows the balance a request has already
// reserved without spending it twice, and a saved game keeps the unspent purse.

// queuedGold is the total of the live gold requests queued for the primary
// purse, saturating at the 32-bit maximum.
func (mw *mapWorld) queuedGold() uint32 {
	var total uint64
	for i, c := range mw.pending {
		if c.Kind != sim.KindPlayerDropGold || c.Player != sim.SelfSlot {
			continue
		}
		if i < len(mw.pendingIgnored) && mw.pendingIgnored[i] {
			continue
		}
		total += uint64(c.Group)
	}
	return uint32(min(total, uint64(^uint32(0))))
}

// previewPurse is the primary purse the inventory displays.
func (mw *mapWorld) previewPurse() uint32 {
	purse, queued := mw.world.Purse(sim.SelfSlot), mw.queuedGold()
	if queued >= purse {
		return 0
	}
	return purse - queued
}

// dropGoldFromPurse drains the viewer's one-shot purse request into the
// pending queue, before the frame's advance.
func (mw *mapWorld) dropGoldFromPurse() {
	req, ok := mw.view.TakeGoldDrop()
	if !ok {
		return
	}
	mw.enqueueGoldDrop(req)
}

// enqueueGoldDrop turns a purse request into a player-addressed command. The
// amount is clamped to the previewed balance, so queued requests together
// never reserve more than the purse; the simulation checks affordability again
// when it applies the command. A request made at the subject's own cell reads
// that cell now.
func (mw *mapWorld) enqueueGoldDrop(req ui.GoldDrop) {
	if !mw.invSubjectSet || !mw.invParty.primary {
		return
	}
	amount := min(req.Amount, mw.previewPurse())
	if amount == 0 {
		return
	}
	at := sim.CellPoint{X: req.X, Y: req.Y}
	if req.AtSubject {
		e, ok := mw.entity(sim.EntityID(mw.invSubject.ID))
		if !ok {
			return
		}
		at = sim.CellPoint{X: e.X, Y: e.Y}
	}
	mw.pending = append(mw.pending, sim.DropGold(sim.SelfSlot, amount, at))
}
