package data

// HeroBody is one body name — the string the original composes and then
// compares, and the directory element its art sits under.
//
// It is a DEFINED TYPE and not a bare string because the set is closed and
// every function here takes one or answers one, while a class name, a sheet
// address and a registry key are all strings living in these same packages. The
// render tier's own lookup is keyed by the plain string, that tier being unable
// to import this package at all.
type HeroBody string

// The base body names. The five shielded variants are not here: a shield is a
// SUFFIX the composition below appends, not a separate name to choose, and
// writing both forms as constants would make the suffix rule optional.
//
// BodyBowman is an ALIAS with no art: it shares BodyArcher's class key and ships
// no sheet under either directory. It is carried because the arm is in the
// image, and because a name-driven lookup with a dead alias is exactly what the
// corpus used to tell this mapping apart from a sheet-driven one.
const (
	BodyUnarmed     HeroBody = "unarmed"
	BodySwordsman   HeroBody = "swordsman"
	BodySwordsman2H HeroBody = "swordsman2h"
	BodyAxeman      HeroBody = "axeman"
	BodyAxeman2H    HeroBody = "axeman2h"
	BodyClubman     HeroBody = "clubman"
	BodyPikeman     HeroBody = "pikeman"
	BodyArcher      HeroBody = "archer"
	BodyBowman      HeroBody = "bowman"
	BodyCrossbowman HeroBody = "xbowman"
	BodyMage        HeroBody = "mage"
	BodyMageStaff   HeroBody = "mage_st"
)

// HeroShieldSuffix is what an occupied second equipment slot appends to a body
// name.
//
// The suffix is appended UNCONDITIONALLY in the original, so it can produce a
// name no arm carries — a two-handed body with a shield, say. That is not an
// error and needs no guard: such a name simply matches nothing, and
// HeroBodyClass's fallback is what the character is then drawn as.
const HeroShieldSuffix HeroBody = "_"

// HeroUnmatchedClass is the class key a body name matching no arm leaves
// standing, and it is the roster's own bare-handed human body.
//
// IT IS A REAL BEHAVIOUR AND NOT AN ERROR PATH. The original stores this literal
// into the frame selector's subscript BEFORE the compare chain runs, and the
// chain is seventeen compare-and-store arms — so a name none of them matches is
// not rejected, it is simply never overwritten. A consumer that treats an
// unmatched name as a failure is describing a state the original does not have.
const HeroUnmatchedClass int32 = 1

// heroBodyClass is the seventeen-arm table: seventeen names onto sixteen
// distinct class keys, two of the names sharing one key.
//
// It is read by lookup only, never ranged, so its order is not a fact about
// anything. The keys are class ids of the unit registry, the same domain a
// placed record's key lives in. HERO-APPEAR-042.
var heroBodyClass = map[HeroBody]int32{
	"unarmed": 1, "unarmed_": 2,
	"swordsman": 3, "swordsman_": 4, "swordsman2h": 5,
	"axeman": 7, "axeman_": 8, "axeman2h": 9,
	"clubman": 10, "clubman_": 11,
	"pikeman": 12, "pikeman_": 13,
	"archer": 14, "bowman": 14, "xbowman": 15,
	"mage": 23, "mage_st": 24,
}

// HeroBodyClass is the class key a body name is drawn as, and whether a name
// actually matched.
//
// IT IS TOTAL, and the pair is what makes it honest. The key comes back for
// every input because the original always has one; the bool is what separates a
// name that MATCHED the first arm from a name that matched nothing and fell back
// on the same value, which a single return cannot express. This is the only
// place in the tree that knows the fallback's value.
func HeroBodyClass(b HeroBody) (int32, bool) {
	if id, ok := heroBodyClass[b]; ok {
		return id, true
	}
	return HeroUnmatchedClass, false
}

