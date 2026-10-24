package ui

import (
	"errors"
	"strings"
	"testing"
)

// refusingPanelEdits is a host whose campaign refuses every test play.
type refusingPanelEdits struct{ mapPanelEdits }

var errNoTestPlay = errors.New("Test play plays first-game maps only")

func (p *refusingPanelEdits) TestPlayRefusal() error { return errNoTestPlay }
func (p *refusingPanelEdits) TestPlay() (string, error) {
	p.plays++
	return "", errNoTestPlay
}

// onlyEndless is the facts and findings of a map with a start point and no win.
func onlyEndless() (InspectionMapFacts, []InspectionFinding) {
	f := playable()
	f.Wins = nil
	return f, []InspectionFinding{{Rule: RuleVictory, Text: "nothing wins this map"}}
}

// Test play of a map whose only warning is the endless one starts at once; a
// save into the mod and any other warning still ask.
func TestTestPlayStartsAtOnceWhenOnlyTheEndlessWarningRemains(t *testing.T) {
	f, rows := onlyEndless()
	p := &mapPanelEdits{facts: f, rows: rows}
	e, _ := mapPanelEditor(t, p, nil)
	_ = e.Key("testplay")
	if e.prompt.kind == promptNotice || p.plays != 1 {
		t.Fatalf("endless map: prompt %+v, %d runs", e.prompt, p.plays)
	}
	_ = e.Key("saveas")
	if e.prompt.kind != promptNotice || p.modSaves != 0 {
		t.Fatalf("the save into the mod did not ask: prompt %+v", e.prompt)
	}
	_ = e.Key("escape")

	f2, rows2 := endless()
	p.facts, p.rows, p.plays = f2, rows2, 0
	_ = e.Key("testplay")
	if e.prompt.kind != promptNotice || p.plays != 0 {
		t.Fatalf("a missing start point did not ask: prompt %+v, %d runs", e.prompt, p.plays)
	}
	if notes := strings.Join(e.noticeSpec().Notes, " / "); !strings.Contains(notes, "Start point:") || !strings.Contains(notes, "Victory:") {
		t.Fatalf("the notice does not list both warnings: %q", notes)
	}
	_ = e.Key("escape")
	if e.prompt.open() || p.plays != 0 {
		t.Fatal("Cancel started the test play or left the notice")
	}
	_ = e.Key("testplay")
	_ = e.Key("enter")
	if p.plays != 1 {
		t.Fatalf("Continue ran %d test plays", p.plays)
	}
}

// A refusal of the host comes before any notice.
func TestTestPlayRefusalComesBeforeTheNotice(t *testing.T) {
	f, rows := endless()
	p := &refusingPanelEdits{mapPanelEdits{facts: f, rows: rows}}
	e, _ := mapPanelEditor(t, &p.mapPanelEdits, nil)
	e.doc.Edits = p
	_ = e.Key("testplay")
	if e.prompt.kind == promptNotice || e.Message() != errNoTestPlay.Error() || p.plays != 1 {
		t.Fatalf("prompt %+v, message %q, %d runs", e.prompt, e.Message(), p.plays)
	}
}
