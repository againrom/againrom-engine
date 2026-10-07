package main

// The whole character sheet.
//
// Everything the panel states for a party member, this tool now states for
// every placement of a map: the four statistics, the two pools, a blow's
// range and what it meets, the five weapon-keyed skill positions, the
// elemental family, an experience VALUE, the sight and the rate. It is the
// second half of what -databin already reported — the eight combat numbers
// and the rate were there since 0049 and 0091; the statistics, the skill
// positions, the elemental family and the experience columns were not, and a
// question about a unit's own character could not be answered without a new
// program each time.
//
// NOTHING HERE NAMES pkg/sim (Rules; 0137 T4 brief). cmd/classdump's row in
// the import DAG stops at pkg/mapload, so the numbers a built world carries —
// the pools, the damage pair, absorption, the two combat numbers, sight and
// speed — arrive here as sheetEntity, this package's OWN plain-field type,
// copied by both call sites from world.Entities()' return through Go's type
// inference: `for i, e := range ents { sheetEntity{hp: e.HP, ...} }` names no
// type of pkg/sim any more than domainName's own uint8(ents[i].Domain)
// already does in databin.go.

import (
	"fmt"
	"io"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// sheetEntity is one placement's own numbers, off the built world:
// everything a fight already reads, none of it recomputed here. It stands
// beside mapload.Sheet rather than inside it, because Sheet is a pure
// function of the map and the table and these fields are not — they are
// the world's, at whatever difficulty it was built with.
//
// EVERY FIELD IS A BUILTIN. Naming sim.Entity here would put pkg/sim on
// cmd/classdump's row of the import DAG, which internal/archtest's allow-map
// does not carry and this task may not add to it (Rules).
type sheetEntity struct {
	hp, maxHP          int32
	mana, maxMana      int32
	dmgBase, dmgSpread int32
	absorption         int32
	toHit, defence     int32
	scanRange          uint8
	speed              int32
	xpValue            int32
}

var sheetSkillSlots = [5]int32{data.SkillBlade, data.SkillAxe, data.SkillBludgen, data.SkillPike, data.SkillShoot}

func reportSheets(w io.Writer, m *alm.Map, tbl *mapload.Table, ents []sheetEntity) error {
	sheets := mapload.PlacedSheets(m, tbl)
	// byIndex re-keys sheets by the placement's own slice index. Writing
	// sim.EntityID(i) to build that key directly would name pkg/sim in this
	// file; int(id) instead converts the RANGE VARIABLE's own inferred type,
	// which needs no import, on domainName's own precedent above.
	byIndex := make(map[int]mapload.Sheet, len(sheets))
	for id, s := range sheets {
		byIndex[int(id)] = s
	}

	if _, err := fmt.Fprintf(w,
		"\ncharacter sheets: %d placement(s) (FR-9)\n"+
			"  '-' marks UNSTATED and is never a measurement (P-4): Weight has no column\n"+
			"  on any definition row, and a PERSON's XP is the units collection's own\n"+
			"  per-slot value, which the humans collection does not carry (FR-9).\n",
		len(m.Units)); err != nil {
		return err
	}

	for i, u := range m.Units {
		r := mapload.Resolve(u, tbl)
		row := templateRow{resolved: r.Found(), arm: r.Arm.String(), entryIndex: r.Index}
		if r.Found() && tbl != nil {
			if r.Arm == mapload.ArmUnits {
				row.entry = tbl.Units.EntryName(r.Index)
			} else {
				row.entry = tbl.Humans.EntryName(r.Index)
			}
		}
		s, hasSheet := byIndex[i]
		if err := printSheetBlock(w, i, row, s, hasSheet, ents[i]); err != nil {
			return err
		}
	}
	return nil
}

func printSheetBlock(w io.Writer, i int, row templateRow, s mapload.Sheet, hasSheet bool, e sheetEntity) error {
	if _, err := fmt.Fprintf(w, "  sheet %d  band %-8s entry %s\n",
		i, bandOf(row), describeEntry(row)); err != nil {
		return err
	}

	body, agility, mind, spirit := "-", "-", "-", "-"
	if hasSheet {
		body = fmt.Sprintf("%d", s.Body)
		agility = fmt.Sprintf("%d", s.Reaction)
		mind = fmt.Sprintf("%d", s.Mind)
		spirit = fmt.Sprintf("%d", s.Spirit)
	}
	if _, err := fmt.Fprintf(w, "    Body %s  Agility %s  Mind %s  Spirit %s\n",
		body, agility, mind, spirit); err != nil {
		return err
	}

	// Health and mana, the damage range and the three combat cells: read off
	// the entity ALWAYS, resolved or not, exactly as reportCombat's own eight
	// already are for an unresolved row — a placement that resolved to
	// nothing still stood in a built world, and that world still carries the
	// constructor's own numbers for it.
	if _, err := fmt.Fprintf(w, "    Health %d/%d  Mana %d/%d\n",
		e.hp, e.maxHP, e.mana, e.maxMana); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "    Dmg %d-%d  Absorb %d  Attack %d  Defense %d\n",
		e.dmgBase, e.dmgBase+e.dmgSpread, e.absorption, e.toHit, e.defence); err != nil {
		return err
	}

	skill := make([]string, 0, 2*len(sheetSkillSlots))
	for _, slot := range sheetSkillSlots {
		v := "-"
		if hasSheet {
			v = fmt.Sprintf("%d", s.Skill[slot])
		}
		skill = append(skill, data.SkillName(slot), v)
	}
	if _, err := fmt.Fprintf(w, "    Skill %s\n", strings.Join(skill, " ")); err != nil {
		return err
	}

	elemental := make([]string, 0, 2*len(sheetSkillSlots))
	for k, slot := range sheetSkillSlots {
		v := "-"
		if hasSheet {
			v = fmt.Sprintf("%d", s.Elemental[k])
		}
		elemental = append(elemental, data.MageSkillName(slot), v)
	}
	if _, err := fmt.Fprintf(w, "    Elemental %s\n", strings.Join(elemental, " ")); err != nil {
		return err
	}

	xp := fmt.Sprintf("%d", e.xpValue)
	if hasSheet && s.Band == mapload.BandPerson {
		xp = "-"
	}
	if _, err := fmt.Fprintf(w, "    Weight -  XP %s  Sight %d  Speed %d\n",
		xp, e.scanRange, e.speed); err != nil {
		return err
	}
	return nil
}
