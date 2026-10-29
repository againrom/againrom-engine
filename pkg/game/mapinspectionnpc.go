package game

import (
	"fmt"
	"math"
	"slices"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func (x *MapInspector) npcFields(m *mapedit.Editor, index int, value int64) (int64, []mapedit.FieldSet, error) {
	if !x.table.ClassPlacementFields() || value < -1 || value > math.MaxUint16 {
		return 0, nil, fmt.Errorf("NPC record is outside this placement's native key domain")
	}
	view, err := m.Map()
	if err != nil {
		return 0, nil, err
	}
	u := view.AuthoredUnits[index]
	if mapload.Resolve(u, x.table).Arm == mapload.ArmUnits {
		return 0, nil, fmt.Errorf("a creature placement cannot take an NPC record")
	}
	if value == -1 && u.Flags&1 == 0 {
		return int64(u.ClassSubID), nil, nil
	}
	probe := u
	flag := int64(0)
	if value >= 0 {
		if _, ok := x.table.NPCCatalog.Entry(int32(value)); !ok {
			return 0, nil, fmt.Errorf("NPC record %d does not exist", value)
		}
		probe.Flags |= 1
		probe.ClassSubID, flag = uint16(value), 1
	} else {
		probe.Flags &^= 1
		probe.ClassSubID = 0
	}
	if !mapload.Resolve(probe, x.table).Found() {
		return 0, nil, fmt.Errorf("NPC choice %d resolves to no Human definition", value)
	}
	if probe.Flags == u.Flags {
		return int64(probe.ClassSubID), nil, nil
	}
	return int64(probe.ClassSubID), []mapedit.FieldSet{{Key: "npcflag", Value: flag}}, nil
}

func (x *MapInspector) npcChoices(maxID int64, roles bool) []ui.InspectionChoice {
	var choices []ui.InspectionChoice
	if x.table == nil {
		return choices
	}
	for _, e := range x.table.NPCCatalog.Records() {
		id := int64(e.ID)
		if id < 0 || id > maxID {
			continue
		}
		if roles {
			if role, ok := x.speakerRole(id); ok {
				choices = append(choices, ui.InspectionChoice{Value: id, Label: fmt.Sprintf("%d: %s", id, x.inspectionUTF8(role))})
				continue
			}
		}
		name := e.Name
		line := id
		if !roles {
			line--
		}
		if name == "" && line >= 0 && line < int64(len(x.table.NPCNames)) {
			name = x.table.NPCNames[line]
		}
		if name == "" {
			name = e.Section
		}
		choices = append(choices, ui.InspectionChoice{Value: id, Label: fmt.Sprintf("%d: %s", id, x.inspectionUTF8(x.messageUTF8([]byte(name))))})
	}
	slices.SortFunc(choices, func(a, b ui.InspectionChoice) int { return int(a.Value - b.Value) })
	return choices
}
