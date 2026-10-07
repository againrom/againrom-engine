package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapload"
)

// The tool's table verb over a SYNTHETIC archive: this file writes the table's
// bytes from the documented wire primitives, internal/synth wraps them in a .res
// and writes a map beside it, and the real vfs, the real walk, the real searches
// and the real world builder run over them. No game install is read, and the
// only files touched are the test's own temp dir (golden rule 2).
//
// It is also the one place in the tree where the parsed collection meets the
// interface the definition tier searches through — pkg/data may not import the
// format tier — so this is where that seam is exercised rather than asserted.

type binWriter struct{ b []byte }

func (w *binWriter) u8(v byte)    { w.b = append(w.b, v) }
func (w *binWriter) u16(v uint16) { w.b = binary.LittleEndian.AppendUint16(w.b, v) }
func (w *binWriter) u32(v uint32) { w.b = binary.LittleEndian.AppendUint32(w.b, v) }

func (w *binWriter) str(s string) {
	w.u8(byte(len(s)))
	w.b = append(w.b, s...)
}

// strArray writes a COUNTED array — the group title arrays' shape. strs writes
// a fixed run with no count word, which is what an entry's own trailing strings
// are. The fixture has to be able to spell both, or a walk that confused them
// would be tested against a stream that made the same mistake.
func (w *binWriter) strArray(ss []string) {
	w.u16(uint16(len(ss)))
	for _, s := range ss {
		w.str(s)
	}
}

func (w *binWriter) strs(ss []string) {
	for _, s := range ss {
		w.str(s)
	}
}

func (w *binWriter) params(p []int32) {
	w.u16(uint16(len(p)))
	for _, v := range p {
		w.u32(uint32(v))
	}
}

// emptyGroup is a group with no titles and one zero-count collection per
// sibling: the shortest valid form, used for the six groups this test does not
// exercise.
func (w *binWriter) emptyGroup(collections int) {
	w.strArray(nil)
	for i := 0; i < collections; i++ {
		w.u32(0)
	}
}

// binRow is a 38-cell parameter row with every cell empty but the named slots.
func binRow(slots map[int]int32) []int32 {
	p := make([]int32, 38)
	for i := range p {
		p[i] = -1
	}
	for s, v := range slots {
		p[s] = v
	}
	return p
}

// synthDataBin writes a whole eight-group table whose Units and Humans
// collections carry one written row each that a placement can reach.
func synthDataBin() []byte {
	w := &binWriter{}
	w.emptyGroup(2) // A: Shapes, Materials
	w.emptyGroup(1) // B: Magic
	w.emptyGroup(3) // C: Armors, Shields, Weapons
	w.emptyGroup(1) // D: MagicItems

	// E: Units — count 3, so entries 1 and 2 are written and entry 0 reserved.
	// Entry 2 is nameless with no parameters, which is the shape 62 of the
	// shipped 118 take.
	w.strArray([]string{"titleE0", "titleE1"})
	w.u32(3)
	w.str("Beast")
	// Slot 0x20 is the movement domain, written non-ground so the report has a
	// domain other than the default to say: a fixture where every placement is
	// on the ground could not tell a column that is read from one that is not.
	//
	// Slots 11 to 18 are the nine source columns the eight combat numbers come
	// off — the damage minimum and maximum, the routing switch, the to-hit, the
	// defence, the absorption and the cadence pair. No two of the values are
	// equal and none is a constructor default, so a report printing one column
	// where it means another is a visible failure rather than a coincidence.
	//
	// Slot 10 is the SCAN RANGE, at a value that is neither the constructor's 5
	// nor any other column on the row, so a report printing a neighbouring
	// column in its place is a visible failure.
	w.params(binRow(map[int]int32{0x1d: 200, 0x1e: 0, 4: 100, 0x20: 3, 10: 9,
		11: 3, 12: 9, 13: 3, 14: 40, 15: 12, 16: 2, 17: 11, 18: 7}))
	w.strs(make([]string, 2))
	w.str("")
	w.params(nil)
	w.strs(make([]string, 2))

	// F: Humans — one written row on type 7, and ten trailing strings.
	w.strArray([]string{"titleF0"})
	w.u32(2)
	w.str("Man")
	w.params(binRow(map[int]int32{0x10: 7, 0x18: 900, 8: 6}))
	w.strs(make([]string, 10))

	// G: Buildings — count 3, so entries 1 and 2 are written and entry 0 is the
	// reserved one. Entry 1 is a 2x1 whose attach set names both cells and whose
	// blocking set names only the first, so one cell closes and one opens; entry
	// 2 is nameless with no parameters, the short row a placement must skip.
	w.strArray([]string{"sizeX", "sizeY", "t2", "t3", "Passability", "BuildingPresent"})
	w.u32(3)
	w.str("Bridge")
	w.params([]int32{2, 1, -1, -1, 0b01, 0b11})
	w.str("")
	w.params(nil)
	w.emptyGroup(1) // H: Spells
	return w.b
}

