package game

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// formConditions are the script form fixture's checks: reads of variables
// 50 and 60, and the constants 1 and 0.
var formConditions = []scriptNode{
	{label: "Var50", opcode: 19, id: 1, values: []uint32{50}, tags: []uint32{1}},
	{label: "One", opcode: 0x10002, id: 2, values: []uint32{1}, tags: []uint32{1}},
	{label: "Var60", opcode: 19, id: 3, values: []uint32{60}, tags: []uint32{1}},
	{label: "Zero", opcode: 0x10002, id: 4, values: []uint32{0}, tags: []uint32{1}},
}

// formActions are the fixture's actions: the drop cell, the setup (id 1)
// and five actions a trigger can run (11..15); Set73 is action 6.
var formActions = []scriptNode{
	{label: "Drop location", opcode: dropActionOpcode, id: 99, values: []uint32{16, 16}, tags: []uint32{5, 6}},
	{label: "Set50", opcode: 3, id: 1, values: []uint32{50, 1}, tags: []uint32{1, 1}},
	{label: "Set70", opcode: 3, id: 11, values: []uint32{70, 11}, tags: []uint32{1, 1}},
	{label: "Inc71", opcode: 8, id: 12, values: []uint32{71}, tags: []uint32{1}},
	{label: "Inc71 again", opcode: 8, id: 13, values: []uint32{71}, tags: []uint32{1}},
	{label: "Set72", opcode: 3, id: 14, values: []uint32{72, 5}, tags: []uint32{1, 1}},
	{label: "Set73", opcode: 3, id: 15, values: []uint32{73, 1}, tags: []uint32{1, 1}},
}

// formMap is an original 40x40 map of the extended fixture's unit and n
// triggers, n at least 2. Trigger 0, "Setup", sets variable 50 to 1;
// trigger 1, "Full", holds three pairs (variable 50 is 1, then 1 is 1
// twice) and four actions; the rest never hold (1 is 0).
func formMap(t *testing.T, n int) []byte {
	t.Helper()
	triggers := []goalTrigger{
		{name: "Setup", left: 2, right: 2, actions: []uint32{1}},
		{name: "Full", left: 1, right: 2, actions: []uint32{11, 12, 13, 14}, raw: map[int]uint32{0x88: 2, 0x8c: 2, 0x90: 2, 0x94: 2}},
	}
	for len(triggers) < n {
		triggers = append(triggers, goalTrigger{name: "Idle", left: 2, right: 4})
	}
	return synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: extendedUnits, Type7Payload: goalScript(formActions, formConditions, triggers)})
}

// formHost is the editor host of e.
func formHost(e *ui.MapEditor) *mapInspectionEdits { return e.Document().Edits.(*mapInspectionEdits) }

