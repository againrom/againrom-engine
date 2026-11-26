package game

import (
	"bytes"
	"fmt"
	"math"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/a2c"
	"againrom/pkg/formats/alch"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// storedCharacter is what a map entry takes from a character file: the
// generator answers the record fixes, its purse and its per-skill experience
// (DIV-5247), and the file's bytes as read.
type storedCharacter struct {
	result  ui.ChargenResult
	purse   uint32
	skillXP [data.SkillSlots]int32
	// hasXP is a record that states experience; a record of zeros keeps the
	// generator's start skills.
	hasXP bool
	// doc is the loaded document a rewrite keeps; nil is a new file.
	doc []byte
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

// sessionFile is one listed character file of the game's row in store, and
// what this process's sessions carried for the row's files.
type sessionFile struct {
	store   CharacterStore
	row     characterStoreRow
	name    string
	carried sessionCarries
}

// sessionCarry is what a session's end leaves for the next entry of its file
// in this process: the bytes it wrote, and the party and purse they were
// written from, which also hold what the file cannot (DIV-5259).
type sessionCarry struct {
	doc   []byte
	party []mapload.PartyMember
	purse uint32
}

// sessionCarries are the carries by listed file name.
type sessionCarries map[string]sessionCarry

// mapSession is one multiplayer session the front end runs in-process: the
// stored character, the party it makes, its purse, the chosen map and the
// file the character came from.
type mapSession struct {
	character storedCharacter
	party     []mapload.PartyMember
	purse     uint32
	entry     MapEntry
	file      sessionFile
	// end writes the gains of the map's party back to the file (DIV-5258)
	// and answers the line the character list shows.
	end func(ms *Mission) string
}

// newMapSession reads file and pairs it with source, which must be a map of
// the game's multiplayer map choice. When this process's last session of the
// file wrote the bytes it holds, the party and purse are the ones that
// session carried; otherwise they are the file's.
func (f *FrontEnd) newMapSession(file sessionFile, source string) (*mapSession, error) {
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
	c, err := f.readStoredCharacter(file.store, file.row, file.name)
	if err != nil {
		return nil, err
	}
	party, purse := f.storedParty(c), c.purse
	if k, ok := file.carried[file.name]; ok && bytes.Equal(k.doc, c.doc) {
		party, purse = mapload.CloneParty(k.party), k.purse
	}
	if len(party) == 0 {
		return nil, fmt.Errorf("no party for the stored character")
	}
	s := &mapSession{character: c, party: party, purse: purse, entry: kept[at], file: file}
	s.end = func(ms *Mission) string { return f.endSession(s, ms) }
	return s, nil
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
	return f.prepareFreshGame(missionRequest{party: s.party, initialPurse: &purse, fill: sessionEntry(s)}, 0)
}

// advance is a session map's advance over the driver's: a won map's notice,
// which the driver answers with the map list, and a leave through the game
// menu end the session for the player on the character list (DIV-5260).
// Every other answer is the driver's.
func (s *mapSession) advance(ms *Mission, driver ui.MapAdvance) ui.MapAdvance {
	return func(actions ...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
		if len(actions) > 0 && actions[0] == ui.NoticeLeave {
			return ui.NoticeToCharacters, s.end(ms), nil
		}
		dest, msg, open := driver(actions...)
		if dest == ui.NoticeToMapList {
			return ui.NoticeToCharacters, s.end(ms), nil
		}
		return dest, msg, open
	}
}

// endSession writes the gains of ms's party back to s's file through the
// character store: the survivors are normalized and carried as a won
// mission's are, and the carried character's statistics, the purse and its
// skill experience become the record over the loaded document. A character
// that did not survive leaves the file as it was. A refusal is the answer.
func (f *FrontEnd) endSession(s *mapSession, ms *Mission) string {
	if ms == nil || ms.World == nil {
		return ""
	}
	if err := ms.World.NormalizeMissionSurvivors(sim.SelfSlot); err != nil {
		return fmt.Sprintf("the character was not written: %v", err)
	}
	party := carriedParty(ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster)
	if len(party) == 0 || party[0].Carry == nil || len(ms.Start.IDs) == 0 {
		return ""
	}
	if hero, ok := ms.World.Entity(ms.Start.IDs[0]); !ok || !hero.Alive() {
		return ""
	}
	gains := s.character
	gains.result.Stats = slices.Clone(gains.result.Stats)
	h := party[0].Hero
	for i, v := range [4]int32{h.Body, h.Reaction, h.Mind, h.Spirit} {
		gains.result.Stats[characterStats[i]] = int(v)
	}
	gains.purse = ms.World.Purse(sim.SelfSlot)
	gains.skillXP = party[0].Carry.SkillXP
	_, doc, err := f.writeCharacter(s.file.row, s.file.store, gains, s.file.name)
	if err != nil {
		return fmt.Sprintf("the character was not written: %v", err)
	}
	if s.file.carried != nil {
		s.file.carried[s.file.name] = sessionCarry{doc: doc, party: mapload.CloneParty(party), purse: gains.purse}
	}
	return ""
}
