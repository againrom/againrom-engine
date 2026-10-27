package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
)

type currentSecondLocation struct{ Kind, ID int }
type currentSecondCampaign struct {
	Bank      [1024]int32
	Current   currentSecondLocation
	Available []currentSecondLocation
	Room      *secondTownRoom      `json:",omitempty"`
	Aux       *secondAux           `json:",omitempty"`
	Shop      *currentSecondShop   `json:",omitempty"`
	Party     []currentPartyMember `json:",omitempty"`
}

const maxSecondAvailable = 130

func (c *currentSecondCampaign) UnmarshalJSON(raw []byte) error {
	var fields struct {
		Bank      json.RawMessage
		Current   currentSecondLocation
		Available json.RawMessage
		Room      *secondTownRoom
		Aux       json.RawMessage
		Shop      *currentSecondShop
		Party     []currentPartyMember
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&fields); err != nil {
		return err
	}
	bank, err := currentScriptRows[int32](fields.Bank, 1024)
	if err != nil || len(bank) != 1024 {
		return fmt.Errorf("current second campaign bank must contain 1024 values")
	}
	available, err := currentScriptRows[currentSecondLocation](fields.Available, maxSecondAvailable)
	if err != nil {
		return err
	}
	next := currentSecondCampaign{Current: fields.Current, Available: available, Room: fields.Room, Shop: fields.Shop, Party: fields.Party}
	copy(next.Bank[:], bank)
	if next.Aux, err = decodeSecondAux(fields.Aux); err != nil {
		return err
	}
	if err := next.validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// decodeSecondAux reads exactly eight entries of exactly five fields. An
// absent or all-zero array decodes as absent, the form capture writes.
func decodeSecondAux(raw json.RawMessage) (*secondAux, error) {
	if raw == nil {
		return nil, nil
	}
	rows, err := currentScriptRows[json.RawMessage](raw, 8)
	if err != nil || rows == nil || len(rows) != 8 {
		return nil, fmt.Errorf("current second campaign auxiliary array must hold 8 entries")
	}
	var aux secondAux
	for i, row := range rows {
		fields, err := currentScriptRows[int32](row, 5)
		if err != nil || len(fields) != 5 {
			return nil, fmt.Errorf("current second campaign auxiliary entry must hold 5 fields")
		}
		copy(aux[i][:], fields)
	}
	if aux == (secondAux{}) {
		return nil, nil
	}
	return &aux, nil
}

func (c *currentSecondCampaign) validate() error {
	if err := validateSecondParty(c.Party); err != nil {
		return err
	}
	if c.Shop != nil {
		if err := c.Shop.validate(); err != nil {
			return err
		}
		units := secondBagUnits(c.Party)
		for _, it := range c.Shop.Hero {
			units += it.Quantity
		}
		if units > maxSecondBagUnits {
			return fmt.Errorf("current second campaign bag holds too many units")
		}
	}
	if c.Current.Kind == 2 {
		return c.validateTown()
	}
	mission := c.Current
	if mission.Kind != 1 || !secondSaveMission(mission.ID) || c.Room != nil {
		return fmt.Errorf("current second campaign requires a supported ordinary mission")
	}
	if mission.ID == 10 || mission.ID == 20 { // R2-SESSION-023, DIV-2633
		if len(c.Available) != 1 || c.Available[0] != mission {
			return fmt.Errorf("current second campaign requires the selected mission")
		}
	} else {
		return c.validateOrdinaryAvailable()
	}
	return nil
}

func secondSaveMission(id int) bool { return id > 0 && id < 128 }

func (c *currentSecondCampaign) validateOrdinaryAvailable() error {
	// R2-ENGINE-149, R2-ENGINE-330, DIV-2633: bank 775 zero, towns 1..3.
	if c.Bank[775] != 0 || len(c.Available) == 0 || len(c.Available) > maxSecondAvailable {
		return fmt.Errorf("current second campaign requires bounded ordinary availability")
	}
	seen := make(map[currentSecondLocation]bool, len(c.Available))
	for _, location := range c.Available {
		valid := location.Kind == 1 && secondSaveMission(location.ID) || location.Kind == 2 && location.ID >= 1 && location.ID <= 3
		if !valid || seen[location] {
			return fmt.Errorf("current second campaign contains an invalid or duplicate destination")
		}
		seen[location] = true
	}
	if !seen[c.Current] {
		return fmt.Errorf("current second campaign requires the selected mission")
	}
	return nil
}

func (c *currentSecondCampaign) validateTown() error {
	// R2-ENGINE-231, R2-SESSION-077, DIV-2412, DIV-2633.
	if c.Current == (currentSecondLocation{2, 2}) || c.Current == (currentSecondLocation{2, 3}) {
		if c.Room == nil || *c.Room > secondTownShop {
			return fmt.Errorf("current second campaign requires a quiet later town")
		}
		return c.validateOrdinaryAvailable()
	}
	town, mission := (currentSecondLocation{2, 1}), (currentSecondLocation{1, 10})
	if c.Current != town || c.Bank[768] != 10 || c.Room == nil || *c.Room > secondTownShop || len(c.Available) < 1 || len(c.Available) > 2 || c.Available[0] != town || len(c.Available) == 2 && c.Available[1] != mission {
		return fmt.Errorf("current second campaign requires a quiet first town")
	}
	return nil
}

func captureSecondCampaign(c *secondCampaign) *currentSecondCampaign {
	if c == nil {
		return nil
	}
	out := &currentSecondCampaign{Bank: c.bank, Current: currentSecondLocation{c.current.kind, c.current.id}, Shop: captureSecondShop(c.shop, c.legacyHero), Party: captureSecondParty(c)}
	if c.current.kind == placeTown {
		room := c.room
		out.Room = &room
	}
	for _, l := range c.available {
		out.Available = append(out.Available, currentSecondLocation{l.kind, l.id})
	}
	return out
}
func (c *currentSecondCampaign) clone() *currentSecondCampaign {
	if c == nil {
		return nil
	}
	out := *c
	out.Available = slices.Clone(c.Available)
	out.Shop = c.Shop.clone()
	if c.Party != nil {
		out.Party = make([]currentPartyMember, len(c.Party))
		for i, p := range c.Party {
			out.Party[i] = captureCurrentParty(p.Entity, p.restore())
		}
	}
	if c.Aux != nil {
		aux := *c.Aux
		out.Aux = &aux
	}
	if c.Room != nil {
		room := *c.Room
		out.Room = &room
	}
	return &out
}
func (c *currentSecondCampaign) restore() *secondCampaign {
	if c == nil {
		return nil
	}
	out := &secondCampaign{bank: c.Bank, current: partyPlace{c.Current.Kind, c.Current.ID}, shop: c.Shop.restore(), party: restoreSecondParty(c.Party)}
	if c.Shop != nil {
		out.legacyHero = slices.Clone(c.Shop.Hero)
	}
	if c.Aux != nil {
		// A document with the array predates the records in the shop: they
		// are rebuilt from the bank's departures (R2-ENGINE-330).
		out.shop.records = replayRecords(c.Bank)
	}
	if c.Room != nil {
		out.room = *c.Room
	}
	for _, l := range c.Available {
		out.available = append(out.available, partyPlace{l.Kind, l.ID})
	}
	return out
}

func validateAuthoredGame(doc *sav.DocumentData, a *currentActionData, game base.Game) error {
	var authored base.Game
	if a != nil && a.Session != nil {
		authored = a.Session.Game
	}
	if campaignOf(game).ownsGame(authored) != nil {
		return fmt.Errorf("save game %s does not match installed game %s", authored.Normal(), game.Normal())
	}
	return campaignOf(authored).validateAuthored(doc, a)
}

// refuseSecondFileDebt refuses, before any state changes, what the engine
// cannot yet hold of a second-game file: a save made during a mission
// (DIV-4155) and a book spell past the engine's book (DIV-4156).
func refuseSecondFileDebt(sf *sav.File) error {
	if sf.Head.Mission != 0 {
		return fmt.Errorf("a second-game save made during mission %d cannot be loaded yet", sf.Head.Mission)
	}
	chars, err := sf.Party()
	if err != nil {
		return err
	}
	slots := len(sim.Spellbook{}.Slots)
	for _, c := range chars {
		for _, spell := range c.Spells {
			if int(spell.ID) > slots {
				return fmt.Errorf("second-game save: %s's book holds spell %d; the engine's book holds %d", c.Name, spell.ID, slots)
			}
		}
	}
	return nil
}

// secondFileSession is the current-session record a native second-game file
// states through its campaign tail: the Scenario bank, the four shop
// records (R2-SESSION-156, R2-SESSION-187), the availability records and
// the current one (R2-SESSION-238, R2-SESSION-155), and the Player-1 hero's
// bag (R2-SESSION-236). The tail stores no town room; a town opens at its
// square (DIV-4153).
func secondFileSession(sf *sav.File) (*currentSessionData, error) {
	tail, err := sf.SecondTail()
	if err != nil {
		return nil, err
	}
	current, ok := tail.CurrentLocation()
	if !ok {
		return nil, fmt.Errorf("second-game save names no current location")
	}
	var shop secondShop
	for l, record := range tail.Shops {
		for k, cell := range record {
			shop.records[l][k] = secondshop.Category(cell)
		}
	}
	c := &currentSecondCampaign{Bank: tail.Bank, Current: currentSecondLocation{int(current.Kind), int(current.ID)},
		Shop: captureSecondShop(shop, nil)}
	if uint64(len(tail.Locations)) > maxSecondAvailable {
		return nil, fmt.Errorf("second-game save holds %d locations", len(tail.Locations))
	}
	for _, l := range tail.Locations {
		c.Available = append(c.Available, currentSecondLocation{int(l.Kind), int(l.ID)})
	}
	if c.Current.Kind == placeTown {
		room := secondTownSquare
		c.Room = &room
	}
	bag, err := secondFileBag(sf)
	if err != nil {
		return nil, err
	}
	if len(bag) != 0 {
		hero := secondHero()
		hero.CarriedItems = bag
		c.Party = []currentPartyMember{captureCurrentParty(0, hero)}
	}
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("second-game save: %w", err)
	}
	return &currentSessionData{Second: c}, nil
}

