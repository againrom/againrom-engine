package sav

import "fmt"

// A CLASS LAYOUT IS A TABLE THIS PACKAGE'S DECODER IS DRIVEN BY, and not a hole
// an opaque byte slice sits in.
//
// The object stream is a Microsoft CArchive stream and names its own classes.
// Each writes its members through its own Serialize, and this file is those
// programmes: a class is a name, whether its record opens with the Token head,
// the members past that head IN FILE ORDER, and how its length is determined.
//
// A class whose programme names an object of a class nobody has read carries no
// members, and its bytes are carried VERBATIM. That fallback is what makes the
// round trip byte-exact — but it is the fallback and not the mechanism. Filling
// a Members slice is the whole of what it takes to decode a class: nothing in
// the reader, the writer or the round trip is written around a class being
// opaque.

// Kind is how one member is read and written. The two arithmetic kinds are here
// rather than at the call site because they are properties of the FIELD: a
// caller that had to remember to apply them is a caller that will forget, and
// forgetting the first one shows about 1.54 billion gold.
type Kind uint8

const (
	// KindU8, KindU16 and KindU32 are plain little-endian integers.
	KindU8 Kind = iota
	KindU16
	KindU32

	// KindCString is the MFC short form: a length byte then that many bytes
	// of 8-bit text.
	KindCString

	// KindRaw is Len bytes carried and not interpreted.
	KindRaw

	// KindU32Obfuscated is a dword stored XOR 0x5c073f4d. The constant is an
	// INVOLUTION, so read and write are the same operation.
	//
	// IT IS PLAYER'S ALONE and must not spread: over the eleven Serialize
	// bodies and every sub-serializer they call, the transform occurs at four
	// sites and all four are inside Player::Serialize. For every other class
	// the bytes in the file are the bytes in memory. That census is graded
	// MEDIUM by the round that made it and names its own blindness — it
	// cannot see a field already held obfuscated in memory, because it read
	// the writers of the record rather than the writers of each field.
	KindU32Obfuscated

	// KindU16Saturated is a dword in memory written as min(v, 0x7fff). The
	// clamp is the original's and is reproduced rather than corrected: a
	// larger value does not survive a save and nothing says so. It is
	// Player's alone for the same reason, and at the same grade.
	KindU16Saturated

	// KindIdentity is the object's OWN ADDRESS in the writing process,
	// written verbatim, and KindReference is another object's. They are one
	// identity map: the load arm binds old-address to new-object as it reads
	// the identity. The missing-key rule belongs to the specific reference
	// consumer, not to this numeric field kind. Token's named resolver nulls
	// a miss; stage-zero order repair preserves a missing raw key
	// (SAV-PTRMAP-035, SAV-HUMRESUME-460, SAV-ACTORINPUT-547).
	//
	// They are distinct kinds rather than plain dwords because a writer must
	// treat them as identities: a save authored from nothing must mint a
	// unique non-zero key per object and write every reference as the key of
	// its target. The numeric value is arbitrary; the identity is not.
	KindIdentity
	KindReference
)

// width answers how many bytes a member of this kind occupies in the file, and
// whether that width is fixed. A KindCString's width depends on its own length
// byte and is answered by the decoder instead.
func (k Kind) width(raw int) (int, bool) {
	switch k {
	case KindU8:
		return 1, true
	case KindU16, KindU16Saturated:
		return 2, true
	case KindU32, KindU32Obfuscated, KindIdentity, KindReference:
		return 4, true
	case KindRaw:
		return raw, true
	}
	return 0, false
}

// Member is one field of a class's serialized record.
type Member struct {
	// Name is what this package calls the field. Where the original's own
	// object offset is the only name anybody has for it, that offset is the
	// name — "B40" is the byte at Building+0x40 — so a later decode can
	// rename it without anybody having to work out which field was meant.
	Name string

	// Kind is how it is read and written.
	Kind Kind

	// Len is the byte count for KindRaw and is ignored otherwise.
	Len int
}

