package sav

import "fmt"

// The object programmes serve both the exact envelope index and narrower
// semantic projections. Raw members are never searched for object tags.
//
// The programmes below are the serializers, in file order, from
// knowledge/formats/sav/format.md — SAV-MEMBER-036 for the eleven bodies,
// SAV-HUMAN-043 for Humanoid's thirteen references, SAV-EMBED-039 for the eight
// embedded sites and their classes, SAV-WLIST-040 for the unnamed u16 list,
// SAV-SPELLBK-041 for the spellbook's skipped index, SAV-DIARY-042 for the two
// runtime arrays, SAV-SPELL-044 for the nine-byte Spell.
//
// SAV-MEMBER-036 IS NARROWED AND THE NARROWED CLAUSE IS THE ONE THIS FILE
// DEPENDS ON. Its published Human clause — "Unit plus 24 raw bytes" — is short
// by thirteen object references, and a walk built on it desynchronises eight
// bytes into the first Human of every save and never recovers. Humanoid below
// carries the thirteen.

// The fourteen u16 statistic words of a Unit record, in file order
// (SAV-UNITFLD-049). The names are that claim's reading, whose alignment is
// fixed by two constructor immediates rather than by plausibility: the
// constructor writes 100 to +0x98 and 50 to +0x9e, and the file reads exactly
// 100 and 50 at those two word positions on 100 of 100 Human records.
//
// +0x8e and +0x90 (StatOwnWeight, StatLoad) were the two of the fourteen this
// package once carried Unknown, named only by their object offsets. SAV-792
// identifies +0x8e as the actor's own carried weight and +0x90 as the load
// ITEM-LOAD-005 already derives from it; both are named here, and the wire
// lookup keys below stay "U8E"/"U90" because they select the same object
// members the corpus was decoded under. The load import path must not
// recompute StatLoad from the record's own container at read time — SAV-794
// shows the original does not, and a stored value an empty container could
// not have produced (e.g. own weight 178, stored load 181) is not corruption.
const (
	StatBody = iota
	StatReaction
	StatMind
	StatSpirit
	StatSpeed
	StatOwnWeight
	StatLoad
	StatCapacity
	StatHealth
	StatHealthMax
	StatHealthRegen
	StatMana
	StatManaMax
	StatManaRegen
	UnitStatWords
)

// statNames are the member names the fourteen words are read under, in the same
// order as the constants above.
var statNames = [UnitStatWords]string{
	"Body", "Reaction", "Mind", "Spirit", "Speed", "U8E", "U90", "Capacity",
	"Health", "HealthMax", "HealthRegen", "Mana", "ManaMax", "ManaRegen",
}

// stepOp is one construct of a serialization programme.
//
// A Member (class.go) can express a straight run of fixed-width fields and a
// CString, which is every construct the record-oriented decode needs. It cannot
// express a counted list, an object reference, a presence flag or an embedded
// object, and those four are why a Unit record has no length. The ops below are
// exactly the constructs Member cannot carry; where a class's own fields ARE a
// straight run, the programme reuses class.go's Member slices rather than
// restating them, so no field list exists twice in this package.
type stepOp uint8

const (
	// stpRun is a straight run of Members, read by the same widths class.go's
	// own decode reads them by.
	stpRun stepOp = iota
	// stpClass runs another class's whole programme in place. It is how a
	// derived class's record begins with its base's.
	stpClass
	// stpObjRef is one `ar << CObject*`.
	stpObjRef
	// stpList is a u32 count then that many object references.
	stpList
	// stpU16List is the unnamed class at vtable L08172 (SAV-WLIST-040): a
	// count then that many u16. It has no runtime descriptor and is
	// serialized directly through its vtable, so it never appears as a class
	// record and is never introduced by a tag.
	stpU16List
	// stpDWordArray and stpWordArray are CDWordArray and CWordArray: a count
	// then n*4 or n*2 raw bytes (SAV-DIARY-042).
	stpDWordArray
	stpWordArray
	// stpFlagged is a u8 presence flag; the sub-programme runs only when the
	// flag is 1.
	stpFlagged
	// stpContainer is the container serializer (ITEM-SAVE-014): the CObList
	// elements, then +0x1c, then +0x20.
	stpContainer
	// stpRefRun is n object references in a row, which is Humanoid's twelve
	// worn slots. The index the original skips is a fact about its own array,
	// not about the file: twelve references are written.
	stpRefRun
	// stpSpellbook is Spellbook::Serialize: u32 +0x18, u32 n, then n-1
	// references (SAV-SPELLBK-041).
	stpSpellbook
	// stpGroups is Player's tail: a u32 count then that many group records,
	// each serialized INLINE with no class record and no tag (SAV-OBJ-016).
	stpGroups
	// stpInline runs a named class's whole programme at the cursor as a
	// record of its own, with NO TAG AND NO INDEX.
	//
	// IT IS NOT stpClass AND IT IS NOT stpObjRef, and keeping the three apart
	// is the point. stpClass is inheritance: a derived class's record begins
	// with its base's, and the members merge into one record. stpObjRef is
	// `ar << CObject*`: the archive writes a tag, the object takes the next
	// index and a later back-reference can name it. This op is a virtual call
	// through the object's own vtable (slot +0x8), which writes the record's bytes and nothing else. A programme
	// that modelled it as either of the other two would be off by two bytes in
	// one direction or by an index in the other.
	stpInline
	// stpRawArray is a CArchive count followed by n-byte opaque elements.
	// SAV-CLASSSER-176 gives Outpost's eight-byte records; no element meaning
	// is inferred by the document walker.
	stpRawArray
)

