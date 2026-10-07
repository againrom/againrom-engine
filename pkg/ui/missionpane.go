package ui

import (
	"image"

	"againrom/pkg/render/text"
)

const characterPaneSeamW = 16

// The mission screen's character pane: the same widget the town screens
// compose, in the mission's own right column.
//
// ONE OBJECT, TWO PARENTS. `TOWN-353` (High) establishes that the original has
// one instance of this widget, re-parented between the mission frame and the
// shop view, with the six rectangles and the per-pixel slot map identical on
// both screens and only the gates differing. This build had TWO
// implementations of it — `pkg/ui/townshell.go`'s DrawTownCharacterRegion for
// the town and shop, and a separate doll box in `pkg/ui/inventory.go` for the
// mission — and the corner controls are what the split was hiding: four of the
// six rectangles existed on neither screen, and the two that existed were
// shop-only.
//
// WHAT THE MISSION DREW BEFORE, measured at the shipped 640x480: the id-7 slot
// at `characterPanelBoxRect` drew the statistics card alone, with no mode of
// its own, and the doll drew in a SEPARATE authored box below it at
// `dollBoxRect`, which `rightColumnBox` refuses at 640x480 and at 800x600
// because 488+260 exceeds both frames. So at every resolution the owner plays
// at, the mission had no doll at all. The pane's own two presentation modes
// give it one.
//
// THE MODE IS THE DOLL SWITCH. `TOWN-351` (High) reads `this+0x70` as the
// pane's presentation mode, forced to 1 by the constructor, inverted by rect
// C's own `0x412` arm and read four times by the paint routine; `L11936`
// skips the statistics text when it is non-zero. hudPanelDoll is stored
// inverted so SHOWN is the zero value (hudtoggles.go), so a fresh Viewer
// starts on the doll exactly as a freshly constructed widget does, and
// hudShown(hudPanelDoll) IS `this+0x70 != 0` with no second field to keep in
// step. Rect C, the Tab key (`TOWN-355`, `AI-KEY-125`) and the D binding all
// flip the one flag.

// SetCharacterPaneArt hands the viewer the pane's two mode-switched shipped
// bodies and the nine shipped corner bitmaps.
//
// IT IS THE FRONT END'S OWN PUSH, on SetSpellbook's and SetCommandPanelArt's
// shape: `pkg/ui` opens no archive, so the tier that can resolves
// `humanbackr.bmp`/`humanbackl.bmp`, `textbackr.bmp`/`textbackl.bmp` and the
// eight corner nodes once and pushes them here. A viewer holding none composes
// the authored fill and outline DrawTownCharacterRegion drew before story
// 1021, so a hand-built fixture and a missing node both still draw a pane.
//
// It bumps panelSerial, which is the panel cache's own "the layout changed"
// signal (panel.go): the composed picture is a function of this art, and a
// push that did not drop the cache would leave the previous body on screen
// until the subject next moved.
func (v *Viewer) SetCharacterPaneArt(figure, stats TownPane, corners *CharacterPaneCornerArt) {
	v.paneFigureArt, v.paneStatsArt, v.paneCornerArt = figure, stats, corners
	v.panelPic, v.panelSerial = nil, v.panelSerial+1
}

func (v *Viewer) SetCharacterPaneFillerArt(art TownPane) {
	if art.Body == nil {
		v.paneFillerArt, v.fillerImg = nil, nil
		return
	}
	bodySize := art.Body.Bounds().Size()
	seamW := 0
	if art.Seam != nil {
		seamW = art.Seam.Bounds().Dx()
	}
	pic := image.NewRGBA(image.Rect(0, 0, seamW+bodySize.X, bodySize.Y))
	drawTownPane(pic, art,
		image.Rect(seamW, 0, seamW+bodySize.X, bodySize.Y),
		image.Rect(0, 0, seamW, bodySize.Y))
	v.paneFillerArt, v.fillerImg = pic, nil
}

// characterPaneRect is where the mission's pane stands in WINDOW PIXELS, and
// whether one stands anywhere at all. It is `SHOP-FIGURE-041`'s own id-7 slot,
// which `hud.go`'s characterPanelBoxRect already placed.
func (v *Viewer) characterPaneRect() (image.Rectangle, bool) {
	return characterPanelBoxRect(image.Pt(v.frameW, v.frameH))
}