// How a class's record length is determined.
const (
	// ExtentComputable is a class whose programme is complete but whose
	// length depends on its content — a counted list, a string, a presence
	// flag. Such a record can be walked; it cannot be tabled.
	ExtentComputable = -1

	// ExtentUnread is a class whose programme names an embedded object of a
	// class NOBODY HAS READ. Such a record cannot be walked at all, and this
	// is the state that keeps a save from being authored from nothing.
	ExtentUnread = -2
)

// Class is one serialized class.
type Class struct {
	// Name is the class name the stream introduces it by.
	Name string

	// Head reports that the record opens with the 37-byte Token head.
	Head bool

	// Members are the fields past the head, in file order. An empty slice is
	// a class whose members are not decoded, NOT a class with no members.
	Members []Member

	// Extent is the record's byte length past the head where that is a
	// constant, or ExtentComputable or ExtentUnread.
	Extent int

	// Consecutive reports that instances of this class are written one after
	// another at the top level of the stream, so that stepping the record
	// length lands on the next instance's tag.
	//
	// IT IS A SEPARATE FACT FROM HAVING A FIXED LENGTH, and finding that out
	// cost a round. `Effect` is exactly 44 bytes and still cannot be chained,
	// because an Effect is an ELEMENT OF ANOTHER OBJECT'S COUNTED LIST: the
	// walk lands back inside the enclosing record, not on a tag. `Building`
	// chains precisely because buildings are consecutive. A table with only
	// a length would make those two look alike.
	Consecutive bool

	// Base is the class this one's record begins with past the head, where
	// its Serialize calls another's. A Weapon record is an Item record and
	// then Weapon's own members; a Human record is a Unit record and then 24
	// raw bytes and nothing else.
	Base string
}

// Fixed reports that a record of this class is always the same length, and what
// that length is. It is the only case a walk can step over without decoding, and
// it is why Building is the class this package can chain.
func (c Class) Fixed() (int, bool) {
	if c.Extent < 0 || c.Base != "" {
		return 0, false
	}
	n := c.Extent
	if c.Head {
		n += TokenLen
	}
	return n, true
}

// TokenLen is the placeable-object head: ONE ROUTINE, Token::Serialize, and 37
// bytes rather than the 16 a corpus reading measured.
//
// The corpus reading was not wrong about its fields, it was wrong about their
// OWNER: the head's first twelve bytes are a raw copy out of a separate object
// the token points at, and the cell, the fine position and the state word all
// live inside that block. So their offsets survive unchanged, which is why the
// scan in actors.go did not have to move.
const TokenLen = 37

// tokenMembers is Token::Serialize in file order.
var tokenMembers = []Member{
	// The twelve bytes copied out of *(this+0x10). SAV-OBJ-014's cellA,
	// cellB, fineX, fineY, the unnamed word and the state dword are all
	// inside here; the object they belong to has not been identified, so
	// this package reads them by their positions in the block (see Actor)
	// and does not claim to know what the block is.
	{Name: "Block12", Kind: KindRaw, Len: 12},
	{Name: "RuntimeID", Kind: KindU32},
	{Name: "T0C", Kind: KindU8},
	{Name: "T0E", Kind: KindU16},
	// Whose LOW u16 is the map's own unit id — the same offset, head+0x13,
	// the corpus reading already used.
	{Name: "T08", Kind: KindU32},
	{Name: "T18", Kind: KindU16},
	{Name: "T1C", Kind: KindU32},
	{Name: "Identity", Kind: KindIdentity},
	{Name: "Reference", Kind: KindReference},
}

