package ui

type MapStructureUse func(entity, structure uint32)

func (v *Viewer) SetStructureUseSink(fn MapStructureUse) { v.structureUseSink = fn }

func (v *Viewer) usableStructure(ref InspectionSubject) bool {
	if ref.Kind != InspectionStructure {
		return false
	}
	c := v.structureInfo[ref.ID]
	return c != nil && c.Usable && (c.ID == 15 || c.ID == 16 || c.ID == 28 || c.ID == 29)
}
