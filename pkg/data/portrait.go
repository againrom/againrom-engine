package data

import "strconv"

// A unit's picture: whether it is COMPOSED out of equipment layers or LOADED as
// one flat file, and where the flat one lives (`UNIT-PICT-035`, `UNIT-PICT-036`,
// `UNIT-PICT-037`).
//
// THE TWO ARE EXCLUSIVE AND ONE TEST DECIDES. The engine sets a bit while
// building a drawable and both picture builders test the same mask: set →
// compose the doll out of a figure sheet and its worn layers, clear → format a
// name and load a bitmap. What sets it is the class's own type id, and the
// partition over the shipped roster is exact.
//
// NOTHING HERE READS AN ARCHIVE. This file formats addresses and answers one
// predicate; who opens them is the wiring tier's, exactly as ItemIconPath and
// ItemFigureBasePath already are.

const (
	// portraitTree is the archive directory a flat picture lives under, spelled
	// with its trailing separator like this package's other two trees. The
	// archive's own lookup is case-insensitive and separator-agnostic
	// (pkg/formats/res), so this spelling is a convention and not a claim about
	// how the node is written in the index.
	portraitTree = "infowindow/"

	// portraitExt is the extension, and it is NOT this engine's own sprite
	// format: `graphics\infowindow` ships plain 24-bit Windows bitmaps
	// (`SPR256-PICT-043`). That is why it is spelled here rather than routed
	// through spritePath, which is a `.256` formatter for a different kind.
	portraitExt = ".bmp"

	// PortraitW and PortraitH are the one geometry every shipped node has:
	// 86 of 86 on the EN root and 81 of 81 on the RU (`SPR256-PICT-043`), and
	// the same 160x240 as an equipment figure sheet — so the engine's two ways
	// of getting a picture for an actor land on one size.
	//
	// The engine's own drawing sites push the pair as constants at twelve
	// places (`UNIT-PICT-038`), which makes it an ENGINE limit rather than a
	// corpus one: a differently sized portrait needs every node replaced.
	PortraitW, PortraitH = 160, 240

	// PortraitNameMax is how many bytes of a picture name either registry can
	// hold, NUL included — both fields are fixed inline arrays read as 0x10
	// bytes (`UNIT-PICT-038`, `REG-PICT-083`). An engine limit; the longest
	// shipped live name is nine characters. Nothing below enforces it: a
	// caller formats whatever a registry gave it, and a name that could not
	// have come from one simply names no node.
	PortraitNameMax = 16

	// composedClassLimit is the type id at and above which an actor does NOT
	// compose a doll (`UNIT-PICT-035`). Below it the engine sets the compose
	// bit; at or above it the field is never written and stays zero, so the
	// picture builders take the flat-file arm.
	composedClassLimit = 0x1a
)

// ComposesFigure reports whether a unit class's actor is drawn by COMPOSING a
// figure — a base sheet with one layer per occupied equipment slot — rather
// than by loading a flat portrait (`UNIT-PICT-035`).
//
// IT IS THE ENGINE'S OWN TEST AND NOT THE SHIPPED PARTITION. The test is a type
// id below 0x1a; what the shipped roster makes of it is that ids 1..24 compose
// and ids 26, 27 and 64..80 load a file, the roster being 1..27 and 64..80
// (`UNIT-APPEAR-030`). Writing the test rather than the partition is what keeps
// this correct for a class the shipped registries do not carry.
//
// A CONSEQUENCE WORTH KNOWING BEFORE READING InfoPicture: the thirteen human
// classes carry an InfoPicture that can never be formatted, and twelve of those
// values name no node on either root (`REG-PICT-083`). Reading one is not
// finding a missing file — it is reading dead data.
func ComposesFigure(classID int32) bool { return classID < composedClassLimit }

// FigureHasHorse is the original doll-layer gate (HERO-DOLL-078). It applies
// to a unit's own class, not a player hero's equipment-derived world body.
func FigureHasHorse(classID int32) bool { return classID >= 0x11 && classID <= 0x15 }

// FigureIsHero reports whether an actor's type id takes the drawable's hero
// arm, the only arm that sets the bit admitting the warrior background
// (UNIT-PICT-035, HERO-FIGURE-144). A zero-mode Humans placement keeps its
// table type id and never takes it (PARTY-M20-031).
func FigureIsHero(typeID int32) bool { return typeID >= 0x20 && typeID < 0x40 }

