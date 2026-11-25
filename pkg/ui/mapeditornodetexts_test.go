package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// nodeTexts are ten text fields as a host gives them for a second-game node.
func nodeTexts() []InspectionField {
	out := make([]InspectionField, 10)
	for i := range out {
		out[i] = InspectionField{Key: fmt.Sprintf("text%d", i), Label: fmt.Sprintf("Text %d", i+1), Text: true, TextValue: fmt.Sprintf("line %d", i+1)}
	}
	return out
}

// A second-game node's window lists its ten text fields after its
// parameters, each editable, and Apply sends a changed one as text under
// its own key (R2-ENGINE-1681).
func TestSecondGameNodeWindowShowsAndSendsItsTextFields(t *testing.T) {
	node := mission53Teleport
	node.Texts = nodeTexts()
	e := teleportEditor(t, node)
	for i, want := range node.Texts {
		f := objField(e, want.Key)
		if f < 0 || e.obj.fields[f].ReadOnly || !e.obj.fields[f].Text || e.obj.buf[f] != want.TextValue || e.obj.fields[f].Label != want.Label {
			t.Fatalf("text field %d: %+v", i, e.obj.fields)
		}
	}
	f := objField(e, "text3")
	e.obj.buf[f] = "changed"
	v, _ := e.scriptPopup()
	got, err := e.scriptValues(v)
	if err != nil {
		t.Fatal(err)
	}
	if w := []InspectionFieldValue{{Key: "text3", Text: true, TextValue: "changed"}}; !slices.Equal(got, w) {
		t.Fatalf("Apply sends %+v, want %+v", got, w)
	}
}

// On a node whose operation no claim decodes, the name and the text fields
// stay editable; the operation and its parameters are shown only.
func TestSecondGameUndecodedNodeKeepsItsTextsEditable(t *testing.T) {
	node := mission53Teleport
	node.Opcode = 9999
	node.Texts = nodeTexts()
	e := teleportEditor(t, node)
	for _, key := range []string{"name", "text0", "text9"} {
		if f := objField(e, key); f < 0 || e.obj.fields[f].ReadOnly {
			t.Fatalf("%s is not editable: %+v", key, e.obj.fields)
		}
	}
	for i, f := range e.obj.fields {
		if (f.Key == "opcode" || f.Key == "value0") && !f.ReadOnly {
			t.Fatalf("field %d %s is editable on an undecoded node", i, f.Key)
		}
	}
	e.obj.buf[objField(e, "name")] = "renamed"
	e.obj.buf[objField(e, "text9")] = "last"
	v, _ := e.scriptPopup()
	got, err := e.scriptValues(v)
	if err != nil {
		t.Fatal(err)
	}
	w := []InspectionFieldValue{{Key: "name", Text: true, TextValue: "renamed"}, {Key: "text9", Text: true, TextValue: "last"}}
	if len(got) != len(w) || !slices.Contains(got, w[0]) || !slices.Contains(got, w[1]) {
		t.Fatalf("Apply sends %+v, want %+v", got, w)
	}
}

// A text field's caption comes from the host in UTF-8 and reaches the row
// through the font encoder, like every other caption of the window.
func TestSecondGameTextCaptionsPassThroughTheFontEncoder(t *testing.T) {
	node := mission53Teleport
	node.Texts = nodeTexts()
	for i := range node.Texts {
		node.Texts[i].Label = fmt.Sprintf("Текст %d", i+1)
	}
	e := teleportEditor(t, node)
	e.hostText = func(s string) string { return "font:" + s }
	v, _ := e.scriptPopup()
	spec, _ := e.scriptSpec(v)
	seen := 0
	for _, r := range spec.Rows {
		if strings.Contains(r.Label, "Текст") && !strings.HasPrefix(r.Label, "font:") {
			t.Fatalf("caption not encoded: %q", r.Label)
		}
		if strings.HasPrefix(r.Label, "font:Текст") {
			seen++
		}
	}
	if seen != 10 {
		t.Fatalf("%d encoded text captions, want 10", seen)
	}
}
