package game

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const modBodiesDir = "../modrt/testdata/mods"

// modBodyFront is a front end on the lawful install under the example mod id,
// or under no mod when id is empty. The mod folder is copied into a temporary
// directory, where the bodies it names receive sheets drawn by code. With
// launcher set the front end takes the launcher's steps (the mod set, then the
// bodies); without it only the body data is applied, so a SAV carries no mod
// mark and its bytes compare with an unmodded run.
func modBodyFront(t *testing.T, id string, launcher bool) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if id == "" {
		return f
	}
	dir := t.TempDir()
	if err := os.CopyFS(filepath.Join(dir, id), os.DirFS(filepath.Join(modBodiesDir, id))); err != nil {
		t.Fatal(err)
	}
	entries, err := mod.Resolve(dir, []string{id})
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range res.Bodies.Bodies {
		writeBodySheets(t, b)
	}
	if launcher {
		if err := f.SetMods(res.Rules, res.Set, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SetModBodies(res.Bodies); err != nil {
		t.Fatal(err)
	}
	return f
}

// warHammerCode is the iron War Hammer of the install's weapon table, found by
// its row name.
func warHammerCode(t *testing.T, f *FrontEnd) uint16 {
	t.Helper()
	weapons := f.Table.Weapons
	for row := 1; row < weapons.Len(); row++ {
		if weapons.EntryName(row) == "War Hammer" {
			return uint16(data.ComposeItemCode(0, 1, 0, row))
		}
	}
	t.Fatal("the weapon table names no War Hammer")
	return 0
}

// modBodyCase is one way the War Hammer hero is drawn.
type modBodyCase struct {
	mod  string
	body data.HeroBody
}

var modBodyCases = []modBodyCase{
	{"", "axeman2h"},
	{"war-hammer-mace", "clubman"},
	{"war-hammer-body", "hammer2h"},
}

// TestReleaseModBodyDrawsTheWarHammerHero opens mission 10 twice per case: once
// with the hero already wielding the War Hammer, once with a bare-handed hero
// who equips it. Both draw the body the case names; the class key the world
// holds is the one it holds without a mod in every case.
func TestReleaseModBodyDrawsTheWarHammerHero(t *testing.T) {
	shippedClass, equippedClass := int32(-1), int32(-1)
	for _, c := range modBodyCases {
		t.Run(caseName(c.mod), func(t *testing.T) {
			// Opened wielding the hammer.
			f := modBodyFront(t, c.mod, true)
			code := warHammerCode(t, f)
			_, id, name, class := hammerHeroMission(t, f)
			if name != c.body {
				t.Fatalf("the War Hammer hero resolves to %q, want %q", name, c.body)
			}
			checkModBodyArt(t, f.live, id, c.body)
			if shippedClass < 0 {
				shippedClass = class
			}
			if e, _ := f.live.entity(id); class != shippedClass || e.Class != shippedClass {
				t.Fatalf("class key %d, entity class %d, want the shipped %d", class, e.Class, shippedClass)
			}
			writeModBodyRender(t, f, c, f.live.art[id])

			// Bare-handed, then equipping the hammer.
			g := modBodyFront(t, c.mod, true)
			mw, gid := heroSwingMission(t, g, code)
			mw.refreshAppearance()
			checkModBodyArt(t, mw, gid, c.body)
			e, _ := mw.entity(gid)
			if equippedClass < 0 {
				equippedClass = e.Class
			}
			if e.Class != equippedClass {
				t.Fatalf("after equipping, entity class %d, want %d as without a mod", e.Class, equippedClass)
			}
		})
	}
}

// hammerHeroMission opens mission 10 with a hero who already wields the War
// Hammer, returning the hero, the body his equipment resolves to and its class
// key.
func hammerHeroMission(t *testing.T, f *FrontEnd) (*mapWorld, sim.EntityID, data.HeroBody, int32) {
	t.Helper()
	code := warHammerCode(t, f)
	party := f.ChargenParty(ui.ChargenResult{Name: "Hammer", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	hero := &party[0]
	hero.Worn[0], hero.WornItems[0], hero.Weapon = code, mapload.ItemInstanceFromCode(code, f.Table), nil
	name, dir, class, ok := data.HeroAppearance(f.Bodies, equipmentFromSlots(hero.Worn), false, false)
	if !ok {
		t.Fatal("the War Hammer hero resolves no body")
	}
	hero.Class, hero.Body, hero.BodyDir = class, string(name), dir
	a := f.App("mod body")
	t.Cleanup(a.StopAudio)
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	return f.live, equipmentReturnHero(t, f), name, class
}

func caseName(id string) string {
	if id == "" {
		return "no-mod"
	}
	return id
}

// checkModBodyArt fails unless the hero id is drawn with body, the class
// record the bundle holds for that body under one of the two armour
// directories (the armour the mission dresses the hero in picks which).
func checkModBodyArt(t *testing.T, mw *mapWorld, id sim.EntityID, body data.HeroBody) {
	t.Helper()
	got := mw.art[id]
	drawn := "a class outside the body set"
	for _, dir := range []string{data.HeroDirHeroes, data.HeroDirHeroesLight} {
		key := data.HeroBodyKey(dir, body)
		if c := mw.units.Bodies[key]; c != nil && c == got {
			return
		}
	}
	for key, c := range mw.units.Bodies {
		if c == got {
			drawn = key
		}
	}
	t.Fatalf("hero %d is drawn with %s, want the %s body", id, drawn, body)
}

// writeModBodyRender writes the hero's standing frames (eight facings) and one
// swing (octant 2) under AGAINROM_MOD_BODY_OUT, when set, for the owner. The
// pictures leave no file in the repository.
func writeModBodyRender(t *testing.T, f *FrontEnd, c modBodyCase, art *terrain.UnitClass) {
	t.Helper()
	out := os.Getenv("AGAINROM_MOD_BODY_OUT")
	if out == "" || art == nil {
		return
	}
	if !filepath.IsAbs(out) {
		t.Fatal("AGAINROM_MOD_BODY_OUT must be absolute")
	}
	var stand, swing []*image.RGBA
	for facing := 0; facing < 8; facing++ {
		if i, m := terrain.SelectStandingFrame(art.Anim, len(art.Frames), facing); i >= 0 && i < len(art.Frames) {
			stand = append(stand, framePicture(art.Frames[i], m))
		}
	}
	for run := 0; run < len(art.Anim.AttackTrack); run++ {
		if i, m, ok := terrain.SelectAttackFrame(art.Anim, len(art.Frames), 2, run); ok {
			swing = append(swing, framePicture(art.Frames[i], m))
		}
	}
	dir := filepath.Join(out, filepath.Base(f.Archives.Root))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, row := range map[string][]*image.RGBA{"stand": stand, "swing": swing} {
		var buf bytes.Buffer
		if err := png.Encode(&buf, stripOf(row, 3)); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, caseName(c.mod)+"-"+name+".png"), buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func framePicture(f *terrain.StaticFrame, mirror bool) *image.RGBA {
	src := f.RGBA()
	if !mirror {
		return src
	}
	b := src.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(b.Max.X-1-(x-b.Min.X), y, src.At(x, y))
		}
	}
	return out
}

// stripOf lays pictures side by side on a grey ground, scaled by n.
func stripOf(pics []*image.RGBA, n int) *image.RGBA {
	w, h := 1, 1
	for _, p := range pics {
		w += p.Bounds().Dx() + 4
		h = max(h, p.Bounds().Dy()+8)
	}
	strip := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(strip, strip.Bounds(), &image.Uniform{color.RGBA{0x70, 0x70, 0x70, 0xff}}, image.Point{}, draw.Src)
	x := 2
	for _, p := range pics {
		r := image.Rect(x, 4, x+p.Bounds().Dx(), 4+p.Bounds().Dy())
		draw.Draw(strip, r, p, p.Bounds().Min, draw.Over)
		x += p.Bounds().Dx() + 4
	}
	big := image.NewRGBA(image.Rect(0, 0, w*n, h*n))
	for y := 0; y < h*n; y++ {
		for x := 0; x < w*n; x++ {
			big.Set(x, y, strip.At(x/n, y/n))
		}
	}
	return big
}

// modBodyRun is the hashed outcome of one War Hammer session: the hero equips
// the hammer in mission 10, strikes the nearest other person and the mission
// runs on; then a mission SAVE is written at a fixed clock.
type modBodyRun struct {
	hash  uint64
	world []byte
	sav   []byte
	art   *terrain.UnitClass
}

// modBodySession plays that session with the hammer worn from the mission's
// start (worn) or equipped from the pack after it.
func modBodySession(t *testing.T, id string, worn bool) modBodyRun {
	t.Helper()
	f := modBodyFront(t, id, false)
	var mw *mapWorld
	var hero sim.EntityID
	if worn {
		mw, hero, _, _ = hammerHeroMission(t, f)
	} else {
		mw, hero = heroSwingMission(t, f, warHammerCode(t, f))
	}
	mw.refreshAppearance()
	var victim sim.EntityID
	found := false
	for _, e := range mw.world.Entities() {
		if e.ID != hero && e.Owner != sim.SelfSlot && e.Alive() && !e.OffMap {
			victim, found = e.ID, true
			break
		}
	}
	if !found {
		t.Fatal("the mission holds no other person to strike")
	}
	hv, _ := mw.entity(hero)
	if err := mw.world.HeadlessPlace(victim, hv.X+1, hv.Y); err != nil {
		t.Fatal(err)
	}
	mw.strike(uint32(hero), uint32(victim))
	for tick := 0; tick < 40; tick++ {
		mw.tick()
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	run := modBodyRun{hash: mw.world.Hash(), world: marshalWorld(t, mw.world), art: mw.art[hero]}
	name, err := save(true)
	if errors.Is(err, errSavingUnavailable) {
		// The second game writes no SAV yet; the world bytes stand alone.
		return run
	}
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE wrote %q: %v", name, err)
	}
	if run.sav, err = store.Read(name); err != nil {
		t.Fatal(err)
	}
	return run
}

// TestReleaseModBodyLeavesWorldAndSaveUnchanged plays the same War Hammer
// session with no mod and under each example mod's body data: the World hash,
// the world bytes and, where the game writes one, the mission SAV bytes agree,
// while the drawn body differs.
func TestReleaseModBodyLeavesWorldAndSaveUnchanged(t *testing.T) {
	for _, worn := range []bool{true, false} {
		t.Run(map[bool]string{true: "worn-at-start", false: "equipped"}[worn], func(t *testing.T) {
			modBodySessionsAgree(t, worn)
		})
	}
}

func modBodySessionsAgree(t *testing.T, worn bool) {
	t.Helper()
	plain := modBodySession(t, "", worn)
	again := modBodySession(t, "", worn)
	if plain.hash != again.hash || !bytes.Equal(plain.world, again.world) || !bytes.Equal(plain.sav, again.sav) {
		t.Fatal("two unmodded sessions disagree; the comparison below would prove nothing")
	}
	if base := BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))); strings.HasPrefix(base, "rom1") && len(plain.sav) == 0 {
		t.Fatalf("%s wrote no mission SAV", base)
	}
	t.Logf("mission SAV %d bytes, world %d bytes, hash %#x", len(plain.sav), len(plain.world), plain.hash)
	for _, id := range []string{"war-hammer-mace", "war-hammer-body"} {
		got := modBodySession(t, id, worn)
		if got.art == nil || got.art == plain.art || terrainBodyFramesEqual(got.art, plain.art) {
			t.Fatalf("%s: the hero is drawn as without the mod", id)
		}
		if got.hash != plain.hash {
			t.Errorf("%s: World hash %#x, want %#x", id, got.hash, plain.hash)
		}
		if !bytes.Equal(got.world, plain.world) {
			t.Errorf("%s: world bytes differ", id)
		}
		if !bytes.Equal(got.sav, plain.sav) {
			t.Errorf("%s: mission SAV bytes differ (%d against %d bytes)", id, len(got.sav), len(plain.sav))
		}
	}
}