// step is one element of a programme.
type step struct {
	op stepOp
	// name is the site's name, under which the walk files what it read.
	name string
	// class is stpClass's target.
	class string
	// n is stpRefRun's reference count.
	n int
	// members is stpRun's field list.
	members []Member
	// sub is stpFlagged's guarded programme.
	sub []step
}

// itemMembers is Item::Serialize past the head and the effect list
// (ITEM-SAVE-014).
//
// F40 IS THE APPEARANCE WORD AND IT IS THE ITEM'S WHOLE IDENTITY.
// ITEM-APPEAR-023 reads it as `(material << 12) | (slot << 8) | (shape << 5) |
// row`, assembled by one builder that five routines call with the same four
// arguments — where material is the material.reg index, row is the definition
// row index the head's own +0x0c carries, and the kind is the equipment slot.
// That is field for field the same sixteen-bit word data.ItemCode composes, so
// a consumer one tier up needs no mapping table to turn a saved item into one
// of this tree's: the file already holds the code.
var itemMembers = []Member{
	{Name: "F40", Kind: KindU16},
	{Name: "F42", Kind: KindU16},
	{Name: "F44", Kind: KindU8},
	{Name: "F45", Kind: KindU8},
	{Name: "F46", Kind: KindU8},
	{Name: "F48", Kind: KindU16},
	{Name: "F4A", Kind: KindU16},
	{Name: "F47", Kind: KindU8},
}

// spellMembers is Spell::Serialize (SAV-SPELL-044): nine bytes, and the record
// binds an identity key although the class is not a Token.
//
// MAGIC-SPELL-001 identifies S08 as spell ID, S09 as range, S0A as the
// defensive flag and S0C as mana cost. These are serialized instance values,
// not a reason to replace them with the definition row's defaults.
var spellMembers = []Member{
	{Name: "S08", Kind: KindU8},
	{Name: "S09", Kind: KindU8},
	{Name: "S0A", Kind: KindU8},
	{Name: "S0C", Kind: KindU16},
	{Name: "This", Kind: KindIdentity},
}

// unitStoreA is Unit::Serialize's store arm up to the two equipment
// references: 19 bytes.
var unitStoreA = []Member{
	{Name: "U49", Kind: KindU8},
	{Name: "U4A", Kind: KindU8},
	{Name: "U4B", Kind: KindU8},
	{Name: "U4C", Kind: KindU8},
	{Name: "U50", Kind: KindRaw, Len: 4},
	{Name: "U54", Kind: KindRaw, Len: 4},
	{Name: "U58", Kind: KindRaw, Len: 4},
	{Name: "U60", Kind: KindU8},
	{Name: "U61", Kind: KindU8},
	{Name: "U6C", Kind: KindU8},
}

// unitStoreB is the name, the fourteen statistic words and the run past them:
// 55 bytes plus the CString's own length byte and its text.
var unitStoreB = buildUnitStoreB()

func buildUnitStoreB() []Member {
	out := []Member{{Name: "Name", Kind: KindCString}}
	for _, n := range statNames {
		out = append(out, Member{Name: n, Kind: KindU16})
	}
	return append(out,
		Member{Name: "UA2", Kind: KindU8},
		Member{Name: "UA3", Kind: KindU8},
		Member{Name: "UA0", Kind: KindU16},
		Member{Name: "UA4", Kind: KindU16},
		Member{Name: "U12C", Kind: KindU8},
		Member{Name: "U130", Kind: KindU32},
		Member{Name: "U134", Kind: KindU8},
		Member{Name: "U135", Kind: KindU8},
		Member{Name: "U136", Kind: KindU8},
		Member{Name: "U138", Kind: KindU32},
		// The death stage. SAV-DEATH-051 grades it High as the field that
		// identifies a fully decayed corpse, where the published health
		// value does not: four of eight stage-5 corpses read -10007 and
		// four read -10001.
		Member{Name: "Stage", Kind: KindU8},
		Member{Name: "U148", Kind: KindU32},
		Member{Name: "U144", Kind: KindU32},
	)
}

