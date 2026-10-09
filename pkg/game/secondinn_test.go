package game

import (
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

func innAt(town int, stage int32, slots map[int]int32) *secondCampaign {
	c := &secondCampaign{current: secondLocation{2, town}, available: []secondLocation{{2, town}}}
	c.bank[768] = stage
	for slot, value := range slots {
		c.bank[slot] = value
	}
	return c
}

// The stage bodies are selected by slot 768 for any town (R2-ENGINE-215):
// stage 10 (R2-ENGINE-161) and stage 30 with its slot 927 gate
// (R2-ENGINE-216); other stages offer nothing of their own.
func TestSecondInnStageOptions(t *testing.T) {
	stage10 := []secondInnOption{{kind: 0, topic: 9, npc: 207}, {kind: 0, topic: 8, npc: 2108}, {kind: 3, topic: 10, npc: 517}}
	full30 := []secondInnOption{{kind: 3, topic: 30, npc: 22}, {kind: 3, topic: 31, npc: 2108}, {kind: 0, topic: 39, npc: 2110}}
	gated30 := []secondInnOption{{kind: 3, topic: 30, npc: 22}, {kind: 0, topic: 39, npc: 2110}}
	for _, tc := range []struct {
		town  int
		stage int32
		slots map[int]int32
		want  []secondInnOption
	}{
		{1, 10, nil, stage10},
		{2, 10, nil, stage10},
		{2, 30, nil, full30},
		{3, 30, nil, full30},
		{2, 30, map[int]int32{927: 1}, gated30},
		{2, 30, map[int]int32{927: -1}, gated30},
		{2, 20, nil, nil},
		{2, 40, nil, nil},
		{1, 0, nil, nil},
		{2, 110, nil, nil},
	} {
		if got := innAt(tc.town, tc.stage, tc.slots).innOptions(); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("town %d stage %d %v: %v want %v", tc.town, tc.stage, tc.slots, got, tc.want)
		}
	}
}

// The fifteen fixed continuation stores and their gates (R2-ENGINE-217).
func TestSecondInnContinuationGates(t *testing.T) {
	npc5 := secondInnOption{kind: 0, topic: 79, npc: 5}
	npc675 := secondInnOption{kind: 0, topic: 78, npc: 675}
	topic62 := secondInnOption{kind: 3, topic: 62, npc: 22}
	f := func(topic int) secondInnOption { return secondInnOption{kind: 3, topic: topic, npc: 2022} }
	for _, tc := range []struct {
		name  string
		town  int
		stage int32
		slots map[int]int32
		want  []secondInnOption
	}{
		{"npc5 and 675", 3, 70, map[int]int32{774: 1, 770: 1}, []secondInnOption{npc5, npc675}},
		{"stage 60 is not above 60", 3, 60, map[int]int32{774: 1, 770: 1}, nil},
		{"signed stage", 3, -70, map[int]int32{774: 1, 770: 1}, nil},
		{"774 must equal 1", 3, 70, map[int]int32{774: 2, 770: 1}, []secondInnOption{npc675}},
		{"770 must equal 1", 3, 70, map[int]int32{774: 1, 770: -1}, []secondInnOption{npc5}},
		{"other town", 2, 70, map[int]int32{774: 1, 770: 1}, nil},
		{"family A zero/zero", 2, 20, map[int]int32{959: 1}, []secondInnOption{f(74)}},
		{"family A zero/nonzero", 2, 20, map[int]int32{959: 1, 781: 3}, []secondInnOption{f(75)}},
		{"family A nonzero/nonzero", 2, 20, map[int]int32{959: 1, 776: 1, 781: 1}, []secondInnOption{f(76)}},
		{"family A nonzero/zero", 2, 20, map[int]int32{959: 1, 776: -1}, []secondInnOption{f(77)}},
		{"family A blocked", 2, 20, map[int]int32{959: 1, 972: 1}, []secondInnOption{f(84)}},
		{"family B blocked", 2, 20, map[int]int32{970: 1, 981: 1}, []secondInnOption{f(93)}},
		{"family C blocked", 2, 20, map[int]int32{980: 1, 990: 1}, nil},
		{"families in town 3", 3, 20, map[int]int32{959: 1}, nil},
		{"topic 62 at 60", 2, 60, map[int]int32{949: 1}, []secondInnOption{topic62}},
		{"topic 62 below 60", 2, 50, map[int]int32{949: 1}, nil},
		{"topic 62 after 958", 2, 60, map[int]int32{949: 1, 958: 1}, nil},
		{"order", 2, 60, map[int]int32{949: 1, 959: 1, 970: 0}, []secondInnOption{f(74), topic62}},
	} {
		if got := innAt(tc.town, tc.stage, tc.slots).innOptions(); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %v want %v", tc.name, got, tc.want)
		}
	}
}

