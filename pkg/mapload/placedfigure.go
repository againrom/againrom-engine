package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// placementSexFlag is bit 2 of a placement's flag word: the spawner copies it
// into bit 7 of the face byte it writes for a person (ALM-FLAGPATH-109).
const placementSexFlag = 4

// PlacedFigure resolves a person's figure from his constructor mode and face
// byte (ALM-CLS-054, DLG-FACEBYTE-043). A creature or unresolved key has no
// figure. A Hero-mode NPC uses the row's separate face and gender columns.
func PlacedFigure(u alm.Unit, t *Table) (data.FigureDir, int, bool) {
	r := Resolve(u, t)
	if r.Arm == ArmUnits || !r.Found() {
		return "", 0, false
	}
	c := t.humans()
	d, err := data.NewHumanDef(c.EntryName(r.Index), c.EntryParams(r.Index))
	if err != nil {
		return "", 0, false
	}
	dir, face := placedFigure(u, r, d, t)
	return dir, face, true
}

// placedFigure is PlacedFigure for a placement whose resolution and row the
// caller already holds.
func placedFigure(u alm.Unit, r Resolution, d data.HumanDef, t *Table) (data.FigureDir, int) {
	heroMode := r.Arm == ArmNPC && t.npc().Hero(int32(u.ClassSubID))
	if !heroMode && data.ComposesFigure(d.TypeID) {
		b, stored := t.unitKeys().placedFaceByte(u, r.Arm)
		if !stored {
			gender := int32(1)
			cells := t.humans().EntryParams(r.Index)
			if len(cells) > 18 && cells[18] != -1 {
				gender = cells[18]
			}
			b = uint8(d.Face) | uint8(gender<<7)
			if b&0x7f == 0 {
				b |= 1
			}
		}
		return data.FigureForFaceByte(d.TypeID, b)
	}
	return data.FigureFor(d.TypeID, d.Face, d.Gender)
}

func (classKeys) placedFaceByte(u alm.Unit, arm Arm) (uint8, bool) { return spawnerFaceByte(u, arm) }

// A second-game person keeps the face byte of his Humans row: its face column
// with the gender in bit 7 (R2-ASSET-172, R2-ENGINE-681).
func (serverKeys) placedFaceByte(alm.Unit, Arm) (uint8, bool) { return 0, false }

func (classKeys) clientHero(alm.Unit, Resolution) bool { return false }

// The second game's client map constructor gives a person whose NPC word is
// 21..23 the hero class; the server keeps the row's type id (R2-ENGINE-680).
func (serverKeys) clientHero(u alm.Unit, r Resolution) bool {
	if r.Arm != ArmServerID || !r.Found() {
		return false
	}
	word := PersonNPCWord(int32(u.ServerID))
	return word >= clientHeroWordLo && word <= clientHeroWordHi
}

const clientHeroWordLo, clientHeroWordHi = 21, 23

// PersonNPCWord is the NPC word the second game's Humans template load stores
// for a row's server id: the id, or (id/10)%1000 for an id above 10000
// (R2-ENGINE-525).
func PersonNPCWord(serverID int32) uint16 {
	word := uint16(serverID)
	if word > 10000 {
		word = (word / 10) % 1000
	}
	return word
}

// PlacedClientHero is the hero type id the client draws a placed person with
// while his World type id stays the row's, and false for every other placement:
// the row's gender plus 2 for a mana column, above 0x21 (R2-ENGINE-680).
func PlacedClientHero(u alm.Unit, t *Table) (int32, bool) {
	r := Resolve(u, t)
	if r.Arm == ArmUnits || !r.Found() || !t.unitKeys().clientHero(u, r) {
		return 0, false
	}
	d, err := data.NewHumanDef(t.humans().EntryName(r.Index), t.humans().EntryParams(r.Index))
	if err != nil {
		return 0, false
	}
	dir, _ := data.FigureFor(d.TypeID, d.Face, d.Gender)
	return sim.HeroTypeID(d.Profile().ManaColumn, dir.Female()), true
}

// spawnerFaceByte is the face byte the spawner writes for a person placement,
// and false where it writes none: the secondary key's low byte with bit 7 taken
// from bit 2 of the flag word. The type-key arm always writes it, the
// definition-id arm when the secondary key's word is not zero, and the npc arm
// never does (ALM-FLAGPATH-109). A byte whose face number is zero is left to the
// row: the Human basis refuses face 0 and a save is never refused (DIV-1574).
func spawnerFaceByte(u alm.Unit, arm Arm) (uint8, bool) {
	switch arm {
	case ArmHumansByType:
	case ArmServerID:
		if u.ClassSubID == 0 {
			return 0, false
		}
	default:
		return 0, false
	}
	b := uint8(u.ClassSubID) &^ 0x80
	if u.Flags&placementSexFlag != 0 {
		b |= 0x80
	}
	return b, b&0x7f != 0
}
