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

// loadRoomTip is a room entry's build (R2-ENGINE-308).
func (t *townScreen) loadRoomTip(room string) {
	c := t.state()
	t.tip = secondRoomTip{of: c, room: room, revision: t.tip.revision + 1}
	d, ok := t.composedTown()
	if !ok || t.pc == nil {
		return
	}
	t.tip.text = t.tips().entryText(roomTipRow(t.in, d.Town, room))
}

// arriveRoomTip builds the shown room's popup when the town is shown after
// hidden or under another campaign.
func (t *townScreen) arriveRoomTip(shown bool, room string) {
	if !shown {
		t.tip.away = true
		return
	}
	if t.tip.away || t.tip.of != t.state() {
		t.loadRoomTip(room)
	}
}

func (t *townScreen) entryRoomTipView(town, room string) ui.TipPanelView {
	p := t.tip
	if p.of == nil || p.of != t.state() || p.room != room || p.closed || p.text == "" || t.pc == nil {
		return ui.TipPanelView{}
	}
	return t.tips().view(roomTipRow(t.in, town, room), p.text, p.revision)
}
