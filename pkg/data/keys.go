package data

import "againrom/pkg/formats/reg"

// One key table per registry, and one descriptor beside it.
//
// The table is the single place a registry's inventory is written: every row is
// a key's literal name, the kind it is read at, and the address of the field it
// lands in. Splitting those apart is what lets a key be validated but never
// stored, or stored but never kind-checked; keys_test.go pins each table against
// its struct so neither can happen silently.
//
// The descriptor carries what differs between the three registries and nothing
// that does not: the [Global] count key, the section-name prefix the loader
// constructs its lookups from, the sprite-path prefix, whether the registry has
// a [Files] table, and whether Parent is legal in it.

// keyKind is the value kind a key is read at: the .reg value type a well-formed
// record must carry, and the Go type its field holds.
type keyKind int

const (
	kindInt   keyKind = iota // reg.TypeInt      -> int32
	kindStr                  // reg.TypeString   -> string
	kindArray                // reg.TypeIntArray -> []int32
)

func (k keyKind) String() string {
	switch k {
	case kindInt:
		return "int"
	case kindStr:
		return "str"
	case kindArray:
		return "array"
	}
	return "kind?"
}

// keyRow is one row of a key table.
//
// ptr must yield the address of the field named by name, at the Go type kind
// implies. That the field belongs to this struct and carries that type is
// checked by keys_test.go; that it is the RIGHT field of that type is not
// something a reflect test can see, and is pinned instead by the written-out
// inventory round-trip.
type keyRow[T any] struct {
	name string
	kind keyKind
	ptr  func(*T) any
}

// descriptor is what a loader needs to know about a registry beyond its key
// table.
type descriptor struct {
	countKey      string // the [Global] key holding the class count
	sectionPrefix string // a class section is this + the section index
	spritePrefix  string // the archive directory a sprite entry sits under
	hasFiles      bool   // the registry has a [Files] table and a FileCount
	parentLegal   bool   // a class in this registry may carry Parent
}

// The three registries. structures.reg names its count plainly, has no [Files]
// table — each structure carries its sprite path itself — and never inherits: a
// Parent on a structure is malformed, not ignored.
var (
	unitDesc      = descriptor{countKey: "UnitCount", sectionPrefix: "Unit", spritePrefix: "units/", hasFiles: true, parentLegal: true}
	objectDesc    = descriptor{countKey: "ObjectCount", sectionPrefix: "Object", spritePrefix: "objects/", hasFiles: true, parentLegal: true}
	structureDesc = descriptor{countKey: "Count", sectionPrefix: "Structure", spritePrefix: "structures/", hasFiles: false, parentLegal: false}
)

