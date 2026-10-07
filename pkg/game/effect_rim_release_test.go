package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func effectRimOutputPath(root, output string) (string, error) {
	if !filepath.IsAbs(output) {
		return "", fmt.Errorf("absolute AGAINROM_EFFECT_RIM_WITNESS_DIR required")
	}
	root, err := editorPhysicalDirectory(root)
	if err != nil {
		return "", err
	}
	output = filepath.Join(output, fmt.Sprintf("%x", sha256.Sum256([]byte(filepath.ToSlash(root)))))
	parent, suffix := filepath.Clean(output), ""
	for {
		if _, err := os.Lstat(parent); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", err
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", fmt.Errorf("effect rim output has no existing ancestor: %s", output)
		}
		suffix = filepath.Join(filepath.Base(parent), suffix)
		parent = next
	}
	parent, err = editorPhysicalDirectory(parent)
	if err != nil {
		return "", err
	}
	output = filepath.Join(parent, suffix)
	rel, err := filepath.Rel(root, output)
	if err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
		return "", fmt.Errorf("effect rim output is inside the installed asset root")
	}
	return output, nil
}

func effectRimWitnessDir(t *testing.T, f *FrontEnd) string {
	t.Helper()
	root, err := filepath.Abs(f.Archives.Root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := effectRimOutputPath(root, os.Getenv("AGAINROM_EFFECT_RIM_WITNESS_DIR"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

type effectRimDrawReceipt struct {
	Kind, SourceSHA256 string
	Geometry           [6]float64
	Scale              [4]float32
	Unit, Mark         bool
	Opaque, Pink       int
}

func effectRimArt(t *testing.T, v *ui.Viewer, e ui.MapEntity) (*image.RGBA, []effectRimDrawReceipt) {
	t.Helper()
	draws, err := v.HeadlessArtDraws()
	if err != nil {
		t.Fatal(err)
	}
	cam := v.Camera()
	art := image.NewRGBA(image.Rect(0, 0, cam.ViewW, cam.ViewH))
	var receipts []effectRimDrawReceipt
	for _, d := range draws {
		if d.Filter != ebiten.FilterNearest {
			t.Fatal("art witness requires production nearest sampling")
		}
		r := effectRimDrawReceipt{Kind: d.Kind, SourceSHA256: fmt.Sprintf("%x", sha256.Sum256(d.Pixels.Pix)),
			Scale: [4]float32{d.ColorScale.R(), d.ColorScale.G(), d.ColorScale.B(), d.ColorScale.A()},
			Unit:  d.Kind == "sprite" && d.Frame == e.Frame}
		for _, mark := range e.Marks {
			r.Mark = r.Mark || d.Effect != nil && d.Effect == mark.Sheet.Frame(mark.Mark.Phase)
		}
		for y := range 2 {
			for x := range 3 {
				r.Geometry[y*3+x] = d.Geometry.Element(y, x)
			}
		}
		for i := 0; i < len(d.Pixels.Pix); i += 4 {
			p := d.Pixels.Pix[i : i+4]
			if p[3] > 0 {
				r.Opaque++
				if p[0] > p[1] && p[2] > p[1] {
					r.Pink++
				}
			}
		}
		receipts = append(receipts, r)
		corners := []image.Point{d.Pixels.Rect.Min, {X: d.Pixels.Rect.Max.X, Y: d.Pixels.Rect.Min.Y}, d.Pixels.Rect.Max, {X: d.Pixels.Rect.Min.X, Y: d.Pixels.Rect.Max.Y}}
		left, top, right, bottom := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for _, p := range corners {
			x, y := d.Geometry.Apply(float64(p.X), float64(p.Y))
			left, top, right, bottom = math.Min(left, x), math.Min(top, y), math.Max(right, x), math.Max(bottom, y)
		}
		box := image.Rect(int(math.Floor(left)), int(math.Floor(top)), int(math.Ceil(right)), int(math.Ceil(bottom))).Intersect(art.Rect)
		inverse := d.Geometry
		inverse.Invert()
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				sx, sy := inverse.Apply(float64(x)+0.5, float64(y)+0.5)
				p := image.Pt(int(math.Floor(sx)), int(math.Floor(sy)))
				if !p.In(d.Pixels.Rect) {
					continue
				}
				c, under := d.Pixels.RGBAAt(p.X, p.Y), art.RGBAAt(x, y)
				a := float64(c.A) * float64(r.Scale[3]) / 255
				channel := func(s, dest uint8, scale float32) uint8 {
					return uint8(min(max(math.Round(float64(s)*float64(scale)+float64(dest)*(1-a)), 0), 255))
				}
				if d.Kind == "shadow" {
					c.R, c.G, c.B = 0, 0, 0
				}
				art.SetRGBA(x, y, color.RGBA{channel(c.R, under.R, r.Scale[0]), channel(c.G, under.G, r.Scale[1]), channel(c.B, under.B, r.Scale[2]), channel(c.A, under.A, r.Scale[3])})
			}
		}
	}
	return art, receipts
}

func releaseEffectRims(t *testing.T, f *FrontEnd, m *alm.Map, minV int, dir string) {
	t.Run("generated Shield", func(t *testing.T) { releaseEffectRimSpell(t, f, m, minV, dir, 18) })
	t.Run("starting Bless book", func(t *testing.T) { releaseEffectRimSpell(t, f, m, minV, dir, 23) })
}

func releaseEffectRimSpell(t *testing.T, f *FrontEnd, m *alm.Map, minV int, dir string, spell uint16) {
	t.Helper()
	party := f.ChargenParty(ui.ChargenResult{Name: "Effect art", Choices: []int{0, 1, 0}, Stats: []int{30, 30, 40, 40}})
	stockBook := party[0].KnownSpells
	fixture := "installed generated mage"
	if spell == 23 {
		party[0].KnownSpells |= 1 << spell
		fixture = "installed generated mage with explicit starting Bless23 book test input; not stock chargen knowledge"
	}
	app := f.App("effect rim witness")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for i := 0; i < 600; i++ {
		if _, _, up := f.LiveNotice(); up {
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
			break
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if live == nil || len(live.mission.ids) != 1 {
		t.Fatal("installed mission10 did not open the generated mage")
	}
	id := live.mission.ids[0]
	mage, _ := live.entity(id)
	if mage.KnownSpells&(1<<spell) == 0 {
		t.Fatalf("starting mage book%#x does not know spell%d", mage.KnownSpells, spell)
	}
	cam := live.view.Camera()
	centre := func() {
		mage, _ = live.entity(id)
		y := int(mage.Y)*32 + statusBarAnchorLift(m, minV, int(mage.X), int(mage.Y))
		cam.CenterOn(float64(mage.X)*32+16, float64(y)+16)
		cam.X, cam.Y = math.Floor(cam.X), math.Floor(cam.Y)
		cam.Clamp()
	}
	centre()
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	for range 16 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	centre()
	if _, _, err := app.HeadlessSpellPoint(uint32(spell)); err != nil {
		if err := app.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
	click := func(x, y int) {
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
	}
	sx, sy, err := app.HeadlessSpellPoint(uint32(spell))
	if err != nil {
		t.Fatal(err)
	}
	click(sx, sy)
	mx, my, err := app.HeadlessEntityPoint(uint32(id))
	if err != nil {
		t.Fatal(err)
	}
	start, mana := live.world.Tick(), mage.Mana
	click(mx, my)
	if got := live.pending; len(got) != 1 || got[0].Kind != sim.KindCast || got[0].Entity != id || got[0].X != int32(id) || got[0].Y != int32(spell) {
		t.Fatalf("ordinary App self-target queued%+v", got)
	}
	for i := 0; !live.world.HasEffectSpell(id, spell); i++ {
		if i == 256 {
			t.Fatalf("App spell%d did not attach in256 frames", spell)
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		centre()
		gx := (int(mage.X)-3)*32 + 16 - int(cam.X)
		gy := int(mage.Y)*32 + statusBarAnchorLift(m, minV, int(mage.X)-3, int(mage.Y)) + 16 - int(cam.Y)
		for _, edge := range []string{"right-press", "right-release"} {
			if err := app.HeadlessPointer(edge, gx, gy); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, selected := live.view.SelectedUnit(); selected {
		t.Fatal("ordinary right clicks did not deselect the mage")
	}
	centre()
	var projected ui.MapEntity
	for _, e := range live.entityDraws() {
		if e.ID == uint32(id) {
			projected = e
		}
	}
	mage, _ = live.entity(id)
	if projected.SpellFX <= 0 || len(projected.Marks) == 0 || mage.Mana >= mana || !live.world.HasEffectSpell(id, spell) || projected.TokenSize > 1 || live.view.Mode() != ui.ModeDisplaced {
		t.Fatalf("lasting cast: FX%d marks%d mana%d->%d effects%+v", projected.SpellFX, len(projected.Marks), mana, mage.Mana, live.world.ActiveEffects())
	}
	before, effects := live.world.Hash(), live.world.ActiveEffects()
	var firstArt []byte
	var firstDraws []effectRimDrawReceipt
	proof := map[string]any{"root": f.Archives.Root, "mission": 10, "spell": spell, "known_spells": mage.KnownSpells,
		"fixture": fixture, "generated_known_spells": stockBook,
		"tick_before": start, "tick_effect": live.world.Tick(), "mana_before": mana, "mana_after": mage.Mana,
		"spell_fx": projected.SpellFX, "school": projected.SpellFXSchool, "marks": len(projected.Marks), "effects": effects,
		"world_hash": fmt.Sprintf("%x", before), "art_limit": "CPU raster of production drawArt submissions; excludes terrain, overlays, shroud and HUD; no GPU readback",
		"rim_limit": "HeadlessStatusBarFrame omits Effect entries", "pink_test": "nontransparent source pixels with R>G and B>G"}
	for i, show := range []bool{false, true, false} {
		if i == 0 {
			if on, _ := live.view.UnitOverlay(); on {
				t.Fatal("fresh App enabled diagnostic units")
			}
		} else {
			live.view.SetUnits(show, nil)
		}
		frame, under, err := live.view.HeadlessStatusBarFrame()
		if err != nil {
			t.Fatal(err)
		}
		border := image.Pt(int(mage.X)*32+8-int(cam.X), int(mage.Y)*32+31+statusBarAnchorLift(m, minV, int(mage.X), int(mage.Y))-int(cam.Y))
		if !border.In(frame.Rect) {
			t.Fatal("rim witness border is outside the viewport")
		}
		want := under.RGBAAt(border.X, border.Y)
		if show {
			switch projected.SpellFXSchool {
			case 4:
				want = color.RGBA{230, 230, 106, 255}
			case 5:
				want = color.RGBA{140, 255, 154, 255}
			default:
				t.Fatalf("installed spell%d school%d lacks the independent rim oracle", spell, projected.SpellFXSchool)
			}
			if want == under.RGBAAt(border.X, border.Y) {
				t.Fatal("installed border cannot distinguish the diagnostic color from its underlay")
			}
		}
		if got := frame.RGBAAt(border.X, border.Y); got != want {
			t.Errorf("toggle%d diagnostics%v border%v=%v, want%v", i, show, border, got, want)
		}
		art, draws := effectRimArt(t, live.view, projected)
		unit, marks, pink, drawnPink := 0, 0, 0, 0
		for _, d := range draws {
			if d.Unit && d.Opaque > 0 && d.Scale[3] > 0 {
				unit++
			}
			if d.Mark && d.Opaque > 0 && d.Scale[3] > 0 {
				marks++
				pink += d.Pink
			}
		}
		for p := 0; p < len(art.Pix); p += 4 {
			if art.Pix[p+3] > 0 && art.Pix[p] > art.Pix[p+1] && art.Pix[p+2] > art.Pix[p+1] {
				drawnPink++
			}
		}
		t.Logf("toggle%d school%d FX%d unit%d marks%d pink%d rasterPink%d", i, projected.SpellFXSchool, projected.SpellFX, unit, marks, pink, drawnPink)
		if unit != 1 || marks != len(projected.Marks) || spell == 23 && (pink == 0 || drawnPink == 0) {
			t.Fatal("production painter omitted nonempty installed unit/mark art or Bless pink pixels")
		}
		if i == 0 {
			firstArt, firstDraws = bytes.Clone(art.Pix), draws
		} else if !bytes.Equal(firstArt, art.Pix) || !reflect.DeepEqual(firstDraws, draws) {
			t.Fatal("diagnostic toggle changed production art submissions or their CPU raster")
		}
		if live.world.Hash() != before || !reflect.DeepEqual(live.world.ActiveEffects(), effects) {
			t.Fatal("presentation witness changed World or lasting effects")
		}
		stage := fmt.Sprintf("spell%d-toggle%d-%v", spell, i, show)
		writeStatusBarFrame(t, filepath.Join(dir, stage+"-rim.png"), frame)
		writeStatusBarFrame(t, filepath.Join(dir, stage+"-art.png"), art)
		proof[stage] = map[string]any{"border": border, "pixel": want, "under_pixel": under.RGBAAt(border.X, border.Y), "draws": draws, "raster_pink_pixels": drawnPink, "art_sha256": fmt.Sprintf("%x", sha256.Sum256(art.Pix)), "rim_sha256": fmt.Sprintf("%x", sha256.Sum256(frame.Pix))}
	}
	raw, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("spell%d-receipts.json", spell)), raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestEffectRimOutputRequiresOutsideAbsoluteDirectory(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "install")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, output string
		valid        bool
	}{
		{"missing", "", false},
		{"relative", "out", false},
		{"root", root, false},
		{"inside", filepath.Join(root, "new", "out"), false},
		{"outside", filepath.Join(base, "new", "out"), true},
	}
	if runtime.GOOS == "windows" {
		for drive := 'A'; drive <= 'Z'; drive++ {
			path := fmt.Sprintf("%c:/", drive)
			if _, err := os.Lstat(path); os.IsNotExist(err) {
				cases = append(cases, struct {
					name, output string
					valid        bool
				}{"absent-drive", filepath.Join(path, "new", "out"), false})
				break
			}
		}
		if len(cases) != 6 {
			t.Fatal("no absent drive root for the bounded ancestor control")
		}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := effectRimOutputPath(root, tc.output)
			if (err == nil) != tc.valid {
				t.Fatalf("output%q resolved%q error%v", tc.output, out, err)
			}
			if _, err := os.Stat(filepath.Join(root, "new")); !os.IsNotExist(err) {
				t.Fatal("output validation wrote inside the install")
			}
		})
	}
}
