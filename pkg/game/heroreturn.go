package game

import (
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const returnLineLife = 30 * time.Second

func (mw *mapWorld) waitingHero() (sim.EntityID, bool) {
	if mw == nil || mw.mission == nil || mw.mission.fall != fallReturns || mw.world == nil {
		return 0, false
	}
	for _, id := range mw.mission.guarded {
		if e, ok := mw.entity(id); ok && e.Owner == sim.SelfSlot && !e.Alive() && e.CorpseRemoved() {
			return id, true
		}
	}
	return 0, false
}

func (mw *mapWorld) returnHero() bool {
	id, ok := mw.waitingHero()
	if !ok {
		return false
	}
	m := mw.mission
	at := m.state.Start.Drop
	if m.hasReturnCell {
		at = m.returnCell
	}
	mw.pending = append(mw.pending, sim.ReturnHero(id, sim.CellPoint{X: at.X, Y: at.Y}))
	m.returning = append(m.returning, id)
	return true
}

func (mw *mapWorld) noteReturnCell(id sim.EntityID) {
	if m := mw.mission; m != nil && m.fall == fallReturns {
		if e, ok := mw.entity(id); ok {
			m.returnCell, m.hasReturnCell = mapload.Cell{X: e.X, Y: e.Y}, true
		}
	}
}

func (mw *mapWorld) settleHeroReturn() {
	m := mw.mission
	if m == nil || m.fall != fallReturns {
		return
	}
	for _, id := range m.returning {
		if e, ok := mw.entity(id); ok && e.Alive() {
			mw.recomputeDerived(id)
			mw.world.FillReturnedPools(id)
		}
	}
	m.returning = nil
	if _, waiting := mw.waitingHero(); !waiting {
		m.returnOffered = false
		return
	}
	if m.returnOffered || mw.view == nil {
		return
	}
	m.returnOffered = true
	if line := mw.view.Words().Engine.Text("session.return"); line != "" {
		mw.view.PostMessage(line, ui.MessageWhite, returnLineLife)
	}
}