// unitKeys is the 37-key inventory of units/units.reg, in the order spec.md
// lists it so the two can be read side by side. File is an index into [Files].
var unitKeys = []keyRow[UnitClass]{
	{"ID", kindInt, func(c *UnitClass) any { return &c.ID }},
	{"File", kindInt, func(c *UnitClass) any { return &c.File }},
	{"DescText", kindStr, func(c *UnitClass) any { return &c.DescText }},
	{"Sound", kindArray, func(c *UnitClass) any { return &c.Sound }},
	{"InfoPicture", kindStr, func(c *UnitClass) any { return &c.InfoPicture }},
	{"AttackAnimTime", kindArray, func(c *UnitClass) any { return &c.AttackAnimTime }},
	{"AttackAnimFrame", kindArray, func(c *UnitClass) any { return &c.AttackAnimFrame }},
	{"AttackDelay", kindInt, func(c *UnitClass) any { return &c.AttackDelay }},
	{"InMapEditor", kindInt, func(c *UnitClass) any { return &c.InMapEditor }},
	{"Dying", kindInt, func(c *UnitClass) any { return &c.Dying }},
	{"AttackPhases", kindInt, func(c *UnitClass) any { return &c.AttackPhases }},
	{"Palette", kindInt, func(c *UnitClass) any { return &c.Palette }},
	{"DyingPhases", kindInt, func(c *UnitClass) any { return &c.DyingPhases }},
	{"MoveAnimTime", kindArray, func(c *UnitClass) any { return &c.MoveAnimTime }},
	{"MoveAnimFrame", kindArray, func(c *UnitClass) any { return &c.MoveAnimFrame }},
	{"Index", kindInt, func(c *UnitClass) any { return &c.Index }},
	{"MovePhases", kindInt, func(c *UnitClass) any { return &c.MovePhases }},
	{"MoveBeginPhases", kindInt, func(c *UnitClass) any { return &c.MoveBeginPhases }},
	{"Width", kindInt, func(c *UnitClass) any { return &c.Width }},
	{"Height", kindInt, func(c *UnitClass) any { return &c.Height }},
	{"CenterX", kindInt, func(c *UnitClass) any { return &c.CenterX }},
	{"CenterY", kindInt, func(c *UnitClass) any { return &c.CenterY }},
	{"SelectionX1", kindInt, func(c *UnitClass) any { return &c.SelectionX1 }},
	{"SelectionX2", kindInt, func(c *UnitClass) any { return &c.SelectionX2 }},
	{"SelectionY1", kindInt, func(c *UnitClass) any { return &c.SelectionY1 }},
	{"SelectionY2", kindInt, func(c *UnitClass) any { return &c.SelectionY2 }},
	{"Parent", kindInt, func(c *UnitClass) any { return &c.Parent }},
	{"BonePhases", kindInt, func(c *UnitClass) any { return &c.BonePhases }},
	{"ShootOffset", kindArray, func(c *UnitClass) any { return &c.ShootOffset }},
	{"Flip", kindInt, func(c *UnitClass) any { return &c.Flip }},
	{"Projectile", kindInt, func(c *UnitClass) any { return &c.Projectile }},
	{"ShootDelay", kindInt, func(c *UnitClass) any { return &c.ShootDelay }},
	{"TileSize", kindInt, func(c *UnitClass) any { return &c.TileSize }},
	{"IdlePhases", kindInt, func(c *UnitClass) any { return &c.IdlePhases }},
	{"IdleAnimTime", kindArray, func(c *UnitClass) any { return &c.IdleAnimTime }},
	{"IdleAnimFrame", kindArray, func(c *UnitClass) any { return &c.IdleAnimFrame }},
	{"Z", kindInt, func(c *UnitClass) any { return &c.Z }},
}

// objectKeys is the 16-key inventory of objects/objects.reg. File is an index
// into [Files].
var objectKeys = []keyRow[ObjectClass]{
	{"ID", kindInt, func(c *ObjectClass) any { return &c.ID }},
	{"File", kindInt, func(c *ObjectClass) any { return &c.File }},
	{"DescText", kindStr, func(c *ObjectClass) any { return &c.DescText }},
	{"InMapEditor", kindInt, func(c *ObjectClass) any { return &c.InMapEditor }},
	{"Index", kindInt, func(c *ObjectClass) any { return &c.Index }},
	{"Phases", kindInt, func(c *ObjectClass) any { return &c.Phases }},
	{"Width", kindInt, func(c *ObjectClass) any { return &c.Width }},
	{"Height", kindInt, func(c *ObjectClass) any { return &c.Height }},
	{"CenterX", kindInt, func(c *ObjectClass) any { return &c.CenterX }},
	{"CenterY", kindInt, func(c *ObjectClass) any { return &c.CenterY }},
	{"Parent", kindInt, func(c *ObjectClass) any { return &c.Parent }},
	{"DeadObject", kindInt, func(c *ObjectClass) any { return &c.DeadObject }},
	{"IconID", kindInt, func(c *ObjectClass) any { return &c.IconID }},
	{"AnimationTime", kindArray, func(c *ObjectClass) any { return &c.AnimationTime }},
	{"AnimationFrame", kindArray, func(c *ObjectClass) any { return &c.AnimationFrame }},
	{"FireObject", kindInt, func(c *ObjectClass) any { return &c.FireObject }},
}