// The dynamic tail offers kind 1 or 2, ID i+1, when slot 552+i names the
// current town, with the high bit from signed slot 512+i (R2-ENGINE-221).
func TestSecondInnDynamicTail(t *testing.T) {
	c := innAt(2, 20, map[int]int32{552: 2, 532: 1, 553: 2, 533: 2, 513: 4, 554: 2, 534: 3, 555: 1, 535: 1, 556: 2, 536: 1, 516: -5})
	want := []secondInnOption{{kind: 1, npc: 1}, {kind: 2, npc: 2, high: true}, {kind: 1, npc: 5}}
	if got := c.innOptions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("tail %v want %v", got, want)
	}
	if got := c.speakers(); len(got) != 0 {
		t.Fatal("tail options became talk actors", got)
	}
	// After Leave20 town 2 offers kind 1, ID 1 (R2-ENGINE-221).
	l := &secondCampaign{current: secondLocation{1, 20}, available: []secondLocation{{1, 20}}}
	l.bank[768] = 20
	l.completeBank(l.bank)
	l.current = secondLocation{2, 2}
	tail := l.innOptions()
	if tail[len(tail)-1] != (secondInnOption{kind: 1, npc: 1}) {
		t.Fatal("post-Leave20 tail", tail)
	}
}

// TALK: kind 3 admits mission topic; NPC 22 topic 30 stores 533 and 553;
// topic 10 stores 769; kind 0 changes nothing (R2-ENGINE-219,
// R2-SESSION-110).
func TestSecondInnTalkEffects(t *testing.T) {
	for _, tc := range []struct {
		o     secondInnOption
		add   []secondLocation
		slots map[int]int32
	}{
		{secondInnOption{kind: 3, topic: 30, npc: 22}, []secondLocation{{1, 30}}, map[int]int32{533: 1, 553: 2}},
		{secondInnOption{kind: 3, topic: 31, npc: 2108}, []secondLocation{{1, 31}}, nil},
		{secondInnOption{kind: 3, topic: 10, npc: 517}, []secondLocation{{1, 10}}, map[int]int32{769: 1}},
		{secondInnOption{kind: 3, topic: 30, npc: 23}, []secondLocation{{1, 30}}, nil},
		{secondInnOption{kind: 0, topic: 39, npc: 2110}, nil, nil},
		{secondInnOption{kind: 0, topic: 30, npc: 22}, nil, nil},
		{secondInnOption{kind: 1, topic: 0, npc: 1}, nil, nil},
	} {
		c := innAt(2, 30, nil)
		c.bank[533], c.bank[553] = -9, -9
		want := c.bank
		for slot, value := range tc.slots {
			want[slot] = value
		}
		c.talkTo(tc.o)
		c.talkTo(tc.o)
		if c.bank != want || !reflect.DeepEqual(c.available, append([]secondLocation{{2, 2}}, tc.add...)) {
			t.Errorf("%+v: available %v", tc.o, c.available)
		}
	}
}

// The inn lists one TALK row per kind 0 or 3 NPC key in option order; a row
// runs the first option with its key and shows its npc%dtalk%d section.
func TestSecondInnScreenTalksToEachSpeaker(t *testing.T) {
	text := synth.File{Path: "text/town.txt", Data: []byte("#npc517talk10\r\n<NPC=517,PART=1>\r\nFirst page.\r\n#npc2108talk31\r\n<NPC=2108,PART=1>\r\nThirty-one.\r\n#other\r\n")}
	f, app, screen := secondCampaignFixtureFiles(t, false, text)
	c := innAt(2, 30, nil)
	c.available = []secondLocation{{2, 2}, {1, 21}}
	f.Town.second = c
	app.SetTown(screen)
	screen.Choose(0)
	want := []ui.TownRow{{Text: "TALK 22", Choosable: true}, {Text: "TALK 2108", Choosable: true}, {Text: "TALK 2110", Choosable: true}, {Text: "GATES", Choosable: true}}
	if c.room != secondTownInn || !reflect.DeepEqual(screen.Rows(), want) {
		t.Fatal("inn rows", screen.Rows())
	}
	screen.Choose(1)
	if body, ok := screen.dialogueBody(); !ok || body == "" || !reflect.DeepEqual(c.available, []secondLocation{{2, 2}, {1, 21}, {1, 31}}) {
		t.Fatal("TALK 2108", c.available, body)
	}
	screen.AdvanceTownDialogue()
	screen.Choose(0)
	if _, ok := screen.dialogueBody(); ok || !reflect.DeepEqual(c.available, []secondLocation{{2, 2}, {1, 21}, {1, 31}, {1, 30}}) || c.bank[533] != 1 || c.bank[553] != 2 {
		t.Fatal("TALK 22 without a section", c.available)
	}
	screen.Choose(2)
	if len(c.available) != 4 {
		t.Fatal("kind 0 TALK changed availability")
	}
	if !screen.CanSave() {
		t.Fatal("quiet town 2 inn is not a save point")
	}
	screen.Choose(3)
	if c.current != (secondLocation{}) || c.room != secondTownSquare || len(c.available) != 4 {
		t.Fatal("GATES from the inn", c)
	}
}
