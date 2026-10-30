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
	// talking is the inn talk the open conversation shows, nil when none
	// is open; its end runs endInnTalk.
	talking *innOffer
	// offers are the inn rows of the mods' layered book the town screen
	// binds, nil for the shipped book's; they are not saved.
	offers []town.OfferRow
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

// innAct runs inn seat i: Hire or Fire on a hire actor, TALK on a talk
// actor (R2-ENGINE-496).
func (t *townScreen) innAct(c *secondCampaign, i int) ui.TownAction {
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
func (t *townScreen) talk(c *secondCampaign, o innOffer) ui.TownAction {
	payload, err := readSecondTownTalk(t.in, secondTalkKey(o.npc, o.topic))
	if err != nil {
		return ui.TownAction{Msg: err.Error()}
	}
	c.talkTo(o)
	if payload == nil {
		t.endInnTalk(o)
		return ui.TownAction{}
	}
	c.talking = &o
	t.setTownDialogue(payload)
	return ui.TownAction{}
}

// endInnTalk is the end of the inn talk o: the party gains each identity
// the edition adds at that talk's end (DIV-4186).
func (t *townScreen) endInnTalk(o innOffer) {
	t.sess.addTalkIdentities(t.in.townInstall(), o.npc, o.topic)
	t.composed.patrons = nil
}

// setTownDialogue opens a conversation and its speaker resolver.
func (t *townScreen) setTownDialogue(payload []byte) {
	if t.adapter == nil {
		t.adapter = destinationTownCampaign{t}
		t.defaults = townDescriptions["rom2-town1"]
	}
	c := t.state()
	c.talk.open(payload)
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

func (c *secondCampaign) savePoint() bool {
	return c != nil && c.talk.payload == nil && c.talk.part == 0 && captureSecondCampaign(c).validateTown() == nil
}

// noticePending reports whether the campaign holds an open dialogue.
func (c *secondCampaign) noticePending() bool {
	return c != nil && (c.talk.payload != nil || c.talk.part != 0)
}

// secondTownText is the town text file the inn talks are sections of.
const secondTownText = mainPrefix + "text/town.txt"

// readSecondTownTalk reads the npc%dtalk%d section a TALK dispatches; a
// missing section shows no conversation (DIV-2635).
func readSecondTownTalk(install *InstallResources, key string) ([]byte, error) {
	payload, err := install.Archives.Containers.ReadFile(secondTownText)
	if err != nil {
		return nil, fmt.Errorf("initial campaign town: %w", err)
	}
	body := secondGameTextSection(install.textCode().Bytes(payload), key)
	if body == "" {
		return nil, nil
	}
	return []byte(body), nil
}
