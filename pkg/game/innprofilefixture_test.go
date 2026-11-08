package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// firstInnProfile and secondInnProfile are the inn profiles the first and
// the second game's town hosts name.
func firstInnProfile() innProfile { return hostInnProfile(ROM1TownDescription().Host.Inn) }
func secondInnProfile() innProfile {
	return hostInnProfile(townDescriptions["rom2-town1"].Host.Inn)
}

// The inn page hire button's Hire and Fire caption slots (R2-ENGINE-246).
const (
	secondInnHireCaption = 258
	secondInnFireCaption = 259
)

// hireCaption is the caption slot the inn page's hire button row shows
// over the selected seat, nil without one.
func (v roomPageView) hireCaption() *int {
	r := v.d.Room(v.room)
	if r == nil {
		return nil
	}
	for _, b := range r.Buttons {
		if b.Name == "hire" {
			hire, hired := v.hireSeat()
			return innButtonSlot(b, hire, hired)
		}
	}
	return nil
}

// innSeats is the second game's inn page seat words over campaign c, as
// its town screen seats them, and the number of hire seats.
func (c *secondCampaign) innSeats() ([]innOffer, int) {
	seats, hires := innSeats(secondInnProfile(), nil, c.innOptions(), c.hires())
	var out []innOffer
	for _, s := range seats {
		out = append(out, s.word)
	}
	return out, hires
}

// speakers is the second game's inn talk seat words over campaign c.
func (c *secondCampaign) speakers() []innOffer {
	var out []innOffer
	for _, s := range innRoster(secondInnProfile(), nil, c.innOptions()) {
		out = append(out, s.word)
	}
	return out
}

// innRows is the second game's inn page rows over campaign c, without the
// exit row, with Sleep word sleep.
func (c *secondCampaign) innRows(sleep string) []ui.TownRow {
	seats, hires := innSeats(secondInnProfile(), nil, c.innOptions(), c.hires())
	return innRowTexts(innPageRows(secondInnProfile().rows, sleep, seats, hires, c.hired, false))
}

// resolveTavernTalk answers which object InnNPC value npc stands for under
// profile p, over party, stock and npc's record, as tavernTalkObject does.
func resolveTavernTalk(p innProfile, npc int, rec data.NPCFace, known bool, party []mapload.PartyMember, stock []tavernStockActor) tavernTalkObject {
	return innActor(p, npc, innActorInput{party: party, stock: stock, record: rec, known: known}).object
}

// setTownDialogue opens section key's conversation and its speaker resolver.
func (t *townScreen) setTownDialogue(key string, payload []byte) {
	t.startInnTalk(secondInnProfile(), innTalkRequest{key: key, payload: payload, loaded: true, plain: true})
}

// tavernVisibleOffers are the first game's tavern talk seats as offers.
func (t *townScreen) tavernVisibleOffers() []TownOffer {
	seats, hires := t.innPageSeats()
	out := make([]TownOffer, 0, len(seats)-hires)
	for _, s := range seats[hires:] {
		out = append(out, s.offer())
	}
	return out
}

// TavernRoster is the tavern's talk list in the chapter's registry order: every
// live offer, with a kept speaker in the place of each pair the live arrays no
// longer hold.
//
// A pair leaves both arrays when its mission is accepted (SAV-CAMPAIGN-081) and
// a talk cell follows its array position (SAV-1112); the speaker is kept so the
// conversation can be heard again (DIV-1529). A kept speaker's Index is -1 minus
// its registry position, which Take refuses, so hearing it again queues nothing
// (DIV-1530).
//
// NOTHING IS STORED. The live arrays are what a save holds, so a game the
// engine played, one it reloaded and one an original save carries all list
// the same speakers, and the chapter's next advance replaces the registry rows
// with the next chapter's.
//
// A restored chapter's registry is the form its live arrays were written
// against: a save made without the towns mod that now changes the chapter
// holds layer 0's arrays, so a LOAD adds no mod row (chapterForms, innRegistry).
func (t *Town) TavernRoster() []TownOffer {
	seats := innRoster(firstInnProfile(), t, nil)
	var out []TownOffer
	if seats != nil {
		out = make([]TownOffer, len(seats))
	}
	for i, s := range seats {
		out[i] = s.offer()
	}
	return out
}
