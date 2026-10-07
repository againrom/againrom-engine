package data

import (
	"fmt"
	"strconv"
)

// ItemCode is the sixteen-bit word an item carries as its appearance (spec
// Terms "Item code"). Four fields: A, bits 15..12; B, 11..8; C, 7..5; D,
// 4..0. B is the equipment slot the item occupies and also its class; D is
// the position of its defining row inside that class's collection — what
// either FACT MEANS beyond the bits themselves is a later story's question,
// not this file's.
type ItemCode uint16

// ItemClassCarried is the one value of B the seven-digit name spells
// differently (spec Terms "Seven-digit name"; HERO-APPEAR-049): the class
// carried and never worn, whose C and D do not exist as separate fields the
// original's own namer can address — only the whole byte they sit in does.
const ItemClassCarried = 14

// QuestDocumentCode is the carried-item code whose use opens the campaign
// documents panel (ITEM-DOC-053). It is named here because the code is a fact
// about the item identity, while who presents the campaign document stack is
// a front-end rule.
const QuestDocumentCode ItemCode = 0x0e1c

// documentRowMask and documentRow are the ORIGINAL'S OWN GATE, not a
// comparison against QuestDocumentCode (ITEM-DOC-053): the routine that
// raises the documents panel masks the low FIVE bits of the code and compares
// them against 0x1c, after testing that the class field is 14. Class 14 spends
// the whole low byte on the row, so every code with class 14 and a row
// congruent to 28 modulo 32 passes — rows 28, 60, 92, 124, 156, 188, 220 and
// 252, for any material nibble.
const (
	documentRowMask = 0x1f
	documentRow     = 0x1c
)

// RaisesDocuments reports whether an item of code c raises the campaign
// documents panel when it is used (ITEM-DOC-053).
//
// IT REPRODUCES THE MASK AND NOT THE ONE SHIPPED ROW. A test written as
// `c == QuestDocumentCode` is a narrower gate than the original's, and the
// difference is invisible until content authors another row: only
// `MagicItems[28]` ships, so both readings answer the same for every code a
// stock install can produce, and they part company for `MagicItems[60]` and
// the six rows above it, and for every material other than iron.
//
// The routine's third test — bits 0x6 of the object's byte at +0x8 clear — is a
// property of the OBJECT rather than of the code, so it is not readable from
// a bare code and is not applied here. The two bit tests this build does
// apply are both fields of the code itself. DIV-323 is the row for the
// dropped test.
func RaisesDocuments(c ItemCode) bool {
	return c.B() == ItemClassCarried && int(c)&documentRowMask == documentRow
}

// A, B, C and D read ItemCode's four fields, TOTAL over every one of the
// 65536 codes (AC-1): none is refused or repaired, and no sentinel is
// invented for a class that names no item. The four widths are fixed by the
// word itself and never overlap, so A()<<12 | B()<<8 | C()<<5 | D() always
// recomposes the code they were read from.
func (c ItemCode) A() int { return int(c >> 12) }
func (c ItemCode) B() int { return int((c >> 8) & 0xf) }
func (c ItemCode) C() int { return int((c >> 5) & 0x7) }
func (c ItemCode) D() int { return int(c & 0x1f) }

// ComposeItemCode packs a material, a class, a shape and a row into the
// sixteen-bit word an item carries: material in bits 15..12, class in 11..8,
// shape in 7..5 and row in 4..0 — A, B, C and D above, written rather than read.
//
// IT IS THE ONE WRITER OF THE ENCODING. The expression was spelled at three
// call sites before this story added a fourth, and four copies of a bit
// layout is four chances for one of them to drift. Every producer of a code
// in this package now goes through here, which is the write-side of the rule
// the four field readers already hold on the read side.
//
// EACH FIELD IS MASKED TO ITS OWN WIDTH. A caller handing a row of 40 or a
// material of 20 gets that value's low bits rather than a value that has
// overflowed into the field above it, which is what the shifts alone would have
// produced. No shipped table reaches either bound; the mask is what keeps a
// corrupt or hand-built input from composing a code that names some other item.
func ComposeItemCode(material, class, shape, row int) ItemCode {
	return ItemCode(uint16(material&0xf)<<12 | uint16(class&0xf)<<8 |
		uint16(shape&0x7)<<5 | uint16(row&0x1f))
}

