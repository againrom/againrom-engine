package ui

import (
	"fmt"
	"hash/maphash"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"sync"

	"againrom/pkg/render/text"
)

// The unit information panel: what it looks like, what it says, and the one
// value that holds the whole of the first.
//
// EVERYTHING HERE IS A FUNCTION OF ITS ARGUMENTS. Nothing in this file reads
// viewer state, opens a file, reads a clock or needs a graphics context: the
// panel is composed into a plain in-memory image, which is also the form the
// font's own blit paints into, so the pen rule exists once and this package
// holds no second copy of it. What is on screen and what a test asserts are
// then the same pixels.

// PanelField names one thing the panel can state. The set is CLOSED, and what
// closes it is what the tree's live state actually is: a running unit carries a
// name, a health pair, a cell and the eight numbers a blow reads, and a unit the
// loader placed from a party carries a character besides.
//
// A later story that makes another value real adds a constant here and a case
// in panelText — one switch, one place — rather than a second drawing path.
//
// THE NUMBER SPACE IS SHARED WITH THE DEBUG READOUT and it is now allocated end
// to end. `0..3` are this box's original four; `4..15` were reserved for it to
// grow into and **this story spends that reservation exactly** — six character
// fields then six combat fields; `16..31` are the readout's. A thirteenth panel
// field therefore takes **32**, and taking anything below it would collide with
// a box in another file whose resolver would then answer for it.
type PanelField uint8

const (
	// PanelFieldName is the subject's own name. An unnamed subject falls back
	// to its installed unit-class text; with neither source it is absent.
	PanelFieldName PanelField = 0
	// PanelFieldHealth is the health pair, as `<hp>/<max>`.
	PanelFieldHealth PanelField = 1
	// PanelFieldCell is the column and row, as `<x>, <y>`.
	PanelFieldCell PanelField = 2
	// PanelFieldCount is how many units the selection presently holds, in
	// decimal.
	//
	// It is the ONE field that states something about the SELECTION rather than
	// about the unit being described, and it is the second that can be absent:
	// below two it has no value, so its row is skipped and — by the flow rule
	// below, which counts kept rows — costs no space. That is what makes "a
	// single-unit panel is the picture it was before this story" an identity
	// rather than a resemblance.
	PanelFieldCount PanelField = 3

	// The six a unit's CHARACTER states. All six are absent together for a unit
	// the loader knows no character for, which since 0137 is a placement that
	// reached no definition entry rather than every unit this tree does not
	// place from a party.
	//
	// The four statistics are in the order the chargen panel reads them back
	// in — Body, Reaction, Mind, Spirit — which is the one part of this
	// layout that is not ours.
	PanelFieldBody     PanelField = 4
	PanelFieldReaction PanelField = 5
	PanelFieldMind     PanelField = 6
	PanelFieldSpirit   PanelField = 7
	PanelFieldSkill    PanelField = 8
	// PanelFieldWeapon is what he holds, by name. A BARE hero has no such row,
	// which is a state of the original rather than a hole in this one.
	PanelFieldWeapon PanelField = 9
	// PanelFieldWorn is the worn set's own row names, in slot order — the
	// same question PanelFieldWeapon answers about the same subject, so it is
	// declared beside it here rather than at the top of the panel's own 32+
	// run.
	//
	// ITS NUMBER IS THE LOWEST FREE ONE IN THE SHARED SPACE, 30: the
	// readout's own `16..31` reservation above only ever spent `16..29`, and
	// this takes the first of the two rungs it left standing rather than
	// opening a fresh number past 36.
	PanelFieldWorn PanelField = 30

	// The six rows the EIGHT COMBAT NUMBERS take. Every unit on the field has
	// them, and they are the simulation's own values rather than anything
	// re-derived here.
	//
	// PanelFieldDamage is the pair as the sheet composes it — `base` and
	// `base + spread`, the roll's own bounds — and never as base and spread
	// side by side, which would read as a range the roll cannot produce.
	PanelFieldDamage     PanelField = 10
	PanelFieldToHit      PanelField = 11
	PanelFieldDefence    PanelField = 12
	PanelFieldAbsorption PanelField = 13
	// PanelFieldSwing is the attack cadence, charge over relax. It is NOT
	// PanelFieldCadence, which is the readout's clock rate at 16 — two
	// different rates about two different things, and the collision is the
	// whole reason this one is named for the swing.
	PanelFieldSwing PanelField = 14
	// PanelFieldArmorPiercing is the installed ARMOR PIERCING caption. Its
	// original actor predicate is carried separately from combat semantics.
	// It is a heading, not a yes/no value: main.txt slot 191 is the complete
	// original string.
	PanelFieldArmorPiercing PanelField = 15

	// PanelFieldMoveSpeed is how fast the unit moves — the rate input a STEP
	// reads, where the six above are what a BLOW reads.
	//
	// IT IS 32 AND NOT 16..31, which is the allocation rule three constants up
	// being obeyed the first time it binds: `4..15` was this box's reservation
	// and this story already spent it, `16..31` is the readout's, so the next
	// panel field starts a third range. The readout's own speed row is
	// `PanelFieldSpeed` at 21 and states the same quantity in the other box;
	// two names for two boxes, and neither resolver answers for the other.
	PanelFieldMoveSpeed PanelField = 32

	// PanelFieldMana is the mana pair, as `<mana>/<max>` — the health pair's
	// own form. It is the next free value in the shared space three constants
	// up: `0..31` are spent (this box's `0..15`, the readout's `16..31`) and
	// MoveSpeed just took 32, so mana takes 33.
	PanelFieldMana PanelField = 33

	// PanelFieldExperience, PanelFieldProtection and PanelFieldResistance are
	// the rest of the recompute's own derived set reaching the window: the
	// character's experience, his five elemental protections and his five
	// damage-kind resistances. They run 34, 35, 36 by the same allocation rule
	// three constants up applied one more time — `0..31` are spent (this
	// box's `0..15`, the readout's `16..31`), MoveSpeed took 32 and Mana took
	// 33, so this run is the next three free values rather than a reopening of
	// either box's own range.
	PanelFieldExperience PanelField = 34
	PanelFieldProtection PanelField = 35
	PanelFieldResistance PanelField = 36

	// The six skill slots and the five elemental protections, ONE FIELD EACH
	// (0140). The two rows above state the same numbers space-separated, and
	// they stay — a layout may still want a family on one line — but a sheet
	// that puts a name beside every number needs a field per number, because a
	// row states one field and a label belongs to a row.
	//
	// They run 37..47 by the allocation rule at the top of this block applied
	// once more: `0..15` are this box's, `16..31` the readout's, and 32..36 are
	// spent, so this run opens at the next free value rather than reopening
	// either box's range.
	//
	// PanelFieldSkillGeneral IS GATED ON THE BAND and the other five are not.
	// Slot 0 is unstated for a creature — no column of the units collection
	// fills it — so its row is absent there, which is the same answer
	// panelSkillText already gives by printing five positions instead of six.
	// The other five carry a person's trained levels and a creature's
	// weapon-kind numbers, exactly as they already did inside that one row.
	PanelFieldSkillGeneral  PanelField = 37
	PanelFieldSkillBlade    PanelField = 38
	PanelFieldSkillAxe      PanelField = 39
	PanelFieldSkillBludgeon PanelField = 40
	PanelFieldSkillPike     PanelField = 41
	PanelFieldSkillShooting PanelField = 42

	// The five elemental protections, Fire, Water, Air, Earth, Astral — the
	// COLUMN ORDER `UNIT-COMBAT-015` publishes for slots 19..23, which is the
	// order UnitCharacter.Protection is filled in and NOT the order a damage
	// resolver indexes the same five fields in (`UNIT-COMBAT-006`). A layout
	// that reordered these labels without reordering that array would put a
	// true number under a false name.
	PanelFieldProtFire   PanelField = 43
	PanelFieldProtWater  PanelField = 44
	PanelFieldProtAir    PanelField = 45
	PanelFieldProtEarth  PanelField = 46
	PanelFieldProtAstral PanelField = 47

	// PanelFieldSight is how far the unit sees — the derived graph's own Sight
	// for a person and the scan-range column for a creature, carried across on
	// UnitCharacter like the four statistics beside it.
	PanelFieldSight PanelField = 48

	// The four HEADING fields: a row that draws its label and no value (0140).
	// Each resolves to the empty string and reports that it HAS a value, so the
	// row survives panelItems' drop rule and composeItems paints the label
	// alone.
	//
	// THEY EXIST BECAUSE A HEADING IS NOT A LABEL WITH NO ROW. panelItems drops
	// a row no cell of which resolved, so a heading spelled as a label over an
	// unresolvable field would vanish; and a heading spelled as a label on the
	// row BELOW it would move whenever that row's own gate closed. A field of
	// its own is the only shape that puts the word where the author put it and
	// takes it away when the block under it is absent.
	//
	// EACH CARRIES THE GATE OF THE BLOCK IT HEADS, which is the whole reason
	// there are four rather than one: a heading over an absent block is worse
	// than no heading, so MANA's closes on a unit with no mana pool exactly as
	// the mana row's does, and the two skill/resistance headings close with
	// Char.Known exactly as every row beneath them does.
	PanelFieldHealthHeading PanelField = 49
	PanelFieldManaHeading   PanelField = 50
	PanelFieldSkillsHeading PanelField = 51
	PanelFieldResistHeading PanelField = 52
)

const (
	// PanelFieldBlank is an always-present, always-empty cell. This is what
	// lets SIGHT and SPEED (spec B3's own diagram) sit on two separate
	// right-column-only rows instead of being paired on one row spanning both
	// columns.
	PanelFieldBlank PanelField = 54

	PanelFieldManaCardHeading PanelField = 55
	PanelFieldManaCard        PanelField = 56

	// PanelFieldWeight is what the subject is carrying: the load field of the
	// simulation's own actor, drawn with one fractional digit.
	//
	// THE FRACTIONAL DIGIT IS AUTHORED AND THE VALUE IS NOT. The load is a
	// decoded quantity with a decoded derivation; how the original's own
	// sheet turns that integer into the number on screen is not decoded,
	// and this build divides by ten. DIVERGENCES.md DIV-222 carries the row.
	//
	// It is stated for a subject whose producer filled it in
	// (PanelSubject.WeightKnown) and that is a player character (DIV-1857):
	// every other unit leaves the row, and the spellcaster caption takes it.
	PanelFieldWeight PanelField = 57

	// PanelFieldSpellcaster is the installed SPELLCASTER caption, stated only
	// for a mage at the original panel's full visibility level. Keeping both
	// flags in those existing rows preserves the fixed card's row population
	// and 160x242 geometry.
	PanelFieldSpellcaster PanelField = 58

	// PanelFieldRole is an optional installed role line above the subject's own
	// name.
	PanelFieldRole PanelField = 59
)

// UnitCombat carries one unit's simulation-owned combat projection. It is
// comparable so it can be part of the panel refresh key. Known distinguishes
// an unfilled projection from valid zero-valued combat numbers.
type UnitCombat struct {
	Known bool

	// DamageBase and DamageSpread are the roll `base + U[0, spread]`. They are
	// NOT a minimum and a maximum; the row composes the sheet's own pair.
	DamageBase, DamageSpread int

	// WeaponSpellKnown marks a spell release replacing this unit's physical
	// attack, including releases such as Stone Curse that cause no damage.
	WeaponSpellKnown bool

	// WeaponSpellDamageKnown says this unit's current attack is replaced by a
	// damaging weapon spell. Its pair is the same base + U[0, spread] interval
	// the live release uses, kept at int64 because the spell calculation is.
	WeaponSpellDamageKnown             bool
	SpellDamageBase, SpellDamageSpread int64

	ToHit, Defence, Absorption int

	// AttackCharge and AttackRelax are the cadence in ticks.
	AttackCharge, AttackRelax int

	// AlwaysHits is the mark that skips the hit roll.
	AlwaysHits bool
}

// PanelSkillSlots includes General and the five trained skill slots. It stays
// local so this presentation package need not import pkg/data.
const PanelSkillSlots = 6

