package sav

import (
	"errors"
	"fmt"
)

// ErrSpellbook marks malformed or unsupported learned membership. A caller
// must not publish a partial party or substitute a fresh book on this error.
var ErrSpellbook = errors.New("sav: invalid spellbook")

// CharacterSkillSlots is the number of independently serialized skill level
// and skill-XP fields on a Human.
const CharacterSkillSlots = 6

// Character is one of the human participant's own characters, as the walk read
// him: who he is, where he stands, and the state the file records.
//
// IT IS THE OWNER GRAPH AND NOT THE FILE. Every actor reachable from a Player
// is alive (SAV-DEATH-051: 276 of 276 carry death stage 0, health above zero
// and a clear teardown latch), because the format marks removal by unlinking
// from the group's actor list rather than by a flag. Corpses are in the stream
// and outside this list. So is everything else the map holds: its own units,
// the ground items, the sacks, the cell records and the session block are later
// items of the same document (SAV-DOC-053) and are outside the participant's
// subtree, which is what this walk reads.
type Character struct {
	Basis *ActorBasis
	// Class is Human or Unit, and Off the body offset of the record's head.
	Class string
	Off   int

	// Name is the character's own name, 8-bit text and NOT DECODED. Which
	// single-byte page it uses is not established, the same open question the
	// save's slot label carries.
	Name string

	// Cell is the packed (row << 8) | col position and FineX/FineY the
	// sub-cell offset, 0x80 each at rest — the same fixed point a map's unit
	// record stores a position in.
	Cell         uint16
	FineX, FineY uint8

	// RuntimeID is creation order and MapUnitID the map record's own id. A
	// character the map never placed carries no map unit id worth joining on,
	// which is exactly why a resume that joined by it dropped the player's own
	// party.
	RuntimeID uint32
	MapUnitID uint16

	// ArchiveIndex is the record's own tag in the archive's shared object
	// counter (Record.Index) — the same value ActorRecord.ArchiveIndex
	// already carries for this exact record elsewhere in this package, and
	// what sim.SourceBinding.ArchiveIndex names it once admitted (STORY
	// 1135, originaldiaries.go export path): a stable join key that does not
	// depend on MapUnitID, which the party's own lead character carries
	// none of.
	ArchiveIndex uint16

	// Key is the record's own identity key, the dword the file writes at the
	// end of every placeable object's head. It is a POINTER IN THE ORIGINAL'S
	// OWN ADDRESS SPACE and carries no meaning except as a key: it is what the
	// file's own cross-references resolve through on load, and comparing two
	// of them is the only thing to do with one.
	Key uint32

	// Hero reports that the enclosing Player's own starting-character field
	// names this character (SAV-HERO-059). It is the decoded answer to which
	// of a party is the person the player made, which nothing about the actor
	// itself carries: the actor list is a CObList written head to tail with no
	// comparison in the store arm, so a position in it ranks nothing
	// (SAV-GRPORD-058).
	//
	// EXACTLY ONE CHARACTER CARRIES IT on 18 of 18 distinct corpus saves. A
	// consumer must still count rather than take the first: what a file with a
	// second human participant does is Unknown, and a key of zero is treated as
	// naming nobody rather than as matching a record that has no key.
	Hero bool

	// Stats are the fourteen u16 words in file order, addressed by the Stat*
	// constants. Two of them are Unknown and a consumer must not read them.
	Stats     [UnitStatWords]uint16
	LoadState ActorLoadState

	// SkillLevels are the six u16 fields at Unit+0xa8+2i. SkillXP are the six
	// u32 fields at Humanoid+0x1cc+4i. Both arrays are serialized per Human and
	// paired by index (SAV-HEROSKILL-064).
	SkillLevels [CharacterSkillSlots]uint16
	SkillXP     [CharacterSkillSlots]uint32

	// Experience is Unit+0x130, the aggregate of this character's six SkillXP
	// values (SAV-HEROXP-063). It is retained separately because the original
	// serializes both forms.
	Experience uint32

	// DefRow is the head's +0x0c byte, which is a ROW INDEX INTO THE
	// COLLECTION THE CLASS PICKS: Humans for a Human record, Units for a
	// Unit one. The class is the table and there is no kind field selecting
	// it (ITEM-DEF-002, whose reading of Human::Serialize's own resolve at
	// L07758 against L02111 is the same instruction sequence the four
	// item classes use; DAT-OBJ-002 places L02111 as Humans).
	//
	// The row is re-derived at every load rather than stored, so this byte
	// is the whole of what the file says about which definition a character
	// was built from.
	DefRow         uint8
	DisplayBacking uint32

	// Stage is the death stage at +0x13c. It is 0 for every character this
	// walk reaches (SAV-DEATH-051) and is carried so that a consumer can say
	// so rather than assume it.
	Stage uint8

	// Worn is what he has on: the weapon at +0x74, the shield at +0x78 and
	// the twelve armour references, in that file order, with the empty slots
	// absent. The twelve are worn armour by a consumer reading and not by
	// corpus agreement (SAV-CARRY-050).
	Worn []Piece

	// Items is his container at +0x7c, in the order the file holds it.
	Items []Piece

	// ItemCount is how many elements the container DECLARED, which is the
	// count before any of them is interpreted: a consumer that carries fewer
	// than this can say by how much. len(Items) is how many of them resolved
	// to a record.
	ItemCount int

	// HasSpellbook reports the presence flag at +0x140. SpellCount is the
	// declared array size, including omitted slot zero, NOT the number of
	// learned spells. Spells retains every non-null slot in array order.
	HasSpellbook bool
	SpellCount   int
	Spells       []SavedSpell

	// HasDiary reports that the thirteenth Humanoid reference resolved to a
	// record, and JournalLen and JournalWords the two array lengths inside
	// it (SAV-DIARY-042). Diary is the same record decoded into typed,
	// sparse entries (SAV-667, SAV-668, diary.go); Diary.Length repeats
	// JournalLen/JournalWords (SAV-667: the two always agree), kept
	// separately because it is what a byte-for-byte re-export needs even
	// when Diary.Entries is empty.
	HasDiary                 bool
	JournalLen, JournalWords int
	Diary                    Diary
}

