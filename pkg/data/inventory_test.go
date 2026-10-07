package data

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// The inventory round-trip (SC-8, AC-8): one class per registry carrying
// EVERY key of its inventory at its own distinct sentinel, compared against
// a WRITTEN-OUT expected struct.
//
// This is the check keys_test.go's bijection deliberately cannot make. That test
// proves by reflection that every key has a field and every field has a key, and
// that a row's pointer lands on a field of this struct at the kind's Go type —
// but not that it lands on the field the row NAMES. Two rows of one kind can be
// transposed, each still pointing at a real field of the right type, and every
// other test in this package stays green: none of them sets both keys of such a
// pair at once at values that differ. Here every key differs from every other, so
// a transposition has somewhere to fail.
//
// Which is why the expectations are written out by hand and never generated from
// the key table: expectations built from the table under test would agree with a
// transposed row exactly as the loader does, and the blind spot would come back.
//
// Verbatim is the property: no arithmetic is performed on a registry value,
// so a distinct value per key read back IDENTICAL is what pins it — the
// string keys and the arrays included, and including a class that reaches
// most of its values through a Parent, where verbatim means the resolving
// ancestor's key held exactly this.
//
// Beside the round-trip sit the three cases that are VALUES AND NOT ERRORS: a
// class with no DescText, one whose sprite entry names nothing that ships, and
// one carrying a key the inventory does not have.
//
// Fixtures are synthetic .reg byte streams built by internal/synth and parsed
// back by pkg/formats/reg. Nothing here opens an archive, and none exists to
// open.

// checkRoundTrip compares a loaded class against a written-out expectation, field
// by field so a mismatch names the key rather than dumping two structs — and
// checks the EXPECTATION first, on the two properties that make the comparison
// mean what it claims:
//
//   - every exported field of want is NON-ZERO. Every inventory key carries a
//     sentinel here, so a zero is either a key the fixture forgot to set or an
//     exported field no key table row fills — and a zero on both sides would
//     otherwise agree and pass.
//   - no two exported fields of want carry the SAME value. Two keys sharing a
//     sentinel could be transposed with nothing to show it, which is the one
//     failure this whole test exists for.
//
// The walk is over exported fields, every one of which is an inventory key
// (keys_test.go). The single unexported field, base, is not a key: it is derived
// from File at load and is pinned by sprite_test.go.
func checkRoundTrip[T any](t *testing.T, label string, got, want *T) {
	t.Helper()

	gv, wv := reflect.ValueOf(got).Elem(), reflect.ValueOf(want).Elem()
	typ := gv.Type()

	sentinel := make(map[string]string, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		w := wv.Field(i).Interface()
		if wv.Field(i).IsZero() {
			t.Errorf("%s: the expectation leaves %s at its zero value — every key of this "+
				"inventory carries a distinct sentinel, so a zero here is a key the fixture "+
				"does not set, or a field no key fills", label, f.Name)
			continue
		}
		key := fmt.Sprintf("%T:%v", w, w)
		if prev, dup := sentinel[key]; dup {
			t.Errorf("%s: %s and %s share the value %v — two keys at one value could be "+
				"transposed with nothing to show it", label, prev, f.Name, w)
		}
		sentinel[key] = f.Name
	}

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		g, w := gv.Field(i).Interface(), wv.Field(i).Interface()
		if !reflect.DeepEqual(g, w) {
			t.Errorf("%s: %s = %v, want %v — the registry value at that key, verbatim",
				label, f.Name, g, w)
		}
	}
}

// --- units: all 37 keys ------------------------------------------------------

// One value per array key, each its own. Contents differ everywhere; lengths
// differ too wherever Validation leaves the choice free — ShootOffset is 16
// because the contract states that length outright, and each animation pair is
// written at one length because the contract states that too.
var (
	unitSound           = []int32{510, 511, 512, 513, 514}
	unitAttackAnimTime  = []int32{520, 521}
	unitAttackAnimFrame = []int32{530, 531}
	unitMoveAnimTime    = []int32{540, 541, 542}
	unitMoveAnimFrame   = []int32{550, 551, 552}
	unitShootOffset     = []int32{
		560, 561, 562, 563, 564, 565, 566, 567,
		568, 569, 570, 571, 572, 573, 574, 575,
	}
	unitIdleAnimTime  = []int32{580, 581, 582, 583}
	unitIdleAnimFrame = []int32{590, 591, 592, 593}

	childMoveAnimTime  = []int32{640, 641, 642}
	childMoveAnimFrame = []int32{650, 651, 652}
)

