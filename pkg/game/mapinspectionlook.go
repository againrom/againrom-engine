package game

import (
	"fmt"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func (x *MapInspector) lookFields(m *mapedit.Editor, index int, key string, value int64) ([]mapedit.FieldSet, error) {
	view, err := m.Map()
	if err != nil {
		return nil, err
	}
	u := view.AuthoredUnits[index]
	if !x.table.ClassPlacementFields() || !gearOffered(u, x.table) {
		return nil, fmt.Errorf("face and sex placement fields require an ordinary Human; this placement takes the %s arm", mapload.Resolve(u, x.table).Arm)
	}
	if u.ClassSubID&0x7f != 0 {
		return nil, nil
	}
	dir, face, ok := mapload.PlacedFigure(u, x.table)
	if !ok || face < 1 || face > 127 {
		return nil, fmt.Errorf("the inherited face cannot be represented by this placement")
	}
	if key == "sex" {
		return []mapedit.FieldSet{{Key: "face", Value: int64(face)}}, nil
	}
	if value != 0 {
		sex := int64(0)
		if dir.Female() {
			sex = 1
		}
		return []mapedit.FieldSet{{Key: "sex", Value: sex}}, nil
	}
	return nil, nil
}

func (u *unitTemplateView) unitFieldValue(row *ui.InspectionField) {
	if u != nil && row.Key == "face" {
		dir, _, ok := mapload.PlacedFigure(u.m.Units[u.index], u.s.inspector.table)
		if ok {
			row.Icons = u.s.inspector.recordPictures[fmt.Sprintf("face:%s", dir)]
		}
	}
	if u != nil && row.Key == "npc" {
		row.Icons = u.s.inspector.npcPortraits(row.Choices)
	}
	if u != nil && row.Key == "npc" && u.m.Units[u.index].Flags&1 == 0 {
		row.Value = -1
		return
	}
	if u == nil || row.Key != "sex" || u.m.Units[u.index].ClassSubID&0x7f != 0 {
		return
	}
	dir, _, ok := mapload.PlacedFigure(u.m.Units[u.index], u.s.inspector.table)
	if ok {
		row.Value = 0
		if dir.Female() {
			row.Value = 1
		}
	}
}
