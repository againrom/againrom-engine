package game

// A mod's spell pictures in the game: the book icon in the book cell and the
// cast picture and burst in flight, on both games. Every picture, sheet and
// registry here is synthetic, made in the test.

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/internal/witness"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// spellPicturePNG is a w x h PNG of one colour.
func spellPicturePNG(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return witness.Bytes(t, img)
}

// spellPictureSheet256 is a .256 sheet of frames 2x1 frames, each pixel of
// colour c through the sheet's own table.
func spellPictureSheet256(frames int, c color.RGBA) []byte {
	pal := make([]color.RGBA, 8)
	pal[3] = c
	var out []synth.Frame256
	for range frames {
		out = append(out, synth.Frame256{Width: 2, Height: 1, Pixels: []synth.Pixel256{{Index: 3, Opaque: true}, {Index: 3, Opaque: true}}})
	}
	return synth.Sheet256(synth.Sheet256Options{Palette: pal, Frames: out})
}

// spellPicturesGame is a game's spell table and a graphics container with
// the book atlas, a projectile registry whose rows give the cast picture of
// spell n and picture 1 an install sheet, and those sheets.
func spellPicturesGame(t *testing.T, spells string, n int) (*mapload.Table, *vfs.FS) {
	t.Helper()
	g := data.SpellPresentationNamed(spells).Game
	rows := dbCollection{{}}
	for _, s := range rules.SpellProfile(g) {
		rows = append(rows, dbEntry{name: strings.ReplaceAll(s.Name, "_", " "), params: []int32{0, 7, 1, 0, 1, 0, 5, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0}})
	}
	node := func(name string, v int32) synth.RegNode { return synth.RegNode{Name: name, Kind: 0x02, Int: v} }
	file := func(path string) synth.RegNode { return synth.RegNode{Name: "File", Kind: 0x00, Str: path} }
	install := spellPictureSheet256(1, color.RGBA{G: 0xc0, A: 0xff})
	fs := sackGraphicsContainers(t, []synth.File{
		{Path: data.SpellIconAtlasPath, Data: synthBMP(data.SpellIconAtlasW, data.SpellIconAtlasH, color.RGBA{B: 0xff, A: 0xff})},
		{Path: strings.TrimPrefix(ProjectileRegistry, graphicsPrefix), Data: synth.ProjectilesReg(2,
			[]synth.RegNode{node("ID", int32(data.CastPicture(n))), file("install\\cast"), node("Phases", 1), node("Palette", 1), node("Homing", 1)},
			[]synth.RegNode{node("ID", 1), file("install\\shot"), node("Phases", 1), node("Palette", 1)})},
		{Path: data.ProjectilePrefix + "install/cast.256", Data: install},
		{Path: data.ProjectilePrefix + "install/shot.256", Data: install},
	})
	return &mapload.Table{Edition: &base.Edition{Spells: spells}, Spells: rows}, fs
}

// spellPicturesFront is a front end of the game over table and fs, its
// projectile art and flight inputs loaded as a run loads them.
func spellPicturesFront(t *testing.T, table *mapload.Table, fs *vfs.FS) *FrontEnd {
	t.Helper()
	t2 := *table
	game := base.GameROM1
	if mapload.SpellGameOf(table) == rules.SpellGameSecond {
		game = base.GameROM2
	}
	presentation := data.SpellPresentationNamed(game.Edition().Spells)
	in, err := loadFlightInputs(fs, presentation, filesOf(game).flight)
	if err != nil {
		t.Fatal(err)
	}
	t2.FlightInputs = in
	a := &Archives{Containers: fs, Base: base.Match{Profile: base.Profile{Game: game}}}
	return &FrontEnd{InstallResources: InstallResources{Table: &t2, Archives: a, Projectiles: loadProjectilesOrNil(fs, presentation.Game)}}
}

// spellPicturesMod is the spell rows of a mod folder holding files, read as
// the mods' run reads them.
func spellPicturesMod(t *testing.T, files map[string][]byte) mod.SpellData {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "m")
	for rel, b := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	kind, _ := mod.WrittenKind(mod.SpellsFile)
	var d mod.Data
	if err := kind.ReadHeld(mod.DataReader{Mod: "m", Dir: dir}, &d, mod.SpellsFile, files[mod.SpellsFile]); err != nil {
		t.Fatal(err)
	}
	return d.Spells
}