// inventoryUnitsReg holds three classes: the one Parent names, the full class,
// and a child that reaches 31 of its 37 keys through it.
//
// File is 3 against a four-entry [Files] table whose other entries are never
// referenced: a loader reading "the first entry" rather than the class's own File
// resolves a different path, and 3 is a sentinel like any other.
func inventoryUnitsReg(t *testing.T) *reg.Reg {
	t.Helper()
	files := []synth.RegNode{
		regStr("File0", "unreferenced-0"),
		regStr("File1", "unreferenced-1"),
		regStr("File2", "unreferenced-2"),
		regStr("File3", `Unit\Sprite`),
	}
	return spriteUnitsReg(t, files,
		// The class Parent names. It sets nothing but its own ID, so nothing it
		// holds can reach the full class and stand in for a key that class sets
		// itself — every value below is the full class's own.
		regDir("Unit0", regInt("ID", 7)),

		// The full class: every key of the inventory, in key-table order.
		regDir("Unit1",
			regInt("ID", 101),
			regInt("File", 3),
			regStr("DescText", "unit-desc-text"),
			regInts("Sound", unitSound...),
			regStr("InfoPicture", "unit-info-picture"),
			regInts("AttackAnimTime", unitAttackAnimTime...),
			regInts("AttackAnimFrame", unitAttackAnimFrame...),
			regInt("AttackDelay", 110),
			regInt("InMapEditor", 111),
			regInt("Dying", 112),
			regInt("AttackPhases", 113),
			regInt("Palette", 114),
			regInt("DyingPhases", 115),
			regInts("MoveAnimTime", unitMoveAnimTime...),
			regInts("MoveAnimFrame", unitMoveAnimFrame...),
			regInt("Index", 116),
			regInt("MovePhases", 117),
			regInt("MoveBeginPhases", 118),
			regInt("Width", 119),
			regInt("Height", 120),
			regInt("CenterX", 121),
			regInt("CenterY", 122),
			regInt("SelectionX1", 123),
			regInt("SelectionX2", 124),
			regInt("SelectionY1", 125),
			regInt("SelectionY2", 126),
			regInt("Parent", 7),
			regInt("BonePhases", 127),
			regInts("ShootOffset", unitShootOffset...),
			regInt("Flip", 128),
			regInt("Projectile", 129),
			regInt("ShootDelay", 130),
			regInt("TileSize", 131),
			regInt("IdlePhases", 132),
			regInts("IdleAnimTime", unitIdleAnimTime...),
			regInts("IdleAnimFrame", unitIdleAnimFrame...),
			regInt("Z", 133),
		),

		// The child: its own ID, its own Parent, one scalar of each kind and one
		// whole animation pair. Everything else — File and its sprite base among
		// them — arrives through the Parent.
		regDir("Unit2",
			regInt("ID", 102),
			regInt("Parent", 101),
			regStr("DescText", "unit-child-desc-text"),
			regInt("Width", 141),
			regInts("MoveAnimTime", childMoveAnimTime...),
			regInts("MoveAnimFrame", childMoveAnimFrame...),
		),
	)
}

// fullUnitWant is the expectation for Unit1, written out key by key rather than
// derived from unitKeys: a row of that table repointed at another field of the
// same kind must fail against this struct, and an expectation generated from the
// table would move with the row instead.
func fullUnitWant() UnitClass {
	return UnitClass{
		ID:              101,
		File:            3,
		DescText:        "unit-desc-text",
		Sound:           unitSound,
		InfoPicture:     "unit-info-picture",
		AttackAnimTime:  unitAttackAnimTime,
		AttackAnimFrame: unitAttackAnimFrame,
		AttackDelay:     110,
		InMapEditor:     111,
		Dying:           112,
		AttackPhases:    113,
		Palette:         114,
		DyingPhases:     115,
		MoveAnimTime:    unitMoveAnimTime,
		MoveAnimFrame:   unitMoveAnimFrame,
		Index:           116,
		MovePhases:      117,
		MoveBeginPhases: 118,
		Width:           119,
		Height:          120,
		CenterX:         121,
		CenterY:         122,
		SelectionX1:     123,
		SelectionX2:     124,
		SelectionY1:     125,
		SelectionY2:     126,
		Parent:          7,
		BonePhases:      127,
		ShootOffset:     unitShootOffset,
		Flip:            128,
		Projectile:      129,
		ShootDelay:      130,
		TileSize:        131,
		IdlePhases:      132,
		IdleAnimTime:    unitIdleAnimTime,
		IdleAnimFrame:   unitIdleAnimFrame,
		Z:               133,
	}
}

