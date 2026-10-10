package game

import (
	"bytes"
	"encoding/json"
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
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

// secondGeneratorCampaign is the campaign part of the second game's
// description. A part that does not decode is a build defect.
var secondGeneratorCampaign = func() *secondGenerator {
	g, err := decodeSecondGenerator(generatorDescriptions["rom2"].Campaign)
	if err != nil {
		panic(err)
	}
	return g
}()

// generatorTemplate is the Humans row a hero picture names, by name.
func generatorTemplate(humans data.Collection, name string) (data.HumanDef, int, bool) {
	i := data.FindHumanByName(humans, name)
	if name == "" || i == data.NotFound {
		return data.HumanDef{}, data.NotFound, false
	}
	d, err := data.NewHumanDef(name, humans.EntryParams(i))
	if err != nil {
		return data.HumanDef{}, data.NotFound, false
	}
	return d, i, true
}

// heroTemplate is the template name of the picture standing for a sex and a
// class.
func heroTemplate(l *ui.GeneratorDescription, female, mage bool) string {
	for _, h := range l.PreCreate.Heroes {
		if (h.Sex != 0) == female && (h.Class != 0) == mage {
			return h.Template
		}
	}
	return ""
}

// secondGeneratorSetup gives each picture its template row's statistics as
// its preset (R2-ENGINE-285, R2-ASSET-076), and an accepted result shows the
// first town.
func secondGeneratorSetup(f *FrontEnd, setup *ui.ChargenSetup) {
	setup.AcceptShowsTown = true
	l := f.generator()
	for i, h := range l.PreCreate.Heroes {
		if i >= len(setup.Presets) {
			break
		}
		if d, _, ok := generatorTemplate(missionHumans(f.Table), h.Template); ok {
			setup.Presets[i] = []int{int(d.Body), int(d.Reaction), int(d.Mind), int(d.Spirit)}
		}
	}
}

// secondGeneratorWeapon is the weapon the producer equips for a class, a sex
// and a chosen school skill: the fighter's weapon of that skill, or the
// mage's staff casting that school's spell (R2-ENGINE-290).
func secondGeneratorWeapon(g *secondGenerator, t *mapload.Table, mage, female bool, skill int) *data.Weapon {
	if t == nil || skill < 0 || skill >= len(g.FighterWeapons) {
		return nil
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
		return nil
	}
	return w
}

// secondGeneratorParty is the second game's producer: the template row of
// the chosen picture with the result's statistics, school skills cleared,
// the fifth skill and the chosen skill at their levels, and the chosen
// skill's weapon in place of the row's own (R2-ENGINE-290). Item modifiers
// and the experience the original computes are not carried (DIV-2772).
func secondGeneratorParty(f *FrontEnd, res ui.ChargenResult) []mapload.PartyMember {
	g := secondGeneratorCampaign
	female := chargenChoiceIndex(res, chargenChoiceSex) != 0
	mage := chargenChoiceIndex(res, chargenChoiceClass) != 0
	skill := chargenChoiceIndex(res, chargenChoiceSkill)
	slot := data.SkillBlade + int32(skill)

	humans := missionHumans(f.Table)
	base, i, ok := generatorTemplate(humans, heroTemplate(f.generator(), female, mage))
	profile, face, book, cells := chargenProfile(base, ok, humans, i, mage)
	spread := data.Spread{
		Body:     chargenStatValue(res, chargenStatBody),
		Reaction: chargenStatValue(res, chargenStatReaction),
		Mind:     chargenStatValue(res, chargenStatMind),
		Spirit:   chargenStatValue(res, chargenStatSpirit),
	}
	party := assembleParty(partyInputs{
		Name:    res.Name,
		Spread:  spread,
		Slot:    slot,
		Mage:    mage,
		Profile: profile,
		Dir:     data.FigureDirFor(mage, female),
		Face:    face,
		Book:    book,
		Weapon:  secondGeneratorWeapon(g, f.Table, mage, female, skill),
		Cells:   cells,
		List:    f.Bodies,
		Table:   f.Table,
	})
	h := &party[0].Hero
	for s := data.SkillBlade; s < data.SkillSlots; s++ {
		h.Skill[s] = 0
	}
	h.Skill[data.SkillSlots-1] = g.FifthSkill
	if slot > data.SkillGeneral && slot < data.SkillSlots {
		h.Skill[slot] = g.ChosenSkill
	}
	return party
}

// startSecondGenerated is the second game's generator commit: a new campaign
// at its first town with the hero the result makes, the character-choice
// slots set from the class and the sex, the session difficulty and the
// Player's opening gold (R2-ENGINE-283, R2-SESSION-077, R2-SESSION-131).
func (f *FrontEnd) startSecondGenerated(res ui.ChargenResult) error {
	g := secondGeneratorCampaign
	level, err := campaignDifficulty(int64(res.Difficulty))
	if err != nil {
		return err
	}
	if f.Archives == nil || f.Archives.Containers == nil {
		return fmt.Errorf("initial campaign town: no install")
	}
	if _, err := f.Archives.Containers.ReadFile(mainPrefix + "text/town.txt"); err != nil {
		return fmt.Errorf("initial campaign town: %w", err)
	}
	party := f.ChargenParty(res)
	f.startSecondCampaignWith(party)
	f.Difficulty = level
	f.Town.gold = g.Gold
	bank := &f.Town.second.bank
	bank[g.MageSlot] = boolDWORD(chargenChoiceIndex(res, chargenChoiceClass) != 0)
	bank[g.FemaleSlot] = boolDWORD(chargenChoiceIndex(res, chargenChoiceSex) != 0)
	return nil
}

func boolDWORD(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
