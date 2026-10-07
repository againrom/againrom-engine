package sim

import "fmt"

// HeadlessKill kills one ordinary actor through the mission script's own
// health write, setUnitProperty, with the first non-targetable health. That
// routine runs the ordinary death transition; the fall dwell, the terminal
// loot, the decay stages and the removal of a body that leaves none are left
// to the caller's ordinary Steps. Script checks run only inside Step.
func (w *World) HeadlessKill(id EntityID) error {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return fmt.Errorf("sim: HeadlessKill names entity %d, which this world does not hold", id)
	}
	if w.entities[i].MaxHP <= 0 {
		return fmt.Errorf("sim: HeadlessKill entity %d has no ordinary health system", id)
	}
	w.headlessKillIndex(i)
	return nil
}

// headlessKillIndex reports whether actor i crossed the targetable boundary.
// A body already past it is left to the ordinary decay.
func (w *World) headlessKillIndex(i int) bool {
	e := w.entities[i]
	if e.MaxHP <= 0 || !e.OrdinaryTargetable() {
		return false
	}
	w.setUnitProperty(i, propertyHealth, decayBonesHP)
	return true
}

// HeadlessKillPlayer applies HeadlessKill to every ordinary actor owned by one
// script player and reports how many crossed the targetable boundary. A player
// the world does not name is refused as a likely scenario typo.
func (w *World) HeadlessKillPlayer(player uint32) (int, error) {
	found := false
	killed := 0
	for i := range w.entities {
		if w.entities[i].Owner != player {
			continue
		}
		found = true
		if w.headlessKillIndex(i) {
			killed++
		}
	}
	if !found {
		return 0, fmt.Errorf("sim: HeadlessKillPlayer names player %d, which owns no entity in this world", player)
	}
	return killed, nil
}

// HeadlessPlace is the script's own placement: instant 16's removal, then
// instant 18's bounded search near (x, y), which may pick a neighbour. A failed
// search returns the actor through instant 17. Nothing else is written.
func (w *World) HeadlessPlace(id EntityID, x, y int32) error {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return fmt.Errorf("sim: HeadlessPlace names entity %d, which this world does not hold", id)
	}
	if _, ok := w.cellIndex(x, y); !ok {
		return fmt.Errorf("sim: HeadlessPlace anchor (%d,%d) is outside %dx%d", x, y,
			w.bounds.Width, w.bounds.Height)
	}
	if w.entities[i].OffMap {
		return fmt.Errorf("sim: HeadlessPlace names off-map entity %d", id)
	}
	w.takeOffMap(i)
	if w.placeNear(i, x, y) {
		return nil
	}
	w.returnToMap(i)
	return fmt.Errorf("sim: HeadlessPlace found no cell near (%d,%d) that fits entity %d", x, y, id)
}

// HeadlessHeal restores one ordinary restorative target to its authored
// maximum health through the mission script's own health write,
// setUnitProperty, which runs the shared fallen-to-living transition.
func (w *World) HeadlessHeal(id EntityID) error {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return fmt.Errorf("sim: HeadlessHeal names entity %d, which this world does not hold", id)
	}
	e := w.entities[i]
	if !e.restorativeTargetable() {
		return fmt.Errorf("sim: HeadlessHeal entity %d at %d/%d is not a restorative target", id, e.HP, e.MaxHP)
	}
	w.setUnitProperty(i, propertyHealth, e.MaxHP)
	return nil
}

// HeadlessDamage lowers one ordinary target's health by amount through the
// mission script's own health write, setUnitProperty, to the current health
// minus amount. A result that needs a killer comes from an ordinary attack.
func (w *World) HeadlessDamage(id EntityID, amount int32) error {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return fmt.Errorf("sim: HeadlessDamage names entity %d, which this world does not hold", id)
	}
	e := w.entities[i]
	if amount <= 0 || e.MaxHP <= 0 || !e.OrdinaryTargetable() {
		return fmt.Errorf("sim: HeadlessDamage of %d on entity %d at %d/%d is not an ordinary wound", amount, id, e.HP, e.MaxHP)
	}
	w.setUnitProperty(i, propertyHealth, e.HP-amount)
	return nil
}