// CharacterBand says which definition collection a character's numbers were
// read from: the creature band, the person band, or — its zero value —
// neither. It is named for what pkg/mapload's own Band already calls the
// split, and for the word this repo already uses for it
// (cmd/classdump/databin.go's bandOf): a creature resolved against the units
// collection, a person against the humans one.
//
// IT IS THREE-VALUED AND NOT TWO, unlike pkg/mapload's Band, and that is a
// deliberate difference from the tier below rather than a drift from it.
// pkg/mapload's Band rides on a map lookup that is either present or absent,
// so its own third state — "nothing stated" — is spent on absence from
// that map and never needs a value of the type at all. This type rides on
// UnitCharacter, which is already comparable and already carries an explicit
// Known flag for "nothing stated" about the character as a whole; giving it
// only two values and reusing Known for the third state would make one state
// expressible two ways — Known == false, or this field left unset on a
// Known character — and let the two disagree. A bare bool has the same
// defect one level down, so the type is an enumeration with an explicit
// unknown case instead.
//
// THE ZERO VALUE IS CharacterBandUnknown ON PURPOSE: every UnitCharacter
// this package built before this story, and every
// one a caller outside pkg/ui still constructs without setting this field,
// keeps meaning exactly what an unfilled UnitCharacter meant before this
// story existed — the skill row's own six-wide, General-included form below.
//
// It is a small integer and nothing wider because PanelSubject carries a
// UnitCharacter and MUST stay comparable with == — the panel's refresh key
// holds one — and a comparable value is the whole of what this type is
// required to be.
type CharacterBand uint8

const (
	// CharacterBandUnknown is the zero value: this character's band is not
	// stated. Every UnitCharacter built anywhere before this story carries
	// it, and so does one a caller never sets the field on.
	CharacterBandUnknown CharacterBand = iota
	// CharacterBandPerson is a character resolved against the humans
	// collection, by any of the three routes that reach it — a party member
	// among them (0137 spec Vocabulary "Band"). Its skill row states all six
	// positions, General included.
	CharacterBandPerson
	// CharacterBandCreature is a character resolved against the units
	// collection. Its skill positions 1..5 carry its weapon-kind columns and
	// not a skill level of any kind, and position 0 (General) is unstated —
	// no column of that collection fills it. The skill row's own formatter
	// reads this value to print five positions instead of six, rather than a
	// sentinel in the unstated one.
	CharacterBandCreature
)

// PanelWornSlots is the width of a subject's worn row: twelve, the worn set's
// own slot count (0128 spec Terms, "Worn set"). It is a LOCAL constant for
// PanelSkillSlots' own reason: this package takes no import from pkg/sim, so
// it cannot spell sim.EquipSlots and must carry the number itself.
const PanelWornSlots = 12

// UnitCharacter is what a unit's sheet states beyond those eight: the four
// statistics a blow's numbers were derived from, the six skill levels he
// carries, and the weapon in his hand.
//
// THE SENTENCE THIS DOC BLOCK USED TO OPEN WITH — "a fact about the load
// and not about the world" — no longer covers the whole type. Body,
// Reaction, Mind, Spirit and Weapon still are: none of them is on an entity,
// none is hashed, and none survives a world's byte form — they are the
// loader's inputs, fixed when the map opened. Skills and Experience are NOT,
// any more: they are read off the simulation entity ON THE TICK THE READOUT
// IS BUILT, the way the eight combat numbers beside this struct already are,
// so they change tick to tick, are hashed, and survive the byte form as part
// of the entity's own state — this struct only carries a COPY, and the map
// that seeds one supplies their starting value, which the next readout built
// overwrites. Known still says the tier that placed this unit knew a
// character for it — which USED TO MEAN the party and now means any unit
// whose placement reached a definition entry, so the unit that states
// nothing here is the one nothing could be resolved for rather than every
// unit on the field but one.
type UnitCharacter struct {
	Known         bool
	UnitNameIndex int
	Name          string
	// Mage selects the second, spell-school name carried by each of the five
	// shared skill slots. The level stays in the same slot; only its
	// presentation changes. Keeping this beside Skills prevents a panel from
	// correctly carrying Water at slot 2 while labelling that value Axe.
	Mage bool

	// Band says which definition collection this character's numbers were read
	// from — CharacterBandUnknown, CharacterBandPerson or
	// CharacterBandCreature — and its own doc carries the full reason it is
	// three-valued. The only reader that branches on it today is the skill
	// row's formatter below, which a creature narrows to five positions; no
	// other row's gate or source changes on it.
	Band CharacterBand

	Body, Reaction, Mind, Spirit int

	// Skills is the six per-slot levels, in slot order — slot 0 General,
	// slots 1..5 the five the character sheet shows (0125 spec Vocabulary).
	// EVERY VALUE IS STATED, zero included: unlike the single named slot this
	// replaces, there is no "trained nothing" case that withholds the row — a
	// zero here is a fact about the slot, not an absence of one.
	Skills [PanelSkillSlots]int

	// Weapon is what he holds, by name, and empty for a bare hero.
	Weapon string

	// Experience is the total of the six per-slot experiences the entity
	// carries — SkillXP's own sum, not a recompute of the levels above:
	// unlike Body/Reaction/Mind/Spirit, this number is INCREMENTED as blows
	// land rather than derived fresh each time it is read. It is stated only
	// when Known, exactly as the four statistics above are.
	Experience int

	// Protection and Resistance are the five elemental protections and the
	// five damage-kind resistances a recompute produces. Protection is displayed
	// in Fire-through-Astral column order. Resistance is displayed in
	// Blade-through-Shooting order, which is also the order XPSlot 1..5 selects
	// in the physical resolver (1039).
	//
	// FIXED-SIZE ARRAYS OF int, DELIBERATELY NOT SLICES: UnitCharacter is
	// carried inside PanelSubject, which must stay comparable with == because
	// the panel's own refresh key holds one, and a [5]int is comparable where
	// a []int would make the whole subject stop being one.
	Protection [5]int
	Resistance [5]int

	// Sight is how far this unit sees, in cells: the recompute's own derived
	// Sight for a person, the scan-range column for a creature (0140). It sits
	// here rather than beside Speed on PanelSubject because it arrives from the
	// same two producers the four statistics do and on the same gate — a unit
	// nothing could be resolved for states no sight either.
	//
	// Native derived profiles carry whole cells. Original humans can also
	// supply Sight256 below, preserving their sheet's fractional readout.
	Sight int
	// Original humans retain fractional sight in unsigned 8.8 units.
	Sight256 uint16
}

// CharacterSkillName is the one presentation mapping used by both the live
// panel and the production headless trace. Slots are shared by the two
// archetypes: a mage's slot 2 is Water, where a fighter's is Axe. Slot 0 is
// General for both and an out-of-range slot names nothing.
func CharacterSkillName(mage bool, slot int) string {
	if slot == 0 {
		return "General"
	}
	if slot < 1 || slot > 5 {
		return ""
	}
	if mage {
		return [...]string{"", "Fire", "Water", "Air", "Earth", "Astral"}[slot]
	}
	return [...]string{"", "Blade", "Axe", "Bludgeon", "Pike", "Shooting"}[slot]
}

func panelSkillSlot(f PanelField) (int, bool) {
	switch f {
	case PanelFieldSkillGeneral:
		return 0, true
	case PanelFieldSkillBlade:
		return 1, true
	case PanelFieldSkillAxe:
		return 2, true
	case PanelFieldSkillBludgeon:
		return 3, true
	case PanelFieldSkillPike:
		return 4, true
	case PanelFieldSkillShooting:
		return 5, true
	}
	return 0, false
}

func panelLabel(s PanelSubject, f PanelField, fallback string) string {
	if slot, ok := panelSkillSlot(f); ok {
		base := 30
		if s.Char.Mage {
			base = 36
		}
		if slot > 0 && s.Words.PanelCaptions[base+slot-1] != "" {
			return s.Words.PanelCaptions[base+slot-1]
		}
		return strings.ToUpper(CharacterSkillName(s.Char.Mage, slot))
	}
	index := map[PanelField]int{
		PanelFieldBody: 15, PanelFieldReaction: 16, PanelFieldMind: 17, PanelFieldSpirit: 18,
		PanelFieldHealthHeading: 19, PanelFieldManaHeading: 20, PanelFieldManaCardHeading: 20,
		PanelFieldSight: 21, PanelFieldMoveSpeed: 22, PanelFieldDamage: 23,
		PanelFieldAbsorption: 24, PanelFieldToHit: 25, PanelFieldDefence: 26,
		PanelFieldSkillsHeading: 27, PanelFieldResistHeading: 28,
		PanelFieldWeight: 35, PanelFieldExperience: 46,
		PanelFieldSpellcaster: 190, PanelFieldArmorPiercing: 191,
		PanelFieldProtFire: 41, PanelFieldProtWater: 42, PanelFieldProtAir: 43,
		PanelFieldProtEarth: 44, PanelFieldProtAstral: 45,
	}[f]
	if index != 0 && s.Words.PanelCaptions[index] != "" {
		return s.Words.PanelCaptions[index]
	}
	return fallback
}

// PanelCorner is the window corner the panel is anchored to.
type PanelCorner uint8

const (
	PanelTopLeft     PanelCorner = 0
	PanelTopRight    PanelCorner = 1
	PanelBottomLeft  PanelCorner = 2
	PanelBottomRight PanelCorner = 3
)

// PanelSubject is EVERYTHING THE PANEL STATES: the one unit it describes — its
// stated values and the id that says which unit they belong to — and how many
// units the selection it was drawn from presently holds.
//
// It is a value type of builtins and one point, so it is copied by assignment
// and compared with ==, which is what lets the refresh rule be a comparison of
// two of these rather than a walk of a snapshot. It names no simulation type:
// the id is the id's own width, carried across the seam like every other.
//
// SELECTED IS A PROPERTY OF THE SELECTION, NOT OF THE UNIT, and it lives on
// a value named for one unit ON PURPOSE — a reader who does not know why
// will move it out. What the picture is a function of is exactly what
// decides when it is redrawn, because panelKey holds this value and nothing
// else about what is stated. Carried anywhere else, a selection that grew or
// shrank while the described unit stayed identical would compare equal, and
// the panel would go on stating the old number until something about the
// unit happened to change. THE TWO GROUPS RIDE INSIDE IT rather than beside
// it. Both are comparable values of builtins, so the whole subject is still
// copied by assignment and compared with ==, and the refresh rule below
// therefore covers a statistic and a combat number for free — a value that
// moved the picture and was not in here would leave a stale box on screen.
type PanelSubject struct {
	Kind    InspectionKind
	ClassID int32
	ID      uint32
	Name    string
	// Role is an optional presentation-only line immediately above Name in
	// CompactPanelLayout. It is separate from Name so an installed role does
	// not replace the actual subject identity selected by PanelSubjectName.
	Role      string
	HP, MaxHP int
	// Mana and MaxMana are the mana pair, in the health pair's own shape: a
	// maximum of zero is a unit with no mana pool.
	Mana, MaxMana int
	Cell          image.Point
	// Unplaced marks a transient subject which has not entered a map. It keeps
	// the shared sheet honest: generation states CELL -, - rather than
	// inventing a map coordinate before Play; a live Viewer always leaves it
	// false and therefore states its actual map cell.
	Unplaced bool
	Selected int

	// UnitNameIndex and DetailLevel are presentation-only inputs decoded by
	// TEXT-UI-034..036. UnitNameIndex supplies the class-name fallback only
	// when Name is empty. They never enter simulation or persistence. Current
	// town, shop and generator panels use full detail (7); the mission panel
	// states the unit's knowledge level.
	UnitNameIndex int
	DetailLevel   int
	DetailSet     bool
	Words         Words

	// OriginalPanel carries the three actor values which gate the two final
	// fixed captions in the original information panel. They stay in their
	// original shape deliberately: Flags is the actor word at +0x18c, XPValue
	// the dword at +0x1c, and Byte14A the derived byte at +0x14a. Naming either
	// caption after a convenient simulation property loses the original
	// predicates: slot 190 tests bit 0 clear and XPValue nonzero, while slot
	// 191 tests bits 0 and 4 clear and Byte14A nonzero.
	//
	// This is presentation-only state. It is copied into the panel key with
	// the rest of PanelSubject and reaches neither simulation nor persistence.
	OriginalPanel OriginalPanelActor

	Combat UnitCombat
	Char   UnitCharacter

	// Speed is the unit's own rate input, and its row is gated on Combat.Known
	// rather than on a flag of its own. The two arrive off the same entity in
	// the same push, so one flag covers both and a second would be a second
	// thing to forget; a subject nobody filled in states neither.
	Speed int

	// Worn is the worn set's own row names, one per equipment slot in slot
	// order, empty for a slot nothing fills. It carries a NAME ALONE and
	// nothing this package could derive one from: the panel formats nothing
	// about an item and looks nothing up, so what reaches this field is already
	// a piece's own row name, turned from a worn code by whichever producer
	// holds the item collections — not the running window, which carries
	// none.
	//
	// IT IS A FIXED-SIZE ARRAY AND NOT A SLICE, for UnitCharacter.Skills' own
	// reason above: PanelSubject is carried inside panelKey and compared with
	// ==, and a slice would end that.
	Worn [PanelWornSlots]string

	// Weight is the subject's carried load in the weight column's own units,
	// and WeightKnown is whether a producer filled it in.
	//
	// THE PAIR IS A VALUE AND A FLAG rather than a value alone, because zero
	// is a real load: a character with an empty doll and an empty pack carries
	// nothing, and the sheet states `0.0` for him. A subject nobody filled in
	// has to be distinguishable from that, and no other field of this struct
	// answers for it — Combat.Known and Char.Known are set by producers that
	// know nothing about what a unit is carrying.
	//
	// IT IS THE LOAD AND NOT THE WORN WEIGHT. What the actor is wearing counts
	// in full and what he is carrying counts half; the two are combined by one
	// statement of the law (sim.CarriedLoad) before they reach this field, and
	// this package neither knows nor restates it.
	Weight      int
	WeightKnown bool

	// KnownSpells is the subject's own spellbook membership bitmask, carried
	// unchanged from the simulation entity that produced this subject —
	// MapEntity's own field of the same name. It is zero for a subject no
	// producer filled in and for one that simply knows no spell.
	//
	// It is read only for the monster spell-list hover hint
	// (TEXT-HOVERTEXT-052), and only when Char.Band is CharacterBandCreature:
	// the cited card helper builds that formatted list for a monster, not for
	// a person's own spellbook, which the mission spell bar already states.
	KnownSpells uint32
}

