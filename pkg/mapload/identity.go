package mapload

import (
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/herokind"
)

// UnitKind is who a placement is, in the map editor's words: a creature (a
// Units row), a person (a Humans row) or a named character (an npc record
// and the Humans row it names).
type UnitKind uint8

const (
	UnitCreature UnitKind = iota
	UnitPerson
	UnitNamed
)

// Identity is who a placement is: its kind, the Units or Humans row its keys
// reach, and a named character's npc record.
type Identity struct {
	Kind UnitKind
	Row  int
	NPC  int32
}

// kindOf is the kind of the arm a placement takes.
func kindOf(a Arm) UnitKind {
	switch a {
	case ArmUnits:
		return UnitCreature
	case ArmNPC:
		return UnitNamed
	}
	return UnitPerson
}

// Identify is who u is: Resolve's answer as an identity. False when its keys
// reach no row; the identity then holds only the kind of the arm taken.
func Identify(u alm.Unit, t *Table) (Identity, bool) {
	r := Resolve(u, t)
	return Identity{Kind: kindOf(r.Arm), Row: r.Index, NPC: r.NPC}, r.Found()
}

// HeroRow is the Humans row a start gives a placement for one kind of
// starting hero.
type HeroRow struct {
	Hero herokind.Kind
	Row  int
}

// IdentifyAt is who u is in a start of mission: Identify's answer, or for
// a placement on an npc record that start composes from the starting hero
// (REG-SCN-098) the named character of that record, its Row NotFound, with
// the row each kind of starting hero gets. False when u's keys reach no row
// for any hero.
func IdentifyAt(u alm.Unit, t *Table, mission int) (Identity, []HeroRow, bool) {
	id, ok := Identify(u, t)
	if ok || id.Kind != UnitNamed {
		return id, nil, ok
	}
	var rows []HeroRow
	for k := herokind.Kind(0); herokind.Every.Holds(k); k++ {
		if got, ok := Identify(u, tableForHero(t, k, mission)); ok {
			rows = append(rows, HeroRow{Hero: k, Row: got.Row})
		}
	}
	return id, rows, len(rows) > 0
}

// npcPlacement is the placement of npc record npc alone.
func npcPlacement(npc int32) (alm.Unit, bool) {
	if npc < 0 || npc > 0xffff {
		return alm.Unit{}, false
	}
	return alm.Unit{Flags: npcFlagBit, ClassSubID: uint16(npc)}, true
}

// Identities is every identity the table's key rule can key a placement to
// in a start of mission, its kinds in the rule's order. The named kind is
// present only where the table has npc records.
func Identities(t *Table, mission int) []Identity {
	return t.unitKeys().identities(t, mission)
}

// Place is u keyed to id in a start of mission: the keys id's rung reads
// are written so that IdentifyAt answers id, and every other byte of u is
// kept. A composed named character takes the keys of the first hero kind's
// row. A placement that already is id comes back unchanged. False when the
// key rule cannot key a placement to id.
func Place(u alm.Unit, id Identity, t *Table, mission int) (alm.Unit, bool) {
	if was, _, ok := IdentifyAt(u, t, mission); ok && was == id {
		return u, true
	}
	on, key := t, id
	if npc, ok := npcPlacement(id.NPC); ok && id.Kind == UnitNamed && id.Row == data.NotFound {
		if _, rows, ok := IdentifyAt(npc, t, mission); ok {
			on, key.Row = tableForHero(t, rows[0].Hero, mission), rows[0].Row
		}
	}
	out := t.unitKeys().place(u, key, on)
	got, _, ok := IdentifyAt(out, t, mission)
	return out, ok && got == id
}

// rowParams is row i's parameters, nil outside the collection.
func rowParams(c data.Collection, i int) []int32 {
	if c == nil || i < 1 || i >= c.Len() {
		return nil
	}
	return c.EntryParams(i)
}

// placeable is the identities among candidates Place can key a placement to.
func placeable(t *Table, mission int, candidates []Identity) []Identity {
	var out []Identity
	for _, id := range candidates {
		if _, ok := Place(alm.Unit{}, id, t, mission); ok {
			out = append(out, id)
		}
	}
	return out
}

