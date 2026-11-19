package game

import (
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// A sex word that sorts the cells is not an extra condition; tips stays.
func TestCellsKeepTheTagAttributesButNotTheSexTheyAreSortedBy(t *testing.T) {
	heroes := [4]EventAudience{}
	for h, mage := range []bool{true, false, true, false} {
		female := h < 2
		heroes[h] = EventAudience{HeroFemale: female, HeroMage: mage, Speaker: func(int) (bool, bool, bool) { return female, mage, true }}
	}
	d := decodeMessageMarkup("<part=1,npc=21,female,tips=2>\nShe<part=1,npc=21,male,tips=2>\nHe")
	cells := messageCells(d, heroes)
	if len(cells) != 2 {
		t.Fatalf("%d cells, want 2: %+v", len(cells), cells)
	}
	for i, c := range cells {
		if c.Other != "tips=2" {
			t.Errorf("cell %d holds %q beside its sex, want tips=2", i, c.Other)
		}
	}
}

// The refusal names the attribute; one press on the row's x lets Apply write.
func TestDialogueRefusalNamesTheTagAttributeAndOneActionClearsIt(t *testing.T) {
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	startDialogueRow(t, e)
	dlgType(t, e, ui.DialogueLineText, "First")
	dlgApply(t, e)
	page := uint32(0)
	for n := uint32(1); n < 8 && page == 0; n++ {
		if parts := s.MessageParts(n); len(parts) == 1 && parts[0].Body == "First" {
			page = n
		}
	}
	if page == 0 {
		t.Fatal("no page holds the first write")
	}
	workspacePage(t, e, messageFilename(page), "<part=1,iamfemale>\nShe<part=1,iammale,tips=2>\nHe")
	dlgChoose(t, e, "d.after", "Reply choice")
	dlgType(t, e, "d.rt", "Hi")
	dlgApply(t, e)
	lines := trigLines(e)
	if !strings.Contains(lines, `the tag holds "tips=2"`) || !strings.Contains(lines, "Also in the tag") {
		t.Fatalf("the refusal does not name the attribute:\n%s", lines)
	}
	if len(s.DialogueNames()) != 0 {
		t.Fatalf("a refused Apply wrote a file")
	}
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: 3, K: 0, Flag: false})
	trigClick(t, e, r, ok, "Also in the tag cross")
	dlgApply(t, e)
	if len(s.DialogueNames()) != 1 {
		t.Fatalf("Apply after the crosses wrote no file:\n%s", trigLines(e))
	}
	if got := string(s.workspace.Pages["event01.txt"]); !strings.Contains(got, "tips=2") {
		t.Fatalf("the message page lost its tag attribute: %q", got)
	}
}