// synthWorld writes the archive the verb reads: its identity comes from the host
// filename, so the file is named for the archive it stands in for.
func synthWorld(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "world.res")
	arc := synth.Archive([]synth.File{{Path: "data/data.bin", Data: synthDataBin()}})
	if err := os.WriteFile(path, arc, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func synthMap(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "scn.alm")
	b := synth.ALM(synth.ALMOptions{
		Width: 32, Height: 32,
		Objects: []synth.ALMObject{
			{X: 0x0A80, Y: 0x0A80, Kind: 1, Field0C: 1}, // resolves entry 1; keep its health seed
			{X: 0x0C80, Y: 0x0A80, Kind: 2},             // resolves entry 2, which is short
			{X: 0x0E80, Y: 0x0A80, Kind: 200},           // names no entry at all
		},
		Units: []synth.ALMUnit{
			{X: 0x0A80, Y: 0x0A80, ClassID: 200},         // the units arm, resolving
			{X: 0x0B80, Y: 0x0A80, ClassID: 7},           // the humans arm, reaching an entry
			{X: 0x0C80, Y: 0x0A80, ClassID: 7, Flags: 1}, // the npc arm — below the class-key floor,
			// which is the only band the flag word diverts from
			{X: 0x0D80, Y: 0x0A80, ClassID: 201}, // the units arm, matching nothing
		},
	})
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// fieldsOfLine finds the one line whose first whitespace-separated field is head
// and returns that line's fields. Comparing FIELDS rather than the rendered line
// keeps the assertion on what the report says and off how wide its columns
// happen to be.
func fieldsOfLine(t *testing.T, out, head string) []string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) > 0 && f[0] == head {
			return f
		}
	}
	t.Fatalf("no line beginning %q in:\n%s", head, out)
	return nil
}

func sameFields(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestDataBinVerbCensusesTheTable: the walk tiles the node, and the counts come
// back as the fixture wrote them.
func TestDataBinVerbCensusesTheTable(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-databin", synthWorld(t)}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := out.String()
	size := strconv.Itoa(len(synthDataBin()))

	// The node's size and the walk's own cursor are the same number, and that is
	// the claim the run exists to make: the grammar tiles the file.
	consumed := []string{"world/data/data.bin:", size, "of", size, "byte(s)", "consumed,", "0", "left", "over"}
	if f := fieldsOfLine(t, got, "world/data/data.bin:"); !sameFields(f, consumed) {
		t.Errorf("the consumption line is %q, want %q", f, consumed)
	}

	// collection, written entries, titles, rows carrying parameters.
	for _, want := range [][]string{
		{"Units", "2", "2", "1"},
		{"Humans", "1", "1", "1"},
		{"Shapes", "0", "0", "0"},
		{"Spells", "0", "0", "0"},
	} {
		if f := fieldsOfLine(t, got, want[0]); !sameFields(f, want) {
			t.Errorf("the %s row is %q, want %q", want[0], f, want)
		}
	}
	for _, want := range []string{"Units parameter widths: 38 x1", "Units rows with parameters (1): 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("the census does not carry %q:\n%s", want, got)
		}
	}
}