func TestUnitInventoryRoundTrip(t *testing.T) {
	cs := loadUnits(t, inventoryUnitsReg(t))
	want := fullUnitWant()
	checkRoundTrip(t, "unit", unitByID(t, cs, 101), &want)
}

func TestUnitInventoryRoundTripThroughAParent(t *testing.T) {
	cs := loadUnits(t, inventoryUnitsReg(t))

	want := fullUnitWant()
	want.ID = 102
	want.Parent = 101
	want.DescText = "unit-child-desc-text"
	want.Width = 141
	want.MoveAnimTime = childMoveAnimTime
	want.MoveAnimFrame = childMoveAnimFrame

	checkRoundTrip(t, "unit through a parent", unitByID(t, cs, 102), &want)
}

// --- objects: all 16 keys ----------------------------------------------------

var (
	objectAnimationTime  = []int32{610, 611}
	objectAnimationFrame = []int32{620, 621}
)

func inventoryObjectsReg(t *testing.T) *reg.Reg {
	t.Helper()
	files := []synth.RegNode{
		regStr("File0", "unreferenced-0"),
		regStr("File1", "unreferenced-1"),
		regStr("File2", `Obj\Sprite`),
	}
	return spriteObjectsReg(t, files,
		regDir("Object0", regInt("ID", 9)),
		regDir("Object1",
			regInt("ID", 201),
			regInt("File", 2),
			regStr("DescText", "object-desc-text"),
			regInt("InMapEditor", 210),
			regInt("Index", 211),
			regInt("Phases", 212),
			regInt("Width", 213),
			regInt("Height", 214),
			regInt("CenterX", 215),
			regInt("CenterY", 216),
			regInt("Parent", 9),
			regInt("DeadObject", 217),
			regInt("IconID", 218),
			regInts("AnimationTime", objectAnimationTime...),
			regInts("AnimationFrame", objectAnimationFrame...),
			regInt("FireObject", 219),
		),
	)
}

func TestObjectInventoryRoundTrip(t *testing.T) {
	cs, err := LoadObjectClasses(inventoryObjectsReg(t))
	if err != nil {
		t.Fatalf("LoadObjectClasses: %v", err)
	}
	got, ok := cs.ByID(201)
	if !ok {
		t.Fatal("ByID(201) missed")
	}

	want := ObjectClass{
		ID:             201,
		File:           2,
		DescText:       "object-desc-text",
		InMapEditor:    210,
		Index:          211,
		Phases:         212,
		Width:          213,
		Height:         214,
		CenterX:        215,
		CenterY:        216,
		Parent:         9,
		DeadObject:     217,
		IconID:         218,
		AnimationTime:  objectAnimationTime,
		AnimationFrame: objectAnimationFrame,
		FireObject:     219,
	}
	checkRoundTrip(t, "object", got, &want)
}

// --- structures: all 23 keys -------------------------------------------------

var (
	structureAnimTime  = []int32{710, 711, 712}
	structureAnimFrame = []int32{720, 721, 722}
)

// This registry has no [Files] table and no Parent: File is the sprite path
// itself, and every value below is the class's own.
//
// TileWidth and FullHeight are the two sentinels the contract constrains: a
// non-empty AnimMask's length must be their product, so 3 and 5 stand against a
// mask of 15 bytes. They still differ from each other, so a transposition of the
// two is visible here even though the product would not move.
func inventoryStructuresReg(t *testing.T) *reg.Reg {
	t.Helper()
	return spriteStructuresReg(t,
		regDir("Structure0",
			regInt("ID", 301),
			regStr("DescText", "structure-desc-text"),
			regStr("File", `Str\Path\Mixed`),
			regInt("TileWidth", 3),
			regInt("TileHeight", 311),
			regInt("FullHeight", 5),
			regInt("SelectionX1", 312),
			regInt("SelectionX2", 313),
			regInt("SelectionY1", 314),
			regInt("SelectionY2", 315),
			regInt("ShadowY", 316),
			regInt("Phases", 317),
			regStr("Picture", "structure-picture"),
			regStr("AnimMask", "mask-of-fifteen"),
			regInts("AnimTime", structureAnimTime...),
			regInts("AnimFrame", structureAnimFrame...),
			regInt("Indestructible", 318),
			regInt("IconID", 319),
			regInt("Usable", 320),
			regInt("Flat", 321),
			regInt("LightRadius", 322),
			regInt("LightPulse", 323),
			regInt("VariableSize", 324),
		),
	)
}

