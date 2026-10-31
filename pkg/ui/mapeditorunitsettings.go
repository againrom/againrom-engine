package ui

import (
	"fmt"
	"image"
	"slices"
)

func (e *MapEditor) nativeUnitWindow() bool {
	return len(e.obj.pending) > 0 || slices.ContainsFunc(e.obj.fields, func(f InspectionField) bool { return f.Section != "" })
}

func (e *MapEditor) unitWindowWidth() int {
	if e.objectInPanel() {
		// The person window takes the window less 32: in the panel it fills it.
		return e.sidePanelRect(panelRight).Dx() + 32
	}
	if width := e.sizes[popObject].X; width > 0 {
		return min(e.window.X, width)
	}
	return e.window.X
}

type InspectionUnitSettings interface {
	UnitSettings(int, []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error)
}

type InspectionUnitPreview struct {
	Doll, Face *image.RGBA
	FaceWindow image.Rectangle
}

type unitSettingsView struct {
	key     string
	entity  MapEntity
	fields  []InspectionField
	picture *InspectionUnitPreview
	ok      bool
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
		values = append(values, o.pending...)
		valid := true
		for i, f := range o.fields {
			if f.ReadOnly {
				continue
			}
			if o.buf[i] == fieldInput(f) {
				continue
			}
			v, err := fieldNumber(f, o.buf[i])
			if err != nil {
				valid = false
				continue
			}
			values = append(values, InspectionFieldValue{Key: f.Key, Value: v, Step: uint32(len(o.pending) + 1)})
		}
		entity, fields, picture, err := host.UnitSettings(index, values)
		if err == nil {
			o.unit.entity, o.unit.fields, o.unit.ok = entity, fields, true
			o.unit.picture = picture
			if valid && e.nativeUnitWindow() && len(values) > len(o.pending) {
				focus := ""
				typing := false
				if o.focus >= 0 && o.focus < len(o.fields) {
					focus = o.fields[o.focus].Key
					typing = e.input.target == &o.buf[o.focus]
				}
				o.pending = values
				o.fields, o.buf, o.focus, o.open = fields, nil, -1, -1
				for i, f := range fields {
					o.buf = append(o.buf, fieldInput(f))
					if f.Key == focus {
						o.focus = i
					}
				}
				if typing && o.focus >= 0 && o.buf[o.focus] == e.input.model.Text {
					e.input.target = &o.buf[o.focus]
				}
				o.unit.key = fmt.Sprint(o.buf)
				o.note = ""
			}
		} else if e.nativeUnitWindow() {
			o.note = err.Error()
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