// playerMembers is the Player record: seventeen fields in one straight run,
// past a record that has no Token head.
//
// This slice IS the decoder. Nothing else in this package knows where a
// player's money sits. What follows it in the file — a counted list of group
// records, 32 raw bytes and an inline Diary (SAV-PLDIARY-054) — is NOT here,
// because a group record embeds an object whose extent varies with its content;
// see Classes["Player"].Extent and the Player programme in program.go.
var playerMembers = []Member{
	{Name: "Name", Kind: KindCString},
	{Name: "Slot", Kind: KindU16},
	{Name: "SlotAgain", Kind: KindU32},
	{Name: "Raw10", Kind: KindRaw, Len: 8},
	{Name: "F44", Kind: KindU8},
	{Name: "Participant", Kind: KindU32},
	{Name: "F2C", Kind: KindU16},
	{Name: "Money", Kind: KindU32Obfuscated},
	{Name: "Outcome", Kind: KindU8},
	{Name: "F3D", Kind: KindU8},
	{Name: "F48", Kind: KindU32Obfuscated},
	{Name: "F50", Kind: KindU32},
	{Name: "F54", Kind: KindU16Saturated},
	{Name: "F4C", Kind: KindU16Saturated},
	{Name: "F58", Kind: KindU32},
	// THE PARTICIPANT'S OWN STARTING CHARACTER, at Player+0x34
	// (SAV-HERO-059). It is an identity key: written at L08350 through the
	// u32 primitive, read at L08351 and then resolved through the identity
	// map by L08352, so its value is a POINTER IN THE ORIGINAL'S OWN ADDRESS
	// SPACE and means nothing except as a key. It equals the key of exactly one
	// actor in this Player's own groups on 18 of 18 distinct saves.
	//
	// It is the field, not the id, that says who the player's hero is. The
	// rival reading — that it names the last actor added — is excluded by
	// value on game0007.sav, where it names an actor that is neither first nor
	// last in file order.
	{Name: "Hero", Kind: KindU32},
	{Name: "This", Kind: KindIdentity},
}

// buildingMembers is Building::Serialize past the Token head: 40 bytes, no
// count, no string and no branch. It is the ONE class in this table whose whole
// record this package both decodes and re-emits from the decoded form.
var buildingMembers = []Member{
	{Name: "B52", Kind: KindRaw, Len: 22},
	{Name: "B40", Kind: KindU8},
	{Name: "B42", Kind: KindU16},
	{Name: "B44", Kind: KindU16},
	{Name: "B46", Kind: KindU16},
	{Name: "B48", Kind: KindU8},
	{Name: "B60", Kind: KindU8},
	{Name: "B61", Kind: KindU8},
	{Name: "B64", Kind: KindU32},
	{Name: "B68", Kind: KindU32},
}

// effectMembers is Effect::Serialize past the head: seven bytes, no variable
// part.
var effectMembers = []Member{
	{Name: "E3C", Kind: KindU8},
	{Name: "E3D", Kind: KindU8},
	{Name: "E40", Kind: KindU32},
	{Name: "E0C", Kind: KindU8},
}

