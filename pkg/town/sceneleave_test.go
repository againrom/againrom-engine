package town

import "testing"

// A hidden scene releases its room's audio only when it was the shown one:
// a scene held hidden, never shown or already left, must not release a room
// another scene of the same name has opened.
func TestHiddenSceneReleasesItsRoomOnlyOnce(t *testing.T) {
	held, hh := newTestView(t, baseTown(), testLoader())
	for i := 0; i < 3; i++ {
		held.SetActive(false, false)
	}
	if got := hh.take(); got != "" {
		t.Fatalf("a scene never shown released its room: %q", got)
	}
	shown, h := newTestView(t, baseTown(), testLoader())
	shown.SetActive(true, true)
	h.take()
	for i := 0; i < 3; i++ {
		shown.SetActive(false, false)
	}
	if got := h.take(); got != "leave" || shown.Present() {
		t.Fatalf("a shown scene hidden three times: %q", got)
	}
	shown.SetActive(true, true)
	if !shown.Present() {
		t.Fatal("a scene shown again was not entered")
	}
	h.take()
	shown.SetActive(false, false)
	if got := h.take(); got != "leave" {
		t.Fatalf("a scene shown again and hidden: %q", got)
	}
}
