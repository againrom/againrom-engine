package game

import (
	"fmt"
	"math"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// A group is the group number its member units carry in the unit record
// (ALM-UNIT-048); a script names a group by that number alone (ALM-TRIG-046),
// so one number belongs to one player's units. The editor's group actions
// write that field on each unit, as the unit popup's group field does.

// SetUnitsGroup writes group on every unit refs names as one edit; group 0
// takes them out of their groups. It refuses a record that is not a unit,
// units of two players, and a number a unit of another player carries.
func (s *mapInspectionEdits) SetUnitsGroup(refs []ui.RecordRef, group int) (*ui.InspectionDocument, error) {
	view, err := s.model.Map()
	if err != nil {
		return nil, err
	}
	if group < 0 || int64(group) > math.MaxUint32 {
		return nil, fmt.Errorf("group number %d is outside 0..%d", group, uint32(math.MaxUint32))
	}
	owner := int64(-1)
	in := map[int]bool{}
	for _, ref := range refs {
		if ref.Kind != "Unit" || ref.Index < 0 || ref.Index >= len(view.AuthoredUnits) {
			return nil, fmt.Errorf("no unit %s #%d", ref.Kind, ref.Index)
		}
		u := view.AuthoredUnits[ref.Index]
		if owner >= 0 && int64(u.Owner) != owner {
			return nil, fmt.Errorf("units of players %d and %d cannot share a group", owner, u.Owner)
		}
		owner, in[ref.Index] = int64(u.Owner), true
	}
	if group != 0 {
		for i, u := range view.AuthoredUnits {
			if !in[i] && int64(u.GroupID) == int64(group) && int64(u.Owner) != owner {
				return nil, fmt.Errorf(s.inspector.word("editor.add.group.other"), group, u.Owner)
			}
		}
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		changed := false
		mark := m.Mark()
		for _, ref := range refs {
			if int64(view.AuthoredUnits[ref.Index].GroupID) == int64(group) {
				continue
			}
			if err := m.SetField(mapedit.KindUnit, ref.Index, "group", int64(group)); err != nil {
				return false, err
			}
			changed = true
		}
		m.Squash(mark)
		return changed, nil
	})
}

// FreeGroup is the lowest group number no unit of the map carries.
func (s *mapInspectionEdits) FreeGroup() int { return int(s.freeGroup()) }