// SavedSpell is a non-null sparse-book slot. The values are the serialized
// instance fields (MAGIC-SPELL-001, SAV-SPELL-044), including values a current
// simulation may not yet model. Shared archive references produce detached
// values here and retain the same archive index and identity key.
type SavedSpell struct {
	Slot         int
	ID           uint8
	Range        uint8
	Defensive    uint8
	ManaCost     uint16
	ArchiveIndex uint16
	Key          uint32
}

// KnownSpells is this character's saved learned membership, never a template.
// Party validates the ID and slot before publishing a SavedSpell.
func (c Character) KnownSpells() uint32 {
	return savedSpellMembership(c.Spells)
}

// Piece is one item a character wears or carries.
type Piece struct {
	// Class is the record's own class: Weapon, Armor, Shield or Item.
	Class string

	// Code is the item's appearance word at +0x40. ITEM-APPEAR-023 reads it
	// as `(material << 12) | (slot << 8) | (shape << 5) | row`, which is
	// field for field the sixteen-bit code this tree's own data.ItemCode
	// composes — so this value needs no mapping to become one.
	Code uint16

	// Row is the head's +0x0c definition row index, and Stack the count at
	// +0x42. Row is the same five bits Code's low field carries, kept
	// separately because the head writes it as a whole byte and a row past 31
	// would corrupt the code's own shape field (ITEM-APPEAR-023).
	Row   uint8
	Stack uint16

	// Kind and Price are Item's concrete-kind byte and Token value. Effects
	// preserves every supported record — class "Effect", state 0 — in archive
	// order.
	Kind    uint8
	Price   int32
	Weight  int16
	Effects []ItemEffect

	// Exact concrete-class members. The definition row is Row, not Code's
	// low bits or the Weapon/Armor own-kind byte (ITEM-DEF-002).
	W52         [24]byte
	W6A         [22]byte
	W50         uint8
	A52         [22]byte
	A50         uint8
	S50         [22]byte
	WeaponSpell *SavedSpell

	// UnsupportedEffectStates reports records diverted before the general
	// class-Effect/state-0 dispatcher — a nonzero Token state, or a class
	// other than "Effect". They are not silently rewritten into state 0, and
	// they no longer refuse the container that holds them (docs/DIVERGENCES.md).
	UnsupportedEffectStates []uint8
}