// characterPaneModeFlag resolves TOWN-346's own `flag` from this frame's
// pane in its own ABSOLUTE screen rectangle — characterPaneModeFlag's own
// doc explains why that rect and no other.
func (v *Viewer) characterPaneModeFlag() bool {
	rect, ok := v.characterPaneRect()
	if !ok {
		return false
	}
	return characterPaneModeFlag(rect, CharacterPaneMission, v.frameH)
}

// characterPaneStatistics is the interactive pane's presentation mode,
// statistics when true. See this file's header for why it is hudPanelDoll's
// own switch: the ONE character-panel box trades between the figure and the
// statistics card under rect C / hudPanelDoll on every shipped screen
// (`SHOP-FIGURE-041`, `TOWN-353`), and characterPaneModeFlag's own doc
// explains what its own predicate gates instead (the fourth column child's
// own readout text, not how many boxes this one has).
func (v *Viewer) characterPaneStatistics() bool {
	return !v.hudShown(hudPanelDoll)
}

// heldItemBox is where the pane takes an item released on it, and whether the
// selection has a pane to take it. It is the pane's own rectangle in EITHER
// mode: `TOWN-348` has the release handler test the held item before any
// rectangle, and `TOWN-351` names only the paint routine and rect C's toggle
// as readers of the mode, so a pack item dropped on the statistics card is
// worn as one dropped on the figure is. dollBox is the figure's own question
// and answers false in statistics mode.
func (v *Viewer) heldItemBox() (image.Rectangle, bool) {
	if _, ok := v.dollSubject(); !ok {
		return image.Rectangle{}, false
	}
	return v.characterPaneRect()
}

// characterPaneView is the projection this frame's pane is composed from and
// its hit tests are answered from — the shop's own shopCharacterView for the
// mission.
//
// THE SESSION IS CharacterPaneMission, `TOWN-352`'s own value for the mission
// frame, so rects A, B, C and F are live and the party picker's D and E are
// gated off (`1 & 0x226 == 0`). That is the original's behaviour and it is a
// change from what this build did: the two picker rectangles answered on every
// screen that drew them, which `DIV-217` recorded.
//
// PaneRect IS THE LOCAL BOX AND NOT THE SCREEN RECTANGLE. The picture is
// composed into its own (160+characterPaneSeamW)x242 image and blitted at
// the slot's own corner minus the seam width (panelPresent, panel.go), so
// every panel-relative expression in characterpane.go is evaluated against
// (characterPaneSeamW,0)-(160+characterPaneSeamW,242) here: the body keeps
// its own 160x242 footprint and the seam occupies the columns to its left,
// on the SAME canvas rather than a second blit. The screen-coordinate
// answers the hit tests need are characterPaneScreenView's, below, and are
// unaffected — they still read the pane's own 160-wide slot, never the seam.
func (v *Viewer) characterPaneView(local bool) TownCharacterView {
	rect, ok := v.characterPaneRect()
	if !ok {
		return TownCharacterView{}
	}
	pane := rect
	if local {
		pane = image.Rect(characterPaneSeamW, 0, rect.Dx()+characterPaneSeamW, rect.Dy())
	}
	view := TownCharacterView{
		Session:    CharacterPaneMission,
		ScreenH:    v.frameH,
		ModeFlag:   v.characterPaneModeFlag(),
		PaneRect:   pane,
		Statistics: v.characterPaneStatistics(),
		PackOpen:   v.hudShown(hudPanelPack),
		BookOpen:   v.hudShown(hudPanelBook),
		CornerArt:  v.paneCornerArt,
		FigurePane: v.paneFigureArt,
		StatsPane:  v.paneStatsArt,
		Font:       v.font,
		CardFont:   v.cardFont(),
		CardLayout: &v.panelLayout,
	}
	if ref, ok := v.hoverInspection(); ok {
		view.Subject, view.HasSubject = v.inspectionPanel(ref)
		view.Figure = v.inspectionFigure(ref)
		return view
	}
	selected := presentSelected(v.sel, v.entities)
	present := v.visiblePanelSelection(selected)
	switch len(present) {
	case 0:
		// The figure pane stays empty and the separate card below stays the
		// empty statistics page. With nothing selected both state the zero
		// lines (`TEXT-UI-047`); a selection whose first unit the fog hides is
		// not nothing selected, so it states none.
		view.Statistics = false
		if len(selected) == 0 {
			view.SelectionStatus, _ = SelectionStatusLines(v.words, 0)
		}
	case 1:
		s := v.panelSubjectFromPresent(present)
		view.Subject, view.HasSubject = s, true
	default:
		if status, ok := SelectionStatusLines(v.words, len(present)); ok {
			view.SelectionStatus = status
		}
	}
	if src, ok := v.dollSubject(); ok && len(present) > 0 {
		view.Figure = dollPicture(src)
	}
	return view
}

