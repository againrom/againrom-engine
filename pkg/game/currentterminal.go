package game

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"

	"againrom/pkg/sim"
)

func pureCurrentTerminalActorValue(value sim.ActorValues, id sim.EntityID) bool {
	if value.CurrentTerminal == nil || value.CurrentTerminal.ID != id {
		return false
	}
	value.CurrentTerminal = nil
	return reflect.DeepEqual(value, sim.ActorValues{})
}

func currentTerminalObjectIdentity(state *SnapshotSAVDocument, object uint16) (uint32, error) {
	if state == nil || state.Document == nil || object == 0 || int(object) > len(state.Document.Objects) {
		return 0, fmt.Errorf("current terminal actor has an invalid SAV object")
	}
	record := &state.Document.Objects[object-1]
	switch record.Class {
	case "Unit", "Human", "Humanoid":
	default:
		return 0, fmt.Errorf("current terminal object %d is not an actor", object)
	}
	key, err := savedStructureValue(record, "Identity")
	if err != nil || key == 0 {
		return 0, fmt.Errorf("current terminal actor object %d has no SAV identity", object)
	}
	return key, nil
}

func currentTerminalActionObject(state *SnapshotSAVDocument, a *currentActionData, id sim.EntityID) (uint16, uint32, error) {
	if a == nil {
		return 0, 0, fmt.Errorf("current terminal actor %d has no current action state", id)
	}
	value, ok := a.Values[id]
	if !ok || !pureCurrentTerminalActorValue(value, id) {
		return 0, 0, fmt.Errorf("current terminal actor %d has no pure current value", id)
	}
	var object uint16
	bindings := 0
	for _, binding := range a.Bindings {
		if binding.Structure || binding.ID != id {
			continue
		}
		bindings++
		if bindings > 1 || binding.Missing != (binding.Object == 0) {
			return 0, 0, fmt.Errorf("current terminal actor %d has an invalid action binding", id)
		}
		object = binding.Object
	}
	if bindings == 0 {
		return 0, 0, fmt.Errorf("current terminal actor %d has no action object", id)
	}
	if object == 0 {
		// Departed before any SAV object bound it: the document has no record.
		return 0, 0, nil
	}
	if !slices.Contains(state.Document.DeadActors, object) {
		return 0, 0, fmt.Errorf("current terminal actor %d is not bound to a DeadActors root", id)
	}
	key, err := currentTerminalObjectIdentity(state, object)
	return object, key, err
}

func captureCurrentTerminalValues(w *sim.World, a *currentActionData) error {
	for _, terminal := range w.CurrentTerminalActors() {
		if _, exists := a.Values[terminal.ID]; exists {
			return fmt.Errorf("current terminal actor %d collides with a live value", terminal.ID)
		}
		row := terminal
		a.Values[terminal.ID] = sim.ActorValues{CurrentTerminal: &row}
	}
	return nil
}

// currentTerminalActorBindings resolves terminal IDs to SAV objects.
func currentTerminalActorBindings(state *SnapshotSAVDocument, world *sim.World) (map[sim.EntityID]uint16, error) {
	out := map[sim.EntityID]uint16{}
	if world == nil {
		return nil, fmt.Errorf("current terminal actor has no world")
	}
	if state == nil || state.Document == nil {
		if len(world.CurrentTerminalActors()) == 0 {
			return out, nil
		}
		return nil, fmt.Errorf("current terminal actor has no current document")
	}
	a, actionErr := readCurrentActions(state.Document)
	if actionErr != nil {
		return nil, actionErr
	}
	seenObjects, seenKeys := map[uint16]bool{}, map[uint32]bool{}
	for _, terminal := range world.CurrentTerminalActors() {
		var object uint16
		actorBinding := false
		for _, actor := range state.Actors {
			if actor.EntityID != terminal.ID {
				continue
			}
			if actorBinding {
				return nil, fmt.Errorf("current terminal actor %d has repeated SAV bindings", terminal.ID)
			}
			actorBinding = true
			object = actor.ObjectIndex
		}
		var identity uint32
		if !actorBinding {
			// No SAV actor ever bound it: it stays out of the document and its
			// action binding is Missing. A prior Missing binding stays so.
			if a == nil || a.Values[terminal.ID].CurrentTerminal == nil {
				continue
			}
			object, identity, actionErr = currentTerminalActionObject(state, a, terminal.ID)
			if actionErr != nil {
				return nil, actionErr
			}
			if object == 0 {
				continue
			}
		} else {
			if object == 0 || int(object) > len(state.Document.Objects) {
				return nil, fmt.Errorf("current terminal actor %d has an invalid SAV actor binding", terminal.ID)
			}
			var err error
			identity, err = currentTerminalObjectIdentity(state, object)
			if err != nil {
				return nil, err
			}
			if a != nil && a.Values[terminal.ID].CurrentTerminal != nil {
				boundObject, boundIdentity, err := currentTerminalActionObject(state, a, terminal.ID)
				if err != nil || boundObject != object || boundIdentity != identity {
					return nil, fmt.Errorf("current terminal actor %d conflicts with its current action binding", terminal.ID)
				}
			}
		}
		if _, exists := out[terminal.ID]; exists || seenObjects[object] || seenKeys[identity] {
			return nil, fmt.Errorf("repeated current terminal actor %d", terminal.ID)
		}
		seenObjects[object], seenKeys[identity] = true, true
		out[terminal.ID] = object
	}
	return out, nil
}

