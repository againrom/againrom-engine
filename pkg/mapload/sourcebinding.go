package mapload

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// SourceActorSeed supplies only native policy for fields the bounded original
// importer does not restore. It neither equips table-authored items nor derives
// a Human sheet. The caller must overlay saved fields before admitting it.
func SourceActorSeed(s sim.SourceBinding, t *Table) (sim.Entity, error) {
	e := sim.Entity{Class: int32(uint8(s.TypeID)), TypeID: int32(s.TypeID), Humanoid: s.ActorClass() == 2}
	row := int(s.DefinitionRow())
	if s.ActorClass() == 1 {
		if t == nil || t.Units == nil || row >= t.Units.Len() {
			return e, fmt.Errorf("source Unit row %d unavailable", row)
		}
		d, err := data.NewUnitDef(t.Units.EntryName(row), t.Units.EntryParams(row))
		if err != nil && row == 0 {
			// An installed Units collection holds no definition at row 0. A
			// hired Catapult or Ballista carries that row through an original
			// resave, which restores it literally (SAV-ACTORCTOR-546); its
			// TypeID and Face are the spawn key a placement resolves. The
			// binding keeps the saved row.
			n := data.FindUnit(t.Units, int32(uint8(s.TypeID)), int32(s.Face))
			if g, gd, ok := ghostRow(t.Units); n == data.NotFound && ok && gd.TypeID == int32(uint8(s.TypeID)) {
				// A Control Spirit Ghost saved with row 0 and face 0 matches
				// no key; the cast constructs it from the row named Ghost
				// (MAGIC-SING-019).
				n = g
			}
			if n != data.NotFound {
				d, err = data.NewUnitDef(t.Units.EntryName(n), t.Units.EntryParams(n))
			}
		}
		if err != nil {
			return e, err
		}
		e.RotationSpeed, e.DyingTime = d.RotationSpeed, d.DyingTime
		e.Withdraw, e.Wimpy, e.SeeInvisible = d.Withdraw, d.Wimpy, sightOf(d.SeeInvisible)
		e.XPValue, e.GoldChance, e.TreasureMin, e.TreasureMax = d.XPValue, d.GoldChance, d.TreasureMin, d.TreasureMax
		e.AlwaysHits = d.AlwaysHits
		return e, nil
	}
	if s.ActorClass() != 2 || t == nil || t.Humans == nil || row >= t.Humans.Len() {
		return e, fmt.Errorf("source actor class %d Human row %d unavailable", s.Class, row)
	}
	d, err := sourceHumanDef(t, row)
	if err != nil {
		return e, err
	}
	e.RotationSpeed, e.DyingTime = d.RotationSpeed, d.DyingTime
	e.XPValue, e.GainsXP = data.UnitDefaults().XPValue, sim.InPersistBand(e.TypeID)
	e.SuppressCorpseLoot = SuppressesCorpseLoot(t.Humans.EntryName(row))
	return e, nil
}

// sourceHumanDef is the definition a saved Human actor's row names. A row
// that carries no parameters at all, which is row 0 of the installed Humans
// collection, holds no definition; the actor's own fields come from the save
// and the constructor's defaults stand for what a definition would add
// (SAV-ACTORCTOR-546).
func sourceHumanDef(t *Table, row int) (data.HumanDef, error) {
	if len(t.Humans.EntryParams(row)) == 0 {
		return data.HumanDefaults(), nil
	}
	return data.NewHumanDef(t.Humans.EntryName(row), t.Humans.EntryParams(row))
}

// SourceActorPerson binds presentation and a potential carry template to the
// canonical source basis. No live inventory, pools or spellbook is cached here.
func SourceActorPerson(e sim.Entity, name string, t *Table) (PartyMember, error) {
	s := e.SourceBinding
	row := int(s.DefinitionRow())
	if s.ActorClass() != 2 || t == nil || t.Humans == nil || row >= t.Humans.Len() {
		return PartyMember{}, fmt.Errorf("source person row %d unavailable", row)
	}
	d, err := sourceHumanDef(t, row)
	if err != nil {
		return PartyMember{}, err
	}
	// A DAT row has separate face and gender columns; the saved actor below
	// class 0x1a packs gender into Face's high bit (UNIT-PICT-038). Passing
	// that byte to FigureFor as a DAT column requested nonexistent face138.
	dir, face := data.FigureFor(int32(s.TypeID), int32(s.Face), d.Gender)
	if s.TypeID < 0x1a {
		dir, face = data.FigureForFaceByte(int32(s.TypeID), s.Face)
	} else if s.TypeID >= 0x20 && s.TypeID < 0x40 {
		// Hero wire classes carry both axes in TypeID instead (HERO-APPEAR-041).
		axes := s.TypeID - 0x21
		dir, face = data.FigureDirFor(axes&2 != 0, axes&1 != 0), int(s.Face)
	}
	h := SourceHumanState(e.SourceNow(), e.ActorLoad.Accumulator)
	return PartyMember{ID: fmt.Sprintf("join:%d", e.ID), Name: name,
		PlayerCharacter: sim.InPersistBand(e.TypeID), Class: e.Class, Mage: dir.Mage(),
		Profile: d.Profile(), FigureDir: string(dir), FigureFace: face, Hero: h.Hero(),
		SuppressCorpseLoot: e.SuppressCorpseLoot}, nil
}

func SourceActorDomain(raw uint8) (sim.Domain, error) {
	if raw < 1 || raw > 3 {
		return 0, fmt.Errorf("unsupported saved actor movement domain %d", raw)
	}
	return domainForCode(int32(raw)), nil
}