// Name is a code's seven-digit name, by the original's own two rules (AC-2,
// HERO-APPEAR-049). When B is ItemClassCarried the name is A and B as two
// digits each and the code's WHOLE LOW BYTE — C and D together, the one
// byte they sit in, rather than the two fields separately — as three
// digits; otherwise A and B as two digits each, C as one and D as two. Both
// forms give exactly seven digits for every code in the domain: A and B
// never exceed 15, C never exceeds 7, D never exceeds 31, and the low byte
// never exceeds 255 — each fits the width its own rule spells, so no code
// fails to name.
//
// IT IS NOT CALLED String. A Stringer is invoked by every %v and %s without
// the call site spelling anything, which would let the seven-digit name
// stand in for the raw code by accident wherever one is formatted — and a
// reader of such a call site would have no way to tell which of the two they
// are looking at. Name must be spelled to be reached, so a caller and a
// reader of its code both see which form of the value they are holding.
func (c ItemCode) Name() string {
	if c.B() == ItemClassCarried {
		return fmt.Sprintf("%02d%02d%03d", c.A(), c.B(), int(c&0xff))
	}
	return fmt.Sprintf("%02d%02d%1d%02d", c.A(), c.B(), c.C(), c.D())
}

// FigureDir is the archive directory a character's figure sheets sit under
// — one of four, by whether the character is a mage and whether female
// (HERO-APPEAR-051).
type FigureDir string

// The four figure directories, exactly as the archive spells them
// (HERO-APPEAR-051).
const (
	FigureDirManFighter   FigureDir = "mfighter"
	FigureDirManMage      FigureDir = "mmage"
	FigureDirWomanFighter FigureDir = "ffighter"
	FigureDirWomanMage    FigureDir = "fmage"
)

// FigureDirFor is the figure directory a mage bit and a sex bit choose. It
// is TOTAL: every one of the four combinations names one of the four shipped
// directories, and none is refused.
func FigureDirFor(mage, female bool) FigureDir {
	switch {
	case mage && female:
		return FigureDirWomanMage
	case mage:
		return FigureDirManMage
	case female:
		return FigureDirWomanFighter
	default:
		return FigureDirManFighter
	}
}

// Mage reports whether d is one of the two decoded mage figure directories.
// Unknown directories answer false rather than being guessed from their text.
func (d FigureDir) Mage() bool {
	return d == FigureDirManMage || d == FigureDirWomanMage
}

// Female reports whether d is one of the two decoded woman figure
// directories, on Mage's own terms: an unknown directory answers false
// rather than being guessed from its text.
//
// It exists because the dialogue's speaker predicate reads BOTH axes off a
// directory (`DLG-SPEAKER-023`'s `Female` and `MySex` terms), and Mage above
// was the only one of the two ever asked for. FigureDirFor is the inverse of
// the pair and the four cases here are its own four.
func (d FigureDir) Female() bool {
	return d == FigureDirWomanFighter || d == FigureDirWomanMage
}

