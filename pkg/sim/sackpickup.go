package sim

import "fmt"

// A sack pickup is one actor taking one ground sack. This file is its one
// builder: the order's admission, the arrival test, the take, the quest
// documents moved to the starting hero and the completion. Drivers call
// OrderSackPickup, PickUpSackUnderfoot, SettleSackPickup and CheatPickupAll;
// TakeSack, MoveCarried, CompleteSackPickup and cancelSackPickup are reached
// in production only from here.

// SackDocuments is where a pickup sends the quest documents a sack holds: Code
// is the quest document's item code and Holder the starting hero. With Held
// false there is no starting hero, and a sack holding a document is refused.
type SackDocuments struct {
	Code   uint16
	Holder EntityID
	Held   bool
}

// SackPickup is what one pickup did, for the lines the player reads: the
// taker, the sack, each holder's pack before and after the take (the taker,
// then the documents' holder when he is another actor), and the local
// player's purse before and after, the pairs a script grant reports too.
// Order is the command a driver queues for SackPickupOrder, and for any
// outcome when Next is set: the collect order's next search.
type SackPickup struct {
	Taker   EntityID
	Sack    Sack
	Holders []ItemGrant
	Purse   PurseCredit
	Order   Command
	Next    bool
}

// pickupCollects is the Spell operand of CollectPickUp.
const pickupCollects uint16 = 1

// SackPickupOutcome is how one pickup call ended.
type SackPickupOutcome uint8

const (
	// SackPickupNone: no living taker on the map, or no sack to take.
	SackPickupNone SackPickupOutcome = iota
	// SackPickupWaiting: the taker has not arrived on the cell.
	SackPickupWaiting
	// SackPickupOrder: the taker is mid-strike; the driver queues Order, the
	// pickup order for the taker's own cell, instead of a take.
	SackPickupOrder
	// SackPickupRefused: the take was refused.
	SackPickupRefused
	// SackPickupTaken: the sack was taken; the completion runs on the next Step.
	SackPickupTaken
)

// OrderSackPickup admits a pickup order: a sack stands at the cell and the
// taker is a living actor on the map (ITEM-PICK-016). It answers the order,
// which the driver queues beside any physical progress (AI-356).
func (w *World) OrderSackPickup(taker EntityID, at CellPoint) (order Command, ok bool) {
	if _, sack := w.groundSackAt(at.X, at.Y); !sack {
		return order, false
	}
	if i := indexOfEntity(w.entities, taker); i < 0 || !w.entities[i].Alive() || w.entities[i].OffMap {
		return order, false
	}
	return PickUp(taker, at), true
}

// OrderSackCollect plans one hero's collect key through the pickup admission:
// the collect pickup of the found sack; refused when the search finds none.
// SettleSackPickup calls it again after each collect take.
// DIV-4882, DIV-4883, DIV-4884.
func (w *World) OrderSackCollect(taker EntityID) (order Command, ok bool) {
	e, ok := w.collectTaker(taker)
	if !ok {
		return order, false
	}
	if _, ok = w.OrderSackPickup(taker, CellPoint{X: e.X, Y: e.Y}); ok {
		return CollectPickUp(taker, CellPoint{X: e.X, Y: e.Y}), true
	}
	probe := func(x, y int64) bool {
		if x < 0 || y < 0 || x >= int64(w.bounds.Width) || y >= int64(w.bounds.Height) {
			return false
		}
		at := CellPoint{X: int32(x), Y: int32(y)}
		_, accepted := w.OrderSackPickup(taker, at)
		if accepted {
			order = CollectPickUp(taker, at)
		}
		return accepted
	}
	if walkCellRings(cellOf(&e), int64(e.SightWord(e.Humanoid)>>8), cellRingWalk{firstStep: true}, probe) {
		return order, true
	}
	return order, false
}

