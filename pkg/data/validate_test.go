package data

import (
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// Validation, case by case: one fixture per malformed case the contract
// names, counting the Parent family as four and the two [Files] faults as
// two (SC-5, AC-5). Each must yield a NIL collection and an error carrying
// the section it happened in and, where the fault has one, the key — and
// none may panic, a panic here failing the run as a panic.
//
// Every fixture is malformed in EXACTLY ONE WAY, so the check that fires is the
// one the row names rather than whichever runs first. Fixtures are synthetic .reg
// byte streams built by internal/synth and parsed back by pkg/formats/reg;
// nothing here reads an install.

// rawUnits builds a units registry from an explicit [Global] and [Files] so a
// fixture can put the fault in either. The shared builders in inherit_test.go
// and sprite_test.go keep both well-formed on purpose, which is what makes them
// useless here.
func rawUnits(t *testing.T, global, files []synth.RegNode, sections ...synth.RegNode) *reg.Reg {
	t.Helper()
	return parseReg(t, append([]synth.RegNode{
		regDir("Global", global...),
		regDir("Files", files...),
	}, sections...))
}

// The three loaders reduced to one shape: whether the collection came back nil,
// and the error. Each is called at its own type because a typed nil pointer put
// into an interface is not a nil interface, so the emptiness has to be decided
// where the type is still known.
func unitsFail(r *reg.Reg) (bool, error) {
	cs, err := LoadUnitClasses(r)
	return cs == nil, err
}

func objectsFail(r *reg.Reg) (bool, error) {
	cs, err := LoadObjectClasses(r)
	return cs == nil, err
}

func structuresFail(r *reg.Reg) (bool, error) {
	cs, err := LoadStructureClasses(r)
	return cs == nil, err
}

func TestEveryMalformedCaseIsRejected(t *testing.T) {
	for _, tc := range []struct {
		label   string
		section string // the section the error must name
		key     string // the key it must name; "" where the fault has none
		load    func(*testing.T) (bool, error)
	}{
		// --- a missing or non-int [Global] count ---------------------------
		{
			label: "the class count is missing", section: "Global", key: "UnitCount",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(rawUnits(t,
					[]synth.RegNode{regInt("FileCount", 1)},
					[]synth.RegNode{regStr("File0", "unit")},
					regDir("Unit0", regInt("ID", 1), regInt("File", 0))))
			},
		},
		{
			label: "the class count is not an int", section: "Global", key: "UnitCount",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(rawUnits(t,
					[]synth.RegNode{regStr("UnitCount", "1"), regInt("FileCount", 1)},
					[]synth.RegNode{regStr("File0", "unit")},
					regDir("Unit0", regInt("ID", 1), regInt("File", 0))))
			},
		},
		{
			// FileCount is a [Global] count of the same standing: it is the
			// bound File's domain is stated against, so a registry that has a
			// [Files] table and no readable FileCount cannot have that bound
			// evaluated at all.
			label: "FileCount is missing", section: "Global", key: "FileCount",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(rawUnits(t,
					[]synth.RegNode{regInt("UnitCount", 1)},
					[]synth.RegNode{regStr("File0", "unit")},
					regDir("Unit0", regInt("ID", 1), regInt("File", 0))))
			},
		},
		{
			label: "FileCount is not an int", section: "Global", key: "FileCount",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(rawUnits(t,
					[]synth.RegNode{regInt("UnitCount", 1), regStr("FileCount", "1")},
					[]synth.RegNode{regStr("File0", "unit")},
					regDir("Unit0", regInt("ID", 1), regInt("File", 0))))
			},
		},

		// --- a missing dense section ---------------------------------------
		{
			// The count says two, the tree holds Unit0 alone. The error names
			// the index where the density breaks, not the count.
			label: "a dense section is missing", section: "Unit1",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 2,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0))))
			},
		},

		// --- a duplicate ID -------------------------------------------------
		{
			label: "two classes carry one ID", section: "Unit1", key: "ID",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 2,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0)),
					regDir("Unit1", regInt("ID", 1), regInt("File", 0))))
			},
		},

		// --- the Parent family, four cases ----------------------------------
		{
			label: "Parent names an ID no class holds", section: "Unit1", key: "Parent",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 2,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0)),
					regDir("Unit1", regInt("ID", 2), regInt("File", 0), regInt("Parent", 9))))
			},
		},
		{
			// Forward: the named parent's section index is not below the
			// child's, so the engine would read an unpopulated slot.
			label: "Parent is forward", section: "Unit0", key: "Parent",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 2,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0), regInt("Parent", 2)),
					regDir("Unit1", regInt("ID", 2), regInt("File", 0))))
			},
		},
		{
			// A cycle is caught as the forward edge every cycle contains: there
			// is no cycle detector, and none is needed.
			label: "Parent is cyclic", section: "Unit0", key: "Parent",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 2,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0), regInt("Parent", 2)),
					regDir("Unit1", regInt("ID", 2), regInt("File", 0), regInt("Parent", 1))))
			},
		},
		{
			label: "Parent names the class's own ID", section: "Unit0", key: "Parent",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 1,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0), regInt("Parent", 1))))
			},
		},
		{
			// structures.reg does not inherit at all: Parent is not an unknown
			// key there to be ignored, it is malformed.
			label: "Parent on a structure", section: "Structure1", key: "Parent",
			load: func(t *testing.T) (bool, error) {
				return structuresFail(spriteStructuresReg(t,
					regDir("Structure0", regInt("ID", 1), regStr("File", "one")),
					regDir("Structure1", regInt("ID", 2), regStr("File", "two"), regInt("Parent", 1))))
			},
		},

		// --- a File index outside [0, FileCount) ------------------------------
		{
			label: "File indexes past FileCount", section: "Unit0", key: "File",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(rawUnits(t,
					[]synth.RegNode{regInt("UnitCount", 1), regInt("FileCount", 1)},
					[]synth.RegNode{regStr("File0", "unit")},
					regDir("Unit0", regInt("ID", 1), regInt("File", 1))))
			},
		},
		{
			label: "File is negative", section: "Unit0", key: "File",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(rawUnits(t,
					[]synth.RegNode{regInt("UnitCount", 1), regInt("FileCount", 1)},
					[]synth.RegNode{regStr("File0", "unit")},
					regDir("Unit0", regInt("ID", 1), regInt("File", -1))))
			},
		},

		// --- a missing or empty [Files] entry at a referenced index ----------
		{
			// Inside the bound and behind nothing: FileCount says two entries,
			// the table holds one.
			label: "[Files] has no entry at the referenced index", section: "Unit0", key: "File",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(rawUnits(t,
					[]synth.RegNode{regInt("UnitCount", 1), regInt("FileCount", 2)},
					[]synth.RegNode{regStr("File0", "unit")},
					regDir("Unit0", regInt("ID", 1), regInt("File", 1))))
			},
		},
		{
			label: "the referenced [Files] entry is empty", section: "Unit0", key: "File",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(rawUnits(t,
					[]synth.RegNode{regInt("UnitCount", 1), regInt("FileCount", 1)},
					[]synth.RegNode{regStr("File0", "")},
					regDir("Unit0", regInt("ID", 1), regInt("File", 0))))
			},
		},

		// --- a known key of the wrong kind -----------------------------------
		{
			label: "a known scalar key at another type", section: "Unit0", key: "Width",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 1,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0), regStr("Width", "wide"))))
			},
		},
		{
			// The one string that is not this fault is the EMPTY one, the
			// editor's "none" marker, which stands as an array of length 0.
			label: "a non-empty string where an array is expected", section: "Unit0", key: "Sound",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 1,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0), regStr("Sound", "x"))))
			},
		},

		// --- ShootOffset of a length other than 16 ---------------------------
		{
			label: "ShootOffset is not 16 long", section: "Unit0", key: "ShootOffset",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 1,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0),
						regInts("ShootOffset", 1, 2, 3, 4))))
			},
		},

		// --- a non-empty AnimMask whose length is not TileWidth x FullHeight --
		{
			label: "AnimMask is not TileWidth x FullHeight long", section: "Structure0", key: "AnimMask",
			load: func(t *testing.T) (bool, error) {
				return structuresFail(spriteStructuresReg(t,
					regDir("Structure0", regInt("ID", 1), regStr("File", "one"),
						regInt("TileWidth", 2), regInt("FullHeight", 3),
						regStr("AnimMask", "xxxxx"))))
			},
		},

		// --- a paired animation key without its partner, or of another length -
		{
			label: "an animation key without its partner", section: "Unit0", key: "AttackAnimTime",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 1,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0),
						regInts("AttackAnimTime", 1, 2, 3))))
			},
		},
		{
			label: "an animation pair at two lengths", section: "Unit0", key: "MoveAnimTime",
			load: func(t *testing.T) (bool, error) {
				return unitsFail(unitsReg(t, 1,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0),
						regInts("MoveAnimTime", 1, 2, 3), regInts("MoveAnimFrame", 4, 5))))
			},
		},
		{
			// The reason the rule reads RESOLVED values: each node here is
			// well-formed and the parent loads clean, but the child writes one
			// half of the pair at three and inherits the other at seven.
			label: "an animation pair inherited asymmetrically", section: "Unit1", key: "IdleAnimTime",
			load: func(t *testing.T) (bool, error) {
				seven := []int32{1, 2, 3, 4, 5, 6, 7}
				return unitsFail(unitsReg(t, 2,
					regDir("Unit0", regInt("ID", 1), regInt("File", 0),
						regInts("IdleAnimTime", seven...), regInts("IdleAnimFrame", seven...)),
					regDir("Unit1", regInt("ID", 2), regInt("File", 0), regInt("Parent", 1),
						regInts("IdleAnimTime", 8, 9, 10))))
			},
		},
		{
			label: "an object animation key without its partner", section: "Object0", key: "AnimationTime",
			load: func(t *testing.T) (bool, error) {
				return objectsFail(objectsReg(t, 1,
					regDir("Object0", regInt("ID", 0), regInt("File", 0),
						regInts("AnimationTime", 1, 2))))
			},
		},
		{
			label: "a structure animation pair at two lengths", section: "Structure0", key: "AnimTime",
			load: func(t *testing.T) (bool, error) {
				return structuresFail(spriteStructuresReg(t,
					regDir("Structure0", regInt("ID", 1), regStr("File", "one"),
						regInts("AnimTime", 1, 2, 3), regInts("AnimFrame", 4, 5))))
			},
		},
	} {
		t.Run(tc.label, func(t *testing.T) {
			gotNil, err := tc.load(t)
			if err == nil {
				t.Fatalf("loaded clean, want an error naming %s", tc.section)
			}
			if !gotNil {
				t.Errorf("a collection came back beside the error %v — a malformed "+
					"registry yields nil, never a partial collection", err)
			}
			want := tc.section + ": "
			if tc.key != "" {
				want += tc.key + ": "
			}
			if !strings.HasPrefix(err.Error(), want) {
				t.Errorf("error is %q, want it to start %q — the shape is "+
					"<section>: <key>: <what>", err.Error(), want)
			}
		})
	}
}

