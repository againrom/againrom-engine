package game_test

// Tests for a mission's event text.
//
// Every payload here is written by hand from the markup contract. The tag
// vocabulary is the game's and the bodies are ASCII, so no shipped bytes and no
// non-ASCII literal enter a fixture.

import (
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

func TestEventTextPathPadsTheEventNumberAndNotTheMission(t *testing.T) {
	for _, tc := range []struct {
		mission, event int
		want           string
	}{
		{10, 1, "main/text/battle/m10/event01.txt"},
		{10, 11, "main/text/battle/m10/event11.txt"},
		{100, 9, "main/text/battle/m100/event09.txt"},
		{1, 123, "main/text/battle/m1/event123.txt"},
	} {
		got, ok := game.EventTextPath(tc.mission, tc.event)
		if !ok || got != tc.want {
			t.Fatalf("EventTextPath(%d, %d) = %q, %v; want %q, true", tc.mission, tc.event, got, ok, tc.want)
		}
	}
}

func TestEventTextPathRefusesANumberThatNamesNothing(t *testing.T) {
	for _, tc := range [][2]int{{0, 1}, {-1, 1}, {10, 0}, {10, -3}} {
		if got, ok := game.EventTextPath(tc[0], tc[1]); ok {
			t.Fatalf("EventTextPath(%d, %d) = %q, true; want it refused", tc[0], tc[1], got)
		}
	}
}

// A named event text that does not ship produces nothing at all — and the
// SHAPE of the answer is what enforces that, because there is no error for a
// caller to log, fall back on, or fail a mission with.
func TestReadEventTextIsSilentWhenNothingShips(t *testing.T) {
	present := selSource{name: "main/text/battle/m10/event01.txt", data: []byte("<part=1>hello")}

	if b, ok := game.ReadEventText(present, 10, 1); !ok || string(b) != "<part=1>hello" {
		t.Fatalf("a shipped entry read back as %q, %v", b, ok)
	}
	for _, tc := range []struct {
		name           string
		src            terrain.EntrySource
		mission, event int
	}{
		{"no source", nil, 10, 1},
		{"entry absent", present, 10, 2},
		{"mission names nothing", present, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, ok := game.ReadEventText(tc.src, tc.mission, tc.event)
			if ok || b != nil {
				t.Fatalf("ReadEventText = %q, %v; want nil, false", b, ok)
			}
		})
	}
}

func TestEventPart(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		part    int
		want    string
		found   bool
	}{
		{
			name:    "part 1 of a single-part file",
			payload: "<part=1>The caravan is waiting.",
			part:    1, want: "The caravan is waiting.", found: true,
		},
		{
			name:    "a later part, bounded by the next tag",
			payload: "<part=1>first<part=2>second<part=3>third",
			part:    2, want: "second", found: true,
		},
		{
			name:    "the last part runs to the end of the payload",
			payload: "<part=1>first<part=2>last one",
			part:    2, want: "last one", found: true,
		},
		{
			name:    "a speaker tag ends a part and is not interpreted",
			payload: "<part=1><npc=51>spoken<part=2>b",
			part:    1, want: "", found: true,
		},
		{
			name:    "tag bodies are lowercased before the test",
			payload: "<PART=1>shouted",
			part:    1, want: "shouted", found: true,
		},
		{
			name:    "the tag body is a substring test, not an equality",
			payload: "<part=1,npc=51>joined",
			part:    1, want: "joined", found: true,
		},
		{
			name:    "no part tag at all",
			payload: "<npc=51>just a speaker",
			part:    1, found: false,
		},
		{
			name:    "no tag at all",
			payload: "bare prose with no markup",
			part:    1, found: false,
		},
		{
			name:    "an empty payload",
			payload: "",
			part:    1, found: false,
		},
		{
			name:    "an unterminated tag ends the scan",
			payload: "<part=1 and no close",
			part:    1, found: false,
		},
		{
			name:    "a part the file does not carry",
			payload: "<part=1>only one",
			part:    2, found: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := game.EventPart([]byte(tc.payload), tc.part, game.EventAudience{})
			if ok != tc.found || (ok && got != tc.want) {
				t.Fatalf("EventPart(%q, %d) = %q, %v; want %q, %v",
					tc.payload, tc.part, got, ok, tc.want, tc.found)
			}
		})
	}
}

// THE SUBSTRING HAZARD, reproduced rather than corrected. `part=10` contains
// `part=1`, and the first matching tag in file order wins — so a file whose
// part 10 comes first answers a request for part 1 with the part-10 body. No
// shipped file on either root trips this; an authored one could, and the
// original would answer the same way.
func TestEventPartReproducesThePrefixHazard(t *testing.T) {
	payload := []byte("<part=10>ten came first<part=1>one came second")
	got, ok := game.EventPart(payload, 1, game.EventAudience{})
	if !ok || got != "ten came first" {
		t.Fatalf("EventPart(part 1) = %q, %v; want the part-10 body, which is what the original returns", got, ok)
	}
	// And with the file in its natural order the hazard does not fire: part 1
	// is found first and part 10 still resolves to its own body.
	natural := []byte("<part=1>one<part=10>ten")
	if got, ok := game.EventPart(natural, 1, game.EventAudience{}); !ok || got != "one" {
		t.Fatalf("natural order, part 1 = %q, %v; want \"one\"", got, ok)
	}
	if got, ok := game.EventPart(natural, 10, game.EventAudience{}); !ok || got != "ten" {
		t.Fatalf("natural order, part 10 = %q, %v; want \"ten\"", got, ok)
	}
}

