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