// ItemEffect is the persisted effect identity imported from an original save.
type ItemEffect struct {
	Kind    uint8
	Mode    uint8
	Operand uint32
}

// Stat answers one of the fourteen words, or zero for an index that is not one.
func (c Character) Stat(i int) uint16 {
	if i < 0 || i >= UnitStatWords {
		return 0
	}
	return c.Stats[i]
}

// Row and Col unpack the cell.
func (c Character) Row() int { return int(c.Cell >> 8) }
func (c Character) Col() int { return int(c.Cell & 0xff) }

// Party is the human participant's distinct characters, in first-reference
// order. The walk reads groups and their actor lists in file order. Repeated
// archive references name one character, not additional party members; only the
// exact source-record offset is deduplicated, never a key, runtime ID or map ID.
//
// THAT ORDER IS NOT PARTICIPANT-FIRST AND IT RANKS NOTHING. SAV-GRPORD-058
// reads both arms of the group's actor-list serializer: the store arm is a
// head-to-tail CObList walk with twenty-four instructions and no comparison in
// them, and the load arm reproduces the file order exactly, so the order in the
// file is the order of appends and is runtime state. The same five actors, by
// identity key, appear as one group of five, five groups of one and three
// groups of 3/1/1 across one session — three partitions and three orders. A
// consumer must not read position as identity.
//
// This projection does not otherwise reorder. PartyWalk exposes every raw list
// occurrence. WHICH CHARACTER IS THE PLAYER'S OWN IS A FIELD, not a position:
// Character.Hero carries it (SAV-HERO-059). pkg/game.RestoreParty is where the
// party's own ordering rule is applied.
//
// IT RETURNS THE DISTINCT CHARACTERS READ ALONGSIDE ITS ERROR. An incomplete
// walk keeps its decoded prefix; PartyWalk retains occurrence-level evidence
// about where the next reference went wrong.
func (f *File) Party() ([]Character, error) {
	chars, _, err := f.PartyWalk()
	seen := make(map[int]bool, len(chars))
	unique := chars[:0]
	for _, c := range chars {
		if !seen[c.Off] {
			seen[c.Off] = true
			unique = append(unique, c)
		}
	}
	return unique, err
}

// PartyWalk returns every actor-list occurrence, including repeated archive
// references, beside the Player record. Its length is not a unique party or
// hero count. It is for a caller that needs the raw list or the walk's own
// extent — the available proof that the programme tiled the record exactly
// rather than merely not crashing. The
// Player record ends where the top-level list's next element begins
// (SAV-DOC-053), so the word at Record.End is an object tag naming Player or
// the null reference. A walk that ended anywhere else read some field at the
// wrong width.
//
// IT IS ALSO WHERE THE HERO FLAG IS SET, because this is the one place holding
// the Player record and its characters at once. The comparison is by identity
// key; a key of zero on either side names nobody.
func (f *File) PartyWalk() ([]Character, *Record, error) {
	rec, err := f.Walk()
	if rec == nil {
		return nil, nil, err
	}
	hero := rec.value("Hero")
	// A complete archive knows the later Player suffix owner. Partial forensic
	// walks retain the earlier Token-only projection and its diagnostic.
	var effective map[int]*ActorBasis
	var owners map[int]ActorReference
	if doc, _, exactErr := f.exactDocument(); exactErr == nil {
		graph, graphErr := savedActorGraph(doc.players)
		if graphErr == nil {
			effective = make(map[int]*ActorBasis, len(graph.Actors))
			owners = make(map[int]ActorReference, len(graph.Actors))
			for _, actor := range graph.Actors {
				effective[actor.Off] = actor.Character.Basis
				owners[actor.Off] = actor.Owner
			}
		}
	}
	var out []Character
	for _, a := range rec.Refs["Actors"] {
		c, cerr := character(a)
		if cerr != nil {
			if err == nil {
				err = cerr
			} else if errors.Is(cerr, ErrSpellbook) && !errors.Is(err, ErrSpellbook) {
				// Keep the first diagnostic, but never let it mask a later
				// fatal book error. Join at most once: a large malformed actor
				// list must not create a recursively growing error chain.
				err = errors.Join(err, cerr)
			}
			continue
		}
		c.Hero = hero != 0 && c.Key == hero
		c.Basis = effective[a.Off]
		if owner, known := owners[a.Off]; c.Basis == nil && known {
			c.Basis, cerr = partyBasis(a, owner, rec)
		} else if c.Basis == nil {
			c.Basis, cerr = actorBasis(a, []*Record{rec})
		}
		if cerr != nil {
			return nil, rec, cerr
		}
		out = append(out, c)
	}
	return out, rec, err
}

