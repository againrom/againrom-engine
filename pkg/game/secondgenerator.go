package game

import (
	"bytes"
	"encoding/json"
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondGenerator is the campaign part of the second game's generator
// description: the school skill levels and the weapon the producer gives the
// hero a result makes, and the two character-choice bank slots and the
// opening gold the commit writes (R2-ENGINE-290, R2-ENGINE-283,
// R2-SESSION-131).
type secondGenerator struct {
	ChosenSkill     int32    `json:"chosen-skill"`
	FifthSkill      int32    `json:"fifth-skill"`
	FighterWeapons  []string `json:"fighter-weapons"`
	FemaleStaff     string   `json:"female-staff"`
	MaleStaff       string   `json:"male-staff"`
	StaffSpells     []string `json:"staff-spells"`
	StaffSpellLevel int      `json:"staff-spell-level"`
	MageSlot        int      `json:"mage-slot"`
	FemaleSlot      int      `json:"female-slot"`
	Gold            int      `json:"gold"`
	Cite            []string `json:"cite"`
}

// decodeSecondGenerator reads a description's campaign part strictly.
func decodeSecondGenerator(raw json.RawMessage) (*secondGenerator, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var g secondGenerator
	if err := dec.Decode(&g); err != nil {
		return nil, fmt.Errorf("second generator campaign: %w", err)
	}
	var bank [1024]int32
	switch {
	case len(g.FighterWeapons) != data.SkillSlots-1 || len(g.StaffSpells) != data.SkillSlots-1:
		return nil, fmt.Errorf("second generator campaign: %d weapons and %d staff spells", len(g.FighterWeapons), len(g.StaffSpells))
	case g.MageSlot < 0 || g.MageSlot >= len(bank) || g.FemaleSlot < 0 || g.FemaleSlot >= len(bank):
		return nil, fmt.Errorf("second generator campaign: bank slots %d and %d", g.MageSlot, g.FemaleSlot)
	case g.Gold < 0:
		return nil, fmt.Errorf("second generator campaign: gold %d", g.Gold)
	}
	return &g, nil
}

// secondGeneratorCampaigns are the decoded campaign parts of the generator
// descriptions that carry one, by description name. A part that does not
// decode is a build defect.
var secondGeneratorCampaigns = func() map[string]*secondGenerator {
	out := map[string]*secondGenerator{}
	for name, d := range generatorDescriptions {
		if len(d.Campaign) == 0 {
			continue
		}
		g, err := decodeSecondGenerator(d.Campaign)
		if err != nil {
			panic(err)
		}
		out[name] = g
	}
	return out
}()

// secondGeneratorCampaign is the campaign part of the generator description
// the profile's edition names, or an error when that description has none.
func (f *FrontEnd) secondGeneratorCampaign() (*secondGenerator, error) {
	name := f.Base().Profile.Edition().Generator
	if g := secondGeneratorCampaigns[name]; g != nil {
		return g, nil
	}
	return nil, fmt.Errorf("generator %q has no second-game campaign part", name)
}

// generatorCampaignWeapon is the weapon a campaign part equips for a class, a
// sex and a chosen school skill: the fighter's weapon of that skill, or the
// mage's staff casting that school's spell (R2-ENGINE-290). The cell is the
// equipment cell the weapon came from, effects included.
func generatorCampaignWeapon(g *secondGenerator, t *mapload.Table, mage, female bool, skill int) (*data.Weapon, string) {
	if t == nil || skill < 0 || skill >= len(g.FighterWeapons) {
		return nil, ""
	}
	cell := g.FighterWeapons[skill]
	if mage {
		staff := g.MaleStaff
		if female {
			staff = g.FemaleStaff
		}
		cell = fmt.Sprintf("%s {castSpell=%s:%d}", staff, g.StaffSpells[skill], g.StaffSpellLevel)
	}
	_, _, w, err := mapload.HumanRowEquipment([]string{cell}, t)
	if err != nil {
		return nil, ""
	}
	return w, cell
}

// startGeneratedTown is the town start of a generator result: a new campaign
// at its first town with the hero the result makes, the character-choice
// slots set from the class and the sex, the session difficulty and the
// Player's opening gold, the slots and the gold read from the description's
// campaign part (R2-ENGINE-283, R2-SESSION-077, R2-SESSION-131).
func (f *FrontEnd) startGeneratedTown(res ui.ChargenResult) error {
	g, err := f.secondGeneratorCampaign()
	if err != nil {
		return err
	}
	level, err := campaignDifficulty(int64(res.Difficulty))
	if err != nil {
		return err
	}
	if f.Archives == nil || f.Archives.Containers == nil {
		return fmt.Errorf("initial campaign town: no install")
	}
	if err := installText(f.Archives.Containers, townTextSource.at(), TextCode{}).err(); err != nil {
		return fmt.Errorf("initial campaign town: %w", err)
	}
	party := f.ChargenParty(res)
	f.startSecondCampaignWith(party)
	f.Difficulty = level
	f.Town.gold = g.Gold
	bank := &f.Town.second.bank
	sim.SetScenario(bank, int64(g.MageSlot), boolDWORD(chargenChoiceIndex(res, chargenChoiceClass) != 0))
	sim.SetScenario(bank, int64(g.FemaleSlot), boolDWORD(chargenChoiceIndex(res, chargenChoiceSex) != 0))
	return nil
}

func boolDWORD(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
