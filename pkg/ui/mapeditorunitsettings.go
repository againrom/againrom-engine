package ui

import (
	"fmt"
	"image"
	"slices"
	"time"
)

// unitTypingPause is how long a unit window text field must stay unchanged
// while it is typed in before the preview is built for it.
const unitTypingPause = 300 * time.Millisecond

func (e *MapEditor) nativeUnitWindow() bool {
	return len(e.obj.pending) > 0 || slices.ContainsFunc(e.obj.fields, func(f InspectionField) bool { return f.Section != "" })
}

func (e *MapEditor) unitWindowWidth() int {
	if e.objectInPanel() {
		// The person window takes the window less 32: in the panel it fills it.
		return e.sidePanelRect(panelRight).Dx() + 32
	}
	if width := e.places[e.storeRow(popObject)].size.X; width > 0 {
		return min(e.window.X, width)
	}
	return e.window.X
}

type InspectionUnitSettings interface {
	UnitSettings(int, []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error)
}

// InspectionUnitPreview is the person window's figure: the unit's doll as
// the game composes it.
type InspectionUnitPreview struct {
	Doll *image.RGBA
}

type unitSettingsView struct {
	key     string
	entity  MapEntity
	fields  []InspectionField
	picture *InspectionUnitPreview
	ok      bool
	// heldKey is the draft a typing pause is waiting out, and heldAt when it
	// last changed.
	heldKey string
	heldAt  time.Time
}

func (e *MapEditor) unitSettings() (*PanelSubject, []InspectionField) {
	return e.unitSettingsFor(false)
}

// unitTextTyping reports that the keyboard edits a text field of the unit
// window.
func (e *MapEditor) unitTextTyping() bool {
	o := &e.obj
	return e.objectTyping() && o.fields[o.focus].Text && e.input.target == &o.buf[o.focus]
}

// unitSettingsFor builds the unit window's preview for the draft. While a text
// field is typed in, the draft is built once it has stayed unchanged for
// unitTypingPause; commit builds it at once.
func (e *MapEditor) unitSettingsFor(commit bool) (*PanelSubject, []InspectionField) {
	kind, index := e.selectedRecord()
	host, ok := e.doc.Edits.(InspectionUnitSettings)
	if kind != KindUnit || !ok {
		return nil, nil
	}
	o := &e.obj
	key := fmt.Sprint(o.buf)
	rebuild := o.unit.key != key
	if rebuild && !commit && o.unit.ok && e.unitTextTyping() {
		now := time.Now()
		if o.unit.heldKey != key {
			o.unit.heldKey, o.unit.heldAt = key, now
		}
		rebuild = (cadence{period: unitTypingPause, gate: gateAtLeast}).due(o.unit.heldAt, now)
	}
	if rebuild {
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
			if f.Text {
				values = append(values, InspectionFieldValue{Key: f.Key, Text: true, TextValue: o.buf[i], Step: uint32(len(o.pending) + 1)})
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

// unitFigureSize is the size of the shown person's figure as the game draws
// it, zero while the host gives none.
func (e *MapEditor) unitFigureSize() image.Point {
	if p := e.obj.unit.picture; p != nil && p.Doll != nil {
		return p.Doll.Bounds().Size()
	}
	return image.Point{}
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
