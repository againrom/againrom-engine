package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

// A trigger parameter field that names a unit, a structure, a group or an
// area has an "on the map" button: the next click on the map fills the field
// with the value the field stores for what is under the pointer. The node
// popup and the trigger wizard share this pick.

// paramTarget is what a trigger parameter field stores of a map pick.
type paramTarget uint8

const (
	targetNone      paramTarget = iota
	targetUnit                  // a unit's number
	targetStructure             // a structure's number
	targetGroup                 // a group number, read from a unit of the group
	targetBox                   // an area as "x1,y1,x2,y2"
	targetSquare                // an area as a point "x,y" with a radius field beside it
)

// paramPickText is the status line's words of one target: what the click
// picks, and what it found when nothing fits.
type paramPickText struct {
	pickWord string
	missWord string
}

var paramPickTexts = map[paramTarget]paramPickText{
	targetUnit:      {pickWord: "editor.param.pickunit", missWord: "editor.param.nounit"},
	targetStructure: {pickWord: "editor.param.pickstructure", missWord: "editor.param.nostructure"},
	targetGroup:     {pickWord: "editor.param.pickgroup", missWord: "editor.param.nogroup"},
	targetBox:       {pickWord: "editor.param.pickarea", missWord: "editor.param.noarea"},
	targetSquare:    {pickWord: "editor.param.picksquare", missWord: "editor.param.nosquare"},
}

// pickedValue is what a click found: a number for a unit, a structure or a
// group, the area for an area.
type pickedValue struct {
	number int64
	area   InspectionArea
}

// nodeFieldTarget is what field f of a node popup stores of a map pick.
func (e *MapEditor) nodeFieldTarget(v scriptView, f int) paramTarget {
	o := &e.obj
	if v.node == nil || f < 0 || f >= len(o.fields) {
		return targetNone
	}
	n, ok := strings.CutPrefix(o.fields[f].Key, "value")
	slot, err := strconv.Atoi(n)
	if !ok || err != nil || slot < 0 || slot >= len(o.types) {
		return targetNone
	}
	switch e.nodeParameterKind(v, uint32(o.signature[0]), o.types, slot) {
	case scriptUnit:
		return targetUnit
	case scriptStructure:
		return targetStructure
	case scriptGroup:
		return targetGroup
	}
	return targetNone
}

// wizardFieldTarget is what field f of the trigger wizard stores of a map
// pick.
func (e *MapEditor) wizardFieldTarget(f int) paramTarget {
	r, p := e.wizardRowOf(f)
	if r < 0 || p < 0 {
		return targetNone
	}
	row := e.obj.wiz.rows[r]
	if row.item < 0 {
		return targetNone
	}
	params := wizItems(row.condition)[row.item].params
	if p >= len(params) {
		return targetNone
	}
	switch params[p].kind {
	case wpUnit:
		return targetUnit
	case wpGroup:
		return targetGroup
	case wpBox:
		return targetBox
	case wpCell:
		if e.wizardRadiusField(f) >= 0 {
			return targetSquare
		}
	}
	return targetNone
}

// pickFieldTarget is what the field the armed pick fills stores, none when
// that field is gone.
func (e *MapEditor) pickFieldTarget() paramTarget {
	o := &e.obj
	if o.pickField < 0 || o.pickField >= len(o.fields) || o.pickField >= len(o.buf) || o.doc != e.doc || o.record != e.selected {
		return targetNone
	}
	v, ok := e.scriptPopup()
	if !ok {
		return targetNone
	}
	if o.wiz != nil {
		return e.wizardFieldTarget(o.pickField)
	}
	return e.nodeFieldTarget(v, o.pickField)
}

// armParamPick arms the pick that fills field f: an area for an area button,
// else a unit, structure or group.
func (e *MapEditor) armParamPick(f int, area bool) {
	o := &e.obj
	o.focus, o.pickField = f, f
	e.leave(leaveDrag)
	if area {
		e.armPick(pickParamArea)
		return
	}
	e.armPick(pickParamUnit)
}

// paramPickStatus is the status line's part of the armed parameter pick: what
// the click picks, and what the last click found.
func (e *MapEditor) paramPickStatus() []string {
	var out []string
	if w := paramPickTexts[e.pickFieldTarget()].pickWord; w != "" {
		out = append(out, e.word(w))
	}
	if e.pickNote != "" {
		out = append(out, e.pickNote)
	}
	return out
}