func TestScriptRoundTripPreservesTriggerName(t *testing.T) {
	for _, name := range []string{"Roundtrip", "Вход"} {
		t.Run(name, func(t *testing.T) {
			e, _ := openGoalMap(t, formMap(t, 2))
			s := formHost(e)
			if _, err := s.SetFields(ui.KindTrigger, 1, []ui.InspectionFieldValue{{Key: "name", Text: true, TextValue: name}}); err != nil {
				t.Fatal(err)
			}
			original, err := s.model.Map()
			if err != nil {
				t.Fatal(err)
			}
			originalScript, err := original.Script()
			if err != nil || originalScript.Triggers[1].Name != name {
				t.Fatalf("original name: %v, %+v", err, originalScript.Triggers)
			}
			if _, err := s.ExtendScript(); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SetFields(ui.KindTrigger, 1, []ui.InspectionFieldValue{{Key: "left3", Value: 3}, {Key: "right3", Value: 4}, {Key: "cmp3", Value: 0}}); err != nil {
				t.Fatal(err)
			}
			extendedPath := filepath.Join(t.TempDir(), "extended.almx")
			if _, err := s.SaveAs(extendedPath); err != nil {
				t.Fatal(err)
			}
			d, err := s.inspector.Open(extendedPath)
			if err != nil {
				t.Fatal(err)
			}
			s = d.Edits.(*mapInspectionEdits)
			extended, err := s.model.Map()
			if err != nil {
				t.Fatal(err)
			}
			extendedScript, err := extended.Script()
			if err != nil || extendedScript.Triggers[1].Name != name || extendedScript.Triggers[1].Pairs() != 4 {
				t.Fatalf("extended model: %v, %+v", err, extendedScript.Triggers)
			}
			program, _, err := mapload.CompileScript(extended, mapload.ScriptRefs{})
			if err != nil || !program.Extended() {
				t.Fatal("extended mapload", err)
			}
			if _, err := s.SetFields(ui.KindTrigger, 1, []ui.InspectionFieldValue{{Key: "left3", Value: 0}, {Key: "right3", Value: 0}}); err != nil {
				t.Fatal(err)
			}
			beforeFold, err := s.model.Map()
			if err != nil {
				t.Fatal(err)
			}
			model, err := beforeFold.Script()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.FoldScript(); err != nil {
				t.Fatal(err)
			}
			plainPath := filepath.Join(t.TempDir(), "plain.alm")
			if _, err := s.SaveAs(plainPath); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(plainPath)
			if err != nil {
				t.Fatal(err)
			}
			folded, err := mapedit.New(b)
			if err != nil {
				t.Fatal(err)
			}
			foldedMap, err := folded.Map()
			if err != nil {
				t.Fatal(err)
			}
			foldedScript, err := foldedMap.Script()
			if err != nil {
				t.Fatal(err)
			}
			for i, tr := range model.Triggers {
				want, err := mapedit.NewScriptTrigger(tr)
				if err != nil {
					t.Fatal(err)
				}
				got, err := folded.Record(mapedit.KindTrigger, i)
				if err != nil || !bytes.Equal(got, want) {
					t.Errorf("fold writer mismatch %d: %v", i, err)
				}
			}
			program, _, err = mapload.CompileScript(foldedMap, mapload.ScriptRefs{})
			if err != nil || program.Extended() {
				t.Fatal("folded mapload", err)
			}
			if got := foldedScript.Triggers[1].Name; got != name {
				t.Errorf("fold changed trigger name: %q -> %q", name, got)
			}
		})
	}
}

func TestConditionlessFoldUsesTriggerWriter(t *testing.T) {
	e, _ := openGoalMap(t, formMap(t, 2))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	var values []ui.InspectionFieldValue
	for i := range 3 {
		values = append(values, ui.InspectionFieldValue{Key: fmt.Sprintf("left%d", i)}, ui.InspectionFieldValue{Key: fmt.Sprintf("right%d", i)})
	}
	if _, err := s.SetFields(ui.KindTrigger, 1, values); err != nil {
		t.Fatal(err)
	}
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	script, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	trigger := script.Triggers[1]
	if trigger.Pairs() != 0 || len(trigger.Acts) != 4 {
		t.Fatalf("conditionless model: %+v", trigger)
	}
	if _, err := s.FoldScript(); err != nil {
		t.Fatal(err)
	}
	got, err := s.model.Record(mapedit.KindTrigger, 1)
	if err != nil {
		t.Fatal(err)
	}
	want, err := mapedit.NewScriptTrigger(trigger)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("conditionless fold differs from writer: got %x, want %x", got, want)
	}
}

// sameWords reports whether the wrapped note lines read as want.
func sameWords(lines []string, want string) bool {
	return strings.Join(strings.Fields(strings.Join(lines, " ")), " ") == strings.Join(strings.Fields(want), " ")
}

