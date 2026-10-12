package game

import (
	"fmt"

	"againrom/pkg/ui"
)

// secondInnOption is one packed EnterInn word: kind, topic and NPC key, and
// the high bit the dynamic tail sets (R2-ENGINE-216, R2-ENGINE-221).
type secondInnOption struct {
	kind, topic, npc int
	high             bool
}

func (o secondInnOption) speaker() bool { return o.kind == 0 || o.kind == 3 }

// secondInnEntry is one stage-body store and the predicates that admit it:
// the current record ID (0 for any), the exact stage (0 for any), and bank
// slots that must be zero or nonzero.
type secondInnEntry struct {
	secondInnOption
	town          int
	stage         int32
	zero, nonzero []int
}

func (e secondInnEntry) admits(b *[1024]int32, town int) bool {
	if e.town != 0 && e.town != town || e.stage != 0 && e.stage != b[768] {
		return false
	}
	for _, s := range e.zero {
		if b[s] != 0 {
			return false
		}
	}
	for _, s := range e.nonzero {
		if b[s] == 0 {
			return false
		}
	}
	return true
}

func innStore(kind, topic, npc int) secondInnEntry {
	return secondInnEntry{secondInnOption: secondInnOption{kind: kind, topic: topic, npc: npc}}
}

func (e secondInnEntry) at(town int, stage int32) secondInnEntry {
	e.town, e.stage = town, stage
	return e
}

func (e secondInnEntry) when(zero []int, nonzero ...int) secondInnEntry {
	e.zero, e.nonzero = zero, nonzero
	return e
}

// secondInnShared is the one body stages 60, 70 and 80 select
// (R2-ENGINE-225).
var secondInnShared = []secondInnEntry{
	innStore(3, 70, 680).at(3, 0).when([]int{966}, 771),
	innStore(3, 71, 681).at(3, 0).when([]int{966, 967}, 771),
	innStore(3, 83, 677).at(3, 80).when([]int{979}, 536),
	innStore(3, 61, 2006).at(2, 60).when([]int{957}),
	innStore(3, 63, 2109).at(2, 60).when([]int{959}),
	innStore(3, 73, 2004).at(2, 70).when([]int{969}),
	innStore(3, 72, 2108).at(2, 70).when([]int{968}),
	innStore(3, 81, 2010).at(2, 80).when([]int{977}, 776),
	innStore(3, 82, 2009).at(2, 80).when([]int{776, 978}),
}

// secondInnStages holds every stage case body of EnterInn, its stores in
// order with their gates: stage 10 (R2-ENGINE-161), 30 (R2-ENGINE-216),
// 40 (R2-ENGINE-223), 50 (R2-ENGINE-224), the shared 60/70/80 body
// (R2-ENGINE-225), 90 (R2-ENGINE-226), 100 (R2-ENGINE-227) and 110
// (R2-ENGINE-228). Stage 20 and every unlisted stage have no body.
var secondInnStages = map[int32][]secondInnEntry{
	10: {innStore(0, 9, 207), innStore(0, 8, 2108), innStore(3, 10, 517)},
	30: {innStore(3, 30, 22), innStore(3, 31, 2108).when([]int{927}), innStore(0, 39, 2110)},
	40: {innStore(3, 40, 22), innStore(0, 48, 2108), innStore(3, 41, 2015).when([]int{937}),
		innStore(3, 42, 2111).when([]int{938}), innStore(3, 43, 2004).when([]int{939})},
	50: {innStore(0, 49, 23), innStore(3, 51, 2).when([]int{947}, 533), innStore(3, 53, 2019).when([]int{949})},
	60: secondInnShared,
	70: secondInnShared,
	80: secondInnShared,
	90: {innStore(3, 90, 2004).at(2, 0), innStore(3, 91, 681).at(3, 0).when([]int{987}),
		innStore(3, 92, 2003).at(3, 0).when([]int{988})},
	100: {innStore(3, 100, 2006).at(2, 0), innStore(3, 102, 2109).at(2, 0).when([]int{998}),
		innStore(3, 103, 2005).at(2, 0).when([]int{999}, 987), innStore(3, 101, 681).at(3, 0).when([]int{997})},
	110: {innStore(3, 110, 2006).at(2, 0)},
}

