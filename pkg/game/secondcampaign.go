package game

import (
	"fmt"
	"image"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/town"
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
	shop      Shop
	// party is the hero alone; his bag is the shop's customer.
	party []mapload.PartyMember
	// legacyHero is an older document's hero list waiting for the
	// install's tables to load into the bag.
	legacyHero []secondHeroItem
	// kept is what a loaded tail held that a town SAVE writes back.
	kept *secondTailKept
	// mapOpen is the map's current opening (openMap); nil while the party
	// stands at a location and after a LOAD at the map.
	mapOpen *secondMapOpening
	// innBits is the inn page's copy of each option's high bit where Hire
	// or Fire changed it during the current visit (R2-SESSION-191).
	innBits map[int]bool
	// talking is the inn talk the open conversation shows, nil when none
	// is open; its end runs endInnTalk.
	talking *innOffer
	// offers are the inn rows of the mods' layered book the town screen
	// binds, nil for the shipped book's; they are not saved.
	offers []town.OfferRow
}

func newSecondCampaign() *secondCampaign {
	e, _ := secondBook().catalog.Town(1)
	return newTownState(e, Campaign{}, nil).bankOffers
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

// complete applies the ordinary departure to the bank w left, with the hires
// of party, entered as ids, that fell in w, and answers its output DWORD.
func (c *secondCampaign) complete(w *sim.World, party []mapload.PartyMember, ids []sim.EntityID) int {
	bank, _ := w.ROM2ScenarioState()
	return c.completeBank(bank, fallenHires(party, ids, w)...)
}

// completeBank applies the ordinary departure to the bank a won mission left,
// in which the fallen hires died, and answers the output DWORD: -1 unless a
// case body stores one (R2-ENGINE-145, R2-ENGINE-146, R2-ENGINE-148,
// R2-SESSION-047..050).
func (c *secondCampaign) completeBank(bank [1024]int32, fallen ...int) int {
	if c.shop.policy == nil {
		shop := newShopState(&bankShopProfile, shopRecordRows(c.shop.nativeRecords()))
		c.shop.policy, c.shop.records = shop.policy, shop.records
	}
	n := c.current.id
	output := -1
	c.bank = bank
	secondDepartureReset(&c.bank)
	departHires(&c.bank, fallen)
	sim.SetScenario(&c.bank, int64(896+n), 1)
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
	if row, ok := secondDepartureFor(n); ok {
		output = row.run(c, add)
	}
	c.shop.depart(n)
	if n%10 == 0 {
		sim.SetScenario(&c.bank, 768, c.bank[768]+10)
	}
	c.shop.stage(c.bank[768])
	c.openMap(c.current)
	c.current, c.room = partyPlace{}, secondTownSquare
	return output
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

func (t *townScreen) StartNewGame() error {
	return t.campaignTown().startCampaign()
}

// innAct dispatches the selected seat's action (Hire, Fire, TALK:
// R2-ENGINE-496; the room events: DIV-4372) or the first row, the town's
// one Sleep. i is an inn row: row 0 is Sleep and row i+1 is seat i.
func (t *townScreen) innAct(c *secondCampaign, row int) ui.TownAction {
	seats, hires := c.innSeats()
	i := row - 1
	if row == 0 {
		t.sleep()
		return ui.TownAction{}
	}
	if i < 0 || i >= len(seats) {
		return ui.TownAction{}
	}
	if i >= hires {
		t.raiseShownRoomEvent(roomEventTalk)
		return t.talk(c, seats[i])
	}
	npc := seats[i].npc
	fired := c.hired(npc)
	key := c.seatHire(npc)
	switch {
	case fired:
		t.raiseShownRoomEvent(roomEventDismiss)
	case c.hired(npc):
		t.raiseShownRoomEvent(roomEventHire)
	case key != "":
		t.raiseShownRoomEvent(roomEventHireShort)
	}
	if key != "" {
		return t.openTownText(key)
	}
	return ui.TownAction{}
}

// talk opens a speaker's conversation and runs its TalkTo.
func (t *townScreen) talk(c *secondCampaign, o innOffer) ui.TownAction {
	key := secondTalkKey(o.npc, o.topic)
	payload, err := readSecondTownTalk(t.in, key)
	if err != nil {
		return ui.TownAction{Msg: err.Error()}
	}
	c.talkTo(o)
	if payload == nil {
		t.endInnTalk(o)
		return ui.TownAction{}
	}
	c.talking = &o
	t.setTownDialogue(key, payload)
	return ui.TownAction{}
}

// endInnTalk is the end of the inn talk o: the party gains each identity
// the edition adds at that talk's end (DIV-4186).
func (t *townScreen) endInnTalk(o innOffer) {
	t.sess.addTalkIdentities(t.in.townInstall(), o.npc, o.topic)
	t.composed.patrons = nil
}

// setTownDialogue opens section key's conversation and its speaker resolver.
func (t *townScreen) setTownDialogue(key string, payload []byte) {
	if t.adapter == nil {
		t.adapter = destinationTownCampaign{t}
		t.defaults = townDescriptions["rom2-town1"]
	}
	t.talkPage().openSection(key, payload)
	t.resetTownSpeech()
	t.resolver = t.townSpeakers()
	t.openTalk(t, payload, false)
}

// townSpeakers resolves a town speaker over the carried party as a mission
// does over its world.
func (t *townScreen) townSpeakers() speakerResolver {
	party := t.sess.Carried
	cast := newSpeakerCast(party, nil)
	cast.worn = func(id sim.EntityID) data.Equipment {
		if int(id) < 0 || int(id) >= len(party) {
			return data.Equipment{}
		}
		return mapload.EquipmentFromParty(party[id])
	}
	r := speakerResolver{units: t.in.Units, npcFaces: t.in.NPCFaces,
		portraits: make(map[string]*image.RGBA), figurePics: make(map[figureCacheKey]*image.RGBA), cast: cast}
	r.campaignActors = map[int]mapload.PartyMember{}
	seats, _ := t.state().innSeats()
	for _, seat := range seats {
		_, controlled := tableEdition(t.in.Table).PartyIdentity(seat.npc)
		if !controlled && seat.kind != 1 && seat.kind != 2 {
			continue
		}
		if p, ok := t.campaignActor(seat.npc); ok {
			r.campaignActors[seat.npc] = p
		}
	}
	if t.in.Archives != nil {
		r.src = t.in.Archives.Containers
	}
	return r
}

func (t *Town) secondSavePoint() bool { return t != nil && t.second.savePoint(t.talk) }

// secondNoticePending reports whether the campaign holds an open dialogue.
func (t *Town) secondNoticePending() bool { return t != nil && t.second.noticePending(t.talk) }

func (c *secondCampaign) savePoint(talk talkCursor) bool {
	return c != nil && talk.payload == nil && talk.part == 0 && captureSecondCampaign(c).validateTown() == nil
}

func (c *secondCampaign) noticePending(talk talkCursor) bool {
	return c != nil && (talk.payload != nil || talk.part != 0)
}

// secondTownText is the town text file the inn talks are sections of.
const secondTownText = mainPrefix + "text/town.txt"

// readSecondTownTalk reads the npc%dtalk%d section a TALK dispatches; a
// missing section shows no conversation (DIV-2635).
func readSecondTownTalk(install *InstallResources, key string) ([]byte, error) {
	town := installText(install.Archives.Containers, townTextSource.at(), install.textCode())
	if err := town.err(); err != nil {
		return nil, fmt.Errorf("initial campaign town: %w", err)
	}
	body := town.section(key)
	if body == "" {
		return nil, nil
	}
	return []byte(body), nil
}