// Each edit an original trigger record cannot hold comes back from the host
// as the question to extend the map and changes nothing; once the map is
// extended the same call lands.
func TestOriginalMapEditsPastTheRecordAskToExtendTheMap(t *testing.T) {
	slots := func(kv ...any) []ui.InspectionFieldValue {
		var out []ui.InspectionFieldValue
		for i := 0; i < len(kv); i += 2 {
			out = append(out, ui.InspectionFieldValue{Key: kv[i].(string), Value: int64(kv[i+1].(int))})
		}
		return out
	}
	for _, c := range []struct {
		name     string
		triggers int
		edit     func(s *mapInspectionEdits) (*ui.InspectionDocument, error)
		landed   func(sc *ui.InspectionScriptModel) bool
	}{
		{"fourth pair", 2, func(s *mapInspectionEdits) (*ui.InspectionDocument, error) {
			return s.SetFields(ui.KindTrigger, 1, slots("left3", 3, "cmp3", 0, "right3", 4))
		}, func(sc *ui.InspectionScriptModel) bool {
			tr := sc.Triggers[1]
			return slices.Equal(tr.Left, []uint32{1, 2, 2, 3}) && slices.Equal(tr.Right, []uint32{2, 2, 2, 4})
		}},
		{"OR group", 2, func(s *mapInspectionEdits) (*ui.InspectionDocument, error) {
			return s.SetFields(ui.KindTrigger, 1, slots("group2", 1))
		}, func(sc *ui.InspectionScriptModel) bool { return slices.Equal(sc.Triggers[1].Group, []uint32{0, 0, 1}) }},
		{"fifth action", 2, func(s *mapInspectionEdits) (*ui.InspectionDocument, error) {
			return s.SetFields(ui.KindTrigger, 1, slots("action4", 15))
		}, func(sc *ui.InspectionScriptModel) bool {
			return slices.Equal(sc.Triggers[1].Acts, []uint32{11, 12, 13, 14, 15})
		}},
		{"variable past the registers", 2, func(s *mapInspectionEdits) (*ui.InspectionDocument, error) {
			return s.SetField(ui.KindAction, 6, "value0", 150)
		}, func(sc *ui.InspectionScriptModel) bool { return sc.Variables == 151 }},
		{"trigger past 1000", 1000, func(s *mapInspectionEdits) (*ui.InspectionDocument, error) {
			return s.AddScript(ui.KindTrigger)
		}, func(sc *ui.InspectionScriptModel) bool { return len(sc.Triggers) == 1001 }},
	} {
		t.Run(c.name, func(t *testing.T) {
			e, _ := openGoalMap(t, formMap(t, c.triggers))
			s := formHost(e)
			before := bytes.Clone(s.model.Bytes())
			d, err := c.edit(s)
			var need *ui.ScriptFormNeeded
			if d != nil || !errors.As(err, &need) || need.Fold {
				t.Fatalf("the edit did not ask to extend the map: %v", err)
			}
			if !bytes.Equal(s.model.Bytes(), before) || s.CanUndo() {
				t.Fatal("the edit that asked changed the map")
			}
			if d, err = s.ExtendScript(); err != nil || !d.Script.Extended {
				t.Fatal("ExtendScript", err)
			}
			if d, err = c.edit(s); err != nil || !d.Script.Extended || !c.landed(d.Script) {
				t.Fatalf("on the extended map the edit gives %v", err)
			}
		})
	}
}

