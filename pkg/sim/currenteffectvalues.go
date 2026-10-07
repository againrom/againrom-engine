package sim

import (
	"fmt"
	"slices"
)

type EffectNumericResidue struct {
	Spell uint16
	Wire  uint16
	Lift  int64
}

func restoreAttachedEffectWidths(w *World, values map[EntityID]ActorValues) error {
	next := slices.Clone(w.attached)
	for _, actor := range w.entities {
		rows := values[actor.ID].EffectWidths
		if len(rows) > 29 {
			return fmt.Errorf("sim: current effect width population exceeds spell slots")
		}
		seen := map[uint16]bool{}
		for _, row := range rows {
			magnitude := int64(int16(row.Wire)) + row.Lift
			if row.Spell > 28 || seen[row.Spell] || row.Lift == 0 || row.Lift%65536 != 0 || row.Lift < -1<<31 || row.Lift > 1<<31 || magnitude < -1<<31 || magnitude > 1<<31-1 {
				return fmt.Errorf("sim: invalid current effect magnitude lift")
			}
			seen[row.Spell] = true
			index := -1
			for i, effect := range next {
				if effect.Target == actor.ID && effect.Spell == row.Spell {
					index = i
					break
				}
			}
			if index < 0 {
				return fmt.Errorf("sim: current effect width lost its attachment")
			}
			effect := &next[index]
			if effect.Magnitude != int32(int16(effect.Magnitude)) {
				return fmt.Errorf("sim: current effect width requires ordinary magnitude")
			}
			if uint16(effect.Magnitude) == row.Wire {
				effect.Magnitude = int32(magnitude)
			}
		}
	}
	w.attached = next
	return nil
}
