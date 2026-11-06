package game

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/town"
)

// matrixEvents are the events the town hosts and pages raise on a room, spelled
// here by their description names so the recorded matrix does not follow the
// code that raises them. "no-such-event" is an event no description binds.
var matrixEvents = []string{"cell-press", "leave-press", "talk", "dismiss", "hire", "hire-short", "teach", "no-such-event"}

// matrixDescriptions are the shipped town descriptions of both games, by the
// name they answer to in the catalog.
var matrixDescriptions = []string{"rom1-square", "rom2-town1", "rom2-town2", "rom2-town3"}

// roomSoundMatrix is every sound request each event of each town room made
// when roomSound was recorded, as "town/room/event: source:key ...". A row
// absent here is a room that stays silent on that event.
var roomSoundMatrix = []string{
	"rom1-square/tavern/cell-press: tavern-interior:town/inn/helper.wav",
	"rom1-square/tavern/dismiss: tavern-interior:noadd.wav",
	"rom1-square/tavern/hire: tavern-interior:add.wav",
	"rom1-square/tavern/hire-short: tavern-interior:town/shop/nofit.wav",
	"rom1-square/tavern/leave-press: tavern-interior:out.wav",
	"rom1-square/tavern/talk: tavern-interior:talk.wav",
	"rom1-square/shop/button-press-0: shop-interior:undo.wav",
	"rom1-square/shop/button-press-3: shop-interior:out.wav",
	"rom1-square/shop/buy: shop-interior:town/buy.wav",
	"rom1-square/shop/buy-short: shop-interior:town/shop/nofit.wav",
	"rom1-square/shop/pack-arrow: shop-interior:scroll.wav",
	"rom1-square/shop/rack-changed: shop-interior:town/shop/depart.wav",
	"rom1-square/shop/sell: shop-interior:town/sell.wav",
	"rom1-square/shop/shelf-arrow: shop-interior:scroll.wav",
	"rom1-square/shop/table-drop: shop-interior:put_on.wav",
	"rom1-square/shop/table-pick: shop-interior:put_off.wav",
	"rom1-square/school/leave-press: school-training:out.wav",
	"rom1-square/school/teach: school-training:town/school/teach.wav",
	"rom2-town1/inn/dismiss: inn-interior:noadd.wav",
	"rom2-town1/inn/hire: inn-interior:add.wav",
	"rom2-town1/inn/hire-short: inn-interior:town/shop/nofit.wav",
	"rom2-town1/inn/leave-press: inn-interior:out.wav",
	"rom2-town1/inn/talk: inn-interior:talk.wav",
	"rom2-town1/shop/button-press-0: shop-interior:undo.wav",
	"rom2-town1/shop/button-press-3: shop-interior:out.wav",
	"rom2-town1/shop/buy: shop-interior:town/buy.wav",
	"rom2-town1/shop/buy-short: shop-interior:town/shop/nofit.wav",
	"rom2-town1/shop/doll-off: shop-interior:put_off.wav",
	"rom2-town1/shop/doll-on: shop-interior:put_on.wav",
	"rom2-town1/shop/pack-arrow: shop-interior:scroll.wav",
	"rom2-town1/shop/rack-changed: shop-interior:town/shop/depart.wav",
	"rom2-town1/shop/sell: shop-interior:town/sell.wav",
	"rom2-town1/shop/shelf-arrow: shop-interior:scroll.wav",
	"rom2-town1/shop/table-drop: shop-interior:put_on.wav",
	"rom2-town1/shop/table-pick: shop-interior:put_off.wav",
	"rom2-town2/inn/dismiss: inn-interior:noadd.wav",
	"rom2-town2/inn/hire: inn-interior:add.wav",
	"rom2-town2/inn/hire-short: inn-interior:town/shop/nofit.wav",
	"rom2-town2/inn/leave-press: inn-interior:out.wav",
	"rom2-town2/inn/talk: inn-interior:talk.wav",
	"rom2-town2/shop/button-press-0: shop-interior:undo.wav",
	"rom2-town2/shop/button-press-3: shop-interior:out.wav",
	"rom2-town2/shop/buy: shop-interior:town/buy.wav",
	"rom2-town2/shop/buy-short: shop-interior:town/shop/nofit.wav",
	"rom2-town2/shop/doll-off: shop-interior:put_off.wav",
	"rom2-town2/shop/doll-on: shop-interior:put_on.wav",
	"rom2-town2/shop/pack-arrow: shop-interior:scroll.wav",
	"rom2-town2/shop/rack-changed: shop-interior:town/shop/depart.wav",
	"rom2-town2/shop/sell: shop-interior:town/sell.wav",
	"rom2-town2/shop/shelf-arrow: shop-interior:scroll.wav",
	"rom2-town2/shop/table-drop: shop-interior:put_on.wav",
	"rom2-town2/shop/table-pick: shop-interior:put_off.wav",
	"rom2-town3/inn/dismiss: inn-interior:noadd.wav",
	"rom2-town3/inn/hire: inn-interior:add.wav",
	"rom2-town3/inn/hire-short: inn-interior:town/shop/nofit.wav",
	"rom2-town3/inn/leave-press: inn-interior:out.wav",
	"rom2-town3/inn/talk: inn-interior:talk.wav",
	"rom2-town3/shop/button-press-0: shop-interior:undo.wav",
	"rom2-town3/shop/button-press-3: shop-interior:out.wav",
	"rom2-town3/shop/buy: shop-interior:town/buy.wav",
	"rom2-town3/shop/buy-short: shop-interior:town/shop/nofit.wav",
	"rom2-town3/shop/doll-off: shop-interior:put_off.wav",
	"rom2-town3/shop/doll-on: shop-interior:put_on.wav",
	"rom2-town3/shop/pack-arrow: shop-interior:scroll.wav",
	"rom2-town3/shop/rack-changed: shop-interior:town/shop/depart.wav",
	"rom2-town3/shop/sell: shop-interior:town/sell.wav",
	"rom2-town3/shop/shelf-arrow: shop-interior:scroll.wav",
	"rom2-town3/shop/table-drop: shop-interior:put_on.wav",
	"rom2-town3/shop/table-pick: shop-interior:put_off.wav",
}

