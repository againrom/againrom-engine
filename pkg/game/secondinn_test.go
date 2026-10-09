package game

import (
	"maps"
	"math"
	"reflect"
	"slices"
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

// The stage bodies are selected by slot 768 for any town (R2-ENGINE-215).
// Stage 20 and unlisted stages have no body (R2-ENGINE-161); with every gate
// at its zero-bank value each body offers these lists in store order.
func TestSecondInnStageOptions(t *testing.T) {
	o := func(kind, topic, npc int) secondInnOption { return secondInnOption{kind: kind, topic: topic, npc: npc} }
	stage10 := []secondInnOption{o(0, 9, 207), o(0, 8, 2108), o(3, 10, 517)}
	full30 := []secondInnOption{o(3, 30, 22), o(3, 31, 2108), o(0, 39, 2110)}
	gated30 := []secondInnOption{o(3, 30, 22), o(0, 39, 2110)}
	full40 := []secondInnOption{o(3, 40, 22), o(0, 48, 2108), o(3, 41, 2015), o(3, 42, 2111), o(3, 43, 2004)}
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
		{1, 0, nil, nil},
		{2, 35, nil, nil},
		{2, 120, nil, nil},
		{2, 40, nil, full40},
		{3, 40, nil, full40},
		{2, 40, map[int]int32{937: 1, 939: -1}, []secondInnOption{o(3, 40, 22), o(0, 48, 2108), o(3, 42, 2111)}},
		{2, 50, nil, []secondInnOption{o(0, 49, 23), o(3, 53, 2019)}},
		{2, 50, map[int]int32{533: 1}, []secondInnOption{o(0, 49, 23), o(3, 51, 2), o(3, 53, 2019)}},
		{2, 60, nil, []secondInnOption{o(3, 61, 2006), o(3, 63, 2109)}},
		{2, 70, nil, []secondInnOption{o(3, 73, 2004), o(3, 72, 2108)}},
		{2, 80, nil, []secondInnOption{o(3, 82, 2009)}},
		{2, 80, map[int]int32{776: 1}, []secondInnOption{o(3, 81, 2010)}},
		{3, 60, nil, nil},
		{3, 60, map[int]int32{771: 1}, []secondInnOption{o(3, 70, 680), o(3, 71, 681)}},
		{3, 80, map[int]int32{771: 1, 536: 1}, []secondInnOption{o(3, 70, 680), o(3, 71, 681), o(3, 83, 677)}},
		{2, 90, nil, []secondInnOption{o(3, 90, 2004)}},
		{3, 90, nil, []secondInnOption{o(3, 91, 681), o(3, 92, 2003)}},
		{2, 100, nil, []secondInnOption{o(3, 100, 2006), o(3, 102, 2109)}},
		{2, 100, map[int]int32{987: 1}, []secondInnOption{o(3, 100, 2006), o(3, 102, 2109), o(3, 103, 2005)}},
		{3, 100, nil, []secondInnOption{o(3, 101, 681)}},
		{2, 110, nil, []secondInnOption{o(3, 110, 2006)}},
		{3, 110, nil, nil},
		{1, 110, nil, nil},
	} {
		if got := innAt(tc.town, tc.stage, tc.slots).innOptions(); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("town %d stage %d %v: %v want %v", tc.town, tc.stage, tc.slots, got, tc.want)
		}
	}
}