// collectTaker is the collect key's member test: a living hero of the local
// player on the map, in a game whose pickup key collects.
func (w *World) collectTaker(taker EntityID) (Entity, bool) {
	i := indexOfEntity(w.entities, taker)
	if !w.game.orders.collectSacks || i < 0 {
		return Entity{}, false
	}
	e := w.entities[i]
	return e, e.Alive() && !e.OffMap && e.Owner == SelfSlot && InPersistBand(e.TypeID)
}

// PickUpSackUnderfoot is the pickup key and the headless take: the taker takes
// the sack on its own cell once it has arrived there. A taker mid-strike gets
// the pickup order for its cell instead, so the take waits for the strike.
func (w *World) PickUpSackUnderfoot(taker EntityID, docs SackDocuments) (SackPickup, SackPickupOutcome) {
	i := indexOfEntity(w.entities, taker)
	if i < 0 {
		return SackPickup{}, SackPickupNone
	}
	e := &w.entities[i]
	if e.HasAttackTarget && e.AttackPhase != AttackReady && !(e.PendingOrder.Kind == PendingPickup && e.PendingOrder.RowAdmitted) {
		order, ok := w.OrderSackPickup(taker, CellPoint{X: e.X, Y: e.Y})
		if !ok {
			return SackPickup{}, SackPickupNone
		}
		return SackPickup{Taker: taker, Order: order}, SackPickupOrder
	}
	if !e.Alive() || e.OffMap {
		return SackPickup{}, SackPickupNone
	}
	if !w.sackPickupArrived(i) {
		return SackPickup{}, SackPickupWaiting
	}
	sack, ok := w.groundSackAt(e.X, e.Y)
	if !ok {
		return SackPickup{}, SackPickupNone
	}
	return w.pickUpSack(taker, sack, docs)
}

// SackPickupSettled is one pending pickup the settle ended, and how.
type SackPickupSettled struct {
	SackPickup
	Outcome SackPickupOutcome
}

// SettleSackPickup is the after-Step settle of the pending pickup orders, run
// by the driver once after each Step over every actor holding one, in entity
// order. A dead or off-map taker's order is cancelled. A sack gone from the
// order's cell completes the pickup without a transfer (AI-645) once the
// taker stands at rest on the cell, or off it with his walk ended, as when
// another actor holds the cell, so he never waits for ever. An arrived taker
// takes the sack; a refused take ends the order, so it never loops. After a
// take, or a sack gone, a collect pickup answers its next search in Order
// with Next set: the next pickup, or Stand Ground.
func (w *World) SettleSackPickup(docs SackDocuments) []SackPickupSettled {
	var settled []SackPickupSettled
	for i := range w.entities {
		e := w.entities[i]
		if e.PendingOrder.Kind != PendingPickup {
			continue
		}
		if !e.Alive() || e.OffMap {
			w.cancelSackPickup(e.ID)
			continue
		}
		collect := e.PendingOrder.Collect
		sack, ok := w.groundSackAt(e.PendingOrder.X, e.PendingOrder.Y)
		there := e.X == e.PendingOrder.X && e.Y == e.PendingOrder.Y
		if !w.sackPickupArrived(i) || !there && (ok || e.HasTarget) {
			continue
		}
		if !ok {
			w.CompleteSackPickup(e.ID)
			settled = append(settled, SackPickupSettled{w.collectNext(SackPickup{Taker: e.ID}, collect), SackPickupNone})
			continue
		}
		taken, outcome := w.pickUpSack(e.ID, sack, docs)
		w.cancelSackPickup(e.ID)
		if outcome == SackPickupTaken {
			taken = w.collectNext(taken, collect)
		}
		settled = append(settled, SackPickupSettled{taken, outcome})
	}
	return settled
}

