package game

import (
	"fmt"
	"image"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type secondTownRoom uint8

const (
	secondTownSquare secondTownRoom = iota
	secondTownInn
	secondTownShop
)

// secondAux is the eight-entry auxiliary array of five DWORD fields the
// departure stores outside the bank: the shop assortment records of
// locations 2 and 3, four categories each, fields minimum, maximum, draws,
// quantity bound and mask (R2-ENGINE-330). The campaign holds them in its
// shop; a current SAV this engine wrote before carries the array itself.
type secondAux [8][5]int32

type secondCampaign struct {
	bank      [1024]int32
	available []partyPlace
	current   partyPlace
	room      secondTownRoom
	talk      talkCursor
	shop      secondShop
	// party is the hero alone; his bag is the shop's customer.
	party []mapload.PartyMember
	// legacyHero is an older document's hero list waiting for the
	// install's tables to load into the bag.
	legacyHero []secondHeroItem
	// mapOpen is the map's current opening (openMap); nil while the party
	// stands at a location and after a LOAD at the map.
	mapOpen *secondMapOpening
	// innBits is the inn page's copy of each option's high bit where Hire
	// or Fire changed it during the current visit (R2-SESSION-191).
	innBits map[int]bool
}

func newSecondCampaign() *secondCampaign {
	initial := partyPlace{kind: placeTown, id: 1}
	c := &secondCampaign{available: []partyPlace{initial}, current: initial, shop: newSecondShop()}
	c.bank[768] = 10
	return c
}

func (c *secondCampaign) has(location partyPlace) bool {
	for _, candidate := range c.available {
		if candidate == location {
			return true
		}
	}
	return false
}

// leaveTown is the type-2 departure: town 1 removes its node, a later town
// keeps it, and both clear current (R2-ENGINE-144).
func (c *secondCampaign) leaveTown() bool {
	if c.current.kind != placeTown || !c.gateOpen() {
		return false
	}
	if c.current.id == 1 {
		remaining := c.available[:0]
		for _, l := range c.available {
			if l != c.current {
				remaining = append(remaining, l)
			}
		}
		c.available = remaining
	}
	c.openMap(c.current)
	c.current = partyPlace{}
	return true
}

// gateOpen is town 1's gate rule: open once bank 769 is not zero
// (R2-SESSION-128); every later town's gate is open.
func (c *secondCampaign) gateOpen() bool {
	return c.current.id != 1 || c.bank[769] != 0
}

func (c *secondCampaign) canEnter(n int) bool {
	return c.current == (partyPlace{}) && c.has(partyPlace{kind: placeMission, id: n})
}

// finish admits the ordinary completion of mission n. won is the front end's
// one win predicate: the script outcome or the announced client outcome, which
// the victory command reaches through the same completion (R2-ENGINE-303).
func (c *secondCampaign) finish(n int, w *sim.World, won bool) error {
	if !secondContinues(n) || c.current != (partyPlace{kind: placeMission, id: n}) || !c.has(c.current) || w == nil || !won {
		return fmt.Errorf("campaign continuation after mission %d is unavailable", n)
	}
	bank, ok := w.ROM2ScenarioState()
	if !ok || bank[775] != 0 {
		return fmt.Errorf("campaign continuation requires ordinary state without bank775 restoration")
	}
	return nil
}

// secondContinues names the missions whose victory the controller continues:
// every ordinary ID whose slot 896+ID lies inside the bank (R2-SESSION-052).
func secondContinues(n int) bool { return n > 0 && n < 128 }

// complete applies the ordinary departure and answers its output DWORD.
func (c *secondCampaign) complete(w *sim.World) int {
	bank, _ := w.ROM2ScenarioState()
	return c.completeBank(bank)
}

// completeBank applies the ordinary departure to the bank a won mission left
// and answers the output DWORD: -1 unless a case body stores one
// (R2-ENGINE-145, R2-ENGINE-146, R2-ENGINE-148, R2-SESSION-047..050).
func (c *secondCampaign) completeBank(bank [1024]int32) int {
	n := c.current.id
	output := -1
	c.bank = bank
	c.bank[773] = 0
	for i := 0; i < 20; i++ {
		if c.bank[532+i] != 0 {
			c.bank[532+i] = 1
			if c.bank[512+i] != 0 {
				c.bank[532+i] = 2
			}
		}
		c.bank[512+i] = 0
	}
	c.bank[896+n] = 1
	remaining := c.available[:0]
	for _, l := range c.available {
		if l != c.current {
			remaining = append(remaining, l)
		}
	}
	c.available = remaining
	add := func(l partyPlace) {
		if !c.has(l) {
			c.available = append(c.available, l)
		}
	}
	switch n {
	case 10:
		add(partyPlace{1, 20})
		output = 1
	case 20:
		add(partyPlace{2, 2})
		if c.bank[772] != 0 {
			add(partyPlace{1, 21})
		}
		c.bank[532], c.bank[552] = 1, 2
	case 30:
		output = 2
	case 31:
		add(partyPlace{1, 32})
	case 40:
		add(partyPlace{1, 50})
		add(partyPlace{1, 60})
		c.bank[773] = 23
	case 50:
		// The bank780-gated fixed record has no resolved type or ID and is
		// not appended (DIV-2629).
		add(partyPlace{2, 3})
		c.bank[771], c.bank[534], c.bank[554] = 1, 1, 3
	case 60:
		add(partyPlace{1, 80})
		c.bank[537], c.bank[557], c.bank[774] = 1, 2, 1
	case 70:
		c.bank[535], c.bank[555] = 1, 3
		if c.bank[777] == 0 && c.bank[778] != 0 {
			output = 3
		}
		c.bank[777], c.bank[770] = 1, 1
	case 80:
		if c.bank[778] == 0 && c.bank[777] != 0 {
			output = 3
		}
		c.bank[778] = 1
	case 100:
		c.bank[538], c.bank[558] = 1, 2
	case 110:
		output = 4
		if c.bank[779] != 0 {
			output = 5
		}
	}
	c.shop.depart(n)
	if n%10 == 0 {
		c.bank[768] += 10
	}
	c.shop.stage(c.bank[768])
	c.openMap(c.current)
	c.current, c.room = partyPlace{}, secondTownSquare
	return output
}

type secondCampaignScreen struct {
	// shopController is the shop room's page.
	shopController
	session *CampaignSession
	install *InstallResources
	open    missionDoor
	frame   func() *ui.DialogFrame
	start   func() error
	// tags is how the town dialogue's tag bodies are read.
	tags eventTags
	// composed is the composed town's presentation.
	composed *secondTownState
	// mapArt is the install's world-map art; worldMap is the open map, built
	// for one opening of one campaign.
	mapArt     func() *worldMapAssets
	worldMap   *worldMapState
	worldMapOf *secondCampaign
	// worldMapAt is the opening worldMap was built for.
	worldMapAt *secondMapOpening
	// townTalkSeat is the town talk window; the campaign holds its pages.
	townTalkSeat
	// options holds the tips option.
	options *PersistenceContext
	tipArt  func() *ui.TipPanelArt
	tip     secondRoomTip
	// talkFaces resolves the open conversation's speaker faces.
	talkFaces speakerResolver
}

func (f *FrontEnd) startSecondCampaign() error {
	f.startSecondCampaignWith(MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table))
	return nil
}

