package game

import (
	"errors"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// The wizard's global-node refusal asks the fold's count: a plan with one
// condition and one action naming mod globals is refused on a plain map with
// the count the fold would report, and lifted on an extended map.
func TestWizardGlobalLimitIsTheFoldsCount(t *testing.T) {
	plan := ui.WizardTrigger{
		Pairs:   []ui.WizardPair{{Left: ui.WizardNode{Opcode: 0x4102}}},
		Actions: []ui.WizardNode{{Opcode: 0x4103}},
	}
	if got := mapedit.CountGlobalNodes(planScript(plan)); got != 2 {
		t.Fatalf("the fold's count of the plan is %d, want 2", got)
	}
	var limit *mapedit.ScriptLimitError
	if err := wizardRoom(plan, scriptprofile.First, false); !errors.As(err, &limit) || limit.Limit != mapedit.LimitGlobal || limit.Got != 2 {
		t.Fatalf("a plain map answered %v", err)
	}
	if err := wizardRoom(plan, scriptprofile.Extended, true); err != nil {
		t.Fatalf("an extended map refused the plan: %v", err)
	}
}

// The wizard's dialogue refusal asks the fold's count: an interactive
// message action is refused on a plain map, a plain message is not.
func TestWizardDialogueLimitIsTheFoldsCount(t *testing.T) {
	dialogue := ui.WizardNode{Opcode: 2, Label: "talk"}
	dialogue.Types[0], dialogue.Types[1] = 1, 1
	dialogue.Values[1] = 1
	plain := dialogue
	plain.Values[1] = 0
	var limit *mapedit.ScriptLimitError
	err := wizardRoom(ui.WizardTrigger{Actions: []ui.WizardNode{dialogue}}, scriptprofile.First, false)
	if !errors.As(err, &limit) || limit.Limit != mapedit.LimitDialogue || limit.Got != 1 {
		t.Fatalf("a dialogue action answered %v", err)
	}
	if err := wizardRoom(ui.WizardTrigger{Actions: []ui.WizardNode{plain}}, scriptprofile.First, false); err != nil {
		t.Fatalf("a plain message was refused: %v", err)
	}
	if err := wizardRoom(ui.WizardTrigger{Actions: []ui.WizardNode{dialogue}}, scriptprofile.Extended, true); err != nil {
		t.Fatalf("an extended map refused the dialogue: %v", err)
	}
}
