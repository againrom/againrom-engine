package game

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// authorUndying makes, in the node popups of trigger "Near", its first pair's
// left condition a health-below -25 on unit 10 and its first action a set
// undying on unit 10, applying each.
func authorUndying(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	trigDetails(t, e)
	r, ok := e.RowButtonRect(2, 0, false)
	trigClick(t, e, r, ok, "Edit pair 1 left")
	trigChoose(t, e, 1, "Health below (Againrom)")
	r, ok = e.FieldRect(3)
	trigClick(t, e, r, ok, "health")
	for range 4 {
		_ = e.Key("backspace")
	}
	e.TextInput("-25")
	r, ok = e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply condition")
	r, ok = e.PopupButtonRect(2)
	trigClick(t, e, r, ok, "Back")
	r, ok = e.RowButtonRect(11, 0, false)
	trigClick(t, e, r, ok, "Edit action 1")
	trigChoose(t, e, 1, "Set undying (Againrom)")
	trigChoose(t, e, 2, "10")
	r, ok = e.FieldRect(3)
	trigClick(t, e, r, ok, "undying")
	for range 4 {
		_ = e.Key("backspace")
	}
	e.TextInput("1")
	r, ok = e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply action")
}

// The editor offers both Againrom nodes with their parameters, writes them,
// marks the map Againrom-only, and a saved copy reopens with the same nodes,
// the same mark and the node popup's note.
func TestEditorAuthorsTheUndyingNodesAndMarksTheMap(t *testing.T) {
	b := triggerMap(t)
	e, path := openGoalMap(t, b)
	if e.Document().AgainromOnly || strings.Contains(e.StatusLine(), "Againrom-only map") {
		t.Fatal("a map without Againrom nodes is marked")
	}
	authorUndying(t, e)
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := m.Script()
	c, a := sc.Conditions[0], sc.Actions[0]
	if c.Opcode != mapload.AgainromScriptOperation(true) || c.Type[0] != mapload.ScriptUnit || c.Value[0] != 10 ||
		c.Type[1] != mapload.ScriptInt || int32(c.Value[1]) != -25 {
		t.Fatalf("the health-below node reads %+v\n%s", c, trigLines(e))
	}
	if a.Opcode != mapload.AgainromScriptOperation(false) || a.Type[0] != mapload.ScriptUnit || a.Value[0] != 10 ||
		a.Type[1] != mapload.ScriptInt || a.Value[1] != 1 {
		t.Fatalf("the undying node reads %+v\n%s", a, trigLines(e))
	}
	if !e.Document().AgainromOnly || !strings.Contains(e.StatusLine(), "Againrom-only map") {
		t.Fatalf("the authored map is not marked: %q", e.StatusLine())
	}
	out := path + ".out.alm"
	if _, err := e.Document().Edits.SaveAs(out); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(out)
	if err != nil || !bytes.Equal(saved, e.Document().SourceBytes) {
		t.Fatalf("the saved copy differs: %v", err)
	}
	re, _ := openGoalMap(t, saved)
	if !re.Document().AgainromOnly || !strings.Contains(re.StatusLine(), "Againrom-only map") {
		t.Fatalf("the reopened map is not marked: %q", re.StatusLine())
	}
	re.Select(trigRecord(t, re, "Trigger", 0), true)
	trigDetails(t, re)
	lines := trigLines(re)
	for _, want := range []string{"Health below (Againrom)", "Set undying (Againrom)", "Unit 10", "-25"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("the reopened trigger lacks %q:\n%s", want, lines)
		}
	}
	r, ok := re.RowButtonRect(2, 0, false)
	trigClick(t, re, r, ok, "Edit pair 1 left")
	if lines := trigLines(re); !strings.Contains(lines, "Againrom operation: the original game does nothing with it.") ||
		!strings.Contains(lines, "Health below -25") {
		t.Fatalf("the reopened node popup:\n%s", lines)
	}
}

// The editor refuses a health-below X outside the signed 16-bit health field
// and an undying state other than 0 or 1, and writes nothing.
func TestEditorRefusesUndyingValuesOutsideTheirFields(t *testing.T) {
	e, _ := openGoalMap(t, triggerMap(t))
	authorUndying(t, e)
	before := e.Document().SourceBytes
	set := e.Document().Edits.(ui.InspectionFieldSet)
	for _, tc := range []struct {
		kind  string
		value int64
	}{{"Condition", 32768}, {"Condition", -32769}, {"Action", 2}, {"Action", -1}} {
		if _, err := set.SetFields(tc.kind, 0, []ui.InspectionFieldValue{{Key: "value1", Value: tc.value}}); err == nil {
			t.Fatalf("%s value %d accepted", tc.kind, tc.value)
		}
		if !bytes.Equal(e.Document().SourceBytes, before) {
			t.Fatalf("%s value %d wrote bytes", tc.kind, tc.value)
		}
	}
	// The popup's order: types, values, then the operation. The value is
	// checked again on the node the whole set leaves.
	_, err := set.SetFields("Condition", 2, []ui.InspectionFieldValue{
		{Key: "type0", Value: 4}, {Key: "type1", Value: 1}, {Key: "type2"},
		{Key: "value0", Value: 10}, {Key: "value1", Value: 40000}, {Key: "opcode", Value: int64(mapload.AgainromScriptOperation(true))}})
	if err == nil || !strings.Contains(err.Error(), "health below 40000") || !bytes.Equal(e.Document().SourceBytes, before) {
		t.Fatalf("an operation change to health-below 40000: %v", err)
	}
	if _, err := set.SetFields("Condition", 0, []ui.InspectionFieldValue{{Key: "value1", Value: -32768}}); err != nil {
		t.Fatalf("the field floor refused: %v", err)
	}
}
