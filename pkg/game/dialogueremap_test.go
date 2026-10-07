package game

import (
	"againrom/internal/synth"
	"againrom/pkg/ui"
	"testing"
)

func TestMissionDriverPagesWithoutRepeatingBackdropShow(t *testing.T) {
	mw, v := missionDialogue(t, "<part=1>\r\none\r\n<part=2>\r\ntwo", nil)
	_, n, _ := v.DialogueBackdropPlan()
	if n != 1 {
		t.Fatal("first show", n)
	}
	mw.advanceNotice(ui.NoticeAdvance)
	_, n, _ = v.DialogueBackdropPlan()
	if n != 1 {
		t.Fatal("driver invoked show on page", n)
	}
	mw.advanceNotice(ui.NoticeAdvance)
	_, n, _ = v.DialogueBackdropPlan()
	if n != 0 {
		t.Fatal("driver close retained backdrop", n)
	}
}

func TestTownBackdropInvocationTracksOpenNotPage(t *testing.T) {
	f := speakerFrontEnd(t)
	s := f.TownScreen().(*townScreen)
	offer := TownOffer{NPC: 22, Mission: 32}
	if !s.openTownDialogue(TownTavern, offer, 22) || s.TownDialogueShows() != 1 {
		t.Fatal("first town show omitted")
	}
	s.AdvanceTownDialogue()
	if s.TownDialogueShows() != 1 {
		t.Fatal("town page invoked show")
	}
	if !s.openTownDialogue(TownTavern, offer, 22) || s.TownDialogueShows() != 2 {
		t.Fatal("explicit second town show became idempotent")
	}
	for n := 0; n < 3 && s.room == roomTalk; n++ {
		s.AdvanceTownDialogue()
	}
	if s.TownDialogueShows() != 0 {
		t.Fatal("closed town retained show depth")
	}
	if !s.openTownDialogue(TownTavern, offer, 22) || s.TownDialogueShows() != 1 {
		t.Fatal("new conversation inherited closed depth")
	}
	s.room = roomTavern
	f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/inn/npc/npc22m32.txt", Data: []byte("<part=2>\r\nlater")}})
	if !s.openTownDialogue(TownTavern, offer, 22) || s.TownDialogueShows() != 1 || s.dialogue.displayPart != 0 {
		t.Fatal("failed first lookup suppressed show")
	}
}