// OriginalPanelActor is the bounded actor-state projection consumed only by
// the two conditional installed captions. Known distinguishes an old caller
// which supplied none of these values from an actor whose values are all zero.
type OriginalPanelActor struct {
	Known   bool
	Flags   uint32
	XPValue int32
	Byte14A uint8
}

// PanelCell is one labelled or unlabelled field. A row's left cell has
// always had this shape, unnamed; a row's optional right cell is now a value
// of it.
type PanelCell struct {
	Field PanelField
	Label string
}

// PanelRow is one line of the panel: which field it states, the label drawn
// before the value, an OPTIONAL second cell sharing the line, and — in a
// placed layout — where inside the panel it sits.
//
// Label is drawn as bytes by the same font the value is, so a label may hold
// any byte the atlas has a record for. An EMPTY label is not a blank label: the
// value then begins at the row's own origin and the layout's label gap is not
// spent, which is what lets a title row sit flush with the labels beneath it
// instead of indented by a gap that follows nothing.
type PanelRow struct {
	// FullWidth lets an unpaired weight or XP value use the whole card.
	FullWidth bool
	Field     PanelField
	Label     string
	At        image.Point

	// BeforeRows leaves this many flowing line slots before the row. It is
	// ignored by placed layouts. Installed compact cards apply their own
	// grouped row positions after the generic flowing layout.
	BeforeRows int

	// Right is the row's optional second cell. A nil Right states one field per
	// row, exactly as every row did before this story — which is what makes a
	// layout that never sets it, like the readout's, compose the identical
	// picture it composed before, by construction rather than by a guard
	// anywhere in this file.
	//
	// IT IS A POINTER TO A SMALL TYPE, NOT A SECOND (Field, Label) PAIR BESIDE
	// THE FIRST: PanelField's zero value is PanelFieldName, so a bare second
	// Field would silently name the name field for every row that carries no
	// second cell — every literal already in this repo, edited to say "none"
	// instead of left unset. A nil pointer has no such middle state, and
	// PanelLayout is already non-comparable (it holds a slice), so a pointer
	// inside it costs nothing that was being relied on.
	Right *PanelCell

	Center bool

	LabelIndent int
}

// PanelLayout is the WHOLE of what the panel looks like — its geometry, its
// colours, its background, its labels, and which field each row states in which
// order. The composition below reads its appearance from this value and from
// nothing else, so there is no dimension, colour, string or ordering anywhere
// in the drawing path for it to disagree with.
//
// THIS IS THE SUBSTITUTION POINT. The original's information panel is not
// decoded — the research establishes positively that its layout cannot be
// settled without the interface layer, so what this project ships is AUTHORED.
// Replacing it later with the original's is supplying a different value of this
// type: a background picture, placed rows at its own offsets, its own colours.
// Nothing else moves. The same seam is what makes the panel customisable, which
// is that goal seen from the user's side rather than the archaeologist's.
//
// It is handed out by AuthoredPanelLayout as a fresh value rather than held in
// a package variable: a variable carrying a row slice is writable from anywhere
// and would make "two viewers sharing a layout produce the same panel" false by
// mutation rather than by design.
type PanelLayout struct {
	// Corner and Margin place the panel in the window.
	Corner PanelCorner
	Margin image.Point

	// Size is the panel's box. A zero width or height means FIT TO CONTENT on
	// that axis: the box is derived from the rows, so a fit box clips none of
	// them. MinWidth is a floor on a fitted width and never a ceiling — a name
	// wider than it widens the panel, since a truncated name is a third thing
	// beside stating a value and omitting it.
	Size     image.Point
	MinWidth int

	// Pad is the inset from the box to the content, on all four sides. Gap is
	// the vertical space between flowing rows. LabelGap is the space left after
	// a label's own pen before its value begins.
	//
	// ColumnGap is that same space, spent once per box rather than once per
	// row: the gap between the widest DRAWN left cell and the shared
	// right-column origin. It is a layout field and not a constant for the
	// substitution point's own reason — what the panel looks like is a value
	// of PanelLayout — and a zero ColumnGap is legal: it means the two
	// columns touch, which is the author's business.
	Pad       image.Point
	Gap       int
	LabelGap  int
	ColumnGap int

	AlignValues bool

	// FixedColumns divides a fixed-width box into two content columns whose
	// widths and origins depend only on the box, padding and ColumnGap. Text
	// may be fitted inside those columns, but can never move their boundary.
	// It is deliberately opt-in so fit-to-content and legacy layouts retain
	// their content-derived geometry.
	FixedColumns bool
	// CompactCard selects the narrow character-card typography and safe inset.
	CompactCard bool
	RightInset  int

	// Flow stacks the rows from Pad, one line height plus Gap apart; otherwise
	// each row is PLACED at its own At.
	//
	// It is one bool on the layout rather than an optional offset per row
	// because a per-row option has a representable middle state — some rows
	// flowing, some placed — that no reader could lay out unambiguously, and
	// one bool has none.
	Flow bool

	// Background is drawn as given at the panel's own origin, clipped to the
	// box and never scaled. A nil one is the authored frame instead: Fill under
	// a one-pixel Border.
	Background *image.RGBA

	Fill       color.RGBA
	Border     color.RGBA
	LabelColor color.RGBA
	ValueColor color.RGBA

	Rows []PanelRow
}

// sidebarWidth is the width of the window's RIGHT-HAND COLUMN, in pixels, and
// the one number both boxes in it are built from: the unit panel below and the
// minimap above (minimap.go). It is authored, like everything about either box.
//
// 400 WAS CHOSEN FROM A MEASUREMENT AND 300 OVERRIDES IT, which is a cost and
// not a free change: cmd/paneldump over missions 10 and 20 against both installs
// composes the fullest party hero at 387 pixels wide and a placed creature at
// 288, so 400 cleared the widest panel this tree draws and 300 does not. The
// panel's width is fixed, so a row wider than it — a WORN list of many pieces, a
// very long class name — is CUT by the font's own bounds test rather than
// widening the box. The creature panel still clears it; the fullest hero sheet
// loses its tail. That is the trade the ruling costs, it is measured rather than
// guessed, and it is written here because it is the one thing a reader of this
// constant cannot see.
//
// IT WAS 300 AND IS NOW 160 (owner). `SESS-VIEW-028` (High) gives the map
// view's own construction rect as the screen minus a 160-pixel right strip,
// and the mission's own unit panel — formerly 300 wide and fit-to-content,
// ~373 tall for a shipped party member (`DIV-213`) — is replaced by the
// reused CompactPanelLayout card in that strip's own decoded id-7 slot
// (hud.go's characterPanelBoxRect). The doll follows it in the newly
// available id-8 slot; the worn set does not fit there and is not drawn
// during a mission (`DIV-200`). sidebarWidth is MissionPanelW under a second
// name rather than a second literal, so the two cannot drift the way 400 and
// the minimap's own former fallback once did.
//
// AuthoredPanelLayout ITSELF NO LONGER READS THIS CONSTANT (below,
// authoredPanelWidth instead): it is not part of the mission's right column
// any more (viewer.go's default panelLayout is CompactPanelLayout), and its
// own remaining callers — the chargen preview's oracle and the release-
// integration Card, both `pkg/game` tests — still measure a 300-wide box. A
// panel that is no longer drawn in the column it was once sized for has no
// reason to shrink when the column does.
const sidebarWidth = MissionPanelW

const authoredPanelWidth = 300

