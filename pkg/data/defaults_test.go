package data

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// The absent-everywhere defaults, over synthetic registries (SC-12, AC-10).
//
// inherit_test.go pins what a class takes FROM ITS ANCESTORS. These pin the
// question with no ancestor in it: what a key set by no section on the whole
// chain resolves to. It is not zero — zero is a legal object ID, so a DeadObject
// nobody sets would name the class whose ID is 0 — and it is not one value
// either, InMapEditor's default being 0 where the rest are -1.
//
// Nothing here reads an install: internal/synth writes the .reg byte stream and
// pkg/formats/reg parses it back, as everywhere else in this package.

func objectByID(t *testing.T, cs *ObjectClasses, id int32) *ObjectClass {
	t.Helper()
	c, ok := cs.ByID(id)
	if !ok {
		t.Fatalf("ByID(%d) missed", id)
	}
	return c
}

func loadObjects(t *testing.T, r *reg.Reg) *ObjectClasses {
	t.Helper()
	cs, err := LoadObjectClasses(r)
	if err != nil {
		t.Fatalf("LoadObjectClasses: %v", err)
	}
	return cs
}

// SC-12 (AC-10). One parent sets three keys, its child sets none of them,
// and a third class sits on no chain at all. The child's three come from the
// parent; the third class's come from the default table, which is where the
// break lives: falling through to the Go zero reads 0, 0, 0 and cannot tell
// "nobody set DeadObject" from "DeadObject is Object0".
func TestAKeyNoSectionSetsTakesItsDefault(t *testing.T) {
	cs := loadObjects(t, objectsReg(t, 3,
		regDir("Object0", regInt("ID", 0), regInt("File", 0),
			regInt("DeadObject", 7), regInt("Width", 64), regInt("InMapEditor", 1)),
		regDir("Object1", regInt("ID", 1), regInt("File", 0), regInt("Parent", 0)),
		regDir("Object2", regInt("ID", 2), regInt("File", 0)),
	))

	child := objectByID(t, cs, 1)
	if child.DeadObject != 7 || child.Width != 64 || child.InMapEditor != 1 {
		t.Errorf("child: DeadObject/Width/InMapEditor = %d/%d/%d, want 7/64/1 — the parent sets all three",
			child.DeadObject, child.Width, child.InMapEditor)
	}

	loner := objectByID(t, cs, 2)
	if loner.DeadObject != -1 {
		t.Errorf("DeadObject = %d, want -1 — a class with no chain names no dead form, "+
			"and 0 would name the class whose ID is 0", loner.DeadObject)
	}
	if loner.Width != -1 {
		t.Errorf("Width = %d, want -1", loner.Width)
	}
	if loner.InMapEditor != 0 {
		t.Errorf("InMapEditor = %d, want 0 — this key's default IS zero, and the fix must leave it there",
			loner.InMapEditor)
	}
}

// The default must not reach a class that WRITES the value it defaults to, nor a
// class that inherits one: both are values some section states, and only the
// third case — set nowhere — is the default's.
func TestAWrittenZeroIsNotADefault(t *testing.T) {
	cs := loadObjects(t, objectsReg(t, 2,
		regDir("Object0", regInt("ID", 0), regInt("File", 0), regInt("DeadObject", 5)),
		regDir("Object1", regInt("ID", 1), regInt("File", 0), regInt("Parent", 0),
			regInt("DeadObject", 0)),
	))

	if got := objectByID(t, cs, 1).DeadObject; got != 0 {
		t.Errorf("DeadObject = %d, want 0 — a written 0 overrides the parent and is not the default", got)
	}
	if got := objectByID(t, cs, 0).DeadObject; got != 5 {
		t.Errorf("DeadObject = %d, want 5", got)
	}
}

// structures.reg has no default table: its per-key defaults are not decoded,
// so a key set nowhere keeps the Go zero — and this registry does not
// inherit, so "set nowhere" is the class's own section omitting it. That is
// the scope limit written down, not a defect — and a test, because the
// alternative is discovering later that a decoded table was quietly
// generalised.
func TestStructuresKeepTheZero(t *testing.T) {
	cs, err := LoadStructureClasses(spriteStructuresReg(t,
		regDir("Structure0", regInt("ID", 1), regStr("File", "keep")),
	))
	if err != nil {
		t.Fatalf("LoadStructureClasses: %v", err)
	}
	s, ok := cs.ByID(1)
	if !ok {
		t.Fatal("ByID(1) missed")
	}
	if s.Phases != 0 || s.ShadowY != 0 || s.IconID != 0 {
		t.Errorf("structure Phases/ShadowY/IconID = %d/%d/%d, want 0/0/0 — structures.reg has no decoded default table",
			s.Phases, s.ShadowY, s.IconID)
	}
}

