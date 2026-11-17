package ui

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/words"
)

// cornerHost is the card host with the unit's facing pad and a figure of a
// person's size.
type cornerHost struct {
	*cardHost
	choices []InspectionChoice
	pad     []image.Point
}

func (h *cornerHost) facingField(value int64) InspectionField {
	return InspectionField{Key: "facing", Label: "Facing", Section: "identity", Value: value, Choices: h.choices, ChoicePad: h.pad}
}

func (h *cornerHost) Fields(kind RecordKind, index int) []InspectionField {
	fields := h.cardHost.Fields(kind, index)
	if kind == KindUnit {
		fields = append(fields, h.facingField(0))
	}
	return fields
}

// UnitSettings hands the facing back as the draft set it, as the game's
// host does.
func (h *cornerHost) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	entity, fields, _, err := h.cardHost.UnitSettings(index, values)
	facing := int64(0)
	for _, v := range values {
		if v.Key == "facing" {
			facing = v.Value
		}
	}
	return entity, append(fields, h.facingField(facing)), &InspectionUnitPreview{Doll: image.NewRGBA(image.Rect(0, 0, 64, 112))}, err
}

func cornerEditor(t *testing.T, lang string) *MapEditor {
	t.Helper()
	e := wideEditor(t, lang, 0, "plain", map[string]int64{"Unit/identity": 300})
	h := &cornerHost{cardHost: &cardHost{wideHost: e.doc.Edits.(*wideHost)}, pad: []image.Point{{0, 0}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}}
	w := words.For(words.Language(lang))
	for n := range 9 {
		key := fmt.Sprintf("editor.unit.facing.%d", n)
		if n == 0 {
			key = "editor.unit.facing.random"
		}
		h.choices = append(h.choices, InspectionChoice{Value: int64(n), Label: w.Text(key)})
	}
	e.open = func(string) (*InspectionDocument, error) {
		d := wideDocument(h.wideHost)
		d.Edits = h
		return d, nil
	}
	if err := e.Open("views"); err != nil {
		t.Fatal(err)
	}
	return e
}
