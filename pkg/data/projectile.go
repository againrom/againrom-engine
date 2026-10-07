package data

import (
	"fmt"
	"strings"

	"againrom/pkg/formats/reg"
)

// The projectile registry and the arithmetic a cast performs on a spell id.
//
// Three facts live here and nowhere else in this tree: which record a picture id
// names, which picture a spell id computes, and how long each picture's object
// lives. All three are properties of the game rather than of a drawing, so they
// are readable with no archive, no window and no world behind them.

// Projectile is one [ProjectileN] section of graphics/projectiles/projectiles.reg
// (`REG-PROJ-086`). A field name is the registry's own literal key spelling.
//
// The record array the engine builds is indexed by ID and not by section number,
// so ID is an address: every consumer of a projectile — the cast spawner, the
// draw, the unit shot and the client dispatcher — reaches a record by picture id.
// Projectiles below is keyed the same way for that reason.
//
// Every field but File is int32, the type the .reg parser stores an integer at,
// kept rather than widened so a negative value stays visibly negative. File is
// the registry's own string and is not an index into a [Files] table — this
// registry has none, unlike units.reg and objects.reg.
type Projectile struct {
	ID             int32
	File           string
	Phases         int32
	RotationPhases int32
	Width          int32
	Height         int32
	Palette        int32
	Homing         int32
	Flip           int32
	SFX            int32
	A16            int32
}

// The per-key defaults, each the value the engine's own loader PUSHes before
// reading the key and therefore the live value wherever the key is absent
// (`REG-PROJ-086`). On the shipped file eight rows default Width, Height,
// RotationPhases and SFX, seven default A16, three default Palette, nineteen
// default Homing and twenty-nine default Flip, so these are not corner cases.
//
// ID's own default of -1 is not among them: a row that states no ID names no
// address and is refused rather than loaded at -1.
const (
	projectilePhases         = -1
	projectileRotationPhases = 0x10
	projectileWidth          = 0x40
	projectileHeight         = 0x40
)

// Projectiles is the loaded projectiles.reg, keyed by picture id.
type Projectiles struct {
	byID map[int32]*Projectile
}

// ByID answers the record a picture id names. A picture with no record is a
// miss and not an error: the engine's own draw refuses a picture past its
// array by drawing nothing, and 21 of the 28 spell ids compute a picture
// with no record at either parity.
func (p *Projectiles) ByID(id int32) (*Projectile, bool) {
	if p == nil {
		return nil, false
	}
	c, ok := p.byID[id]
	return c, ok
}

// Len is how many records loaded.
func (p *Projectiles) Len() int {
	if p == nil {
		return 0
	}
	return len(p.byID)
}

// LoadProjectiles resolves a parsed projectiles.reg into its records.
//
// It reads [Global] Count and then the sections Projectile0..Projectile(Count-1)
// by CONSTRUCTED name, which is what the engine's own loop does; the node table
// is never walked, so build order is numeric order and a section the file holds
// under some other name is not loaded.
//
// A SECTION THAT IS NOT THERE IS SKIPPED, and so is one stating no ID. Count
// is the file's own claim about itself and a file whose claim overshoots its
// sections is still readable data; a row with no ID names no address and
// could only be stored at one this loader invented.
//
// A missing [Global] Count IS an error: a registry that does not say how many
// rows it has is not this registry.
//
// A nil registry loads nothing and returns no error, the shape LoadSpells
// already uses for an absent collection.
func LoadProjectiles(r *reg.Reg) (*Projectiles, error) {
	out := &Projectiles{byID: make(map[int32]*Projectile)}
	if r == nil {
		return out, nil
	}
	count, ok := r.GetInt("Global", "Count")
	if !ok {
		return nil, fmt.Errorf("projectiles.reg: [Global] Count is absent")
	}
	for i := int32(0); i < count; i++ {
		section := fmt.Sprintf("Projectile%d", i)
		id, ok := r.GetInt(section, "ID")
		if !ok {
			continue
		}
		file, _ := r.GetString(section, "File")
		out.byID[id] = &Projectile{
			ID:             id,
			File:           file,
			Phases:         projectileInt(r, section, "Phases", projectilePhases),
			RotationPhases: projectileInt(r, section, "RotationPhases", projectileRotationPhases),
			Width:          projectileInt(r, section, "Width", projectileWidth),
			Height:         projectileInt(r, section, "Height", projectileHeight),
			Palette:        projectileInt(r, section, "Palette", 0),
			Homing:         projectileInt(r, section, "Homing", 0),
			Flip:           projectileInt(r, section, "Flip", 0),
			SFX:            projectileInt(r, section, "SFX", 0),
			A16:            projectileInt(r, section, "A16", 0),
		}
	}
	return out, nil
}

