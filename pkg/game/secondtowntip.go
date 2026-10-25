package game

import "againrom/pkg/ui"

// secondRoomTip is the one room popup; each entry rebuilds it (R2-ENGINE-310).
type secondRoomTip struct {
	// of is the campaign it was built in.
	of       *secondCampaign
	room     string
	text     string
	closed   bool
	revision uint64
	// away is set while the composed town is hidden.
	away bool
}

func (t *secondCampaignScreen) tips() roomTipBuilder {
	b := roomTipBuilder{in: t.install, pc: t.options}
	if t.tipArt != nil {
		b.art = t.tipArt()
	}
	return b
}

// loadRoomTip is a room entry's build (R2-ENGINE-308).
func (t *secondCampaignScreen) loadRoomTip(room string) {
	c := t.state()
	t.tip = secondRoomTip{of: c, room: room, revision: t.tip.revision + 1}
	d, ok := t.composedTown()
	if !ok || t.options == nil {
		return
	}
	t.tip.text = t.tips().entryText(roomTipRow(t.install, d.Town, room))
}

// arriveRoomTip builds the shown room's popup when the town is shown after
// hidden or under another campaign.
func (t *secondCampaignScreen) arriveRoomTip(shown bool, room string) {
	if !shown {
		t.tip.away = true
		return
	}
	if t.tip.away || t.tip.of != t.state() {
		t.loadRoomTip(room)
	}
}

func (t *secondCampaignScreen) roomTipView(town, room string) ui.TipPanelView {
	p := t.tip
	if p.of == nil || p.of != t.state() || p.room != room || p.closed || p.text == "" || t.options == nil {
		return ui.TipPanelView{}
	}
	return t.tips().view(roomTipRow(t.install, town, room), p.text, p.revision)
}

func (t *secondCampaignScreen) CloseTip() { t.tip.closed = true }

// ToggleTips leaves the open popup (R2-ENGINE-311).
func (t *secondCampaignScreen) ToggleTips() {
	if t.options != nil {
		t.options.setTipsOff(!t.options.tipsOffNow())
	}
}