// AuthoredPanelLayout is what this project ships: a frame in the BOTTOM RIGHT
// carrying the subject's character sheet — the four statistics against the two
// pools, what a blow throws against what it meets, the skill column against the
// elemental one, then the totals.
//
// THE CORNER AND THE ARRANGEMENT ARE THE OWNER'S. He moved the box to the
// bottom right and gave the arrangement as a photograph of the original's
// own character sheet, asking for it "by the placement of the information".
// So the ORDER and the PAIRING below are his; the geometry, the colours, the
// corner margin and the abbreviations remain ours, and `UNIT-PANEL-011`'s
// finding — that the original's own layout cannot be recovered from the
// executable at all — is unchanged and is why this is still an authored
// value and not a reproduction.
//
// WHAT THIS HAS AND THE PHOTOGRAPH DOES NOT, and why each stays:
//   - GENERAL, the zeroth skill slot. The original's sheet does not show it and
//     this tree carries it, so dropping it would destroy the only place a
//     reader can see it. It is absent for a creature, which states no such slot.
//   - SELECTED, WEAPON, WORN, SWING and CELL — every one of them a value this
//     panel already stated before this story. They follow the sheet rather than
//     being dropped into it, so the block above reads as the photograph and
//     nothing that was on screen has left it.
//
// THE WEAPON-KIND FAMILY IS THE ONE ROW THIS STORY DID DROP, and it destroys no
// unique information, which is the test. For a CREATURE those five numbers are
// the same five the skill column above prints — pkg/mapload copies one array
// into the other at build time so the two can never disagree (sheet.go's own
// FR-4a note). For a PERSON they are five zeros: nothing in this build derives
// the family for a human and no equipment modifier fills it either
// (data.EquipMod's own doc). Five zeros and a duplicate are what came off the
// panel, not a fact.
//
// Every value in it is OURS. No source ranks these fields, sets these colours
// or puts the panel in this corner, and none is claimed to reproduce anything.
// THE COUNT ROW'S LABEL IS OURS TOO (0055 spec, "The wording is ours"): nothing
// decoded in this tree names the text the original puts beside that number, or
// says whether it puts it on this panel at all. The number is the selection's;
// the word in front of it is a choice.
//
// WHERE THE ARRANGEMENT CAME FROM, AND WHY THAT IS NOT A DECODE. `UNIT-PANEL-011`
// establishes positively that the original's own layout cannot be recovered from
// the executable at all — past `+0x14a` the block is addressed by computed index
// and no displacement sweep can name a consumer — and that finding stands. What
// changed is the SOURCE OF THE AUTHORED CHOICE, not its status: the owner, who
// rules as author, gave the order and the pairing from his own screen. Owner
// testimony about the game is a question (B3); an owner RULING about what this
// project should look like is a fact, and a layout is his to rule on. Nothing
// below is offered as evidence of what the original does, and no claim id is
// cited for any of it.
//
// A LABEL MAY BE ABBREVIATED; A VALUE MAY NOT. `DMG` and `ABS` trade letters
// of a label for room in the second column; no number or string any field
// states is touched. The labels themselves are the photograph's own words
// where it has one — BODY, AGILITY, MIND, SPIRIT, ATTACK, DEFENSE, SKILLS,
// RESISTANCE — including `AGILITY` for the statistic this repo's own code
// calls Reaction, which is the one place a label and a field name now
// deliberately disagree.
//
// COLUMNGAP IS 14: picked for this font's own scale, not decoded — nothing
// dates a gap between two authored columns any more than it dates the columns
// themselves.
func AuthoredPanelLayout() PanelLayout {
	return PanelLayout{
		Corner: PanelBottomRight,
		Margin: image.Pt(12, 12),
		// THE WIDTH IS PINNED AND THE HEIGHT FITS. panelBox fits each axis
		// independently, so naming X alone is what gives a fixed column whose
		// height still follows the rows a subject actually states. MinWidth is
		// deliberately NOT set beside it: it applies to a fitted width only
		// (panelBox's own doc), so a value here would be dead configuration
		// that reads like a second, disagreeing rule.
		Size:      image.Pt(authoredPanelWidth, 0),
		Pad:       image.Pt(10, 8),
		Gap:       3,
		LabelGap:  8,
		ColumnGap: 14,
		Flow:      true,

		Fill:       color.RGBA{R: 0x10, G: 0x12, B: 0x18, A: 0xff},
		Border:     color.RGBA{R: 0x8a, G: 0x74, B: 0x46, A: 0xff},
		LabelColor: color.RGBA{R: 0x8f, G: 0x9a, B: 0xa8, A: 0xff},
		ValueColor: color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff},

		Rows: []PanelRow{
			// OURS, AND ABOVE THE SHEET: the count states something about the
			// SELECTION rather than about the subject, and it is absent below two —
			// so the single-unit panel, which is the photograph's own case, opens on
			// the name exactly as it does.
			{Field: PanelFieldCount, Label: "SELECTED"},
			{Field: PanelFieldName},

			// The four statistics against the two pools. The pools are STACKED
			// — a heading row, then the pair under it — which is what puts four
			// left-hand statistics beside four right-hand lines.
			{Field: PanelFieldBody, Label: "BODY",
				Right: &PanelCell{Field: PanelFieldHealthHeading, Label: "HEALTH"}},
			{Field: PanelFieldReaction, Label: "AGILITY",
				Right: &PanelCell{Field: PanelFieldHealth}},
			{Field: PanelFieldMind, Label: "MIND",
				Right: &PanelCell{Field: PanelFieldManaHeading, Label: "MANA"}},
			{Field: PanelFieldSpirit, Label: "SPIRIT",
				Right: &PanelCell{Field: PanelFieldMana}},

			// What a blow throws, what it meets.
			{Field: PanelFieldDamage, Label: "DMG",
				Right: &PanelCell{Field: PanelFieldAbsorption, Label: "ABSORB"}},
			{Field: PanelFieldToHit, Label: "ATTACK",
				Right: &PanelCell{Field: PanelFieldDefence, Label: "DEFENSE"}},

			// The two columns, each under its own heading: the skill slots on
			// the left, the five elemental protections on the right. Every row
			// of both blocks is gated on Char.Known, and so are the headings, so
			// the whole block is present or absent together.
			{Field: PanelFieldSkillsHeading, Label: "SKILLS",
				Right: &PanelCell{Field: PanelFieldResistHeading, Label: "RESISTANCE"}},
			{Field: PanelFieldSkillBlade, Label: "BLADE",
				Right: &PanelCell{Field: PanelFieldProtFire, Label: "FIRE"}},
			{Field: PanelFieldSkillAxe, Label: "AXE",
				Right: &PanelCell{Field: PanelFieldProtWater, Label: "WATER"}},
			{Field: PanelFieldSkillBludgeon, Label: "BLUDGEON",
				Right: &PanelCell{Field: PanelFieldProtAir, Label: "AIR"}},
			{Field: PanelFieldSkillPike, Label: "PIKE",
				Right: &PanelCell{Field: PanelFieldProtEarth, Label: "EARTH"}},
			{Field: PanelFieldSkillShooting, Label: "SHOOTING",
				Right: &PanelCell{Field: PanelFieldProtAstral, Label: "ASTRAL"}},
			// OURS: the slot the photograph's sheet does not show and this tree
			// carries. It is under the five rather than above them so the block
			// still reads as the photograph, and it is absent for a creature,
			// which states no such slot.
			{Field: PanelFieldSkillGeneral, Label: "GENERAL"},

			// The photograph's own WEIGHT row, below the two columns and above the
			// totals, which is where it sits on the compact card too.
			{Field: PanelFieldWeight, Label: "WEIGHT",
				Right: &PanelCell{Field: PanelFieldSpellcaster, Label: "SPELLCASTER"}},

			// The totals, each on its own line, as the photograph has them.
			{Field: PanelFieldExperience, Label: "XP",
				Right: &PanelCell{Field: PanelFieldArmorPiercing, Label: "ARMOR PIERCING"}},
			{Field: PanelFieldSight, Label: "SIGHT"},
			{Field: PanelFieldMoveSpeed, Label: "SPEED"},

			// OURS, BELOW THE SHEET — every one of these was on this panel before
			// the arrangement changed, and the photograph simply has no row for it. A
			// name is the widest thing the panel draws, so neither it nor the worn
			// set leaves room for a second cell.
			{Field: PanelFieldWeapon, Label: "WEAPON"},
			{Field: PanelFieldWorn, Label: "WORN"},
			{Field: PanelFieldSwing, Label: "SWING"},
			{Field: PanelFieldCell, Label: "CELL"},
		},
	}
}

var compactPanelInsetMemo struct {
	sync.Mutex
	seed  maphash.Seed
	byKey map[compactPanelInsetKey]image.Point
}

type compactPanelInsetKey struct {
	bounds image.Rectangle
	sum    uint64
}

func compactPanelInset(background *image.RGBA) image.Point {
	fallback := image.Pt(4, 3)
	if background == nil {
		return fallback
	}
	b := background.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return fallback
	}
	memo := &compactPanelInsetMemo
	memo.Lock()
	if memo.byKey == nil {
		memo.seed, memo.byKey = maphash.MakeSeed(), map[compactPanelInsetKey]image.Point{}
	}
	var sum maphash.Hash
	sum.SetSeed(memo.seed)
	memo.Unlock()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		at := background.PixOffset(b.Min.X, y)
		sum.Write(background.Pix[at : at+4*b.Dx()])
	}
	key := compactPanelInsetKey{bounds: b, sum: sum.Sum64()}
	memo.Lock()
	cached, ok := memo.byKey[key]
	memo.Unlock()
	if ok {
		return cached
	}
	inset := measureCompactPanelInset(background)
	memo.Lock()
	if len(memo.byKey) >= 64 {
		clear(memo.byKey)
	}
	memo.byKey[key] = inset
	memo.Unlock()
	return inset
}

func measureCompactPanelInset(background *image.RGBA) image.Point {
	b := background.Bounds()

	counts := make(map[color.RGBA]int)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		run, n := background.RGBAAt(b.Min.X, y), 0
		for x := b.Min.X; x < b.Max.X; x++ {
			c := background.RGBAAt(x, y)
			if c == run {
				n++
				continue
			}
			counts[run] += n
			run, n = c, 1
		}
		counts[run] += n
	}
	var fill color.RGBA
	best := -1
	for c, n := range counts {
		if n > best {
			best, fill = n, c
		}
	}

	left := 0
	for x := b.Min.X; x < b.Max.X; x++ {
		n := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if background.RGBAAt(x, y) == fill {
				n++
			}
		}
		if n*4 >= b.Dy() {
			left = x - b.Min.X
			break
		}
	}

	top := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		n := 0
		for x := b.Min.X; x < b.Max.X; x++ {
			if background.RGBAAt(x, y) == fill {
				n++
			}
		}
		if n*4 >= b.Dx() {
			top = y - b.Min.Y
			break
		}
	}

	return image.Pt(left+2, top+2)
}

// compactPanelW and compactPanelH are the card's fixed size in both callers:
// the chargen preview (`chargen_page.go`), the town statistics pane
// (`townshell.go`), and, since story `1023`, the mission's own character
// panel (`hud.go`'s characterPanelBoxRect, which reads compactPanelH to place
// the doll below it). One name in one place, so the three cannot disagree the
// way sidebarWidth's own former 400/300 pair once did.
const (
	compactPanelW = 160
	compactPanelH = 242
)

func CompactPanelLayout(background *image.RGBA) PanelLayout {
	rightInset := 0
	if background != nil {
		rightInset = 19
	}
	return PanelLayout{
		CompactCard:  true,
		RightInset:   rightInset,
		Size:         image.Pt(compactPanelW, compactPanelH),
		Pad:          compactPanelInset(background),
		Gap:          1,
		LabelGap:     2,
		ColumnGap:    6,
		Flow:         true,
		AlignValues:  true,
		FixedColumns: true,
		Background:   characterCardBackground(background),

		Fill:       color.RGBA{R: 0x10, G: 0x12, B: 0x18, A: 0xff},
		Border:     color.RGBA{R: 0x8a, G: 0x74, B: 0x46, A: 0xff},
		LabelColor: color.RGBA{189, 158, 74, 255},
		ValueColor: color.RGBA{107, 154, 123, 255},

		Rows: []PanelRow{
			{Field: PanelFieldRole, Center: true},
			{Field: PanelFieldName, Center: true},

			{Field: PanelFieldBody, Label: "BODY", BeforeRows: 1,
				Right: &PanelCell{Field: PanelFieldHealthHeading, Label: "HEALTH"}},
			{Field: PanelFieldReaction, Label: "AGILITY",
				Right: &PanelCell{Field: PanelFieldHealth}},
			{Field: PanelFieldMind, Label: "MIND",
				Right: &PanelCell{Field: PanelFieldManaCardHeading, Label: "MANA"}},
			{Field: PanelFieldSpirit, Label: "SPIRIT",
				Right: &PanelCell{Field: PanelFieldManaCard}},

			{Field: PanelFieldDamage, Label: "DMG",
				Right: &PanelCell{Field: PanelFieldAbsorption, Label: "ABSORB"}},
			{Field: PanelFieldToHit, Label: "ATTACK",
				Right: &PanelCell{Field: PanelFieldDefence, Label: "DEFENSE"}},

			{Field: PanelFieldSkillsHeading, Label: "SKILLS",
				Right: &PanelCell{Field: PanelFieldResistHeading, Label: "RESISTANCE"}},
			{Field: PanelFieldSkillBlade, Label: "BLADE",
				Right: &PanelCell{Field: PanelFieldProtFire, Label: "FIRE"}},
			{Field: PanelFieldSkillAxe, Label: "AXE",
				Right: &PanelCell{Field: PanelFieldProtWater, Label: "WATER"}},
			{Field: PanelFieldSkillBludgeon, Label: "BLUDGEON",
				Right: &PanelCell{Field: PanelFieldProtAir, Label: "AIR"}},
			{Field: PanelFieldSkillPike, Label: "PIKE",
				Right: &PanelCell{Field: PanelFieldProtEarth, Label: "EARTH"}},
			{Field: PanelFieldSkillShooting, Label: "SHOOTING",
				Right: &PanelCell{Field: PanelFieldProtAstral, Label: "ASTRAL"}},

			// The photograph's own WEIGHT row, in the vertical space 1022 left for
			// it: below the last resistance row and above the totals.
			{Field: PanelFieldWeight, Label: "WEIGHT", FullWidth: true,
				Right: &PanelCell{Field: PanelFieldSpellcaster, Label: "SPELLCASTER"}},

			{Field: PanelFieldExperience, Label: "XP", FullWidth: true,
				Right: &PanelCell{Field: PanelFieldArmorPiercing, Label: "ARMOR PIERCING"}},
			{Field: PanelFieldSight, Label: "SIGHT", Center: true},
			{Field: PanelFieldMoveSpeed, Label: "SPEED", Center: true},
		},
	}
}