// spellPictureBook is the book of a hero knowing every spell of game's
// table, built by the mission's route: each cell's icon through the book
// icon cut and the cells marked for the bar's paste.
func spellPictureBook(f *FrontEnd, atlas []uint16, src entrySource) []ui.SpellEntry {
	table := bookScrollTable()
	e := sim.Entity{Mind: 30, Book: sim.Spellbook{State: sim.BookPresent}}
	for id := uint16(1); id <= 28; id++ {
		sim.LearnBookSpell(&e, id, table)
	}
	strip := loadSpellAtlas(src)
	own := modSpellIcons(f.Table)
	icon := func(id uint16) *image.RGBA { return cutBookIcon(strip, atlas, own, id) }
	book, _ := selectedSpellbook(testFirstGame, sim.Rules{}, []sim.Entity{e}, table, nil, ui.Words{}, icon, atlas)
	markMovedSpells(book, atlas, atlas, own)
	return book
}

// A mod's row gives a spell with a book cell and one without its book icon,
// its cast picture with the row's sheet keys and its burst: the book draws
// each icon over the spell's cell, the casts draw the mod's frames at the
// spell's two pictures, and the flight reads the mod's rows, on the first
// and the second game. A spell the row leaves out draws as before.
func TestModSpellPicturesDrawInTheBookAndInFlightOnBothGames(t *testing.T) {
	red, gold := color.RGBA{R: 0xd0, A: 0xff}, color.RGBA{R: 0xd0, G: 0xb0, A: 0xff}
	for _, spells := range []string{"rom1", "rom2"} {
		t.Run(spells, func(t *testing.T) {
			g := data.SpellPresentationNamed(spells).Game
			ground := data.SpellPresentationNamed(spells).BookRows
			n := ground[0]
			var bare uint16
			for id := uint16(1); int(id) <= len(rules.SpellProfile(g)); id++ {
				if !slices.Contains(ground, id) {
					bare = id
					break
				}
			}
			table, fs := spellPicturesGame(t, spells, int(n))
			before := spellPicturesFront(t, table, fs)
			plain := spellPictureBook(before, ground, fs)

			row := func(id uint16, rest string) string {
				return "[[spell]]\ntarget = \"" + rules.SpellProfile(g)[id-1].Name + "\"\n" + rest
			}
			src := row(n, "icon = \"assets/spells/a.png\"\ncast = \"assets/spells/a-cast.16a\"\ncast_row = { rotation_phases = 1, width = 30 }\nburst = \"assets/spells/a-burst.256\"\n")
			files := map[string][]byte{
				"assets/spells/a.png":       spellPicturePNG(t, 36, 36, red),
				"assets/spells/a-cast.16a":  regCurSheet16A(),
				"assets/spells/a-burst.256": spellPictureSheet256(3, gold),
			}
			if bare != 0 {
				src += row(bare, "icon = \"assets/spells/b.png\"\n")
				files["assets/spells/b.png"] = spellPicturePNG(t, 36, 36, gold)
			}
			files[mod.SpellsFile] = []byte(src)
			f := spellPicturesFront(t, table, fs)
			if err := f.SetModSpells(spellPicturesMod(t, files)); err != nil {
				t.Fatal(err)
			}

			book := spellPictureBook(f, ground, fs)
			for i, id := range ground {
				switch {
				case id == n:
					if !book[i].Moved || book[i].Icon == nil || book[i].Icon.RGBAAt(5, 5) != red {
						t.Fatalf("cell %d of spell %d: moved %v, icon %v, want the mod's red icon pasted", i, id, book[i].Moved, book[i].Icon)
					}
				case book[i].Moved || !slices.Equal(iconPix(book[i].Icon), iconPix(plain[i].Icon)):
					t.Fatalf("cell %d of spell %d changed with no picture of its own", i, id)
				}
			}
			if bare != 0 {
				if pic := modSpellIcons(f.Table)[bare]; pic == nil || pic.RGBAAt(0, 0) != gold {
					t.Fatalf("spell %d with no book cell has no mod icon", bare)
				}
			}

			cast, burst := data.CastPicture(int(n)), data.BurstPicture(int(n))
			sheet := f.Projectiles.Sheets[cast]
			want := effectSheetFrames(regCurSheet16A())
			if sheet == nil || len(sheet.Frames) != 1 || !slices.Equal(sheet.Frames[0].Pixels, want[0].Pixels) || sheet.CenterX != 15 || sheet.RotationPhases != 1 {
				t.Fatalf("cast picture %d: %+v, want the mod's one frame, centre 15", cast, sheet)
			}
			if !f.Projectiles.Homes(cast) {
				t.Fatal("the cast picture lost the install row's homing")
			}
			if b := f.Projectiles.Sheets[burst]; b == nil || len(b.Frames) != 3 || b.Phases != 3 || b.Frames[0].Pixels[0] != gold {
				t.Fatalf("burst picture %d: %+v, want the mod's three gold frames", burst, b)
			}
			if before.Projectiles.Sheets[burst] != nil {
				t.Fatal("the fixture install already draws the burst")
			}
			if a, b := before.Projectiles.Sheets[1], f.Projectiles.Sheets[1]; a == nil || b == nil || !slices.Equal(a.Frames[0].Pixels, b.Frames[0].Pixels) {
				t.Fatal("a picture the mod leaves out draws other pixels")
			}
			rows := syntheticPictureRows(g, f.Projectiles)
			if i := slices.IndexFunc(rows, func(r flightPictureRow) bool { return r.ID == int32(burst) }); i < 0 || !rows[i].Drawn || rows[i].Phases != 3 {
				t.Fatalf("the second game's flight rows lack the mod's burst: %+v", rows)
			}
			if in := f.Table.FlightInputs; in == nil && g == rules.SpellGameFirst {
				t.Fatal("the first game's run holds no flight inputs from the registry")
			} else if in != nil {
				if p := in.Pictures[int32(burst)]; !p.Present || p.Phases != 3 {
					t.Fatalf("the first game's flight inputs lack the mod's burst: %+v", p)
				}
				if before.Table.FlightInputs.Pictures[int32(burst)].Present {
					t.Fatal("the install's flight inputs already hold the burst")
				}
			}
		})
	}
}