// characterPaneScreenView is characterPaneView in the frame's own coordinates,
// for the hit tests and for the name-row coverage test.
func (v *Viewer) characterPaneScreenView() TownCharacterView { return v.characterPaneView(false) }

// characterPaneCornerAt is which of the six corners a window pixel reaches,
// in `TOWN-346`'s own test order.
//
// IT RETURNS EVERY MATCH AND NOT THE FIRST. `TOWN-355` establishes that rect A
// overlaps rect D over 27x32 pixels and rect E overlaps rect F over 25x31,
// with no early return between the six tests, so one click inside an overlap
// posts two different messages. On the mission screen D and E are gated off,
// so neither overlap has both members live and the slice is at most one long
// there today; it becomes two the moment a shop is opened from a mission
// (`CharacterPaneShopInMission`), which is the case `DIV-310` records.
func (v *Viewer) characterPaneCornerAt(x, y int) []CharacterPaneCorner {
	rect, ok := v.characterPaneRect()
	if !ok {
		return nil
	}
	return CharacterPaneCornersAt(rect, CharacterPaneMission, v.frameH, image.Pt(x, y))
}

// pressCharacterPaneCorner runs whichever corners a release at (x,y) reaches,
// in order, and reports whether any did.
//
// EACH CORNER'S EFFECT IS THIS BUILD'S OWN CONSUMER OF THE MESSAGE THE
// ORIGINAL POSTS, never a re-derivation of it:
//
//   - Rect A posts `0x40e`, which `SESS-VIEW-028` names as the map view's
//     `Inventory/IsOpen` open/close pair and `TOWN-356` gives the
//     `BackPackOp.bmp`/`BackPackCl.bmp` art for. Here that is hudPanelPack,
//     the carried pack's own bar.
//   - Rect B posts `0x40f`, the `SpellBook/IsOpen` pair with
//     `BookOpened.bmp`/`BookClosed.bmp` art. Here that is hudPanelBook.
//   - Rect C posts `0x412` and inverts the presentation mode.
//   - Rect F posts `0x416`, which `MENU-ESC-010` reads on the frame window's
//     own `VK_ESCAPE` arm under `campaign+0x3dc == 1`, so it raises exactly
//     what Esc raises.
//
// Rects C and F play fixed slot 1 before their state/message transition;
// town/shop own the D/E routes. VIDEO-SFX-013 establishes that `0xdc` in the
// older TOWN-347 slice is playback priority, not the selector.
//
// DIV-314
func (v *Viewer) pressCharacterPaneCorner(x, y int) bool {
	corners := v.characterPaneCornerAt(x, y)
	for _, c := range corners {
		switch c {
		case CharacterPaneBackpack:
			v.toggleHudPanel(hudPanelPack)
		case CharacterPaneBook:
			v.toggleHudPanel(hudPanelBook)
		case CharacterPaneMode:
			v.PlayUISound(UISoundCampaignPanel)
			v.toggleHudPanel(hudPanelDoll)
		case CharacterPaneMenu:
			v.PlayUISound(UISoundCampaignPanel)
			v.paneMenuRequest = true
		}
	}
	return len(corners) > 0
}

// TakeCharacterPaneMenu reads the pending in-mission menu request rect F
// raised, if any, and clears it in the SAME statement — TakeInventoryEquip's
// own one-shot shape, so a drained request cannot be drained twice.
//
// THE TIER ABOVE RAISES THE MENU; this file only holds the request. Esc
// already reaches the same surface through the front end's own binding, and
// rect F is the second route the original gives it (`MENU-ESC-010`).
func (v *Viewer) TakeCharacterPaneMenu() bool {
	if !v.paneMenuRequest {
		return false
	}
	v.paneMenuRequest = false
	return true
}