// unitStoreC is the run past the two presence flags: 17 bytes.
var unitStoreC = []Member{
	{Name: "U5C", Kind: KindU32},
	{Name: "U64", Kind: KindU32},
	{Name: "U44", Kind: KindU32},
	{Name: "U40", Kind: KindU32},
	{Name: "U48", Kind: KindU8},
}

// programmes is every class this package can WALK, as opposed to decode as a
// straight run.
//
// THE FIXED PART OF Unit SUMS TO 603 AND THAT IS A CHECK, NOT A COINCIDENCE.
// SAV-UNITLEN-045 states the sum term by term — 37 head + 4 list count + 2 + 2
// + 462 raw + 2 + 19 + 1 + 55 + 1 + 1 + 17 — and the programme below reproduces
// it term for term in the same order. A programme that had a field at the wrong
// width, or a construct in the wrong place, would not.
var programmes = map[string][]step{
	"Token": {{op: stpRun, members: tokenMembers}},

	"Item": {
		{op: stpClass, class: "Token"},
		{op: stpList, name: "Effects"},
		{op: stpRun, members: itemMembers},
	},
	"Shield": {
		{op: stpClass, class: "Item"},
		{op: stpRun, members: []Member{{Name: "S50", Kind: KindRaw, Len: 22}}},
	},
	"Armor": {
		{op: stpClass, class: "Item"},
		{op: stpRun, members: []Member{
			{Name: "A52", Kind: KindRaw, Len: 22},
			{Name: "A50", Kind: KindU8},
		}},
	},
	"Weapon": {
		{op: stpClass, class: "Item"},
		{op: stpRun, members: []Member{
			{Name: "W52", Kind: KindRaw, Len: 24},
			{Name: "W6A", Kind: KindRaw, Len: 22},
			{Name: "W50", Kind: KindU8},
		}},
		{op: stpObjRef, name: "WeaponSpell"},
	},

	"Building": {
		{op: stpClass, class: "Token"},
		{op: stpRun, members: buildingMembers},
	},
	"Effect": {
		{op: stpClass, class: "Token"},
		{op: stpRun, members: effectMembers},
	},
	"Sack": {
		{op: stpClass, class: "Token"},
		{op: stpRun, members: []Member{{Name: "S3C", Kind: KindU32}}},
		{op: stpContainer, name: "Contents"},
	},
	// These programmes let a top-level document walk reach the later Sack
	// collection through every published Token-lineage graph. They decode
	// framing only; no spell or building state is installed by GroundSacks.
	// SAV-CLASSSER-173..176 establish each width and reference order.
	"VirtualCaster": {
		{op: stpClass, class: "Token"},
		{op: stpRun, members: []Member{{Name: "VC3C", Kind: KindU8}, {Name: "VC40", Kind: KindRaw, Len: 6}}},
	},
	"SpellEffect": {
		{op: stpClass, class: "Token"},
		{op: stpRun, members: []Member{{Name: "SE40", Kind: KindU8}, {Name: "SE41", Kind: KindU8}}},
	},
	"PointEffect": {
		{op: stpClass, class: "SpellEffect"},
		{op: stpObjRef, name: "PE48"},
		{op: stpRun, members: []Member{{Name: "PE44", Kind: KindReference}}},
	},
	"AreaEffect": {
		{op: stpClass, class: "SpellEffect"},
		{op: stpRun, members: []Member{{Name: "AE48", Kind: KindRaw, Len: 4}, {Name: "AE4C", Kind: KindU16}}},
		{op: stpObjRef, name: "AE44"},
	},
	"SpellTransport": {
		{op: stpClass, class: "SpellEffect"},
		{op: stpObjRef, name: "ST44"},
		{op: stpObjRef, name: "ST48"},
		{op: stpRun, members: []Member{{Name: "ST4C", Kind: KindU16}}},
	},
	"Effect_DirectDamage": {
		{op: stpClass, class: "Effect"},
		{op: stpRun, members: []Member{{Name: "EDD48", Kind: KindRaw, Len: 24}}},
	},
	"Outpost": {
		{op: stpClass, class: "Building"},
		{op: stpRun, members: []Member{{Name: "O84", Kind: KindU32}, {Name: "O88", Kind: KindU32}, {Name: "O80", Kind: KindU32}, {Name: "O8C", Kind: KindU32}}},
		{op: stpRawArray, name: "O6C", n: 8},
	},
	"Tavern": {
		{op: stpClass, class: "Building"},
		{op: stpRun, members: []Member{{Name: "T9C", Kind: KindU32}}},
	},
	"Shop": {
		{op: stpClass, class: "Building"},
		// SHOP-SAVE-015: the value cap, never shelf stock.
		{op: stpRun, members: []Member{{Name: "S70", Kind: KindU32}}},
	},

	"Diary": {
		{op: stpDWordArray, name: "Journal"},
		{op: stpWordArray, name: "JournalWords"},
		{op: stpRun, members: []Member{{Name: "D2C", Kind: KindReference}}},
	},
	"Spell":     {{op: stpRun, members: spellMembers}},
	"Spellbook": {{op: stpSpellbook, name: "Spells"}},

	// Player::Serialize HAS THREE TAIL CALLS, NOT TWO (SAV-PLDIARY-054). A
	// programme without it ends the record early — by 722 bytes on all 18
	// distinct corpus files, whose two arrays are 119 long — and reads the
	// CDWordArray's own count as whatever construct it expected next.
	"Player": {
		{op: stpRun, members: playerMembers},
		{op: stpGroups, name: "Groups"},
		{op: stpRun, members: []Member{{Name: "PRaw32", Kind: KindRaw, Len: 32}}},
		{op: stpInline, name: "Diary", class: "Diary"},
	},

	"Unit": {
		{op: stpClass, class: "Token"},
		{op: stpList, name: "Effects"},
		{op: stpU16List, name: "U15C"},
		{op: stpU16List, name: "U178"},
		{op: stpRun, name: "UnitBlocks", members: []Member{
			{Name: "UA6", Kind: KindRaw, Len: 24},
			{Name: "UBE", Kind: KindRaw, Len: 22},
			{Name: "U114", Kind: KindRaw, Len: 24},
			{Name: "UD4", Kind: KindRaw, Len: 64},
			{Name: "U154", Kind: KindRaw, Len: 180},
			{Name: "U158", Kind: KindRaw, Len: 148},
		}},
		{op: stpU16List, name: "U158_90"},
		{op: stpRun, name: "UnitControl", members: unitStoreA},
		{op: stpObjRef, name: "HeldWeapon"},
		{op: stpObjRef, name: "HeldShield"},
		{op: stpRun, name: "UnitState", members: unitStoreB},
		{op: stpObjRef, name: "U68"},
		{op: stpFlagged, name: "HasInventory", sub: []step{{op: stpContainer, name: "Inventory"}}},
		{op: stpFlagged, name: "HasSpellbook", sub: []step{
			{op: stpClass, class: "Spellbook"},
		}},
		{op: stpRun, members: unitStoreC},
	},

	"Humanoid": {
		{op: stpClass, class: "Unit"},
		{op: stpRun, name: "HumanoidXP", members: []Member{{Name: "H1CC", Kind: KindRaw, Len: 24}}},
		{op: stpRefRun, name: "Worn", n: 12},
		{op: stpObjRef, name: "Diary"},
	},
	"Human": {{op: stpClass, class: "Humanoid"}},
}

