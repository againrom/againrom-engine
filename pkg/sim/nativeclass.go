package sim

import "fmt"

type NativeClass struct {
	Present bool
	Fighter bool
}

// skillMage selects the class arm without using a changing mana maximum.
func skillMage(e Entity) bool {
	if e.ActorLoad.Source.Class == 2 {
		return !e.ActorLoad.Source.Fighter
	}
	if e.NativeClass.Present {
		return !e.NativeClass.Fighter
	}
	return isMage(e)
}

func (w *World) SetNativeClass(id EntityID, class NativeClass) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].ActorLoad.Source.Class != 0 || class.Present && !w.entities[i].Humanoid || !class.Present && class.Fighter {
		return false
	}
	w.entities[i].NativeClass = class
	return true
}

func (w *World) nativeClassFault() error {
	for _, e := range w.entities {
		if !e.NativeClass.Present && e.NativeClass.Fighter || e.NativeClass.Present && (e.ActorLoad.Source.Class != 0 || !e.Humanoid) {
			return fmt.Errorf("sim: invalid native class for actor %d", e.ID)
		}
	}
	return nil
}