func TestTheTwoReadingsThatMustLoad(t *testing.T) {
	cs := loadUnits(t, unitsReg(t, 2,
		// ShootOffset absent with no ancestor to take one from. Only a
		// non-nil resolved value of a length other than 16 is malformed.
		regDir("Unit0", regInt("ID", 1), regInt("File", 0)),
		// Sound at four. "Always length 5" is an inventory MEASUREMENT, not a
		// rule the contract states, so nothing checks it.
		regDir("Unit1", regInt("ID", 2), regInt("File", 0), regInts("Sound", 7, 8, 9, 10)),
	))

	if got := unitByID(t, cs, 1).ShootOffset; got != nil {
		t.Errorf("ShootOffset = %v, want nil — an absent array with no ancestor resolves to nil", got)
	}
	if got := unitByID(t, cs, 2).Sound; !slices.Equal(got, []int32{7, 8, 9, 10}) {
		t.Errorf("Sound = %v, want [7 8 9 10] verbatim", got)
	}
}

// The accept side of the three length rules: the values Validation allows must
// pass them, inherited ones included. A rule written one comparison too strict
// fails here and nowhere else.
func TestTheLengthRulesAcceptWhatTheContractAllows(t *testing.T) {
	shoot := make([]int32, 16)
	for i := range shoot {
		shoot[i] = int32(i)
	}

	units := loadUnits(t, unitsReg(t, 2,
		regDir("Unit0", regInt("ID", 1), regInt("File", 0),
			regInts("ShootOffset", shoot...),
			regInts("AttackAnimTime", 1, 2, 3), regInts("AttackAnimFrame", 4, 5, 6)),
		// Both halves of the pair, and ShootOffset, arrive by inheritance at
		// their parent's lengths.
		regDir("Unit1", regInt("ID", 2), regInt("File", 0), regInt("Parent", 1)),
	))
	child := unitByID(t, units, 2)
	if got := len(child.ShootOffset); got != 16 {
		t.Errorf("inherited ShootOffset has length %d, want 16", got)
	}
	if got, want := len(child.AttackAnimTime), len(child.AttackAnimFrame); got != want || got != 3 {
		t.Errorf("inherited pair is %d and %d long, want 3 and 3", got, want)
	}

	structures, err := LoadStructureClasses(spriteStructuresReg(t,
		// A mask at exactly TileWidth * FullHeight.
		regDir("Structure0", regInt("ID", 1), regStr("File", "one"),
			regInt("TileWidth", 2), regInt("FullHeight", 3), regStr("AnimMask", "xxxxxx"),
			regInts("AnimTime", 1, 2), regInts("AnimFrame", 3, 4)),
		// The shape 52 of the 66 shipped structures carry: all three animation
		// keys stored as the empty-string sentinel, geometry set regardless.
		regDir("Structure1", regInt("ID", 2), regStr("File", "two"),
			regInt("TileWidth", 2), regInt("FullHeight", 3), regStr("AnimMask", ""),
			regStr("AnimTime", ""), regStr("AnimFrame", "")),
	))
	if err != nil {
		t.Fatalf("LoadStructureClasses: %v", err)
	}
	full, ok := structures.ByID(1)
	if !ok {
		t.Fatal("structures.ByID(1) missed")
	}
	if got := full.AnimMask; got != "xxxxxx" {
		t.Errorf("AnimMask = %q, want %q verbatim", got, "xxxxxx")
	}
	empty, ok := structures.ByID(2)
	if !ok {
		t.Fatal("structures.ByID(2) missed")
	}
	if empty.AnimMask != "" || empty.AnimTime != nil || empty.AnimFrame != nil {
		t.Errorf("the empty-sentinel structure loaded as %q / %v / %v, want \"\" and two nils",
			empty.AnimMask, empty.AnimTime, empty.AnimFrame)
	}
}

