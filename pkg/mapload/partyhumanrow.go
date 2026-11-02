package mapload

import (
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// PartyHumanRow is the installed Humans row a party member's actor is bound
// to: the chargen row for the starting hero, the campaign row for a companion,
// the hire row for a Human mercenary, else the first row of his TypeID. Zero
// is no row. The SAV writer stores this row and the mission start reads the
// member's dying time from it (DAT-SCHEMA-007).
func PartyHumanRow(member, hero PartyMember, t *Table) int {
	if t == nil || t.Humans == nil {
		return 0
	}
	if member.StartingHero {
		fig := data.FigureDir(member.FigureDir)
		_, row, ok := data.ChargenBase(t.Humans, fig.Mage(), fig.Female())
		if !ok || row <= 0 || row > maxTokenRow {
			// A hero carrying a party identity is built from a Start_
			// template (R2-ENGINE-105), not a chargen row; he keeps the row
			// his loaded record names (R2-ENGINE-319: template load writes
			// it to +0xc).
			_, identified := PartyIdentity(member, t)
			if row := int(member.DefinitionRow); identified && row > 0 && row < t.Humans.Len() {
				return row
			}
			return 0
		}
		return row
	}
	if member.CampaignHireID() != 0 {
		return int(member.DefinitionRow)
	}
	if row, ok := identityRow(member, t); ok {
		return row
	}
	if member.CompanionNPC != 0 {
		k := StartingHeroKind(HeroHold{Party: []PartyMember{hero}})
		if serverID, ok := t.NPC.CampaignServerID(int32(member.CompanionNPC), 0, k.Mage(), k.Female()); ok {
			if row := data.FindHumanByServerID(t.Humans, serverID); row > 0 && row <= maxTokenRow {
				return row
			}
		}
	}
	if member.MercenaryType != 0 && member.MercenaryType != 1 && member.MercenaryType != 2 {
		if row := int(member.DefinitionRow); row > 0 && row < t.Humans.Len() {
			return row
		}
		if row := data.FindHumanByName(t.Humans, member.Name); row > 0 && row <= maxTokenRow {
			return row
		}
	}
	row := data.FindHumanByType(t.Humans, member.Class)
	if row <= 0 || row > maxTokenRow {
		return 0
	}
	return row
}

// BindHeroBand is the hero-band binder. It calls bind(k, i) for each actor
// the identity source of the table's edition names for script value
// 10001+k, in that source's order; i indexes the party, then placed. A
// caller keeps the first actor bound to each k.
func BindHeroBand(t *Table, party []PartyMember, placed []data.HeroTraits, bind func(k, i int)) {
	source, e := heroBand(t)
	source.bind(t, e, party, placed, bind)
}

// PartyIdentity is the party identity member carries under the identity
// source of the table's edition: the hero's own, or a companion's identity
// row (R2-ENGINE-105, R2-ENGINE-106). The role source gives none.
func PartyIdentity(member PartyMember, t *Table) (int, bool) {
	source, e := heroBand(t)
	return source.identity(member, e)
}

// CompanionIdentity is the identity row a companion carrying identity id is
// built from under the identity source of the table's edition. The role
// source names none, and the hero's own identity is no companion's.
func CompanionIdentity(id int, t *Table) (base.PartyIdentity, bool) {
	source, e := heroBand(t)
	return source.companion(id, e)
}

// heroBandSource is one identity source, the row an edition's HeroBand
// names.
type heroBandSource interface {
	identity(member PartyMember, e *base.Edition) (int, bool)
	companion(id int, e *base.Edition) (base.PartyIdentity, bool)
	bind(t *Table, e *base.Edition, party []PartyMember, placed []data.HeroTraits, bind func(k, i int))
}

var heroBandSources = [...]heroBandSource{
	base.HeroBandRole:     roleBand{},
	base.HeroBandIdentity: identityBand{},
}

// TableEdition is the edition a definition table was read under; a nil
// table, or one read without an edition, is the first game's. A caller reads
// the edition and never writes through it.
func TableEdition(t *Table) *base.Edition {
	if t == nil || t.Edition == nil {
		first := base.Game("").Edition()
		return &first
	}
	return t.Edition
}

// heroBand is the identity source of the table's edition, with that
// edition.
func heroBand(t *Table) (heroBandSource, *base.Edition) {
	e := TableEdition(t)
	return heroBandSources[e.HeroBand], e
}

// identityBand binds 10001+k to the first party member carrying identity
// HeroIdentity+k (R2-ENGINE-107).
type identityBand struct{}

func (identityBand) identity(member PartyMember, e *base.Edition) (int, bool) {
	if member.StartingHero {
		return e.HeroIdentity, true
	}
	if id := member.CampaignHireID(); id != 0 {
		return id, true
	}
	if _, ok := e.PartyIdentity(member.CompanionNPC); ok && !member.Hired() {
		return member.CompanionNPC, true
	}
	return 0, false
}

func (identityBand) companion(id int, e *base.Edition) (base.PartyIdentity, bool) {
	p, ok := e.PartyIdentity(id)
	return p, ok && id != e.HeroIdentity
}

func (b identityBand) bind(_ *Table, e *base.Edition, party []PartyMember, _ []data.HeroTraits, bind func(k, i int)) {
	hero := e.HeroIdentity
	for i, p := range party {
		if id, ok := b.identity(p, e); ok && id >= hero && id-hero <= 999 {
			bind(id-hero, i)
		}
	}
}

// roleBand binds 10001+k to the first actor that passes hero ordinal k's
// role test against the starting hero (TRIG-HEROORD-075, TRIG-HEROTPL-076).
// No member carries a party identity under it.
type roleBand struct{}

func (roleBand) identity(PartyMember, *base.Edition) (int, bool) { return 0, false }

func (roleBand) companion(int, *base.Edition) (base.PartyIdentity, bool) {
	return base.PartyIdentity{}, false
}

func (roleBand) bind(t *Table, _ *base.Edition, party []PartyMember, placed []data.HeroTraits, bind func(k, i int)) {
	if len(party) == 0 {
		return
	}
	primary := 0
	for i, p := range party {
		if p.StartingHero {
			primary = i
			break
		}
	}
	var npc *data.NPCDefs
	if t != nil {
		npc = t.NPC
	}
	roster := make([]data.HeroTraits, len(party), len(party)+len(placed))
	for i, p := range party {
		roster[i] = heroTraits(p)
	}
	roster = append(roster, placed...)
	for k := 0; k <= maxHeroOrdinal; k++ {
		if i, ok := npc.ResolveHero(k, roster[primary], roster); ok {
			bind(k, i)
		}
	}
}

// maxHeroOrdinal: ordinal k reads section 21+k of a 256-section table (TRIG-HEROFAIL-078).
const maxHeroOrdinal = 234

// heroTraits is what a hero ordinal reads of a member; a hired unit is never a woman.
func heroTraits(p PartyMember) data.HeroTraits {
	return data.HeroTraits{
		Female: !p.Hired() && data.FigureDir(p.FigureDir).Female(),
		Mage:   p.Mage,
		Face:   uint8(p.FigureFace) & 0x3f,
	}
}

// identityRow is the Humans row of the identity row a companion was built
// from.
func identityRow(member PartyMember, t *Table) (int, bool) {
	id, ok := PartyIdentity(member, t)
	if !ok || member.StartingHero {
		return 0, false
	}
	p, _ := t.Edition.PartyIdentity(id)
	row := data.FindHumanByServerID(t.Humans, p.ServerID)
	return row, row > 0 && row <= maxTokenRow
}

// partyDyingTime is the dying time SourceActorSeed reads for the binding the
// SAV writer gives this member, so a fresh member and the same member loaded
// from a SAV take one value. A member with no bound row keeps none.
func partyDyingTime(member, hero PartyMember, typeID int32, t *Table) (int32, error) {
	row := PartyHumanRow(member, hero, t)
	binding := sim.SourceBinding{Class: sim.GeneratedHumanBinding, TokenRow: uint16(row), TypeID: uint16(typeID)}
	if row <= 0 && binding.DefinitionRow() == 0 || t == nil || t.Humans == nil || int(binding.DefinitionRow()) >= t.Humans.Len() {
		return 0, nil
	}
	seed, err := SourceActorSeed(binding, t)
	if err != nil {
		return 0, err
	}
	return seed.DyingTime, nil
}