// panelText is the ONE place a subject becomes a field's text, and the one
// place that says which fields exist at all.
//
// It reports whether the field has a value for this subject. TWO fields can
// fail to. The name: a subject with neither an own name nor a resolved class
// name has none. And the count, BELOW TWO — a panel already describes the
// one unit it is about, so a selection of one has nothing to add and says
// nothing, which is the whole of how this story leaves the single-unit panel
// where it found it.
//
// The other two always have one: a unit that exists stands on a cell and
// carries a health pair, and a zero or negative health is a STATE and not an
// absence — those are the numbers that say a unit is down or gone, so they are
// stated as they stand, unclamped.
//
// A field this build does not define answers false, so an unknown constant in a
// layout omits its row rather than drawing an empty one.
func PanelSubjectName(s PanelSubject) (string, bool) {
	if s.Name != "" {
		return s.Name, true
	}
	if s.Kind != InspectionStructure && s.UnitNameIndex >= 0 && s.UnitNameIndex < len(s.Words.UnitNames) && s.Words.UnitNames[s.UnitNameIndex] != "" {
		return s.Words.UnitNames[s.UnitNameIndex], true
	}
	return "", false
}

func panelText(s PanelSubject, f PanelField) (string, bool) {
	if s.Kind == InspectionStructure {
		switch f {
		case PanelFieldHealth:
			return fmt.Sprintf("%d/%d", s.HP, s.MaxHP), true
		case PanelFieldHealthHeading:
			return "", true
		case PanelFieldName, PanelFieldCell:
		default:
			return "", false
		}
	}
	switch f {
	case PanelFieldRole:
		return s.Role, s.Role != ""
	case PanelFieldName:
		return PanelSubjectName(s)
	case PanelFieldHealth:
		return fmt.Sprintf("%d/%d", s.HP, s.MaxHP), panelPoolPositive(s, s.MaxHP) && panelDetailAtLeast(s, 1)
	case PanelFieldMana:
		// THE HEALTH ARM'S OWN SHAPE, and the one field here that can be absent
		// for a reason other than "unknown": a unit with no mana system has a
		// maximum of zero, and that is a fact about the unit rather than a caller
		// that never filled the field in — the same "nothing to say" rule
		// PanelFieldCount already uses, on a different test.
		return fmt.Sprintf("%d/%d", s.Mana, s.MaxMana), s.MaxMana > 0 && panelDetailAtLeast(s, 1)
	case PanelFieldCell:
		if s.Unplaced {
			return "-, -", true
		}
		return fmt.Sprintf("%d, %d", s.Cell.X, s.Cell.Y), true
	case PanelFieldCount:
		return fmt.Sprintf("%d", s.Selected), s.Selected >= 2

	case PanelFieldBody:
		return fmt.Sprintf("%d", s.Char.Body), s.Char.Known && panelDetailAbove(s, 4)
	case PanelFieldReaction:
		return fmt.Sprintf("%d", s.Char.Reaction), s.Char.Known && panelDetailAbove(s, 4)
	case PanelFieldMind:
		return fmt.Sprintf("%d", s.Char.Mind), s.Char.Known && panelDetailAbove(s, 4)
	case PanelFieldSpirit:
		return fmt.Sprintf("%d", s.Char.Spirit), s.Char.Known && panelDetailAbove(s, 4)
	case PanelFieldSkill:
		// UNCONDITIONAL WHEN KNOWN. The old gate here was "a name and a level
		// above zero", which is exactly what hid the row the owner asked for: a
		// hero trained in nothing, or in one slot out of six, now states the same
		// six numbers any other Known character does. Char.Band is passed through
		// so the formatter, not this switch, decides the row's width.
		return panelSkillText(s.Char.Skills, s.Char.Band), s.Char.Known && panelDetailAbove(s, 6)
	case PanelFieldWeapon:
		return s.Char.Weapon, s.Char.Known && s.Char.Weapon != "" && panelDetailAtLeast(s, 7)
	case PanelFieldWorn:
		worn, ok := panelWornText(s.Worn)
		return worn, ok && panelDetailAtLeast(s, 7)
	case PanelFieldExperience:
		return fmt.Sprintf("%d", s.Char.Experience), s.Char.Known
	case PanelFieldProtection:
		return panelFamilyText(s.Char.Protection), s.Char.Known && panelDetailAtLeast(s, 6)
	case PanelFieldResistance:
		return panelFamilyText(s.Char.Resistance), s.Char.Known && panelDetailAtLeast(s, 6)
	case PanelFieldSight:
		if s.Char.Sight256 != 0 {
			// HERO-104: whole = raw >> 8, one decimal truncated toward zero.
			raw := int(s.Char.Sight256)
			return fmt.Sprintf("%d.%d", raw>>8, (raw&0xff)*10>>8), s.Char.Known && panelDetailAbove(s, 1)
		}
		return fmt.Sprintf("%d.0", s.Char.Sight), s.Char.Known && panelDetailAbove(s, 1)

	case PanelFieldSkillGeneral:
		// THE ONE SKILL SLOT WITH A GATE OF ITS OWN: position 0 is unstated for a
		// creature, so this row is absent there rather than stating a zero nothing
		// filled in.
		return fmt.Sprintf("%d", s.Char.Skills[0]),
			s.Char.Known && s.Char.Band != CharacterBandCreature && panelDetailAbove(s, 6)
	case PanelFieldSkillBlade:
		return fmt.Sprintf("%d", s.Char.Skills[1]), s.Char.Known && panelDetailAbove(s, 6)
	case PanelFieldSkillAxe:
		return fmt.Sprintf("%d", s.Char.Skills[2]), s.Char.Known && panelDetailAbove(s, 6)
	case PanelFieldSkillBludgeon:
		return fmt.Sprintf("%d", s.Char.Skills[3]), s.Char.Known && panelDetailAbove(s, 6)
	case PanelFieldSkillPike:
		return fmt.Sprintf("%d", s.Char.Skills[4]), s.Char.Known && panelDetailAbove(s, 6)
	case PanelFieldSkillShooting:
		return fmt.Sprintf("%d", s.Char.Skills[5]), s.Char.Known && panelDetailAbove(s, 6)

	case PanelFieldProtFire:
		return fmt.Sprintf("%d", s.Char.Protection[0]), s.Char.Known && panelDetailAtLeast(s, 6)
	case PanelFieldProtWater:
		return fmt.Sprintf("%d", s.Char.Protection[1]), s.Char.Known && panelDetailAtLeast(s, 6)
	case PanelFieldProtAir:
		return fmt.Sprintf("%d", s.Char.Protection[2]), s.Char.Known && panelDetailAtLeast(s, 6)
	case PanelFieldProtEarth:
		return fmt.Sprintf("%d", s.Char.Protection[3]), s.Char.Known && panelDetailAtLeast(s, 6)
	case PanelFieldProtAstral:
		return fmt.Sprintf("%d", s.Char.Protection[4]), s.Char.Known && panelDetailAtLeast(s, 6)

	case PanelFieldHealthHeading:
		// A HEADING RESOLVES TO NOTHING AND SAYS IT HAS SOMETHING. The empty
		// value is what composeItems paints after the label — nothing — and
		// the true keeps panelItems from dropping the row. Health is stated
		// for every unit that exists, so this heading is too.
		return "", panelPoolPositive(s, s.MaxHP) && panelDetailAtLeast(s, 1)
	case PanelFieldManaHeading:
		// THE MANA ROW'S OWN GATE, restated rather than referenced: a unit with no
		// mana pool has a maximum of zero, and a heading standing over a row that
		// is not there is the defect this gate exists to prevent.
		return "", s.MaxMana > 0 && panelDetailAtLeast(s, 1)
	case PanelFieldSkillsHeading:
		return "", s.Char.Known && panelDetailAbove(s, 6)
	case PanelFieldResistHeading:
		return "", s.Char.Known && panelDetailAtLeast(s, 6)

	case PanelFieldBlank:
		return "", true
	case PanelFieldManaCardHeading:
		return "", (s.MaxMana > 0 || (s.Char.Known && s.Char.Band != CharacterBandCreature)) && panelDetailAtLeast(s, 1)
	case PanelFieldManaCard:
		return fmt.Sprintf("%d/%d", s.Mana, s.MaxMana), (s.MaxMana > 0 || (s.Char.Known && s.Char.Band != CharacterBandCreature)) && panelDetailAtLeast(s, 1)

	case PanelFieldDamage:
		// OWNER-AUTHORED UI SEMANTICS: a staff whose attack releases a spell has
		// no separate physical-damage row. DMG states the spell interval the live
		// release uses; every other weapon states its physical interval.
		if s.Combat.WeaponSpellKnown || s.Combat.WeaponSpellDamageKnown {
			if !s.Combat.WeaponSpellDamageKnown {
				return "", false
			}
			return fmt.Sprintf("%d-%d", s.Combat.SpellDamageBase,
					s.Combat.SpellDamageBase+s.Combat.SpellDamageSpread),
				s.Combat.Known && panelDetailAbove(s, 2)
		}
		return fmt.Sprintf("%d-%d", s.Combat.DamageBase,
				s.Combat.DamageBase+s.Combat.DamageSpread),
			s.Combat.Known && panelDetailAbove(s, 2)
	case PanelFieldToHit:
		return fmt.Sprintf("%d", s.Combat.ToHit), s.Combat.Known && panelDetailAbove(s, 2)
	case PanelFieldDefence:
		return fmt.Sprintf("%d", s.Combat.Defence), s.Combat.Known && panelDetailAbove(s, 3)
	case PanelFieldAbsorption:
		return fmt.Sprintf("%d", s.Combat.Absorption), s.Combat.Known && panelDetailAbove(s, 3)
	case PanelFieldSwing:
		return fmt.Sprintf("%d/%d", s.Combat.AttackCharge, s.Combat.AttackRelax), s.Combat.Known && panelDetailAtLeast(s, 3)
	case PanelFieldArmorPiercing:
		return "", s.OriginalPanel.Known && s.OriginalPanel.Flags&0x11 == 0 &&
			s.OriginalPanel.Byte14A != 0 && panelDetailFull(s)
	case PanelFieldSpellcaster:
		// A creature states the caption only when it holds a spellbook: in
		// every original creature record the experience value is nonzero, so
		// that test alone would caption a plain animal (DIV-1809).
		creatureWithoutSpells := s.Char.Band == CharacterBandCreature && s.KnownSpells == 0
		return "", s.OriginalPanel.Known && s.OriginalPanel.Flags&0x1 == 0 &&
			s.OriginalPanel.XPValue != 0 && !creatureWithoutSpells && panelDetailFull(s)
	case PanelFieldMoveSpeed:
		return fmt.Sprintf("%d", s.Speed), s.Combat.Known && panelDetailAbove(s, 1)
	case PanelFieldWeight:
		return panelWeightText(s.Weight), s.WeightKnown && panelPlayerCharacter(s)
	}
	return "", false
}

func panelPlayerCharacter(s PanelSubject) bool {
	return !s.OriginalPanel.Known || s.OriginalPanel.Flags&0x1 != 0
}

func panelDetailAbove(s PanelSubject, threshold int) bool {
	return !s.DetailSet || s.DetailLevel > threshold
}

func panelDetailAtLeast(s PanelSubject, level int) bool {
	return !s.DetailSet || s.DetailLevel >= level
}

func panelDetailFull(s PanelSubject) bool {
	return !s.DetailSet || s.DetailLevel == 7
}

func panelPoolPositive(s PanelSubject, maximum int) bool {
	return !s.DetailSet || maximum > 0
}

