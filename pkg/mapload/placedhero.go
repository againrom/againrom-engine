package mapload

import (
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// PlacedHero is a map placement the hero-ordinal scan can return: its record
// index in m.Units, its authored unit id and what an ordinal reads of it.
type PlacedHero struct {
	Record int
	UnitID uint16
	Traits data.HeroTraits
}

// PlacedHeroes lists, in the scan's order, the placements of m an ordinal can
// resolve to after the party (TRIG-MAPORD-105, TRIG-MAPNAME-106,
// TRIG-MAPORD-108): player 1's placements in record order, as its flat list
// holds them behind the party, then every other placement in record order, as
// the registry holds them. A placement is listed only when the spawner keeps
// it (its owner resolves, its type id is not 0) and the npc arm names it from
// a non-empty npcnames line. The cell placement is taken to succeed, the
// server's +0x11c gate to be 0, and no actor of an earlier mission to remain.
// mission selects the row a composed npc section takes for the party's primary.
func PlacedHeroes(m *alm.Map, t *Table, party []PartyMember, mission int) []PlacedHero {
	if m == nil || t == nil || t.NPC == nil || t.Humans == nil {
		return nil
	}
	tt := tableForPartyNPCs(t, party, mission)
	owners := len(m.Groups)
	if owners == 0 {
		owners = 1
	}
	var first, rest []PlacedHero
	for i, u := range m.Units {
		if u.Owner < 1 || int(u.Owner) > owners {
			continue
		}
		traits, ok := placedHeroTraits(u, tt, party)
		if !ok {
			continue
		}
		p := PlacedHero{Record: i, UnitID: u.UnitID, Traits: traits}
		if u.Owner == 1 {
			first = append(first, p)
		} else {
			rest = append(rest, p)
		}
	}
	return append(first, rest...)
}

// placedHeroTraits is the type id, class bit and face the npc arm gives a
// named placement (TRIG-MAPNAME-106). Units-arm and other Humans-arm
// placements carry no name and answer false.
func placedHeroTraits(u alm.Unit, t *Table, party []PartyMember) (data.HeroTraits, bool) {
	r := Resolve(u, t)
	if r.Arm != ArmNPC || !r.Found() {
		return data.HeroTraits{}, false
	}
	if placedHeroName(u, t, party) == "" {
		return data.HeroTraits{}, false
	}
	name := t.Humans.EntryName(r.Index)
	h, err := data.NewHumanDef(name, t.Humans.EntryParams(r.Index))
	if err != nil {
		return data.HeroTraits{}, false
	}
	mage := h.ManaMax > 0
	typeID := h.TypeID
	if t.NPC.Hero(int32(u.ClassSubID)) {
		dir, _ := data.FigureFor(h.TypeID, h.Face, h.Gender)
		typeID = sim.HeroTypeID(mage, dir.Female())
	}
	if typeID == 0 {
		return data.HeroTraits{}, false
	}
	return data.HeroTraits{Female: typeID == 0x22 || typeID == 0x24, Mage: mage, Face: placedHeroFace(name, h.Face)}, true
}

// placedHeroName is the npcnames line the spawner writes as the actor's name:
// line n for npc section n, and line selector+21 for a composed section,
// where the selector is the composed archetype (female +1, mage +2).
func placedHeroName(u alm.Unit, t *Table, party []PartyMember) string {
	id := int32(u.ClassSubID)
	line := int(id)
	if _, composed := t.composedNPCIndex(id); composed {
		primary := partyPrimary(party)
		mage, female, _ := t.NPC.ComposedArchetype(id, primary.Mage, data.FigureDir(primary.FigureDir).Female())
		line = 21
		if female {
			line++
		}
		if mage {
			line += 2
		}
	}
	if line < 1 || line > len(t.NPCNames) {
		return ""
	}
	return t.NPCNames[line-1]
}

// placedHeroFace is the Humans face column, replaced by a positive decimal
// suffix after the row name's last '.'. A column below 0 with no suffix
// leaves the base value, which no claim reads; it is taken as 0.
func placedHeroFace(name string, column int32) uint8 {
	face := column
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
		if n, err := strconv.ParseInt(name[dot+1:], 10, 32); err == nil && n > 0 && !strings.ContainsAny(name[dot+1:], "+-") {
			face = int32(n)
		}
	}
	if face < 0 {
		face = 0
	}
	return uint8(face) & 0x3f
}

func partyPrimary(party []PartyMember) PartyMember {
	if len(party) == 0 {
		return PartyMember{}
	}
	for _, p := range party {
		if p.StartingHero {
			return p
		}
	}
	return party[0]
}
