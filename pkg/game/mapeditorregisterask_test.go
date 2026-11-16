package game

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// registerMap is an original map whose variable 50 sits on the register the
// next check takes, with 50 checks in the script; trigger "T" tests checks
// 1 and 2.
func registerMap(t *testing.T) []byte {
	t.Helper()
	conditions := make([]scriptNode, 50)
	for i := range conditions {
		conditions[i] = scriptNode{label: "Constant", opcode: 0x10002, id: uint32(i + 1), tags: []uint32{kInt}, values: []uint32{uint32(i % 2)}}
	}
	actions := []scriptNode{
		{label: "set", opcode: 3, id: 1, tags: []uint32{kInt, kInt}, values: []uint32{50, 1}},
		{label: "win", opcode: 4, id: 2},
	}
	return synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions,
		[]goalTrigger{{name: "T", left: 1, right: 2, actions: []uint32{1}}})})
}

// A refusal for a check register a variable uses asks to extend the map: No
// changes nothing, and once the map is extended the same edit lands.
func TestCheckRegisterOfAVariableAsksToExtendTheMap(t *testing.T) {
	newCheck := ui.WizardNode{Opcode: 19, Types: [10]uint32{1}, Values: [10]uint32{3}}
	for _, c := range []struct {
		name string
		edit func(s *mapInspectionEdits) (*ui.InspectionDocument, error)
	}{
		{"trigger copy", func(s *mapInspectionEdits) (*ui.InspectionDocument, error) { return s.CopyTrigger(0) }},
		{"trigger wizard", func(s *mapInspectionEdits) (*ui.InspectionDocument, error) {
			return s.WriteTrigger(ui.WizardTrigger{Index: 0, Pairs: []ui.WizardPair{{Left: newCheck, Right: ui.WizardNode{ID: 2, Keep: true}}}, Actions: []ui.WizardNode{{ID: 1, Keep: true}}})
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e, _ := openGoalMap(t, registerMap(t))
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
			if d, err = c.edit(s); err != nil || !d.Script.Extended || len(d.Script.Conditions) <= 50 {
				t.Fatalf("on the extended map the edit gives %v", err)
			}
		})
	}
}

// endsWords reports whether the wrapped note lines end with want: the
// question's own text follows the refusal's.
func endsWords(lines []string, want string) bool {
	return strings.HasSuffix(strings.Join(strings.Fields(strings.Join(lines, " ")), " "), strings.Join(strings.Fields(want), " "))
}

// isFormNeeded reports whether err is the question to extend the map.
func isFormNeeded(err error) bool {
	var need *ui.ScriptFormNeeded
	return errors.As(err, &need)
}

// A bare limit of the editor library asks with no reason of its own; the
// game's own refusal brings its words.
func TestFormQuestionCarriesOnlyTheGamesOwnWords(t *testing.T) {
	e, _ := openGoalMap(t, registerMap(t))
	m := formHost(e).model
	limit := &mapedit.ScriptLimitError{Limit: mapedit.LimitPairs, Max: 3, Got: 4}
	var need *ui.ScriptFormNeeded
	if !errors.As(formNeeded(m, limit), &need) || need.Why != "" {
		t.Fatalf("a bare limit asks with %+v", need)
	}
	if !errors.As(formNeeded(m, lifted(limit, "own words")), &need) || need.Why != "own words" {
		t.Fatalf("a lifted limit asks with %+v", need)
	}
}

// A Russian editor's trigger-count question carries the refusal in Russian.
func TestFormQuestionForTheTriggerCountIsInTheEditorsLanguage(t *testing.T) {
	for _, c := range []struct {
		profile base.Profile
		sel     int
	}{{base.Profile{Language: "english"}, 0}, {base.Profile{Language: "russian"}, text.SelectorConverting}} {
		x := heroLabelInspector(t, c.profile, c.sel)
		e := openHeroLabelMap(t, x, formMap(t, 1000))
		_, err := formHost(e).AddScript(ui.KindTrigger)
		var need *ui.ScriptFormNeeded
		want := fmt.Sprintf(words.For(c.profile.Language).Text("editor.script.triggersfull"), 1000)
		if !errors.As(err, &need) || need.Why != want {
			t.Fatalf("%s: the question reads %v, want %q", c.profile.Language, err, want)
		}
	}
}