// groupProgramme is one group record, which is written INLINE inside a Player's
// tail with no class record and no tag of its own (SAV-OBJ-016), so it is not a
// class this package can be asked for by name.
var groupProgramme = []step{
	{op: stpU16List, name: "G20"},
	{op: stpRun, members: []Member{{Name: "G3C", Kind: KindRaw, Len: 80}}},
	{op: stpU16List, name: "G4C"},
	{op: stpList, name: "Actors"},
	{op: stpRun, members: []Member{
		{Name: "G1C", Kind: KindU32},
		{Name: "G40", Kind: KindU32},
		{Name: "G44", Kind: KindU32},
	}},
}

// Record is one object the walk read, with the objects it references filed
// under the site that referenced them.
type Record struct {
	// Class is the class the stream named it, Off the body offset of its
	// first byte and End the offset one past its last.
	Class    string
	Off, End int

	// Index is the object's place in the archive's shared counter, which is
	// what a back-reference names. IT IS 0 FOR A RECORD THE ARCHIVE DID NOT
	// NUMBER — one written through its owner's vtable rather than as
	// `ar << CObject*`, which is the Player's own Diary (SAV-PLDIARY-054). The
	// counter starts at 1, so 0 is a value no tagged object holds.
	Index uint16

	// Value, Text and Raw are the members the programme's straight runs read,
	// merged across the whole programme including the base classes'.
	Value map[string]uint32
	Text  map[string]string
	Raw   map[string][]byte

	// Refs are the objects this record referenced, by site name and in file
	// order. A null reference is not filed, so a site's slice is shorter than
	// its width whenever a slot was empty.
	Refs map[string][]*Record

	// RefSlots is the opt-in complete archive projection, including every nil
	// slot. Refs remains the historical non-null view. A serializer must never
	// reconstruct slot positions from Counts and that compact view.
	RefSlots map[string][]*Record

	// SpellSlots retains the sparse book's slot positions, including nulls.
	// Element zero is serialized slot 1; the original omits slot 0.
	// Refs["Spells"] remains the non-null projection for existing readers.
	SpellSlots []*Record
	// WornSlots keeps all twelve Humanoid armor references, including nulls.
	// The compact Refs["Worn"] view cannot identify an empty interior slot.
	WornSlots []*Record
	// Groups are inline records, not archive objects. The flat Actors view
	// remains available, but membership and trailing Group fields stay distinct.
	Groups []*Record

	// Counts are the element counts of the lists this record carries, by site
	// name. A count is kept separately from Refs because a list of six
	// references four of which are null is a different fact from a list of
	// four.
	Counts map[string]int

	// ContainerTailOff is the body offset of a stpContainer site's own +0x1c
	// dword, by site name (+0x20 immediately follows it). It exists because,
	// unlike a Member's fixed offset, a container's own tail sits after a
	// variable-length reference list this package does not otherwise retain
	// the end position of — the same problem SetSpellEffects' own refSpan
	// solves by recomputation; this field records the position once, at the
	// moment the walk is already there, instead of recomputing it.
	ContainerTailOff map[string]int

	// ListOff is the body offset of a stpU16List site's first element, by
	// site name. It is filled for every such site regardless of Raw
	// retention, so a writer can locate and patch the exact file bytes of a
	// named u16 list without re-deriving the walker's own layout arithmetic.
	// A record built by hand rather than by the walker leaves it empty.
	ListOff map[string]int

	// Serialized substructure starts, for independent raw-byte witnesses.
	// UnitState begins at its CString, not at any decoded statistic value.
	UnitBlocksOff, UnitControlOff, UnitStateOff, HumanoidXPOff int
	UnitRoutesOff                                              int
	// Structure starts only, for independent Player/Diary byte readers.
	GroupActorRefOffs []int
	DiaryRefOff       int
}

