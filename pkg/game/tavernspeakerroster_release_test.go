package game

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// releaseTavernApp loads the newest save in store through App's own load input
// and opens the tavern from the town square.
func releaseTavernApp(t *testing.T, f *FrontEnd, store SaveStore) (*ui.App, *townScreen) {
	t.Helper()
	app := f.App("tavern speaker")
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	for _, target := range []string{"load game", "@first", "TAVERN"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatalf("activate %q: %v", target, err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	return app, registerTownFront(f.TownScreen().(*townScreen), f)
}

// hearTavernSpeaker presses one tavern speaker's cell through App input and
// reads the conversation to its end. The press opens the speaker's own
// conversation and the speaker stays in the roster; the purse and the gates
// are as they were unless the caller expects otherwise.
func hearTavernSpeaker(t *testing.T, app *ui.App, s *townScreen, speaker TownOffer) {
	t.Helper()
	target := fmt.Sprintf("NPC %d", speaker.NPC)
	want := fmt.Sprintf("main/text/inn/npc/npc%02dm%d.txt", speaker.NPC, speaker.Mission)
	if err := app.HeadlessActivate(target); err != nil {
		t.Fatalf("press on %s: %v", target, err)
	}
	if path, _ := s.townTextPath(); s.room != roomTalk || path != want {
		t.Fatalf("press on %s opened room %v on %q, want the conversation %s", target, s.room, path, want)
	}
	for n := 0; s.room == roomTalk && n < 64; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomTavern {
		t.Fatalf("the conversation with %s ended in room %v, want the tavern", target, s.room)
	}
	found := false
	for _, cell := range s.tavernSurfaceCells() {
		found = found || cell.Semantic == target
	}
	if !found {
		t.Fatalf("%s left the roster", target)
	}
}

// hearTavernSpeakerTwice hears one speaker twice inside one tavern visit. The
// mission is queued and is at the gates only once the visit ends by Escape; the
// second press queues another hearing, and hearing the speaker again in the next
// visit hands over nothing.
func hearTavernSpeakerTwice(t *testing.T, app *ui.App, f *FrontEnd, s *townScreen, speaker TownOffer) {
	t.Helper()
	gold := f.Town.Gold()
	for press := 1; press <= 2; press++ {
		hearTavernSpeaker(t, app, s, speaker)
		if containsMission(f.Town.Available(), speaker.Mission) || len(s.innQueue) != press {
			t.Fatalf("press %d: available %v, queued %v; want nothing at the gates and %d queued hearings of mission %d", press, f.Town.Available(), s.innQueue, press, speaker.Mission)
		}
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if s.room != roomSquare || !containsMission(f.Town.Available(), speaker.Mission) || len(s.innQueue) != 0 {
		t.Fatalf("leaving the tavern: room %v, available %v, queued %v; want the square and mission %d at the gates", s.room, f.Town.Available(), s.innQueue, speaker.Mission)
	}
	gates := f.Town.Available()
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	hearTavernSpeaker(t, app, s, speaker)
	if got := f.Town.Available(); !reflect.DeepEqual(got, gates) || len(s.innQueue) != 0 {
		t.Fatalf("hearing the kept speaker changed the gates from %v to %v or queued %v", gates, got, s.innQueue)
	}
	t.Logf("npc%d handed over mission %d when the visit ended; gates %v; purse %d", speaker.NPC, speaker.Mission, f.Town.Available(), f.Town.Gold())
	if f.Town.Gold() != gold {
		t.Fatalf("hearing npc%d changed the purse from %d to %d", speaker.NPC, gold, f.Town.Gold())
	}
}

// hearKeptTavernSpeakerTwice hears a speaker whose pair a loaded SAV already
// lacks, twice: each press opens his conversation and the gates and the purse
// stay what the load left.
func hearKeptTavernSpeakerTwice(t *testing.T, app *ui.App, f *FrontEnd, s *townScreen, speaker TownOffer) {
	t.Helper()
	gates, gold := f.Town.Available(), f.Town.Gold()
	if !containsMission(gates, speaker.Mission) {
		t.Fatalf("the loaded game has %v at the gates, want mission %d", gates, speaker.Mission)
	}
	for press := 1; press <= 2; press++ {
		hearTavernSpeaker(t, app, s, speaker)
		if got := f.Town.Available(); !reflect.DeepEqual(got, gates) || len(s.innQueue) != 0 || f.Town.Gold() != gold {
			t.Fatalf("press %d on the kept speaker: gates %v, queued %v, purse %d; want %v, nothing queued and %d", press, got, s.innQueue, f.Town.Gold(), gates, gold)
		}
	}
}

// A tavern speaker who handed a mission over stays in the roster, and a second
// press opens the same conversation again, through App input on the installed
// first town: in the game the speaker was heard in, and in the game a SAV
// reloads, where the pair is already gone from the campaign arrays.
func TestReleaseTavernSpeakerStaysAndRepeats(t *testing.T) {
	f := releaseFront(t)
	f.Carried = f.NextParty()
	f.arriveInTown()
	var speaker TownOffer
	for _, o := range f.Town.Offers(TownTavern) {
		if o.Mission > 0 {
			speaker = o
			break
		}
	}
	if speaker.Mission == 0 {
		t.Fatalf("chapter %d tavern has no speaker with a mission: %+v", f.Town.Chapter(), f.Town.Offers(TownTavern))
	}
	t.Logf("chapter %d: npc%d hands over mission %d", f.Town.Chapter(), speaker.NPC, speaker.Mission)
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Unix(200, 0), payload); err != nil {
		t.Fatal(err)
	}
	app, s := releaseTavernApp(t, f, store)
	hearTavernSpeakerTwice(t, app, f, s, speaker)

	dir := t.TempDir()
	save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("town SAVE = %q, %v", name, err)
	}
	g := releaseFront(t)
	app, s = releaseTavernApp(t, g, SaveStore{Dir: dir})
	if g.Town.progress == nil {
		t.Fatal("the SAV did not load as an ordinary campaign")
	}
	if slices.Contains(g.Town.progress.innNPC, speaker.NPC) && slices.Contains(g.Town.progress.innMission, speaker.Mission) {
		t.Fatalf("the SAV still holds the pair of npc%d, mission %d", speaker.NPC, speaker.Mission)
	}
	arrays := [2][]int{slices.Clone(g.Town.progress.innNPC), slices.Clone(g.Town.progress.innMission)}
	hearKeptTavernSpeakerTwice(t, app, g, s, speaker)
	if got := [2][]int{g.Town.progress.innNPC, g.Town.progress.innMission}; !reflect.DeepEqual(got, arrays) {
		t.Fatalf("hearing the speaker again changed the campaign arrays from %v to %v", arrays, got)
	}
}