// matrixScreen answers the town screen that shows description name, with the
// sound recorder attached and every key the description's rooms name in the
// sound bank.
func matrixScreen(t *testing.T, name string) (*townScreen, *town.Description, *roomSoundRecorder) {
	t.Helper()
	if name == "rom1-square" {
		f := shellFrontEnd()
		rec := addRoomSounds(f)
		d := f.townUI.townDescription()
		for i := range d.Rooms {
			if d.Rooms[i].Scene != nil {
				addSceneSounds(f, d, d.Rooms[i].Name)
			}
		}
		return f.townUI, d, rec
	}
	f, app, screen := secondCampaignFixtureFiles(t, false)
	rec := addRoomSounds(f)
	var id int
	if _, err := fmt.Sscanf(name, "rom2-town%d", &id); err != nil {
		t.Fatal(err)
	}
	c := innAt(id, 30, nil)
	c.room = secondTownInn
	f.Town.second = c
	app.SetTown(screen)
	if screen.composed == nil {
		screen.composed = newTownSceneState(nil)
	}
	d := screen.townDescription()
	if d == nil || d.Town != name {
		t.Fatalf("the screen shows %v, want %s", d, name)
	}
	for i := range d.Rooms {
		if d.Rooms[i].Scene != nil {
			addSceneSounds(f, d, d.Rooms[i].Name)
		}
	}
	return screen, d, rec
}

// matrixEventNames answers matrixEvents with every event any shipped
// description binds a sound to.
func matrixEventNames() []string {
	seen := map[string]bool{}
	for _, e := range matrixEvents {
		seen[e] = true
	}
	for _, d := range townDescriptions {
		for i := range d.Rooms {
			if d.Rooms[i].Scene == nil {
				continue
			}
			for _, tr := range d.Rooms[i].Scene.Sounds.Triggers {
				seen[tr.On] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for e := range seen {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}

// recordRoomSoundMatrix drives every event through roomSound on every room of
// every shipped town description and answers the non-silent rows.
func recordRoomSoundMatrix(t *testing.T) []string {
	t.Helper()
	var rows []string
	for _, name := range matrixDescriptions {
		screen, d, rec := matrixScreen(t, name)
		for i := range d.Rooms {
			room := d.Rooms[i].Name
			if d.Rooms[i].Scene == nil {
				continue
			}
			for _, ev := range matrixEventNames() {
				rec.endAll()
				from := len(rec.requests)
				screen.roomSound(room, ev)
				if keys := rec.keys(from); len(keys) != 0 {
					rows = append(rows, fmt.Sprintf("%s/%s/%s: %s", name, room, ev, strings.Join(keys, " ")))
				}
			}
		}
	}
	return rows
}

// The events the host, the tavern, the inn and the school raise are the
// events the descriptions of those pages bind sounds to: every raised event
// is bound by some description, and no tavern, inn or school page binds an
// event nothing raises.
func TestRoomEventsAreTheDescriptionsEvents(t *testing.T) {
	raised := []roomEvent{roomEventCellPress, roomEventLeave, roomEventTalk, roomEventDismiss, roomEventHire, roomEventHireShort, roomEventTeach}
	bound := map[string]bool{}
	for _, d := range townDescriptions {
		for _, room := range []string{"tavern", "inn", "school"} {
			r := d.Room(room)
			if r == nil || r.Scene == nil {
				continue
			}
			for _, tr := range r.Scene.Sounds.Triggers {
				bound[tr.On] = true
				if !slices.Contains(raised, roomEvent(tr.On)) {
					t.Errorf("%s %s binds %q, which no host or page raises", d.Town, room, tr.On)
				}
			}
		}
	}
	for _, ev := range raised {
		if !bound[string(ev)] {
			t.Errorf("event %q is bound by no tavern, inn or school page", ev)
		}
		if !slices.Contains(matrixEvents, string(ev)) {
			t.Errorf("event %q is not in the recorded matrix", ev)
		}
	}
}

// TestTownRoomSoundMatrix: every event of every town room of both games, run
// through roomSound, requests the sounds recorded before the room sounds were
// raised as events by the host, the tavern, the inn and the school.
func TestTownRoomSoundMatrix(t *testing.T) {
	got := recordRoomSoundMatrix(t)
	if !slices.Equal(got, roomSoundMatrix) {
		t.Fatalf("the room sound matrix changed; recorded rows:\n%s", strings.Join(got, "\n"))
	}
}
