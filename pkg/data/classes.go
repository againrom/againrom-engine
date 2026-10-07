package data

// The three class types, one per graphics registry: units/units.reg (34
// classes), objects/objects.reg (82) and structures/structures.reg (66). Each
// carries every key of its registry's measured key inventory as one exported
// field at that key's kind — int32, string or []int32 — and nothing else, so a
// loaded class is the registry section and no derived quantity.
//
// A FIELD NAME IS THE REGISTRY'S OWN LITERAL KEY SPELLING, and that is all it
// asserts (REG-LOC-016: the labels are the game's own, present as ASCII in the
// records). Apart from ID, Parent, File, DescText and the two structure geometry
// keys whose product bounds AnimMask, WHAT A KEY DOES IS NOT ESTABLISHED — so no
// comment below says what a value means. Values are loaded verbatim; nothing
// here computes on one.
//
// int32 is the type the .reg parser stores an integer at, kept rather than
// widened so a negative value stays visibly negative rather than folding.
//
// Each struct carries ONE UNEXPORTED FIELD, base: the sprite path SpritePath and
// OverlayPath append their extensions to, resolved once at load (sprite.go). It
// is derived from File and the registry's [Files] table rather than read from a
// key, so exporting it would make an exported field something other than an
// inventory key — and the bijection below is what says a class is its registry
// section and no derived quantity.
//
// Each struct is pinned against its key table in keys.go by a reflect-driven
// test, not by convention: a field with no row would never be loaded, and a row
// with no field would be validated and then dropped.

// UnitClass is one [UnitN] section of units/units.reg — 34 classes, 37 keys
// (REG-UNITS-018). Only ID, File, DescText and Sound are set by every class; a
// key a class omits is inherited from its Parent, or left zero — this registry's
// own per-key defaults are not decoded, unlike objects.reg's.
type UnitClass struct {
	// ID is how a placed record and a Parent name this class — never the
	// section index, which for this registry equals neither ID nor ID-1
	// (REG-KEY-044). The domain is sparse: 1..80 over 34 classes.
	ID int32
	// File indexes the registry's [Files] table; it is not a path.
	File int32
	// DescText is the class's own name text. The .reg format defines no
	// character encoding and this package applies none, so it holds the
	// registry's bytes.
	DescText string

	Sound           []int32
	InfoPicture     string
	AttackAnimTime  []int32
	AttackAnimFrame []int32
	AttackDelay     int32
	InMapEditor     int32
	Dying           int32
	AttackPhases    int32
	Palette         int32
	DyingPhases     int32
	MoveAnimTime    []int32
	MoveAnimFrame   []int32
	Index           int32
	MovePhases      int32
	MoveBeginPhases int32
	Width           int32
	Height          int32
	CenterX         int32
	CenterY         int32
	SelectionX1     int32
	SelectionX2     int32
	SelectionY1     int32
	SelectionY2     int32

	// Parent holds the ID of another class in this registry when the key is
	// present, and presence alone decides: 0 is a real reference to the class
	// whose ID is 0, not an absence.
	Parent int32

	BonePhases    int32
	ShootOffset   []int32
	Flip          int32
	Projectile    int32
	ShootDelay    int32
	TileSize      int32
	IdlePhases    int32
	IdleAnimTime  []int32
	IdleAnimFrame []int32
	Z             int32

	// base is this class's sprite path without the extension, resolved at load
	// from File and the registry's [Files] table; see sprite.go. Not a key, and
	// so not exported.
	base string
}

// TierLimit is the most tiers a class may have: four.
//
// It is a COMPILED CONSTANT in the original — the two parallel per-tier arrays
// on a class record sit 0x10 apart, so each holds four dwords and a fifth entry
// would store past one into the other — and it is a compiled constant here, with
// TierCount's clamp as its only reader. Nothing else in this tree spells the
// number for this purpose, which is what makes the seam one edit wide.
//
// Raising it costs NO SHIPPED BYTE OF OURS and is not free on the data side: a
// class cannot gain a fifth tier without a fifth colour-table node shipped
// beside its sheet, a larger Palette in the shipped registry, and a further
// definition-table row carrying that tier's own column value. The wire field the
// tier travels in is already wider than the limit — one byte read under a
// six-bit mask, so 63 values are reachable and 4 are used — so the ceiling is
// this array width and nothing else.
const TierLimit = 4