// secondFileBag is the Player-1 hero's bag as the second campaign's bag
// units: one unit per stacked piece, each the bag's item form.
func secondFileBag(sf *sav.File) ([]sim.ItemInstance, error) {
	chars, err := sf.Party()
	if err != nil {
		return nil, err
	}
	return secondHeroBag(chars)
}

func secondHeroBag(chars []sav.Character) ([]sim.ItemInstance, error) {
	var out []sim.ItemInstance
	for _, c := range chars {
		if !c.Hero {
			continue
		}
		for _, p := range c.Items {
			unit := sim.ItemInstance{Code: p.Code, Kind: p.Kind, Price: p.Price}
			for _, e := range p.Effects {
				unit.Effects = append(unit.Effects, sim.ItemEffect{Kind: e.Kind, Mode: e.Mode, Operand: e.Operand})
			}
			if _, ok := secondshop.FromInstance(unit); !ok {
				return nil, fmt.Errorf("second-game save: bag item %#04x has no bag form", p.Code)
			}
			n := max(int(p.Stack), 1)
			if len(out)+n > maxSecondBagUnits {
				return nil, fmt.Errorf("second-game save: bag holds more than %d units", maxSecondBagUnits)
			}
			for range n {
				out = append(out, unit.Clone())
			}
		}
	}
	return out, nil
}