// collectNext answers a collect pickup's next search from the taker's cell;
// a failed search ends the collecting at Stand Ground (DIV-4884).
func (w *World) collectNext(taken SackPickup, collect bool) SackPickup {
	if !collect {
		return taken
	}
	if taken.Order, taken.Next = w.OrderSackCollect(taken.Taker); !taken.Next {
		if _, ok := w.collectTaker(taken.Taker); ok {
			taken.Order, taken.Next = GroupStance(taken.Taker, OrderStandGround, 0), true
		}
	}
	return taken
}

// CheatPickupAll is #pickup all: every ground sack, first to last, is the
// hero's pickup (MENU-108, R2-ENGINE-301), and told hears each take. Without a
// starting hero the hero holds the documents. A refused take stops it.
func (w *World) CheatPickupAll(hero EntityID, docs SackDocuments, told func(SackPickup)) error {
	if !docs.Held {
		docs.Holder, docs.Held = hero, true
	}
	return w.cheatPickupEach(hero, func(sack Sack) error {
		taken, outcome := w.pickUpSack(hero, sack, docs)
		if outcome != SackPickupTaken {
			return fmt.Errorf("sim: cheat pickup refused the sack at (%d,%d)", sack.X, sack.Y)
		}
		if told != nil {
			told(taken)
		}
		return nil
	})
}

// SackPickupArrived is the arrival test: the actor has finished its crossing,
// its cycle and its order progress and stands at its cell's centre. Only the
// pickup calls it in production.
func (w *World) SackPickupArrived(id EntityID) bool {
	i := indexOfEntity(w.entities, id)
	return i >= 0 && w.sackPickupArrived(i)
}

func (w *World) sackPickupArrived(i int) bool {
	e := &w.entities[i]
	if e.Transit != 0 || e.AttackPhase != AttackReady && !(e.PendingOrder.Kind == PendingPickup && e.PendingOrder.RowAdmitted) || w.ActorOrderProgress(e.ID) != 0 {
		return false
	}
	x, y, present := w.ActorFinePosition(e.ID)
	return !present || x == 128 && y == 128
}

// groundSackAt is the first ground sack at (x, y) as Sacks answers it.
func (w *World) groundSackAt(x, y int32) (Sack, bool) {
	for _, s := range w.sacks {
		if s.X == x && s.Y == y {
			return publicSack(s), true
		}
	}
	return Sack{}, false
}

// pickUpSack is the take of one sack: the transfer, the quest documents moved
// to their holder, and the completion on the next Step. A sack with documents
// and no holder able to carry them is refused before anything moves.
func (w *World) pickUpSack(taker EntityID, sack Sack, docs SackDocuments) (SackPickup, SackPickupOutcome) {
	taken := SackPickup{Taker: taker, Sack: sack, Purse: PurseCredit{Player: SelfSlot, Before: w.Purse(SelfSlot)}}
	holders := []EntityID{taker}
	if docs.Held && docs.Holder != taker {
		holders = append(holders, docs.Holder)
	}
	for _, holder := range holders {
		before, _ := w.CarriedStacks(holder)
		taken.Holders = append(taken.Holders, ItemGrant{Holder: holder, Before: before})
	}
	var documents uint32
	for _, code := range sack.Items {
		if code == docs.Code {
			documents++
		}
	}
	refused := SackPickup{Taker: taker, Sack: sack}
	if documents > 0 {
		if !docs.Held {
			return refused, SackPickupRefused
		}
		if _, ok := w.Carried(docs.Holder); !ok {
			return refused, SackPickupRefused
		}
	}
	if err := w.TakeSack(taker, sack.X, sack.Y); err != nil {
		return refused, SackPickupRefused
	}
	if documents > 0 && taker != docs.Holder {
		if err := w.MoveCarried(taker, docs.Holder, docs.Code, documents); err != nil {
			return refused, SackPickupRefused
		}
	}
	w.CompleteSackPickup(taker)
	for k := range taken.Holders {
		taken.Holders[k].After, _ = w.CarriedStacks(taken.Holders[k].Holder)
	}
	taken.Purse.After = w.Purse(SelfSlot)
	return taken, SackPickupTaken
}