// Classes is every class this package has a programme or a name for.
//
// THE SET OF CLASSES IS NOT THE SET A CORPUS SHOWS. Four saves introduce eleven;
// the image carries TWENTY-EIGHT serializable classes, and one save already in
// the corpus introduces a twelfth, Spell. Seventeen of the twenty-eight have
// unread Serialize bodies. A reader that knew only the eleven would fail on any
// save carrying a spellbook and, by construction, on any save from a mission
// with a shop, a tavern or an outpost — so the names are all here, and Lookup
// degrades on a name that is not.
var Classes = map[string]Class{
	// Read, decoded here, and written CONSECUTIVELY, which is what makes it
	// the one class this package walks.
	"Building": {Name: "Building", Head: true, Members: buildingMembers, Extent: 40,
		Consecutive: true},
	// Read and decoded, 44 bytes total — and NOT consecutive. An Effect is an
	// element of the `+0x20` counted list of the object it is attached to, so
	// stepping 44 lands back inside the enclosing record and never on a tag.
	// The length is right and the chain is impossible; those are two facts and
	// this row now carries both.
	"Effect": {Name: "Effect", Head: true, Members: effectMembers, Extent: 7},
	// Read, and decoded here as far as its own straight run goes. Its tail —
	// a counted list of group records and 32 raw bytes — is not, because a
	// group record embeds an object at +0x4c whose class is unread.
	"Player": {Name: "Player", Members: playerMembers, Extent: ExtentUnread},

	// Read, programme complete, length depends on content: a counted list of
	// object references, a string, or a presence flag.
	"Item":   {Name: "Item", Head: true, Extent: ExtentComputable},
	"Shield": {Name: "Shield", Head: true, Base: "Item", Extent: 22},
	"Armor":  {Name: "Armor", Head: true, Base: "Item", Extent: 23},
	"Weapon": {Name: "Weapon", Head: true, Base: "Item", Extent: ExtentComputable},
	"Sack":   {Name: "Sack", Head: true, Extent: ExtentComputable},

	// PROGRAMME COMPLETE, NO CONSTANT. All eight embedded sites are now
	// resolved, and what that established is that `Unit` HAS NO LENGTH — not
	// that one is still missing. Its programme holds four counted lists, a
	// CString, three object references and two presence flags; the fixed part
	// sums to 603 and the floor is 609 + the name's length. Anything here that
	// wanted a constant for `Unit` would be the thing to change, so there is
	// none.
	"Unit": {Name: "Unit", Head: true, Extent: ExtentComputable},

	// `Humanoid` IS A CLASS OF ITS OWN AND IT IS WHERE THE BYTES ARE. An
	// earlier round read eleven Serialize bodies and attributed this one's
	// output to `Human`; this is the twelfth. It writes `Unit`, then 24 raw
	// bytes from +0x1cc, then THIRTEEN OBJECT REFERENCES — twelve from an
	// array for indices 1..12 with index 0 skipped, then one more from +0x1e4.
	//
	// THE OLD READING WAS WRONG AND NOT MERELY SHORT. A walk built on
	// "`Unit` + 24" desynchronises eight bytes into the first `Human` of a
	// save and never recovers, and a writer that believed it emits a record
	// the game's own loader reads twenty-six bytes short. This package carried
	// that reading in `Human`'s row; it does not now.
	"Humanoid": {Name: "Humanoid", Head: true, Base: "Unit", Extent: ExtentComputable},
	// And `Human` itself writes NOTHING of its own — its store arm is empty.
	"Human": {Name: "Human", Head: true, Base: "Humanoid", Extent: 0},

	// Programme complete, no constant: two embedded arrays and a reference.
	"Diary": {Name: "Diary", Extent: ExtentComputable},

	// Named by the image's own class descriptors, Serialize unread. They are
	// here so that meeting one is a known class with no programme rather than
	// an unknown name, which are different problems for a caller.
	"Spell": {Name: "Spell", Extent: ExtentComputable},
	// Reached from `*(Unit+0x140)` behind a presence flag, and it emits its
	// own references with the same skipped-index idiom `Humanoid` uses.
	"Spellbook": {Name: "Spellbook", Extent: ExtentComputable},
	// The two array classes the Diary embeds.
	"CDWordArray":   {Name: "CDWordArray", Extent: ExtentComputable},
	"CWordArray":    {Name: "CWordArray", Extent: ExtentComputable},
	"TableLine":     {Name: "TableLine", Extent: ExtentUnread},
	"Token":         {Name: "Token", Head: true, Extent: 0},
	"VirtualCaster": {Name: "VirtualCaster", Head: true, Extent: ExtentUnread},
	"SpellEffect":   {Name: "SpellEffect", Head: true, Extent: ExtentUnread},
	"PointEffect":   {Name: "PointEffect", Head: true, Base: "SpellEffect", Extent: ExtentUnread},
	"AreaEffect":    {Name: "AreaEffect", Head: true, Base: "SpellEffect", Extent: ExtentUnread},
	"SpellTransport": {Name: "SpellTransport", Head: true, Base: "SpellEffect",
		Extent: ExtentUnread},
	"Outpost": {Name: "Outpost", Head: true, Base: "Building", Extent: ExtentUnread},
	"Tavern":  {Name: "Tavern", Head: true, Base: "Building", Extent: ExtentUnread},
	"Shop":    {Name: "Shop", Head: true, Base: "Building", Extent: ExtentUnread},
}

