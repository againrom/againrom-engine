package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
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
		b, stored := spawnerFaceByte(u, r.Arm)
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