func humansCensusDataBin(rows []struct {
	name   string
	params []int32
}) []byte {
	w := &binWriter{}
	w.emptyGroup(2) // A: Shapes, Materials
	w.emptyGroup(1) // B: Magic
	w.emptyGroup(3) // C: Armors, Shields, Weapons
	w.emptyGroup(1) // D: MagicItems
	w.emptyGroup(1) // E: Units -- this test's whole question is the Humans collection
	w.strArray(nil) // F: Humans
	w.u32(uint32(len(rows) + 1))
	for _, r := range rows {
		w.str(r.name)
		w.params(r.params)
		w.strs(make([]string, 10))
	}
	w.emptyGroup(1) // G: Buildings
	w.emptyGroup(1) // H: Spells
	return w.b
}

func TestDataBinVerbCensusesTheHumansCollection(t *testing.T) {
	rows := []struct {
		name   string
		params []int32
	}{
		// Body caps at 50 (raw 999), column 1: derives to 217 -- by far the
		// largest ratio of derived to column (217), and ABOVE its column.
		{"Strong", binRow(map[int]int32{0: 999, 4: 1})},
		// Body 1, column 500: derives to a small POSITIVE 2 -- BELOW its
		// column, and nowhere near the largest ratio (2/500).
		{"Weak", binRow(map[int]int32{0: 1, 4: 500})},
		// All-default Body/Reaction/Mind/Spirit derive to 70 -- the same
		// figure TestDataBinVerbResolvesAMapsPlacements' own "Man" row and
		// pkg/mapload's own suite already pin for that same all-default
		// shape. Its column is set to that SAME 70, so it derives EQUAL.
		{"Match", binRow(map[int]int32{4: 70})},
		{"NoColumn", binRow(map[int]int32{4: 0})},
		// Body 0: the WHOLE arm gates to 0 (recompute.go step 3), BELOW its
		// positive column of 42 -- the one row counted as zero derived.
		{"NoBody", binRow(map[int]int32{0: 0, 4: 42})},
	}
	arc := synth.Archive([]synth.File{{Path: "data/data.bin", Data: humansCensusDataBin(rows)}})
	path := filepath.Join(t.TempDir(), "world.res")
	if err := os.WriteFile(path, arc, 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := run([]string{"-databin", path}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := out.String()

	if want := "humans health: 5 row(s), column sum 613, derived sum 359"; !strings.Contains(got, want) {
		t.Errorf("the census does not carry %q:\n%s", want, got)
	}
	if want := "derived above column 2, below 2, equal 1; zero column 1, zero derived 1"; !strings.Contains(got, want) {
		t.Errorf("the census does not carry %q:\n%s", want, got)
	}
	if want := `largest ratio: "Strong" derived 217 against column 1`; !strings.Contains(got, want) {
		t.Errorf("the census does not carry %q:\n%s", want, got)
	}
}

// TestDataBinVerbResolvesAMapsPlacements: every placement lands on a named arm,
// and the health, speed and domain reported are the ones the world carries — the
// resolved placement at half again its maximum, everything else at the
// provisional pair, and the one whose class entry names the air column reported
// as flying where the other three are not.
//
// The synthetic table names no speed column, so every placement here takes the
// constructor's own default and the alphabet is one value wide; what the line
// pins is that the number is REPORTED and reported off the entity, and the
// census beside it that a map's alphabet is summed and counted.
func TestDataBinVerbResolvesAMapsPlacements(t *testing.T) {
	res := synthWorld(t)
	m := synthMap(t, filepath.Dir(res))

	var out bytes.Buffer
	if err := run([]string{"-databin", res, m, "3"}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "4 placement(s) at difficulty 3") {
		t.Errorf("the report does not name its map and difficulty:\n%s", got)
	}

	for _, want := range [][]string{
		{"0", "key", "0x00c8/0x0000", "arm", "units", "entry", "1", "owner", "0", "healthMax", "150", "speed", "10", "domain", "air"},
		{"1", "key", "0x0007/0x0000", "arm", "humans", "entry", "1", "owner", "0", "healthMax", "70", "speed", "10", "domain", "ground"},
		{"2", "key", "0x0007/0x0000", "arm", "npc", "entry", "0", "owner", "0", "healthMax", "100", "speed", "10", "domain", "ground"},
		{"3", "key", "0x00c9/0x0000", "arm", "units", "entry", "0", "owner", "0", "healthMax", "100", "speed", "10", "domain", "ground"},
	} {
		if f := fieldsOfLine(t, got, want[0]); !sameFields(f, want) {
			t.Errorf("placement %s reads %q, want %q", want[0], f, want)
		}
	}
	for _, want := range [][]string{
		{"arm", "npc", "taken", "1", "reached", "an", "entry", "0"},
		{"arm", "server-id", "taken", "0", "reached", "an", "entry", "0"},
		{"arm", "humans", "taken", "1", "reached", "an", "entry", "1"},
		{"arm", "units", "taken", "2", "reached", "an", "entry", "1"},
		// All three domains are printed, zeros included, and they sum to the
		// placement count — which is what makes the census checkable against
		// the per-placement lines above it.
		{"domain", "ground", "3"},
		{"domain", "ghost", "0"},
		{"domain", "air", "1"},
		{"domain", "total", "4", "of", "4", "placement(s)"},
		// The speed alphabet: one value here, carried by all four, and the
		// distinct count beside it.
		{"speed", "10", "4", "placement(s)"},
		{"speed", "--", "1", "distinct", "value(s)"},
	} {
		found := false
		for _, line := range strings.Split(got, "\n") {
			if sameFields(strings.Fields(line), want) {
				found = true
			}
		}
		if !found {
			t.Errorf("the arm census has no %q:\n%s", want, got)
		}
	}

	// The default value is the one a caller who says nothing gets.
	out.Reset()
	if err := run([]string{"-databin", res, m}, &out); err != nil {
		t.Fatalf("run without a difficulty: %v", err)
	}
	if !strings.Contains(out.String(), "at difficulty 2") {
		t.Errorf("the default difficulty is not 2:\n%s", out.String())
	}
	if f := fieldsOfLine(t, out.String(), "0"); f[len(f)-5] != "100" {
		t.Errorf("at difficulty 2 the resolved placement reads %q, want its unadjusted maximum 100", f)
	}
}

// TestDataBinVerbRefusesWhatItCannotUse: a difficulty outside the three, a
// difficulty that is not a number, and an archive holding no table.
func TestDataBinVerbRefusesWhatItCannotUse(t *testing.T) {
	res := synthWorld(t)
	m := synthMap(t, filepath.Dir(res))

	var out bytes.Buffer
	if err := run([]string{"-databin", res, m, "4"}, &out); err == nil {
		t.Error("difficulty 4 was accepted")
	}
	if err := run([]string{"-databin", res, m, "high"}, &out); err == nil {
		t.Error("a non-numeric difficulty was accepted")
	}

	empty := filepath.Join(t.TempDir(), "world.res")
	if err := os.WriteFile(empty, synth.Archive([]synth.File{{Path: "data/other.bin", Data: []byte{1}}}), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-databin", empty}, &out); err == nil {
		t.Error("an archive holding no table was accepted")
	}
}

// TestTheTableVerbsArgumentShapes: a lone -databin is a verb missing its
// archive, not an archive named -databin, so it must be a usage error and not
// "no such file".
func TestTheTableVerbsArgumentShapes(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-databin"}, &out); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Errorf("a lone -databin gave %v, want a usage error", err)
	}
	if err := run([]string{"-databin", "a", "b", "c", "d"}, &out); err == nil ||
		!strings.Contains(err.Error(), "usage") {
		t.Errorf("too many arguments gave %v, want a usage error", err)
	}
}

func TestDataBinVerbCensusesAMapsStructures(t *testing.T) {
	res := synthWorld(t)
	m := synthMap(t, filepath.Dir(res))

	var out bytes.Buffer
	if err := run([]string{"-databin", res, m}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := out.String()

	// Entry 1 resolves; entry 2 is short; key 200 names nothing. Entry 1 is a
	// 2x1 attaching both cells, closing the one its blocking set names and
	// opening the other.
	for _, want := range []string{
		"3 placed structure(s)",
		"structures: 1 resolved, 1 unresolved, 1 short, 0 zero-extent",
		"cells: 2 attached (1 closed, 1 opened), 0 dropped, 0 refused, 0 abandoned",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the report is missing %q in:\n%s", want, got)
		}
	}

	ci := strings.Index(got, "placed structure(s)")
	pi := strings.Index(got, "placement(s) at difficulty")
	if ci < 0 || pi < 0 {
		t.Fatalf("the report has no structure census or no placement report:\n%s", got)
	}
	if ci > pi {
		t.Errorf("the structure census prints after the world is built; a refused unit entry would lose it")
	}
}

// TestDataBinVerbReportsTheSeededScriptField is the 1033 B3 line: the
// structure+0x42 value check opcode 21 reads and instant opcode 26 writes, as
// mapload.Structures seeds it from the definition table.
//
// The fixture exercises all three arms of the seed in one map. Entry 1 carries
// -1 at parameter position 3 and a nonzero placement word, so it seeds 0xffff:
// the store truncates
// to the width of the destination the claim names, and a negative table cell is
// what that path looks like. Entry 2 is a short row and key 200 names no entry,
// and each of those placements keeps the zero value while still yielding a
// structure.
//
// No shipped entry reaches the truncation -- healthMax runs 0..30000 over all 66
// shipped entries on both roots -- so the fixture is the only place it is
// witnessed.
func TestDataBinVerbReportsTheSeededScriptField(t *testing.T) {
	res := synthWorld(t)
	m := synthMap(t, filepath.Dir(res))

	var out bytes.Buffer
	if err := run([]string{"-databin", res, m}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := out.String()

	want := "script field +0x42: 3 seeded, 2 zero, range 0..65535, values 0x2 65535x1"
	if !strings.Contains(got, want) {
		t.Errorf("the report is missing %q in:\n%s", want, got)
	}

	// The reader listing beneath it. The synthetic map carries a zero-length
	// type-7 body, which alm refuses, so this fixture reaches the listing's
	// compile-failure line. That the failure is REPORTED rather than swallowed
	// is the property: a listing printing nothing here would read as "no node
	// touches the field" for a map whose script was never read.
	if want := "+0x42 readers: the script did not compile"; !strings.Contains(got, want) {
		t.Errorf("the report is missing %q in:\n%s", want, got)
	}
}

// TestTheFieldReaderListingSeparatesItsFourCases witnesses the line shapes no
// shipped map produces. All 16 authored check-21 nodes on the campaign resolve
// to a structure their map placed, so without this the only rendering ever
// exercised would be the one that cannot go wrong.
func TestTheFieldReaderListingSeparatesItsFourCases(t *testing.T) {
	for _, tc := range []struct {
		name string
		refs []mapload.StructureFieldRef
		err  error
		want string
	}{
		{"no node authored", nil, nil,
			"+0x42 readers: none authored on this map"},
		{"the script did not compile", nil, errUsage,
			"+0x42 readers: the script did not compile"},
		{"a check on a placed structure",
			[]mapload.StructureFieldRef{{Node: 13, Ref: 9, HasRef: true, Placed: true, Value: 30000}},
			nil, "check node 13 reads structure 9, seeded +0x42 = 30000"},
		{"an instant on a placed structure",
			[]mapload.StructureFieldRef{{Node: 4, Writes: true, Ref: 2, HasRef: true, Placed: true, Value: 1}},
			nil, "instant node 4 writes structure 2, seeded +0x42 = 1"},
		{"a reference that did not resolve",
			[]mapload.StructureFieldRef{{Node: 7}},
			nil, "check node 7 reads no structure: its reference did not resolve"},
		{"a structure the map did not place",
			[]mapload.StructureFieldRef{{Node: 8, Ref: 99, HasRef: true}},
			nil, "check node 8 reads structure 99, which this map did not place"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := printStructureFieldReaders(&out, tc.refs, tc.err); err != nil {
				t.Fatalf("printStructureFieldReaders: %v", err)
			}
			if got := out.String(); !strings.Contains(got, tc.want) {
				t.Errorf("printed %q, want a line carrying %q", got, tc.want)
			}
		})
	}
}

// combatHeading is the line the template-numbers report opens with. The test
// locates its section by this rather than by a line number, so a row added above
// it does not silently point these assertions at the wrong report — the verb
// prints an index-led line in TWO reports and only one of them is this one.
const combatHeading = "template combat numbers"

// combatRowFields returns the fields of the combat report's row for placement
// index, searching only AFTER the report's own heading.
func combatRowFields(t *testing.T, out string, index string) []string {
	t.Helper()
	lines := strings.Split(out, "\n")
	start := -1
	for i, line := range lines {
		if strings.Contains(line, combatHeading) {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("no %q report in:\n%s", combatHeading, out)
	}
	for _, line := range lines[start:] {
		if f := strings.Fields(line); len(f) > 0 && f[0] == index {
			return f
		}
	}
	t.Fatalf("the combat report has no row %s in:\n%s", index, out)
	return nil
}

// TestTheTableVerbPrintsTheTemplateCombatNumbers is AC-10's runnable half and
// SC-10: the report the owner compares against the game's own panel.
//
// It runs at ALL THREE difficulties, because two of the eight move with the
// setting and a report taken at one value could not be compared with a panel
// read at another.
func TestTheTableVerbPrintsTheTemplateCombatNumbers(t *testing.T) {
	res := synthWorld(t)
	m := synthMap(t, filepath.Dir(res))

	for _, tc := range []struct {
		diff    string
		toHit   string
		defence string
	}{
		{"1", "40", "12"},
		{"2", "40", "12"},
		// The hard setting adds its constant to these two and to NOTHING else on
		// the row, which is what the six unchanged columns beside them assert.
		{"3", "90", "62"},
	} {
		t.Run("difficulty "+tc.diff, func(t *testing.T) {
			var out bytes.Buffer
			if err := run([]string{"-databin", res, m, tc.diff}, &out); err != nil {
				t.Fatalf("run: %v", err)
			}
			got := out.String()

			// THE PERSON'S OWN SIGHT COLUMN, read off the world beside his eight.
			// It is his row's, not the creature row's and not the constructor's,
			// which is the whole of what a per-band column has to show.
			if f := combatRowFields(t, got, "1"); len(f) > 11 && f[11] != "6" {
				t.Errorf("placement 1 reads sight %q, want his row's own 6", f[11])
			}

			// THE ARMED PERSON'S ROW IS CALLED ONE. This is the criterion: the
			// report may not describe a row as unequipped while having armed it,
			// and placement 1 is the row that would have been so described.
			if f := combatRowFields(t, got, "1"); len(f) < 3 || f[2] != "person" {
				t.Errorf("placement 1 reads band %q, want %q", f[2:min(3, len(f))], "person")
			}

			// THE CAVEAT IS IN THE OUTPUT, not only in this repository. A reader
			// of a pasted report has nothing but the report.
			//
			// It is asserted PER BAND, because the blanket sentence this used to
			// pin became false the moment a person's equipment was resolved: the
			// report would have gone on telling a reader to discount an armed
			// person's damage as his template's, and this very check would have
			// held it there. A check whose expectation is the defect cannot fail
			// on it.
			for _, phrase := range []string{
				"CREATURE row", "TEMPLATE'S NUMBERS AND EQUIPMENT IS NOT APPLIED",
				"PERSON row IS ARMED",
			} {
				if !strings.Contains(got, phrase) {
					t.Errorf("the report does not say %q:\n%s", phrase, got)
				}
			}

			// The resolved placement: its eight, one column at a time.
			want := []string{"0", "0x00c8/0x0000", "creature", "11", "7", tc.toHit, tc.defence, "2", "3", "6", "true", "9"}
			if f := combatRowFields(t, got, "0"); !sameFields(f[:len(want)], want) {
				t.Errorf("the resolved placement reads %q, want %q", f[:len(want)], want)
			} else if entry := strings.Join(f[len(want):], " "); !strings.Contains(entry, "Beast") {
				t.Errorf("the resolved placement names its entry as %q, want the entry it reached", entry)
			}

			// The two that reach NO entry: the constructor's cadence, six
			// zeros, and SAID SO rather than shown as a row of zeros a reader
			// would have to interpret.
			for _, tcRow := range []struct{ index, key, arm string }{
				{"2", "0x0007/0x0000", "npc"},
				{"3", "0x00c9/0x0000", "units"},
			} {
				want := []string{tcRow.index, tcRow.key, "-", "8", "4", "0", "0", "0", "0", "0", "false", "5"}
				f := combatRowFields(t, got, tcRow.index)
				if !sameFields(f[:len(want)], want) {
					t.Errorf("placement %s reads %q, want %q", tcRow.index, f[:len(want)], want)
					continue
				}
				entry := strings.Join(f[len(want):], " ")
				if !strings.Contains(entry, "none") || !strings.Contains(entry, tcRow.arm) {
					t.Errorf("placement %s describes its entry as %q, want it named as unresolved "+
						"on the %s arm", tcRow.index, entry, tcRow.arm)
				}
			}
		})
	}
}

// TestTheCombatReportLeavesTheVerbsExistingRowsAlone is T3's scope fence: the
// report is ADDED, so every line the verb printed before it must still be there,
// unchanged, and the exit code for a refusal must still be a refusal.
func TestTheCombatReportLeavesTheVerbsExistingRowsAlone(t *testing.T) {
	res := synthWorld(t)
	m := synthMap(t, filepath.Dir(res))

	var out bytes.Buffer
	if err := run([]string{"-databin", res, m, "3"}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := out.String()

	// The existing per-placement line, the arm census, the domain census and
	// the speed alphabet — all still present and all still ahead of the new
	// report. The owner field is now part of that per-placement line too.
	for _, want := range [][]string{
		{"0", "key", "0x00c8/0x0000", "arm", "units", "entry", "1", "owner", "0", "healthMax", "150", "speed", "10", "domain", "air"},
		{"arm", "units", "taken", "2", "reached", "an", "entry", "1"},
		{"domain", "total", "4", "of", "4", "placement(s)"},
		{"speed", "--", "1", "distinct", "value(s)"},
	} {
		found := false
		for _, line := range strings.Split(got, "\n") {
			if sameFields(strings.Fields(line), want) {
				found = true
			}
		}
		if !found {
			t.Errorf("the added report displaced %q:\n%s", want, got)
		}
	}
	if i, j := strings.Index(got, "distinct value(s)"), strings.Index(got, combatHeading); i < 0 || j < 0 || i > j {
		t.Errorf("the new report is not after the verb's existing output (%d vs %d)", i, j)
	}

	// And a refused difficulty is still refused, with no report printed at all.
	out.Reset()
	if err := run([]string{"-databin", res, m, "4"}, &out); err == nil {
		t.Error("difficulty 4 was accepted")
	}
	if strings.Contains(out.String(), combatHeading) {
		t.Errorf("a refused run still printed the combat report:\n%s", out.String())
	}
}