// Every row of a default table must name a kindInt row of its own registry's
// key table. defaultsFor skips a row it cannot resolve, which is what keeps a
// rule silent over a registry that has no such key — and would also swallow a
// misspelling, so the bijection is asserted rather than trusted.
func TestEveryDefaultRowNamesAnIntKeyOfItsOwnTable(t *testing.T) {
	seen := make(map[string]bool, len(objectDefaults))
	for _, rule := range objectDefaults {
		if seen[rule.key] {
			t.Errorf("%s: two default rows for one key", rule.key)
		}
		seen[rule.key] = true

		k := rowIndex(objectKeys, rule.key)
		if k < 0 {
			t.Errorf("%s: no such key in the objects key table", rule.key)
			continue
		}
		if objectKeys[k].kind != kindInt {
			t.Errorf("%s: key table kind is %s, but only an int key has a default", rule.key, objectKeys[k].kind)
		}
	}

	// The units table, pinned row by row (0024 SC-1): seventeen -1s, the eight
	// decoded 0s, TileSize 1 — and NO row for InMapEditor, the key the engine's
	// unit record has no field for. Every value here is a hand-written literal.
	wantUnit := map[string]int32{
		"ID": -1, "File": -1,
		"AttackPhases": -1, "DyingPhases": -1, "MovePhases": -1,
		"MoveBeginPhases": -1, "BonePhases": -1, "Index": -1,
		"Width": -1, "Height": -1, "CenterX": -1, "CenterY": -1,
		"SelectionX1": -1, "SelectionX2": -1, "SelectionY1": -1, "SelectionY2": -1,
		"Parent":     -1,
		"IdlePhases": 0, "Dying": 0, "Palette": 0, "Projectile": 0,
		"ShootDelay": 0, "AttackDelay": 0, "Z": 0, "Flip": 0,
		"TileSize": 1,
	}
	if got, want := len(unitDefaults), len(wantUnit); got != want {
		t.Errorf("unit default rows = %d, want %d", got, want)
	}
	seenUnit := make(map[string]bool, len(unitDefaults))
	for _, rule := range unitDefaults {
		if seenUnit[rule.key] {
			t.Errorf("%s: two unit default rows for one key", rule.key)
		}
		seenUnit[rule.key] = true

		k := rowIndex(unitKeys, rule.key)
		if k < 0 {
			t.Errorf("%s: no such key in the units key table", rule.key)
			continue
		}
		if unitKeys[k].kind != kindInt {
			t.Errorf("%s: key table kind is %s, but only an int key has a default", rule.key, unitKeys[k].kind)
		}
		want, ok := wantUnit[rule.key]
		if !ok {
			t.Errorf("%s: a unit default row the contract does not state", rule.key)
			continue
		}
		if rule.value != want {
			t.Errorf("%s: unit default = %d, want %d", rule.key, rule.value, want)
		}
		if rule.noInherit {
			t.Errorf("%s: noInherit, but no units.reg key is — File inherits here, opposite to objects.reg", rule.key)
		}
	}

	// The one table that must stay empty, so its registry keeps the zero.
	if len(structureDefaults) != 0 {
		t.Errorf("structure default rows = %d, want 0 — this registry's defaults are not decoded",
			len(structureDefaults))
	}

	// defaultsFor answers 0 and false for a row with no rule, which is what
	// makes a key with no row — objects' IconID, units' InMapEditor — keep the
	// Go zero.
	values, noInherit := defaultsFor(objectKeys, objectDefaults)
	if got := values[rowIndex(objectKeys, "IconID")]; got != 0 {
		t.Errorf("IconID default = %d, want 0 — no rule names it, and none is decoded", got)
	}
	for k, row := range objectKeys {
		if noInherit[k] && row.name != "File" {
			t.Errorf("%s does not inherit, but File is the only key that does not", row.name)
		}
	}
	unitValues, unitNoInherit := defaultsFor(unitKeys, unitDefaults)
	if got := unitValues[rowIndex(unitKeys, "InMapEditor")]; got != 0 {
		t.Errorf("InMapEditor default = %d, want 0 — no row names it, the unit record having no field for it", got)
	}
	for k := range unitKeys {
		if unitNoInherit[k] {
			t.Errorf("unit %s does not inherit, but every units.reg key inherits", unitKeys[k].name)
		}
	}
}

