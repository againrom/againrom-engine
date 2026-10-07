package game

import (
	"reflect"
	"testing"
)

// The numpad's Enter key does what the main Enter key does in a conversation
// window: each turns the page and the last page closes the window
// (DLG-KEYS-040). The two keys reach App as one Enter, so the state they leave
// behind on every page is one and the same.

// TestNumpadEnterTurnsTheDialoguePagesAsEnterDoes presses each of the two Enter
// keys on every page of a shop, a school and an inn quest conversation through
// App input. Every press changes the state, and the numpad's key leaves the
// state the main key leaves on every page.
func TestNumpadEnterTurnsTheDialoguePagesAsEnterDoes(t *testing.T) {
	for _, q := range []struct {
		name    string
		chapter int
		door    string
		room    townRoom
		inn     bool
	}{
		{"shop", 30, "SHOP", roomShop, false},
		{"school", 40, "SCHOOL", roomSchool, false},
		{"inn", 30, "TAVERN", roomTavern, true},
	} {
		traces := map[string][]string{}
		for _, key := range []string{"enter", "numpad-enter"} {
			t.Run(q.name+"/"+key, func(t *testing.T) {
				f, app, s := dialogueKeysApp(t, q.chapter)
				if err := app.HeadlessActivate(q.door); err != nil {
					t.Fatal(err)
				}
				if q.inn {
					if err := app.HeadlessActivate("NPC 22"); err != nil {
						t.Fatal(err)
					}
				}
				if s.room != roomTalk || s.said != 1 {
					t.Fatalf("the conversation is not open on its first page: room %d, page %d", s.room, s.said)
				}
				trace := []string{dialogueMoment(f, app, s)}
				for page := 1; page <= dialogueKeysPages; page++ {
					if err := app.HeadlessKey(key); err != nil {
						t.Fatal(err)
					}
					moment := dialogueMoment(f, app, s)
					if moment == trace[len(trace)-1] {
						t.Fatalf("%s on page %d changed nothing from %q", key, page, moment)
					}
					if page < dialogueKeysPages && (s.room != roomTalk || s.said != page+1) {
						t.Fatalf("%s on page %d left room %d on page %d, want page %d", key, page, s.room, s.said, page+1)
					}
					trace = append(trace, moment)
				}
				if s.room != q.room {
					t.Fatalf("the last %s left room %d, want the room behind the window, %d", key, s.room, q.room)
				}
				traces[key] = trace
			})
		}
		if !reflect.DeepEqual(traces["numpad-enter"], traces["enter"]) {
			t.Errorf("%s: numpad Enter left\n%q\nbut Enter left\n%q", q.name, traces["numpad-enter"], traces["enter"])
		}
	}
}