// A fourth pair added in the trigger window of an original map asks, in
// the editor's language, whether to extend the map. Cancel changes nothing;
// Yes extends it and applies the pair. The next edits past the original
// record (an OR group, a fifth action, a variable past 100) ask nothing.
// Save writes an .almx beside the .alm, which stays as it was; the .almx
// reopens as the saved model and compiles to the extended program.
func TestTriggerWindowExtendsTheMapOnceAndSavesItAsAlmx(t *testing.T) {
	b := formMap(t, 2)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			e, x, w := scriptAddEditor(t, l, b)
			s := formHost(e)
			own, ok := s.OwnFile()
			if !ok {
				t.Fatal("the map has no file of its own")
			}
			before := bytes.Clone(s.model.Bytes())
			e.Select(trigRecord(t, e, "Trigger", 1), true)
			trigDetails(t, e)
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: -1, K: 0, Flag: false})
			trigClick(t, e, r, ok, "Add condition pair")
			trigChoose(t, e, 11, "#3 ")
			trigChoose(t, e, 13, "#4 ")
			apply := func() {
				r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
				trigClick(t, e, r, ok, "Apply")
			}
			apply()
			asked := func() {
				t.Helper()
				if !slices.Contains(e.OpenWindows(), "prompt") || !sameWords(e.PromptNotes(), w("editor.scriptform.text")) {
					t.Fatalf("no question to extend the map: %q %q", e.OpenWindows(), e.PromptNotes())
				}
			}
			asked()
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 1})
			trigClick(t, e, r, ok, "Cancel")
			if slices.Contains(e.OpenWindows(), "prompt") || !bytes.Equal(s.model.Bytes(), before) || s.CanUndo() || e.Document().Script.Extended {
				t.Fatalf("Cancel changed the map: %q", e.OpenWindows())
			}
			if !slices.Contains(e.OpenWindows(), "object") {
				t.Fatal("Cancel closed the trigger window")
			}
			apply()
			asked()
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 0})
			trigClick(t, e, r, ok, "Extend")
			sc := e.Document().Script
			if !sc.Extended || !slices.Equal(sc.Triggers[1].Left, []uint32{1, 2, 2, 3}) || !slices.Equal(sc.Triggers[1].Right, []uint32{2, 2, 2, 4}) {
				t.Fatalf("Yes left %+v:\n%s", sc.Triggers[1], trigLines(e))
			}

			// An OR group and a fifth action, asked nothing.
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: -1, K: 1, Flag: false})
			if !ok {
				t.Fatalf("no Add OR group (wizard %v):\n%s", e.WizardShown(), trigLines(e))
			}
			trigClick(t, e, r, ok, "Add OR group")
			trigChoose(t, e, 14, "#1 ")
			trigChoose(t, e, 16, "#4 ")
			e.ScrollDetails(100) // the action rows, past the window's height
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: -1, K: 0, Flag: true})
			trigClick(t, e, r, ok, "Add action")
			e.ScrollDetails(100)
			trigChoose(t, e, 21, "#15 ")
			apply()
			if slices.Contains(e.OpenWindows(), "prompt") {
				t.Fatalf("the extended map asked again: %q", e.PromptNotes())
			}
			tr := e.Document().Script.Triggers[1]
			if !slices.Equal(tr.Left, []uint32{1, 2, 2, 3, 1}) || !slices.Equal(tr.Group, []uint32{0, 0, 0, 0, 1}) || !slices.Equal(tr.Acts, []uint32{11, 12, 13, 14, 15}) {
				t.Fatalf("the trigger reads %+v:\n%s", tr, trigLines(e))
			}

			// A variable past 100.
			d, err := s.SetField(ui.KindAction, 6, "value0", 150)
			if err != nil || d.Script.Variables != 151 {
				t.Fatal("variable 150", err)
			}
			saved, err := s.SaveInPlace()
			if err != nil {
				t.Fatal(err)
			}
			if want := strings.TrimSuffix(own, filepath.Ext(own)) + ".almx"; saved != want {
				t.Fatalf("saved %s, want %s", saved, want)
			}
			if got, err := os.ReadFile(own); err != nil || !bytes.Equal(got, b) {
				t.Fatal("the .alm changed", err)
			}
			almx, err := os.ReadFile(saved)
			if err != nil {
				t.Fatal(err)
			}
			again, err := x.Open(saved)
			if err != nil {
				t.Fatal(err)
			}
			if !again.Script.Extended || again.Script.Variables != d.Script.Variables || !reflect.DeepEqual(again.Script.Triggers, d.Script.Triggers) {
				t.Fatalf("the reopened .almx reads %+v, saved %+v", again.Script.Triggers[1], d.Script.Triggers[1])
			}
			if l.name != "en" {
				return
			}
			world := openExtendedMission(t, extendedScriptFront(t, almx), 160)
			p := world.Script()
			if p == nil || !p.Extended() || p.Variables() != 151 || len(p.Triggers()) != 2 {
				t.Fatal("the mission does not run the extended program")
			}
			if full := p.Triggers()[1]; len(full.Conditions) != 5 || len(full.Actions) != 5 || full.Conditions[3].Group != 0 || full.Conditions[4].Group != 1 {
				t.Fatalf("Full compiled to %+v", full)
			}
			for v, n := range map[int32]int32{50: 1, 70: 11, 71: 2, 72: 5, 150: 1} {
				if got := world.ScriptRegister(v); got != n {
					t.Errorf("variable %d is %d, want %d", v, got, n)
				}
			}
		})
	}
}