func TestAUnitKeyNoSectionSetsTakesItsDefault(t *testing.T) {
	units := loadUnits(t, unitsReg(t, 3,
		regDir("Unit0", regInt("ID", 1)),
		regDir("Unit1", regInt("ID", 2), regInt("Parent", 1)),
		regDir("Unit2"),
	))

	all := units.All()
	if len(all) != 3 {
		t.Fatalf("All() = %d classes, want 3", len(all))
	}
	loner := all[2]

	for _, tc := range []struct {
		key       string
		got, want int32
	}{
		{"ID", loner.ID, -1},
		{"File", loner.File, -1},
		{"AttackPhases", loner.AttackPhases, -1},
		{"DyingPhases", loner.DyingPhases, -1},
		{"MovePhases", loner.MovePhases, -1},
		{"MoveBeginPhases", loner.MoveBeginPhases, -1},
		{"BonePhases", loner.BonePhases, -1},
		{"Index", loner.Index, -1},
		{"Width", loner.Width, -1},
		{"Height", loner.Height, -1},
		{"CenterX", loner.CenterX, -1},
		{"CenterY", loner.CenterY, -1},
		{"SelectionX1", loner.SelectionX1, -1},
		{"SelectionX2", loner.SelectionX2, -1},
		{"SelectionY1", loner.SelectionY1, -1},
		{"SelectionY2", loner.SelectionY2, -1},
		{"Parent", loner.Parent, -1},
		{"IdlePhases", loner.IdlePhases, 0},
		{"Dying", loner.Dying, 0},
		{"Palette", loner.Palette, 0},
		{"Projectile", loner.Projectile, 0},
		{"ShootDelay", loner.ShootDelay, 0},
		{"AttackDelay", loner.AttackDelay, 0},
		{"Z", loner.Z, 0},
		{"Flip", loner.Flip, 0},
		{"TileSize", loner.TileSize, 1},
		{"InMapEditor", loner.InMapEditor, 0},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d — the key is set by no section at all", tc.key, tc.got, tc.want)
		}
	}

	// The chain: the child's Width is set neither by its own section nor by its
	// ancestor's, so the default arrives through the chain intact.
	child := unitByID(t, units, 2)
	if child.Width != -1 || child.TileSize != 1 || child.InMapEditor != 0 {
		t.Errorf("child Width/TileSize/InMapEditor = %d/%d/%d, want -1/1/0 — an ancestor that sets nothing hands nothing down",
			child.Width, child.TileSize, child.InMapEditor)
	}

	// No section on the chain sets File. The FIELD reads the -1 default, and
	// the sprite path is empty: the base is resolved from the eff node, which is
	// nil, so File's -1 is never fed to the [Files] bound and a class resolving
	// no File resolves no sprite path.
	if child.File != -1 {
		t.Errorf("child File = %d, want -1", child.File)
	}
	if got := child.SpritePath(); got != "" {
		t.Errorf("child SpritePath() = %q, want empty — a chain that never sets File names no sprite", got)
	}
	if got := loner.SpritePath(); got != "" {
		t.Errorf("loner SpritePath() = %q, want empty", got)
	}
}

// SC-13 (AC-11). File is the one objects.reg key whose default is unconditional:
// a class whose own section omits it takes -1 whatever its ancestors hold, where
// the same shape in units.reg inherits. No shipped class can show this — every
// object class writes its own File — so this fixture is the only witness there
// is, and the two registries are loaded side by side because the difference
// between them is the whole claim.
func TestFileDoesNotInheritInObjectsButDoesInUnits(t *testing.T) {
	files := []synth.RegNode{regStr("File0", "first"), regStr("File1", `Art\Second`)}

	objects := loadObjects(t, spriteObjectsReg(t, files,
		regDir("Object0", regInt("ID", 0), regInt("File", 1), regInt("Width", 9)),
		regDir("Object1", regInt("ID", 1), regInt("Parent", 0)),
	))

	parent := objectByID(t, objects, 0)
	if got, want := parent.SpritePath(), "objects/Art/Second.256"; got != want {
		t.Fatalf("parent SpritePath() = %q, want %q", got, want)
	}

	child := objectByID(t, objects, 1)
	if child.File != -1 {
		t.Errorf("object child: File = %d, want -1 — File does not inherit in this registry", child.File)
	}
	if child.Width != 9 {
		t.Errorf("object child: Width = %d, want 9 — every other scalar still inherits", child.Width)
	}
	if sprite, overlay := child.SpritePath(), child.OverlayPath(); sprite != "" || overlay != "" {
		t.Errorf("object child paths = %q/%q, want empty — a File at its default names no sprite",
			sprite, overlay)
	}

	units := loadUnits(t, spriteUnitsReg(t, files,
		regDir("Unit0", regInt("ID", 10), regInt("File", 1)),
		regDir("Unit1", regInt("ID", 11), regInt("Parent", 10)),
	))
	unitChild, ok := units.ByID(11)
	if !ok {
		t.Fatal("ByID(11) missed")
	}
	if unitChild.File != 1 {
		t.Errorf("unit child: File = %d, want 1 — units.reg inherits File like any other scalar", unitChild.File)
	}
	if got, want := unitChild.SpritePath(), "units/Art/Second.256"; got != want {
		t.Errorf("unit child SpritePath() = %q, want %q", got, want)
	}
}