// structureKeys is the 23-key inventory of structures/structures.reg: the
// sixteen every class carries, then the seven that are partial. File is a path,
// not an index, and there is no Parent row — this registry does not inherit.
var structureKeys = []keyRow[StructureClass]{
	{"ID", kindInt, func(c *StructureClass) any { return &c.ID }},
	{"DescText", kindStr, func(c *StructureClass) any { return &c.DescText }},
	{"File", kindStr, func(c *StructureClass) any { return &c.File }},
	{"TileWidth", kindInt, func(c *StructureClass) any { return &c.TileWidth }},
	{"TileHeight", kindInt, func(c *StructureClass) any { return &c.TileHeight }},
	{"FullHeight", kindInt, func(c *StructureClass) any { return &c.FullHeight }},
	{"SelectionX1", kindInt, func(c *StructureClass) any { return &c.SelectionX1 }},
	{"SelectionX2", kindInt, func(c *StructureClass) any { return &c.SelectionX2 }},
	{"SelectionY1", kindInt, func(c *StructureClass) any { return &c.SelectionY1 }},
	{"SelectionY2", kindInt, func(c *StructureClass) any { return &c.SelectionY2 }},
	{"ShadowY", kindInt, func(c *StructureClass) any { return &c.ShadowY }},
	{"Phases", kindInt, func(c *StructureClass) any { return &c.Phases }},
	{"Picture", kindStr, func(c *StructureClass) any { return &c.Picture }},
	{"AnimMask", kindStr, func(c *StructureClass) any { return &c.AnimMask }},
	{"AnimTime", kindArray, func(c *StructureClass) any { return &c.AnimTime }},
	{"AnimFrame", kindArray, func(c *StructureClass) any { return &c.AnimFrame }},
	{"Indestructible", kindInt, func(c *StructureClass) any { return &c.Indestructible }},
	{"IconID", kindInt, func(c *StructureClass) any { return &c.IconID }},
	{"Usable", kindInt, func(c *StructureClass) any { return &c.Usable }},
	{"Flat", kindInt, func(c *StructureClass) any { return &c.Flat }},
	{"LightRadius", kindInt, func(c *StructureClass) any { return &c.LightRadius }},
	{"LightPulse", kindInt, func(c *StructureClass) any { return &c.LightPulse }},
	{"VariableSize", kindInt, func(c *StructureClass) any { return &c.VariableSize }},
}

// --- the absent-everywhere defaults ----------------------------------------
//
// A key NO SECTION ON A CLASS'S CHAIN SETS does not resolve to zero. The engine
// passes a per-key default to every scalar read of objects.reg, and zero is a
// legal object ID: read as 0, a DeadObject nobody sets names the class whose ID
// is 0 — Object0 — instead of naming nothing.
//
// One row per key, next to the table that says the key exists, because that is
// the shape of the evidence: one immediate per read site, not a registry-wide
// rule with exceptions (REG-OBJ-046). A key with no row keeps its type's zero,
// which is what structures.reg does wholesale — its per-key defaults are not
// decoded, and the contract states that limit rather than letting the two
// decoded tables read as general.

// scalarDefault is one int key's absent-everywhere value.
//
// noInherit says the default is unconditional: the key does not inherit at all,
// so a class whose OWN section omits it takes value whatever an ancestor holds.
// objects.reg's File is the only such key — its default is a literal, where
// every other scalar of that registry defaults to the parent's resolved field.
// It is declared with the row it belongs to and read by the resolution in
// load.go.
type scalarDefault struct {
	key       string
	value     int32
	noInherit bool
}