// HeroBodyName composes the drawn body name: the base name the equipment slot
// yielded, the shield suffix, and the two substitutions.
//
// THE ORDER IS THE ORIGINAL'S AND IT IS LOAD-BEARING. The dying substitution
// replaces the whole composed name rather than modifying it, so a dying
// character's shield stops mattering; the mage substitution then fires only on
// the name `unarmed`, which is why a mage carrying something that yields another
// body keeps that body. Reversing the two would leave a dying mage in the
// fighter's fallen body.
//
// mage and dying are the two facts, not the two bits: which flag on which field
// carries them is the caller's business and not this law's.
func HeroBodyName(base HeroBody, shield, mage, dying bool) HeroBody {
	name := base
	if shield {
		name += HeroShieldSuffix
	}
	if dying {
		name = BodyUnarmed
	}
	if mage && name == BodyUnarmed {
		name = BodyMage
		if dying {
			name = BodyMageStaff
		}
	}
	return name
}

// The two directories hero body art ships under, and how many armour material
// blocks name one.
const (
	HeroDirHeroes      = "heroes"
	HeroDirHeroesLight = "heroes_l"

	// HeroMaterials is the count of armour material blocks, and it is also the
	// domain of the index: the original takes it from the top FOUR BITS of the
	// armour slot's word, so an out-of-range index is unrepresentable there.
	HeroMaterials = 16
)

// heroMaterialDir is the directory each armour material block names, in block
// order. Both shipped roots carry these sixteen values identically
// (HERO-APPEAR-043).
var heroMaterialDir = [HeroMaterials]string{
	HeroDirHeroes, HeroDirHeroes, HeroDirHeroes, HeroDirHeroes,
	HeroDirHeroes, HeroDirHeroes, HeroDirHeroes, HeroDirHeroes,
	HeroDirHeroesLight, HeroDirHeroesLight, HeroDirHeroesLight,
	HeroDirHeroesLight, HeroDirHeroesLight, HeroDirHeroesLight,
	HeroDirHeroes, HeroDirHeroesLight,
}

// HeroMaterialDir is one material block's directory, and false for an index
// outside the sixteen.
//
// The false arm cannot be reached from a four-bit field and exists for a
// caller that composes the index some other way — which, since 0134, is
// HeroArmourFacts, and even that reader cannot reach it: the four-bit field
// IS the sixteen, so the refusal below stays theoretical for every caller
// this tree has and exists for HeroBodyDir's own contract rather than for
// anything reachable through it.
func HeroMaterialDir(material int) (string, bool) {
	if material < 0 || material >= HeroMaterials {
		return "", false
	}
	return heroMaterialDir[material], true
}

// HeroBodyDir is the directory a body's art sits under: the mage arm, the
// fighter-with-no-armour arm, or the armour material's own.
//
// IT TAKES THE FACTS AND NOT THE SLOTS. The corpus establishes that one slot
// supplies the directory and grades the reading of that slot as ARMOUR below
// the arithmetic around it — so this law consumes "is a material known,
// and which" rather than an equipment array. A tree that later grows an
// equipment channel answers the two from wherever it finds them and nothing
// here has to move.
//
// The bool is HeroMaterialDir's, and it is false ONLY on the armour arm with an
// index outside the sixteen. Both special arms are total.
func HeroBodyDir(mage bool, material int, armoured bool) (string, bool) {
	switch {
	case mage:
		return HeroDirHeroes, true
	case !armoured:
		return HeroDirHeroesLight, true
	default:
		return HeroMaterialDir(material)
	}
}

// HeroArmourSlot is the equipment slot whose item supplies the body
// directory — 8, in the original's own 1..12 numbering that Equipment
// already uses. It is a fact about WHICH slot, not about what a slot's code
// means; HeroArmourFacts, below, is what reads it, and HeroBodyDir above
// still takes the two facts rather than the slot number itself (plan D-2's
// own rejection).
const HeroArmourSlot = 8

// HeroArmourFacts reads equipment slot HeroArmourSlot and answers the two
// facts HeroBodyDir's armour arm consumes: the material index — ItemCode's
// own field A, the top four bits, which ItemCode already exposes — and
// whether the slot is occupied, Equipment.Occupied's own reading of field D.
//
// HeroArmourSlot IS ALWAYS ONE OF EQUIPMENT'S OWN SLOTS (it is 8, and
// Equipment's domain is 1..12), so the bool Code and Occupied each also
// answer is always true here and is not a second fact worth this function
// reporting — the same reading HeroBodyFor already gives slot 1's own
// always-valid number.
func HeroArmourFacts(e Equipment) (material int, armoured bool) {
	code, _ := e.Code(HeroArmourSlot)
	armoured, _ = e.Occupied(HeroArmourSlot)
	return code.A(), armoured
}

