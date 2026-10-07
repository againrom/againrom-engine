package game

import (
	"testing"

	"againrom/pkg/ui"
)

// The tavern and school refusals post no message line. A hire refused for its
// price requests tavern sound slot 0xa0 and nothing else does (TAVERN-LINES-022).
func TestTownRefusalsPostNoLineAndHireRefusalRequestsItsSlot(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI

	if act := s.townSurfaceButton(tavernButtonSleep); act.Msg != "" {
		t.Fatalf("Sleep = %q, want no line even without a shop to restock", act.Msg)
	}

	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	f.Town.mercPool[3], f.Town.gold = 2, 49
	if act := s.townSurfaceButton(tavernButtonHire); act.Msg != "" {
		t.Fatalf("refused hire posted %q", act.Msg)
	}
	if got := s.tavernSlotRequests[tavernSlotHireRefused]; got != 1 {
		t.Fatalf("hire refusal sound requests = %d, want 1", got)
	}
	act := s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, true)
	if act.Msg != "" || s.tavernSlotRequests[tavernSlotHireRefused] != 2 || len(s.tavernSlotRequests) != 1 {
		t.Fatalf("double-click refusal = %q, requests %v", act.Msg, s.tavernSlotRequests)
	}
	if f.Town.MercenaryHired(3) || f.Town.Gold() != 49 {
		t.Fatal("refused hire changed the purse or the roster")
	}

	// A hire within the purse still posts its line and requests no refusal.
	f.Town.gold = 1000
	if act := s.townSurfaceButton(tavernButtonHire); act.Msg != "" || !f.Town.MercenaryHired(3) || s.tavernSlotRequests[tavernSlotHireRefused] != 2 {
		t.Fatalf("hire = %q, requests %v", act.Msg, s.tavernSlotRequests)
	}

	// Training: no selection and a purse below the price post no line.
	s.room, s.schoolCell = roomSchool, schoolNoSelection
	if act := s.townSurfaceButton(0); act.Msg != "" {
		t.Fatalf("Train without selection = %q", act.Msg)
	}
	s.schoolCell = 0
	f.Town.gold = 0
	level := f.Carried[0].Hero.Skill[1]
	if act := s.townSurfaceButton(0); act.Msg != "" || f.Carried[0].Hero.Skill[1] != level {
		t.Fatalf("Train past the purse = %q, level %d -> %d", act.Msg, level, f.Carried[0].Hero.Skill[1])
	}
	f.Town.gold = 10000
	if act := s.townSurfaceButton(0); act.Msg == "" || f.Carried[0].Hero.Skill[1] != level+1 {
		t.Fatalf("Train within the purse = %q, level %d", act.Msg, f.Carried[0].Hero.Skill[1])
	}
	if len(s.tavernSlotRequests) != 1 {
		t.Fatalf("training requested tavern slots: %v", s.tavernSlotRequests)
	}
}