// Lookup answers the class a stream names, and DEGRADES rather than refusing.
//
// A name this table does not carry is answered as a class with no programme and
// an unread extent, so a reader meets "a class whose record I cannot walk" —
// which it already handles — instead of "a name I have never heard of", which
// would be a second failure mode for the same situation. Twenty-eight names are
// known and seventeen of them have no programme; a twenty-ninth would be no
// different in kind.
func Lookup(name string) Class {
	if c, ok := Classes[name]; ok {
		return c
	}
	return Class{Name: name, Extent: ExtentUnread}
}

// Fields is one record decoded by a class layout: each member's value, its text
// where it has any, and the body offset it was read at.
//
// The offsets are what every edit writes through, so a member that moves — as
// every member past a CString does, with its length — moves for the reader and
// the writer in one place.
type Fields struct {
	Value map[string]uint32
	Text  map[string]string
	Raw   map[string][]byte
	Off   map[string]int
	// Start and End bound the record in the body.
	Start, End int
}

// programme answers the members of a record of this class in file order,
// INCLUDING its base class's, and whether the whole programme is known.
//
// A derived class's record is its base's record followed by its own members, so
// the two are concatenated rather than the derived class restating the base —
// which is what makes "a Human record is a Unit record plus 24 bytes" a fact
// this table can hold rather than a comment somebody has to keep true.
func (c Class) programme(depth int) ([]Member, bool) {
	if depth > 4 {
		return nil, false
	}
	var out []Member
	if c.Base != "" {
		base, ok := Lookup(c.Base).programme(depth + 1)
		if !ok {
			return nil, false
		}
		out = append(out, base...)
	} else if c.Head {
		out = append(out, tokenMembers...)
	}
	if c.Extent == ExtentUnread || c.Extent == ExtentComputable {
		// A class whose own part is not a straight run of known members
		// cannot contribute a programme even where its base can.
		if len(c.Members) == 0 {
			return nil, false
		}
	}
	if len(c.Members) == 0 && c.Extent != 0 {
		return nil, false
	}
	return append(out, c.Members...), true
}

// decode reads one record of class c at off, driven by its programme.
//
// It answers an error rather than a partial value: a record that runs off the
// end of the stream, or a class with no programme, is a record this package
// cannot claim to have read.
func (c Class) decode(b []byte, off int) (Fields, error) {
	members, ok := c.programme(0)
	if !ok {
		return Fields{}, fmt.Errorf("sav: class %s has no decoded member layout", c.Name)
	}
	f := Fields{
		Value: make(map[string]uint32, len(members)),
		Text:  make(map[string]string),
		Raw:   make(map[string][]byte),
		Off:   make(map[string]int, len(members)),
		Start: off,
	}
	p := off
	for _, m := range members {
		if p < 0 || p > len(b) {
			return Fields{}, fmt.Errorf("sav: %s.%s at %d is outside the stream", c.Name, m.Name, p)
		}
		f.Off[m.Name] = p
		if m.Kind == KindCString {
			if p >= len(b) {
				return Fields{}, fmt.Errorf("sav: %s.%s has no length byte", c.Name, m.Name)
			}
			n := int(b[p])
			if n == 0xff {
				return Fields{}, fmt.Errorf("sav: %s.%s: extended or Unicode CString is unsupported", c.Name, m.Name)
			}
			if p+1+n > len(b) {
				return Fields{}, fmt.Errorf("sav: %s.%s of %d bytes overruns the stream",
					c.Name, m.Name, n)
			}
			f.Text[m.Name] = string(b[p+1 : p+1+n])
			p += 1 + n
			continue
		}
		w, ok := m.Kind.width(m.Len)
		if !ok || w <= 0 {
			return Fields{}, fmt.Errorf("sav: %s.%s has no width", c.Name, m.Name)
		}
		if p+w > len(b) {
			return Fields{}, fmt.Errorf("sav: %s.%s overruns the stream", c.Name, m.Name)
		}
		switch m.Kind {
		case KindU8:
			f.Value[m.Name] = uint32(b[p])
		case KindU16, KindU16Saturated:
			f.Value[m.Name] = uint32(u16(b, p))
		case KindU32, KindIdentity, KindReference:
			f.Value[m.Name] = u32(b, p)
		case KindU32Obfuscated:
			f.Value[m.Name] = u32(b, p) ^ obfuscator
		case KindRaw:
			f.Raw[m.Name] = b[p : p+w]
		}
		p += w
	}
	f.End = p
	return f, nil
}