// The three registries' default tables.
//
// objects.reg carries twelve rows: the eleven int keys the engine's own class
// record has a field for, plus Parent, which its loader reads into a local and
// tests against -1 — that test is what says Parent's own absent value is -1,
// where 0 is a real edge. Left out on purpose: IconID, which that record has no
// field for, so no default is decided for it; DescText, a str no claim gives a
// default; and the two array keys, whose absence is the array guard's business
// and resolves to nil.
//
// units.reg carries its whole scalar inventory but one: seventeen keys default
// to -1, eight to 0, TileSize to 1 — and InMapEditor has NO row, the engine's
// unit record having no field for it, IconID's shape above. Parent's -1 is the
// no-parent value; presence still overrides. No row is noInherit: File keeps
// inheriting here, opposite to objects.reg's, and a class whose chain never
// sets File still resolves no sprite path, because the sprite base is computed
// from the resolved NODE — nil when the key is set nowhere — not from the
// filled field (sprite.go).
var (
	structureDefaults []scalarDefault // not decoded: a key set nowhere keeps the Go zero, and this registry does not inherit at all

	unitDefaults = []scalarDefault{
		{key: "ID", value: -1},
		{key: "File", value: -1}, // inherits, unlike objects.reg's File: not noInherit
		{key: "AttackDelay", value: 0},
		{key: "Dying", value: 0},
		{key: "AttackPhases", value: -1},
		{key: "Palette", value: 0},
		{key: "DyingPhases", value: -1},
		{key: "Index", value: -1},
		{key: "MovePhases", value: -1},
		{key: "MoveBeginPhases", value: -1},
		{key: "Width", value: -1},
		{key: "Height", value: -1},
		{key: "CenterX", value: -1},
		{key: "CenterY", value: -1},
		{key: "SelectionX1", value: -1},
		{key: "SelectionX2", value: -1},
		{key: "SelectionY1", value: -1},
		{key: "SelectionY2", value: -1},
		{key: "Parent", value: -1},
		{key: "BonePhases", value: -1},
		{key: "Flip", value: 0},
		{key: "Projectile", value: 0},
		{key: "ShootDelay", value: 0},
		{key: "TileSize", value: 1},
		{key: "IdlePhases", value: 0},
		{key: "Z", value: 0},
	}

	objectDefaults = []scalarDefault{
		{key: "ID", value: -1},
		{key: "File", value: -1, noInherit: true}, // the one key that does not inherit
		{key: "InMapEditor", value: 0},            // 0 here is the decoded default, not a fallthrough
		{key: "Index", value: -1},
		{key: "Phases", value: -1},
		{key: "Width", value: -1},
		{key: "Height", value: -1},
		{key: "CenterX", value: -1},
		{key: "CenterY", value: -1},
		{key: "Parent", value: -1},
		{key: "DeadObject", value: -1},
		{key: "FireObject", value: -1},
	}
)

// --- the resolved-length rules ---------------------------------------------
//
// Three of Validation's cases are about a RESOLVED value's length rather
// than a key's presence or type: ShootOffset's 16, a structure's AnimMask
// against TileWidth x FullHeight, and a paired animation key that must
// resolve to the same length as its partner. They are evaluated after
// inheritance, so a pair the child writes on one key and inherits on the
// other is caught as the mismatch it is — which neither node can show on
// its own.
//
// These are tables over key names, as the key table is, and so they sit beside
// it rather than on the descriptor, which carries one scalar per registry
// difference and is compared as a whole value.
//
// WHAT IS ABSENT HERE IS AS LOAD-BEARING AS WHAT IS PRESENT. Sound's "always
// length 5" is an inventory measurement, not a rule Validation states, so no row
// asserts it: a rejection the contract never states would refuse data the
// contract accepts.

// fixedLen is an array key whose one legal resolved length the contract states
// outright. A resolved length of 0 — the key absent with no ancestor to take it
// from — is legal and is not this rule's business: only a value that is there
// and is the wrong length is malformed.
type fixedLen struct {
	key string
	n   int
}

// animPair is two array keys that must resolve to one length. "Without its
// partner" needs no rule of its own: an absent partner resolves to length 0, and
// 0 differs from any length the other key has.
type animPair struct{ time, frame string }

// maskRule is a str key whose resolved length must equal the product of two int
// keys, checked only where the string is non-empty — 52 of the 66 shipped
// structures carry an empty AnimMask. An empty key name means the registry has
// no such rule.
type maskRule struct{ key, width, height string }

// lengths is one registry's rules. validate_test.go pins every name here to a
// row of that registry's key table, so a rule cannot name a key that does not
// exist and be silently skipped.
type lengths struct {
	fixed []fixedLen
	pairs []animPair
	mask  maskRule
}

