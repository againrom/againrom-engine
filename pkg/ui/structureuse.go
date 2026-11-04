package ui

type MapStructureUse func(entity, structure uint32)

func (v *Viewer) SetStructureUseSink(fn MapStructureUse) { v.structureUseSink = fn }

func (v *Viewer) structureState(ref InspectionSubject) (MapStructure, bool) {
	if ref.Kind != InspectionStructure {
		return MapStructure{}, false
	}
	for _, s := range v.structureStates {
		if s.ID == ref.ID {
			return s, true
		}
	}
	return MapStructure{}, false
}

func (v *Viewer) usableStructure(ref InspectionSubject) bool {
	s, ok := v.structureState(ref)
	return ok && s.UseAdmission != nil && s.UseAdmission.Usable()
}

// structurePickGate reports whether the structure at (col,row) may be picked
// now: in current sight, or, when the game's mission screen data admits it, a
// usable structure on an explored cell, where it stays drawn.
func (v *Viewer) structurePickGate(ref InspectionSubject, col, row int) bool {
	return v.fogGateSack(col, row) || v.missionScreen.PickUsableInExploredGround && v.usableStructure(ref) && v.fogGateGround(col, row)
}
