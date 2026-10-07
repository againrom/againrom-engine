package game

import (
	"fmt"
	"image"
	"math"
	"slices"

	"againrom/pkg/sim"
)

// These clocks select attack and walking frames without changing actor state.
type currentAnimationClock struct {
	Motion *currentMotionClock `json:",omitempty"`
	Entity sim.EntityID
	Swing  *int             `json:",omitempty"`
	Phase  *sim.AttackPhase `json:",omitempty"`
}

type currentMotionClock struct {
	Distance, Tick int
	Previous       *currentMotionCell `json:",omitempty"`
}

type currentMotionCell struct{ X, Y int32 }

func captureCurrentMapMotion(mw *mapWorld) []currentAnimationClock {
	var rows []currentAnimationClock
	for _, e := range mw.world.EntityView() {
		clock, clocked := mw.walk[e.ID]
		cell, positioned := mw.prev[e.ID]
		if !clocked && !positioned {
			continue
		}
		motion := &currentMotionClock{Distance: clock.dist, Tick: clock.tick}
		if positioned {
			motion.Previous = &currentMotionCell{int32(cell.X), int32(cell.Y)}
		}
		rows = append(rows, currentAnimationClock{Entity: e.ID, Motion: motion})
	}
	return rows
}

func mergeCurrentMapMotion(rows, motion []currentAnimationClock) []currentAnimationClock {
	for _, next := range motion {
		index := slices.IndexFunc(rows, func(row currentAnimationClock) bool { return row.Entity == next.Entity })
		if index < 0 {
			rows = append(rows, next)
		} else {
			rows[index].Motion = next.Motion
		}
	}
	slices.SortFunc(rows, func(a, b currentAnimationClock) int {
		if a.Entity < b.Entity {
			return -1
		}
		if a.Entity > b.Entity {
			return 1
		}
		return 0
	})
	return rows
}

func captureCurrentAnimation(r SnapshotResidue) []currentAnimationClock {
	ids := make(map[uint32]bool, len(r.Swing)+len(r.Phase))
	for id := range r.Swing {
		ids[id] = true
	}
	for id := range r.Phase {
		ids[id] = true
	}
	for _, run := range r.CastRuns {
		delete(ids, uint32(run.Entity))
	}
	out := make([]currentAnimationClock, 0, len(ids))
	for id := range ids {
		row := currentAnimationClock{Entity: sim.EntityID(id)}
		if value, ok := r.Swing[id]; ok {
			row.Swing = &value
		}
		if value, ok := r.Phase[id]; ok {
			phase := sim.AttackPhase(value)
			row.Phase = &phase
		}
		out = append(out, row)
	}
	slices.SortFunc(out, func(a, b currentAnimationClock) int {
		if a.Entity < b.Entity {
			return -1
		}
		if a.Entity > b.Entity {
			return 1
		}
		return 0
	})
	return out
}

func validateCurrentAnimation(rows []currentAnimationClock, runs []SnapshotCastRun) error {
	if len(rows) > 131072 {
		return fmt.Errorf("current animation population exceeds bound")
	}
	seen := make(map[sim.EntityID]bool, len(rows))
	casting := make(map[sim.EntityID]bool, len(runs))
	for _, run := range runs {
		casting[run.Entity] = true
	}
	for _, row := range rows {
		if seen[row.Entity] || casting[row.Entity] && (row.Swing != nil || row.Phase != nil) || row.Swing == nil && row.Phase == nil && row.Motion == nil || row.Swing != nil && *row.Swing < -1 || row.Phase != nil && *row.Phase > sim.AttackBoundaryTwo {
			return fmt.Errorf("invalid current animation clock")
		}
		if row.Motion != nil && (row.Motion.Distance < 0 || row.Motion.Distance > math.MaxInt32 || row.Motion.Tick < 0 || row.Motion.Tick > math.MaxInt32) {
			return fmt.Errorf("invalid current movement presentation clock")
		}
		seen[row.Entity] = true
	}
	return nil
}

func (mw *mapWorld) restoreCurrentAnimation(rows []currentAnimationClock, ids map[sim.EntityID]sim.EntityID) error {
	for _, row := range rows {
		id, ok := ids[row.Entity]
		if !ok {
			return fmt.Errorf("current animation actor lacks a binding")
		}
		if row.Motion != nil {
			mw.walk[id] = walkClock{dist: row.Motion.Distance, tick: row.Motion.Tick}
			delete(mw.prev, id)
			if cell := row.Motion.Previous; cell != nil {
				mw.prev[id] = image.Pt(int(cell.X), int(cell.Y))
			}
		}
		if row.Swing != nil {
			mw.swing[id] = *row.Swing
		}
		if row.Phase != nil {
			mw.phase[id] = *row.Phase
		}
	}
	return nil
}
