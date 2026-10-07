package data

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// The sprite paths, over synthetic registries: internal/synth writes the .reg
// byte stream, pkg/formats/reg parses it back, and the loader only ever sees a
// tree it could have got from a real one. Nothing here opens an archive, and
// none exists to open — which is the point, the paths being pure strings.
//
// That no exported field was added for base is not asserted here: keys_test.go's
// bijection counts a struct's exported fields against its measured inventory, so
// an exported base would fail it on both counts.

// spriteUnitsReg and spriteObjectsReg put the caller's own [Files] table behind
// the classes, which the fixtures in inherit_test.go do not: the stored path is
// the subject here. FileCount follows the table, so an index inside it is inside
// the bound too.
func spriteUnitsReg(t *testing.T, files []synth.RegNode, sections ...synth.RegNode) *reg.Reg {
	t.Helper()
	return parseReg(t, append([]synth.RegNode{
		regDir("Global", regInt("UnitCount", int32(len(sections))), regInt("FileCount", int32(len(files)))),
		regDir("Files", files...),
	}, sections...))
}

func spriteObjectsReg(t *testing.T, files []synth.RegNode, sections ...synth.RegNode) *reg.Reg {
	t.Helper()
	return parseReg(t, append([]synth.RegNode{
		regDir("Global", regInt("ObjectCount", int32(len(sections))), regInt("FileCount", int32(len(files)))),
		regDir("Files", files...),
	}, sections...))
}

// spriteStructuresReg has no [Files] table and no FileCount: each structure
// carries its path itself.
func spriteStructuresReg(t *testing.T, sections ...synth.RegNode) *reg.Reg {
	t.Helper()
	return parseReg(t, append([]synth.RegNode{
		regDir("Global", regInt("Count", int32(len(sections)))),
	}, sections...))
}

// spriter is what all three class types are here: the pair of methods, so one
// table covers the three registries and a type missing either fails to compile.
type spriter interface {
	SpritePath() string
	OverlayPath() string
}

func loadOneUnit(t *testing.T, r *reg.Reg) *UnitClass {
	t.Helper()
	return unitByID(t, loadUnits(t, r), 1)
}

// SC-7 (AC-7). A unit, an object and a structure, each on a stored path
// carrying several backslashes and mixed case, compared BYTE FOR BYTE: the
// prefix, the slash direction, the case, the ".256" and the "b" before it.
//
// Byte-for-byte is what catches a transformation that looks equivalent —
// lowercasing (the archive folds case on lookup; this layer must not),
// filepath.ToSlash (a no-op on a Unix host, so a green Windows run would prove
// nothing), and filepath.Clean or path.Clean, which the last row pins on every
// host by carrying a doubled separator and a "." element that cleaning eats.
func TestSpritePathsAreExactStrings(t *testing.T) {
	const mixed = `Fix\Ture\MiXeD\CasE`
	const messy = `Kee\\p\.\As\Is`

	// The referenced index is never 0: an implementation reading "the first
	// entry" rather than the class's own File would pass at index 0.
	unitFiles := []synth.RegNode{regStr("File0", "unused"), regStr("File1", mixed)}
	objectFiles := []synth.RegNode{
		regStr("File0", "unused"), regStr("File1", "unused"), regStr("File2", mixed),
	}

	unit := loadOneUnit(t, spriteUnitsReg(t, unitFiles,
		regDir("Unit0", regInt("ID", 1), regInt("File", 1))))

	objects, err := LoadObjectClasses(spriteObjectsReg(t, objectFiles,
		regDir("Object0", regInt("ID", 1), regInt("File", 2))))
	if err != nil {
		t.Fatalf("LoadObjectClasses: %v", err)
	}
	object, ok := objects.ByID(1)
	if !ok {
		t.Fatal("objects.ByID(1) missed")
	}

	structures, err := LoadStructureClasses(spriteStructuresReg(t,
		regDir("Structure0", regInt("ID", 1), regStr("File", mixed))))
	if err != nil {
		t.Fatalf("LoadStructureClasses: %v", err)
	}
	structure, ok := structures.ByID(1)
	if !ok {
		t.Fatal("structures.ByID(1) missed")
	}

	uncleaned := loadOneUnit(t, spriteUnitsReg(t,
		[]synth.RegNode{regStr("File0", "unused"), regStr("File1", messy)},
		regDir("Unit0", regInt("ID", 1), regInt("File", 1))))

	for _, tc := range []struct {
		label           string
		c               spriter
		sprite, overlay string
	}{
		{"unit", unit, "units/Fix/Ture/MiXeD/CasE.256", "units/Fix/Ture/MiXeD/CasEb.256"},
		{"object", object, "objects/Fix/Ture/MiXeD/CasE.256", "objects/Fix/Ture/MiXeD/CasEb.256"},
		{"structure", structure, "structures/Fix/Ture/MiXeD/CasE.256", "structures/Fix/Ture/MiXeD/CasEb.256"},
		{"nothing is cleaned", uncleaned, "units/Kee//p/./As/Is.256", "units/Kee//p/./As/Isb.256"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			if got := tc.c.SpritePath(); got != tc.sprite {
				t.Errorf("SpritePath() = %q, want %q", got, tc.sprite)
			}
			if got := tc.c.OverlayPath(); got != tc.overlay {
				t.Errorf("OverlayPath() = %q, want %q — the sprite path with a b "+
					"inserted before the extension, not a second lookup", got, tc.overlay)
			}
		})
	}
}

