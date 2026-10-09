package game

import "fmt"

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
// DIV-2635). Kinds 1 and 2 are not talk actors (DIV-2636).
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

// talkTo applies a TALK: kind 3 admits mission topic and kind 0 changes
// nothing (R2-ENGINE-219, R2-ENGINE-229, R2-ENGINE-230). Topic 10, the
// initial record, also stores slot 769 (R2-SESSION-110); NPC 22 topic 30
// stores slots 533 and 553.
func (c *secondCampaign) talkTo(o secondInnOption) {
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
