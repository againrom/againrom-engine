package archtest

import (
	"os"
	"strings"
	"testing"
)

// Outside the room page file, no production code of the game or its screens
// chooses a town room's sound: hosts and pages raise events, and the room
// description's sound table binds each event to its sound.
func TestTownRoomSoundsAreChosenByTheDescription(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := RoomSoundSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s; raise a room event and bind its sound in the room description", s)
	}
	keys, err := RoomSoundKeys(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"out.wav", "town/school/teach.wav", "talk.wav", "town/inn/helper.wav"} {
		if !keys[want] {
			t.Errorf("the descriptions' sound tables do not request %s", want)
		}
	}
}

func roomSoundHits(src string, keys map[string]bool) string {
	return strings.Join(roomSoundsInSource("x.go", []byte(src), keys), ", ")
}

// One planted instance per shape: each is named, and a look-alike is not.
func TestRoomSoundScanSeesEachShape(t *testing.T) {
	keys := map[string]bool{"out.wav": true}
	for _, tc := range []struct{ name, src, want string }{
		{"roomSound with a room and an event", `package x
func f(t *T) { t.roomSound("tavern", "hire") }`, "x.go:2 roomSound is called with a string"},
		{"shownRoomSound with an event", `package x
func f(t *T) { t.shownRoomSound("leave-press") }`, "x.go:2 shownRoomSound is called with a string"},
		{"raiseRoomEvent with a converted string", `package x
func f(t *T) { t.raiseRoomEvent(roomTavern, roomEvent("hire")) }`, "x.go:2 raiseRoomEvent is called with a string"},
		{"raiseShownRoomEvent with an event", `package x
func f(t *T) { t.raiseShownRoomEvent("talk") }`, "x.go:2 raiseShownRoomEvent is called with a string"},
		{"shopRoomSound with an event", `package x
func f(t *T) { t.shopRoomSound("buy") }`, "x.go:2 shopRoomSound is called with a string"},
		{"a sound name passed to a bank", `package x
func f(b *B) { b.namedSample("any/where.wav") }`, "x.go:2 namedSample is passed a sound name"},
		{"a key passed to the scene", `package x
func f(s *S) { s.Trigger("out.wav") }`, "x.go:2 Trigger is passed a sound name, x.go:2 a string is the key out.wav of a room sound table"},
		{"a file name passed to the scene", `package x
func f(s *S) { s.Trigger("any/where.wav") }`, "x.go:2 Trigger is passed a sound name"},
		{"a map from an action to a sound", `package x
var m = map[int]string{1: "any/where.wav"}`, "x.go:2 a map names a sound"},
		{"a table key held as a string", `package x
const k = "out.wav"`, "x.go:2 a string is the key out.wav of a room sound table"},
		{"events raised by constants", `package x
func f(t *T) { t.raiseRoomEvent(roomTavern, roomEventHire); t.raiseShownRoomEvent(roomEventTalk) }`, ""},
		{"a map of names without sounds", `package x
var m = map[int]string{1: "tavern"}`, ""},
		{"a call of another function with a string", `package x
func f(t *T) { t.speak("hire"); t.Request("hire") }`, ""},
	} {
		if got := roomSoundHits(tc.src, keys); got != tc.want {
			t.Errorf("%s: hits = %q, want %q", tc.name, got, tc.want)
		}
	}
}
