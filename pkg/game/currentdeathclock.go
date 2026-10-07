package game

import (
	"fmt"

	"againrom/pkg/sim"
)

const maxCurrentDeathAge = 1<<31 - 1

type currentDeathAge struct {
	Entity sim.EntityID
	Age    int
}

func (mw *mapWorld) captureCurrentDeathAges() []currentDeathAge {
	var rows []currentDeathAge
	for _, e := range mw.world.Entities() {
		if e.Alive() {
			continue
		}
		stamp, ok := mw.died[e.ID]
		if !ok {
			continue
		}
		age := mw.scene - stamp
		if age > maxCurrentDeathAge {
			age = maxCurrentDeathAge
		}
		rows = append(rows, currentDeathAge{Entity: e.ID, Age: age})
	}
	return rows
}

func validateCurrentDeathAges(rows []currentDeathAge, bound map[sim.EntityID]bool) error {
	if len(rows) > 131072 {
		return fmt.Errorf("current death-age population exceeds bound")
	}
	seen := make(map[sim.EntityID]bool, len(rows))
	for _, row := range rows {
		if seen[row.Entity] || row.Age < 0 || row.Age > maxCurrentDeathAge || !bound[row.Entity] {
			return fmt.Errorf("invalid current death age for actor %d", row.Entity)
		}
		seen[row.Entity] = true
	}
	return nil
}

func (mw *mapWorld) restoreCurrentDeathAges(rows []currentDeathAge, ids map[sim.EntityID]sim.EntityID) error {
	for _, row := range rows {
		id, ok := ids[row.Entity]
		if !ok {
			return fmt.Errorf("current death-age actor lacks a binding")
		}
		e, ok := mw.world.Entity(id)
		if !ok || e.Alive() {
			return fmt.Errorf("current death-age actor %d is not fallen", id)
		}
	}
	for _, row := range rows {
		mw.died[ids[row.Entity]] = mw.scene - row.Age
	}
	return nil
}
