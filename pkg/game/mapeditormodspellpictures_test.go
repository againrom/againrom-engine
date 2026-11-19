package game

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/internal/witness"
	"againrom/pkg/data"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// spellsEdits is the spells page's edits as the window holds them now.
func spellsEdits(e *ui.MapEditor) *mapInspectionEdits {
	return e.Document().Edits.(*mapInspectionEdits)
}

// spellsSet makes one edit of the spells page through the window's route.
func spellsSet(e *ui.MapEditor, key, value string) error {
	_, err := spellsEdits(e).SetWorkspaceField(ui.ModSpellsScope, key, value)
	return err
}

// spellsPageField is the spells page's field key, false for none.
func spellsPageField(e *ui.MapEditor, key string) (ui.InspectionField, bool) {
	fields := spellsEdits(e).WorkspaceFields(ui.ModSpellsScope)
	i := slices.IndexFunc(fields, func(f ui.InspectionField) bool { return f.Key == key })
	if i < 0 {
		return ui.InspectionField{}, false
	}
	return fields[i], true
}

// hostFile writes b as a host file named name, outside every mod.
func hostFile(t *testing.T, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// chooseSpell clicks catalog spell name.
func chooseSpell(t *testing.T, e *ui.MapEditor, name string) {
	t.Helper()
	cell := catalogCell(t, e, name)
	spellsPointer(e, cell, cell)
	if spellsEdits(e).spell != name {
		t.Fatalf("the page chose %q, want %q", spellsEdits(e).spell, name)
	}
}

// The chosen spell's icon, cast picture and burst are picked as host files:
// each is copied under assets/spells and named in the spell's row in one
// step, the book cell and the catalog draw the icon, a spell the game's
// book has no picture for included, and a file of the wrong sort or size is
// refused.
func TestSpellsPagePicksTheChosenSpellsPictures(t *testing.T) {
	_, e, _ := spellsPage(t)
	red := color.RGBA{R: 0xd0, A: 0xff}
	first := rules.SpellProfile(rules.SpellGameFirst)
	ground := data.FirstSpellPresentation().BookRows
	var bare uint16
	for id := uint16(1); int(id) <= len(first); id++ {
		if !slices.Contains(ground, id) {
			bare = id
			break
		}
	}
	name := first[bare-1].Name
	chooseSpell(t, e, name)
	if err := spellsSet(e, "spell.icon", hostFile(t, "Bare.png", spellPicturePNG(t, 36, 36, red))); err != nil {
		t.Fatal(err)
	}
	if err := spellsSet(e, "spell.cast", hostFile(t, "Bare-Cast.16a", regCurSheet16A())); err != nil {
		t.Fatal(err)
	}
	files := spellsEdits(e).workspace.Project.Files
	if files["assets/spells/bare.png"] == nil || !bytes.Equal(files["assets/spells/bare-cast.16a"], regCurSheet16A()) {
		t.Fatalf("the project holds no copied pictures: %v", slices.Sorted(func(yield func(string) bool) {
			for k := range files {
				if !yield(k) {
					return
				}
			}
		}))
	}
	want := "target = \"" + name + "\"\nicon = \"assets/spells/bare.png\"\ncast = \"assets/spells/bare-cast.16a\"\n"
	if src := spellsSource(e); !strings.Contains(src, want) {
		t.Fatalf("the spells file:\n%s\nwant the row:\n%s", src, want)
	}
	grid, _ := spellsPageField(e, spellsGridKey)
	v := int64(slices.Index(rules.SpellCatalog(), name) + 1)
	if pic := grid.Icons[v]; pic == nil || pic.RGBAAt(3, 3) != red {
		t.Fatal("the catalog draws no mod icon for a spell the game's book has no picture for")
	}
	if look, ok := spellsPageField(e, "spell.icon.look"); !ok || look.Picture == nil {
		t.Fatal("the page shows no picked icon")
	}
	chooseSpell(t, e, first[ground[0]-1].Name)
	if err := spellsSet(e, "spell.icon", hostFile(t, "first.png", spellPicturePNG(t, 36, 36, red))); err != nil {
		t.Fatal(err)
	}
	book, _ := spellsPageField(e, spellsBookKey)
	if c := book.Book.Cells[0]; !c.Moved || c.Icon == nil || c.Icon.RGBAAt(3, 3) != red {
		t.Fatalf("book cell 1 draws %+v, want the mod's icon pasted", c)
	}

	book2 := words.For("english")
	for _, tc := range []struct{ key, file, want string }{
		{"spell.icon", hostFile(t, "x.16a", regCurSheet16A()), "x.16a"},
		{"spell.icon", hostFile(t, "small.png", spellPicturePNG(t, 10, 10, red)), "10x10"},
		{"spell.burst", hostFile(t, "b.png", spellPicturePNG(t, 36, 36, red)), "b.png"},
		{"spell.burst", hostFile(t, "b.256", []byte("not a sheet")), strings.TrimSuffix(book2.Text("editor.spells.err.sheet"), ".")},
	} {
		if err := spellsSet(e, tc.key, tc.file); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s = %s: %v, want a refusal holding %q", tc.key, filepath.Base(tc.file), err, tc.want)
		}
	}
	if err := spellsSet(e, "spell.icon", ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(spellsSource(e), "first.png") {
		t.Fatal("an emptied icon field left the icon in the row")
	}
}

// otherGameInstall writes a stand-in install of the second game: the five
// archives and the second game's marks, its graphics holding a book atlas of
// colour atlas, a projectile registry giving spell n its cast picture and
// burst, and their sheets. No byte of it is a game's.
func otherGameRoot(t *testing.T, n int, atlas color.RGBA) string {
	t.Helper()
	node := func(name string, v int32) synth.RegNode { return synth.RegNode{Name: name, Kind: 0x02, Int: v} }
	file := func(path string) synth.RegNode { return synth.RegNode{Name: "File", Kind: 0x00, Str: path} }
	plain := synth.Archive([]synth.File{{Path: "a.bin", Data: []byte("x")}})
	files := map[string][]byte{
		MainArchive: plain, ScenarioArchive: plain, WorldArchive: plain, MoviesArchive: plain,
		"allods2.exe": []byte("x"), "scenario.dll": []byte("x"),
		GraphicsArchive: synth.Archive([]synth.File{
			{Path: data.SpellIconAtlasPath, Data: synthBMP(data.SpellIconAtlasW, data.SpellIconAtlasH, atlas)},
			{Path: strings.TrimPrefix(ProjectileRegistry, graphicsPrefix), Data: synth.ProjectilesReg(2,
				[]synth.RegNode{node("ID", int32(data.CastPicture(n))), file("other\\cast"), node("Phases", 1), node("RotationPhases", 1), node("Width", 20), node("Height", 22), node("Palette", 1), node("A16", 1)},
				[]synth.RegNode{node("ID", int32(data.BurstPicture(n))), file("other\\burst"), node("Phases", 1), node("A16", 1)})},
			{Path: data.ProjectilePrefix + "other/cast.16a", Data: regCurSheet16A()},
			{Path: data.ProjectilePrefix + "other/burst.16a", Data: regCurSheet16A()},
		}),
	}
	dir := t.TempDir()
	for name, b := range files {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// «Use the other game» is shut with its reason while the starter lists no
// install of the other game; with one listed it copies the chosen spell's
// icon cell, its cast picture and burst with their registry rows into
// assets/spells and writes the row marked imported, in one step. Publishing
// the mod then warns and still writes it.
func TestSpellsPageCopiesTheOtherGamesPicturesAndPublishWarns(t *testing.T) {
	x, e, _ := spellsPage(t)
	chooseSpell(t, e, "Fire_Arrow")
	absent, ok := spellsPageField(e, spellOtherGameKey)
	if !ok || absent.Action || !strings.Contains(absent.TextValue, x.word("editor.spells.othergame.absent")) {
		t.Fatalf("with no install of the other game the copy shows %+v", absent)
	}

	second := data.SpellPresentationNamed("rom2")
	var name string
	var n int
	for _, c := range rules.SpellCatalog() {
		id1, in1 := rules.SpellNumber(rules.SpellGameFirst, c)
		id2, in2 := rules.SpellNumber(rules.SpellGameSecond, c)
		if in1 && in2 && slices.Contains(second.BookRows, uint16(id2)) {
			name, n = c, id2
			if !slices.Contains(data.FirstSpellPresentation().BookRows, uint16(id1)) {
				break
			}
		}
	}
	if name == "" {
		t.Fatal("no spell of both games has a book cell in the second")
	}
	purple := color.RGBA{R: 0x80, B: 0x80, A: 0xff}
	root := otherGameRoot(t, n, purple)
	x.testPlay.Installs = func() []string { return []string{root} }
	x.testPlay.ModsFolder = filepath.Join(t.TempDir(), "mods")
	x.spellsLook().otherTried = false
	chooseSpell(t, e, name)
	if f, ok := spellsPageField(e, spellOtherGameKey); !ok || !f.Action {
		t.Fatalf("with the other game listed the copy shows %+v", f)
	}
	if err := spellsSet(e, spellOtherGameKey, ""); err != nil {
		t.Fatal(err)
	}
	stem := strings.ToLower(rules.SpellKey(name))
	files := spellsEdits(e).workspace.Project.Files
	icon, err := witness.Decode(bytes.NewReader(files["assets/spells/"+stem+".png"]))
	if err != nil || icon.Bounds().Dx() != data.SpellIconSide || color.RGBAModel.Convert(icon.At(4, 4)) != purple {
		t.Fatalf("the copied icon (%v) is not the other game's cell", err)
	}
	for _, field := range []string{"cast", "burst"} {
		if !bytes.Equal(files["assets/spells/"+stem+"-"+field+".16a"], regCurSheet16A()) {
			t.Fatalf("the %s sheet is not the other game's", field)
		}
	}
	want := "icon = \"assets/spells/" + stem + ".png\"\ncast = \"assets/spells/" + stem + "-cast.16a\"\n" +
		"cast_row = { phases = 1, rotation_phases = 1, width = 20, height = 22, palette = 1, flip = 0 }\n" +
		"burst = \"assets/spells/" + stem + "-burst.16a\"\nburst_row = { phases = 1, rotation_phases = 16, width = 64, height = 64, palette = 0, flip = 0 }\nimported = true\n"
	if src := spellsSource(e); !strings.Contains(src, want) {
		t.Fatalf("the spells file:\n%s\nwant:\n%s", src, want)
	}
	if f, ok := spellsPageField(e, spellFieldPrefix+"imported"); !ok || !strings.Contains(f.TextValue, x.word("editor.spells.imported")) {
		t.Fatal("the page does not say the pictures are copied from the other game")
	}

	if _, err := spellsEdits(e).SetWorkspaceField("package", "save", ""); err != nil {
		t.Fatal(err)
	}
	s := spellsEdits(e)
	if _, err := s.SetWorkspaceField("package", "publish", ""); err != nil {
		t.Fatal(err)
	}
	s = spellsEdits(e)
	if note := s.SaveNote(); !strings.Contains(note, x.word("editor.mod.published")) || !strings.Contains(note, strings.SplitN(x.word("editor.mod.publishedimported"), "(", 2)[0]) || !strings.Contains(note, rules.SpellKey(name)) {
		t.Fatalf("the publish note %q carries no warning about the copied pictures", note)
	}
	if _, err := os.Stat(filepath.Join(x.testPlay.ModsFolder, mod.PackedName(x.mod.id))); err != nil {
		t.Fatalf("publish wrote no packed mod: %v", err)
	}
}
