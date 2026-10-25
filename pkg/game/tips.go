package game

import (
	"image"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// loadTip constructs one room popup and tests the global option once.
func (t *townScreen) loadTip(room townRoom) {
	if t == nil || t.sess == nil {
		return
	}
	dst, text := t.tipTextSource(room)
	if dst == nil {
		return
	}
	t.tipRevision++
	t.tipClosed[room] = false
	if room == roomShop {
		t.shopTipSecond = false
	}
	if t.pc.tipsOffNow() {
		*dst = ""
		return
	}
	t.readTipText(dst, text)
}

// rereadTip updates an existing popup's content without constructing another.
func (t *townScreen) rereadTip(room townRoom) {
	if t == nil || t.sess == nil {
		return
	}
	dst, text := t.tipTextSource(room)
	if dst != nil && *dst != "" {
		t.readTipText(dst, text)
	}
}

func (t *townScreen) tipTextSource(room townRoom) (*string, *ui.TipText) {
	var dst *string
	switch room {
	case roomSquare:
		dst = &t.townTip
	case roomShop:
		dst = &t.shopTip
	case roomSchool:
		dst = &t.schoolTip
	case roomTavern:
		dst = &t.tavernTip
	}
	if row := t.roomTip(room); row != nil {
		return dst, &row.Text
	}
	return dst, nil
}

// roomTip is the room's row of the game's tip table, nil when the room shows
// no tip.
func (t *townScreen) roomTip(room townRoom) *ui.TipRow {
	if t == nil {
		return nil
	}
	return TipTable(t.profile().Edition()).Row("room-enter", townRoomName(room), "")
}

// roomTipView projects a room's popup at its row's rectangle. The bottom edge
// follows the text when the row asks for that, and grows when the engine's
// wrap would cut the text at the rectangle (DIV-2719).
func (t *townScreen) roomTipView(room townRoom, text string) ui.TipPanelView {
	row := t.roomTip(room)
	if row == nil {
		row = &ui.TipRow{}
	}
	v := t.tipView(room, text, row.Rectangle())
	v.Owner = row.Owner()
	if row.Fit || !ui.TipPanelFits(v) {
		v.Rect = ui.TipPanelShrinkRect(v.Rect, t.in.tipFont(), text)
	}
	return v
}

// advanceShopSecondTip retexts the open shop popup with its row's second
// text once per shop activation, when the table holds an item
// (TOWN-517). It reads no TipsMode: the popup exists only if the flag was set
// at activation.
func (t *townScreen) advanceShopSecondTip() {
	if t == nil || t.sess == nil || t.shopTip == "" || t.shopTipSecond {
		return
	}
	if shop := t.sess.Shop; shop == nil || len(shop.Table()) == 0 {
		return
	}
	row := t.roomTip(roomShop)
	if row == nil || row.Second.File == "" {
		return
	}
	t.shopTipSecond = true
	t.readTipText(&t.shopTip, &row.Second)
}

func (t *townScreen) readTipText(dst *string, text *ui.TipText) {
	if text == nil {
		*dst = ""
		return
	}
	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}
	*dst = readTipText(src, *text, t.in.textCode())
}

// tipView projects the constructed popup and the live global checkbox flag.
func (t *townScreen) tipView(room townRoom, text string, rect image.Rectangle) ui.TipPanelView {
	if t == nil || t.sess == nil || t.tipClosed[room] {
		return ui.TipPanelView{}
	}
	return ui.TipPanelView{
		Revision:    t.tipRevision,
		Rect:        rect,
		Text:        text,
		ToggleOn:    !t.pc.tipsOffNow(),
		CloseLabel:  t.in.Words.TipClose,
		ToggleLabel: t.in.Words.TipShowNext,
		Art:         t.art.tipPanel(),
		Font:        t.in.tipFont(),
	}
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

// hoverStyle is the generator description's hover style, which every hover
// box of the game draws in, and the text font it names; nil where the
// description names none or the generator did not load.
func (in *InstallResources) hoverStyle() (*ui.GeneratorHover, *text.Font) {
	if a := in.ChargenAssets; a != nil && a.Presentation != nil && a.Presentation.Layout != nil {
		return a.Presentation.Layout.Hover, a.Presentation.Font
	}
	return nil, nil
}

// CloseTip dismisses the current room's constructed popup.
func (t *townScreen) CloseTip() {
	if t == nil || int(t.room) >= len(t.tipClosed) {
		return
	}
	t.tipClosed[t.room] = true
}

// ToggleTips flips the permanent suppression (spec behaviour 4). Satisfies
// ui.TipScreen. It does not touch tipClosed or any already-loaded tip text:
// TOWN-186's own gate is tested at a room's own construction, so the effect
// reaches the next entry, not the panel already showing.
func (t *townScreen) ToggleTips() {
	if t == nil || t.sess == nil {
		return
	}
	t.pc.setTipsOff(!t.pc.tipsOffNow())
}

func (t *townScreen) ResumeTown() { t.loadTip(t.room) }