// The two trees a name's addresses are built under, spelled with a trailing
// separator the way keys.go's own spritePrefix fields are ("units/",
// "objects/", "structures/") — a directory name and nothing about case or
// cleaning that sprite.go's header does not already say.
const (
	itemInventoryTree = "inventory/"
	itemEquipmentTree = "equipment/"

	// itemFigureGroup is the one of a figure directory's two groups that holds
	// a sheet for every slot — the original's own name for it. The other
	// group, itemFigureSecondaryGroup, carries a sheet for only four of the ten
	// slots that carry content at all.
	itemFigureGroup = "primary"

	// itemFigureSecondaryGroup is the figure directory's OTHER group
	// (0151-layers-and-names T2; HERO-APPEAR-050): the opposite side of a
	// paired body part — a second glove, a second shoulder piece — shipped
	// for exactly four of the twelve equipment slots (ItemFigureSecondarySlots
	// below). 0110-inventory's own plan already measured it, 784 primary
	// sheets against 144 secondary across the four figure directories, and
	// deliberately left it unaddressed because no slot but the first could be
	// occupied yet ("Removed", docs/0110-inventory/provenance.md). A later
	// story widened equipping to all twelve slots without widening the
	// compositor, which is the owner's reported defect 2: a glove or a
	// shoulder piece drawn on one side alone.
	itemFigureSecondaryGroup = "secondary"

	// itemIconExt is the icon's own extension: spr16's domain, never spr256's
	// (ITEM-APPEAR-024). It has no place in sprite.go's spritePath or
	// overlayPath — both ".256" formatters for a different sprite kind — so
	// it is spelled once here instead.
	itemIconExt = ".16a"
)

// ItemIconPath is the icon a code addresses: its name under the inventory
// tree, with the sixteen-bit sprite extension (ITEM-APPEAR-024).
func ItemIconPath(c ItemCode) string {
	return itemInventoryTree + c.Name() + itemIconExt
}

// ItemFigureLayerPath is the figure layer a code addresses: its name with
// the 256-colour sprite extension, under the equipment tree, inside dir,
// inside itemFigureGroup — the group that holds a sheet for every slot
// (HERO-APPEAR-051).
//
// It goes through spritePath, sprite.go's own ".256" formatter, rather than
// appending the extension itself: a figure layer and a hero body sheet are
// the same sprite kind at the same extension, and a second formatter here
// could drift from that one's spelling of it — the same reason
// data.HeroSheetPath goes through it rather than appending ".256" of its
// own.
func ItemFigureLayerPath(dir FigureDir, c ItemCode) string {
	return spritePath(itemEquipmentTree + string(dir) + "/" + itemFigureGroup + "/" + c.Name())
}

var ItemFigureSecondarySlots = map[int]bool{4: true, 8: true, 9: true, 10: true}

// HasItemFigureSecondaryLayer reports that equipment slot n is one of the
// four ItemFigureSecondarySlots may carry a second sheet for.
func HasItemFigureSecondaryLayer(n int) bool { return ItemFigureSecondarySlots[n] }

// ItemFigureSecondaryLayerPath is a code's SECOND figure layer — the
// opposite side of a paired body part such as a glove or a shoulder piece
// (0151-layers-and-names T2; HERO-APPEAR-050). Same name, same directory,
// itemFigureSecondaryGroup in place of itemFigureGroup. A caller should only
// address this for a slot HasItemFigureSecondaryLayer reports true for; nothing
// here refuses a slot outside that set, because a code alone carries no
// slot number to check.
func ItemFigureSecondaryLayerPath(dir FigureDir, c ItemCode) string {
	return spritePath(itemEquipmentTree + string(dir) + "/" + itemFigureSecondaryGroup + "/" + c.Name())
}

// ItemFigureBasePath is a character's base sheet: a face number directly
// inside dir, under the equipment tree (HERO-APPEAR-051).
//
// IT TAKES NO ITEM CODE, AND THAT IS DELIBERATE — it is the one of the
// three addresses not built over Name. A face is character generation's own
// input, authored rather than resolved from anything an item carries; this
// tree has no character generation, so a face arrives as a plain number.
func ItemFigureBasePath(dir FigureDir, face int) string {
	return spritePath(itemEquipmentTree + string(dir) + "/" + strconv.Itoa(face))
}

// FigureStepKind says what a FigureStep does to the picture.
type FigureStepKind uint8

