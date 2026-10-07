package game

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// TestReleaseNumpadEnterActsAsEnter presses the numpad's Enter key, and the main
// Enter key beside it, through App input on the installed first town: on every
// page of the shop's, the school's and the inn's quest conversation, and on every
// row of the square's Esc menu. The numpad's key leaves the state the main key
// leaves each time, and the main key changes the state wherever the row can be
// chosen, so each press acts (DLG-KEYS-040, MENU-INPUT-016).
func TestReleaseNumpadEnterActsAsEnter(t *testing.T) {
	keys := []string{"enter", "numpad-enter"}
	for _, q := range []struct {
		name string
		door string
		room townRoom
		inn  bool
	}{
		{"shop", "SHOP", roomShop, false},
		{"school", "SCHOOL", roomSchool, false},
		{"inn", "TAVERN", roomTavern, true},
	} {
		traces := map[string][]string{}
		for _, key := range keys {
			t.Run("conversation/"+q.name+"/"+key, func(t *testing.T) {
				f := releaseFront(t)
				chapter := 0
				if q.room == roomSchool {
					chapter = schoolOfferChapter(t, f)
				}
				app, s := roomExitApp(t, f, chapter)
				s.CloseTip()
				building := map[townRoom]TownBuilding{roomShop: TownShop, roomSchool: TownSchool, roomTavern: TownTavern}[q.room]
				var offer TownOffer
				for _, o := range f.Town.Offers(building) {
					if o.Mission > 0 {
						offer = o
						break
					}
				}
				if offer.Mission == 0 {
					t.Fatalf("chapter %d's %s holds no mission to hand over", f.Town.Chapter(), q.name)
				}
				roomExitEnter(t, app, s, q.door, q.room)
				if q.inn {
					if err := app.HeadlessActivate(fmt.Sprintf("NPC %d", offer.NPC)); err != nil {
						t.Fatal(err)
					}
				}
				if s.room != roomTalk || s.said != 1 {
					t.Fatalf("the %s conversation is not open on its first page: room %d, page %d", q.name, s.room, s.said)
				}
				pages := len(s.townLines())
				trace := []string{dialogueMoment(f, app, s)}
				for page := 1; page <= pages; page++ {
					if err := app.HeadlessKey(key); err != nil {
						t.Fatal(err)
					}
					moment := dialogueMoment(f, app, s)
					if moment == trace[len(trace)-1] {
						t.Fatalf("%s on page %d of %d changed nothing from %q", key, page, pages, moment)
					}
					if page < pages && (s.room != roomTalk || s.said != page+1) {
						t.Fatalf("%s on page %d of %d left room %d on page %d, want page %d", key, page, pages, s.room, s.said, page+1)
					}
					trace = append(trace, moment)
				}
				if s.room != q.room {
					t.Fatalf("the last %s left room %d, want the room behind the window, %d", key, s.room, q.room)
				}
				t.Logf("%s, %s: %d pages, mission %d, NPC %d", q.name, key, pages, offer.Mission, offer.NPC)
				traces[key] = trace
			})
		}
		if traces["enter"] != nil && !reflect.DeepEqual(traces["numpad-enter"], traces["enter"]) {
			t.Errorf("%s: numpad Enter left\n%q\nbut Enter left\n%q", q.name, traces["numpad-enter"], traces["enter"])
		}
	}

	// The square's Esc menu, open on its first row.
	openMenu := func(t *testing.T) (*ui.App, []ui.PickerRow) {
		t.Helper()
		f := releaseFront(t)
		app, s := roomExitApp(t, f, 0)
		s.CloseTip()
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if app.Screen() != ui.ScreenGameMenu {
			t.Fatalf("Escape on the square landed on screen %v, want the menu", app.Screen())
		}
		return app, app.HeadlessRows()
	}
	rowCount := -1
	t.Run("menu/rows", func(t *testing.T) {
		_, rows := openMenu(t)
		rowCount = len(rows)
	})
	if rowCount == 0 {
		t.Fatal("the square's Esc menu lists no row")
	}
	for row := 0; row < rowCount; row++ {
		moments := map[string]string{}
		var untouched string
		var choosable bool
		for _, key := range keys {
			t.Run(fmt.Sprintf("menu/row %d/%s", row, key), func(t *testing.T) {
				app, rows := openMenu(t)
				for n := 0; n < row; n++ {
					if err := app.HeadlessKey("down"); err != nil {
						t.Fatal(err)
					}
				}
				untouched, choosable = escMenuPressMoment(app), rows[row].Choosable
				if err := app.HeadlessKey(key); err != nil {
					t.Fatal(err)
				}
				moments[key] = escMenuPressMoment(app)
			})
		}
		if moments["enter"] == "" {
			continue
		}
		if moments["numpad-enter"] != moments["enter"] {
			t.Errorf("menu row %d: numpad Enter left %q but Enter left %q", row, moments["numpad-enter"], moments["enter"])
		}
		switch acted := moments["enter"] != untouched; {
		case choosable && !acted:
			t.Errorf("menu row %d can be chosen and Enter changed nothing from %q", row, untouched)
		case !choosable && acted:
			t.Errorf("menu row %d cannot be chosen and Enter changed %q to %q", row, untouched, moments["enter"])
		}
	}
	if rowCount > 0 {
		t.Logf("%d rows of the square's Esc menu, each pressed with both keys", rowCount)
	}
}

// escMenuPressMoment is what a press on a menu row could change: the screen, the
// rows the screen lists and the line the front end posted.
func escMenuPressMoment(app *ui.App) string {
	var texts []string
	for _, r := range app.HeadlessRows() {
		texts = append(texts, r.Text)
	}
	return fmt.Sprintf("screen %v, rows %q, line %q", app.Screen(), texts, app.HeadlessMessage())
}