// panelWeightText is a load drawn with one fractional digit: the value's own
// quotient and remainder by ten, which is the decimal-tenths convention
// HERO-104 read at the sheet's other fractional site.
//
// THE DIVISOR IS AUTHORED. The load itself is decoded and so is its
// derivation; the step from that integer to the number the original's sheet
// prints is not, and DIVERGENCES.md DIV-222 carries the row. Nothing else in
// this tree divides the load, so a later decode moves this one expression.
//
// A NEGATIVE LOAD IS SIGNED ONCE AND ITS DIGITS ARE TAKEN FROM THE MAGNITUDE.
// Go's own remainder carries the sign of the dividend, so `-5 / 10` and
// `-5 % 10` are 0 and -5, which would print `0.-5`. A negative load is
// reachable: the shipped Weapons collection carries a row whose weight column
// is -1, so an actor wearing one is carrying less than nothing.
func panelWeightText(v int) string {
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	return fmt.Sprintf("%s%d.%d", sign, v/10, v%10)
}

// panelFamilyText is a five-wide family — a Protection or a Resistance —
// as its five numbers separated by single spaces, e.g. "12 12 12 12 12".
func panelFamilyText(v [5]int) string {
	return fmt.Sprintf("%d %d %d %d %d", v[0], v[1], v[2], v[3], v[4])
}

// panelSkillText is the skill row, in slot order, separated by single spaces
// — e.g. "0 0 0 0 10 0" for a hero trained only in Pike (slot 4) at level
// 10. That still holds on both bands; only the WIDTH now differs between
// them.
//
// band DECIDES THE WIDTH AND NOTHING ELSE. Every other value of band,
// CharacterBandUnknown included, prints the full six — which is what keeps
// a caller that predates this story, and never sets Band at all, drawing
// exactly the row it always drew.
func panelSkillText(v [PanelSkillSlots]int, band CharacterBand) string {
	if band == CharacterBandCreature {
		return fmt.Sprintf("%d %d %d %d %d", v[1], v[2], v[3], v[4], v[5])
	}
	return fmt.Sprintf("%d %d %d %d %d %d", v[0], v[1], v[2], v[3], v[4], v[5])
}

// panelWornText is the worn row: every non-empty name of v, in slot order,
// joined by ", " — and reports NOTHING STATED when none of the twelve
// slots carries one (AC-8). It skips the empty slots rather than joining
// twelve entries with the holes left in: an unfilled slot is not a piece
// with no name, so it contributes neither a name nor a separator, and the
// whole row is the SUBJECT'S OWN "nothing to say" case, which is what keeps
// a subject nobody filled composing the picture it composed before this
// story rather than a row reading "not stated" in place of no row at all.
func panelWornText(v [PanelWornSlots]string) (string, bool) {
	names := make([]string, 0, PanelWornSlots)
	for _, n := range v {
		if n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return "", false
	}
	return strings.Join(names, ", "), true
}

// panelLine is one row resolved against a subject and a font: where it starts
// inside the panel, its two runs of text, and how wide the pair is.
type panelLine struct {
	field  PanelField
	at     image.Point
	label  string
	value  string
	valueX int // relative to at.X
	width  int

	// right, rightLabel, rightValue, rightX and rightValueX are the row's
	// second column, present only when right is true. rightX is the SAME number
	// on every line in the box — the widest drawn left cell plus ColumnGap
	// — which is what keeps the column straight; rightValueX is the right
	// label's own pen plus the gap, relative to rightX, exactly as valueX is
	// relative to at.X for the left cell.
	right       bool
	rightLabel  string
	rightValue  string
	rightX      int
	rightValueX int

	// leftIndent carries PanelRow.LabelIndent through: added to at.X for the
	// label and value draws alone, never for the right cell's, so a row that
	// sets it moves without pulling its own right-column heading with it.
	leftIndent int

	// wrap is the second row of a card name too wide for one, drawn at wrapAt.
	wrap   string
	wrapAt image.Point
}

// panelItem is one row ALREADY RESOLVED TO TEXT: its label, the value that row
// states, and — in a placed layout — where inside the box it sits.
//
// It is the seam the composition splits at. Everything below this line is
// geometry, measurement and paint over a list of these, and knows nothing
// about what a row means; everything above it turns some subject into one.
// That is what lets a second box — the debug readout, whose values come
// from the world and the view rather than from a unit — reuse this whole
// path instead of copying it, without either box being able to reach the
// other's resolution.
type panelItem struct {
	fullWidth         bool
	field, rightField PanelField
	label             string
	value             string
	at                image.Point // a placed layout's own offset; unread in flow
	// preferValue marks a caption resolved from the active install rather
	// than the layout's authored fallback. Only that wider localized-caption
	// case may shorten the caption before its numeric value.
	preferValue bool

	// right, rightLabel and rightValue are the row's second cell, resolved
	// exactly as the first is.
	right                  bool
	rightLabel, rightValue string
	rightPreferValue       bool

	// center carries PanelRow.Center through to layoutLines, which is where
	// a row's own X coordinate is decided.
	center bool

	// indent carries PanelRow.LabelIndent through unchanged.
	indent int

	// beforeRows carries PanelRow.BeforeRows to the flowing Y walk.
	beforeRows int
}

// String is the row as one line of text, in the drawn order: the left cell,
// then — when the row carries a resolved right one — two spaces and the
// right cell. It is what PanelStatement lists, and it is deliberately not a
// format a picture could be reconstructed from.
func (it panelItem) String() string {
	s := it.value
	if it.label != "" {
		s = it.label + " " + it.value
	}
	if it.right {
		r := it.rightValue
		if it.rightLabel != "" {
			r = it.rightLabel + " " + it.rightValue
		}
		s += "  " + r
	}
	return s
}

// panelItems resolves the layout's rows against the subject in row order.
//
// A right cell that resolved over an unresolved left one SLIDES INTO THE
// LEFT POSITION here, not in the drawing and not in the layout author's
// head: there is no such thing as a hole in the left column, so a pairing
// like the shipped layout's SKILL/XP row — a hero who trained nothing has
// no SKILL and still has XP — needs no rule of its own at the call site.
func panelItems(l PanelLayout, s PanelSubject) []panelItem {
	out := make([]panelItem, 0, len(l.Rows))
	for _, r := range l.Rows {
		value, ok := panelText(s, r.Field)
		if l.CompactCard && s.Char.Mage && r.Field == PanelFieldToHit {
			value = ""
		}

		var rLabel, rValue string
		var rOK bool
		if r.Right != nil {
			rValue, rOK = panelText(s, r.Right.Field)
			rLabel = panelLabel(s, r.Right.Field, r.Right.Label)
		}
		label := panelLabel(s, r.Field, r.Label)
		if l.CompactCard && s.Char.Mage && r.Field == PanelFieldToHit {
			label = ""
		}

		it := panelItem{at: r.At, center: r.Center, indent: r.LabelIndent, beforeRows: r.BeforeRows,
			preferValue: label != r.Label, field: r.Field}
		if r.Right != nil {
			it.rightField = r.Right.Field
		}
		switch {
		case ok && rOK:
			it.label, it.value = label, value
			it.right, it.rightLabel, it.rightValue = true, rLabel, rValue
			it.rightPreferValue = rLabel != r.Right.Label
		case ok:
			it.label, it.value = label, value
		case rOK && l.CompactCard && (r.Right.Field == PanelFieldAbsorption || r.Right.Field == PanelFieldDefence):
			it.right, it.rightLabel, it.rightValue = true, rLabel, rValue
			it.rightPreferValue = rLabel != r.Right.Label
		case rOK:
			it.field = r.Right.Field
			it.label, it.value = rLabel, rValue
			it.preferValue = rLabel != r.Right.Label
		default:
			continue
		}
		it.fullWidth = r.FullWidth && !it.right
		out = append(out, it)
	}
	return out
}

// PanelStatement is WHAT THE PANEL SAYS, one string per drawn row, with no
// font, no pixels and no geometry anywhere in it.
//
// It exists so that "the box got smaller" and "nothing it stated was lost" are
// two separate readings rather than one impression: the first is a pixel
// measurement that needs the game's own font and therefore a developer tool,
// and the second is this — a list a reader can diff across two layouts, and a
// test can compare without composing anything.
//
// A row's cells are joined by two spaces and a labelled cell reads
// `LABEL value`, which is the drawn order; an unlabelled cell is its value
// alone. A row every cell of which has nothing to state contributes no entry,
// exactly as it contributes no space.
func PanelStatement(l PanelLayout, s PanelSubject) []string {
	items := panelItems(l, s)
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.String())
	}
	return out
}

// RenderCharacterPanel is the one production character-sheet component. Both
// the running Viewer and character generation receive the same model,
// statement order, font conversion, frame and native bounds from this call; a
// destination may move its pixels but may not restyle them.
func RenderCharacterPanel(l PanelLayout, f *text.Font, s PanelSubject) *image.RGBA {
	return composePanel(l, f, s)
}

// PanelLineReport is one row of a composed character panel exactly as the
// production layout placed it: the two runs of text it draws, the pen offsets
// it draws them at, and the row's own right edge relative to At.X.
//
// It exists so an install-gated witness can read what RenderCharacterPanel
// produced instead of recomputing a second layout beside it. A value the
// layout truncated to the empty string reads as the empty string here, and a
// row whose At.X+Width runs past the layout's own Size.X is a row the font
// clipped on the card. Every field is copied from the same panelLine
// composeItems paints.
type PanelLineReport struct {
	At     image.Point
	Label  string
	Value  string
	ValueX int
	Width  int

	// FullValue and FullRightValue are the same two values BEFORE
	// panelFitValue shortened them, so a caller can tell a cell the layout
	// truncated from a cell the subject never stated. A row with
	// Value == "" and FullValue != "" is a value the box had no room for at
	// all; a heading row states neither.
	FullValue      string
	FullRightValue string

	Right       bool
	RightLabel  string
	RightValue  string
	RightX      int
	RightValueX int

	// Wrap is the second row of a name too wide for one, drawn at WrapAt.
	Wrap   string
	WrapAt image.Point
}

// CharacterPanelReport is the layout RenderCharacterPanel would paint for the
// same arguments, as data. It resolves the rows with panelItems and lays them
// out with layoutLines, which is the pair composeItems itself calls and in the
// same order, so a caller reads the production result rather than a second
// derivation of it.
func CharacterPanelReport(l PanelLayout, f *text.Font, s PanelSubject) []PanelLineReport {
	if f == nil || f.Height() <= 0 {
		return nil
	}
	items := panelItems(l, s)
	full := make([][2]string, len(items))
	for i, it := range items {
		full[i] = [2]string{it.value, it.rightValue}
	}
	lines := layoutLines(l, f, items)
	out := make([]PanelLineReport, 0, len(lines))
	for i, ln := range lines {
		r := PanelLineReport{
			At: ln.at, Label: ln.label, Value: ln.value, ValueX: ln.valueX, Width: ln.width,
			Right: ln.right, RightLabel: ln.rightLabel, RightValue: ln.rightValue,
			RightX: ln.rightX, RightValueX: ln.rightValueX, Wrap: ln.wrap, WrapAt: ln.wrapAt,
		}
		if i < len(full) {
			r.FullValue, r.FullRightValue = full[i][0], full[i][1]
		}
		out = append(out, r)
	}
	return out
}

// PanelStatement resolves exactly what this Viewer would state for its current
// production selection. It is a read-only seam for headless release witnesses;
// selection and field adaptation remain panelSubject's ordinary path.
func (v *Viewer) PanelStatement() ([]string, bool) {
	if v == nil {
		return nil, false
	}
	s, ok := v.panelSubject()
	if !ok {
		return nil, false
	}
	return PanelStatement(v.panelLayout, s), true
}

// panelLines resolves the layout's rows against the subject and lays the
// survivors out.
func panelLines(l PanelLayout, f *text.Font, s PanelSubject) []panelLine {
	return layoutLines(l, f, panelItems(l, s))
}

// panelCellMetrics is one cell's shape against a font: the value's offset
// from the cell's own origin, for PLACEMENT, and how wide the cell measures,
// for fitting a box around it.
//
// The value is placed by the label's own PEN — where a following run of text
// starts — plus gap, and the cell is measured by the label's own BOX, and the
// two are different numbers on purpose: the box runs wider than the pen
// wherever a glyph's art overhangs its advance, so placing by it would open a
// hairline that varied with the last letter, and measuring by the pen alone
// would let a fitted box clip that overhang, since overhanging ink is still
// ink. An empty label spends no gap.
func panelCellMetrics(f *text.Font, gap int, label, value string) (valueX, width int) {
	labelW := 0
	if label != "" {
		valueX = f.Advance(label) + gap
		labelW, _ = f.Measure(label)
	}
	vw, _ := f.Measure(value)
	width = valueX + vw
	if labelW > width {
		width = labelW
	}
	return valueX, width
}