// An extended map whose script does not fit the original records is never
// written under an .alm name. Edited back to fit, Save As to an .alm asks
// to fold the script, and Cancel writes nothing; Save over its own .alm
// asks the same, and Yes writes trigger records equal to NewScriptTrigger's
// of the extended triggers, with no extended record left.
func TestAnExtendedMapSavesAsAlmOnlyWhenItFitsAndTheQuestionAgrees(t *testing.T) {
	b := formMap(t, 2)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			e, _, w := scriptAddEditor(t, l, b)
			s := formHost(e)
			own, _ := s.OwnFile()
			dir := t.TempDir()
			e.Select(trigRecord(t, e, "Trigger", 1), true)
			trigDetails(t, e)
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: -1, K: 0, Flag: true})
			trigClick(t, e, r, ok, "Add action")
			trigChoose(t, e, 15, "#15 ")
			apply := func() {
				r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
				trigClick(t, e, r, ok, "Apply")
			}
			apply()
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 0})
			trigClick(t, e, r, ok, "Extend")
			if sc := e.Document().Script; !sc.Extended || len(sc.Triggers[1].Acts) != 5 {
				t.Fatalf("the fifth action did not extend the map:\n%s", trigLines(e))
			}
			plain := filepath.Join(dir, "plain.alm")
			if _, err := s.SaveAs(plain); err == nil || errors.As(err, new(*ui.ScriptFormNeeded)) {
				t.Fatalf("an extended map that does not fit was saved as .alm: %v", err)
			}
			if _, err := os.Stat(plain); !os.IsNotExist(err) {
				t.Fatal("the refused save wrote a file", err)
			}

			// The fifth action removed: the script fits again.
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: 15, K: 1, Flag: false})
			if !ok {
				t.Fatalf("no Remove action 5 (wizard %v):\n%s", e.WizardShown(), trigLines(e))
			}
			trigClick(t, e, r, ok, "Remove action 5")
			apply()
			if tr := e.Document().Script.Triggers[1]; !e.Document().Script.Extended || len(tr.Acts) != 4 {
				t.Fatalf("the trigger reads %+v:\n%s", tr, trigLines(e))
			}
			m, err := alm.Open(s.model.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			ext, err := m.Script()
			if err != nil || !ext.Extended {
				t.Fatal("the model is not extended", err)
			}
			closeAll := func() {
				for i := 0; i < 6 && e.OpenWindows() != nil; i++ {
					_ = e.Key("escape")
				}
			}
			closeAll()
			n, ok := e.MenuRow("file", "saveto")
			if !ok {
				t.Fatal("the Map menu has no Save As row")
			}
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: "file"})
			trigClick(t, e, r, ok, "the Map entry")
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlMenuRow, N: n})
			trigClick(t, e, r, ok, "the Save As row")
			e.TextInput(plain)
			_ = e.Key("enter")
			if !sameWords(e.PromptNotes(), w("editor.scriptform.foldtext")) {
				t.Fatalf("Save As to .alm asked %q", e.PromptNotes())
			}
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 1})
			trigClick(t, e, r, ok, "Cancel")
			if _, err := os.Stat(plain); !os.IsNotExist(err) || !e.Document().Script.Extended {
				t.Fatal("Cancel wrote or folded", err)
			}

			// Save over its own .alm: the same question; Yes folds and writes.
			if _, err := s.SaveInPlace(); !errors.As(err, new(*ui.ScriptFormNeeded)) {
				t.Fatalf("Save over the .alm: %v", err)
			}
			closeAll()
			if err := e.Key("saveas"); !slices.Contains(e.OpenWindows(), "prompt") || !sameWords(e.PromptNotes(), w("editor.scriptform.foldtext")) {
				t.Fatalf("Save over the .alm asked %q (%v)", e.PromptNotes(), err)
			}
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 0})
			trigClick(t, e, r, ok, "Save as .alm")
			got, err := os.ReadFile(own)
			if err != nil {
				t.Fatal(err)
			}
			folded, err := alm.Open(got)
			if err != nil {
				t.Fatal(err)
			}
			if sc, err := folded.Script(); err != nil || sc.Extended || len(sc.Triggers) != len(ext.Triggers) {
				t.Fatalf("the saved .alm's script: %v", err)
			}
			saved, err := mapedit.New(got)
			if err != nil || saved.Extended() {
				t.Fatal("the saved .alm carries the extended record", err)
			}
			for i, tr := range ext.Triggers {
				want, err := mapedit.NewScriptTrigger(tr)
				if err != nil {
					t.Fatal(err)
				}
				if rec, err := saved.Record(mapedit.KindTrigger, i); err != nil || !bytes.Equal(rec, want) {
					t.Fatalf("trigger %d's record differs from NewScriptTrigger's: %v", i, err)
				}
			}
		})
	}
}

