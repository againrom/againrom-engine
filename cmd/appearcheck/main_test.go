package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// Everything in this file is SYNTHETIC (golden rule 2; spec T5's own "no
// game install present"): every collection, every table and every party
// member is built in the test, and every name in it is this test's own
// invention, never a shipped one. appearcheck's own read of a lawful install
// is exercised by hand against both roots, and that evidence is recorded
// where the lane's own verification lives — nothing here opens an archive.

// testEntry and testCollection are a minimal data.Collection, on
// cmd/wearcheck/main_test.go's own precedent.
type testEntry struct {
	name string
}

type testCollection []testEntry

func (c testCollection) Len() int                    { return len(c) }
func (c testCollection) EntryName(i int) string      { return c[i].name }
func (c testCollection) EntryParams(i int) []int32   { return nil }
func (c testCollection) EntryStrings(i int) []string { return nil }

// testTable is a table with one weapon, one shield and two armours at known
// rows — enough to exercise resolveItemName's own slot-decides-collection
// rule (weapon/shield/armor) without shipping a single game name.
func testTable() *mapload.Table {
	return &mapload.Table{
		Weapons: testCollection{{name: ""}, {name: "Test Sword"}},
		Shields: testCollection{{name: ""}, {name: "Test Shield"}},
		Armors:  testCollection{{name: ""}, {name: "Test Helm"}, {name: "Test Boots"}},
	}
}

func TestResolveItemNameByClassCollection(t *testing.T) {
	tbl := testTable()

	if name, ok := resolveItemName(1, data.ItemCode(1), tbl); !ok || name != "Test Sword" {
		t.Errorf("slot 1 row 1 = %q, %v; want %q, true", name, ok, "Test Sword")
	}
	if name, ok := resolveItemName(2, data.ItemCode(1), tbl); !ok || name != "Test Shield" {
		t.Errorf("slot 2 row 1 = %q, %v; want %q, true", name, ok, "Test Shield")
	}
	if name, ok := resolveItemName(7, data.ItemCode(2), tbl); !ok || name != "Test Boots" {
		t.Errorf("slot 7 row 2 = %q, %v; want %q, true", name, ok, "Test Boots")
	}
}

// TestResolveItemNameReportsWhatThisBuildCannotResolve is the refusal side:
// a row past the collection's own length, a row whose own entry is empty, a
// table with no collection for the slot's own class, and a nil table all
// answer (\"\", false) rather than an empty name printed as though it were
// one (spec T5's own instruction).
func TestResolveItemNameReportsWhatThisBuildCannotResolve(t *testing.T) {
	tbl := testTable()

	if _, ok := resolveItemName(1, data.ItemCode(5), tbl); ok {
		t.Error("row 5 is past Weapons' own length (2): want unresolved")
	}
	if _, ok := resolveItemName(1, data.ItemCode(0), tbl); ok {
		t.Error("row 0's own entry is empty: want unresolved")
	}
	if _, ok := resolveItemName(2, data.ItemCode(1), &mapload.Table{}); ok {
		t.Error("a table with no Shields collection: want unresolved")
	}
	if _, ok := resolveItemName(1, data.ItemCode(1), nil); ok {
		t.Error("a nil table: want unresolved")
	}
}

// fakeSource is a terrain.EntrySource holding exactly the addresses this
// test stocked it with — no vfs.FS, no archive, so this file can exercise
// printBlock (and, through it, game.LoadHeroBody) with no install anywhere
// near it (golden rule 2).
type fakeSource map[string][]byte

func (s fakeSource) ReadFile(address string) ([]byte, error) {
	if b, ok := s[address]; ok {
		return b, nil
	}
	return nil, errors.New("not found")
}

// graphicsAddress turns a container-relative path (data.HeroSheetPath's own
// return shape) into the address game.LoadHeroBody actually reads: the
// graphics container's own identity segment in front of it. It is DERIVED
// from game.GraphicsArchive through vfs.Identity — the same route
// pkg/game's own tests (statics_test.go's graphicsEntry) use — rather than a
// second "graphics/" literal, on T5 correction 1's own instruction not to
// spell that prefix a second time: a literal here could drift from
// pkg/game's unexported one and this test would still pass.
func graphicsAddress(t *testing.T, path string) string {
	t.Helper()
	identity, err := vfs.Identity(game.GraphicsArchive)
	if err != nil {
		t.Fatalf("vfs.Identity(%q): %v", game.GraphicsArchive, err)
	}
	return identity + "/" + path
}