// panelFitValue truncates value, dropping trailing runes, until it ends at or
// before avail measured from the cell's own origin (label's pen plus gap,
// panelCellMetrics' own valueX) — down to the empty string if nothing of it
// fits beside its own label (1022 round 3, P fix). An empty value is returned
// unchanged: there is nothing to fit.
//
// THIS HAS NO ONE-CHARACTER FLOOR, unlike townShellTextLayout's own
// truncation of the member name and the DOLL/STATS label. Those two are a
// short authored string centred alone in its own rectangle, where a
// one-character remainder is still inside the box in every case that arises
// (DIV-191's own truncation ruling). A card VALUE sits beside an authored
// LABEL this function does not shorten, so the label alone can already reach
// avail on a row whose left column is wide (measured, 1022 round 3: a
// BLUDGEON/SHOOTING-width left column leaves single digits of room for the
// right column's own label) — a one-character floor there would still end up
// past the box, which is the defect this function exists to close, not a
// smaller instance of it. Nothing drawn is the box's own guarantee holding
// in every case, at the cost that an unrepresentable value draws blank
// rather than one unreadable digit.
//
// valueX depends only on label and gap, not on value, so it is measured once
// before the loop rather than recomputed on every shortened candidate.
func panelFitValue(f *text.Font, gap int, label, value string, avail int) string {
	if value == "" {
		return value
	}
	valueX, _ := panelCellMetrics(f, gap, label, value)
	for value != "" {
		vw, _ := f.Measure(value)
		if valueX+vw <= avail {
			return value
		}
		value = value[:len(value)-1]
	}
	return value
}

// panelFitLabelForValue shortens a label only when a fixed column cannot show
// both that install's caption and the complete value. The value is the main
// information on the owner's fixed 160-pixel card (DIV-191), so it gets the
// budget first; the remaining bytes of the localized caption occupy the space
// to its left. A heading with no value receives the whole budget. The resolved
// panel statement remains complete because layoutLines works on its own item
// copy and performs this only in the paint geometry tier.
func panelFitLabelForValue(f *text.Font, gap int, label, value string, avail int) string {
	if label == "" {
		return label
	}
	labelBudget := avail
	if value != "" {
		vw, _ := f.Measure(value)
		labelBudget -= vw + gap
	}
	if labelBudget < 0 {
		labelBudget = 0
	}
	for label != "" {
		lw, _ := f.Measure(label)
		if lw <= labelBudget {
			return label
		}
		label = label[:len(label)-1]
	}
	return label
}

// layoutLines places and measures already-resolved rows in order.
//
// A DROPPED ROW COSTS NO SPACE, and in flow that is a property of the walk
// rather than a correction applied afterwards: the flow index counts kept rows,
// so a row the resolution above dropped leaves no gap behind it.
//
// It walks the items twice: the first pass takes the widest left cell, so the
// right column's origin is one number for the whole box; the second places
// both cells.
func layoutLines(l PanelLayout, f *text.Font, items []panelItem) []panelLine {
	var cardItems []panelItem
	if l.CompactCard && l.Background != nil {
		cardItems = append([]panelItem(nil), items...)
	}
	lineH := f.Height()
	contentWidth := l.Size.X - l.Pad.X - max(l.Pad.X, l.RightInset)
	fixedColumnW := 0
	if l.FixedColumns && l.Size.X > 0 {
		fixedColumnW = (contentWidth - l.ColumnGap) / 2
		if fixedColumnW < 0 {
			fixedColumnW = 0
		}
	}

	// A fixed-size layout must not let a value run past its canvas. DIV-191
	// accepts truncation and requires the main information to fit, so a value
	// too wide for its column is shortened here, before the column origin is
	// measured. DIV-218
	if l.Size.X > 0 {
		for i := range items {
			budget := contentWidth
			if fixedColumnW > 0 && !items[i].center && !items[i].fullWidth {
				budget = fixedColumnW
				if items[i].preferValue {
					items[i].label = panelFitLabelForValue(f, l.LabelGap, items[i].label, items[i].value, budget)
				}
			}
			items[i].value = panelFitValue(f, l.LabelGap, items[i].label, items[i].value, budget)
		}
	}

	var leftEdge int
	if l.AlignValues && fixedColumnW > 0 {
		leftEdge = fixedColumnW
	} else if l.AlignValues {
		for _, it := range items {
			if it.center || it.value == "" {
				// A centred row neither takes nor sets the shared edge.
				continue
			}
			valueX, _ := panelCellMetrics(f, l.LabelGap, it.label, it.value)
			vw, _ := f.Measure(it.value)
			if e := valueX + vw; e > leftEdge {
				leftEdge = e
			}
		}
	}

	rightX := 0
	if fixedColumnW > 0 {
		rightX = fixedColumnW + l.ColumnGap
	} else {
		widestLeft := 0
		for _, it := range items {
			if !it.right {
				continue
			}
			_, w := panelCellMetrics(f, l.LabelGap, it.label, it.value)
			if l.AlignValues && !it.center && it.value != "" && leftEdge > w {
				w = leftEdge
			}
			w += it.indent
			if w > widestLeft {
				widestLeft = w
			}
		}
		rightX = widestLeft + l.ColumnGap
	}

	if l.Size.X > 0 {
		budget := l.Size.X - 2*l.Pad.X - rightX
		if fixedColumnW > 0 {
			budget = fixedColumnW
		}
		for i := range items {
			if !items[i].right {
				continue
			}
			if fixedColumnW > 0 && items[i].rightPreferValue {
				items[i].rightLabel = panelFitLabelForValue(f, l.LabelGap, items[i].rightLabel, items[i].rightValue, budget)
			}
			items[i].rightValue = panelFitValue(f, l.LabelGap, items[i].rightLabel, items[i].rightValue, budget)
		}
	}

	var rightEdge int
	if l.AlignValues && fixedColumnW > 0 {
		rightEdge = fixedColumnW
	} else if l.AlignValues {
		for _, it := range items {
			if it.center || !it.right || it.rightValue == "" {
				continue
			}
			rValueX, _ := panelCellMetrics(f, l.LabelGap, it.rightLabel, it.rightValue)
			rvw, _ := f.Measure(it.rightValue)
			if e := rValueX + rvw; e > rightEdge {
				rightEdge = e
			}
		}
	}

	out := make([]panelLine, 0, len(items))
	flowRow := 0
	for _, it := range items {
		valueX, width := panelCellMetrics(f, l.LabelGap, it.label, it.value)
		if l.AlignValues && !it.center && it.value != "" {
			vw, _ := f.Measure(it.value)
			edge := leftEdge
			if it.fullWidth {
				edge = contentWidth
			}
			// leftEdge is a MAXIMUM over every item's own natural edge, so
			// leftEdge-vw can never sit left of this item's own natural
			// valueX — the item that set leftEdge gets x == valueX exactly,
			// and every shorter one moves right, never overlapping its label.
			if x := edge - vw; x > valueX {
				valueX = x
				if x+vw > width {
					width = x + vw
				}
			}
		}
		ln := panelLine{field: it.field, label: it.label, value: it.value, valueX: valueX, width: width, leftIndent: it.indent}

		if it.right {
			rValueX, rWidth := panelCellMetrics(f, l.LabelGap, it.rightLabel, it.rightValue)
			if l.AlignValues && it.rightValue != "" {
				rvw, _ := f.Measure(it.rightValue)
				if x := rightEdge - rvw; x > rValueX {
					rValueX = x
					if x+rvw > rWidth {
						rWidth = x + rvw
					}
				}
			}
			ln.right, ln.rightLabel, ln.rightValue = true, it.rightLabel, it.rightValue
			ln.rightX, ln.rightValueX = rightX, rValueX
			if w := rightX + rWidth; w > ln.width {
				ln.width = w
			}
		}

		if l.Flow {
			flowRow += it.beforeRows
			ln.at = image.Pt(l.Pad.X, l.Pad.Y+flowRow*(lineH+l.Gap))
			flowRow++
		} else {
			ln.at = it.at
		}
		if it.center && l.Size.X > 0 {
			// A centred row is not in the aligned column, so ln.width is the
			// name's own measured width.
			x := (l.Size.X - ln.width) / 2
			if x < 0 {
				x = 0
			}
			ln.at.X = x
		}
		out = append(out, ln)
	}
	if l.CompactCard && l.Background != nil {
		fitCharacterCardLines(l, f, out, cardItems)
	}
	return out
}

// panelBox is the panel's size for these lines: the layout's own where it names
// one, and the content's extent plus a padding otherwise.
//
// ONE FORMULA SERVES BOTH ROW MODES. A fitted box is the furthest right and
// lowest any line reaches, plus the pad — which in flow reduces to the padding
// on both sides of the widest line and of the stack, and in a placed layout is
// the extent of wherever the author put things. That is what makes "a fitted
// box clips none of its rows" arithmetic rather than a promise.
//
// Each axis fits independently, so a layout may pin a width and let the height
// follow. The minimum applies to a FITTED width only: a named size is the
// author's and is not silently widened.
func panelBox(l PanelLayout, f *text.Font, lines []panelLine) image.Point {
	w, h := l.Size.X, l.Size.Y
	if w > 0 && h > 0 {
		return image.Pt(w, h)
	}
	lineH := f.Height()
	right, bottom := 0, 0
	for _, ln := range lines {
		if r := ln.at.X + ln.width; r > right {
			right = r
		}
		if b := ln.at.Y + lineH; b > bottom {
			bottom = b
		}
	}
	if w <= 0 {
		w = right + l.Pad.X
		if w < l.MinWidth {
			w = l.MinWidth
		}
	}
	if h <= 0 {
		h = bottom + l.Pad.Y
	}
	return image.Pt(w, h)
}

// composePanel paints the panel for one subject and returns it, or nil when
// there is nothing to paint: no font, a font holding no record, or a box with
// no area.
//
// NOTHING PAINTED LEAVES THE BOX, and no rectangle arithmetic of ours enforces
// that: the destination IS the box, so the font's own clip — which is its
// bounds test and not a per-glyph rejection — is what drops a row a layout
// placed outside. A guard here would be the one branch with no counterpart in
// the measurement beside it.
func composePanel(l PanelLayout, f *text.Font, s PanelSubject) *image.RGBA {
	return composeItems(l, f, panelItems(l, s))
}

// composeItems paints a box of already-resolved rows and returns it, or nil
// when there is nothing to paint: no font, a font holding no record, or a
// box with no area. It is composePanel with the subject taken out, and it is
// the whole of what both boxes share.
//
// A row's right cell, where layoutLines resolved one, is drawn at the shared
// right-column origin by the same label-then-value rule the left cell
// already uses. NOTHING PAINTED LEAVES THE BOX for it either, and for the
// same reason: the destination is the box, panelBox already covers rightX
// plus the right cell's own width, and the font's clip drops the rest.
func composeItems(l PanelLayout, f *text.Font, items []panelItem) *image.RGBA {
	if f == nil || f.Height() <= 0 {
		return nil
	}
	lines := layoutLines(l, f, items)
	box := panelBox(l, f, lines)
	if box.X <= 0 || box.Y <= 0 {
		return nil
	}

	img := image.NewRGBA(image.Rect(0, 0, box.X, box.Y))
	if l.Background != nil {
		drawPanelBackground(img, l.Background)
	} else {
		drawFrame(img, panelFrame(image.Rectangle{Max: box}, l.Fill, l.Border))
	}
	paint := img
	if l.CompactCard && l.Background != nil {
		paint = image.NewRGBA(img.Bounds())
	}
	first := text.CapturedLen()
	for _, ln := range lines {
		labelColor := l.LabelColor
		if l.CompactCard && (ln.field == PanelFieldSkillsHeading || ln.field == PanelFieldResistHeading) {
			labelColor = characterCardHeading
		}
		if ln.label != "" {
			f.Draw(paint, ln.label, ln.at.X+ln.leftIndent, ln.at.Y, labelColor)
		}
		valueColor := l.ValueColor
		if l.CompactCard && (ln.field == PanelFieldName || ln.field == PanelFieldRole) {
			valueColor = l.LabelColor
		}
		f.Draw(paint, ln.value, ln.at.X+ln.leftIndent+ln.valueX, ln.at.Y, valueColor)
		if ln.wrap != "" {
			f.Draw(paint, ln.wrap, ln.wrapAt.X, ln.wrapAt.Y, valueColor)
		}

		if ln.right {
			if ln.rightLabel != "" {
				f.Draw(paint, ln.rightLabel, ln.at.X+ln.rightX, ln.at.Y, labelColor)
			}
			f.Draw(paint, ln.rightValue, ln.at.X+ln.rightX+ln.rightValueX, ln.at.Y, l.ValueColor)
		}
	}
	if paint != img {
		// The text sits on the background once the layer goes down, so the
		// background is what lies under each glyph.
		text.Underlay(first, img)
		mask := image.NewAlpha(img.Bounds())
		for y := 0; y < box.Y; y++ {
			lo, hi := box.X, -1
			for x := 0; x < box.X; x++ {
				if characterCardWritable(img.RGBAAt(x, y)) {
					lo, hi = min(lo, x), max(hi, x)
				}
			}
			// The writing surface has shaded pixels within its outline. They
			// remain writable; matching only the flat color punches glyph holes.
			for x := lo; x <= hi; x++ {
				mask.SetAlpha(x, y, color.Alpha{255})
			}
		}
		draw.DrawMask(img, img.Bounds(), paint, image.Point{}, mask, image.Point{}, draw.Over)
	}
	return img
}