// newRecord is one empty record of a class, at an offset, under an archive
// index. An index of 0 means the archive did not number it.
func newRecord(class string, off int, index uint16) *Record {
	return &Record{
		Class:    class,
		Off:      off,
		Index:    index,
		Value:    make(map[string]uint32, recordValueHint(class)),
		Text:     map[string]string{},
		Raw:      map[string][]byte{},
		Refs:     map[string][]*Record{},
		RefSlots: map[string][]*Record{},
		Counts:   map[string]int{},

		ContainerTailOff: map[string]int{},
		ListOff:          map[string]int{},
	}
}

// value answers one member, or zero.
func (r *Record) value(name string) uint32 {
	if r == nil {
		return 0
	}
	return r.Value[name]
}

// The CArchive object-reference tags. A tag of 0 is the null arm, 0xffff
// introduces a new class, the high bit names a class already introduced, and
// anything else is a back-reference to an object already read (SAV-STREAM-013).
const (
	nullTag     = 0x0000
	newClassTag = 0xffff
)

// maxWalkDepth bounds the recursion. The deepest published nesting is Player →
// group → actor → item, so a bound well past it turns a reference cycle into an
// error rather than a stack overflow.
const maxWalkDepth = 16

// maxListElements bounds a counted list. A count read at the wrong offset is
// arbitrary, and an arbitrary count must not become an allocation.
const maxListElements = 1 << 16

// walker holds the archive state one walk needs: the position, the shared
// index counter, and what each index named.
type walker struct {
	b       []byte
	p       int
	next    uint16
	classes map[uint16]string
	objects map[uint16]*Record
	// retainGraph keeps every named array/header and nullable reference site
	// needed by the inverse programme. Ordinary narrow readers remain unchanged.
	retainGraph bool
}

// Walk reads the first non-null top-level Player, retaining the first-record
// owner convention. It does not select among multiple human participants.
func (f *File) Walk() (*Record, error) {
	if f == nil {
		return nil, fmt.Errorf("sav: party requires a save")
	}
	if f.Head.End < 0 || f.Head.End > len(f.Body) {
		return nil, fmt.Errorf("sav: Player list offset is outside the document")
	}
	w := &walker{b: f.Body, p: f.Head.End, next: 1, classes: map[uint16]string{}, objects: map[uint16]*Record{}}
	if f.Head.PlayerCount > maxListElements {
		return nil, fmt.Errorf("sav: Player count exceeds bound")
	}
	for i := uint32(0); i < f.Head.PlayerCount; i++ {
		rec, err := w.object(0)
		if err != nil {
			return rec, err
		}
		if rec == nil {
			continue
		}
		if rec.Class != "Player" {
			return rec, fmt.Errorf("sav: first non-null top-level record is %s, not Player", rec.Class)
		}
		return rec, nil
	}
	return nil, fmt.Errorf("sav: Player list has no non-null owner record")
}

