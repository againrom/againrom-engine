package game

import "againrom/pkg/sim"

type projectileCounter struct {
	retained uint16
	world    *sim.World
}

func (c *projectileCounter) value() uint16 {
	if c == nil {
		return 0
	}
	if c.world != nil {
		return c.world.SavedProjectiles().FreeIndex
	}
	return c.retained
}

func (c *projectileCounter) track(w *sim.World) {
	if c != nil && w != nil && !w.SecondGame() {
		c.world = w
	}
}

func (c *projectileCounter) detach() {
	if c != nil {
		c.retained = c.value()
		c.world = nil
	}
}

func seedProjectileCounter(w *sim.World, counter uint16) {
	if w == nil || w.SecondGame() {
		return
	}
	store := w.SavedProjectiles()
	store.FreeIndex = counter
	w.SetSavedProjectiles(store)
}

func (s *CampaignSession) detachProjectileCounter() {
	if s.live != nil {
		s.live.projectileCounter.detach()
	}
}