func terrainBodyFramesEqual(a, b *terrain.UnitClass) bool {
	if a == nil || b == nil || len(a.Frames) != len(b.Frames) || a.Width != b.Width || a.Height != b.Height {
		return false
	}
	for i := range a.Frames {
		if a.Frames[i] != b.Frames[i] {
			return false
		}
	}
	return true
}

// TestReleaseModBodyRedrawsAfterALoad writes a mission SAV of the War Hammer
// hero under each example mod's body data and loads it cold under the same
// data: the save holds the shipped body name, and the reopened mission draws
// the mod's body again.
func TestReleaseModBodyRedrawsAfterALoad(t *testing.T) {
	for _, c := range modBodyCases[1:] {
		t.Run(caseName(c.mod), func(t *testing.T) {
			run := modBodySession(t, c.mod, true)
			if run.sav == nil {
				t.Skip("this game writes no SAV yet")
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "hammer.sav"), run.sav, 0o644); err != nil {
				t.Fatal(err)
			}
			g := modBodyFront(t, c.mod, false)
			_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
			open, town, err := load("hammer.sav")
			if err != nil || town {
				t.Fatalf("LOAD of the mission SAV: town=%v err=%v", town, err)
			}
			a := g.App("mod body loaded")
			t.Cleanup(a.StopAudio)
			a.SetCutscenes(nil)
			if err := a.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			checkModBodyArt(t, g.live, equipmentReturnHero(t, g), c.body)
		})
	}
}