// object reads one `ar << CObject*` at the cursor: the tag, then, on the two
// arms that carry data, the object's own record.
func (w *walker) object(depth int) (*Record, error) {
	if depth > maxWalkDepth {
		return nil, fmt.Errorf("sav: object nesting past %d at %d", maxWalkDepth, w.p)
	}
	tag, err := w.u16()
	if err != nil {
		return nil, err
	}
	switch {
	case tag == nullTag:
		return nil, nil
	case tag == newClassTag:
		name, err := w.classRecord()
		if err != nil {
			return nil, err
		}
		return w.body(name, depth)
	case tag&instanceTagBit != 0:
		name, ok := w.classes[tag&^instanceTagBit]
		if !ok {
			return nil, fmt.Errorf("sav: tag %#04x at %d names class index %d, which the "+
				"stream has not introduced", tag, w.p-2, tag&^instanceTagBit)
		}
		return w.body(name, depth)
	default:
		// A BACK-REFERENCE IS NOT A SECOND COPY. The tag names an object
		// already read, and the file carries nothing further, so the walk
		// hands back the record it already has.
		rec, ok := w.objects[tag]
		if !ok {
			return nil, fmt.Errorf("sav: back-reference to object %d at %d, which the walk "+
				"has not read", tag, w.p-2)
		}
		return rec, nil
	}
}

// classRecord reads a class introduction — the schema, the name length and the
// name — and takes the next index for it.
func (w *walker) classRecord() (string, error) {
	if w.p+4 > len(w.b) {
		return "", fmt.Errorf("sav: class introduction at %d overruns the stream", w.p)
	}
	if u16(w.b, w.p) != 1 {
		return "", fmt.Errorf("sav: class schema %d at %d is unsupported", u16(w.b, w.p), w.p)
	}
	w.p += 2
	n := int(u16(w.b, w.p))
	w.p += 2
	if n == 0 || n > maxClassName || w.p+n > len(w.b) {
		return "", fmt.Errorf("sav: class name of %d bytes at %d is not one", n, w.p)
	}
	name := string(w.b[w.p : w.p+n])
	w.p += n
	if w.next == 0 || w.next&instanceTagBit != 0 {
		return "", fmt.Errorf("sav: archive class index space exhausted at %d", w.p)
	}
	w.classes[w.next] = name
	w.next++
	return name, nil
}

// body reads one instance's own bytes, driven by its class's programme.
//
// The index is taken BEFORE the programme runs, because the objects the
// programme reaches take the indices after this one and a back-reference to
// this object must name it rather than its first child.
func (w *walker) body(class string, depth int) (*Record, error) {
	prog, ok := programmes[class]
	if !ok {
		return nil, fmt.Errorf("sav: class %s has no serialization programme, so the walk "+
			"cannot step over its record", class)
	}
	if w.next == 0 || w.next&instanceTagBit != 0 {
		return nil, fmt.Errorf("sav: archive object index space exhausted at %d", w.p)
	}
	rec := newRecord(class, w.p, w.next)
	w.objects[w.next] = rec
	w.next++
	if err := w.programme(prog, rec, depth); err != nil {
		return rec, fmt.Errorf("%s at %d: %w", class, rec.Off, err)
	}
	rec.End = w.p
	return rec, nil
}

// programme runs a class's steps in file order against rec.
func (w *walker) programme(prog []step, rec *Record, depth int) error {
	for _, s := range prog {
		if err := w.step(s, rec, depth); err != nil {
			return err
		}
	}
	return nil
}