// startSecondCampaignWith drops the session and starts a new campaign at its
// first town with party.
func (f *FrontEnd) startSecondCampaignWith(party []mapload.PartyMember) {
	releaseWorldAudio(f.runtimeAudio(), f.endLive())
	f.CampaignSession.clear(Campaign{})
	f.Town.second = newSecondCampaign()
	f.Carried = party
}

func (f *FrontEnd) secondCampaignScreen(tags eventTags) ui.TownScreen {
	t := &secondCampaignScreen{session: &f.CampaignSession, install: &f.InstallResources,
		open: f.MissionOpener, frame: f.gameMenuArt, start: f.startSecondCampaign, tags: tags,
		composed: newSecondTownState(f), mapArt: f.worldMapAssets,
		options: &f.PersistenceContext, tipArt: f.tipArt}
	t.talkSource = t
	t.bindShop()
	return t
}

func (t *secondCampaignScreen) StartNewGame() error {
	if _, err := t.install.Archives.Containers.ReadFile(mainPrefix + "text/town.txt"); err != nil {
		return fmt.Errorf("initial campaign town: %w", err)
	}
	if err := t.start(); err != nil {
		return err
	}
	return nil
}

func (t *secondCampaignScreen) state() *secondCampaign {
	if t.session.Town == nil {
		return nil
	}
	return t.session.Town.second
}