// ruleKeys is every key a rule set names, at the kind the rule reads it at: an
// array for a length, a str for the mask, an int for each of its dimensions.
func ruleKeys(l lengths) map[string]keyKind {
	out := make(map[string]keyKind)
	for _, r := range l.fixed {
		out[r.key] = kindArray
	}
	for _, r := range l.pairs {
		out[r.time] = kindArray
		out[r.frame] = kindArray
	}
	if l.mask.key != "" {
		out[l.mask.key] = kindStr
		out[l.mask.width] = kindInt
		out[l.mask.height] = kindInt
	}
	return out
}

// A length rule naming a key its registry's table does not carry yields no
// check at all — silently, since that is also how a rule stays out of a registry
// whose inventory lacks the key. So a misspelled key would not fail a fixture,
// it would leave one passing for the wrong reason; and a key named at the wrong
// kind would read a Go zero and compare against nothing. Both are caught here.
func TestEveryLengthRuleNamesAKeyOfItsOwnTable(t *testing.T) {
	for _, tc := range []struct {
		label  string
		keys   map[string]keyKind
		kindOf func(string) (keyKind, bool)
	}{
		{"units", ruleKeys(unitLengths), func(n string) (keyKind, bool) {
			i := rowIndex(unitKeys, n)
			if i < 0 {
				return 0, false
			}
			return unitKeys[i].kind, true
		}},
		{"objects", ruleKeys(objectLengths), func(n string) (keyKind, bool) {
			i := rowIndex(objectKeys, n)
			if i < 0 {
				return 0, false
			}
			return objectKeys[i].kind, true
		}},
		{"structures", ruleKeys(structureLengths), func(n string) (keyKind, bool) {
			i := rowIndex(structureKeys, n)
			if i < 0 {
				return 0, false
			}
			return structureKeys[i].kind, true
		}},
	} {
		if len(tc.keys) == 0 {
			t.Errorf("%s: no length rule at all", tc.label)
		}
		for name, want := range tc.keys {
			got, ok := tc.kindOf(name)
			if !ok {
				t.Errorf("%s: a length rule names %q, which is not a key of this registry — "+
					"the rule would be skipped, not applied", tc.label, name)
				continue
			}
			if got != want {
				t.Errorf("%s: a length rule reads %q as %v, but its key table holds it at %v",
					tc.label, name, want, got)
			}
		}
	}
}
