package sim

import "fmt"

type NativeTraining struct {
	Present bool
	Levels  [skillSlots]int32
}

// TrainedSkills keeps historical actors on their old inverse until a caller
// supplies a known base. Saturated historical levels cannot recover lost history.
func (e Entity) TrainedSkills(bonus [skillSlots]int32) [skillSlots]int32 {
	if e.NativeTraining.Present {
		return e.NativeTraining.Levels
	}
	base := e.Skill
	for j := range base {
		base[j] -= bonus[j]
		if j > 0 {
			base[j] = max(0, base[j])
		}
	}
	return base
}

// SetNativeTraining installs a known base without changing the effective sheet.
// An optional repair mask updates only already-known bytes of selected trained
// words. Unknown bytes and the remaining history keep their current values.
func (w *World) SetNativeTraining(id EntityID, training NativeTraining, repaired ...[skillSlots]bool) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].ActorLoad.Source.Class != 0 || !training.Present && training.Levels != ([skillSlots]int32{}) {
		return false
	}
	e := &w.entities[i]
	e.NativeTraining = training
	if training.Present && len(repaired) != 0 {
		for slot := 1; slot < skillSlots; slot++ {
			if !repaired[0][slot] {
				continue
			}
			at, level := 2+2*slot, uint16(training.Levels[slot])
			if e.NativeBasis.BaseByteKnown(at) {
				e.NativeBasis.Base[at] = byte(level)
			}
			if e.NativeBasis.BaseByteKnown(at + 1) {
				e.NativeBasis.Base[at+1] = byte(level >> 8)
			}
		}
	}
	return true
}

func (w *World) nativeSkillBonuses(i int) [skillSlots]int32 {
	var bonus [skillSlots]int32
	for j := range bonus {
		bonus[j] = w.wornSkillBonus(i, int32(j))
	}
	return bonus
}

// NativeTrainingNeedsProducer reports an imported sheet the worn-item producer
// cannot reproduce from its known base. It infers no missing bonus.
func (w *World) NativeTrainingNeedsProducer(id EntityID) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || !w.hasSessionClock && w.savedObjects == nil {
		return false
	}
	e := &w.entities[i]
	if e.ActorLoad.Source.Class != 0 || !e.NativeTraining.Present {
		return false
	}
	bonus := w.nativeSkillBonuses(i)
	for j := 1; j < skillSlots; j++ {
		if e.Skill[j] != w.rules.EffectiveSkill(e.NativeTraining.Levels[j], bonus[j]) {
			return true
		}
	}
	return false
}

func (w *World) nativeTrainingFault() error {
	for _, e := range w.entities {
		if !e.NativeTraining.Present && e.NativeTraining.Levels != ([skillSlots]int32{}) {
			return fmt.Errorf("sim: absent native training retains levels for actor %d", e.ID)
		}
	}
	return nil
}
