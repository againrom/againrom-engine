package game

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The current population is explicitly bound in the typed continuation. A
// lifecycle stage alone cannot decide whether a body still owns an Entity.
func currentActorGraph(file *sav.File, state *SnapshotSAVDocument, origins []sav.DocumentObjectOrigin) (sav.SavedActorGraph, error) {
	if state == nil || state.Document == nil {
		return file.ActorGraph()
	}
	a, err := readCurrentActions(state.Document)
	if err != nil {
		return sav.SavedActorGraph{}, err
	}
	if a == nil || a.Values == nil {
		return file.ActorGraph()
	}
	current := make([]uint16, 0, len(a.Values))
	offMap := make(map[uint16]bool)
	presence := make(map[sim.EntityID]bool)
	terminal := make(map[uint16]bool)
	for _, rows := range [][]sim.ActorContinuation{a.Actions.Actors, a.Held} {
		for _, row := range rows {
			if _, exists := presence[row.Entity]; exists {
				return sav.SavedActorGraph{}, fmt.Errorf("repeated current actor presence binding")
			}
			presence[row.Entity] = row.OffMap
		}
	}
	for id, value := range a.Values {
		var found uint16
		missing := false
		for _, b := range a.Bindings {
			if b.ID != id || b.Structure {
				continue
			}
			if found != 0 || missing {
				return sav.SavedActorGraph{}, fmt.Errorf("invalid current actor population binding")
			}
			if b.Missing && b.Object == 0 && pureCurrentTerminalActorValue(value, id) {
				missing = true
				continue
			}
			if b.Missing || b.Object == 0 {
				return sav.SavedActorGraph{}, fmt.Errorf("invalid current actor population binding")
			}
			found = b.Object
		}
		if _, present := presence[id]; missing && !present {
			continue // a terminal actor with no SAV record constructs nothing
		}
		if found == 0 {
			return sav.SavedActorGraph{}, fmt.Errorf("missing current actor population binding")
		}
		outside, present := presence[id]
		if !present {
			retained := false
			for _, object := range state.Document.DeadActors {
				retained = retained || object == found
			}
			if value.CurrentTerminal != nil {
				if !retained || !pureCurrentTerminalActorValue(value, id) {
					return sav.SavedActorGraph{}, fmt.Errorf("missing current actor presence binding")
				}
				// A retired native actor, not current; DeadActors readable.
				// Its Stage is the terminal decay stage, so a non-ground
				// mover's Stage 1 is not a dying body to construct.
				for _, origin := range origins {
					if origin.ObjectIndex == found {
						terminal[origin.ArchiveIndex] = true
					}
				}
				continue
			}
			if !retained || (value.RuntimeID == nil && value.DeadSourceHealth == nil) {
				return sav.SavedActorGraph{}, fmt.Errorf("missing current actor presence binding")
			}
			// Retained source values do not construct an Entity. The
			// continuation transaction checks its exact dead-manager ID
			// and rejects any additional actor operands.
			continue
		}
		var archive uint16
		for _, origin := range origins {
			if origin.ObjectIndex == found {
				archive = origin.ArchiveIndex
				break
			}
		}
		if archive == 0 {
			return sav.SavedActorGraph{}, fmt.Errorf("current actor has no archive origin")
		}
		offMap[archive] = outside
		current = append(current, archive)
	}
	graph, err := file.CurrentActorGraph(current)
	if err != nil {
		return graph, err
	}
	for i := range graph.Actors {
		graph.Actors[i].CurrentOffMap = offMap[graph.Actors[i].ArchiveIndex]
		graph.Actors[i].TerminalActor = terminal[graph.Actors[i].ArchiveIndex]
	}
	return graph, nil
}

func currentActorArchives(graph sav.SavedActorGraph) []uint16 {
	var current []uint16
	for _, actor := range graph.Actors {
		if actor.CurrentEntity {
			current = append(current, actor.ArchiveIndex)
		}
	}
	return current
}

// A stationary native actor uses ordinary Cell as its position authority.
// Frozen or imported movers and active crossings retain their separate phase.
func matchCurrentStationaryPositions(ms *Mission, a *currentActionData) error {
	entities := make(map[sim.EntityID]sim.Entity)
	for _, e := range ms.World.Entities() {
		entities[e.ID] = e
	}
	objects := make(map[sim.EntityID]uint16)
	for _, b := range ms.savedDocument.Actors {
		if !b.Retired {
			objects[b.EntityID] = b.ObjectIndex
		}
	}
	for i := range a.Actions.Actors {
		row := &a.Actions.Actors[i]
		e, present := entities[row.Entity]
		if !present || !e.Alive() || row.OffMap || row.ImportedMotion || row.MotionIssue != "" || row.Transit != 0 || row.TurnRemaining != 0 ||
			row.X < 0 || row.X > 255 || row.Y < 0 || row.Y > 255 {
			continue
		}
		object := objects[row.Entity]
		if object == 0 || int(object) > len(ms.savedDocument.Document.Objects) {
			return fmt.Errorf("current stationary actor lacks an ordinary position binding")
		}
		position, err := savedActorRaw(&ms.savedDocument.Document.Objects[object-1], "Block12", 12)
		if err != nil {
			return err
		}
		cell := binary.LittleEndian.Uint16(position)
		row.X, row.Y = int32(cell&255), int32(cell>>8)
	}
	return nil
}