var (
	unitLengths = lengths{
		fixed: []fixedLen{{"ShootOffset", 16}},
		pairs: []animPair{
			{"AttackAnimTime", "AttackAnimFrame"},
			{"MoveAnimTime", "MoveAnimFrame"},
			{"IdleAnimTime", "IdleAnimFrame"},
		},
	}
	objectLengths = lengths{
		pairs: []animPair{{"AnimationTime", "AnimationFrame"}},
	}
	structureLengths = lengths{
		pairs: []animPair{{"AnimTime", "AnimFrame"}},
		mask:  maskRule{key: "AnimMask", width: "TileWidth", height: "FullHeight"},
	}
)

// --- the node reader -------------------------------------------------------
//
// reg's Get* accessors report one false for three situations — section missing,
// key missing, key present at another value type — and this contract needs the
// third split from the first two: an absent key INHERITS, a wrong-kind key is a
// hard error. Reading that false as "absent" would inherit silently over a
// malformed key. So the section node and the key node are resolved here, over
// Reg.Root, and classified against the key table's own kind.
//
// The ASCII-only fold is this package's own copy: reg's is unexported, and the
// .reg format is frozen, so the copy cannot drift out from under us.

// foldASCII maps 'A'-'Z' to 'a'-'z' and leaves every other byte, ASCII or not,
// unchanged. A byte >= 0x80 compares as itself.
func foldASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b - 'A' + 'a'
	}
	return b
}

// nameEqualFold reports whether a and b are equal under the ASCII-only fold.
func nameEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if foldASCII(a[i]) != foldASCII(b[i]) {
			return false
		}
	}
	return true
}

// findChild returns the first child of n whose name matches under the fold, in
// node-table order, or nil. The format permits duplicate sibling names and
// assigns no meaning to one, so first-in-table-order is the rule two runs agree
// on.
func findChild(n *reg.Node, name string) *reg.Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if nameEqualFold(c.Name, name) {
			return c
		}
	}
	return nil
}

// findSection returns the directory node named name among the registry's
// top-level nodes, or nil if there is none — a name that resolves to a value
// node rather than a directory misses, as it does through reg's own accessors.
func findSection(r *reg.Reg, name string) *reg.Node {
	if r == nil {
		return nil
	}
	sec := findChild(r.Root, name)
	if sec == nil || !sec.Dir {
		return nil
	}
	return sec
}

// keyState is what a section says about one key: the three answers the contract
// treats differently, kept apart.
type keyState int

const (
	keyAbsent    keyState = iota // no node of that name here; the key inherits
	keyPresent                   // a node carrying a value of the asked kind
	keyWrongKind                 // a node of some other kind; a hard error
)

// readKey resolves the key named name inside sec and classifies it against
// kind. The node comes back for keyPresent and is nil otherwise, so a caller
// cannot read a value it was not told it had. A nil section reads as keyAbsent
// rather than panicking; the loader rejects a missing section on its own
// grounds, before any key is read.
//
// The one node that is not what it looks like is the array sentinel: an
// array-valued key may be stored as a ZERO-LENGTH STRING, the editor's "none"
// marker, and that is a present key of length 0, not a kind fault. A non-empty
// string where an array is expected is the fault.
func readKey(sec *reg.Node, name string, kind keyKind) (*reg.Node, keyState) {
	n := findChild(sec, name)
	if n == nil {
		return nil, keyAbsent
	}
	if !kind.fits(n) {
		return nil, keyWrongKind
	}
	return n, keyPresent
}

// fits reports whether n carries a value of this kind. Dir is tested first: a
// directory's Type reads as reg.TypeString and a directory is never a value.
func (k keyKind) fits(n *reg.Node) bool {
	if n == nil || n.Dir {
		return false
	}
	switch k {
	case kindInt:
		return n.Type == reg.TypeInt
	case kindStr:
		return n.Type == reg.TypeString
	case kindArray:
		return n.Type == reg.TypeIntArray || (n.Type == reg.TypeString && n.Str == "")
	}
	return false
}
