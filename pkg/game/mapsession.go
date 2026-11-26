package game

import (
	"fmt"
	"math"

	"againrom/pkg/data"
	"againrom/pkg/formats/a2c"
	"againrom/pkg/formats/alch"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// storedCharacter is what a map entry takes from a character file: the
// generator answers the record fixes, its purse and its per-skill experience
// (DIV-5247).
type storedCharacter struct {
	result  ui.ChargenResult
	purse   uint32
	skillXP [data.SkillSlots]int32
	// hasXP is a record that states experience; a record of zeros keeps the
	// generator's start skills.
	hasXP bool
}

// maxStoredStat bounds a statistic read from a file to the byte the
// generator's pages hold.
const maxStoredStat = 255

// firstStoredCharacter reads an AlCh file: its name and four statistics. The
// short form holds no sex, class, skill or purse (HERO-213, DIV-5247).
func firstStoredCharacter(_ *FrontEnd, b []byte) (storedCharacter, error) {
	c, err := alch.Read(b)
	if err != nil {
		return storedCharacter{}, err
	}
	var stats [4]uint32
	copy(stats[:], c.Stats[:])
	return storedFrom(string(c.Name), make([]int, chargenChoiceCount), stats)
}

// secondStoredCharacter reads an .a2c file: the start unit row of A picks the
// sex and class, E gives the statistics, the money and the experience of
// skills one to five (R2-SESSION-514, R2-SESSION-517).
func secondStoredCharacter(f *FrontEnd, b []byte) (storedCharacter, error) {
	c, err := a2c.Read(b)
	if err != nil {
		return storedCharacter{}, err
	}
	start := a2c.StartRows[c.A.Unit>>6]
	choices, found := f.generatorChoices(start)
	if !found {
		return storedCharacter{}, fmt.Errorf("no generator row names the start unit row %s", start)
	}
	var stats [4]uint32
	for i, v := range c.E.Base {
		stats[i] = uint32(v)
	}
	s, err := storedFrom(string(c.A.Name), choices, stats)
	if err != nil {
		return storedCharacter{}, err
	}
	s.purse = c.E.Money
	for i, xp := range c.E.SkillXP {
		if xp > math.MaxInt32 {
			return storedCharacter{}, fmt.Errorf("skill %d experience %d is out of range", i+1, xp)
		}
		s.skillXP[data.SkillBlade+int32(i)] = int32(xp)
		s.hasXP = s.hasXP || xp != 0
	}
	return s, nil
}

// storedFrom is the generator result of a name, choices and the four
// statistics in characterStats order, refusing a statistic past the byte.
func storedFrom(name string, choices []int, stats [4]uint32) (storedCharacter, error) {
	res := ui.ChargenResult{Name: name, Choices: choices, Stats: make([]int, chargenStatCount)}
	for i, row := range characterStats {
		if stats[i] > maxStoredStat {
			return storedCharacter{}, fmt.Errorf("statistic %d is %d, past %d", i, stats[i], maxStoredStat)
		}
		res.Stats[row] = int(stats[i])
	}
	return storedCharacter{result: res}, nil
}

// mapSession is one multiplayer session the front end runs in-process: the
// stored character, the party it makes, its purse and the chosen map.
type mapSession struct {
	character storedCharacter
	party     []mapload.PartyMember
	purse     uint32
	entry     MapEntry
}

// newMapSession reads the listed character file name of row in store and
// pairs it with source, which must be a map of the game's multiplayer map
// choice.
func (f *FrontEnd) newMapSession(store CharacterStore, row characterStoreRow, name, source string) (*mapSession, error) {
	kept, _ := MultiplayerMaps(f.Maps, filesOf(f.savedGame()).multiplayerMaps)
	at := -1
	for i, e := range kept {
		if e.Source == source {
			at = i
		}
	}
	if at < 0 {
		return nil, fmt.Errorf("%s is not a multiplayer map", source)
	}
	c, err := f.readStoredCharacter(store, row, name)
	if err != nil {
		return nil, err
	}
	party := f.storedParty(c)
	if len(party) == 0 {
		return nil, fmt.Errorf("no party for the stored character")
	}
	return &mapSession{character: c, party: party, purse: c.purse, entry: kept[at]}, nil
}

// storedParty is the generator's party for the stored character. Where the
// record states experience, skills one to five hold it and the levels it
// gives, and the carry keeps it whole into the map.
func (f *FrontEnd) storedParty(c storedCharacter) []mapload.PartyMember {
	party := f.generatorParty(c.result)
	if len(party) == 0 || !c.hasXP {
		return party
	}
	p := mapload.MaterializePartyCarry(party[0], f.Table)
	for s := data.SkillBlade; s < data.SkillSlots; s++ {
		p.Carry.SkillXP[s] = c.skillXP[s]
		p.Hero.Skill[s] = data.SkillLevelFor(c.skillXP[s])
	}
	party[0] = p
	return party
}

// sessionOpener opens s's map through the new-game candidate path: the
// running game is replaced only when the map is adopted. The map start
// places the party at its drop list (DIV-5246); the map is no campaign
// mission, so it advances, hires and saves nothing (DIV-5249).
func (f *FrontEnd) sessionOpener(s *mapSession) (ui.MapOpener, error) {
	purse := s.purse
	return f.prepareFreshGame(missionRequest{party: s.party, initialPurse: &purse, session: s}, 0)
}