// A PLACED PERSON's figure directory is one of four, chosen by a class axis
// (mage or not) and a sex axis. The client draws a person from the type id and
// the face byte on the wire and splits on the type id (`UNIT-PICT-035`,
// `HERO-APPEAR-041`):
//
//	[0x20, 0x40)  the hero arm: bit 0 of id-0x21 is the sex, bit 1 the class,
//	              and the face byte is the face whole.
//	below 0x1a    the second arm: bit 7 of the face byte is the sex, its low
//	              seven bits are the face, and ids 0x17 and 0x18 are the mage.
//	otherwise     no doll; the actor loads a flat portrait.
//
// WHICH ARM A PLACED PERSON TAKES follows his constructor mode. The humans
// constructor overwrites the type id with gender+0x21 (fighter) or gender+0x23
// (mage) only for a non-zero mode, and the one placement that passes it is an
// npc record carrying the Hero flag (`ALM-CLS-054`, `PARTY-M20-031`,
// `DAT-ACT-006`, `DAT-HUMANS-008`). That person takes the hero arm. Every other
// placement keeps its table type id, 1..24 on each shipped row, and takes the
// second arm.
//
// A ZERO-MODE PERSON'S FACE BYTE has two sources. The spawner stores the
// placement's secondary key in it and bit 2 of the placement's flag word in its
// bit 7: on the type-key arm always, on the definition-id arm when the secondary
// key is not zero, on the npc arm never (`ALM-FLAGPATH-109`). Any other person
// keeps the constructor's byte: face default 1, ORed with the gender's low bit
// in bit 7, with missing gender defaulting to 1 (DLG-FACEBYTE-043).
//
// THE MEASUREMENT BEHIND THE ROW'S FIGURE. No shipped Humans row has bit 7 of
// its face column set (0 of 216, both roots), so only the gender column draws a
// woman: 186 of the 462 campaign placements, each on a sheet the install carries.
// That is a real test, because the four directories are stocked unequally
// (`mfighter` 31 faces, `mmage` 13, `ffighter` 10, `fmage` 5): inverting the
// sex axis breaks 224 of the 462 and inverting the class axis breaks 321.
const (
	// mageTypeLo and mageTypeHi bound the type ids the second arm reads as a
	// mage: two ids, the two mage classes of the shipped roster. The hero arm
	// reads the class off the id the constructor wrote, which follows the row's
	// mana column (`HERO-CLASS-013`), so the range is this project's proxy for
	// it. It is corroborated twice: every one of the 57 rows carrying 0x18 has
	// mana or a spellbook and not one of the 153 rows below 0x17 has either, and
	// the fifteen `npc.reg` person records that state a `Mage` flag agree with
	// it 15 of 15.
	mageTypeLo, mageTypeHi = 0x17, 0x18

	// genderFemale is the explicit gender value FigureFor treats as female.
	genderFemale = 1
)

// FigureFor is the figure directory and the face number a placed person draws
// with, given the three columns his row states in slot order: type id, face,
// gender. Hero-mode placements use these separate columns. Zero-mode placed
// persons use their constructor or spawner byte through FigureForFaceByte.
//
// THE FACE IS THE COLUMN, WHOLE. It is not split: the shipped corpus tops out at
// 31 and never sets bit 7. The engine's own bound on a face is 127
// (`UNIT-PICT-038`) and nothing here enforces it; a face no sheet exists for is
// a read that misses, which is a property of the install and not of the input.
//
// IT IS TOTAL. Any three integers answer one of the four shipped directories and
// the face they were given; nothing here refuses, because the engine does not
// either.
func FigureFor(typeID, face, gender int32) (FigureDir, int) {
	mage := typeID >= mageTypeLo && typeID <= mageTypeHi
	return FigureDirFor(mage, gender == genderFemale), int(face)
}

// FigureForFaceByte is the second arm for an actor of a type id below 0x1a: the
// figure directory and face number it draws with, given the face byte it
// carries. Bit 7 is the sex, the low seven bits are the face, and the type ids
// 0x17 and 0x18 are the mage class (`UNIT-PICT-035`). It is total.
func FigureForFaceByte(typeID int32, faceByte uint8) (FigureDir, int) {
	mage := typeID >= mageTypeLo && typeID <= mageTypeHi
	return FigureDirFor(mage, faceByte&0x80 != 0), int(faceByte & 0x7f)
}

// PortraitPath is the flat picture's address as the INFO PANEL and a STRUCTURE
// both form it: the name, verbatim, with no tier (`UNIT-PICT-036`,
// `UNIT-PICT-037`).
//
// NO TIER REACHES EITHER LEAF, and that is the whole difference from
// PortraitTierPath below. The info panel's formatter takes three arguments and
// a structure has no palette tier axis to supply one, so a class with four
// pictures shows its FIRST here whatever tier the actor is drawn at.
//
// The name comes from `units.reg`'s InfoPicture or from `structures.reg`'s
// Picture (`REG-PICT-083` establishes those are the only two fields that name
// one). An empty name answers the empty string rather than a directory with an
// extension stuck on it: nothing names a node, so nothing is addressed.
func PortraitPath(name string) string {
	if name == "" {
		return ""
	}
	return portraitTree + name + portraitExt
}

// PortraitTierPath is the flat picture's address as the DIALOGUE panel forms
// it: the name, then the actor's own tier digit, EXCEPT at tier 1 where the
// digit is dropped (`UNIT-PICT-036`).
//
// THE DROPPED DIGIT IS THE ENGINE'S OWN TRUNCATION, not a naming convention
// this project chose to mirror. The formatter is unconditional and the tier is
// appended every time; a separate test then writes a NUL over the trailing
// character when the tier is 1. That is why the shipped set reads `<name>`,
// `<name>2`, `<name>3`, `<name>4` with no `<name>1` — the missing one is the
// truncation's signature.
//
// SO A TIER-4 MONSTER'S DIALOGUE PORTRAIT AND ITS INFO-PANEL PORTRAIT ARE
// DIFFERENT FILES, on every class that has four. That is a fact about the
// original and not a choice here; both addresses are formatted from one name
// and they differ.
//
// A tier at or below zero is formatted like any other rather than refused: the
// formatter admits any integer (`UNIT-PICT-038` — the bound on the digit is the
// four-wide palette arrays and the shipped file set, not the format), and a
// caller that resolves nothing gets a name that names no node, which is what
// every unresolvable address here already does.
func PortraitTierPath(name string, tier int32) string {
	if name == "" {
		return ""
	}
	if tier == 1 {
		return PortraitPath(name)
	}
	return portraitTree + name + strconv.FormatInt(int64(tier), 10) + portraitExt
}