// The bytes come back exactly as the file holds them. A high byte is a Cyrillic
// character the draw path converts on its way to a glyph; decoding it here would
// corrupt every one of them before the renderer that expects them raw ever sees
// them.
func TestEventPartDoesNotTranscode(t *testing.T) {
	body := []byte{0x8f, 0xe0, 0xa8, 0xa2, 0xa5, 0xe2} // a CP866 word, written as bytes
	payload := append([]byte("<part=1>"), body...)
	got, ok := game.EventPart(payload, 1, game.EventAudience{})
	if !ok || got != string(body) {
		t.Fatalf("EventPart returned % x, want the payload's own bytes % x", got, body)
	}
}

// ---------------------------------------------------------------------------
// The two speaker tests.
// ---------------------------------------------------------------------------

// THE FILE TEST LOOKS FOR THREE LETTERS AND NOT FOUR, and this is the fixture
// that separates it from the per-part one. Every shipped file on both roots
// answers the same to both, so the corpus cannot tell them apart and only an
// authored payload can: prose containing the letters opens the pane on a file
// whose tags name nobody.
func TestEventHasSpeakerIsTheThreeLetterTest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		want    bool
	}{
		{"a tag names a speaker", "<npc=21,part=1>hello", true},
		{"only the prose carries the letters", "<part=1>the npc guild is closed", true},
		{"the letters in mixed case in prose", "<part=1>an NPC stands here", true},
		{"neither", "<part=1>nobody is speaking", false},
		{"the letters split by the tag boundary", "<part=1>n<part=2>pc", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := game.EventHasSpeaker([]byte(tc.payload)); got != tc.want {
				t.Fatalf("EventHasSpeaker(%q) = %v, want %v", tc.payload, got, tc.want)
			}
		})
	}
}

// The two tests DISAGREE on this payload, in both directions, and that is the
// whole reason both exist. A build that used one needle for both would pass
// every corpus measurement and fail exactly here.
func TestTheTwoSpeakerTestsAreDifferentTests(t *testing.T) {
	// The file mentions a speaker; no part names one.
	payload := []byte("<part=1>the npc guild<part=2>is closed")
	if !game.EventHasSpeaker(payload) {
		t.Fatal("the file test must answer yes: the prose carries the three letters")
	}
	for part := 1; part <= 2; part++ {
		if _, named := game.EventPartSpeaker(payload, part, game.EventAudience{}); named {
			t.Fatalf("part %d must name nobody: its tag carries no npc=", part)
		}
	}
}

func TestEventPartSpeakerReadsThePartsOwnTag(t *testing.T) {
	payload := []byte("<NPC=21, Part=1, female, tips=2 >first<Part=2>second<npc=44,part=3>third")
	for _, tc := range []struct {
		part    int
		want    int
		named   bool
		comment string
	}{
		{1, 21, true, "mixed case and spaces around the other terms"},
		{2, 0, false, "a part that names nobody"},
		{3, 44, true, "a later part naming a different speaker"},
		{4, 0, false, "a part that is not there"},
	} {
		got, named := game.EventPartSpeaker(payload, tc.part, game.EventAudience{})
		if got != tc.want || named != tc.named {
			t.Fatalf("part %d (%s) = %d, %v; want %d, %v", tc.part, tc.comment, got, named, tc.want, tc.named)
		}
	}
}

// The number and the naming are reported separately because the original's flag
// is set by the containment alone: a tag carrying `npc=` with no digits still
// refreshes the pane, for a speaker whose number is absent.
func TestEventPartSpeakerSeparatesNamingFromTheNumber(t *testing.T) {
	got, named := game.EventPartSpeaker([]byte("<part=1,npc=>said nothing"), 1, game.EventAudience{})
	if !named {
		t.Fatal("a tag containing npc= names a speaker even with no digits after it")
	}
	if got != 0 {
		t.Fatalf("speaker = %d, want 0: there is no number to read", got)
	}
}

// The speaker comes off the SAME tag the words come from. With the part-number
// hazard firing, the words and the speaker must both come from the part-10 tag
// — one walk, one answer to "which tag is part 1's".
func TestEventPartSpeakerFollowsThePartTheTextCameFrom(t *testing.T) {
	payload := []byte("<npc=44,part=10>ten came first<npc=21,part=1>one came second")
	body, ok := game.EventPart(payload, 1, game.EventAudience{})
	if !ok || body != "ten came first" {
		t.Fatalf("EventPart(1) = %q, %v; want the part-10 body", body, ok)
	}
	got, named := game.EventPartSpeaker(payload, 1, game.EventAudience{})
	if !named || got != 44 {
		t.Fatalf("EventPartSpeaker(1) = %d, %v; want 44 — the speaker of the tag the words came from", got, named)
	}
}

// A payload is the game's own byte string in the game's own code page. Folding
// it as Unicode would replace every high byte with an error rune and change the
// length of the thing being searched; folding only ASCII cannot.
func TestSpeakerTestsDoNotCorruptHighBytes(t *testing.T) {
	cyr := []byte{0x8f, 0xe0, 0xa8, 0xa2, 0xa5, 0xe2} // a CP866 word, written as bytes
	payload := append([]byte("<NPC=25,part=1>"), cyr...)
	if !game.EventHasSpeaker(payload) {
		t.Fatal("the file test must still find the tag past high bytes")
	}
	if got, named := game.EventPartSpeaker(payload, 1, game.EventAudience{}); !named || got != 25 {
		t.Fatalf("EventPartSpeaker = %d, %v; want 25, true", got, named)
	}
	if got, ok := game.EventPart(payload, 1, game.EventAudience{}); !ok || got != string(cyr) {
		t.Fatalf("EventPart returned % x, want the payload's own bytes % x", got, cyr)
	}
	// And a high byte alone is never mistaken for the needle.
	if game.EventHasSpeaker(cyr) {
		t.Fatal("high bytes alone must not answer the file test")
	}
}