func (t *secondCampaignScreen) Header() string {
	if c := t.state(); c != nil && c.current.kind == placeTown {
		if c.room == secondTownInn {
			return "ROM2 campaign: tavern"
		}
		if c.room == secondTownShop {
			return "ROM2 campaign: shop"
		}
		return fmt.Sprintf("ROM2 campaign: town %d", c.current.id)
	}
	return "ROM2 campaign: destinations"
}

func (t *secondCampaignScreen) Rows() []ui.TownRow {
	c := t.state()
	if c == nil {
		return nil
	}
	if d, ok := t.composedTown(); ok {
		return t.composedRows(d, c)
	}
	if c.current.kind == placeTown {
		gates := ui.TownRow{Text: "GATES", Choosable: c.gateOpen()}
		if c.room != secondTownInn {
			return []ui.TownRow{{Text: "TAVERN", Choosable: true}, gates}
		}
		return append(c.innRows(), gates)
	}
	return nil
}

func (t *secondCampaignScreen) Footer() []string {
	c := t.state()
	if c == nil {
		return nil
	}
	var out []string
	for id := 2; id <= 3; id++ { // DIV-2438, DIV-2631
		if town := (partyPlace{2, id}); c.current == town || c.has(town) {
			out = append(out, fmt.Sprintf("Town %d services are unavailable.", id))
		}
	}
	return out
}

func (t *secondCampaignScreen) Choose(i int) ui.TownAction {
	rows := t.Rows()
	if i < 0 || i >= len(rows) || !rows[i].Choosable {
		return ui.TownAction{}
	}
	c := t.state()
	if d, ok := t.composedTown(); ok {
		return t.chooseComposed(d, c, i)
	}
	if c.current.kind != placeTown {
		return ui.TownAction{}
	}
	if i == len(rows)-1 {
		c.leaveTown()
		c.room = secondTownSquare
		return ui.TownAction{}
	}
	if c.room != secondTownInn {
		c.room = secondTownInn
		c.enterInn()
		return ui.TownAction{}
	}
	return t.innAct(c, i)
}

// innAct runs inn seat i: Hire or Fire on a hire actor, TALK on a talk
// actor (R2-ENGINE-496).
func (t *secondCampaignScreen) innAct(c *secondCampaign, i int) ui.TownAction {
	seats, hires := c.innSeats()
	if i >= hires {
		return t.talk(c, seats[i])
	}
	if key := c.hireOrFire(seats[i].npc); key != "" {
		return t.openTownText(key)
	}
	return ui.TownAction{}
}

// talk opens a speaker's conversation and runs its TalkTo.
func (t *secondCampaignScreen) talk(c *secondCampaign, o secondInnOption) ui.TownAction {
	payload, err := readSecondTownTalk(t.install, secondTalkKey(o))
	if err != nil {
		return ui.TownAction{Msg: err.Error()}
	}
	c.talkTo(o)
	if payload != nil {
		t.setTownDialogue(payload)
	}
	return ui.TownAction{}
}

func (t *secondCampaignScreen) Back() bool {
	c := t.state()
	if c == nil {
		return false
	}
	if c.talk.payload != nil {
		t.AdvanceTownDialogue()
		return true
	}
	if d, ok := t.composedTown(); ok {
		return t.backComposed(d)
	}
	if c.room == secondTownInn {
		c.room = secondTownSquare
		return true
	}
	return false
}

func (t *secondCampaignScreen) dialogueBody() (string, bool) {
	r, _ := t.dialoguePart()
	return r.text, r.ok
}

// dialoguePart is the campaign's current section part and the audience it
// was read for.
func (t *secondCampaignScreen) dialoguePart() (dialogueRead, EventAudience) {
	c := t.state()
	if c == nil || c.talk.payload == nil {
		return dialogueRead{}, EventAudience{}
	}
	audience := partyAudience(t.session.Carried, nil)
	audience.tags = t.tags
	return c.talk.read(audience, nil), audience
}

// secondTownTalkProfile is the second game's town talk profile: no frame for
// the frame composer, a font-less window still drawn, the button offered at
// any size, and one show while the window is open.
var secondTownTalkProfile = ui.TownTalkProfile{}