// Each stage-body store with its predicates held is offered, and with any
// one predicate inverted is not: current record ID, exact stage, slots that
// must be zero (inverted to 1 and to -1) and slots that must be nonzero
// (R2-ENGINE-216, R2-ENGINE-223..R2-ENGINE-228).
func TestSecondInnStageGates(t *testing.T) {
	for _, row := range []struct {
		stages        []int32
		town          int // 0: any current record
		exact         int32
		kind, topic   int
		npc           int
		zero, nonzero []int
	}{
		{stages: []int32{30}, kind: 3, topic: 31, npc: 2108, zero: []int{927}},
		{stages: []int32{40}, kind: 3, topic: 40, npc: 22},
		{stages: []int32{40}, kind: 0, topic: 48, npc: 2108},
		{stages: []int32{40}, kind: 3, topic: 41, npc: 2015, zero: []int{937}},
		{stages: []int32{40}, kind: 3, topic: 42, npc: 2111, zero: []int{938}},
		{stages: []int32{40}, kind: 3, topic: 43, npc: 2004, zero: []int{939}},
		{stages: []int32{50}, kind: 0, topic: 49, npc: 23},
		{stages: []int32{50}, kind: 3, topic: 51, npc: 2, zero: []int{947}, nonzero: []int{533}},
		{stages: []int32{50}, kind: 3, topic: 53, npc: 2019, zero: []int{949}},
		{stages: []int32{60, 70, 80}, town: 3, kind: 3, topic: 70, npc: 680, zero: []int{966}, nonzero: []int{771}},
		{stages: []int32{60, 70, 80}, town: 3, kind: 3, topic: 71, npc: 681, zero: []int{966, 967}, nonzero: []int{771}},
		{stages: []int32{60, 70, 80}, town: 3, exact: 80, kind: 3, topic: 83, npc: 677, zero: []int{979}, nonzero: []int{536}},
		{stages: []int32{60, 70, 80}, town: 2, exact: 60, kind: 3, topic: 61, npc: 2006, zero: []int{957}},
		{stages: []int32{60, 70, 80}, town: 2, exact: 60, kind: 3, topic: 63, npc: 2109, zero: []int{959}},
		{stages: []int32{60, 70, 80}, town: 2, exact: 70, kind: 3, topic: 73, npc: 2004, zero: []int{969}},
		{stages: []int32{60, 70, 80}, town: 2, exact: 70, kind: 3, topic: 72, npc: 2108, zero: []int{968}},
		{stages: []int32{60, 70, 80}, town: 2, exact: 80, kind: 3, topic: 81, npc: 2010, zero: []int{977}, nonzero: []int{776}},
		{stages: []int32{60, 70, 80}, town: 2, exact: 80, kind: 3, topic: 82, npc: 2009, zero: []int{776, 978}},
		{stages: []int32{90}, town: 2, kind: 3, topic: 90, npc: 2004},
		{stages: []int32{90}, town: 3, kind: 3, topic: 91, npc: 681, zero: []int{987}},
		{stages: []int32{90}, town: 3, kind: 3, topic: 92, npc: 2003, zero: []int{988}},
		{stages: []int32{100}, town: 2, kind: 3, topic: 100, npc: 2006},
		{stages: []int32{100}, town: 2, kind: 3, topic: 102, npc: 2109, zero: []int{998}},
		{stages: []int32{100}, town: 2, kind: 3, topic: 103, npc: 2005, zero: []int{999}, nonzero: []int{987}},
		{stages: []int32{100}, town: 3, kind: 3, topic: 101, npc: 681, zero: []int{997}},
		{stages: []int32{110}, town: 2, kind: 3, topic: 110, npc: 2006},
	} {
		want := secondInnOption{kind: row.kind, topic: row.topic, npc: row.npc}
		offered := func(town int, stage int32, slots map[int]int32) bool {
			for _, got := range innAt(town, stage, slots).innOptions() {
				if got == want {
					return true
				}
			}
			return false
		}
		for _, stage := range row.stages {
			towns := []int{row.town}
			if row.town == 0 {
				towns = []int{1, 2, 3}
			}
			held := map[int]int32{}
			for _, s := range row.nonzero {
				held[s] = 1
			}
			for _, town := range towns {
				admit := row.exact == 0 || row.exact == stage
				if got := offered(town, stage, held); got != admit {
					t.Errorf("%+v town %d stage %d held: offered %v", want, town, stage, got)
				}
				if !admit {
					continue
				}
				for _, s := range row.nonzero {
					inv := maps.Clone(held)
					inv[s] = 0
					if offered(town, stage, inv) {
						t.Errorf("%+v stage %d offered with slot %d zero", want, stage, s)
					}
				}
				for _, s := range row.zero {
					for _, v := range []int32{1, -1} {
						inv := maps.Clone(held)
						inv[s] = v
						if offered(town, stage, inv) {
							t.Errorf("%+v stage %d offered with slot %d=%d", want, stage, s, v)
						}
					}
				}
			}
			if row.town != 0 {
				for _, other := range []int{1, 2, 3, 4} {
					if other != row.town && offered(other, stage, held) {
						t.Errorf("%+v stage %d offered in town %d", want, stage, other)
					}
				}
			}
			for _, other := range []int32{20, stage - 10, stage + 10, stage + 1} {
				if !slices.Contains(row.stages, other) && offered(max(row.town, 2), other, held) {
					t.Errorf("%+v offered at stage %d", want, other)
				}
			}
		}
	}
}