func iconPix(img *image.RGBA) []byte {
	if img == nil {
		return nil
	}
	return img.Pix
}

// A game run with no mod, and with a mod whose rows name no picture, draws
// the install's book and casts: the same icons, cells and projectile art.
func TestSpellRowsWithoutPicturesDrawAsTheInstall(t *testing.T) {
	ground := data.SpellPresentationNamed("rom1").BookRows
	table, fs := spellPicturesGame(t, "rom1", int(ground[0]))
	before := spellPicturesFront(t, table, fs)
	f := spellPicturesFront(t, table, fs)
	set, inputs := f.Projectiles, f.Table.FlightInputs
	if err := f.SetModSpells(spellPicturesMod(t, map[string][]byte{mod.SpellsFile: []byte("[[spell]]\ntarget = \"Fire_Arrow\"\nmana = 3\n")})); err != nil {
		t.Fatal(err)
	}
	if f.Projectiles != set || f.Table.FlightInputs != inputs || f.Table.Mods.SpellIcons != nil {
		t.Fatal("a mod with no picture changed the projectile art, the flight inputs or the icons")
	}
	a, b := spellPictureBook(before, ground, fs), spellPictureBook(f, ground, fs)
	for i := range a {
		if a[i].Moved != b[i].Moved || !slices.Equal(iconPix(a[i].Icon), iconPix(b[i].Icon)) {
			t.Fatalf("cell %d differs", i)
		}
		if strip := loadSpellAtlas(fs); !slices.Equal(iconPix(a[i].Icon), iconPix(cutSpellIcon(strip, i))) {
			t.Fatalf("cell %d is not the atlas cell", i)
		}
	}
}

// A picture the row names and the mod lacks, an icon of another size and a
// sheet the projectile loader reads no frame from are each refused, naming
// the mod, the file, the line and the picture.
func TestModSpellPicturesRefuseAMissingOrMalformedPicture(t *testing.T) {
	ground := data.SpellPresentationNamed("rom1").BookRows
	table, fs := spellPicturesGame(t, "rom1", int(ground[0]))
	head := "[[spell]]\ntarget = \"Fire_Arrow\"\n"
	for _, tc := range []struct {
		row   string
		files map[string][]byte
		want  string
	}{
		{"icon = \"assets/spells/gone.png\"\n", nil, `spells.toml:3: icon "assets/spells/gone.png"`},
		{"icon = \"assets/spells/s.png\"\n", map[string][]byte{"assets/spells/s.png": spellPicturePNG(t, 10, 12, color.RGBA{A: 0xff})}, `spells.toml:3: icon "assets/spells/s.png"`},
		{"cast = \"assets/spells/c.16a\"\n", map[string][]byte{"assets/spells/c.16a": []byte("not a sheet")}, `spells.toml:3: cast "assets/spells/c.16a": the projectile loader reads no frame from it`},
		{"mana = 2\nburst = \"assets/spells/b.256\"\n", map[string][]byte{"assets/spells/b.256": []byte("x")}, `spells.toml:4: burst "assets/spells/b.256": the projectile loader reads no frame`},
	} {
		files := map[string][]byte{mod.SpellsFile: []byte(head + tc.row)}
		for k, v := range tc.files {
			files[k] = v
		}
		err := spellPicturesFront(t, table, fs).SetModSpells(spellPicturesMod(t, files))
		if err == nil || !strings.Contains(err.Error(), `mod "m"`) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%q: got %v, want %q", tc.row, err, tc.want)
		}
	}
}