// talkView is the town talk as this game reads it: the campaign's current
// section part, trimmed, with the portrait shape when the section names a
// speaker and the face of the speaker the part's tag names.
func (t *secondCampaignScreen) talkView() ui.TownTalkView {
	r, audience := t.dialoguePart()
	if !r.ok {
		return ui.TownTalkView{Profile: secondTownTalkProfile}
	}
	page := ui.TownTalkPage{Text: strings.TrimSpace(r.text),
		Portrait: EventHasSpeaker(dialoguePayload(t.state().talk.payload))}
	if speaker, named := audience.tagRules().speaker(r.tag); named && page.Portrait {
		page.Face, page.FaceWindow, _ = t.talkFaces.SpeakerFace(speaker)
	}
	return ui.TownTalkView{Profile: secondTownTalkProfile, Open: true, Page: page,
		Art: ui.TownTalkArt{Words: t.install.Words, Frame: t.frame(), Font: t.install.Font.Value()}}
}

// setTownDialogue opens a conversation and its speaker resolver.
func (t *secondCampaignScreen) setTownDialogue(payload []byte) {
	c := t.state()
	c.talk.open(payload)
	t.talkFaces = t.townSpeakers()
	t.openTalk(t, false)
}

// townSpeakers resolves a town speaker over the carried party as a mission
// does over its world.
func (t *secondCampaignScreen) townSpeakers() speakerResolver {
	party := t.session.Carried
	cast := newSpeakerCast(party, nil)
	cast.worn = func(id sim.EntityID) data.Equipment {
		if int(id) < 0 || int(id) >= len(party) {
			return data.Equipment{}
		}
		return mapload.EquipmentFromParty(party[id])
	}
	r := speakerResolver{units: t.install.Units, npcFaces: t.install.NPCFaces,
		portraits: make(map[string]*image.RGBA), figurePics: make(map[figureCacheKey]*image.RGBA), cast: cast}
	if t.install.Archives != nil {
		r.src = t.install.Archives.Containers
	}
	return r
}

// turnTownTalk turns the campaign's page; a part with no body closes the
// conversation. Any held campaign turns, open or not.
func (t *secondCampaignScreen) turnTownTalk() bool {
	c := t.state()
	if c == nil {
		return false
	}
	c.talk.next()
	if _, ok := t.dialogueBody(); !ok {
		c.talk.close()
	}
	return true
}

func (t *secondCampaignScreen) CanSave() bool { return t.state().savePoint() }

func (c *secondCampaign) savePoint() bool {
	return c != nil && c.talk.payload == nil && c.talk.part == 0 && captureSecondCampaign(c).validateTown() == nil
}

// noticePending reports whether the campaign holds an open dialogue.
func (c *secondCampaign) noticePending() bool {
	return c != nil && (c.talk.payload != nil || c.talk.part != 0)
}

// readSecondTownTalk reads the npc%dtalk%d section a TALK dispatches; a
// missing section shows no conversation (DIV-2635).
func readSecondTownTalk(install *InstallResources, key string) ([]byte, error) {
	payload, err := install.Archives.Containers.ReadFile(mainPrefix + "text/town.txt")
	if err != nil {
		return nil, fmt.Errorf("initial campaign town: %w", err)
	}
	body := secondGameTextSection(install.textCode().Bytes(payload), key)
	if body == "" {
		return nil, nil
	}
	return []byte(body), nil
}

// TownMusic is the scene the campaign screen requests: a town while one is
// current, else the destination map (R2-ENGINE-335).
func (t *secondCampaignScreen) TownMusic() (ui.MusicScene, bool) {
	if c := t.state(); c != nil && c.current.kind == placeTown {
		return ui.MusicTown, false
	}
	return ui.MusicCampaign, false
}

// TownMusicTrack is the current town's own track, as the game's music
// description names it by town ID (R2-ENGINE-231). The tavern keeps it: no
// tavern body requests music (R2-ENGINE-335).
func (t *secondCampaignScreen) TownMusicTrack() (string, bool) {
	c := t.state()
	if c == nil || c.current.kind != placeTown {
		return "", false
	}
	return GameMusic(t.install.profile()).TownTrack(c.current.id)
}
