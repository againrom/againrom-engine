package game

import (
	"fmt"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// innOptions is EnterInn: the inn rows of the campaign's town catalog the
// current town, the stage in slot 768 and the bank admit, in row order,
// then the hire options of the hire bank (R2-ENGINE-215, R2-ENGINE-217,
// R2-ENGINE-221). It changes no state.
func (c *secondCampaign) innOptions() []innOffer {
	return innOffers(c.offerRows(), innState{town: c.current.id, stage: int(c.bank[768]), bank: &c.bank, hire: true})
}

// offerRows are the inn rows the campaign admits: the rows of the book its
// town screen composes, the shipped catalog's before a screen binds one.
func (c *secondCampaign) offerRows() []town.OfferRow {
	if c.offers != nil {
		return c.offers
	}
	return secondBook().offerRows()
}

// speakers is the inn's talk actors: options of kind 0 and 3, one per NPC
// key, each answering the first option with that key (R2-ENGINE-220,
// DIV-2635). Kinds 1 and 2 are hire actors instead.
func (c *secondCampaign) speakers() []innOffer {
	var out []innOffer
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
func (c *secondCampaign) talkTo(o innOffer) {
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
	mission := partyPlace{kind: placeMission, id: o.topic}
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

// secondTalkKey is the town text section NPC record npc speaks on topic.
func secondTalkKey(npc, topic int) string { return fmt.Sprintf("npc%dtalk%d", npc, topic) }

func boolInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// hires is the inn's hire actors: every option of kind 1 or 2, in option
// order (R2-ENGINE-220).
func (c *secondCampaign) hires() []innOffer {
	var out []innOffer
	for _, o := range c.innOptions() {
		if o.kind == 1 || o.kind == 2 {
			out = append(out, o)
		}
	}
	return out
}

// innSeats is the inn page's seats, the hire actors and then the talk
// actors (R2-ENGINE-497), and the number of hire actors.
func (c *secondCampaign) innSeats() ([]innOffer, int) {
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
func (c *secondCampaign) innWord(npc int) (innOffer, bool) {
	for _, o := range c.innOptions() {
		if o.npc == npc {
			if high, ok := c.innBits[npc]; ok {
				o.high = high
			}
			return o, true
		}
	}
	return innOffer{}, false
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