// partyBasis reads the derive basis of an actor the owner graph carries
// without one, which is a dead actor. Its owner is the graph's effective one:
// the saved reference resolved when the Token loaded, then overwritten by the
// Player whose Group holds the actor (SAV-PTRMAP-035, SAV-GRPOWNER-561,
// PARTY-OWN-001). A walked Group member's saved reference to a later Player,
// or to no Player, therefore names this Player. An effective owner that is
// another Player, or that does not resolve, stops the walk.
func partyBasis(a *Record, owner ActorReference, player *Record) (*ActorBasis, error) {
	b, err := actorBasisFields(a)
	if err != nil || owner.Key == 0 {
		return b, err
	}
	if !owner.Resolved || owner.Class != "Player" {
		return nil, fmt.Errorf("sav: actor %d owner %#x does not resolve", a.Off, owner.Key)
	}
	if owner.ArchiveIndex != player.Index {
		return nil, fmt.Errorf("sav: actor %d owner %#x is another Player", a.Off, owner.Key)
	}
	b.Human.HasOwner, b.Human.ManaReservePercent = true, owner.PlayerF58
	return b, nil
}

// character reads one actor record into the view above.
func character(r *Record) (Character, error) {
	if r.Class != "Human" && r.Class != "Unit" {
		return Character{}, fmt.Errorf("sav: a group's actor list holds a %s, which is not "+
			"a character", r.Class)
	}
	return actorCharacter(r)
}