// drawPanelBackground copies src onto dst from dst's own origin, pixel for
// pixel and never scaled. A source larger than the box is clipped by the loop's
// own bounds; a smaller one leaves the rest of the box transparent, which is
// the author's business and not this function's to fill in.
func drawPanelBackground(dst, src *image.RGBA) {
	b, sb := dst.Bounds(), src.Bounds()
	for y := 0; y < b.Dy() && y < sb.Dy(); y++ {
		for x := 0; x < b.Dx() && x < sb.Dx(); x++ {
			dst.SetRGBA(b.Min.X+x, b.Min.Y+y, src.RGBAAt(sb.Min.X+x, sb.Min.Y+y))
		}
	}
}

// panelKey is everything the presented picture is a function of, as one
// comparable value: which unit, what it states, which layout and font, and
// how big the area it is placed in is.
//
// IT HOLDS THE SUBJECT AND NOT THE ENTITY. An entity's route, step, art and
// frame change on almost every tick and none of them is on the panel, so keying
// on one would rebuild the picture every frame a selected unit walked. The area
// is here because a fitted panel anchored to a corner moves when the window
// resizes, and the serial because a layout is not comparable.
type panelKey struct {
	subject PanelSubject
	serial  int
	area    image.Point

	statistics      bool
	figure          *image.RGBA
	packOpen        bool
	bookOpen        bool
	selectionStatus [3]string
}

// SetFont hands the viewer the game's own font. It is what turns the panel
// on: a viewer holding none draws no panel at all. It turns the notice on
// for the same reason and by the same gate: a viewer holding no font draws
// none, so the notice's own cached picture is dropped here beside the
// panel's — a font replaced under a composed box would otherwise leave the
// old atlas's pixels on screen.
func (v *Viewer) SetFont(f *text.Font) {
	v.font = f
	v.panelSerial++
	v.panelPic = nil
	v.noticeSerial++
	v.noticePic = nil
	v.messagePic = nil
}

// TextSelector names the current font alphabet; an absent font uses selector0.
func (v *Viewer) TextSelector() int {
	if v.font == nil {
		return 0
	}
	return v.font.Selector
}

func (v *Viewer) SetCardFont(f *text.Font) {
	v.panelCardFont = f
	v.panelSerial++
	v.panelPic = nil
}

// cardFont is the font the character panel actually composes with: the
// override when SetCardFont has set one, font otherwise — TownCharacterView's
// own cardFont() fallback (townshell.go), applied to the second card that
// shares its background art and row set.
func (v *Viewer) cardFont() *text.Font {
	if v.panelCardFont != nil {
		return v.panelCardFont
	}
	return v.font
}

// RenderPanel composes the panel for one subject and returns it as a plain
// image, or nil when there is nothing to compose.
//
// It is the exported face of the seam: given a layout, a font and a subject it
// is a function of its arguments, with no viewer, window or graphics context
// anywhere near it. A developer run against a lawful install writes its result
// to a PNG; a front-end reaches the same pixels through the viewer.
func RenderPanel(l PanelLayout, f *text.Font, s PanelSubject) *image.RGBA {
	return composePanel(l, f, s)
}

// SetPanelLayout replaces what the panel looks like, and nothing else about
// the viewer is a function of the value.
func (v *Viewer) SetPanelLayout(l PanelLayout) {
	v.panelLayout = l
	v.panelSerial++
	v.panelPic = nil
}

// panelSubject is what the panel states: the unit it describes — the first
// of the selection, in the selection's own order, among the entries this
// snapshot still holds alive or downed — and how many such entries there
// are.
//
// The filter is presentSelected — CALLED, not copied. The marks a frame draws,
// the orders a left click emits, the blows the two keys land and now the panel
// are the same units, and a fifth reading of "still there to act on" would be a
// fifth chance to disagree. It takes the first element rather than searching:
// that filter already emits in the selection's own ascending order, so "first"
// is that order's first and not a property of the walk.
//
// The count is THAT SLICE'S LENGTH and not a second walk of the selection. The
// number on screen and the unit described then answer to one reading of which
// units are present, so "4 selected" cannot be said over a set the described
// unit was not drawn from — and a dead or vanished id cannot be counted while
// being invisible in every other place the selection is read.
func (v *Viewer) panelSubject() (PanelSubject, bool) {
	if ref, ok := v.hoverInspection(); ok {
		return v.inspectionPanel(ref)
	}
	present := v.visiblePanelSelection(presentSelected(v.sel, v.entities))
	if len(present) == 0 {
		return PanelSubject{}, false
	}
	return v.panelSubjectFromPresent(present), true
}

// cardLevel is the level a unit card draws: the unit's own, or the full level
// while the display reveal is on. The reveal never writes the level back.
func (v *Viewer) cardLevel(e MapEntity) int {
	if v.fogReveal || !e.KnowledgeKnown {
		return cardLevelFull
	}
	return e.Knowledge
}

const cardLevelFull = 7

func (v *Viewer) panelSubjectFromPresent(present []MapEntity) PanelSubject {
	e := present[0]
	// The two groups are CARRIED ACROSS from the entry, not looked up here.
	// Everything the panel states about a unit arrives on the one seam that
	// says which units there are, so there is no second source for the picture
	// to be a function of and nothing for the two to disagree about.
	return PanelSubject{Kind: InspectionUnit, ID: e.ID, Name: e.Name, HP: e.HP, MaxHP: e.MaxHP,
		Mana: e.Mana, MaxMana: e.MaxMana, Cell: e.Cell,
		Selected: len(present), Combat: e.Combat, Char: e.Char, Speed: e.Speed,
		Weight: e.Load, WeightKnown: true, UnitNameIndex: e.UnitNameIndex,
		DetailLevel: v.cardLevel(e), DetailSet: true, Words: v.words, OriginalPanel: e.OriginalPanel,
		KnownSpells: e.KnownSpells}
}

// panelPresent is the picture to draw this frame and where its top-left corner
// goes, or false for a frame that draws no panel at all.
//
// It is the whole of the panel's frame logic and it needs no window, which is
// the only automated guard there is on where the panel lands. Draw performs one
// engine call on the result and decides nothing.
//
// The rebuild rule is the key comparison and nothing else. A frame whose key
// matches re-presents the picture it already holds; any difference
// recomposes, and the upload is dropped in the same statement. `TOWN-353`
// establishes that the mission screen and the shop screen carry the same
// object, so this composes through the same DrawTownCharacterRegion the town
// shell does and shares every rectangle with it (missionpane.go's own
// header).
//
// IT STILL DRAWS WITHOUT A SUBJECT, which is the one gate that moved. The
// card needed one to state anything at all; the pane is furniture —
// `SHOP-FIGURE -041`'s id-7 slot, present whenever the mission frame is —
// and a pane with nothing selected is the shipped body with no figure and no
// card on it, under the two zero-selection lines `TEXT-UI-047` gives. Where
// those lines stand is authored (`DIV-1795`). The font gate stays: this
// package draws no text at all without one. The returned origin moves left by
// the same amount, so the body still lands at rect.Min on screen and the seam
// paints immediately to its left, over the map view rather than displacing it.
func (v *Viewer) panelPresent() (*image.RGBA, image.Point, bool) {
	f := v.cardFont()
	if f == nil {
		return nil, image.Point{}, false
	}
	area := image.Pt(v.frameW, v.frameH)
	rect, ok := characterPanelBoxRect(area)
	if !ok {
		return nil, image.Point{}, false
	}
	view := v.characterPaneView(true)
	key := panelKey{subject: view.Subject, serial: v.panelSerial, area: area,
		statistics: view.Statistics, figure: view.Figure,
		packOpen: view.PackOpen, bookOpen: view.BookOpen,
		selectionStatus: view.SelectionStatus}
	if v.panelPic == nil || key != v.panelKey {
		// ONE STATEMENT, so a fresh picture can never be presented under a
		// texture holding the last one: what the upload reads and the flag
		// that says it must are written together or not at all.
		pic := image.NewRGBA(image.Rect(0, 0, rect.Dx()+characterPaneSeamW, rect.Dy()))
		v.panelText = text.Record(func() { DrawTownCharacterRegion(pic, view) })
		v.panelPic, v.panelFresh = pic, true
		v.panelKey = key
		v.panelBuilds++
	}
	if v.panelPic == nil {
		return nil, image.Point{}, false
	}
	text.Append(v.panelText, 0, 0)
	return v.panelPic, image.Pt(rect.Min.X-characterPaneSeamW, rect.Min.Y), true
}

// panelRect is where the character panel stands in FRAME PIXELS, and whether
// one stands anywhere at all.
func (v *Viewer) panelRect() (image.Rectangle, bool) {
	return characterPanelBoxRect(image.Pt(v.frameW, v.frameH))
}

// panelCaptures reports that the unit panel stands under the window pixel
// (x, y) — inventoryCaptures' own shape (inventory.go), applied to the fourth
// box in this package that has to stop a press reaching the map.
//
// IT IS NEW IN 0140 AND IT CLOSES A REAL HOLE. The panel has always been drawn
// opaque over the world and has never taken a press, which was survivable while
// it was a narrow box fitted to its own text. It was then pinned to sidebarWidth
// and anchored to the bottom right, 400 pixels wide and around 390 tall — and
// every pixel of it was a place where a left click walked the party to
// whatever cell happened to lie under the box. That is the same defect the
// inventory window's own hotfix names, at four times the area.
//
// WHAT IT DOES NOT DO IS LATCH. A gesture BEGUN on the panel and dragged onto
// the map still pans the camera, because the pan anchor is taken in step, which
// runs before this (viewer.go's own note on invGrab), and only invGrab is read
// there. Panning is recoverable and reversible; an order is not, which is why
// the press is stopped here and the drag is left disclosed rather than a second
// latch added for a box nothing inside is draggable.
func (v *Viewer) panelCaptures(x, y int) bool {
	r, ok := v.panelRect()
	return ok && image.Pt(x, y).In(r)
}

// panelOrigin is where the box's top-left corner lands in an area of the given
// size, for this layout's corner and margin.
//
// The arithmetic is exact and takes no clamp: a box too large for the area
// lands where the corner puts it and runs off the far edges, which is the
// clipping the window already does to everything else drawn on it. Shrinking or
// repositioning it would make the panel's place a function of the window size
// in a way nothing could state.
func panelOrigin(l PanelLayout, area, box image.Point) image.Point {
	x, y := l.Margin.X, l.Margin.Y
	if l.Corner == PanelTopRight || l.Corner == PanelBottomRight {
		x = area.X - box.X - l.Margin.X
	}
	if l.Corner == PanelBottomLeft || l.Corner == PanelBottomRight {
		y = area.Y - box.Y - l.Margin.Y
	}
	return image.Pt(x, y)
}