// paramAt is a click of the armed parameter pick at window point p. A click
// that finds nothing the field can store leaves the field as it is and the
// pick armed.
func (e *MapEditor) paramAt(kind pickKind, p image.Point) {
	target := e.pickFieldTarget()
	e.leave(leavePick)
	if target == targetNone {
		return
	}
	var v pickedValue
	var ok bool
	if kind == pickParamArea {
		v, ok = e.areaValueAt(p, target)
	} else {
		v, ok = e.unitValueAt(p, target)
	}
	if !ok {
		e.keepPick(kind)
		e.pickNote = e.word(paramPickTexts[target].missWord)
		return
	}
	e.paramPicked(target, v)
}

// unitValueAt is the number the field of target stores for the record under
// p: the unit's or structure's number a script names it by, or the group
// number of the unit.
func (e *MapEditor) unitValueAt(p image.Point, target paramTarget) (pickedValue, bool) {
	kind, record := scriptUnit, "Unit"
	if target == targetStructure {
		kind, record = scriptStructure, "Structure"
	}
	for _, id := range e.hitsAt(p) {
		r := e.doc.Records[id]
		if r.Kind != record {
			continue
		}
		if target == targetGroup {
			if r.Group != 0 {
				return pickedValue{number: int64(r.Group)}, true
			}
			continue
		}
		for _, t := range e.doc.Script.Targets[kind] {
			if t.Record == id {
				return pickedValue{number: int64(t.Value)}, true
			}
		}
	}
	return pickedValue{}, false
}

// areaValueAt is the trigger area under p that the field of target can
// store; a square needs a point and radius area.
func (e *MapEditor) areaValueAt(p image.Point, target paramTarget) (pickedValue, bool) {
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	if !ok {
		return pickedValue{}, false
	}
	a, ok := e.areaUnder(image.Pt(x, y))
	if !ok {
		return pickedValue{}, false
	}
	if target == targetSquare {
		return pickedValue{area: a}, a.Shape == AreaSquare && a.Radius > 0
	}
	return pickedValue{area: a}, !a.Rect.Intersect(image.Rect(0, 0, e.doc.Width, e.doc.Height)).Empty()
}

// paramPicked writes the picked value into the field the pick armed, in the
// text the field stores when it is typed, and refreshes the field's choices
// so the list holds the value.
func (e *MapEditor) paramPicked(target paramTarget, v pickedValue) {
	o := &e.obj
	f := o.pickField
	o.pickField = -1
	switch target {
	case targetUnit, targetStructure, targetGroup:
		o.buf[f] = strconv.FormatInt(v.number, 10)
		o.fields[f].Choices = e.paramChoices(target, v.number)
	case targetBox:
		r := v.area.Rect.Intersect(image.Rect(0, 0, e.doc.Width, e.doc.Height))
		o.buf[f] = fmt.Sprintf("%d,%d,%d,%d", r.Min.X, r.Min.Y, r.Max.X-1, r.Max.Y-1)
	case targetSquare:
		o.buf[f] = fmt.Sprintf("%d,%d", v.area.Centre.X, v.area.Centre.Y)
		if radius := e.wizardRadiusField(f); radius >= 0 {
			o.buf[radius] = strconv.Itoa(v.area.Radius)
		}
	}
}

// paramChoices are the choices of a unit, structure or group field after the
// pick gave it value.
func (e *MapEditor) paramChoices(target paramTarget, value int64) []InspectionChoice {
	if e.obj.wiz != nil {
		if target == targetGroup {
			return e.wizardChoices(wpGroup, value)
		}
		return e.wizardChoices(wpUnit, value)
	}
	kind := scriptUnit
	switch target {
	case targetStructure:
		kind = scriptStructure
	case targetGroup:
		kind = scriptGroup
	}
	return e.parameterChoices(kind, value)
}

// paramAreaHover is the cells of the area under the pointer while an area
// pick is armed.
func (e *MapEditor) paramAreaHover() (image.Rectangle, bool) {
	if e.pick != pickParamArea || !e.hoverOK {
		return image.Rectangle{}, false
	}
	a, ok := e.areaUnder(e.hoverCell)
	return a.Rect, ok
}
