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
	// talkCursor is the campaign's town talk cursor, nil where the campaign
	// has none to turn; endTalk is what the talk's end does to the campaign.
	talkCursor() *talkCursor
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

func (t *townScreen) roomName() string { return t.campaignTown().room() }
func (t *townScreen) displayRoom() string {
	if t.roomName() == "talk" {
		return townRoomName(t.dialogueRoom())
	}
	return t.roomName()
}
func (t *townScreen) state() *secondCampaign { return t.campaignTown().bank() }