// characterPaneFigureAt reports that the window pixel stands on the pane's own
// painted figure rather than on the pane frame —
// shopDollAreaAt's own question for the mission screen.
//
// It is the geometry alone. Whether a slot is marked there is
// dollFigureSlotAtMask's question, asked after this one.
func (v *Viewer) characterPaneFigureAt(p image.Point) bool {
	box, ok := v.dollBox()
	if !ok {
		return false
	}
	if !p.In(characterPaneFigureRect(box)) {
		return false
	}
	return true
}

// missionCardPresent is the statistics card the mission column draws under
// the fourth child's own background strip, and where its top-left corner
// goes, or false for a frame with no room or no card font.
//
// IT ANSWERS FALSE WHILE THE PANE ITSELF HOLDS THE CARD. The pane's mode is
// still live on this screen through Tab (`DIV-312` records that the
// original's own rect C and its key route are both shut at 1024x768 and this
// build's key is not), so a player who switches the pane to statistics would
// otherwise see the same card twice. The pane's mode decides which box holds
// the card and never how many do.
//
// IT COMPOSES THE BODY ALONE, never DrawTownCharacterRegion: the six corner
// controls belong to the one real widget above, whose own hit test is the
// only one that answers for them (`characterPaneCornerAt`), and painting a
// second, dead set of them would state a control this box does not have. The
// view is the pane's own, with the mode forced and the corner art and figure
// dropped, so the card's rows, font and background are the pane's own in
// every other respect.
//
// ITS CACHE IS panelPresent's, one box over: the picture is a function of the
// subject, the selection lines, the layout serial and the box, so it is
// rebuilt on a key change and uploaded on the fresh flag rather than every
// frame. The lines are in the key because a selection of two or more carries
// no subject, so only its lines tell one count from another.
func (v *Viewer) missionCardPresent() (*image.RGBA, image.Point, bool) {
	return v.missionCardPresentWith(false)
}

// missionCardPresentWith is missionCardPresent, optionally with the card's rows
// left off: the box's background art alone, which the structure readout is
// drawn on.
func (v *Viewer) missionCardPresentWith(blank bool) (*image.RGBA, image.Point, bool) {
	if v.cardFont() == nil {
		return nil, image.Point{}, false
	}
	area := image.Pt(v.frameW, v.frameH)
	rect, ok := missionCardBoxRect(area)
	if !ok {
		return nil, image.Point{}, false
	}
	view := v.characterPaneView(true)
	if view.Statistics {
		return nil, image.Point{}, false
	}
	view.Statistics = true
	view.CornerArt = nil
	view.Figure = nil
	view.SelectionStatus = [3]string{}
	if blank {
		view.Subject, view.HasSubject = PanelSubject{}, false
	}
	key := panelKey{subject: view.Subject, serial: v.panelSerial, area: area,
		statistics: true, figure: nil,
		packOpen: view.PackOpen, bookOpen: view.BookOpen,
		selectionStatus: view.SelectionStatus}
	if v.missionCardPic == nil || key != v.missionCardKey {
		// ONE STATEMENT, panelPresent's own reason: what the upload reads and
		// the flag that says it must are written together or not at all.
		pic := image.NewRGBA(image.Rect(0, 0, rect.Dx()+characterPaneSeamW, rect.Dy()))
		v.missionCardText = text.Record(func() { DrawCharacterPaneBody(pic, view) })
		v.missionCardPic, v.missionCardFresh = pic, true
		v.missionCardKey = key
	}
	text.Append(v.missionCardText, 0, 0)
	return v.missionCardPic, image.Pt(rect.Min.X-characterPaneSeamW, rect.Min.Y), true
}

func (v *Viewer) columnFillerPresent() (image.Image, image.Point, bool) {
	if v.paneFillerArt == nil {
		return nil, image.Point{}, false
	}
	box, ok := columnFillerRect(image.Pt(v.frameW, v.frameH))
	if !ok {
		return nil, image.Point{}, false
	}
	at := box.Min
	at.X -= v.paneFillerArt.Bounds().Dx() - box.Dx()
	return v.paneFillerArt, at, true
}