// envelopeOf is the game the envelope of b names; empty for Asg& and for
// bytes no envelope opens.
func envelopeOf(b []byte) string {
	if p, ok := sav.ProfileOf(b); ok {
		return p.Game
	}
	return ""
}

// ownsEnvelope reports whether the envelope names a game, and game's
// campaign's refusal when that game is another (R2-SESSION-017, DIV-3079).
// The Asg& envelope, which the first game's originals and both games'
// engine documents share, names none.
func ownsEnvelope(envelope string, game base.Game) (named bool, err error) {
	if envelope == "" {
		return false, nil
	}
	return true, campaignOf(game).ownsGame(base.Game(envelope))
}

func validateOriginalGame(saved []byte, game base.Game) error {
	// A native second-game file has no engine document: its envelope names
	// its game.
	if named, err := ownsEnvelope(envelopeOf(saved), game); named {
		return err
	}
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		if refusal := campaignOf(game).undecodedSave(err); refusal != nil {
			return refusal
		}
		sf, parseErr := sav.Open(saved)
		if parseErr != nil {
			return parseErr
		}
		if sf.HasNativeActionsLeaf() {
			return fmt.Errorf("authored current SAV document is malformed: %w", err)
		}
		return nil
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		return err
	}
	return validateAuthoredGame(&doc, a, game)
}
