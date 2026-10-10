package game

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mod"
	"againrom/pkg/render/terrain"
)

// mapSource is an install of a few named entries.
type mapSource map[string][]byte

func (m mapSource) ReadFile(name string) ([]byte, error) {
	if b, ok := m[name]; ok {
		return append([]byte(nil), b...), nil
	}
	return nil, fs.ErrNotExist
}

// weaponNames is a weapon table of names alone; entry 0 is the unwritten row.
type weaponNames []string

func (w weaponNames) Len() int                  { return len(w) }
func (w weaponNames) EntryName(i int) string    { return w[i] }
func (w weaponNames) EntryParams(int) []int32   { return nil }
func (w weaponNames) EntryStrings(int) []string { return nil }

// drawBodySheet draws a body sheet for b from plain shapes: per frame cell a
// filled block in tint for the body and a line for the weapon, its angle
// turning with the direction and the frame, so every cell differs and holds
// visible pixels. Nothing here comes from a game picture.
func drawBodySheet(b mod.BodySheet, tint color.NRGBA) image.Image {
	rows := 1
	for _, n := range []int{b.Move.Frames, b.Attack.Frames, b.Idle.Frames} {
		if n > 0 {
			rows += b.Directions
		}
	}
	cols := max(b.StandingFrames(), b.Move.Frames, b.Attack.Frames, b.Idle.Frames)
	img := image.NewNRGBA(image.Rect(0, 0, cols*b.Width, rows*b.Height))
	cell := func(col, row int, angle float64, swing float64) {
		x0, y0 := col*b.Width, row*b.Height
		for y := b.Height / 4; y < b.Height-b.Height/8; y++ {
			for x := b.Width/2 - b.Width/8; x < b.Width/2+b.Width/8; x++ {
				img.SetNRGBA(x0+x, y0+y, tint)
			}
		}
		weapon := color.NRGBA{R: 0xe0, G: 0xe0, B: 0x30, A: 0xff}
		length := float64(min(b.Width, b.Height)) / 2.2
		a := angle + swing
		for s := 0.0; s < length; s += 0.5 {
			x := x0 + b.Width/2 + int(math.Round(s*math.Cos(a)))
			y := y0 + b.Height/2 - int(math.Round(s*math.Sin(a)))
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 3; dx++ {
					if image.Pt(x+dx, y+dy).In(image.Rect(x0, y0, x0+b.Width, y0+b.Height)) {
						img.SetNRGBA(x+dx, y+dy, weapon)
					}
				}
			}
		}
	}
	for i := 0; i < b.StandingFrames(); i++ {
		cell(i, 0, float64(i)*math.Pi/8, 0)
	}
	row := 1
	for _, a := range []struct {
		n     int
		swing float64
	}{{b.Move.Frames, 0.15}, {b.Attack.Frames, 0.6}, {b.Idle.Frames, 0.05}} {
		for d := 0; d < b.Directions && a.n > 0; d++ {
			for j := 0; j < a.n; j++ {
				cell(j, row, float64(d)*math.Pi/4, a.swing*float64(j))
			}
			row++
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// writeBodySheets draws both of b's sheets into its mod folder.
func writeBodySheets(t *testing.T, b mod.BodySheet) {
	t.Helper()
	for path, tint := range map[string]color.NRGBA{
		b.Heroes:      {R: 0x30, G: 0x50, B: 0xc0, A: 0xff},
		b.HeroesLight: {R: 0x40, G: 0xa0, B: 0x50, A: 0xff},
	} {
		p := filepath.Join(b.Dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, encodePNG(t, drawBodySheet(b, tint)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const testBodies = `[[body]]
name        = "hammer2h"
heroes      = "bodies/h.png"
heroes_l    = "bodies/hl.png"
frame       = [24, 32]
origin      = [12, 28]
directions  = 8
move        = { frames = 3, ticks = 2 }
attack      = { frames = 4, ticks = [1, 2, 3, 1] }
idle        = { frames = 2, ticks = 5 }
weapon-last = true
selection   = [6, 4, 18, 30]
`

// bodyInstall is a synthetic install: a unit registry with the class an
// unknown body falls back on (1, its own dying class) and the mace body's (10),
// mace and bare-handed sheets under both body directories, and a weapon table
// whose rows 1 to 3 are Hands, Sword and War Hammer.
func bodyInstall(t *testing.T) (mapSource, *terrain.UnitSet, weaponNames, data.BodyList) {
	t.Helper()
	i := func(name string, v int32) synth.RegNode { return synth.RegNode{Name: name, Kind: 0x02, Int: v} }
	reg := synth.UnitsReg([]string{`heroes\unarmed\sprites`, `heroes\clubman\sprites`},
		[]synth.RegNode{i("ID", 1), i("File", 0), i("Dying", 1), i("Width", 40), i("Height", 40), i("CenterX", 20), i("CenterY", 36),
			i("DyingPhases", 2), i("SelectionX1", 1), i("SelectionY1", 2), i("SelectionX2", 39), i("SelectionY2", 38)},
		[]synth.RegNode{i("ID", 10), i("File", 1), i("Dying", 1), i("Width", 40), i("Height", 40), i("CenterX", 20), i("CenterY", 36)},
	)
	sheet := synth.Sheet256(synth.Sheet256Options{Palette: []color.RGBA{{}, {R: 0xff}},
		Frames: []synth.Frame256{{Width: 2, Height: 2}, {Width: 2, Height: 2}}})
	src := mapSource{UnitRegistry: reg}
	for _, dir := range []string{data.HeroDirHeroes, data.HeroDirHeroesLight} {
		for _, b := range []data.HeroBody{data.BodyUnarmed, data.BodyClubman} {
			src[graphicsPrefix+data.HeroSheetPath(dir, b)] = sheet
		}
	}
	units, err := LoadUnits(src)
	if err != nil {
		t.Fatal(err)
	}
	if units.Classes[1] == nil || units.Classes[10] == nil {
		t.Fatalf("the fixture registry holds %d classes", len(units.Classes))
	}
	list := data.NewBodyList(data.BodyUnarmed, data.BodySwordsman, data.BodyAxeman2H, "")
	return src, units, weaponNames{"", "Hands", "Sword", "War Hammer", "Rem"}, list
}

// testBodyData is the mod m's two files over a mod folder with both sheets drawn.
func testBodyData(t *testing.T, weapons string) mod.BodyData {
	t.Helper()
	bodies, err := mod.ParseBodies("m", mod.BodiesFile, []byte(testBodies))
	if err != nil {
		t.Fatal(err)
	}
	bodies[0].Dir = t.TempDir()
	writeBodySheets(t, bodies[0])
	mappings, err := mod.ParseWeaponBodies("m", mod.WeaponBodiesFile, []byte(weapons))
	if err != nil {
		t.Fatal(err)
	}
	return mod.BodyData{Weapons: mappings, Bodies: bodies}
}

const testMappings = "[[weapon]]\nweapon = \"War Hammer\"\nbody = \"clubman\"\n[[weapon]]\nrow = 2\nbody = \"hammer2h\"\n"

// A mapping changes the body of its weapon only, and a supplied sheet builds
// a drawn class under both directories that every selection can draw from.
func TestModBodiesMapWeaponsAndBuildASuppliedBody(t *testing.T) {
	src, units, weapons, list := bodyInstall(t)
	bodies := testBodyData(t, testMappings)
	got, built, err := resolveModBodies(src, units, weapons, list, bodies)
	if err != nil {
		t.Fatal(err)
	}
	for row, want := range map[int]data.HeroBody{1: data.BodyUnarmed, 2: "hammer2h", 3: data.BodyClubman} {
		var e data.Equipment
		e.SetCode(1, data.ItemCode(row))
		if b, ok := data.HeroBodyFor(got, e); b != want || !ok {
			t.Errorf("row %d draws %q %v, want %q", row, b, ok, want)
		}
		if b, _ := data.HeroBodyFor(list, e); row != 1 && b == want {
			t.Errorf("row %d: the list the call was given changed", row)
		}
	}
	if len(built) != 2 {
		t.Fatalf("built %d bodies, want hammer2h under 2 directories", len(built))
	}
	b := bodies.Bodies[0]
	anim := modBodyAnim(b)
	for _, dir := range []string{data.HeroDirHeroes, data.HeroDirHeroesLight} {
		c := built[data.HeroBodyKey(dir, "hammer2h")]
		if c == nil {
			t.Fatalf("no body under %s", dir)
		}
		if len(c.Frames) != anim.Total || anim.Total != 16+8*(3+4+2) {
			t.Fatalf("%s: %d frames, descriptor total %d", dir, len(c.Frames), anim.Total)
		}
		if c.Width != 24 || c.Height != 32 || c.CenterX != 12 || c.CenterY != 28 || c.OwnerShaded ||
			c.Selection != image.Rect(6, 4, 18, 30) || c.Boundary != nil || len(c.Tiers) != 0 {
			t.Fatalf("%s geometry %+v", dir, c)
		}
		if c.Corpse == nil || c.Corpse.Width != 40 || len(c.Corpse.Frames) != 2 {
			t.Fatalf("%s: the fallen body is not the shipped dying body", dir)
		}
		seen := map[*terrain.StaticFrame]bool{}
		for facing := 0; facing < 16; facing++ {
			f, _ := terrain.SelectStandingFrame(c.Anim, len(c.Frames), facing)
			seen[c.Frames[f]] = true
		}
		for oct := 0; oct < 8; oct++ {
			for run := 0; run < len(c.Anim.AttackTrack); run++ {
				f, _, ok := terrain.SelectAttackFrame(c.Anim, len(c.Frames), oct, run)
				if !ok {
					t.Fatalf("%s: no attack frame at octant %d run %d", dir, oct, run)
				}
				seen[c.Frames[f]] = true
			}
			for step := 0; step < len(c.Anim.MoveTrack); step++ {
				f, _ := terrain.SelectUnitFrame(c.Anim, len(c.Frames), true, oct, 0, terrain.WalkStepOdometer(step))
				seen[c.Frames[f]] = true
			}
			for tick := 0; tick < len(c.Anim.IdleTrack); tick++ {
				f, _ := terrain.SelectUnitFrame(c.Anim, len(c.Frames), false, oct, tick, 0)
				seen[c.Frames[f]] = true
			}
		}
		if len(seen) != len(c.Frames) {
			t.Fatalf("%s: the selections reach %d of %d frames", dir, len(seen), len(c.Frames))
		}
	}
	if m, ok := got.ModBodyOf("hammer2h"); !ok || !m.WeaponLast {
		t.Fatalf("ModBodyOf(hammer2h) = %+v %v", m, ok)
	}
}

// The choice crosses a front end's filesystem and reads back whole.
func TestBodyChoiceRoundTripsThroughTheFilesystem(t *testing.T) {
	list := data.NewBodyList("unarmed", "swordsman", "axeman2h").
		WithWeaponBody(3, data.BodyClubman).WithWeaponBody(2, "hammer2h").
		WithModBody("hammer2h", data.ModBody{WeaponLast: true}).WithModBody("hammer2h_", data.ModBody{})
	src := mapSource{HeroPictureAddress: []byte("unarmed\r\nswordsman\r\naxeman2h\r\n"), ModBodyChoiceAddress: encodeBodyChoice(list)}
	got, ok := ReadBodyList(src)
	if !ok || got.Len() != 3 {
		t.Fatalf("ReadBodyList: %v %d", ok, got.Len())
	}
	if string(encodeBodyChoice(got)) != string(encodeBodyChoice(list)) {
		t.Fatalf("read back %q, wrote %q", encodeBodyChoice(got), encodeBodyChoice(list))
	}
	plain, _ := ReadBodyList(mapSource{HeroPictureAddress: []byte("unarmed\r\n")})
	if len(plain.WeaponRows()) != 0 || len(plain.ModBodies()) != 0 {
		t.Fatal("an install with no choice read one")
	}
}

// Every refusal names the mod, the file and the line.
func TestModBodiesRefusalsNameModFileAndLine(t *testing.T) {
	src, units, weapons, list := bodyInstall(t)
	cases := []struct {
		name, mappings string
		edit           func(*mod.BodyData)
		want           string
	}{
		{"unknown weapon", "[[weapon]]\nweapon = \"Great Mace\"\nbody = \"clubman\"\n", nil,
			`mod "m": data/weapon-bodies.toml:2: the weapon table has no row named "Great Mace"`},
		{"unknown row", "[[weapon]]\nrow = 9\nbody = \"clubman\"\n", nil,
			`mod "m": data/weapon-bodies.toml:2: the weapon table has no row 9`},
		{"name and row disagree", "[[weapon]]\nweapon = \"War Hammer\"\nrow = 2\nbody = \"clubman\"\n", nil,
			`mod "m": data/weapon-bodies.toml:3: row 2 is "Sword", not "War Hammer" (row 3)`},
		{"twice", "[[weapon]]\nweapon = \"Sword\"\nbody = \"clubman\"\n[[weapon]]\nrow = 2\nbody = \"clubman\"\n", nil,
			`mod "m": data/weapon-bodies.toml:4: weapon row 2 already has a body from mod "m" (data/weapon-bodies.toml:1)`},
		{"blank row", "[[weapon]]\nweapon = \"Rem\"\nbody = \"clubman\"\n", nil,
			`mod "m": data/weapon-bodies.toml:1: weapon row 4 has no body in the install's body list`},
		{"body not shipped", "[[weapon]]\nweapon = \"Sword\"\nbody = \"pikeman\"\n", nil,
			`mod "m": data/weapon-bodies.toml:3: body "pikeman" is not a body the install ships under both heroes and heroes_l`},
		{"shield form", "[[weapon]]\nweapon = \"Sword\"\nbody = \"clubman_\"\n", nil,
			`mod "m": data/weapon-bodies.toml:3: body "clubman_" ends in the shield suffix`},
		{"shipped name supplied", "", func(d *mod.BodyData) { d.Bodies[0].Name = "clubman" },
			`mod "m": data/bodies.toml:2: body "clubman" is a body the install ships`},
		{"missing picture", "", func(d *mod.BodyData) { d.Bodies[0].HeroesLight = "bodies/none.png" },
			`mod "m": data/bodies.toml:4: heroes_l "bodies/none.png": `},
		{"not a picture", "", func(d *mod.BodyData) {
			writeFile(t, filepath.Join(d.Bodies[0].Dir, "bodies", "h.png"), []byte("text"))
		}, `mod "m": data/bodies.toml:3: heroes "bodies/h.png": not a readable PNG`},
		{"rows short", "", func(d *mod.BodyData) {
			b := d.Bodies[0]
			b.Idle.Frames = 0
			short := drawBodySheet(b, color.NRGBA{R: 0xff, A: 0xff})
			writeFile(t, filepath.Join(b.Dir, "bodies", "h.png"), encodePNG(t, short))
		}, `mod "m": data/bodies.toml:3: heroes "bodies/h.png": the 384x544 picture does not cover idle direction 0 frame 0 (the cell at column 0, row 17 of 24x32 frames)`},
		{"a blank frame", "", func(d *mod.BodyData) {
			b := d.Bodies[0]
			img := drawBodySheet(b, color.NRGBA{G: 0xff, A: 0xff}).(*image.NRGBA)
			cell := image.Rect(1*b.Width, 22*b.Height, 2*b.Width, 23*b.Height)
			for y := cell.Min.Y; y < cell.Max.Y; y++ {
				for x := cell.Min.X; x < cell.Max.X; x++ {
					img.SetNRGBA(x, y, color.NRGBA{})
				}
			}
			writeFile(t, filepath.Join(b.Dir, "bodies", "hl.png"), encodePNG(t, img))
		}, `mod "m": data/bodies.toml:4: heroes_l "bodies/hl.png": idle direction 5 frame 1 (the cell at column 1, row 22) holds no visible pixel`},
	}
	for _, c := range cases {
		d := testBodyData(t, c.mappings)
		if c.edit != nil {
			c.edit(&d)
		}
		_, _, err := resolveModBodies(src, units, weapons, list, d)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	if _, _, err := resolveModBodies(src, units, weapons, data.BodyList{}, testBodyData(t, testMappings)); err == nil {
		t.Error("an install with no body list took the mods' bodies")
	}
	var unknown *modItemFileError
	_, _, err := resolveModBodies(src, units, weapons, list, testBodyData(t, "[[weapon]]\nrow = 9\nbody = \"clubman\"\n"))
	if !errors.As(err, &unknown) {
		t.Errorf("the refusal %v is not a file error", err)
	}
}

func writeFile(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