// encode rebuilds a record FROM THE DECODED FORM — not by copying the bytes it
// was read from.
//
// It is the other half of the mechanism and the only way to tell a decode that
// works from a decode that merely does not crash: a field this package reads
// into the wrong variable, or writes at the wrong width, survives a
// carry-the-bytes round trip and does not survive this one.
func (c Class) encode(f Fields) ([]byte, error) {
	members, ok := c.programme(0)
	if !ok {
		return nil, fmt.Errorf("sav: class %s has no decoded member layout", c.Name)
	}
	var out []byte
	for _, m := range members {
		switch m.Kind {
		case KindCString:
			s, ok := f.Text[m.Name]
			if !ok {
				return nil, fmt.Errorf("sav: %s.%s was not read", c.Name, m.Name)
			}
			if len(s) >= 0xff {
				return nil, fmt.Errorf("sav: %s.%s is %d bytes, too long for its length byte",
					c.Name, m.Name, len(s))
			}
			out = append(out, byte(len(s)))
			out = append(out, s...)
			continue
		case KindRaw:
			r, ok := f.Raw[m.Name]
			if !ok || len(r) != m.Len {
				return nil, fmt.Errorf("sav: %s.%s wants %d raw bytes, has %d",
					c.Name, m.Name, m.Len, len(r))
			}
			out = append(out, r...)
			continue
		}
		v, ok := f.Value[m.Name]
		if !ok {
			return nil, fmt.Errorf("sav: %s.%s was not read", c.Name, m.Name)
		}
		switch m.Kind {
		case KindU8:
			out = append(out, uint8(v))
		case KindU16:
			out = append(out, uint8(v), uint8(v>>8))
		case KindU16Saturated:
			if v > saturate {
				v = saturate
			}
			out = append(out, uint8(v), uint8(v>>8))
		case KindU32Obfuscated:
			v ^= obfuscator
			fallthrough
		case KindU32, KindIdentity, KindReference:
			out = append(out, uint8(v), uint8(v>>8), uint8(v>>16), uint8(v>>24))
		default:
			return nil, fmt.Errorf("sav: %s.%s has no encoding", c.Name, m.Name)
		}
	}
	return out, nil
}

// set writes one member back through the layout that read it, applying that
// member's own arithmetic. A name the class does not carry is an error rather
// than a silent no-op, because a caller naming a field that does not exist has a
// bug and not a preference.
func (c Class) set(b []byte, f Fields, name string, v uint32) error {
	members, ok := c.programme(0)
	if !ok {
		return fmt.Errorf("sav: class %s has no decoded member layout", c.Name)
	}
	for _, m := range members {
		if m.Name != name {
			continue
		}
		off, ok := f.Off[name]
		if !ok {
			return fmt.Errorf("sav: %s.%s was not read", c.Name, name)
		}
		switch m.Kind {
		case KindU8:
			if v > 0xff {
				return fmt.Errorf("sav: %s.%s takes one byte, %d does not fit", c.Name, name, v)
			}
			b[off] = uint8(v)
		case KindU16:
			if v > 0xffff {
				return fmt.Errorf("sav: %s.%s takes two bytes, %d does not fit", c.Name, name, v)
			}
			put16(b, off, uint16(v))
		case KindU16Saturated:
			if v > saturate {
				v = saturate
			}
			put16(b, off, uint16(v))
		case KindU32, KindIdentity, KindReference:
			put32(b, off, v)
		case KindU32Obfuscated:
			put32(b, off, v^obfuscator)
		default:
			return fmt.Errorf("sav: %s.%s is not a settable kind", c.Name, name)
		}
		return nil
	}
	return fmt.Errorf("sav: class %s has no member %q", c.Name, name)
}
