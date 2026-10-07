package sim

import "fmt"

// CurrentArchiveCoordinate relocates only file-local construction coordinates.
// A zero destination denotes native absence; object keys and gameplay values
// remain the ordinary record's current values.
type CurrentArchiveCoordinate struct {
	From, To uint16
}

// CurrentSourceGroup is read from a bound ordinary Group during LOAD. It is
// not persisted: only that Group's identity/site may live in a supplement.
type CurrentSourceGroup struct {
	Index         uint32
	Record        *SavedGroup
	Absent        bool
	AdoptSelector bool
}

func (w *World) RestoreCurrentArchiveCoordinates(rows []CurrentArchiveCoordinate, groups map[EntityID]CurrentSourceGroup) error {
	remap, used := map[uint16]uint16{}, map[uint16]bool{}
	current := map[uint16]bool{}
	w.currentArchiveFields(func(index *uint16) {
		if *index != 0 {
			current[*index] = true
		}
	})
	for _, row := range rows {
		if row.From == 0 {
			return fmt.Errorf("sim: null current archive source")
		}
		if _, duplicate := remap[row.From]; duplicate || row.To != 0 && used[row.To] {
			return fmt.Errorf("sim: aliased current archive coordinates")
		}
		remap[row.From], used[row.To] = row.To, true
	}
	for index := range current {
		if _, moved := remap[index]; !moved && used[index] {
			return fmt.Errorf("sim: current archive coordinate collides with an unmoved holder")
		}
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		return err
	}
	n := *w
	if err := n.UnmarshalBinary(raw); err != nil {
		return err
	}
	move := func(index *uint16) {
		if current, ok := remap[*index]; ok {
			*index = current
		}
	}
	n.currentArchiveFields(move)
	seenGroups := map[EntityID]bool{}
	for i := range n.entities {
		e := &n.entities[i]
		if e.SourceBinding.Class != 0 {
			if e.SourceBinding.ArchiveIndex == 0 && !e.SourceBinding.Generated() {
				e.SourceBinding.Class += 3
			}
			if group, ok := groups[e.ID]; ok {
				e.SourceBinding.GroupIndex = group.Index
				if group.Record != nil {
					g := group.Record
					e.SourceBinding.GroupSelector, e.SourceBinding.GroupOwnerKey = g.Selector, g.Owner.Key
					e.SourceBinding.GroupOwnerSlot, e.SourceBinding.GroupOwnerResolved = uint16(g.Owner.Owner), g.Owner.Class != 0
					if group.AdoptSelector {
						e.Group = g.Selector
					}
				} else if group.Absent {
					e.SourceBinding.GroupSelector, e.SourceBinding.GroupOwnerKey = 0, 0
					e.SourceBinding.GroupOwnerSlot, e.SourceBinding.GroupOwnerResolved = 0, false
					if group.AdoptSelector {
						e.Group = 0
					}
				}
			}
		}
		if _, ok := groups[e.ID]; ok {
			seenGroups[e.ID] = true
		}
	}
	if len(groups) != len(seenGroups) {
		return fmt.Errorf("sim: current source Group has no actor")
	}
	for i := range n.originalDead {
		s := &n.originalDead[i].Source
		if s.ArchiveIndex == 0 && !(SourceBinding{Class: s.Class}).Generated() {
			s.Class += 3
		}
	}
	for i := range n.savedStructures {
		s := &n.savedStructures[i]
		if s.ArchiveIndex == 0 && !s.Class.Generated() {
			s.Class += 4
		}
	}
	checked, err := n.MarshalBinary()
	if err != nil {
		return err
	}
	if err := n.UnmarshalBinary(checked); err != nil {
		return err
	}
	*w = n
	return nil
}

func (w *World) currentArchiveFields(ref func(*uint16)) {
	for i := range w.entities {
		if w.entities[i].SourceBinding.Class != 0 {
			ref(&w.entities[i].SourceBinding.ArchiveIndex)
		}
	}
	for i := range w.originalDead {
		s := &w.originalDead[i].Source
		ref(&s.ArchiveIndex)
		if s.HeldWeapon.Present {
			ref(&s.HeldWeapon.ArchiveIndex)
		}
	}
	for i := range w.savedStructures {
		ref(&w.savedStructures[i].ArchiveIndex)
	}
	if w.savedGroups != nil {
		for i := range w.savedGroups.Groups {
			g := &w.savedGroups.Groups[i]
			ref(&g.Reference.Archive)
			ref(&g.Owner.Archive)
			for j := range g.Members {
				ref(&g.Members[j].Archive)
			}
		}
	}
}