// TierCount is how many tiers this class has: its Palette key clamped into
// [0, TierLimit].
//
// Palette is the LENGTH of the class's per-tier table array, so a class at 0 has
// no tiers of its own — it takes the other colouring arm entirely — one at 1 has
// a single table with no subscript, and one at 4 has the four a shipped monster
// ships. Over the 34 shipped classes the key reads 0 on 18, 1 on 3 and 4 on 13.
//
// The clamp is TOTAL and refuses nothing. A key below zero — which is what an
// absent key resolves to in registries whose defaults are -1 — is a class with
// no tiers, and a key above the limit is the limit: the original's own behaviour
// past four is an array overrun, which is not a behaviour to reproduce, and
// refusing the class would fail a run on registry data the layer can still draw.
func (c *UnitClass) TierCount() int {
	switch {
	case c.Palette < 0:
		return 0
	case c.Palette > TierLimit:
		return TierLimit
	}
	return int(c.Palette)
}

// ObjectClass is one [ObjectN] section of objects/objects.reg — 82 classes, 16
// keys (REG-OBJ-039). Only ID and File are set by every class; seven classes
// carry neither DescText nor Parent and are otherwise complete.
//
// This is the one registry whose per-key defaults are decoded (REG-OBJ-046), so
// an int field whose key no section on the chain sets holds -1 here — 0 only for
// InMapEditor, and the Go zero for IconID, which the engine's own class record
// has no field for. See keys.go.
type ObjectClass struct {
	// ID is this class's identity and the key a placement names it by. The
	// domain is 0..81, equal to the section index on all 82 — the one registry
	// where the two coincide, and so the one that can never tell them apart.
	ID int32
	// File indexes the registry's [Files] table; it is not a path.
	File int32
	// DescText is the class's own name text, absent on seven classes.
	DescText string

	InMapEditor int32
	Index       int32
	Phases      int32
	Width       int32
	Height      int32
	CenterX     int32
	CenterY     int32

	// Parent holds the ID of another class in this registry when the key is
	// present; see UnitClass.Parent.
	Parent int32

	DeadObject     int32
	IconID         int32
	AnimationTime  []int32
	AnimationFrame []int32
	FireObject     int32

	// base is this class's sprite path without the extension; see UnitClass.base.
	base string
}

// StructureClass is one [StructureN] section of structures/structures.reg — 66
// classes, 23 keys (REG-STR-040). This registry has no [Files] table and no
// inheritance: File carries a sprite path directly, and no structure carries a
// Parent, so there is no Parent field to carry one into.
//
// The first sixteen keys are set by all 66; the seven after them are partial.
type StructureClass struct {
	// ID is this class's identity and the key a type-4 placement names it by.
	// The domain is 1..66, equal to the section index + 1 on all 66.
	ID int32
	// DescText is the class's own name text.
	DescText string
	// File is a sprite path, backslash-separated and extensionless — not an
	// index, this registry having no [Files] table.
	File string

	// TileWidth and FullHeight bound AnimMask: a non-empty mask's length is
	// their product, which is the one relation the contract validates here.
	TileWidth  int32
	TileHeight int32
	FullHeight int32

	SelectionX1 int32
	SelectionX2 int32
	SelectionY1 int32
	SelectionY2 int32
	ShadowY     int32
	Phases      int32
	Picture     string
	AnimMask    string
	AnimTime    []int32
	AnimFrame   []int32

	Indestructible int32
	IconID         int32
	Usable         int32
	Flat           int32
	LightRadius    int32
	LightPulse     int32
	VariableSize   int32

	// base is this class's sprite path without the extension, resolved at load
	// from File — which here is the path itself, not an index; see sprite.go.
	base string
}