// projectileInt is one key's value or its default. The default is the engine's
// and is stated once, at the field, rather than at every read.
func projectileInt(r *reg.Reg, section, key string, def int32) int32 {
	if v, ok := r.GetInt(section, key); ok {
		return v
	}
	return def
}

// ProjectilePrefix is the directory every projectile sheet sits under, inside
// the graphics container.
const ProjectilePrefix = "projectiles/"

// SpritePath is the archive entry this record's sheet sits at, or "" for a
// record naming no File.
//
// The extension comes from A16 and not from the name: the engine builds a
// .16a sprite object for a non-zero A16 and a .256 one for zero, and those
// two classes choose the blit (REG-PROJ-087). On the shipped registry the
// seven rows with A16 at 0 are the unit shots, pictures 1 to 7.
//
// File's separator is a backslash and the archive's is a forward slash, so the
// one conversion this address needs is made here. It opens nothing and checks
// nothing.
func (p *Projectile) SpritePath() string {
	if p.File == "" {
		return ""
	}
	ext := ".256"
	if p.A16 != 0 {
		ext = ".16a"
	}
	return ProjectilePrefix + strings.ReplaceAll(p.File, "\\", "/") + ext
}

// CastPicture and BurstPicture are the two picture ids a spell computes:
// 2*id + 8 for the object that flies and 2*id + 9 for the burst.
//
// The arithmetic is the engine's and it is not a lookup — no Data.bin column
// carries a picture id. It bounds the reachable picture at 2*28 + 9 = 65.
//
// A negative spell id answers a negative picture, which names no record. Nothing
// is clamped: a picture with no record already draws nothing.
func CastPicture(spell int) int { return 2*spell + 8 }

// BurstPicture is CastPicture's odd sibling.
func BurstPicture(spell int) int { return 2*spell + 9 }

// OverlaySpells are the four spell ids whose RETAINED per-cell area effect has
// art at all: `wall_of_fire` (3), `freezing_cloud` (7), `poison_cloud` (8) and
// `wall_of_earth` (19) (`MAGIC-OVERLAYART-051`).
//
// THE SET IS AN ENGINE LIMIT AND NOT A TABLE. The per-cell draw's four arms test
// mask bits 3, 7, 8 and 19 and index projectile records 15, 23, 25 and 47 as
// literal `.text` immediates. Those are `BurstPicture` of the four ids, but the
// value is baked rather than computed: the picture the message carried was
// already discarded when the client converted it back to a spell id. So editing
// `projectiles.reg` moves the ART behind an id and moves nothing about WHICH
// four ids have any (G2).
//
// Six spells reach the cloud mode. The other two paint their cells, damage
// through them and draw nothing, which is what this build did for all six.
var OverlaySpells = [4]int{3, 7, 8, 19}

// OverlayPicture is the projectile record a retained per-cell overlay draws for
// this spell id, and whether it has one at all.
func OverlayPicture(spell int) (int, bool) {
	for _, id := range OverlaySpells {
		if id == spell {
			return BurstPicture(spell), true
		}
	}
	return 0, false
}

// The picture ids whose flight length is not zero, with the arm each takes
// (`MAGIC-CASTSPAWN-033`). The engine holds them as 51 index bytes over a
// 6-armed jump table; there are seven of them and the other 44 reachable
// pictures take the zero arm.
//
// PictureCellUnits is how many of the engine's own position units one map cell
// spans. A cast's flight length divides a distance measured in them.
const PictureCellUnits = 256

