package game

import (
	"image"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// The tip widget's own text for the four rooms besides the shop (1018 spec
// behaviours 2, 3, 4, 5).
const (
	// SchoolTipPath is main/text/tips/training.txt (TOWN-021, discharged by
	// TOWN-188).
	SchoolTipPath = mainPrefix + "text/tips/training.txt"
	// TavernTipPath is main/text/tips/inn.txt (TOWN-015).
	TavernTipPath = mainPrefix + "text/tips/inn.txt"
	// ChargenFighterTipPath and ChargenMageTipPath are chrgen1f.txt and
	// chrgen1m.txt. THE LETTERS ARE CLASS, NOT SEX (contract's "The 1f/1m
	// trap"; TOWN-187, corrected at research 8990406, an ancestor of this
	// story's pin): chrgen1f.txt is the fighter's text and chrgen1m.txt the
	// mage's, established from both files' own shipped English wording, not
	// from the filename convention graphics.res's equipment/ tree
	// coincidentally shares.
	ChargenFighterTipPath = mainPrefix + "text/tips/chrgen1f.txt"
	ChargenMageTipPath    = mainPrefix + "text/tips/chrgen1m.txt"
	// ChargenDetailTipPath is chrgen2.txt, TOWN-187's second popup text
	// (1022 spec B5): the original replaces the first popup's text with this
	// node, in place, once the detailed page opens, with no class branch.
	ChargenDetailTipPath = mainPrefix + "text/tips/chrgen2.txt"
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
		return &t.townTip, rom1Town.Tip.Text
	case roomShop:
		return &t.shopTip, ShopTip1Path
	case roomSchool:
		return &t.schoolTip, SchoolTipPath
	case roomTavern:
		return &t.tavernTip, TavernTipPath
	}
	return nil, ""
}

func (t *townScreen) readTipText(dst *string, addr string) {
	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}
	*dst, _ = ReadShopTip(src, addr)
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

// tipFont is the font tip and card panels draw in: the generation screen's,
// else the install's.
func (in *InstallResources) tipFont() *text.Font {
	if in.ChargenAssets != nil && in.ChargenAssets.Presentation != nil && in.ChargenAssets.Presentation.Font != nil {
		return in.ChargenAssets.Presentation.Font
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
