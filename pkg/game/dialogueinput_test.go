package game

import (
	"testing"

	"againrom/internal/synth"
)

func TestDialogueAcceptedTailReachesTownAndMission(t *testing.T) {
	first, last := " \tA\nB\v\f ", " \tLast\nRow \f"
	payload := "<part=1>\r\n" + first + "\r\n<part=2>\n" + last + "\x00<part=3>\nIgnored"
	f, app, town := visibleOfferApp(t, 30)
	f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte(payload)}})
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	if town.dialogue.text != first {
		t.Fatalf("town delivered %q want unchanged accepted tail %q", town.dialogue.text, first)
	}
	if err := app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if town.dialogue.text != last {
		t.Fatalf("town page delivered %q want %q", town.dialogue.text, last)
	}
	if err := app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if town.room == roomTalk {
		t.Fatal("town searched beyond NUL")
	}

	driver, viewer := missionDialogue(t, payload, nil)
	if got, _, _ := viewer.NoticeState(); got != first {
		t.Fatalf("mission delivered %q want unchanged accepted tail %q", got, first)
	}
	driver.advanceNotice()
	if got, _, _ := viewer.NoticeState(); got != last {
		t.Fatalf("mission page delivered %q want %q", got, last)
	}
	driver.advanceNotice()
	if _, _, open := viewer.NoticeState(); open {
		t.Fatal("mission searched beyond NUL")
	}
}

func TestDialogueAcceptedTailMissingLFAndBackwardsRangeStayBounded(t *testing.T) {
	for _, payload := range []string{"<part=1>no LF", "<part=1>\r\nNo body CR<part=2>\r\nLater"} {
		f, app, town := visibleOfferApp(t, 30)
		f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte(payload)}})
		if err := app.HeadlessActivate("SHOP"); err != nil {
			t.Fatal(err)
		}
		if town.dialogue.text != "Nothing to say" {
			t.Fatalf("malformed town tail accepted %q", town.dialogue.text)
		}
		driver, _ := missionDialogue(t, "<part=1>\nValid", nil)
		driver.closeNotice()
		driver.mission.src = missionSource{missionEvent(t, 7, 4): []byte(payload)}
		if driver.openDialogue(4) {
			t.Fatal("malformed mission tail accepted", payload)
		}
	}
}