// The trigger wizard's OR group on an original map opens an "Or when"
// group and, applied, asks the same question; Yes writes the condition in
// group 1.
func TestTriggerWizardOrGroupAsksToExtendTheMap(t *testing.T) {
	e, _, w := scriptAddEditor(t, scriptAddLangs[0], formMap(t, 2))
	e.Select(trigRecord(t, e, "Trigger", 1), true)
	if !e.WizardShown() {
		t.Fatalf("the trigger opens no wizard:\n%s", trigLines(e))
	}
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: -1, K: 1, Flag: false})
	trigClick(t, e, r, ok, "Add OR group")
	if !slices.Contains(e.PopupLines(), w("editor.wizard.orwhen")) {
		t.Fatalf("no Or when group:\n%s", trigLines(e))
	}
	trigChoose(t, e, wizField(t, e, "c3.item"), w("editor.wizard.c.always"))
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
	trigClick(t, e, r, ok, "Apply")
	if !slices.Contains(e.OpenWindows(), "prompt") || !sameWords(e.PromptNotes(), w("editor.scriptform.text")) {
		t.Fatalf("no question to extend the map: %q\n%s", e.PromptNotes(), trigLines(e))
	}
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 0})
	trigClick(t, e, r, ok, "Extend")
	sc := e.Document().Script
	if tr := sc.Triggers[1]; !sc.Extended || len(tr.Left) != 4 || !slices.Equal(tr.Group, []uint32{0, 0, 0, 1}) {
		t.Fatalf("the wizard wrote %+v:\n%s", tr, trigLines(e))
	}
}

// An original map edited within its records and saved is the map an
// unchanged editor writes: no extended record, its bytes the base's.
func TestAnOriginalMapSavesWithoutTheExtendedRecord(t *testing.T) {
	b := formMap(t, 2)
	e, _ := openGoalMap(t, b)
	s := formHost(e)
	out := filepath.Join(t.TempDir(), "same.alm")
	if _, err := s.SaveAs(out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil || !bytes.Equal(got, b) {
		t.Fatal("the unedited original map saved other bytes", err)
	}
}
