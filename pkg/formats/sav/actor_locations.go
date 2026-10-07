package sav

import "sort"

// DocumentActorLocation exposes only structure starts within a Unit-family
// record. Off is the Token start, RawBlocksOff starts the six raw blocks,
// ControlOff starts the scalar control run, and StateOff starts the CString
// followed by the scalar state run. The independent reader owns field widths,
// counts and values. ArchiveIndex is an original-stream identity, not native.
// HumanoidXPOff is zero for Unit and starts the raw XP run for Human/Humanoid.
// RoutesOff starts the first embedded u16 list's count, after the effects list.
type DocumentActorLocation struct {
	ArchiveIndex                                           uint16
	Class                                                  string
	Off, RawBlocksOff, ControlOff, StateOff, HumanoidXPOff int
	RoutesOff                                              int
}

func (f *File) DocumentActorLocations() ([]DocumentActorLocation, error) {
	doc, _, err := f.exactDocument()
	if err != nil {
		return nil, err
	}
	var out []DocumentActorLocation
	for index, record := range doc.objects {
		if record.Class == "Unit" || record.Class == "Human" || record.Class == "Humanoid" {
			out = append(out, DocumentActorLocation{
				ArchiveIndex: index, Class: record.Class, Off: record.Off,
				RawBlocksOff: record.UnitBlocksOff, ControlOff: record.UnitControlOff,
				StateOff: record.UnitStateOff, HumanoidXPOff: record.HumanoidXPOff,
				RoutesOff: record.UnitRoutesOff,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ArchiveIndex < out[j].ArchiveIndex })
	return out, nil
}
