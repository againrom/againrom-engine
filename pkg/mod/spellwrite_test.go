package mod

import (
	"strings"
	"testing"
)

// The spells writer keeps every byte outside the fields it changes: the
// comment, the [global] table and a value a setting gives; a removed field
// leaves with its line and a new row is appended after the rest.
func TestSpellsWriterKeepsTheBytesOutsideTheFieldsItChanges(t *testing.T) {
	src := "# the mod's book\n" +
		"[global]\n" +
		"mana_mul = [3, 4]\n" +
		"\n" +
		"[[spell]]\n" +
		"target = \"Fire_Ball\"\n" +
		"mana   = \"@fire_ball_mana\"\n" +
		"book_position = 3\n" +
		"\n" +
		"[[spell]]\n" +
		"target = \"Bless\"\n" +
		"off = true\n"
	kind, ok := WrittenKind(SpellsFile)
	if !ok || !HeldDraft(SpellsFile) || !RebuildsContext(SpellsFile, func(string) bool { return true }) {
		t.Fatal("the spells kind is not a held draft the editor writes")
	}
	settings := []SettingValue{{Key: "fire_ball_mana", Value: Value{Kind: KindInt, Int: 20}}}
	var d Data
	if err := kind.ReadHeld(DataReader{Mod: "m", Settings: settings}, &d, SpellsFile, []byte(src)); err != nil {
		t.Fatal(err)
	}
	same, err := kind.Write(SpellsFile, []byte(src), d)
	if err != nil || string(same) != src {
		t.Fatalf("an unchanged set rewrote the file (%v):\n%s", err, same)
	}
	d.Spells.Rows[0].BookPosition = SpellInt{Set: true, Val: 5}
	d.Spells.Rows[1].Off = SpellBool{}
	d.Spells.Rows[1].BookPosition = SpellInt{Set: true, Val: 3}
	d.Spells.Rows = append(d.Spells.Rows, SpellRow{Target: "Darkness", BookPosition: SpellInt{Set: true, Val: 23}})
	d.Spells.Global.BookSize = SpellInt{Set: true, Val: 24}
	out, err := kind.Write(SpellsFile, []byte(src), d)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(src, "mana_mul = [3, 4]\n\n", "mana_mul = [3, 4]\n\nbook_size = 24\n", 1)
	want = strings.Replace(want, "book_position = 3\n", "book_position = 5\n", 1)
	want = strings.Replace(want, "off = true\n", "book_position = 3\n", 1)
	want += "[[spell]]\ntarget = \"Darkness\"\nbook_position = 23\n"
	if string(out) != want {
		t.Fatalf("written:\n%s\nwant:\n%s", out, want)
	}
	back, err := ParseSpells("m", SpellsFile, out, settingOf(settings))
	if err != nil || len(back.Rows) != 3 || back.Rows[0].Mana.Val != 20 || back.Global.BookSize.Val != 24 {
		t.Fatalf("read back %+v: %v", back, err)
	}
}

// A first [global] key in a file without the table adds the table, not a
// row of an array.
func TestSpellsWriterAddsTheGlobalTable(t *testing.T) {
	kind, _ := WrittenKind(SpellsFile)
	d := Data{Spells: SpellData{Global: SpellGlobal{BookSize: SpellInt{Set: true, Val: 20}}}}
	out, err := kind.Write(SpellsFile, nil, d)
	if err != nil || string(out) != "[global]\nbook_size = 20\n" {
		t.Fatalf("written (%v):\n%s", err, out)
	}
}

// The run of a mod in the editor reads its spells from the project's draft,
// and a project without the file holds none.
func TestSpellsDraftGoesOverTheFolder(t *testing.T) {
	folder := Data{}
	if err := readSpells(DataReader{Mod: "m"}, &folder, SpellsFile, []byte("[[spell]]\ntarget = \"Bless\"\noff = true\n")); err != nil {
		t.Fatal(err)
	}
	d := folder
	if err := draftSpells(DataReader{Mod: "m"}, &d, foldedDrafts{SpellsFile: []byte("[[spell]]\ntarget = \"Haste\"\nbook_position = 1\n")}, nil); err != nil {
		t.Fatal(err)
	}
	if len(d.Spells.Rows) != 1 || d.Spells.Rows[0].Target != "Haste" {
		t.Fatalf("drafted rows %+v", d.Spells.Rows)
	}
	d = folder
	if err := draftSpells(DataReader{Mod: "m"}, &d, foldedDrafts{}, nil); err != nil || !d.Spells.Empty() {
		t.Fatalf("a project without the file: %+v %v", d.Spells, err)
	}
}
