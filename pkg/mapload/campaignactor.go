package mapload

import (
	"fmt"

	"againrom/pkg/data"
)

// CampaignActorProfile selects templates and membership without a game branch.
type CampaignActorProfile struct {
	HumanName                             string
	UnitName                              string
	ServerBase, ServerStride              int32
	StageDivisor, StageBias               int
	Temporary, PlayerCharacter            bool
	RowFigure, RowRotation, RowDefinition bool
}

type CampaignActorSpec struct {
	CharacterSpec
	Profile         CampaignActorProfile
	Identity, Stage int
	MercenaryType   uint8
	Hire            bool
}

// CampaignActor resolves one campaign member through the installed row decoders.
func CampaignActor(t *Table, s CampaignActorSpec, party []PartyMember) (PartyMember, bool) {
	if t == nil {
		return PartyMember{}, false
	}
	profile := s.Profile
	row, unit := s.Row, false
	if s.NPC != 0 && row == 0 && t.Edition != nil {
		if identity, ok := t.Edition.PartyIdentity(int(s.NPC)); ok {
			row = data.FindHumanByServerID(t.Humans, identity.ServerID)
			if row == 0 {
				return PartyMember{}, false
			}
		}
	}
	if profile.HumanName != "" {
		row = data.FindHumanByName(t.Humans, fmt.Sprintf(profile.HumanName, s.Identity, s.Stage))
		if row == 0 {
			return PartyMember{}, false
		}
	}
	if profile.UnitName != "" {
		unit = true
		row = campaignNamedRow(t.Units, profile.UnitName)
		if row == 0 {
			return PartyMember{}, false
		}
	}
	if profile.ServerBase != 0 {
		key := profile.ServerBase + profile.ServerStride*int32(s.Identity)
		row = campaignKeyRow(t.Humans, key, false)
		if row == 0 {
			row, unit = campaignKeyRow(t.Units, key, true), true
		}
		collection := t.Humans
		if unit {
			collection = t.Units
		}
		if profile.StageDivisor != 0 {
			for suffix := int32(1); suffix <= 9; suffix++ {
				if campaignKeyRow(collection, key+suffix, unit) != 0 {
					if variant := campaignKeyRow(collection, key+int32(profile.StageBias+s.Stage/profile.StageDivisor), unit); variant != 0 {
						row = variant
					}
					break
				}
			}
		}
	}
	var p PartyMember
	var ok bool
	if unit {
		if t.Units == nil || row < 1 || row >= t.Units.Len() {
			return PartyMember{}, false
		}
		name := t.Units.EntryName(row)
		d, err := data.NewUnitDef(name, t.Units.EntryParams(row))
		if err != nil {
			return PartyMember{}, false
		}
		p = PartyMember{Name: name, Class: d.TypeID, FigureFace: int(d.Face),
			Hero:    data.Hero{Body: d.Body, Reaction: d.Reaction, Mind: d.Mind, Spirit: d.Spirit},
			Profile: data.Profile{HealthColumn: true, ManaColumn: d.ManaMax > 0}, HiredRotationSpeed: d.RotationSpeed}
	} else {
		s.CharacterSpec.Row = row
		p, ok = characterTemplate(t, s.CharacterSpec, party, profile.HumanName != "")
		if !ok {
			return PartyMember{}, false
		}
		if profile.RowFigure || profile.RowRotation {
			d, err := data.NewHumanDef(t.Humans.EntryName(row), t.Humans.EntryParams(row))
			if err != nil {
				return PartyMember{}, false
			}
			if profile.RowFigure {
				dir, face := data.FigureFor(d.TypeID, d.Face, d.Gender)
				p.FigureDir, p.FigureFace, p.Mage = string(dir), face, dir.Mage()
			}
			if profile.RowRotation {
				p.HiredRotationSpeed = d.RotationSpeed
			}
		}
	}
	p.Temporary, p.PlayerCharacter, p.MercenaryType = profile.Temporary, profile.PlayerCharacter, s.MercenaryType
	if profile.RowDefinition {
		p.DefinitionRow = uint16(row)
		// A hire row above 255 binds the first Humans row of its TypeID
		// (DIV-4249).
		if s.Hire && !unit && row > 255 {
			first := data.FindHumanByType(t.Humans, p.Class)
			if first <= 0 || first > 255 {
				return PartyMember{}, false
			}
			p.DefinitionRow = uint16(first)
		}
	}
	if s.MercenaryType != 0 {
		p.ID = ""
		if !unit && p.Carried == nil {
			p.Carried = make([]uint16, 0)
		}
	}
	if s.Hire {
		if s.Identity < 1 || s.Identity > 20 {
			return PartyMember{}, false
		}
		p.ID = fmt.Sprintf("hire:%d", s.Identity)
		p.CampaignHire, p.CampaignUnit = uint8(s.Identity), unit
	}
	return p, true
}

func campaignNamedRow(c data.Collection, name string) int {
	if c != nil {
		for i := 1; i < c.Len(); i++ {
			if c.EntryName(i) == name {
				return i
			}
		}
	}
	return 0
}

func campaignKeyRow(c data.Collection, key int32, unit bool) int {
	if c != nil {
		for i := c.Len() - 1; i > 0; i-- {
			if c.EntryName(i) == "" {
				continue
			}
			id, ok := data.HumanServerID(c.EntryParams(i))
			if unit {
				id, ok = data.UnitServerID(c.EntryParams(i))
			}
			if ok && id == key {
				return i
			}
		}
	}
	return 0
}