const (
	// FigureStepPaint blits the sheet in colour and tags its pixels with the
	// slot.
	FigureStepPaint FigureStepKind = iota
	// FigureStepTag tags the sheet's pixels with the slot and paints no colour:
	// a mage's slot 9 (HERO-FIGURE-059, finding (c)).
	FigureStepTag
	// FigureStepNone reads and paints nothing. It marks where a slot with no
	// step in the original's pass stands, so that the slot's icon and the
	// clothing layers anchored to it keep a place.
	FigureStepNone
)

// FigureStep is one step of a figure's draw pass: the equipment slot, whether
// the step is that slot's second sheet, and what it does.
type FigureStep struct {
	Slot      int
	Secondary bool
	Kind      FigureStepKind
}

func figurePrimary(n int) FigureStep   { return FigureStep{Slot: n} }
func figureSecondary(n int) FigureStep { return FigureStep{Slot: n, Secondary: true} }
func figureNoStep(n int) FigureStep    { return FigureStep{Slot: n, Kind: FigureStepNone} }

// mageFigureSteps is HERO-FIGURE-059's mage pass, `p7 head p11 p9 s9 p3 s3 p6
// p4 p8 p0 p5 s7`, in equipment slots by HERO-FIGURE-058's index+1 rule. Slot
// 8's primary sheet stands behind the body, so both compositors paint it
// before the body themselves and skip this step; its second sheet is the last
// layer. Slot 9 is tagged and never painted. Slots 11, 3 and 2 have no step.
var mageFigureSteps = []FigureStep{
	figurePrimary(8), figurePrimary(12), figureNoStep(11),
	figurePrimary(10), figureSecondary(10),
	figurePrimary(4), figureSecondary(4),
	figurePrimary(7), figurePrimary(5),
	{Slot: 9, Kind: FigureStepTag},
	figureNoStep(3), figureNoStep(2),
	figurePrimary(1), figurePrimary(6), figureSecondary(8),
}

// FigureDrawSteps is the pass composeUnitFigure and composeInventorySubject
// (pkg/game) draw a figure by; both range over it, which keeps them from
// disagreeing. dir chooses the original's mage pass or its non-mage pass
// (HERO-FIGURE-059). l is the shipped body list and e the equipment; both
// feed FigureHeldLast, which decides the non-mage tail (HERO-FIGURE-060).
//
// The non-mage pass in slots is 12, 11, 7, 4, 5, 9, 10, 1, 8, the second sheet
// of 4, 6, the second sheets of 9 and 10, then slot 1 when FigureHeldLast says
// 1 and slot 2 otherwise. The tail paints one of the two, so slot 2 is not
// painted when slot 1 is last. Slot 3 is dead in both passes. A slot with no
// painting step keeps a FigureStepNone entry; every slot has at least one
// step.
func FigureDrawSteps(dir FigureDir, l BodyList, e Equipment) []FigureStep {
	if dir.Mage() {
		return append([]FigureStep(nil), mageFigureSteps...)
	}
	steps := []FigureStep{
		figurePrimary(12), figurePrimary(11), figurePrimary(7), figurePrimary(4),
		figurePrimary(5), figurePrimary(9), figurePrimary(10), figurePrimary(1),
		figurePrimary(8), figureSecondary(4), figurePrimary(6),
		figureSecondary(9), figureSecondary(10), figureNoStep(3),
	}
	if FigureHeldLast(l, e) == 1 {
		return append(steps, figureNoStep(2), figurePrimary(1))
	}
	return append(steps, figurePrimary(2))
}

// FigureDrawOrder lists the equipment slots in the order FigureDrawSteps first
// reaches them. It names all twelve slots.
func FigureDrawOrder(dir FigureDir, l BodyList, e Equipment) []int {
	var order []int
	seen := map[int]bool{}
	for _, st := range FigureDrawSteps(dir, l, e) {
		if !seen[st.Slot] {
			seen[st.Slot] = true
			order = append(order, st.Slot)
		}
	}
	return order
}