func (w *walker) step(s step, rec *Record, depth int) error {
	switch s.op {
	case stpRun:
		if s.name == "UnitBlocks" {
			rec.UnitBlocksOff = w.p
		} else if s.name == "UnitControl" {
			rec.UnitControlOff = w.p
		} else if s.name == "UnitState" {
			rec.UnitStateOff = w.p
		} else if s.name == "HumanoidXP" {
			rec.HumanoidXPOff = w.p
		}
		return w.run(s.members, rec)
	case stpClass:
		prog, ok := programmes[s.class]
		if !ok {
			return fmt.Errorf("no programme for base class %s", s.class)
		}
		return w.programme(prog, rec, depth)
	case stpInline:
		prog, ok := programmes[s.class]
		if !ok {
			return fmt.Errorf("no programme for inline class %s", s.class)
		}
		// NO TAG IS READ AND NO INDEX IS TAKEN. The archive never saw this
		// object, so w.next must not move and w.objects must not gain an
		// entry: a later back-reference names the objects the archive
		// numbered, and inserting one here would shift every one of them.
		sub := newRecord(s.class, w.p, 0)
		if err := w.programme(prog, sub, depth); err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
		sub.End = w.p
		rec.Refs[s.name] = append(rec.Refs[s.name], sub)
		rec.Counts[s.name]++
		return nil
	case stpObjRef:
		return w.ref(s.name, rec, depth)
	case stpRefRun:
		for i := 0; i < s.n; i++ {
			obj, err := w.object(depth + 1)
			if err != nil {
				return err
			}
			if s.name == "Worn" {
				rec.WornSlots = append(rec.WornSlots, obj)
			}
			if w.retainGraph {
				rec.RefSlots[s.name] = append(rec.RefSlots[s.name], obj)
			}
			if obj != nil {
				rec.Refs[s.name] = append(rec.Refs[s.name], obj)
			}
		}
		rec.Counts[s.name] += s.n
		return nil
	case stpList:
		n, err := w.u32()
		if err != nil {
			return err
		}
		if n > maxListElements {
			return fmt.Errorf("%s declares %d elements", s.name, n)
		}
		rec.Counts[s.name] += int(n)
		for i := uint32(0); i < n; i++ {
			if err := w.ref(s.name, rec, depth); err != nil {
				return err
			}
		}
		return nil
	case stpU16List:
		if s.name == "U15C" {
			rec.UnitRoutesOff = w.p
		}
		n, err := w.count()
		if err != nil {
			return err
		}
		rec.Counts[s.name] += n
		start := w.p
		if err := w.skip(s.name, 2*n); err != nil {
			return err
		}
		rec.ListOff[s.name] = start
		// U15C/U178 are the Unit mover's StaticRoute/DynamicRoute lists
		// (SAV-630, MOVE-ROUTE-004); a route consumer needs their file bytes
		// through the same non-retaining walker ActorGraph uses, the same
		// way U158_90's Patrol already does.
		if rec.Class == "Group" || s.name == "U158_90" || s.name == "U15C" || s.name == "U178" || w.retainGraph {
			rec.Raw[s.name] = append([]byte(nil), w.b[start:w.p]...)
		}
		return nil
	case stpDWordArray:
		n, err := w.count()
		if err != nil {
			return err
		}
		rec.Counts[s.name] += n
		return w.retainArray(rec, s.name, 4*n)
	case stpWordArray:
		n, err := w.count()
		if err != nil {
			return err
		}
		rec.Counts[s.name] += n
		return w.retainArray(rec, s.name, 2*n)
	case stpRawArray:
		n, err := w.count()
		if err != nil {
			return err
		}
		rec.Counts[s.name] += n
		start := w.p
		if err := w.skip(s.name, s.n*n); err != nil {
			return err
		}
		rec.Raw[s.name] = append([]byte(nil), w.b[start:w.p]...)
		return nil
	case stpFlagged:
		if w.p >= len(w.b) {
			if s.name == "HasSpellbook" {
				return fmt.Errorf("%w: missing presence flag", ErrSpellbook)
			}
			return fmt.Errorf("%s has no presence flag", s.name)
		}
		flag := w.b[w.p]
		w.p++
		rec.Value[s.name] = uint32(flag)
		if flag == 0 {
			return nil
		}
		if flag != 1 {
			if s.name == "HasSpellbook" {
				return fmt.Errorf("%w: presence flag is %d", ErrSpellbook, flag)
			}
			return fmt.Errorf("%s presence flag is %d, not 0 or 1", s.name, flag)
		}
		return w.programme(s.sub, rec, depth)
	case stpContainer:
		n, err := w.u32()
		if err != nil {
			return err
		}
		if n > maxListElements {
			return fmt.Errorf("%s declares %d elements", s.name, n)
		}
		rec.Counts[s.name] += int(n)
		for i := uint32(0); i < n; i++ {
			if err := w.ref(s.name, rec, depth); err != nil {
				return err
			}
		}
		// The insert index and the load are STORED, not recomputed
		// (ITEM-SAVE-014), so they are two dwords in the file whether or not
		// the list was empty. The tail's own start is recorded once, here,
		// while the walk already knows it.
		rec.ContainerTailOff[s.name] = w.p
		for _, suffix := range []string{"1C", "20"} {
			v, err := w.u32()
			if err != nil {
				return err
			}
			rec.Value[s.name+suffix] = v
		}
		return nil
	case stpSpellbook:
		header, err := w.u32()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSpellbook, err)
		}
		if w.retainGraph {
			rec.Value[s.name+"Header"] = header
		}
		n, err := w.u32()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSpellbook, err)
		}
		if n > maxListElements {
			return fmt.Errorf("%w: declares %d elements", ErrSpellbook, n)
		}
		// INDEX 0 IS SKIPPED IN BOTH DIRECTIONS (SAV-SPELLBK-041), so a
		// record carries n and n-1 references. A count of zero writes none.
		rec.Counts[s.name] += int(n)
		for i := uint32(1); i < n; i++ {
			obj, err := w.object(depth + 1)
			if err != nil {
				return fmt.Errorf("%w: slot %d: %v", ErrSpellbook, i, err)
			}
			rec.SpellSlots = append(rec.SpellSlots, obj)
			if w.retainGraph {
				rec.RefSlots[s.name] = append(rec.RefSlots[s.name], obj)
			}
			if obj != nil {
				rec.Refs[s.name] = append(rec.Refs[s.name], obj)
			}
		}
		return nil
	case stpGroups:
		n, err := w.u32()
		if err != nil {
			return err
		}
		if n > maxListElements {
			return fmt.Errorf("%s declares %d groups", s.name, n)
		}
		rec.Counts[s.name] += int(n)
		for i := uint32(0); i < n; i++ {
			group := newRecord("Group", w.p, 0)
			if err := w.programme(groupProgramme, group, depth); err != nil {
				return fmt.Errorf("group %d: %w", i, err)
			}
			group.End = w.p
			rec.Groups = append(rec.Groups, group)
			rec.Refs["Actors"] = append(rec.Refs["Actors"], group.Refs["Actors"]...)
			rec.Counts["Actors"] += group.Counts["Actors"]
		}
		return nil
	}
	return fmt.Errorf("sav: programme step %d is not an op this walk runs", s.op)
}

