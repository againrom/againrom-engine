package ui

import (
	"fmt"
	"strconv"
)

type InspectionUnitSettings interface {
	UnitSettings(int, []InspectionFieldValue) (MapEntity, []InspectionField, error)
}

type unitSettingsView struct {
	key    string
	entity MapEntity
	fields []InspectionField
	ok     bool
}

func (e *MapEditor) unitSettings() (*PanelSubject, []InspectionField) {
	kind, index := e.selectedRecord()
	host, ok := e.doc.Edits.(InspectionUnitSettings)
	if kind != "Unit" || !ok {
		return nil, nil
	}
	o := &e.obj
	key := fmt.Sprint(o.buf)
	if o.unit.key != key {
		o.unit = unitSettingsView{key: key}
		var values []InspectionFieldValue
		for i, f := range o.fields {
			if o.buf[i] == fieldInput(f) {
				continue
			}
			v, err := strconv.ParseInt(o.buf[i], 10, 64)
			if err != nil {
				return nil, nil
			}
			values = append(values, InspectionFieldValue{Key: f.Key, Value: v})
		}
		entity, fields, err := host.UnitSettings(index, values)
		if err == nil {
			o.unit.entity, o.unit.fields, o.unit.ok = entity, fields, true
		}
	}
	if !o.unit.ok {
		return nil, nil
	}
	s := e.doc.Viewer.panelSubjectFromPresent([]MapEntity{o.unit.entity})
	if o.move != nil {
		s.Cell = *o.move
	}
	return &s, o.unit.fields
}

func (e *MapEditor) UnitCharacterSubject() (PanelSubject, bool) {
	if e.doc == nil {
		return PanelSubject{}, false
	}
	e.loadObject()
	s, _ := e.unitSettings()
	if s == nil {
		return PanelSubject{}, false
	}
	return *s, true
}