func rowIdentities(kind UnitKind, c data.Collection) []Identity {
	var out []Identity
	for i := 1; c != nil && i < c.Len(); i++ {
		out = append(out, Identity{Kind: kind, Row: i})
	}
	return out
}

func (classKeys) identities(t *Table, mission int) []Identity {
	out := placeable(t, mission, append(rowIdentities(UnitCreature, t.units()), rowIdentities(UnitPerson, t.humans())...))
	var named []Identity
	if t != nil {
		for _, e := range t.NPCCatalog.Records() {
			u, ok := npcPlacement(e.ID)
			if !ok {
				continue
			}
			if id, _, ok := IdentifyAt(u, t, mission); ok && id.Kind == UnitNamed {
				named = append(named, id)
			}
		}
	}
	slices.SortFunc(named, func(a, b Identity) int { return int(a.NPC - b.NPC) })
	return append(out, named...)
}

// place writes the class-key rule's keys of id's rung (MISSION-ARM-006). A
// creature takes its row's type and face as the class and secondary keys. A
// person and a named character take the row's type as the class key, which
// the band test reads; a person clears the npc flag and takes the row's
// server id as the definition id, or 0 to be found by type; a named
// character sets the flag and takes the record as the secondary key. A
// person who was none keeps no secondary key, which the person rung reads as
// its face: 0 is the row's own.
func (classKeys) place(u alm.Unit, id Identity, t *Table) alm.Unit {
	switch id.Kind {
	case UnitCreature:
		typeID, face, ok := data.UnitKeyPair(rowParams(t.units(), id.Row))
		if ok && typeID >= unitsKeyFloor && typeID <= 0xff && face >= 0 && face <= 0xff {
			u.ClassID, u.ClassSubID = int16(typeID), uint16(face)
		}
		return u
	case UnitPerson:
		p := rowParams(t.humans(), id.Row)
		if was, _ := Identify(u, t); was.Kind != UnitPerson {
			u.ClassSubID = 0
		}
		u = humanBand(u, p)
		u.Flags &^= npcFlagBit
		u.DefID = 0
		if sid, ok := data.HumanServerID(p); ok && sid != 0 && uint32(sid) != defIDUnwritten {
			byID := u
			byID.DefID = uint32(sid)
			if got, ok := Identify(byID, t); ok && got == id {
				return byID
			}
		}
		return u
	case UnitNamed:
		_, row := Identify(alm.Unit{Flags: npcFlagBit, ClassSubID: uint16(id.NPC)}, t)
		if !row || id.NPC < 0 || id.NPC > 0xffff {
			return u
		}
		u = humanBand(u, rowParams(t.humans(), id.Row))
		u.Flags |= npcFlagBit
		u.ClassSubID = uint16(id.NPC)
	}
	return u
}

// humanBand is u with the class key of Humans row p's type, when the type
// lies below the units floor.
func humanBand(u alm.Unit, p []int32) alm.Unit {
	if typeID, ok := data.HumanTypeID(p); ok && typeID >= 0 && typeID < unitsKeyFloor {
		u.ClassID = int16(typeID)
	}
	return u
}

func (serverKeys) identities(t *Table, mission int) []Identity {
	return placeable(t, mission, append(rowIdentities(UnitCreature, t.units()), rowIdentities(UnitPerson, t.humans())...))
}

// place writes the server-id rule's keys: the row's server id, and the
// person flag for a person.
func (serverKeys) place(u alm.Unit, id Identity, t *Table) alm.Unit {
	switch id.Kind {
	case UnitCreature:
		if sid, ok := data.UnitServerID(rowParams(t.units(), id.Row)); ok && sid > 0 {
			u.ServerID, u.Flags = uint32(sid), u.Flags&^rom2PersonFlag
		}
	case UnitPerson:
		if sid, ok := data.HumanServerID(rowParams(t.humans(), id.Row)); ok && sid > 0 {
			u.ServerID, u.Flags = uint32(sid), u.Flags|rom2PersonFlag
		}
	}
	return u
}