// innOptions is EnterInn: the stage case selected by slot 768 for any town,
// then the fixed continuation stores and the dynamic tail, in that order
// (R2-ENGINE-215, R2-ENGINE-217, R2-ENGINE-221). It changes no state.
func (c *secondCampaign) innOptions() []secondInnOption {
	b, cur := &c.bank, c.current.id
	var out []secondInnOption
	for _, e := range secondInnStages[b[768]] {
		if e.admits(b, cur) {
			out = append(out, e.secondInnOption)
		}
	}
	stage := b[768]
	if cur == 3 && stage > 60 && b[774] == 1 {
		out = append(out, secondInnOption{kind: 0, topic: 79, npc: 5})
	}
	if cur == 3 && stage > 60 && b[770] == 1 {
		out = append(out, secondInnOption{kind: 0, topic: 78, npc: 675})
	}
	if cur == 2 {
		some := func(from, to int) bool {
			for i := from; i <= to; i++ {
				if b[i] != 0 {
					return true
				}
			}
			return false
		}
		pick := 0
		switch {
		case b[776] == 0 && b[781] != 0:
			pick = 1
		case b[776] != 0 && b[781] != 0:
			pick = 2
		case b[776] != 0 && b[781] == 0:
			pick = 3
		}
		for _, family := range []struct {
			open  bool
			first int
		}{
			{b[959] != 0 && !some(970, 973), 74},
			{some(970, 973) && !some(980, 983), 84},
			{some(980, 983) && !some(989, 992), 93},
		} {
			if family.open {
				out = append(out, secondInnOption{kind: 3, topic: family.first + pick, npc: 2022})
			}
		}
		if stage >= 60 && b[958] == 0 && b[949] != 0 {
			out = append(out, secondInnOption{kind: 3, topic: 62, npc: 22})
		}
	}
	for i := 0; i < 20; i++ {
		if int(b[552+i]) == cur && (b[532+i] == 1 || b[532+i] == 2) {
			out = append(out, secondInnOption{kind: int(b[532+i]), topic: 0, npc: i + 1, high: b[512+i] > 0})
		}
	}
	return out
}

// speakers is the inn's talk actors: options of kind 0 and 3, one per NPC
// key, each answering the first option with that key (R2-ENGINE-220,
// DIV-2635). Kinds 1 and 2 are hire actors instead.
func (c *secondCampaign) speakers() []secondInnOption {
	var out []secondInnOption
	seen := map[int]bool{}
	for _, o := range c.innOptions() {
		if o.speaker() && !seen[o.npc] {
			seen[o.npc] = true
			out = append(out, o)
		}
	}
	return out
}

// talkTo is TalkTo (R2-ENGINE-493). Kind 0 has two arms: NPC 5 at any
// topic stores 536=1, 556=3 and 774=2; NPC 675 topic 78 stores 770=2 and
// adds class 4 to record 3's category 3 (R2-ENGINE-331). Kind 1 toggles the
// hire selection slot 511+ID (R2-SESSION-191). Kind 3 admits mission topic
// (R2-ENGINE-219, R2-ENGINE-229); topic 10 also stores 769 (R2-SESSION-110)
// and NPC 22 topic 30 stores 533 and 553. Kinds 2 and 4..7 store nothing,
// and the high bit is not read.
func (c *secondCampaign) talkTo(o secondInnOption) {
	switch {
	case o.kind == 0 && o.npc == 5:
		c.bank[536], c.bank[556], c.bank[774] = 1, 3, 2
	case o.kind == 0 && o.npc == 675 && o.topic == 78:
		c.bank[770] = 2
		c.shop.addClass4()
	case o.kind == 1 && o.npc >= 1 && o.npc <= secondHireSlots:
		slot := &c.bank[secondHireSelection+o.npc-1]
		*slot = boolInt32(*slot == 0)
	}
	if o.kind != 3 {
		return
	}
	mission := secondLocation{kind: 1, id: o.topic}
	if !c.has(mission) {
		c.available = append(c.available, mission)
	}
	if o.topic == 10 {
		c.bank[769] = 1
	}
	if o.npc == 22 && o.topic == 30 {
		c.bank[533], c.bank[553] = 1, 2
	}
}

