package game

import (
	"fmt"
	"testing"

	"againrom/pkg/ui"
)

// The inn queues the mission of every conversation it opens and registers the
// queue when the player leaves: nothing is committed inside the inn
// (DLG-LIFE-005). The queue is the town screen's own session state, so a loaded
// or a new game starts without one.

// hearInnSpeaker enters the fixture inn through App input and reads NPC 22's
// conversation to its end. The inn holds mission 30 for him.
func hearInnSpeaker(t *testing.T, app *ui.App, s *townScreen) {
	t.Helper()
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("NPC 22"); err != nil {
		t.Fatal(err)
	}
	for n := 0; s.room == roomTalk && n < dialogueKeysPages; n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomTavern {
		t.Fatalf("the conversation left room %d, want the inn", s.room)
	}
}

// TestEveryWayOutOfTheInnCommitsItsQueue leaves the inn by Escape, by its Exit
// button and by the screen's own Back, each time after one conversation. The
// quest is queued, not at the gates, until the room is left.
func TestEveryWayOutOfTheInnCommitsItsQueue(t *testing.T) {
	for _, route := range []string{"Escape", "Exit button", "Back"} {
		t.Run(route, func(t *testing.T) {
			f, app, s := dialogueKeysApp(t, 30)
			hearInnSpeaker(t, app, s)
			if got := f.Town.Available(); len(got) != 0 || len(s.innQueue) != 1 {
				t.Fatalf("inside the inn the gates hold %v and %d are queued, want nothing and 1", got, len(s.innQueue))
			}
			var err error
			switch route {
			case "Escape":
				err = app.HeadlessKey("escape")
			case "Exit button":
				s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonExit}, false)
			case "Back":
				if !s.Back() {
					t.Fatal("Back left no room")
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if s.room != roomSquare {
				t.Fatalf("%s left room %d, want the square", route, s.room)
			}
			if got := f.Town.Available(); !containsMission(got, 30) || len(got) != 1 || len(s.innQueue) != 0 {
				t.Fatalf("after %s the gates hold %v and %d are queued, want [30] and nothing", route, got, len(s.innQueue))
			}
		})
	}
}

// TestALoadedGameDropsWhatTheInnQueued installs a fresh campaign, as a load or
// a new game does, while a conversation's quest is still queued. Nothing of it
// reaches the new game, and leaving the inn there registers nothing.
func TestALoadedGameDropsWhatTheInnQueued(t *testing.T) {
	f, app, s := dialogueKeysApp(t, 30)
	hearInnSpeaker(t, app, s)
	if len(s.innQueue) != 1 {
		t.Fatalf("queued %d, want 1", len(s.innQueue))
	}
	f.installCandidate(&restoreCandidate{town: NewTown(f.Campaign.Value()), fame: SnapshotFame{Known: true}, activate: func() {}})
	f.Town.Arrive()
	if len(s.innQueue) != 0 || s.room != roomSquare {
		t.Fatalf("the new game kept %d queued and room %d", len(s.innQueue), s.room)
	}
	for i, d := range townDoors {
		if d.room == roomTavern {
			s.Choose(i)
		}
	}
	if s.room != roomTavern || !s.Back() {
		t.Fatalf("the new game's inn door opened room %d", s.room)
	}
	if got := f.Town.Available(); len(got) != 0 {
		t.Fatalf("leaving the inn of the new game put %v at the gates, want nothing", got)
	}
}

// TestHearingOneSpeakerAgainQueuesEveryHearing hears the same speaker three
// times in one visit: all three hearings are queued.
func TestHearingOneSpeakerAgainQueuesEveryHearing(t *testing.T) {
	f, app, s := dialogueKeysApp(t, 30)
	hearInnSpeaker(t, app, s)
	for again := 2; again <= 3; again++ {
		if err := app.HeadlessActivate("NPC 22"); err != nil {
			t.Fatal(err)
		}
		for n := 0; s.room == roomTalk && n < dialogueKeysPages; n++ {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
		if len(s.innQueue) != again {
			t.Fatalf("after hearing him %d times %d are queued, want %d", again, len(s.innQueue), again)
		}
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if got := f.Town.Available(); fmt.Sprint(got) != "[30]" {
		t.Fatalf("the gates hold %v, want [30]", got)
	}
}
