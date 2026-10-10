package game

import (
	"image"

	"againrom/pkg/render/text"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// loadTip constructs one room popup and tests the global option once.
func (t *townScreen) loadTip(room townRoom) {
	if t == nil || t.sess == nil {
		return
	}
	dst, addr := t.tipTextSource(room)
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
	t.readTipText(dst, addr)
}

// rereadTip updates an existing popup's content without constructing another.
func (t *townScreen) rereadTip(room townRoom) {
	if t == nil || t.sess == nil {
		return
	}
	dst, addr := t.tipTextSource(room)
	if dst != nil && *dst != "" {
		t.readTipText(dst, addr)
	}
}

func (t *townScreen) tipTextSource(room townRoom) (*string, string) {
	switch room {
	case roomSquare:
		return &t.townTip, t.townDescription().Tip.Text
	case roomShop:
		return &t.shopTip, t.roomTip(room).Text
	case roomSchool:
		return &t.schoolTip, t.roomTip(room).Text
	case roomTavern:
		return &t.tavernTip, t.roomTip(room).Text
	}
	return nil, ""
}

// roomTip answers a room's tip as the profile's descriptions give it: the
// square's from the town description, the others from the room description.
func (t *townScreen) roomTip(room townRoom) town.TipSpec {
	if room == roomSquare {
		if d := t.townDescription(); d != nil {
			return d.Tip
		}
		return town.TipSpec{}
	}
	return roomTipIn(t.roomDescription(), room)
}

func roomTipIn(d *town.Description, room townRoom) town.TipSpec {
	if d == nil {
		return town.TipSpec{}
	}
	name := townRoomName(room)
	for i := range d.Rooms {
		if r := &d.Rooms[i]; r.Name == name && r.Tip != nil {
			return *r.Tip
		}
	}
	return town.TipSpec{}
}

// roomTipView projects a room's popup at the description's rectangle. The
// bottom edge follows the text when the description asks for that, and grows
// when the engine's wrap would cut the text at the rectangle (DIV-2719).
func (t *townScreen) roomTipView(room townRoom, text string) ui.TipPanelView {
	spec := t.roomTip(room)
	v := t.tipView(room, text, spec.Rect.Rectangle())
	if spec.Fit || !ui.TipPanelFits(v) {
		v.Rect = ui.TipPanelShrinkRect(v.Rect, t.in.tipFont(), text)
	}
	return v
}

// advanceShopSecondTip retexts the open shop popup with the description's
// second text once per shop activation, when the table holds an item
// (TOWN-517). It reads no TipsMode: the popup exists only if the flag was set
// at activation.
func (t *townScreen) advanceShopSecondTip() {
	if t == nil || t.sess == nil || t.shopTip == "" || t.shopTipSecond {
		return
	}
	if shop := t.sess.Shop; shop == nil || len(shop.Table()) == 0 {
		return
	}
	second := t.roomTip(roomShop).Second
	if second == "" {
		return
	}
	t.shopTipSecond = true
	t.readTipText(&t.shopTip, second)
}

func (t *townScreen) readTipText(dst *string, addr string) {
	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}
	*dst, _ = ReadShopTip(src, addr, t.in.textCode())
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