// ref reads one object reference and files what it found under name. A null
// reference is counted by the site's own count and files nothing, because a
// record filed for it would be a record with no class.
func (w *walker) ref(name string, rec *Record, depth int) error {
	if rec.Class == "Group" && name == "Actors" {
		rec.GroupActorRefOffs = append(rec.GroupActorRefOffs, w.p)
	}
	if (rec.Class == "Human" || rec.Class == "Humanoid") && name == "Diary" {
		rec.DiaryRefOff = w.p
	}
	obj, err := w.object(depth + 1)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if obj != nil {
		rec.Refs[name] = append(rec.Refs[name], obj)
	}
	// Group and Pack registries also need positional null members. Compact
	// Refs remains unchanged for older actor/count consumers.
	if w.retainGraph || name == "Inventory" || rec.Class == "Group" && name == "Actors" {
		rec.RefSlots[name] = append(rec.RefSlots[name], obj)
	}
	return nil
}

func (w *walker) retainArray(rec *Record, name string, size int) error {
	start := w.p
	if err := w.skip(name, size); err != nil {
		return err
	}
	if w.retainGraph || rec.Class == "Diary" {
		rec.Raw[name] = append([]byte(nil), w.b[start:w.p]...)
	}
	return nil
}

// run reads a straight run of Members at the cursor, into rec.
func (w *walker) run(members []Member, rec *Record) error {
	for _, m := range members {
		if m.Kind == KindCString {
			if w.p >= len(w.b) {
				return fmt.Errorf("%s has no length byte", m.Name)
			}
			n := int(w.b[w.p])
			if n == 0xff {
				return fmt.Errorf("%s: extended or Unicode CString is unsupported", m.Name)
			}
			if w.p+1+n > len(w.b) {
				return fmt.Errorf("%s of %d bytes overruns the stream", m.Name, n)
			}
			rec.Text[m.Name] = string(w.b[w.p+1 : w.p+1+n])
			w.p += 1 + n
			continue
		}
		width, ok := m.Kind.width(m.Len)
		if !ok || width <= 0 {
			return fmt.Errorf("%s has no width", m.Name)
		}
		if w.p+width > len(w.b) {
			return fmt.Errorf("%s overruns the stream", m.Name)
		}
		switch m.Kind {
		case KindU8:
			rec.Value[m.Name] = uint32(w.b[w.p])
		case KindU16, KindU16Saturated:
			rec.Value[m.Name] = uint32(u16(w.b, w.p))
		case KindU32, KindIdentity, KindReference:
			rec.Value[m.Name] = u32(w.b, w.p)
		case KindU32Obfuscated:
			rec.Value[m.Name] = u32(w.b, w.p) ^ obfuscator
		case KindRaw:
			rec.Raw[m.Name] = w.b[w.p : w.p+width]
		}
		w.p += width
	}
	return nil
}

// count reads a CArchive count: a u16, or 0xffff followed by a u32
// (SAV-WLIST-040).
func (w *walker) count() (int, error) {
	n, err := w.u16()
	if err != nil {
		return 0, err
	}
	if n != 0xffff {
		return int(n), nil
	}
	big, err := w.u32()
	if err != nil {
		return 0, err
	}
	if big > maxListElements {
		return 0, fmt.Errorf("sav: a list of %d elements at %d", big, w.p)
	}
	return int(big), nil
}

func (w *walker) u16() (uint16, error) {
	if w.p+2 > len(w.b) {
		return 0, fmt.Errorf("sav: a word at %d is past the %d-byte stream", w.p, len(w.b))
	}
	v := u16(w.b, w.p)
	w.p += 2
	return v, nil
}

func (w *walker) u32() (uint32, error) {
	if w.p+4 > len(w.b) {
		return 0, fmt.Errorf("sav: a dword at %d is past the %d-byte stream", w.p, len(w.b))
	}
	v := u32(w.b, w.p)
	w.p += 4
	return v, nil
}

func (w *walker) skip(what string, n int) error {
	if n < 0 || w.p+n > len(w.b) {
		return fmt.Errorf("%s of %d bytes at %d overruns the stream", what, n, w.p)
	}
	w.p += n
	return nil
}