// The fifteen fixed continuation stores and their gates (R2-ENGINE-217),
// after the stage body's own stores.
func TestSecondInnContinuationGates(t *testing.T) {
	npc5 := secondInnOption{kind: 0, topic: 79, npc: 5}
	npc675 := secondInnOption{kind: 0, topic: 78, npc: 675}
	topic62 := secondInnOption{kind: 3, topic: 62, npc: 22}
	f := func(topic int) secondInnOption { return secondInnOption{kind: 3, topic: topic, npc: 2022} }
	body := func(topic, npc int) secondInnOption { return secondInnOption{kind: 3, topic: topic, npc: npc} }
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
		{"other town", 2, 70, map[int]int32{774: 1, 770: 1}, []secondInnOption{body(73, 2004), body(72, 2108)}},
		{"family A zero/zero", 2, 20, map[int]int32{959: 1}, []secondInnOption{f(74)}},
		{"family A zero/nonzero", 2, 20, map[int]int32{959: 1, 781: 3}, []secondInnOption{f(75)}},
		{"family A nonzero/nonzero", 2, 20, map[int]int32{959: 1, 776: 1, 781: 1}, []secondInnOption{f(76)}},
		{"family A nonzero/zero", 2, 20, map[int]int32{959: 1, 776: -1}, []secondInnOption{f(77)}},
		{"family A blocked", 2, 20, map[int]int32{959: 1, 972: 1}, []secondInnOption{f(84)}},
		{"family B blocked", 2, 20, map[int]int32{970: 1, 981: 1}, []secondInnOption{f(93)}},
		{"family C blocked", 2, 20, map[int]int32{980: 1, 990: 1}, nil},
		{"families in town 3", 3, 20, map[int]int32{959: 1}, nil},
		{"topic 62 at 60", 2, 60, map[int]int32{949: 1}, []secondInnOption{body(61, 2006), body(63, 2109), topic62}},
		{"topic 62 below 60", 2, 50, map[int]int32{949: 1}, []secondInnOption{{kind: 0, topic: 49, npc: 23}}},
		{"topic 62 after 958", 2, 60, map[int]int32{949: 1, 958: 1}, []secondInnOption{body(61, 2006), body(63, 2109)}},
		{"order", 2, 60, map[int]int32{949: 1, 959: 1, 970: 0}, []secondInnOption{body(61, 2006), f(74), topic62}},
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
		{secondInnOption{kind: 0, topic: 48, npc: 2108}, nil, nil},
		{secondInnOption{kind: 0, topic: 49, npc: 23}, nil, nil},
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

// Every kind 3 stage-body topic other than 10 and NPC 22 topic 30 admits
// mission topic and stores no bank slot (R2-ENGINE-229); stage 768 is
// unchanged.
func TestSecondInnLaterTalksAdmitTheirTopic(t *testing.T) {
	topics := 0
	for _, stage := range []int32{40, 50, 60, 70, 80, 90, 100, 110} {
		for _, e := range secondInnStages[stage] {
			if e.kind != 3 {
				continue
			}
			topics++
			c := innAt(2, stage, nil)
			c.bank[533], c.bank[999] = -9, 4
			want := c.bank
			c.talkTo(e.secondInnOption)
			c.talkTo(e.secondInnOption)
			if c.bank != want || !reflect.DeepEqual(c.available, []secondLocation{{2, 2}, {1, e.topic}}) {
				t.Errorf("TALK %d topic %d: available %v", e.npc, e.topic, c.available)
			}
		}
	}
	// Stages 60, 70 and 80 share one body of nine stores; the other bodies
	// hold 14 kind 3 stores (R2-ENGINE-229 counts 23 distinct).
	if topics != 14+3*9 {
		t.Fatal("kind 3 stage stores", topics)
	}
}

// Leave30 adds ten to slot 768 as a DWORD; Leave31 and Leave32 keep it
// (R2-SESSION-111).
func TestSecondLeaveThirtiesStage(t *testing.T) {
	for _, s := range []int32{0, 10, 30, 40, 70, -10, 35, math.MaxInt32 - 5, math.MinInt32} {
		for _, n := range []int{30, 31, 32} {
			c := &secondCampaign{current: secondLocation{1, n}, available: []secondLocation{{1, n}}}
			var bank [1024]int32
			bank[768] = s
			c.completeBank(bank)
			want := s
			if n == 30 {
				want = int32(uint32(s) + 10)
			}
			if c.bank[768] != want {
				t.Errorf("Leave%d from %d: stage %d want %d", n, s, c.bank[768], want)
			}
		}
	}
}
