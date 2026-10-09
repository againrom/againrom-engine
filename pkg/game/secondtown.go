package game

import (
	"fmt"
	"image"

	"againrom/pkg/ui"
)

// firstSecondTown is the second campaign's initial location, type 2 ID 1
// (R2-ENGINE-073). It is drawn as the first game's town (DIV-2672).
var firstSecondTown = secondLocation{kind: 2, id: 1}

// firstTownSchoolCode is the square mask code of the school building. The
// second game's first town has no school room, so the code opens nothing
// and arms no reaction (DIV-2673).
const firstTownSchoolCode = 0xc0

// firstTownView is the first town's presentation: a town screen used only
// for its square exterior and tavern interior, whose room follows the
// campaign state, and the selected inn speaker's NPC key (0 for none).
type firstTownView struct {
	view *townScreen
	pick int
}

func (t *secondCampaignScreen) bindFirstTown(f *FrontEnd) {
	t.view = f.bindTown(&townScreen{shopChosen: shopNoShelf, schoolCell: schoolNoSelection, room: roomGates})
	t.view.gateRule = func() bool {
		c := t.state()
		return c != nil && c.gateOpen()
	}
}

// firstTownRoom is the presentation room the campaign state asks for:
// the square, the tavern, or roomGates for any surface this view does not
// draw.
func (t *secondCampaignScreen) firstTownRoom() townRoom {
	c := t.state()
	if c == nil || c.current != firstSecondTown {
		return roomGates
	}
	if c.room == secondTownInn {
		return roomTavern
	}
	return roomSquare
}

// syncFirstTown moves the presentation into the room the campaign state
// holds, with the first game's own entry and exit steps. A load or a new
// game changes the state under the screen, so every seam call syncs.
func (t *secondCampaignScreen) syncFirstTown() {
	if t.view == nil {
		return
	}
	want := t.firstTownRoom()
	if t.view.room == want {
		return
	}
	t.view.atSquare()
	switch want {
	case roomTavern:
		t.view.destroyRoomAudio()
		t.view.room = roomTavern
		t.view.loadTip(roomTavern)
		t.view.enterTavernInterior()
		t.view.composeShopFaces()
		t.pick = 0
	case roomGates:
		t.view.room = roomGates
	}
}

// firstTownRows is the square's four doors in the first game's door order,
// which the square's mask clicks index, then the inn's speakers.
func (t *secondCampaignScreen) firstTownRows(c *secondCampaign) []ui.TownRow {
	if c.room == secondTownInn {
		var rows []ui.TownRow
		for _, o := range c.speakers() {
			rows = append(rows, ui.TownRow{Text: fmt.Sprintf("TALK %d", o.npc), Choosable: true})
		}
		return rows
	}
	// The shop is closed (DIV-2675) and the school has no room (DIV-2673).
	return []ui.TownRow{{Text: "TAVERN", Choosable: true}, {Text: "SHOP"}, {Text: "SCHOOL"},
		{Text: "GATES", Choosable: c.gateOpen()}}
}

func (t *secondCampaignScreen) chooseFirstTown(c *secondCampaign, i int) ui.TownAction {
	defer t.syncFirstTown()
	if c.room == secondTownInn {
		return t.talk(c, c.speakers()[i])
	}
	switch townDoors[i].room {
	case roomTavern:
		c.room = secondTownInn
	case roomGates:
		c.leaveTown()
		c.room = secondTownSquare
	}
	return ui.TownAction{}
}

// AtTownSquare reports the first town's square, drawn from the install's
// own town art.
func (t *secondCampaignScreen) AtTownSquare() bool {
	return t.firstTownRoom() == roomSquare
}

func (t *secondCampaignScreen) TownSquareView() ui.TownSquareView {
	t.syncFirstTown()
	if !t.AtTownSquare() {
		return ui.TownSquareView{}
	}
	return t.view.TownSquareView()
}

// TownSquarePointer hands the pointer to the square's own reaction. Over
// the school it is handed on as off the picture, so no label, motion or
// sound answers it (DIV-2673).
func (t *secondCampaignScreen) TownSquarePointer(p image.Point) {
	t.syncFirstTown()
	if !t.AtTownSquare() {
		return
	}
	if art := t.install.TownSquareArt.Value(); art != nil && art.Mask != nil && p.In(art.Mask.Bounds()) && art.Mask.ColorIndexAt(p.X, p.Y) == firstTownSchoolCode {
		p = image.Pt(-1, -1)
	}
	t.view.TownSquarePointer(p)
}

func (t *secondCampaignScreen) TownSquareActive(active bool) {
	t.syncFirstTown()
	t.view.TownSquareActive(active && t.AtTownSquare())
}

