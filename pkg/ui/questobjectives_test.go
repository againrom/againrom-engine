package ui

import (
	"slices"
	"strings"
	"testing"
)

func TestQuestObjectivesDrawTheTitleOnItsOwnLine(t *testing.T) {
	f := openMissionMenu(t)
	f.menuFont = gameMenuTestFont()
	f.menuContext.Objective = "Great Mage Skrakan\r\n\r\nYou must get to the tower."
	got := f.questLines()
	want := []string{" " + "Great Mage Skrakan", " " + "You must get to the tower."}
	if !slices.Equal(got, want) {
		t.Fatalf("titled briefing lines = %q, want %q", got, want)
	}
	for _, line := range got {
		if strings.Contains(line, "?") {
			t.Errorf("line %q holds a line-break stand-in", line)
		}
	}
}

func TestQuestObjectivesWithoutATitleStayPlain(t *testing.T) {
	f := openMissionMenu(t)
	f.menuFont = gameMenuTestFont()
	f.menuContext.Objective = "Find the gate."
	if got := f.questLines(); !slices.Equal(got, []string{"Find the gate."}) {
		t.Fatalf("plain briefing lines = %q", got)
	}
}
