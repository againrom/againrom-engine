package game

import "fmt"

// secondInnOption is one packed EnterInn word: kind, topic and NPC key, and
// the high bit the dynamic tail sets (R2-ENGINE-216, R2-ENGINE-221).
type secondInnOption struct {
	kind, topic, npc int
	high             bool
}

func (o secondInnOption) speaker() bool { return o.kind == 0 || o.kind == 3 }

// secondInnStages holds the stage case bodies whose stores the claims
// publish: stage 10 (R2-ENGINE-161) and stage 30 (R2-ENGINE-216). The
// stage-30 NPC 2108 entry is gated on slot 927 in innOptions. Stages 40 to
// 110 offer nothing of their own (DIV-2634).
var secondInnStages = map[int32][]secondInnOption{
	10: {{kind: 0, topic: 9, npc: 207}, {kind: 0, topic: 8, npc: 2108}, {kind: 3, topic: 10, npc: 517}},
	30: {{kind: 3, topic: 30, npc: 22}, {kind: 3, topic: 31, npc: 2108}, {kind: 0, topic: 39, npc: 2110}},
}

// innOptions is EnterInn: the stage case selected by slot 768 for any town,
// then the fixed continuation stores and the dynamic tail, in that order
// (R2-ENGINE-215, R2-ENGINE-217, R2-ENGINE-221). It changes no state.
func (c *secondCampaign) innOptions() []secondInnOption {
	b, cur := &c.bank, c.current.id
	var out []secondInnOption
	for _, o := range secondInnStages[b[768]] {
		if o.npc == 2108 && o.topic == 31 && b[927] != 0 {
			continue
		}
		out = append(out, o)
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
// nothing (R2-ENGINE-219). Topic 10, the initial record, also stores slot
// 769 (R2-SESSION-110); NPC 22 topic 30 stores slots 533 and 553.
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
