package game

import (
	"strings"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestOpenLoadNoticeOpensADialogueTheFrontEndCanRead is 1032 B3's wiring at
// the other end: mapWorld.openLoadNotice posts through the same
// ui.Viewer.SetDialogue a mission-script dialogue uses (world.go's
// openDialogue), so LiveNotice — what a real front end's window and
// cmd/savecheck both read — sees it the same way.
func TestOpenLoadNoticeOpensADialogueTheFrontEndCanRead(t *testing.T) {
	mw := &mapWorld{mission: &missionNotices{}, view: &ui.Viewer{}}
	mw.openLoadNotice([]string{"substituted text"})

	text, kind, open := mw.view.NoticeState()
	if !open {
		t.Fatal("openLoadNotice left the viewer with no open notice")
	}
	if kind != ui.NoticeDialogue {
		t.Errorf("notice kind = %v, want ui.NoticeDialogue", kind)
	}
	if text != "substituted text" {
		t.Errorf("notice text = %q, want %q", text, "substituted text")
	}
	if mw.mission.payload != nil {
		t.Error("openLoadNotice carries an event payload; advanceNotice would try to page it")
	}
}

// TestLoadNoticePagesThroughEveryPageBeforeItCloses is the paging half of
// the same wiring (1032 return 1).
//
// It fails against the code before this return in two ways at once: that
// openLoadNotice took one string, and that advanceNotice consulted EventPart
// first — which answers "no next part" for a nil payload, so the first dismiss
// closed the window and every page after the first was unreachable.
func TestLoadNoticePagesThroughEveryPageBeforeItCloses(t *testing.T) {
	pages := []string{"first page", "second page", "third page"}
	mw := &mapWorld{mission: &missionNotices{}, view: &ui.Viewer{}}
	mw.openLoadNotice(pages)

	for i, want := range pages {
		got, _, open := mw.view.NoticeState()
		if !open {
			t.Fatalf("page %d of %d: no notice open", i+1, len(pages))
		}
		if got != want {
			t.Fatalf("page %d of %d = %q, want %q", i+1, len(pages), got, want)
		}
		if dest, _, _ := mw.advanceNotice(); dest != ui.NoticeStay {
			t.Fatalf("dismissing page %d of %d: dest = %v, want ui.NoticeStay", i+1, len(pages), dest)
		}
	}
	if _, _, open := mw.view.NoticeState(); open {
		t.Error("notice still open after the last page was dismissed")
	}
	if mw.mission.pages != nil {
		t.Error("closeNotice left the pages behind; the next notice would page them again")
	}
}

// TestResumeWorldReturnsTheRefusalTheWayThePlayerReadsIt (1032 return 1).
//
// The refusal sim.UpgradeSaveForm writes is a whole sentence addressed to the
// player and it reaches him unedited on the load window's message line, which
// holds 104 columns. This test fails against the code before this return, where
// resumeWorld wrapped it as "save's world half: ...": nineteen columns of a
// vocabulary the player has no use for, out of the 104 he has.
//
// The second half is the boundary. A byte form that will not DECODE is a
// different failure, and it keeps the prefix that says which half of the save
// went wrong.
func TestResumeWorldReturnsTheRefusalTheWayThePlayerReadsIt(t *testing.T) {
	ms := &Mission{World: &sim.World{}}
	err := resumeWorld(ms, &Snapshot{World: []byte{46}}, nil)
	if err == nil {
		t.Fatal("a byte form at version 46 resumed with no error")
	}
	if !strings.HasPrefix(err.Error(), "this save is too old to open") {
		t.Errorf("the refusal does not reach the caller unchanged: %q", err)
	}

}
