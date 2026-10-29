package game

import (
	"againrom/pkg/mapedit"
	"fmt"
	"strconv"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type inspectionUnitViews struct {
	model    *mapedit.Editor
	actual   []unitSheet
	entities []ui.MapEntity
	pictures map[uint32]*ui.InspectionUnitPreview
}

func (s *mapInspectionEdits) unitViewCache() *inspectionUnitViews {
	if s.unitViews == nil || s.unitViews.model != s.model {
		s.unitViews = &inspectionUnitViews{model: s.model}
	}
	return s.unitViews
}

func (s *mapInspectionEdits) unitViewSheets() []unitSheet {
	c := s.unitViewCache()
	if c.actual != nil {
		return c.actual
	}
	m, err := s.model.Map()
	if err != nil {
		return nil
	}
	c.actual, _ = unitSheets(m, s.inspector.table)
	return c.actual
}

func (s *mapInspectionEdits) UnitSettings(index int, values []ui.InspectionFieldValue) (ui.MapEntity, []ui.InspectionField, *ui.InspectionUnitPreview, error) {
	trial := *s
	if len(values) > 0 {
		trial.model = s.model.Clone()
		trial.batch = trial.model
		if _, err := trial.SetFields("Unit", index, values); err != nil {
			return ui.MapEntity{}, nil, nil, err
		}
	}
	c := trial.unitViewCache()
	if c.entities != nil {
		if index >= 0 && index < len(c.entities) {
			entity := c.entities[index]
			entity.Knowledge, entity.KnowledgeKnown = 7, true
			if pic, known := c.pictures[entity.ID]; known {
				return entity, trial.Fields("Unit", index), pic, nil
			}
		}
	}
	m, err := trial.model.Map()
	if err != nil {
		return ui.MapEntity{}, nil, nil, err
	}
	v, err := ui.NewViewer("", terrain.Grid{Width: m.Width, Height: m.Height, Tiles: terrain.RenderTileWords(m.Tiles), Altitudes: m.Altitudes}, s.inspector.tiles)
	if err != nil {
		return ui.MapEntity{}, nil, nil, err
	}
	if s.inspector.panelWords != nil {
		v.SetWords(*s.inspector.panelWords)
	}
	mw, err := openMapWorld(m, s.inspector.table, s.inspector.units, v)
	if err != nil {
		return ui.MapEntity{}, nil, nil, err
	}
	c.entities = mw.entityDraws()
	if c.pictures == nil {
		c.pictures = make(map[uint32]*ui.InspectionUnitPreview)
	}
	for _, entity := range c.entities {
		if int(entity.ID) != index {
			continue
		}
		id := sim.EntityID(entity.ID)
		c.pictures[entity.ID] = nil
		if fig, ok := mw.figures[id]; ok {
			r := speakerResolver{src: s.inspector.archives.artSource(), figure: &fig, equipment: mw.equipmentOf(id)}
			speaker := -1
			if catalog := s.inspector.table.NPCCatalog; catalog != nil {
				r.npcFaces = catalog.Faces
			}
			if m.Units[index].Flags&1 != 0 {
				speaker = int(m.Units[index].ClassSubID)
			}
			pic, window, _ := r.SpeakerFace(speaker)
			if pic != nil {
				c.pictures[entity.ID] = &ui.InspectionUnitPreview{Doll: pic, Face: pic, FaceWindow: window}
			}
		}
	}
	if len(values) == 0 {
		s.unitViews = c
	}
	for _, entity := range c.entities {
		if int(entity.ID) == index {
			entity.Knowledge, entity.KnowledgeKnown = 7, true
			return entity, trial.Fields("Unit", index), c.pictures[entity.ID], nil
		}
	}
	return ui.MapEntity{}, nil, nil, fmt.Errorf("unit %d has no actor", index)
}

// describe fills the unit popup's view of field row: an identity field's
// meaning; an override field's group, its stored "inherit" value and the
// value the game uses while it inherits; the equipment flag's two choices.
func (u *unitTemplateView) describe(row *ui.InspectionField) {
	if u == nil {
		return
	}
	x := u.s.inspector
	if row.Key == "npc" {
		row.Meaning = x.word("editor.unit.npc.note")
		return
	}
	if row.Key == "face" || row.Key == "sex" {
		row.Group = x.word("editor.unit.look")
		row.Meaning = x.word("editor.unit.look.note")
		return
	}
	if row.Key == "gear" {
		row.Group = x.word("editor.unit.group.equipment")
		row.Choices = []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.unit.meaning.gear"))}, {Value: 1, Label: x.inspectionUTF8(x.word("editor.unit.meaning.skipgear"))}}
		return
	}
	if group := overrideGroup(x, row.Key); group != "" {
		row.Group, row.Inherit, row.Absent = group, true, overrideAbsent(row.Key)
	}
	row.Meaning = u.meaning(row.Key, row.Value)
	if row.Inherit && row.Value == row.Absent && u.actualValues() {
		actual, _ := overrideTemplate(u.actual, row.Key)
		row.Inherited = strconv.FormatInt(actual, 10)
	}
}

// actualValues builds the placement's in-game values once; false when the
// tables cannot build them.
func (u *unitTemplateView) actualValues() bool {
	if !u.actualBuilt {
		u.actualBuilt = true
		if sheets := u.s.unitViewSheets(); u.index < len(sheets) {
			u.actual, u.actualOK = sheets[u.index].Values, true
		}
	}
	return u.actualOK
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
	}
	if _, ok := overrideTemplate(unitValues{}, key); !ok || x.table == nil || value == overrideAbsent(key) || !u.actualValues() {
		return ""
	}
	// A set value carries a line only where the game does not show it as
	// typed: an NPC-arm placement skips the stores (DIV-3580), and the
	// derivation or a lone Shooting byte (DIV-3581) can leave another value.
	actual, _ := overrideTemplate(u.actual, key)
	if mapload.Resolve(u.m.Units[u.index], x.table).Arm == mapload.ArmNPC && key != "hp" && key != "mana" && key != "body" && key != "mind" && key != "spirit" && key != "reaction" {
		return fmt.Sprintf(x.word("editor.unit.meaning.ignored"), actual)
	}
	if actual != value {
		return fmt.Sprintf(x.word("editor.unit.meaning.result"), actual)
	}
	return ""
}

var _ ui.InspectionUnitSettings = (*mapInspectionEdits)(nil)