// HeroBodyBase is the extensionless archive address of a body's art: the unit
// registry's own sprite directory, the body directory, the body name, and the
// fixed last element.
//
// A HERO'S PIXELS DO NOT COME THROUGH A CLASS RECORD'S File. The original
// composes this path and loads the sheet pair from it; only the GEOMETRY — the
// canvas, the centre, the phase scalars and the timelines — comes from the class
// record the name resolved to. So a consumer that draws a player's character
// through the record's own art address is using the placed actor's route for the
// one actor it does not serve.
//
// An empty directory or an empty name answers "", which the two formatters below
// carry through as "" rather than as a bare extension — the same answer a class
// resolving no File gives.
func HeroBodyBase(dir string, b HeroBody) string {
	if dir == "" || b == "" {
		return ""
	}
	return unitDesc.spritePrefix + dir + "/" + string(b) + "/sprites"
}

// HeroSheetPath is the archive entry a body's sheet sits at, and
// HeroOverlayPath the entry beside it — the SAME two formatters a class
// record's own address goes through, so the extension and the sibling's
// inserted "b" exist once in the tree and a body address and a class address
// cannot come to disagree about either.
//
// What the sibling holds is an overlay layer; nothing in this tree reads one
// yet, and the corpus records its consumer as unread.
func HeroSheetPath(dir string, b HeroBody) string { return spritePath(HeroBodyBase(dir, b)) }

// HeroOverlayPath is HeroSheetPath's sibling; see there.
func HeroOverlayPath(dir string, b HeroBody) string { return overlayPath(HeroBodyBase(dir, b)) }

// HeroBodyKey is the bundle key a body's directory and its name compose —
// the ONLY place in this tree that writes that convention. The loader that
// fills the bundle and the driver that reads it both go through this one
// function (D-3's own coupling), so a landing that changes how the two join
// changes it once for both sides rather than rekeying the write and leaving
// the read looking under the old key.
//
// THE JOIN EXISTS SO THAT ONE NAME UNDER TWO DIRECTORIES CANNOT BE CONFUSED
// FOR ONE ANOTHER: before 0134 the bundle held one body keyed by name alone,
// which is exactly the confusion a second directory would produce.
//
// An empty directory or an empty name answers "" — HeroBodyBase's own rule
// for the same two inputs, so a key that addresses nothing and a path that
// addresses nothing agree about which inputs count as nothing.
func HeroBodyKey(dir string, b HeroBody) string {
	if dir == "" || b == "" {
		return ""
	}
	return dir + "/" + string(b)
}

// HeroAppearance is the drawn body name, its directory, the class key and
// whether both name and directory resolved.
//
// The name is the drawn body, a mod's choice for the weapon row included;
// the class key is always the shipped entry's (DIV-2912), so a mod changes
// the picture and no entity, sound or save field. A chosen body, supplied or
// shipped, takes the shield suffix only when its form exists.
func HeroAppearance(l BodyList, e Equipment, mage, dying bool) (HeroBody, string, int32, bool) {
	base, nameOK := HeroBodyFor(l, e)
	shipped, _ := shippedBodyFor(l, e)
	shield, _ := e.Occupied(2) // slot 2 always exists; see HeroArmourFacts.
	name := HeroBodyName(base, shield, mage, dying)
	if shield && !dying {
		if _, modded := l.ModBodyOf(base); modded {
			if _, both := l.ModBodyOf(name); !both {
				name = base
			}
		} else if l.LacksShieldForm(base) {
			name = base
		}
	}

	material, armoured := HeroArmourFacts(e)
	dir, dirOK := HeroBodyDir(mage, material, armoured)

	class, _ := HeroBodyClass(HeroBodyName(shipped, shield, mage, dying)) // total; see its own doc.

	return name, dir, class, nameOK && dirOK
}
