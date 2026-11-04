package game

import "againrom/pkg/town"

type townCampaignAdapter interface {
	book() *townBook
	room() string
	setRoom(string)
	townID() int
	ready() bool
	present() bool
	talking() bool
	// talkSeated reports whether the campaign has a talk to turn, so that
	// the town's one talk cursor is offered; endTalk is what the talk's end
	// does to the campaign.
	talkSeated() bool
	endTalk()
	condition(string) bool
	value(string) int
	savePoint() bool
	leave()
	bank() *secondCampaign
	chapter() int
	unavailableTowns() []int
	startCampaign() error
}

func (t *townScreen) campaignTown() townCampaignAdapter {
	if t.adapter == nil {
		t.adapter = chapterTownCampaign{t}
	}
	return t.adapter
}

func (t *townScreen) hostRules() town.HostSpec {
	if d := t.townDescription(); d != nil && d.Host != nil {
		return *d.Host
	}
	if t.defaults != nil && t.defaults.Host != nil {
		return *t.defaults.Host
	}
	return *ROM1TownDescription().Host
}

// talkPage is the town's one talk cursor, held by the session's town; nil
// where the screen has no town.
func (t *townScreen) talkPage() *talkCursor {
	if t.sess == nil || t.sess.Town == nil {
		return nil
	}
	return &t.sess.Town.talk
}

func (t *townScreen) roomName() string { return t.campaignTown().room() }
func (t *townScreen) displayRoom() string {
	if t.roomName() == "talk" {
		return townRoomName(t.dialogueRoom())
	}
	return t.roomName()
}
func (t *townScreen) state() *secondCampaign { return t.campaignTown().bank() }