// oneFrameSheet is a minimal WELL-FORMED .256 sheet — a palette and one
// frame of no pixels — on pkg/game/heroart_test.go's own precedent
// (bodyRecordSheet/bodyComposedSheet): enough for spr256.Decode to succeed
// and for game.LoadHeroBody's sheets.frames to answer a non-empty slice, so
// a test can drive the "loads=yes" arm through the real decoder rather than
// asserting it by construction.
func oneFrameSheet() []byte {
	pal := make([]color.RGBA, 8)
	pal[1] = color.RGBA{R: 0xff}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
		Frames:  []synth.Frame256{{Width: 1, Height: 1}},
	})
}

// TestPrintBlockFormatsOneArchetype exercises AC-13's own shape end to end
// over a hand-built member: the weapon line, the trained-skill line, an
// occupied slot's name and code, and the body/dir/class/sheet/loads footer
// — with the composed body's own class record in classes and its own sheet
// decodable in src, so loads=yes is reached through game.LoadHeroBody
// itself, not asserted by construction (T5 correction 1).
func TestPrintBlockFormatsOneArchetype(t *testing.T) {
	tbl := testTable()
	m := mapload.PartyMember{
		Class:   3,
		Body:    "swordsman",
		BodyDir: "heroes_l",
		Weapon:  &data.Weapon{Name: "Test Sword"},
	}
	m.Worn[0] = 1 // slot 1 (weapon): Weapons[1] "Test Sword"
	m.Worn[6] = 2 // slot 7 (armour): Armors[2] "Test Boots"

	wantSheet := data.HeroSheetPath(m.BodyDir, data.HeroBody(m.Body))
	classID, matched := data.HeroBodyClass(data.HeroBody(m.Body))
	if !matched {
		t.Fatalf("setup: %q matches no arm of data.HeroBodyClass", m.Body)
	}
	classes := map[int32]*terrain.UnitClass{classID: {Width: 1, Height: 1}}
	src := fakeSource{graphicsAddress(t, wantSheet): oneFrameSheet()}

	var buf bytes.Buffer
	printBlock(&buf, src, classes, tbl, "fighter", "male", m)
	out := buf.String()

	for _, want := range []string{
		"fighter / male   weapon=Test Sword",
		"trained " + data.SkillName(trainedSkillSlot),
		"slot 1 Test Sword 0x0001",
		"slot 7 Test Boots 0x0002",
		"body=swordsman dir=heroes_l class=3",
		"sheet=" + wantSheet,
		"(address inside the graphics container)",
		"loads=yes",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
}

// TestPrintBlockReportsBareHandsUnresolvedSlotAndAnAbsentSheet is the other
// side: no weapon, an occupied slot this table cannot resolve, and a body
// whose class record is known but whose composed sheet the source does not
// hold — so game.LoadHeroBody itself refuses, and printBlock must say
// loads=no rather than take the address's mere composition as success.
func TestPrintBlockReportsBareHandsUnresolvedSlotAndAnAbsentSheet(t *testing.T) {
	tbl := testTable()
	m := mapload.PartyMember{Body: "swordsman", BodyDir: "heroes_l"}
	m.Worn[6] = 9 // slot 7, row 9: past Armors' own length (3)

	classID, matched := data.HeroBodyClass(data.HeroBody(m.Body))
	if !matched {
		t.Fatalf("setup: %q matches no arm of data.HeroBodyClass", m.Body)
	}
	classes := map[int32]*terrain.UnitClass{classID: {Width: 1, Height: 1}}

	var buf bytes.Buffer
	printBlock(&buf, fakeSource{}, classes, tbl, "fighter", "male", m)
	out := buf.String()

	if !strings.Contains(out, "weapon=none") {
		t.Errorf("output = %q, want it to say the member holds no weapon", out)
	}
	if !strings.Contains(out, "<code names nothing this build can resolve>") {
		t.Errorf("output = %q, want the unresolved slot 7 reported plainly", out)
	}
	if !strings.Contains(out, "loads=no") {
		t.Errorf("output = %q, want loads=no for a sheet the archive does not hold", out)
	}
}

func TestPrintBlockSkipsAnEmptySlot(t *testing.T) {
	tbl := testTable()
	m := mapload.PartyMember{Body: "swordsman", BodyDir: "heroes_l"}
	// every slot left at its zero value

	var buf bytes.Buffer
	printBlock(&buf, fakeSource{}, nil, tbl, "fighter", "male", m)
	out := buf.String()

	if strings.Contains(out, "slot ") {
		t.Errorf("output = %q, want no slot line for an all-empty worn set", out)
	}
}

// TestPrintBlockProbesEachArchetypeFreshly is T5 correction 1's own
// isolation requirement: two calls sharing the SAME classes map but naming
// two different bodies, one whose sheet the source holds and one whose it
// does not, must not let the first call's successful load satisfy the
// second's — each printBlock call probes with its own fresh
// *terrain.UnitSet rather than one carried across calls.
func TestPrintBlockProbesEachArchetypeFreshly(t *testing.T) {
	tbl := testTable()
	swordsman := mapload.PartyMember{Body: "swordsman", BodyDir: "heroes_l"}
	mage := mapload.PartyMember{Body: "mage", BodyDir: "heroes_l"}

	swordsmanID, matched := data.HeroBodyClass(data.HeroBody(swordsman.Body))
	if !matched {
		t.Fatalf("setup: %q matches no arm of data.HeroBodyClass", swordsman.Body)
	}
	mageID, matched := data.HeroBodyClass(data.HeroBody(mage.Body))
	if !matched {
		t.Fatalf("setup: %q matches no arm of data.HeroBodyClass", mage.Body)
	}
	classes := map[int32]*terrain.UnitClass{
		swordsmanID: {Width: 1, Height: 1},
		mageID:      {Width: 1, Height: 1},
	}

	sheet := data.HeroSheetPath(swordsman.BodyDir, data.HeroBody(swordsman.Body))
	// mage's own composed sheet is deliberately left out of src.
	src := fakeSource{graphicsAddress(t, sheet): oneFrameSheet()}

	var buf1, buf2 bytes.Buffer
	printBlock(&buf1, src, classes, tbl, "fighter", "male", swordsman)
	printBlock(&buf2, src, classes, tbl, "mage", "male", mage)

	if !strings.Contains(buf1.String(), "loads=yes") {
		t.Errorf("swordsman block = %q, want loads=yes: its own sheet is in src", buf1.String())
	}
	if !strings.Contains(buf2.String(), "loads=no") {
		t.Errorf("mage block = %q, want loads=no: its sheet is absent, and it must not inherit "+
			"the swordsman probe's success", buf2.String())
	}
}

// TestPrintBodyListPrintsIndexAndDrawnClass is the shipped body list's own
// print: each line's index and the class data.HeroBodyClass resolves it to,
// with a name matching no arm disclosed rather than silently given the
// fallback class and nothing said about it.
func TestPrintBodyListPrintsIndexAndDrawnClass(t *testing.T) {
	list := data.BodyList{"unarmed", "not-a-shipped-name", "mage"}
	var buf bytes.Buffer
	printBodyList(&buf, list)
	out := buf.String()

	if !strings.Contains(out, "3 entries") {
		t.Errorf("output = %q, want it to state the list's own length", out)
	}
	wantClass, _ := data.HeroBodyClass("unarmed")
	if !strings.Contains(out, fmt.Sprintf("0: \"unarmed\" -> class %d", wantClass)) {
		t.Errorf("output missing the unarmed entry at index 0: %s", out)
	}
	if !strings.Contains(out, "1: \"not-a-shipped-name\"") || !strings.Contains(out, "no arm matches this name") {
		t.Errorf("output missing the unmatched-name disclosure at index 1: %s", out)
	}
}

// TestClassAndSexLabels is the archetype vocabulary this file authors
// (never a shipped name): the four combinations printReport crosses.
func TestClassAndSexLabels(t *testing.T) {
	if got := classLabel(false); got != "fighter" {
		t.Errorf("classLabel(false) = %q, want fighter", got)
	}
	if got := classLabel(true); got != "mage" {
		t.Errorf("classLabel(true) = %q, want mage", got)
	}
	if got := sexLabel(false); got != "male" {
		t.Errorf("sexLabel(false) = %q, want male", got)
	}
	if got := sexLabel(true); got != "female" {
		t.Errorf("sexLabel(true) = %q, want female", got)
	}
}

// TestRunRefusesBeforeOpeningAnyArchive mirrors cmd/wearcheck's own pattern
// (main_test.go there): every refusal below is reached on its arguments
// alone, with the asset root pointed at nothing, so none of them may read an
// install (spec T5's own "Done when": the no-root refusal, with no install
// present).
func TestRunRefusesBeforeOpeningAnyArchive(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no asset root", []string{}, "asset root"},
		{"an unknown flag", []string{"-nosuchflag"}, "flag"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := run(tc.args, &buf); err == nil {
				t.Fatalf("run(%v) returned no error; it printed %q", tc.args, buf.String())
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("run(%v) = %v, want an error naming %q", tc.args, err, tc.want)
			}
		})
	}
}