// CastFlight is how many ticks the object picture puts on the map lives, at
// a caster-to-target distance of dist in PictureCellUnits.
//
// Zero is the answer for 44 of the 51 pictures and it means NO OBJECT AT ALL:
// the driver reports finished before reaching its own switch, so a zero-length
// cast object never executes a driver arm.
//
// A nonzero arm never answers 0. The two distance arms divide, and a cast at
// point-blank range would otherwise compute a length of zero and be
// indistinguishable from a picture that spawns nothing; one tick is the shortest
// life an object that exists can have. Which is authored, and it is the only
// place this function departs from the table.
//
// A negative dist answers the same as zero, so no input produces a negative
// life.
func CastFlight(picture, dist int) int {
	if dist < 0 {
		dist = 0
	}
	switch picture {
	case 10:
		return atLeastOne(dist / 200)
	case 12:
		return atLeastOne(dist / 384)
	case 20, 30:
		return 1
	case 34, 36:
		return 13
	case 60:
		return 21
	}
	return 0
}

func CastFlies(picture int) bool { return CastFlight(picture, PictureCellUnits) > 0 }

// The two pictures whose object is CARRIED from the caster's point to the
// target's, and the two that draw a path instead.
//
// Only the driver's default arm applies the per-axis step it computed, so of the
// seven pictures with a non-zero flight length only 10 and 12 actually move:
// 34 and 36 take an arm that writes the sheet frame and nothing else, and their
// 13 ticks are a countdown rather than a distance (`MAGIC-BOLTSTILL-072`).
// 20 and 30 snap to the target and 60 stands at the caster.
const (
	PicturePathFirst  = 34
	PicturePathSecond = 36
)

// CastTravels reports whether picture's object changes position over its life.
// It is a narrower question than CastFlies: a picture may have a length and
// still stand still.
func CastTravels(picture int) bool { return picture == 10 || picture == 12 }

// CastDrawsPath reports whether picture draws a generated point list instead of
// one sprite at its own position (`MAGIC-BOLTGATE-069`). Exactly two do.
func CastDrawsPath(picture int) bool {
	return picture == PicturePathFirst || picture == PicturePathSecond
}

// CastTrailSlot is which of the two trail sheets picture leaves behind it, or -1
// for a picture that leaves none (`MAGIC-TRAIL-073`). Exactly two do, and they
// are the same two that travel.
func CastTrailSlot(picture int) int {
	switch picture {
	case 10:
		return 0
	case 12:
		return 1
	}
	return -1
}

// CastTrailLength is how many past positions a trail holds. Six is a hard bound
// in the engine's own code: the driver drops entry 0 when the count has reached
// it, one at a time, before appending this tick's saved position.
const CastTrailLength = 6

func atLeastOne(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// The three burst lifetimes (`MAGIC-BURSTLIFE-034`), each an immediate in one of
// the two senders: 16 in the ring walker's own local, 18 on that walker's
// acid_stream arm, and 22 in the other sender.
//
// The two that are not the default are exactly the two whose sheet holds half
// that many phases — acid has 9 and fireexpl has 11 — so each of those two runs
// one full pass under the one-frame-per-two-ticks clock while the other six run
// 16 ticks against the 22 or 30 a full pass would need.
const (
	burstLifeDefault = 16
	burstLifeAcid    = 18
	burstLifeFire    = 22

	burstPictureAcid = 27
	burstPictureFire = 13
)

// BurstLife is how many ticks the burst object at picture stands for. Every
// picture has one: the lifetime is the sender's choice and not a property of
// the picture, so a picture with no burst record still answers a life, and
// what makes nothing drawn is the missing record rather than a zero here.
func BurstLife(picture int) int {
	switch picture {
	case burstPictureAcid:
		return burstLifeAcid
	case burstPictureFire:
		return burstLifeFire
	}
	return burstLifeDefault
}
