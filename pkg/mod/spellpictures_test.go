package mod

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const spellPicturesRow = "[[spell]]\n" +
	"target = \"Fire_Ball\"\n" +
	"icon = \"assets/spells/fire_ball.png\"\n" +
	"cast = \"assets/spells/fire_ball-cast.16a\"\n" +
	"cast_row = { phases = 8, rotation_phases = 16, width = 64, height = 64, palette = 1 }\n" +
	"burst = \"assets/spells/fire_ball-burst.256\"\n" +
	"imported = true\n"

// A row names its book icon, its cast picture with its registry row and its
// burst; the writer gives the same text back, and a new row's pictures are
// written after its other fields.
func TestSpellRowNamesItsPicturesAndTheWriterKeepsThem(t *testing.T) {
	kind, _ := WrittenKind(SpellsFile)
	var d Data
	if err := kind.ReadHeld(DataReader{Mod: "m", Dir: "D"}, &d, SpellsFile, []byte(spellPicturesRow)); err != nil {
		t.Fatal(err)
	}
	r := d.Spells.Rows[0]
	if r.Icon.Path != "assets/spells/fire_ball.png" || r.Icon.Line != 3 || r.Icon.Mod != "m" || r.Icon.Dir != "D" ||
		r.Cast.Path != "assets/spells/fire_ball-cast.16a" || r.CastRow == nil || r.CastRow.RotationPhases.Val != 16 || r.CastRow.Flip.Set ||
		r.Burst.Path != "assets/spells/fire_ball-burst.256" || r.BurstRow != nil || !r.Imported.Val {
		t.Fatalf("row read as %+v", r)
	}
	if got := d.Spells.ImportedSpells(); !slices.Equal(got, []string{"Fire_Ball"}) {
		t.Fatalf("imported spells %v", got)
	}
	same, err := kind.Write(SpellsFile, []byte(spellPicturesRow), d)
	if err != nil || string(same) != spellPicturesRow {
		t.Fatalf("an unchanged row rewrote the file (%v):\n%s", err, same)
	}
	d.Spells.Rows = append(d.Spells.Rows, SpellRow{Target: "Bless", Icon: SpellPicture{Set: true, Path: "assets/spells/bless.png"},
		Burst: SpellPicture{Set: true, Path: "assets/spells/bless.16a"}, BurstRow: &SpellSheet{Phases: SpellInt{Set: true, Val: 4}}})
	out, err := kind.Write(SpellsFile, []byte(spellPicturesRow), d)
	if err != nil {
		t.Fatal(err)
	}
	want := spellPicturesRow + "[[spell]]\ntarget = \"Bless\"\nicon = \"assets/spells/bless.png\"\nburst = \"assets/spells/bless.16a\"\nburst_row = { phases = 4 }\n"
	if string(out) != want {
		t.Fatalf("written:\n%s\nwant:\n%s", out, want)
	}
}

// A picture field passes the reference rule and its kind's extensions; a
// registry row needs its picture and keeps to its bounds; imported needs a
// picture to mark.
func TestSpellPictureFieldsRefuseAtTheirLine(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		{"icon = \"assets/spells/a.bmp\"", ":3: icon \"assets/spells/a.bmp\" is not a .png file"},
		{"burst = \"assets/spells/a.png\"", ":3: burst \"assets/spells/a.png\" is not a .16a or .256 file"},
		{"cast = 5", ":3: cast must be a path in the mod, not an integer"},
		{"cast_row = { phases = 2 }", ":3: cast_row of Fire_Ball needs cast"},
		{"cast = \"assets/spells/a.16a\"\ncast_row = { phases = 0 }", ":4: cast_row.phases is 0, outside 1..1024"},
		{"cast = \"assets/spells/a.16a\"\ncast_row = { colour = 1 }", ":4: unknown key \"colour\" in cast_row"},
		{"cast = \"assets/spells/a.16a\"\nburst_row = { flip = 1 }", ":4: burst_row of Fire_Ball needs burst"},
		{"mana = 3\nimported = true", ":4: imported marks the pictures of Fire_Ball, and the row names none"},
	} {
		_, err := ParseSpells("m", SpellsFile, []byte("[[spell]]\ntarget = \"Fire_Ball\"\n"+tc.src+"\n"), nil)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%q: got %v, want %q", tc.src, err, tc.want)
		}
	}
}

// Two mods may give one spell its pictures: the merged row keeps each
// picture with the mod and folder that name it, and one field is still set
// by one mod.
func TestMergedSpellRowKeepsEachPicturesMod(t *testing.T) {
	var d Data
	if err := readSpells(DataReader{Mod: "a", Dir: "A"}, &d, SpellsFile, []byte("[[spell]]\ntarget = \"Fire_Ball\"\nicon = \"assets/spells/i.png\"\n")); err != nil {
		t.Fatal(err)
	}
	if err := readSpells(DataReader{Mod: "b", Dir: "B"}, &d, SpellsFile, []byte("[[spell]]\ntarget = \"Fire_Ball\"\ncast = \"assets/spells/c.16a\"\n")); err != nil {
		t.Fatal(err)
	}
	err := readSpells(DataReader{Mod: "c", Dir: "C"}, &d, SpellsFile, []byte("[[spell]]\ntarget = \"Fire_Ball\"\nicon = \"assets/spells/x.png\"\n"))
	if err == nil || !strings.Contains(err.Error(), "icon of \"Fire_Ball\" is already set by mod \"a\"") {
		t.Fatalf("a second icon: %v", err)
	}
	merged, _, err := MergeSpells(d.Spells, []string{"Fire_Ball"})
	if err != nil {
		t.Fatal(err)
	}
	r := merged.Rows[0]
	if r.Icon.Mod != "a" || r.Icon.Dir != "A" || r.Cast.Mod != "b" || r.Cast.Dir != "B" || r.Cast.File != SpellsFile {
		t.Fatalf("merged pictures %+v %+v", r.Icon, r.Cast)
	}
}

// The tree check names the files a spell row reads, and warns at a row whose
// pictures are copies of a game install's.
func TestTreeCheckNamesSpellPicturesAndWarnsOnImported(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "m")
	writeTree(t, dir, map[string]string{
		SpellsFile:                          spellPicturesRow,
		"assets/spells/fire_ball.png":       "p",
		"assets/spells/fire_ball-cast.16a":  "p",
		"assets/spells/fire_ball-burst.256": "p",
	})
	tree, err := OpenTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	desc, err := tree.Describe()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{SpellsFile + ":7: " + ImportedPicturesWarning}; !slices.Equal(desc.Warnings, want) {
		t.Fatalf("warnings %q, want %q", desc.Warnings, want)
	}
	var kinds []string
	for _, f := range desc.Files {
		kinds = append(kinds, f.Path+" "+f.Kind)
	}
	for _, want := range []string{"assets/spells/fire_ball.png spells", "assets/spells/fire_ball-cast.16a spells", "assets/spells/fire_ball-burst.256 spells"} {
		if !slices.Contains(kinds, want) {
			t.Errorf("%q not among %q", want, kinds)
		}
	}
}