func currentTerminalActorObjects(state *SnapshotSAVDocument, world *sim.World) (map[uint16]sim.EntityID, error) {
	bindings, err := currentTerminalActorBindings(state, world)
	if err != nil {
		return nil, err
	}
	out := make(map[uint16]sim.EntityID, len(bindings))
	for id, object := range bindings {
		if old, exists := out[object]; exists && old != id {
			return nil, fmt.Errorf("current terminal SAV actor %d has aliased identities", object)
		}
		out[object] = id
	}
	return out, nil
}

// currentTerminalImportRows validates the document's terminal rows in saved
// ID order. It does not touch the World.
func currentTerminalImportRows(ms *Mission, a *currentActionData) ([]sim.CurrentTerminalActor, error) {
	if ms == nil || ms.World == nil || ms.savedDocument == nil || ms.savedDocument.Document == nil {
		return nil, fmt.Errorf("current terminal actor import has no current mission document")
	}
	var rows []sim.CurrentTerminalActor
	seenObjects, seenKeys := map[uint16]bool{}, map[uint32]bool{}
	for id, values := range a.Values {
		if values.CurrentTerminal == nil {
			continue
		}
		object, key, err := currentTerminalActionObject(ms.savedDocument, a, id)
		if err != nil || object != 0 && (seenObjects[object] || seenKeys[key]) {
			return nil, fmt.Errorf("current terminal actor %d has an invalid SAV identity", id)
		}
		rows = append(rows, *values.CurrentTerminal)
		if object != 0 {
			seenObjects[object], seenKeys[key] = true, true
		}
	}
	slices.SortFunc(rows, func(x, y sim.CurrentTerminalActor) int { return cmp.Compare(x.ID, y.ID) })
	return rows, nil
}

func currentTerminalActorKeys(state *SnapshotSAVDocument) (map[uint32]bool, error) {
	out := map[uint32]bool{}
	if state == nil || state.Document == nil {
		return out, nil
	}
	a, err := readCurrentActions(state.Document)
	if err != nil || a == nil {
		return out, err
	}
	objects, err := currentTerminalActorBindingsFromActions(state, a)
	if err != nil {
		return nil, err
	}
	for object := range objects {
		key, err := savedStructureValue(&state.Document.Objects[object-1], "Identity")
		if err != nil || key == 0 || out[key] {
			return nil, fmt.Errorf("current terminal actor has an ambiguous SAV identity")
		}
		out[key] = true
	}
	return out, nil
}

func currentTerminalActorBindingsFromActions(state *SnapshotSAVDocument, a *currentActionData) (map[uint16]bool, error) {
	out := map[uint16]bool{}
	keys := map[uint32]bool{}
	for id, values := range a.Values {
		if values.CurrentTerminal == nil {
			continue
		}
		object, key, err := currentTerminalActionObject(state, a, id)
		if err != nil || object != 0 && (out[object] || keys[key]) {
			return nil, fmt.Errorf("current terminal actor %d has no unique object", id)
		}
		if object != 0 {
			out[object], keys[key] = true, true
		}
	}
	return out, nil
}