func (t *secondCampaignScreen) AdvanceTownSquareAnimation() {
	t.syncFirstTown()
	if t.AtTownSquare() {
		t.view.AdvanceTownSquareAnimation()
	}
}

// AtTownSurface reports the first town's tavern, drawn as the first game's
// tavern room.
func (t *secondCampaignScreen) AtTownSurface() bool {
	return t.firstTownRoom() == roomTavern
}

func (t *secondCampaignScreen) TavernInteriorActive(active bool) {
	t.syncFirstTown()
	t.view.TavernInteriorActive(active && t.AtTownSurface())
}

func (t *secondCampaignScreen) AdvanceTownSurfaceAnimation() {
	t.syncFirstTown()
	if t.AtTownSurface() {
		t.view.AdvanceTownSurfaceAnimation()
	}
}

// firstTownSpeaker answers the selected speaker while it is still offered.
func (t *secondCampaignScreen) firstTownSpeaker(c *secondCampaign) (secondInnOption, bool) {
	for _, o := range c.speakers() {
		if t.pick != 0 && o.npc == t.pick {
			return o, true
		}
	}
	return secondInnOption{}, false
}

// TownSurface is the tavern room over the install's inn art. Its talk cells
// are the campaign's speakers in option order; no cell is painted until one
// is selected, the first game's rule for a roster without mercenaries
// (TOWN-468). Sleep and Hire stay disabled (DIV-2674).
func (t *secondCampaignScreen) TownSurface() ui.TownSurfaceView {
	t.syncFirstTown()
	c := t.state()
	if c == nil || !t.AtTownSurface() {
		return ui.TownSurfaceView{}
	}
	in := t.view.in
	v := ui.TownSurfaceView{Kind: ui.TownSurfaceTavern, Title: "TAVERN", Font: in.Font.Value(), CardFont: in.tipFont(),
		HiredLabel: in.Words.TavernHired, Hero: t.view.townCharacterView(), HoverCell: -1, SchoolClass: -1,
		TavernArt: in.TownTavernArt.Value(), TavernInterior: t.view.tavernInteriorFrame()}
	v.Hero.NoBook, v.Hero.BookOpen = true, false
	selected, ok := t.firstTownSpeaker(c)
	for _, o := range c.speakers() {
		label := fmt.Sprintf("NPC %d", o.npc)
		v.Cells = append(v.Cells, ui.TownSurfaceCell{Key: label, Label: label, Semantic: label,
			Enabled: true, Portrait: true, TalkOnly: true, Selected: ok && o.npc == selected.npc})
	}
	v.RosterUnpainted = !ok
	v.Buttons = []ui.TownSurfaceButton{
		{Label: in.Words.TavernSleep},
		{},
		{Label: in.Words.TavernTalk, Enabled: ok},
		{Label: in.Words.TavernExit, Enabled: true},
	}
	v.Tip = t.view.tipView(roomTavern, t.view.tavernTip, ui.TipPanelShrinkRect(ui.TavernTipRect, in.tipFont(), t.view.tavernTip))
	return v
}

// TownSurfaceClick selects a speaker, talks to the selected one (Talk or a
// double click), steps the party panel, or leaves for the square.
func (t *secondCampaignScreen) TownSurfaceClick(control ui.TownSurfaceControl, double bool) ui.TownAction {
	t.syncFirstTown()
	c := t.state()
	if c == nil || !t.AtTownSurface() || c.payload != nil {
		return ui.TownAction{}
	}
	switch control.Kind {
	case ui.TownSurfaceControlPrevious:
		return t.view.stepTownMember(-1)
	case ui.TownSurfaceControlNext:
		return t.view.stepTownMember(+1)
	case ui.TownSurfaceControlMode:
		if len(t.view.shopParty()) != 0 {
			t.view.townStats = !t.view.townStats
		}
	case ui.TownSurfaceControlCell:
		speakers := c.speakers()
		if control.Index < 0 || control.Index >= len(speakers) {
			return ui.TownAction{}
		}
		t.pick = speakers[control.Index].npc
		if double {
			return t.talk(c, speakers[control.Index])
		}
	case ui.TownSurfaceControlButton:
		switch control.Index {
		case tavernButtonTalk:
			if o, ok := t.firstTownSpeaker(c); ok {
				return t.talk(c, o)
			}
		case tavernButtonExit:
			t.Back()
			t.syncFirstTown()
		}
	}
	return ui.TownAction{}
}

// TownMusic keeps the first game's room tracks: the tavern's in the inn,
// the town's everywhere else.
func (t *secondCampaignScreen) TownMusic() (ui.MusicScene, bool) {
	if t.AtTownSurface() {
		return ui.MusicTavern, false
	}
	return ui.MusicTown, false
}
