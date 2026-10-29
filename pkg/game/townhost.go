package game

import (
	"fmt"
	"image"

	"againrom/pkg/mapload"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

type chapterTownTalk struct{ host *townScreen }
type sectionTownTalk struct{ host *townScreen }

type townMapPage interface {
	worldMapCampaign
	atMap() bool
	tick() ui.TownAction
}
type chapterTownMap struct{ host *townScreen }
type destinationTownMap struct{ host *townScreen }

func (t *townScreen) mapPage() townMapPage {
	if t.mapRoute == nil {
		if t.hostRules().Leave == "room-map" {
			t.mapRoute = &chapterTownMap{t}
		} else {
			t.mapRoute = &destinationTownMap{t}
		}
	}
	return t.mapRoute
}

func (t *townScreen) AtWorldMap() bool                          { return t != nil && t.mapPage().atMap() }
func (t *townScreen) WorldMapView() ui.WorldMapView             { return t.mapPage().WorldMapView() }
func (t *townScreen) worldMapNow() *worldMapState               { return t.mapPage().worldMapNow() }
func (t *townScreen) WorldMapHover(p image.Point)               { worldMapHover(t.mapPage(), p) }
func (t *townScreen) WorldMapMove(delta int)                    { worldMapMove(t.mapPage(), delta) }
func (t *townScreen) WorldMapChoose() ui.TownAction             { return worldMapChoose(t.mapPage()) }
func (t *townScreen) WorldMapClick(p image.Point) ui.TownAction { return worldMapClick(t.mapPage(), p) }
func (t *townScreen) WorldMapTick() ui.TownAction               { return t.mapPage().tick() }
func (t *townScreen) chooseWorldRecord(i int)                   { t.mapPage().chooseWorldRecord(i) }
func (t *townScreen) leaveWorldMap()                            { t.mapPage().leaveWorldMap() }
func (t *townScreen) arriveWorldMap() ui.TownAction             { return t.mapPage().arriveWorldMap() }

func (t *townScreen) StartsInTown() bool { return t.hostRules().StartsInTown }

func (t *townScreen) bindShop() {
	if t.hostRules().Shop == "bank" {
		t.bindBankShop()
	} else {
		t.bindChapterShop()
	}
}

func (t *townScreen) AtTownShop() bool {
	if t == nil {
		return false
	}
	if t.hostRules().Shop == "bank" {
		return t.bankShopShown()
	}
	return t.displayRoom() == "shop"
}

func (t *townScreen) AtTownSquare() bool {
	if t == nil {
		return false
	}
	d, room, loaded := t.composedRoom()
	if d == nil {
		return !t.hostRules().RequireArt && t.displayRoom() == "square"
	}
	if !loaded && t.hostRules().RequireArt {
		return false
	}
	if room == d.Square.Name {
		return true
	}
	r := d.Room(room)
	return r != nil && r.Page == "inn" && !t.AtTownShop()
}

func (t *townScreen) TownSquareView() ui.TownSquareView {
	if t == nil || t.sess == nil {
		return ui.TownSquareView{}
	}
	d, room, loaded := t.composedRoom()
	if d == nil {
		return ui.TownSquareView{}
	}
	v := ui.TownSquareView{Font: t.in.Font.Value(), Tip: t.hostTipView(d, room)}
	if loaded {
		v.Scene = townRoomView{t: t, d: d, room: room}
	}
	if r := d.Room(room); r != nil {
		t.paneSquareView(&v, r, t.roomArt(d, room))
	}
	return v
}

func (t *townScreen) TownSquareActive(active bool) {
	if t == nil || t.sess == nil {
		return
	}
	if t.hostRules().Pages == "entry" {
		if t.composed != nil && t.campaignTown().ready() && !t.AtTownShop() {
			t.roomActive(active)
		}
		return
	}
	t.squareView().SetActive(active, t.AtTownSquare())
}

func (t *townScreen) TownSquarePointer(p image.Point) {
	if t == nil || t.sess == nil || !t.AtTownSquare() {
		return
	}
	d, room, loaded := t.composedRoom()
	if d == nil || !loaded && t.hostRules().RequireArt {
		return
	}
	t.TownSquareActive(true)
	t.composed.pointer = p
	t.scene(d, room).Pointer(p)
}

func (t *townScreen) AdvanceTownSquareAnimation() {
	if t == nil || t.sess == nil {
		return
	}
	if t.hostRules().Pages == "entry" {
		if !t.AtTownShop() {
			t.advanceRoom()
		}
		return
	}
	if t.AtTownSquare() {
		t.squareView().Advance()
	}
}

func (t *townScreen) CanSave() bool {
	if t == nil || t.sess == nil || !t.saveAdmitted() {
		return false
	}
	if d := t.townDescription(); d != nil {
		return t.squareView().SaveAdmitted()
	}
	return true
}

func (t *townScreen) TownMusic() (ui.MusicScene, bool) {
	if t == nil {
		return ui.MusicTown, false
	}
	d := t.townDescription()
	state := townMusicState{room: t.displayRoom(), present: t.campaignTown().present()}
	if d != nil && t.defaults != nil {
		if r := d.Room(state.room); r == nil || r.MusicRule == nil {
			if fallback := t.defaults.Room(state.room); fallback != nil && fallback.MusicRule != nil {
				d = t.defaults
			}
		}
	}

	if d != nil {
		if r := d.Room(state.room); r != nil && r.MusicRule != nil && r.MusicRule.By == "member-class" && t.sess != nil {
			party := t.nextParty()
			i := townPickerIndex(party, t.shopMember)
			state.mage = len(party) != 0 && party[i].Mage
		}
	}
	return townRoomMusic(d, state)
}

func (t *townScreen) TownMusicTrack() (string, bool) {
	if scene, _ := t.TownMusic(); scene != ui.MusicTown {
		return "", false
	}
	if t.hostRules().TrackSource == "campaign" {
		if c := t.state(); c != nil && c.current.kind == placeTown {
			return GameMusic(t.in.profile()).TownTrack(c.current.id)
		}
		return "", false
	}
	return t.townDescription().Music.Track, true
}

func (t *townScreen) Rows() []ui.TownRow {
	if !t.campaignTown().ready() {
		return nil
	}
	if t.campaignTown().talking() && t.hostRules().Talk == "building" {
		return nil
	}
	d := t.townDescription()
	if d == nil {
		return t.fallbackRows()
	}
	if r := d.Room(t.roomName()); r != nil {
		switch r.Page {
		case "shop", "world-map":
			return nil
		case "school":
			return t.shelfRows(TownSchool)
		case "tavern":
			return t.tavernRows()
		case "inn":
			return append(t.state().innRows(), ui.TownRow{Text: secondInnExit, Choosable: true})
		}
	}
	return t.squareRows()
}

func (t *townScreen) squareRows() []ui.TownRow {
	s := t.squareView()
	if s == nil {
		return nil
	}
	var rows []ui.TownRow
	for i := 0; ; i++ {
		h := s.HotspotForRow(i)
		if h == nil {
			return rows
		}
		label := h.RowText
		if label == "" {
			label = townRowText[h.Name]
		}
		if t.hostRules().Rows == "chapter-hints" {
			hint := ""
			switch h.Hint {
			case "tavern-offers":
				hint = fmt.Sprintf("%d with something to say", t.campaignTown().value(h.Hint))
			case "shop-offers":
				hint = fmt.Sprintf("%d on the shelf", t.campaignTown().value(h.Hint))
			case "school-offers":
				hint = fmt.Sprintf("%d on the board", t.campaignTown().value(h.Hint))
			case "missions":
				hint = fmt.Sprintf("%d mission(s) available", t.campaignTown().value(h.Hint))
			}
			label = fmt.Sprintf("%-8s  %s", label, hint)
		}
		rows = append(rows, ui.TownRow{Text: label, Choosable: true})
	}
}

func (t *townScreen) fallbackRows() []ui.TownRow {
	c := t.state()
	if c == nil || c.current.kind != placeTown {
		return nil
	}
	gates := ui.TownRow{Text: "GATES", Choosable: c.gateOpen()}
	if t.roomName() != "inn" {
		return []ui.TownRow{{Text: "TAVERN", Choosable: true}, gates}
	}
	return append(c.innRows(), gates)
}

func (t *townScreen) Choose(i int) ui.TownAction {
	if t.campaignTown().talking() && t.hostRules().Talk == "building" {
		return t.AdvanceTownDialogue()
	}
	d := t.townDescription()
	if d == nil {
		return t.chooseFallback(i)
	}
	room := t.roomName()
	if r := d.Room(room); r != nil {
		switch r.Page {
		case "school":
			if t.schoolTrainingBusy() {
				return ui.TownAction{}
			}
			return t.openBuildingDialogue(TownSchool, i)
		case "tavern":
			offers := t.tavernVisibleOffers()
			if i < 0 || i >= len(offers) || !t.tavernOfferCanTalk(offers[i]) {
				return ui.TownAction{}
			}
			t.composeShopFaces()
			t.openOfferDialogue(TownTavern, offers[i], offers[i].NPC)
		case "inn":
			rows := t.Rows()
			if i < 0 || i >= len(rows) || !rows[i].Choosable {
				return ui.TownAction{}
			}
			if seats, _ := t.state().innSeats(); i < len(seats) {
				if r.Seats != nil {
					t.selectSeat(t.scene(d, room), r.Seats, i)
				}
				return t.innAct(t.state(), i)
			}
			t.squareView().LeaveRoom(room)
		}
		return ui.TownAction{}
	}
	return t.chooseSquare(i)
}

func (t *townScreen) chooseFallback(i int) ui.TownAction {
	rows := t.Rows()
	if i < 0 || i >= len(rows) || !rows[i].Choosable {
		return ui.TownAction{}
	}
	c := t.state()
	if c == nil || c.current.kind != placeTown {
		return ui.TownAction{}
	}
	if i == len(rows)-1 {
		t.campaignTown().leave()
		return ui.TownAction{}
	}
	if t.roomName() != "inn" {
		t.campaignTown().setRoom("inn")
		c.enterInn()
		return ui.TownAction{}
	}
	return t.innAct(c, i)
}

func (t *townScreen) Back() bool {
	if t.visit != nil {
		return t.closeMissionShop()
	}
	if t.campaignTown().talking() {
		t.AdvanceTownDialogue()
		return true
	}
	room := t.roomName()
	if room == "square" || room == "" {
		return false
	}
	d := t.townDescription()
	if d == nil {
		if room == "inn" {
			t.campaignTown().setRoom("square")
			return true
		}
		return false
	}
	r := d.Room(room)
	if r == nil {
		return false
	}
	if r.Page == "school" && t.schoolTrainingBusy() {
		return true
	}
	if r.Page == "world-map" && t.worldMap != nil && t.worldMap.returnMission != 0 {
		t.arriveWorldMap()
		return true
	}
	t.squareView().LeaveRoom(room)
	return true
}

func (t *townScreen) boundTalkSource() townTalkSource {
	if t.talkSource == nil {
		if t.hostRules().Talk == "section" {
			t.talkSource = &sectionTownTalk{t}
		} else {
			t.talkSource = &chapterTownTalk{t}
		}
	}
	return t.talkSource
}

func (t *townScreen) talkView() ui.TownTalkView {
	if t == nil {
		return ui.TownTalkView{Profile: firstTownTalkProfile}
	}
	return t.boundTalkSource().talkView()
}
func (t *townScreen) turnTownTalk() bool { return t.boundTalkSource().turnTownTalk() }

func (t *townScreen) Header() string {
	if t.hostRules().Header == "location" {
		if t.campaignTown().present() {
			switch t.roomName() {
			case "inn":
				return t.hostRules().HeaderPrefix + ": tavern"
			case "shop":
				return t.hostRules().HeaderPrefix + ": shop"
			}
			return fmt.Sprintf("%s: town %d", t.hostRules().HeaderPrefix, t.campaignTown().townID())
		}
		return t.hostRules().HeaderPrefix + ": destinations"
	}
	ch := t.campaignTown().chapter()
	where := "the town square"
	switch t.roomName() {
	case "gates":
		where = "the gates"
	case "shop":
		where = "the shop"
	case "school":
		where = "the school"
	case "tavern", "inn":
		where = "the tavern"
	case "talk":
		where = t.dialogueBuilding.String()
		if t.dialogueBuilding == TownTavern {
			where = fmt.Sprintf("the tavern - NPC %d", t.npc)
		}
	}
	if ch == 0 {
		return fmt.Sprintf("%s - the campaign offers nothing further", where)
	}
	return fmt.Sprintf("%s - chapter %d    [Esc: back]", where, ch)
}

func (t *townScreen) Footer() []string {
	if t.hostRules().Footer == "services" {
		var out []string
		for _, id := range t.campaignTown().unavailableTowns() {
			out = append(out, fmt.Sprintf("Town %d services are unavailable.", id))
		}
		return out
	}
	party := t.nextParty()
	items, worn := 0, 0
	for _, m := range party {
		items += len(mapload.MemberCarriedItems(m, t.in.Table))
		for _, item := range mapload.MemberItemEquipment(m, t.in.Table) {
			if !item.Empty() {
				worn++
			}
		}
	}
	out := []string{
		fmt.Sprintf("gold %d    party %d    carried %d items, %d worn",
			t.sess.Town.Gold(), len(party), items, worn),
		fmt.Sprintf("available %v    done %d", t.sess.Town.Available(), t.doneCount()),
	}
	return out
}

func (t *townScreen) viewScene(d *town.Description, room string) *town.Scene {
	if room == d.Square.Name && t.hostRules().Pages == "retained" {
		return t.squareView()
	}
	return t.scene(d, room)
}
