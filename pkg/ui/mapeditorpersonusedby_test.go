package ui

import (
	"image"
	"slices"
	"testing"
)

// The person window with a Used by section holds the four layout properties
// at the three sizes.
func TestEditorPersonWindowUsedByHoldsTheLayoutProperties(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1920, 1080}} {
		e, host := popupEditor(t)
		_ = e.Open("popups")
		e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
		id := len(e.doc.Records)
		e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: "Action", Index: 0})
		e.doc.Script = &InspectionScriptModel{
			Actions: []InspectionScriptNode{{Record: id, Label: "Give the potion"}},
			Targets: map[uint32][]InspectionScriptTarget{scriptUnit: {{Record: 0, Uses: []InspectionScriptUse{{Node: id}}}}},
		}
		e.Layout(size.X, size.Y)
		e.Select(0, false)
		e.objectSpec()
		if !e.nativeUnitWindow() {
			t.Fatalf("%v: the person window is not open", size)
		}
		e.obj.section = 0
		s, _, _ := e.objectSpec()
		checkWindowProperties(t, e, s, size)
	}
}

// The person window lists the triggers that use its unit, on each of its
// sections, as the unit popup does. Fails when the person window drops the
// Used by links, which leaves a map maker no way from a person to the
// trigger that names it.
func TestEditorPersonWindowListsItsUsedByLinks(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
	id := len(e.doc.Records)
	e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: "Action", Index: 0})
	e.doc.Script = &InspectionScriptModel{
		Actions: []InspectionScriptNode{{Record: id, Label: "Give the potion"}},
		Targets: map[uint32][]InspectionScriptTarget{scriptUnit: {{Record: 0, Uses: []InspectionScriptUse{{Node: id}}}}},
	}
	e.Layout(800, 600)
	e.Select(0, false)
	e.objectSpec()
	if !e.nativeUnitWindow() {
		t.Fatal("the person window is not open")
	}
	for section := range 4 {
		e.obj.section = section
		_, links, _ := e.objectSpec()
		if !slices.Contains(links, -id-1) {
			t.Errorf("section %d of the person window has no Used by link to the action, links %v", section, links)
		}
	}
}
