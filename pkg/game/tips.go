package game

import (
	"image"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// roomTipBuilder builds both games' town room popups from their tip rows.
type roomTipBuilder struct {
	in  *InstallResources
	pc  *PersistenceContext
	art *ui.TipPanelArt
}

// roomTipRow is room's tip row in town; nil for none.
func roomTipRow(in *InstallResources, town, room string) *ui.TipRow {
	return TipTable(in.profile().Edition()).RoomRow(town, room)
}

// entryText is an entry's text: empty, unread, while the tips option is off.
func (b roomTipBuilder) entryText(row *ui.TipRow) string {
	if b.pc.tipsOffNow() || row == nil {
		return ""
	}
	return b.read(&row.Text)
}

func (b roomTipBuilder) read(text *ui.TipText) string {
	if text == nil {
		return ""
	}
	var src entrySource
	if b.in.Archives != nil {
		src = b.in.Archives.Containers
	}
	return readTipText(src, *text, b.in.textCode())
}

func (b roomTipBuilder) panel(row *ui.TipRow, text string, rect image.Rectangle, revision uint64) ui.TipPanelView {
	return ui.TipPanelView{
		Revision:    revision,
		Rect:        rect,
		Text:        text,
		ToggleOn:    !b.pc.tipsOffNow(),
		CloseLabel:  b.in.Words.TipClose,
		ToggleLabel: b.in.Words.TipShowNext,
		Art:         b.art,
		Font:        b.font(row),
		TextInset:   TipTable(b.in.profile().Edition()).Panel(row).Inset,
	}
}

// view projects a room's popup at its row's rectangle. The bottom edge
// follows the text when the row asks for that, and grows when the engine's
// wrap would cut the text at the rectangle (DIV-2719).
func (b roomTipBuilder) view(row *ui.TipRow, text string, revision uint64) ui.TipPanelView {
	if row == nil {
		row = &ui.TipRow{}
	}
	v := b.panel(row, text, row.Rectangle(), revision)
	v.Owner = row.Owner()
	if row.Fit || !ui.TipPanelFits(v) {
		v.Rect = ui.TipPanelShrinkRect(v.Rect, v.Font, text)
	}
	return v
}

// font is the install's tip font, or the generator's text font where the
// row's panel style names it.
func (b roomTipBuilder) font(row *ui.TipRow) *text.Font {
	a := b.in.ChargenAssets
	if TipTable(b.in.profile().Edition()).Panel(row).Font == "text" && a != nil && a.Presentation != nil && a.Presentation.Font != nil {
		return a.Presentation.Font
	}
	return b.in.tipFont()
}

func (t *townScreen) tips() roomTipBuilder {
	return roomTipBuilder{in: t.in, pc: t.pc, art: t.art.tipPanel()}
}

// reader is the builder without art: reading a text loads no panel art.
func (t *townScreen) reader() roomTipBuilder { return roomTipBuilder{in: t.in, pc: t.pc} }

// roomTip is the town's one room popup in both games. A room entry builds it
// from the room's row (TOWN-516, R2-ENGINE-308); Close hides it until the
// next build (R2-ENGINE-310); a talk's end rereads its text in place. What a
// game adds is its tip table's Rooms rules and rows.
type roomTip struct {
	// of is the second-game campaign it was built in; nil in the first game.
	of       *secondCampaign
	room     string
	text     string
	closed   bool
	revision uint64
	// away is set while the composed town is hidden.
	away bool
	// second is set once the row's second text replaced the first
	// (TOWN-517); each build clears it.
	second bool
}

// roomTipRules is the game's room popup lifecycle data.
func (t *townScreen) roomTipRules() ui.TipRooms {
	if t == nil || t.in == nil {
		return ui.TipRooms{}
	}
	if table := TipTable(t.in.profile().Edition()); table != nil {
		return table.Rooms
	}
	return ui.TipRooms{}
}

// roomTipRow is room's row of the game's tip table in the current town, nil
// when the room shows no tip.
func (t *townScreen) roomTipRow(room string) *ui.TipRow {
	town := ""
	if d := t.townDescription(); d != nil {
		town = d.Town
	}
	return roomTipRow(t.in, town, room)
}

// loadRoomTip is a room entry's build: a new popup for room, its text read
// once while the tips option is on.
func (t *townScreen) loadRoomTip(room string) {
	if t == nil || t.sess == nil {
		return
	}
	t.tip = roomTip{of: t.state(), room: room, revision: t.tip.revision + 1}
	if _, ok := t.composedTown(); !ok || t.pc == nil {
		return
	}
	t.tip.text = t.reader().entryText(t.roomTipRow(room))
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

// rereadTip retexts room's built popup from its row without building
// another: the popup keeps its closed state, revision and second latch.
func (t *townScreen) rereadTip(room string) {
	if t == nil || t.sess == nil || t.tip.room != room || t.tip.text == "" {
		return
	}
	var text *ui.TipText
	if row := t.roomTipRow(room); row != nil {
		text = &row.Text
	}
	t.tip.text = t.reader().read(text)
}

// resetRoomTip forgets the popup's campaign, closed state and latches at a
// new or loaded game; the text and revision stay for the room just entered.
func (t *townScreen) resetRoomTip() {
	t.tip = roomTip{room: t.tip.room, text: t.tip.text, revision: t.tip.revision}
}

// tipShown reports that room's popup exists: its entry built a text, under
// the current campaign, and the player has not closed it.
func (t *townScreen) tipShown(room string) bool {
	if t == nil || t.sess == nil || t.pc == nil {
		return false
	}
	p := t.tip
	return p.of == t.state() && p.room == room && !p.closed && p.text != ""
}

// roomTipView projects room's popup, nothing when it is not shown.
func (t *townScreen) roomTipView(room string) ui.TipPanelView {
	if !t.tipShown(room) {
		return ui.TipPanelView{}
	}
	return t.tips().view(t.roomTipRow(room), t.tip.text, t.tip.revision)
}

// advanceShopSecondTip retexts the built popup with its row's second text
// once per build, while the shop's table holds an item (TOWN-517). It reads
// no TipsMode: the popup has a text only if the option was on at the build,
// and a closed popup still takes the second text.
func (t *townScreen) advanceShopSecondTip() {
	if t == nil || t.sess == nil || t.tip.text == "" || t.tip.second {
		return
	}
	row := t.roomTipRow(t.tip.room)
	if row == nil || row.Second.File == "" {
		return
	}
	if shop := t.sess.Shop; shop == nil || len(shop.Table()) == 0 {
		return
	}
	t.tip.second = true
	t.tip.text = t.reader().read(&row.Second)
}

// CloseTip dismisses the built popup until the next build.
func (t *townScreen) CloseTip() {
	if t == nil {
		return
	}
	t.tip.closed = true
}

// ResumeTown is the game menu's return to the town: where the game's rules
// say so, the shown room's popup is built again.
func (t *townScreen) ResumeTown() {
	if t != nil && t.roomTipRules().Resume && t.tip.room == t.roomName() {
		t.loadRoomTip(t.tip.room)
	}
}

// TownSquareEntryRevision is the popup's revision where the game repaints
// the square at a build, else 0.
func (t *townScreen) TownSquareEntryRevision() uint64 {
	if t == nil || !t.roomTipRules().EntryPaint {
		return 0
	}
	return t.tip.revision
}

// tipFont is the tip panel's own font2: the smaller of the two atlases
// chargenAssets.go resolves at construction, reused here rather than loaded
// again. A carried LoadChargenAssets error, or a nil Presentation, falls
// back to f.Font (font1) on the same degrade every other cosmetic source in
// this front end already has — a missing font2 hides no room, it only
// draws its tip text larger than shipped.
func (f *FrontEnd) tipFont() *text.Font {
	if f == nil {
		return nil
	}
	return f.InstallResources.tipFont()
}

// tipFont is the font tip and card panels draw in: the generation screen's
// text font when its description says the tips draw in it, else the
// install's.
func (in *InstallResources) tipFont() *text.Font {
	if a := in.ChargenAssets; a != nil && a.Presentation != nil && a.Presentation.TipFont != nil {
		return a.Presentation.TipFont
	}
	return in.Font.Value()
}

// missionCardFont is the font the mission screen's character card and
// column texts draw in: the generation screen's text font, in which both
// originals draw the statistics sheet (R2-ASSET-148), else the install's.
func (in *InstallResources) missionCardFont() *text.Font {
	if a := in.ChargenAssets; a != nil && a.Presentation != nil && a.Presentation.Font != nil {
		return a.Presentation.Font
	}
	return in.Font.Value()
}

// hoverStyle is the generator description's hover style, which every hover
// box of the game draws in, and the text font it names; nil where the
// description names none or the generator did not load.
func (in *InstallResources) hoverStyle() (*ui.GeneratorHover, *text.Font) {
	if a := in.ChargenAssets; a != nil && a.Presentation != nil && a.Presentation.Layout != nil {
		return a.Presentation.Layout.Hover, a.Presentation.Font
	}
	return nil, nil
}

// HoverHelp is the game's hover look, every hover box's outside the
// generator; nil where the generator names none.
func (in *InstallResources) HoverHelp() *ui.HoverLook {
	return ui.NewHoverLook(in.hoverStyle())
}

// ToggleTips flips the permanent suppression (spec behaviour 4). Satisfies
// ui.TipScreen. It does not touch the built popup or its text: TOWN-186's
// own gate is tested at a room's own build, so the effect reaches the next
// entry, not the panel already showing.
func (t *townScreen) ToggleTips() {
	if t == nil || t.sess == nil {
		return
	}
	t.pc.setTipsOff(!t.pc.tipsOffNow())
}