func TestSpriteBaseUsesTheResolvedFile(t *testing.T) {
	files := []synth.RegNode{regStr("File0", "unused"), regStr("File1", `Par\Ent`)}
	cs := loadUnits(t, spriteUnitsReg(t, files,
		regDir("Unit0", regInt("ID", 1), regInt("File", 1)),
		regDir("Unit1", regInt("ID", 2), regInt("Parent", 1)), // File inherited
	))

	const want = "units/Par/Ent.256"
	if got := unitByID(t, cs, 2).SpritePath(); got != want {
		t.Errorf("inherited File: SpritePath() = %q, want %q", got, want)
	}
}

func TestSpritePathEmptyWhenNoFileResolves(t *testing.T) {
	units := loadUnits(t, spriteUnitsReg(t, []synth.RegNode{regStr("File0", "unused")},
		regDir("Unit0", regInt("ID", 1)),                      // no File at all
		regDir("Unit1", regInt("ID", 2), regInt("Parent", 1)), // and none to inherit
	))

	// A structure's File is the path itself, so the empty string is the same
	// case: "structures/.256" would be a path this class never had.
	structures, err := LoadStructureClasses(spriteStructuresReg(t,
		regDir("Structure0", regInt("ID", 1)),
		regDir("Structure1", regInt("ID", 2), regStr("File", "")),
	))
	if err != nil {
		t.Fatalf("LoadStructureClasses: %v", err)
	}

	cases := []struct {
		label string
		c     spriter
	}{
		{"unit with no File", unitByID(t, units, 1)},
		{"unit inheriting an absent File", unitByID(t, units, 2)},
	}
	for _, id := range []int32{1, 2} {
		c, ok := structures.ByID(id)
		if !ok {
			t.Fatalf("structures.ByID(%d) missed", id)
		}
		cases = append(cases, struct {
			label string
			c     spriter
		}{"structure with an absent or empty File", c})
	}

	for _, tc := range cases {
		if got := tc.c.SpritePath(); got != "" {
			t.Errorf("%s: SpritePath() = %q, want \"\"", tc.label, got)
		}
		if got := tc.c.OverlayPath(); got != "" {
			t.Errorf("%s: OverlayPath() = %q, want \"\"", tc.label, got)
		}
	}
}

// AC-3. The per-tier colour-table address, BYTE FOR BYTE like the two above and
// for the same reasons: the directory rule, the missing digit at tier 1, the
// preserved case, and that nothing is cleaned. The last row is the case a
// directory rule gets wrong quietly — a base with no separator at all, which
// addresses a container's root.
func TestPalettePathsAreExactStrings(t *testing.T) {
	const mixed = `Fix\Ture\MiXeD\CasE`
	unit := loadOneUnit(t, spriteUnitsReg(t,
		[]synth.RegNode{regStr("File0", "unused"), regStr("File1", mixed)},
		regDir("Unit0", regInt("ID", 1), regInt("File", 1))))

	for tier, want := range map[int]string{
		1: "units/Fix/Ture/MiXeD/palette.pal",
		2: "units/Fix/Ture/MiXeD/palette2.pal",
		3: "units/Fix/Ture/MiXeD/palette3.pal",
		4: "units/Fix/Ture/MiXeD/palette4.pal",
		// Past the count is still a well-formed address, as a class naming art
		// that does not ship still has a well-formed sprite address.
		5: "units/Fix/Ture/MiXeD/palette5.pal",
	} {
		if got := unit.PalettePath(tier); got != want {
			t.Errorf("PalettePath(%d) = %q, want %q", tier, got, want)
		}
	}

	// A base with no separator: the directory is empty and the bare name is the
	// answer. The prefix the loader adds always carries one, so this is reached
	// through the private formatter rather than through a registry.
	if got := palettePath("sprites", 1); got != "palette.pal" {
		t.Errorf("palettePath(%q, 1) = %q, want %q", "sprites", got, "palette.pal")
	}
	if got := palettePath("sprites", 3); got != "palette3.pal" {
		t.Errorf("palettePath(%q, 3) = %q, want %q", "sprites", got, "palette3.pal")
	}
}

// AC-3, the empty answers. A tier below 1 is asking for no file rather than for
// the first, and a class resolving no File has no directory to name one in.
func TestPalettePathIsEmptyBelowTierOneAndWithNoFile(t *testing.T) {
	unit := loadOneUnit(t, spriteUnitsReg(t,
		[]synth.RegNode{regStr("File0", "unused"), regStr("File1", `Some\Where`)},
		regDir("Unit0", regInt("ID", 1), regInt("File", 1))))
	for _, tier := range []int{0, -1, -4096} {
		if got := unit.PalettePath(tier); got != "" {
			t.Errorf("PalettePath(%d) = %q, want \"\"", tier, got)
		}
	}

	fileless := unitByID(t, loadUnits(t, spriteUnitsReg(t,
		[]synth.RegNode{regStr("File0", "unused")},
		regDir("Unit0", regInt("ID", 1)))), 1)
	if got := fileless.SpritePath(); got != "" {
		t.Fatalf("fixture is not File-less: SpritePath() = %q", got)
	}
	for tier := 1; tier <= TierLimit+1; tier++ {
		if got := fileless.PalettePath(tier); got != "" {
			t.Errorf("File-less class: PalettePath(%d) = %q, want \"\"", tier, got)
		}
	}
}

// AC-4. The clamp over its whole stated domain, plus the two boundaries the
// limit itself sits on.
func TestTierCountClampsTheKey(t *testing.T) {
	for key, want := range map[int32]int{
		-4096: 0, -1: 0, 0: 0, 1: 1, 2: 2, 3: 3, 4: 4, 5: 4, 7: 4, 1 << 30: 4,
	} {
		c := &UnitClass{Palette: key}
		if got := c.TierCount(); got != want {
			t.Errorf("Palette = %d: TierCount() = %d, want %d", key, got, want)
		}
	}
	if TierLimit != 4 {
		t.Fatalf("TierLimit = %d, want 4", TierLimit)
	}
}
