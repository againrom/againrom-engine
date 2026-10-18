package game

import (
	"againrom/pkg/mapedit"
	"fmt"
	"strconv"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

type inspectionUnitViews struct {
	model        *mapedit.Editor
	actual, bare []unitSheet
	entities     []ui.MapEntity
}

func (s *mapInspectionEdits) unitViewCache() *inspectionUnitViews {
	if s.unitViews == nil || s.unitViews.model != s.model {
		s.unitViews = &inspectionUnitViews{model: s.model}
	}
	return s.unitViews
}

func (s *mapInspectionEdits) unitViewSheets(bare bool) []unitSheet {
	c := s.unitViewCache()
	if bare && c.bare != nil {
		return c.bare
	}
	if !bare && c.actual != nil {
		return c.actual
	}
	m, err := s.model.Map()
	if err != nil {
		return nil
	}
	if bare {
		c.bare, _ = templateSheets(m, s.inspector.table)
		return c.bare
	}
	c.actual, _ = unitSheets(m, s.inspector.table)
	return c.actual
}

func (s *mapInspectionEdits) UnitSettings(index int, values []ui.InspectionFieldValue) (ui.MapEntity, []ui.InspectionField, error) {
	trial := *s
	if len(values) > 0 {
		trial.model = s.model.Clone()
		trial.batch = trial.model
		if _, err := trial.SetFields("Unit", index, values); err != nil {
			return ui.MapEntity{}, nil, err
		}
	}
	c := trial.unitViewCache()
	if c.entities != nil {
		if index >= 0 && index < len(c.entities) {
			entity := c.entities[index]
			entity.Knowledge, entity.KnowledgeKnown = 7, true
			return entity, trial.Fields("Unit", index), nil
		}
	}
	m, err := trial.model.Map()
	if err != nil {
		return ui.MapEntity{}, nil, err
	}
	v, err := ui.NewViewer("", terrain.Grid{Width: m.Width, Height: m.Height, Tiles: terrain.RenderTileWords(m.Tiles), Altitudes: m.Altitudes}, s.inspector.tiles)
	if err != nil {
		return ui.MapEntity{}, nil, err
	}
	if s.inspector.panelWords != nil {
		v.SetWords(*s.inspector.panelWords)
	}
	mw, err := openMapWorld(m, s.inspector.table, s.inspector.units, v)
	if err != nil {
		return ui.MapEntity{}, nil, err
	}
	c.entities = mw.entityDraws()
	if len(values) == 0 {
		s.unitViews = c
	}
	for _, entity := range c.entities {
		if int(entity.ID) == index {
			entity.Knowledge, entity.KnowledgeKnown = 7, true
			return entity, trial.Fields("Unit", index), nil
		}
	}
	return ui.MapEntity{}, nil, fmt.Errorf("unit %d has no actor", index)
}

func (u *unitTemplateView) meaning(key string, value int64) string {
	if u == nil {
		return ""
	}
	x := u.s.inspector
	switch key {
	case "owner":
		return fmt.Sprintf(x.word("editor.unit.meaning.owner"), value)
	case "class":
		return fmt.Sprintf(x.word("editor.unit.meaning.classkey"), value)
	case "tier":
		return fmt.Sprintf(x.word("editor.unit.meaning.tier"), value)
	case "definition":
		switch mapload.Resolve(u.m.Units[u.index], x.table).Arm {
		case mapload.ArmUnits:
			return x.word("editor.unit.meaning.unitdefinition")
		case mapload.ArmNPC:
			return fmt.Sprintf(x.word("editor.unit.meaning.npcdefinition"), u.m.Units[u.index].ClassSubID)
		}
		if value == 0 || value == 0xcdcdcdcd {
			return x.word("editor.unit.meaning.class")
		}
		return fmt.Sprintf(x.word("editor.unit.meaning.definition"), value)
	case "group":
		return fmt.Sprintf(x.word("editor.unit.meaning.group"), value)
	case "gear":
		if value == 0 {
			return x.word("editor.unit.meaning.gear")
		}
		return x.word("editor.unit.meaning.skipgear")
	}
	if _, ok := overrideTemplate(unitValues{}, key); !ok || x.table == nil {
		return ""
	}
	if !u.actualBuilt {
		u.actualBuilt = true
		if sheets := u.s.unitViewSheets(false); u.index < len(sheets) {
			u.actual, u.actualOK = sheets[u.index].Values, true
		}
	}
	if !u.actualOK {
		return ""
	}
	actual, _ := overrideTemplate(u.actual, key)
	if value == overrideAbsent(key) {
		return fmt.Sprintf(x.word("editor.unit.meaning.template"), actual)
	}
	if mapload.Resolve(u.m.Units[u.index], x.table).Arm == mapload.ArmNPC && key != "hp" && key != "mana" && key != "body" && key != "mind" && key != "spirit" && key != "reaction" {
		return fmt.Sprintf(x.word("editor.unit.meaning.ignored"), actual)
	}
	if actual != value {
		return fmt.Sprintf(x.word("editor.unit.meaning.result"), actual)
	}
	return fmt.Sprintf(x.word("editor.unit.meaning.explicit"), strconv.FormatInt(value, 10))
}

var _ ui.InspectionUnitSettings = (*mapInspectionEdits)(nil)