func secondTalkKey(o secondInnOption) string { return fmt.Sprintf("npc%dtalk%d", o.npc, o.topic) }

// The hire bank: hire h (1..20) has selection slot 511+h, availability slot
// 531+h and town slot 551+h (R2-ENGINE-221, R2-ENGINE-380).
const (
	secondHireSlots        = 20
	secondHireSelection    = 512
	secondHireAvailability = 532
)

func boolInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// hires is the inn's hire actors: every option of kind 1 or 2, in option
// order (R2-ENGINE-220).
func (c *secondCampaign) hires() []secondInnOption {
	var out []secondInnOption
	for _, o := range c.innOptions() {
		if o.kind == 1 || o.kind == 2 {
			out = append(out, o)
		}
	}
	return out
}

// innSeats is the inn page's seats, the hire actors and then the talk
// actors (R2-ENGINE-497), and the number of hire actors.
func (c *secondCampaign) innSeats() ([]secondInnOption, int) {
	hires := c.hires()
	return append(hires, c.speakers()...), len(hires)
}

// innRows is one list row per inn seat: HIRE or FIRE for a hire actor as
// its caption reads (R2-ENGINE-246), TALK for a talk actor.
func (c *secondCampaign) innRows() []ui.TownRow {
	seats, hires := c.innSeats()
	rows := make([]ui.TownRow, len(seats))
	for i, o := range seats {
		verb := "TALK"
		if i < hires {
			verb = "HIRE"
			if c.hired(o.npc) {
				verb = "FIRE"
			}
		}
		rows[i] = ui.TownRow{Text: fmt.Sprintf("%s %d", verb, o.npc), Choosable: true}
	}
	return rows
}

// enterInn starts a visit of the inn page: the page's copy of the high
// bits is the bank's again.
func (c *secondCampaign) enterInn() { c.innBits = nil }

// innWord is the copied option an inn action on low ID npc takes: the first
// option with that low ID (R2-ENGINE-498), its high bit as Hire and Fire
// last left it during this visit of the page (R2-SESSION-191).
func (c *secondCampaign) innWord(npc int) (secondInnOption, bool) {
	for _, o := range c.innOptions() {
		if o.npc == npc {
			if high, ok := c.innBits[npc]; ok {
				o.high = high
			}
			return o, true
		}
	}
	return secondInnOption{}, false
}

// hired answers whether the seat with low ID npc shows the hired mark and
// the Fire caption: its copied word has the high bit (R2-ENGINE-497).
func (c *secondCampaign) hired(npc int) bool {
	o, ok := c.innWord(npc)
	return ok && o.high
}

// hireOrFire is the action on a hire seat (R2-ENGINE-495, R2-ENGINE-496):
// Fire when the copied word has the high bit, which clears it and sends the
// word to TalkTo; otherwise Hire, which on a kind 1 word answers the accept
// section, sets the high bit and sends the word to TalkTo, and on any other
// word answers the reject section only. The section key is "" for Fire.
func (c *secondCampaign) hireOrFire(npc int) string {
	o, ok := c.innWord(npc)
	if !ok {
		return ""
	}
	if c.innBits == nil {
		c.innBits = map[int]bool{}
	}
	if o.high {
		o.high, c.innBits[npc] = false, false
		c.talkTo(o)
		return ""
	}
	if o.kind != 1 {
		return fmt.Sprintf("npc%dreject%d", npc, c.bank[768])
	}
	o.high, c.innBits[npc] = true, true
	c.talkTo(o)
	return fmt.Sprintf("npc%daccept%d", npc, c.bank[768])
}