func TestStructureInventoryRoundTrip(t *testing.T) {
	cs, err := LoadStructureClasses(inventoryStructuresReg(t))
	if err != nil {
		t.Fatalf("LoadStructureClasses: %v", err)
	}
	got, ok := cs.ByID(301)
	if !ok {
		t.Fatal("ByID(301) missed")
	}

	want := StructureClass{
		ID:             301,
		DescText:       "structure-desc-text",
		File:           `Str\Path\Mixed`,
		TileWidth:      3,
		TileHeight:     311,
		FullHeight:     5,
		SelectionX1:    312,
		SelectionX2:    313,
		SelectionY1:    314,
		SelectionY2:    315,
		ShadowY:        316,
		Phases:         317,
		Picture:        "structure-picture",
		AnimMask:       "mask-of-fifteen",
		AnimTime:       structureAnimTime,
		AnimFrame:      structureAnimFrame,
		Indestructible: 318,
		IconID:         319,
		Usable:         320,
		Flat:           321,
		LightRadius:    322,
		LightPulse:     323,
		VariableSize:   324,
	}
	checkRoundTrip(t, "structure", got, &want)

	// File is a path here, not an index, and it is loaded verbatim — the stored
	// backslashes intact. The forward-slashed form is the sprite base's, which is
	// not a field and is pinned by sprite_test.go.
	if got.File != `Str\Path\Mixed` {
		t.Errorf("File = %q, want the stored path verbatim", got.File)
	}
}

// --- the three that are values and not errors --------------------------------

// AC-8's first half. None of these is a case Validation names, so a loader that
// rejected any of them would refuse data the contract takes — and no malformed
// fixture can show that.
func TestTheCasesThatAreValuesAndNotErrors(t *testing.T) {
	// Seven shipped object classes carry neither DescText nor Parent and are
	// otherwise complete: a missing DescText is not an error, and a class's
	// identity is its ID.
	t.Run("a class with no DescText", func(t *testing.T) {
		cs, err := LoadObjectClasses(objectsReg(t, 1,
			regDir("Object0", regInt("ID", 0), regInt("File", 0), regInt("Width", 42))))
		if err != nil {
			t.Fatalf("LoadObjectClasses: %v", err)
		}
		c, ok := cs.ByID(0)
		if !ok {
			t.Fatal("ByID(0) missed — a class with no DescText is still a class, by its ID")
		}
		if c.DescText != "" {
			t.Errorf("DescText = %q, want \"\"", c.DescText)
		}
		if c.Width != 42 {
			t.Errorf("Width = %d, want 42 — the rest of the class loads", c.Width)
		}
	})

	// Seven of the 155 shipped class paths name no archive entry at all — fire
	// variants whose art does not ship. Construction opens nothing and checks
	// nothing, so the path is a value like any other; there is no archive in this
	// test for it to be absent from, which is the point.
	t.Run("a sprite entry naming nothing that ships", func(t *testing.T) {
		cs, err := LoadObjectClasses(spriteObjectsReg(t,
			[]synth.RegNode{regStr("File0", `Fire\NoArt`)},
			regDir("Object0", regInt("ID", 0), regInt("File", 0))))
		if err != nil {
			t.Fatalf("LoadObjectClasses: %v — an absent sprite file is valid shipped data", err)
		}
		c, ok := cs.ByID(0)
		if !ok {
			t.Fatal("ByID(0) missed")
		}
		if got, want := c.SpritePath(), "objects/Fire/NoArt.256"; got != want {
			t.Errorf("SpritePath() = %q, want %q — a value, not an error", got, want)
		}
	})

	// Unknown keys are ignored, at either kind, and the class's own keys are
	// unaffected.
	t.Run("an unknown key is ignored", func(t *testing.T) {
		cs := loadUnits(t, unitsReg(t, 1,
			regDir("Unit0",
				regInt("ID", 5), regInt("File", 0),
				regInt("Mystery", 61), regStr("Legend", "unknown"),
				regInt("Width", 62))))
		c := unitByID(t, cs, 5)
		if c.Width != 62 {
			t.Errorf("Width = %d, want 62 — an unknown key beside it changes nothing", c.Width)
		}
		if len(cs.All()) != 1 {
			t.Errorf("len(All()) = %d, want 1", len(cs.All()))
		}
	})
}
