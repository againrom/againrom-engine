package game

import "testing"

// TestReleaseTownRefusalsPostNoLineOnInstall drives Sleep, a hire past the
// purse and a training past the purse through the installed tavern and school
// of the root under test. None posts a line; the hire refusal requests tavern
// sound slot 0xa0 and the others request no tavern slot.
func TestReleaseTownRefusalsPostNoLineOnInstall(t *testing.T) {
	f := releaseFront(t)
	f.Carried = f.NextParty()
	f.Town.Won(10)
	f.Town.Won(20)
	f.arriveInTown()
	s := f.TownScreen().(*townScreen)
	s.Choose(0)
	for i := 0; s.room == roomTalk && i < 64; i++ {
		s.AdvanceTownDialogue()
	}
	if s.room != roomTavern {
		t.Fatal("production town entry did not reach the tavern")
	}
	s.CloseTip()
	if act := s.townSurfaceButton(tavernButtonSleep); act.Msg != "" || f.Shop.restocks != 1 {
		t.Fatalf("Sleep = %q, restocks %d", act.Msg, f.Shop.restocks)
	}
	mercs := s.tavernMercenaries()
	if len(mercs) == 0 {
		t.Fatal("the first town chapter lists no mercenary squad")
	}
	typ := mercs[0].Type
	s.tavernSelection = tavernCandidateKey{kind: tavernCandidateMercenary, id: typ}
	f.Town.gold = 0
	if act := s.townSurfaceButton(tavernButtonHire); act.Msg != "" || s.tavernSlotRequests[tavernSlotHireRefused] != 1 ||
		f.Town.MercenaryHired(typ) {
		t.Fatalf("hire past the purse = %q, requests %v, hired %v", act.Msg, s.tavernSlotRequests, f.Town.MercenaryHired(typ))
	}
	if len(s.tavernSlotRequests) != 1 {
		t.Fatalf("tavern slot requests = %v, want only the refusal slot", s.tavernSlotRequests)
	}

	s.room, s.schoolCell = roomSchool, schoolNoSelection
	if act := s.townSurfaceButton(0); act.Msg != "" {
		t.Fatalf("Train without selection = %q", act.Msg)
	}
	s.schoolCell = 0
	if f.Carried[0].Mage {
		s.schoolCell = 5
	}
	slot, price, ok := s.selectedSchoolSlot()
	if !ok || price <= 0 {
		t.Fatal("production hero has no priced class skill")
	}
	f.Town.gold = price - 1
	level := f.Carried[0].Hero.Skill[slot]
	if act := s.townSurfaceButton(0); act.Msg != "" || f.Carried[0].Hero.Skill[slot] != level || f.Town.Gold() != price-1 {
		t.Fatalf("Train past the purse = %q, level %d -> %d, gold %d", act.Msg, level, f.Carried[0].Hero.Skill[slot], f.Town.Gold())
	}
	if len(s.tavernSlotRequests) != 1 {
		t.Fatalf("training requested tavern slots: %v", s.tavernSlotRequests)
	}
}