// actorCharacter projects decoded common fields without asserting that exact
// Humanoid has a constructible SAV definition or belongs in the initial party.
func actorCharacter(r *Record) (Character, error) {
	block := r.Raw["Block12"]
	if len(block) != 12 {
		return Character{}, fmt.Errorf("sav: the %s at %d has no head block", r.Class, r.Off)
	}
	skillBlock := r.Raw["UA6"]
	if len(skillBlock) != 24 {
		return Character{}, fmt.Errorf("sav: the %s at %d has no Unit skill block", r.Class, r.Off)
	}
	var skillXPBlock []byte
	if r.Class != "Unit" {
		skillXPBlock = r.Raw["H1CC"]
		if len(skillXPBlock) != 24 {
			return Character{}, fmt.Errorf("sav: the Human at %d has no Humanoid skill-XP block", r.Off)
		}
	}
	c := Character{
		LoadState:    actorLoadState(r),
		Class:        r.Class,
		Off:          r.Off,
		ArchiveIndex: r.Index,
		Name:         r.Text["Name"],
		Cell:         u16(block, 0),
		FineX:        block[4],
		FineY:        block[5],
		RuntimeID:    r.value("RuntimeID"),
		Key:          r.value("Identity"),
		// The map unit id is the LOW u16 of the head's T08 dword — the same
		// head+0x13 identified by SAV-ID-015.
		MapUnitID:      uint16(r.value("T08")),
		DefRow:         uint8(r.value("T0C")),
		DisplayBacking: r.value("U148"),
		Stage:          uint8(r.value("Stage")),
		Experience:     r.value("U130"),
		ItemCount:      r.Counts["Inventory"],
		HasSpellbook:   r.value("HasSpellbook") == 1,
		SpellCount:     r.Counts["Spells"],
	}
	for i, n := range statNames {
		c.Stats[i] = uint16(r.value(n))
	}
	for i := range c.SkillLevels {
		c.SkillLevels[i] = u16(skillBlock, 2+2*i)
		if len(skillXPBlock) != 0 {
			c.SkillXP[i] = u32(skillXPBlock, 4*i)
		}
	}
	var err error
	c.Spells, err = recordSpells(r)
	if err != nil {
		return Character{}, err
	}
	// THE WEAPON AND THE SHIELD ARE WORN TOO. Weapon::Equip stores into
	// actor+0x74 and Shield::Equip into actor+0x78 (ITEM-HUMEQ-030); the
	// twelve at +0x198 are the armour. All fourteen are equipment, so they are
	// one list here and the consumer routes each by the slot its own code
	// carries rather than by which site it was read from.
	for _, site := range []string{"HeldWeapon", "HeldShield", "Worn"} {
		for _, ref := range r.Refs[site] {
			c.Worn = append(c.Worn, piece(ref))
		}
	}
	for _, ref := range r.Refs["Inventory"] {
		c.Items = append(c.Items, piece(ref))
	}
	if d := r.Refs["Diary"]; len(d) > 0 {
		c.HasDiary = true
		c.JournalLen = d[0].Counts["Journal"]
		c.JournalWords = d[0].Counts["JournalWords"]
		diary, err := diaryFromRecord(d[0])
		if err != nil {
			return Character{}, err
		}
		c.Diary = diary
	}
	return c, nil
}

// piece reads one item record into the view above.
func piece(r *Record) Piece {
	p := Piece{
		Class:  r.Class,
		Code:   uint16(r.value("F40")),
		Row:    uint8(r.value("T0C")),
		Stack:  uint16(r.value("F42")),
		Kind:   uint8(r.value("F44")),
		Price:  int32(r.value("T1C")),
		Weight: int16(r.value("F4A")),
	}
	copy(p.W52[:], r.Raw["W52"])
	copy(p.W6A[:], r.Raw["W6A"])
	copy(p.A52[:], r.Raw["A52"])
	copy(p.S50[:], r.Raw["S50"])
	p.W50, p.A50 = uint8(r.value("W50")), uint8(r.value("A50"))
	if refs := r.Refs["WeaponSpell"]; len(refs) == 1 {
		s := refs[0]
		p.WeaponSpell = &SavedSpell{ID: uint8(s.value("S08")), Range: uint8(s.value("S09")), Defensive: uint8(s.value("S0A")), ManaCost: uint16(s.value("S0C")), ArchiveIndex: s.Index, Key: s.value("This")}
	}
	for _, effect := range r.Refs["Effects"] {
		state := uint8(effect.value("E0C"))
		// A non-"Effect" class (Effect_DirectDamage: SAV-EQUIPEFFECT-553 names
		// both classes on an item, but no claim states either is expected
		// here specifically) is diverted the same as a nonzero state: reading
		// its own E0C is still Token's own +0x0c (SAV-TOKEN-034, the same head
		// every subtype opens with), so the diverted value is real, not
		// fabricated — it is the reason for the class, not the state, that is
		// Unknown.
		if effect.Class != "Effect" || state != 0 {
			p.UnsupportedEffectStates = append(p.UnsupportedEffectStates, state)
			continue
		}
		p.Effects = append(p.Effects, ItemEffect{
			Kind: uint8(effect.value("E3C")), Mode: uint8(effect.value("E3D")), Operand: effect.value("E40"),
		})
	}
	return p
}
