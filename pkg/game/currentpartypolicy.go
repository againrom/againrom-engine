package game

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

type currentPartyPolicy struct {
	Temporary, PlayerCharacter                bool
	MercenaryType                             uint8
	CompanionNPC                              int
	Body, BodyDir, FigureDir                  []byte
	Mage                                      bool
	HealthColumn, ManaColumn                  bool
	UnitFighter                               bool
	Face                                      *currentPartyFace `json:",omitempty"`
	HiredRotationSpeed                        int32
	SpellbookRestored, LegacySpellbookPresent bool
	StartingWeaponSpent                       bool `json:"WeaponMaterialized"`
	Carry, Saved                              bool
}

type currentPartyFace struct {
	Wire   uint8
	TypeID uint16
	Value  int
}

func ordinaryPartyFace(c sav.DocumentCharacter) int {
	face := c.Face
	if c.Character.Basis.Human.TypeID < 0x1a {
		face &= 0x7f
	}
	return int(face)
}

// capturePartyPolicy records p's policy. The body name is the one the game
// without mods records for p (data.BodyList.SavedBody over the table's mod
// bodies); a nil table records p's own.
func capturePartyPolicy(p mapload.PartyMember, c sav.DocumentCharacter, table *mapload.Table) *currentPartyPolicy {
	body := p.Body
	if table != nil {
		body = string(table.Mods.Bodies.SavedBody(data.HeroBody(p.Body), p.Class))
	}
	out := &currentPartyPolicy{
		Temporary: p.Temporary, PlayerCharacter: p.PlayerCharacter, MercenaryType: p.MercenaryType,
		CompanionNPC: p.CompanionNPC, Body: []byte(body), BodyDir: []byte(p.BodyDir), FigureDir: []byte(p.FigureDir),
		Mage: p.Mage, HealthColumn: p.Profile.HealthColumn, ManaColumn: p.Profile.ManaColumn,
		HiredRotationSpeed: p.HiredRotationSpeed, SpellbookRestored: p.SpellbookRestored,
		LegacySpellbookPresent: p.SpellbookPresent, StartingWeaponSpent: p.WeaponMaterialized,
		Carry: p.Carry != nil, Saved: p.Saved != nil,
	}
	if c.Character.Class == "Unit" {
		out.UnitFighter = p.Profile.Fighter
	}
	if p.FigureFace != ordinaryPartyFace(c) {
		out.Face = &currentPartyFace{Wire: c.Face, TypeID: c.Character.Basis.Human.TypeID, Value: p.FigureFace}
	}
	return out
}

func (p currentPartyPolicy) member() mapload.PartyMember {
	out := mapload.PartyMember{
		Temporary: p.Temporary, PlayerCharacter: p.PlayerCharacter, MercenaryType: p.MercenaryType,
		CompanionNPC: p.CompanionNPC, Body: string(p.Body), BodyDir: string(p.BodyDir), FigureDir: string(p.FigureDir),
		Mage: p.Mage, Profile: data.Profile{Fighter: p.UnitFighter, HealthColumn: p.HealthColumn, ManaColumn: p.ManaColumn},
		HiredRotationSpeed: p.HiredRotationSpeed, SpellbookRestored: p.SpellbookRestored,
		SpellbookPresent: p.LegacySpellbookPresent,
	}
	if p.StartingWeaponSpent {
		materializeStartingWeapon(&out)
	}
	if p.Carry {
		out.Carry = &mapload.Carry{}
	}
	if p.Saved {
		out.Saved = &mapload.Saved{}
	}
	return out
}

func (p currentPartyPolicy) face(c sav.DocumentCharacter) int {
	if p.Face != nil && p.Face.Wire == c.Face && p.Face.TypeID == c.Character.Basis.Human.TypeID {
		return p.Face.Value
	}
	return ordinaryPartyFace(c)
}
