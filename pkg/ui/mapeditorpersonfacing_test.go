package ui

import (
	"fmt"
	"strconv"
	"testing"
)

// personFacingHost gives the facing pad host's fields the person window's
// sections, so the unit opens in the person window.
type personFacingHost struct{ *facingPadHost }

func (h *personFacingHost) Fields(kind RecordKind, index int) []InspectionField {
	fields := h.facingPadHost.Fields(kind, index)
	if kind == KindUnit {
		for i := range fields {
			fields[i].Section = "identity"
		}
	}
	return fields
}

// The person window shows the facing pad, eight directions and random, in
// both views, and each of its nine buttons sets the unit's facing.
func TestEditorPersonWindowFacingPadInBothViews(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, plain := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/plain=%v", lang, plain), func(t *testing.T) {
				e, h := facingPadEditor(t, lang)
				p := &personFacingHost{h}
				h.self, e.doc.Edits = p, p
				e.props.plain = plain
				e.obj.doc = nil
				e.Select(0, false)
				e.objectSpec()
				if !e.nativeUnitWindow() || e.PlainView() != plain {
					t.Fatalf("person window %v, plain %v", e.nativeUnitWindow(), e.PlainView())
				}
				for n := range 9 {
					_, rr := facingPadRow(t, e, false)
					modesClick(e, center(rr.Buttons[n]))
					if e.obj.buf[0] != strconv.Itoa(n) {
						t.Fatalf("button %d chose %q", n, e.obj.buf[0])
					}
				}
			})
		}
	}
}
