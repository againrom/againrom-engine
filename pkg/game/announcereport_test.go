package game

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The no-window runner's announcement record.
//
// Every payload here is written in this file and no install is opened
// (golden rule 2).

// announceWorld is a scripted world whose one trigger fires and runs an
// instant-2 node, with the raise list a compiled map would carry beside it.
func announceWorld(t *testing.T, event int32) (*sim.World, []mapload.ScriptRaise) {
	t.Helper()
	w := missionWorld(t,
		[]sim.ScriptInstant{{Op: sim.ScriptInstantMessage, Args: [10]int32{event}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	return w, []mapload.ScriptRaise{{Latch: 3, Event: event}}
}

// announceDrive is a PlayWorld watching that world, stepped until the trigger
// has fired, with the given event text shipped at mission 7's own address.
func announceDrive(t *testing.T, event int32, payload string,
	party []mapload.PartyMember, faces map[int32]data.NPCFace) *PlayWorld {
	t.Helper()
	w, raises := announceWorld(t, event)
	src := missionSource{}
	if payload != "" {
		src[missionEvent(t, 7, int(event))] = []byte(payload)
	}
	p := &PlayWorld{World: w, Party: []sim.EntityID{0}}
	p.watchAnnouncements(&Mission{Number: 7, World: w, Raises: raises, Party: party}, src, faces)
	for i := 0; i < 32; i++ {
		p.step(nil)
	}
	return p
}

func TestTheDriveRecordsTheAnnouncementsItRaised(t *testing.T) {
	p := announceDrive(t, 4, "<part=1>hello<part=2>again", nil, nil)
	if len(p.announced) != 1 {
		t.Fatalf("%d announcement(s) recorded, want 1: %+v", len(p.announced), p.announced)
	}
	got := p.announced[0]
	if got.Message != 4 || !got.Shipped || !got.Shown || got.Reserved {
		t.Fatalf("announcement = %+v, want message 4, shipped and shown", got)
	}
	if want := []string{"hello", "again"}; !reflect.DeepEqual(got.Parts, want) {
		t.Fatalf("parts = %q, want %q", got.Parts, want)
	}
	if got.Tick == 0 {
		t.Fatal("the announcement carries no tick")
	}
}

func TestTheRecordedPartIsTheOneThisDrivesHeroReceives(t *testing.T) {
	const payload = "<part=1,iamfemale>she<part=1,iammale>he"
	for _, tc := range []struct {
		name  string
		party []mapload.PartyMember
		want  string
	}{
		{"a woman mage", womanMage(), "she"},
		{"a man fighter", manFighter(), "he"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := announceDrive(t, 4, payload, tc.party, nil)
			if len(p.announced) != 1 || len(p.announced[0].Parts) != 1 ||
				p.announced[0].Parts[0] != tc.want {
				t.Fatalf("recorded %+v, want the single part %q", p.announced, tc.want)
			}
		})
	}
}

func TestAnUnshippedTextIsRecordedAsSilence(t *testing.T) {
	p := announceDrive(t, 4, "", nil, nil)
	if len(p.announced) != 1 {
		t.Fatalf("%d announcement(s), want the raise recorded: %+v", len(p.announced), p.announced)
	}
	if got := p.announced[0]; got.Shipped || got.Shown || len(got.Parts) != 0 {
		t.Fatalf("announcement = %+v, want it recorded as unshipped and unshown", got)
	}
}

// TestAShippedTextThatRefusesThisHeroIsRecordedShippedAndUnshown separates the
// two silences (AC-4): the file is there and was read, and no part survived.
func TestAShippedTextThatRefusesThisHeroIsRecordedShippedAndUnshown(t *testing.T) {
	p := announceDrive(t, 4, "<part=1,iamfemale>she", manFighter(), nil)
	if len(p.announced) != 1 {
		t.Fatalf("%d announcement(s), want 1: %+v", len(p.announced), p.announced)
	}
	if got := p.announced[0]; !got.Shipped || got.Shown || len(got.Parts) != 0 {
		t.Fatalf("announcement = %+v, want shipped and unshown", got)
	}
}

func TestTheReservedNumberIsRecordedWithoutReadingAText(t *testing.T) {
	p := announceDrive(t, ReservedMessageNumber, "", nil, nil)
	if len(p.announced) != 1 {
		t.Fatalf("%d announcement(s), want 1: %+v", len(p.announced), p.announced)
	}
	if got := p.announced[0]; !got.Reserved || got.Shipped || got.Shown {
		t.Fatalf("announcement = %+v, want it recorded as reserved and nothing else", got)
	}
}

func TestTheReportedTextIsDecodedFromTheInstallsCodePage(t *testing.T) {
	word := []byte{0x8f, 0xe0, 0xa8, 0xa2, 0xa5, 0xe2} // a CP866 word
	p := announceDrive(t, 4, "<part=1>"+string(word), nil, nil)
	if len(p.announced) != 1 || len(p.announced[0].Parts) != 1 {
		t.Fatalf("recorded %+v, want one part", p.announced)
	}
	got := p.announced[0].Parts[0]
	if got == string(word) {
		t.Fatal("the part was reported as raw bytes; the report is JSON and must be UTF-8")
	}
	// The expected string is spelled by code point so this file stays ASCII
	// (golden rule 2).
	if want := string([]rune{0x41f, 0x440, 0x438, 0x432, 0x435, 0x442}); got != want {
		t.Fatalf("part = %q, want the decoded word", got)
	}
	// And the same payload reaches the DRAW path unconverted: EventPart is the
	// panel's own reader and it returns the file's own bytes.
	raw, ok := EventPart([]byte("<part=1>"+string(word)), 1, EventAudience{})
	if !ok || raw != string(word) {
		t.Fatalf("EventPart returned % x, want the payload's own bytes % x", raw, word)
	}
}

func TestTheAnnouncementPassWritesNothingToTheWorld(t *testing.T) {
	watchedW, raises := announceWorld(t, 4)
	plainW, _ := announceWorld(t, 4)

	watched := &PlayWorld{World: watchedW, Party: []sim.EntityID{0}}
	watched.watchAnnouncements(&Mission{Number: 7, World: watchedW, Raises: raises},
		missionSource{missionEvent(t, 7, 4): []byte("<part=1>hello")}, nil)
	plain := &PlayWorld{World: plainW, Party: []sim.EntityID{0}}

	for i := 0; i < 32; i++ {
		watched.step(nil)
		plain.step(nil)
		if a, b := watched.World.Hash(), plain.World.Hash(); a != b {
			t.Fatalf("step %d: watched hashes %#016x, unwatched %#016x", i, a, b)
		}
		if a, b := watched.World.Tick(), plain.World.Tick(); a != b {
			t.Fatalf("step %d: tick %d vs %d", i, a, b)
		}
	}
	if len(watched.announced) == 0 {
		t.Fatal("the watched drive recorded nothing, so the comparison proves nothing")
	}
	if len(plain.announced) != 0 {
		t.Fatalf("the unwatched drive recorded %+v", plain.announced)
	}
}

func TestASyntheticWorldRaisesNothing(t *testing.T) {
	p := scenarioWorld(t, scenarioFighter(0, sim.SelfSlot, 4, 4))
	for i := 0; i < 8; i++ {
		p.step(nil)
	}
	if len(p.announced) != 0 {
		t.Fatalf("a synthetic world recorded %+v", p.announced)
	}
}

func TestAWorldAssertionReadsTheAnnouncementsRaised(t *testing.T) {
	build := func(t *testing.T) *PlayWorld {
		return announceDrive(t, 4, "<part=1>hello", nil, nil)
	}
	for _, tc := range []struct {
		name string
		file string
		fail string
	}{
		{"the list the drive raised", `{"announced": [4]}`, ""},
		{"a number it never raised", `{"announced": [5]}`, "want [5]"},
		{"a longer list than it raised", `{"announced": [4, 4]}`, "want [4 4]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := build(t)
			s := writeScenario(t, `{
  "version": 2, "stage": "mission", "mission": 7,
  "steps": [{"command": "assert_world", "world": `+tc.file+`}]
}`)
			err := RunPlayScenario(p, s, new(bytes.Buffer), new(bytes.Buffer))
			if tc.fail == "" {
				if err != nil {
					t.Fatalf("RunPlayScenario = %v, want the assertion to hold", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.fail) {
				t.Fatalf("RunPlayScenario = %v, want a failure naming %q", err, tc.fail)
			}
		})
	}
}

// TestAWorldStatingOnlyAnnouncementsIsAWorldStatingSomething is the validator's
// half: announced alone is a complete world assertion.
func TestAWorldStatingOnlyAnnouncementsIsAWorldStatingSomething(t *testing.T) {
	a := &HeadlessWorldAssertion{Announced: []int32{4}}
	if err := a.validate(); err != nil {
		t.Fatalf("validate() = %v, want a world stating announcements to be accepted", err)
	}
	if err := (&HeadlessWorldAssertion{}).validate(); err == nil {
		t.Fatal("a world stating nothing was accepted")
	}
}

func TestTheReportCarriesTheAnnouncementsOnItsReportSteps(t *testing.T) {
	p := announceDrive(t, 4, "<part=1>hello", nil, nil)
	full := p.worldState(nil, true)
	if len(full.Announcements) != 1 || full.Announcements[0].Message != 4 {
		t.Fatalf("a report step carries %+v, want the one announcement", full.Announcements)
	}
	summary := p.worldState(nil, false)
	if summary.Announcements != nil {
		t.Fatalf("a summary step carries %+v, want none", summary.Announcements)
	}
}

// TestTwoNodesRaisingOneNumberProduceTwoAnnouncements is AC-5.
//
// The shipped campaign authors 254 instant-2 nodes over 242 distinct (map,
// number) pairs, so eleven pairs are authored more than once and one three
// times (TRIG-MSGCORPUS-049). NOTHING DE-DUPLICATES THEM: a number already
// raised is raised again, because the duplicate is a second authored node and
// not a repeat of the first. A recorder that collapsed the two would report
// eleven fewer announcements than the campaign contains, and would do it
// silently.
func TestTwoNodesRaisingOneNumberProduceTwoAnnouncements(t *testing.T) {
	w := missionWorld(t,
		[]sim.ScriptInstant{
			{Op: sim.ScriptInstantMessage, Args: [10]int32{4}},
			{Op: sim.ScriptInstantMessage, Args: [10]int32{4}},
		},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0, 1)})
	src := missionSource{missionEvent(t, 7, 4): []byte("<part=1>hello")}
	p := &PlayWorld{World: w, Party: []sim.EntityID{0}}
	p.watchAnnouncements(&Mission{
		World: w, Number: 7,
		Raises: []mapload.ScriptRaise{{Latch: 3, Event: 4}, {Latch: 3, Event: 4}},
	}, src, nil)
	for i := 0; i < 32; i++ {
		p.step(nil)
	}
	if len(p.announced) != 2 {
		t.Fatalf("%d announcement(s) recorded, want 2: %+v", len(p.announced), p.announced)
	}
	for i, a := range p.announced {
		if a.Message != 4 || !a.Shipped || !a.Shown {
			t.Fatalf("announcement %d = %+v, want message 4, shipped and shown", i, a)
		}
	}
}
