package game

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/vfs"
)

// A projectile restored from a SAV draws the facing its own saved `dir`
// names. The original's draw folds the field as (dir - 8) & 0xf, mirrors
// facings 9 to 15 onto 7 to 1 on a sheet carrying the halving bit, and
// indexes Phases * facing + phase (ANIM-PROJ-026). Two sheets stand for the
// two families a SAV can hold in flight: a spell's travelling picture with
// four phases and the halving bit, and a unit shot's one-phase sheet with
// sixteen stored facings. Every fixture is synthetic; nothing reads an
// install.

const (
	restoredBoltPicture  = 10
	restoredArrowPicture = 1
	restoredBoltPhases   = 4
	restoredFlight       = 6 // segments each restored projectile has left
)

// restoredSheetFacing is the sheet facing ANIM-PROJ-026's fold gives each
// saved dir, spelled out rather than computed: the field's 0 is the sheet's
// north (8) and its 8 the sheet's south (0).
var restoredSheetFacing = [16]int{8, 9, 10, 11, 12, 13, 14, 15, 0, 1, 2, 3, 4, 5, 6, 7}

// restoredFacingFront is the one-actor pool fixture with a graphics archive
// holding the two sheets and their registry: the SAV import reads the
// registry for each picture's Phases, the draw reads the sheets.
func restoredFacingFront(t *testing.T) *FrontEnd {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, b []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	i := func(name string, v int32) synth.RegNode { return synth.RegNode{Name: name, Kind: 0x02, Int: v} }
	s := func(name, v string) synth.RegNode { return synth.RegNode{Name: name, Kind: 0x00, Str: v} }
	frames := func(n int) []synth.Frame256 {
		out := make([]synth.Frame256, n)
		for k := range out {
			out[k] = synth.Frame256{Width: 1, Height: 1, Pixels: []synth.Pixel256{{Index: 1, Opaque: true}}}
		}
		return out
	}
	shared := make(color.Palette, 256)
	for k := range shared {
		shared[k] = color.RGBA{A: 0xff}
	}
	scenario := write(ScenarioArchive, synth.Archive([]synth.File{
		{Path: "10.alm", Data: poolFixtureMap(91)}, {Path: "npc.reg", Data: synth.NPCReg(nil)}}))
	graphics := write(GraphicsArchive, synth.Archive([]synth.File{
		{Path: "projectiles/projectiles.reg", Data: synth.ProjectilesReg(3,
			[]synth.RegNode{i("ID", restoredBoltPicture), s("File", `bolt\sprites`),
				i("Phases", restoredBoltPhases), i("Flip", 1), i("Palette", 1)},
			[]synth.RegNode{i("ID", restoredArrowPicture), s("File", `archer\arrow`), i("Phases", 1)},
			[]synth.RegNode{i("ID", 7), s("File", `catap2\sprites`), i("Phases", 1)})},
		// A halved sheet stores nine facings; an unhalved one sixteen.
		{Path: "projectiles/bolt/sprites.256", Data: synth.Sheet256(synth.Sheet256Options{
			Palette: []color.RGBA{{}, {R: 0xff}}, Frames: frames(restoredBoltPhases * 9)})},
		{Path: "projectiles/archer/arrow.256", Data: synth.Sheet256(synth.Sheet256Options{
			NoPalette: true, Frames: frames(16)})},
		{Path: "projectiles/projectiles.pal", Data: synth.BMP8(image.NewPaletted(image.Rect(0, 0, 1, 1), shared))},
	}))
	containers, err := vfs.Open([]string{scenario, graphics}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := missionFrontEnd(t)
	f.Archives.Containers = containers
	if f.Projectiles, err = LoadProjectiles(containers); err != nil {
		t.Fatal(err)
	}
	f.Table, f.Campaign = actorRegistryTable(), resolved(saveCampaign(), nil)
	f.SetDeterministicFrames(true)
	return f
}

// restoredFacingSource is a complete one-actor document with a bolt and an
// arrow in flight at each of the sixteen saved directions, each flying east
// on its own column with no target. Projectile 2*dir+1 is the bolt and
// 2*dir+2 the arrow. The actor stands on the map's unit 91.
func restoredFacingSource(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	actor := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 23, maxHP: 30}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{actor}}}, {}}, nil)
	source, err := sav.Open(completeCurrentDeadDocument(t, f, savedContainer(body)))
	if err != nil {
		t.Fatal(err)
	}
	store := sav.ProjectileStore{FreeIndex: 33}
	for dir := int32(0); dir < 16; dir++ {
		for k, picture := range []int32{restoredBoltPicture, restoredArrowPicture} {
			id := uint16(1 + 2*dir + int32(k))
			x, y := (4+dir)*256+128, (10+10*int32(k))*256+128
			store.IDs = append(store.IDs, id)
			store.Items = append(store.Items, sav.Projectile{ID: id, X: x, Y: y, Picture: picture,
				Dir: dir, LastAction: 1, Action: 1, ActionDir: dir, ActionX: x + 256*restoredFlight, ActionY: y,
				ActionSegments: restoredFlight})
		}
	}
	if err := source.SetProjectiles(store); err != nil {
		t.Fatal(err)
	}
	return source.Marshal()
}

// restoredFacingColdLoad LOADs a SAV the engine wrote into a fresh front
// through the App's own list.
func restoredFacingColdLoad(t *testing.T, raw []byte, name string) *FrontEnd {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, name), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := restoredFacingFront(t)
	app := f.App("cold SAV LOAD")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, name)
	return f
}

// restoredFacingWant is ANIM-PROJ-026's frame and mirror for one fixture
// picture at a saved dir and phase.
func restoredFacingWant(picture, dir, phase int32) (int, bool) {
	facing := restoredSheetFacing[dir&0xf]
	if picture == restoredArrowPicture {
		return facing + int(phase), false
	}
	if facing > 8 {
		return restoredBoltPhases*(16-facing) + int(phase), true
	}
	return restoredBoltPhases*facing + int(phase), false
}

// checkRestoredFacings requires every restored projectile to keep its saved
// dir and draw the frame ANIM-PROJ-026 names for it at its current phase, and
// returns the draws for comparison across a SAVE.
func checkRestoredFacings(t *testing.T, f *FrontEnd, when string) []string {
	t.Helper()
	items := f.live.world.SavedProjectiles().Items
	draws := f.live.savedProjectileDraws()
	if len(draws) != len(items) {
		t.Fatalf("%s: %d projectiles draw %d sheets", when, len(items), len(draws))
	}
	var out []string
	for k, p := range items {
		d := draws[k]
		if saved := int32(p.ID-1) / 2; p.Dir != saved || p.ActionDir != saved {
			t.Fatalf("%s: projectile %d carries dir %d/%d, not its saved %d", when, p.ID, p.Dir, p.ActionDir, saved)
		}
		frame, mirror := restoredFacingWant(p.Picture, p.Dir, p.Phase)
		if d.Sheet != f.Projectiles.Sheet(int(p.Picture)) || d.Pos != image.Pt(int(p.X), int(p.Y)) ||
			d.Frame != frame || d.Mirror != mirror {
			t.Errorf("%s: picture %d at dir %d phase %d draws frame %d mirror %t at %v, want frame %d mirror %t at (%d,%d)",
				when, p.Picture, p.Dir, p.Phase, d.Frame, d.Mirror, d.Pos, frame, mirror, p.X, p.Y)
		}
		out = append(out, fmt.Sprintf("%v/%d/%t", d.Pos, d.Frame, d.Mirror))
	}
	return out
}

// TestRestoredProjectilesDrawTheFacingTheirSavedDirNames: a bolt and an arrow
// at every saved dir, LOADed cold from a SAV, draw ANIM-PROJ-026's frame; the
// facing holds while they fly; an ordinary menu SAVE mid-flight and a cold
// LOAD of what it wrote draw the same frames, tick for tick, until they land.
// The same SAV with one written dir turned by 8 draws that projectile turned.
func TestRestoredProjectilesDrawTheFacingTheirSavedDirNames(t *testing.T) {
	f := restoredFacingFront(t)
	app, path := openOriginalSAVApp(t, f, restoredFacingSource(t, f), "facing.sav")
	if d := f.live.world.SavedWorldEffectDrivers(); d == nil || len(d.Projectiles) != 32 {
		t.Fatal("the restored projectiles were not armed", d)
	}
	checkRestoredFacings(t, f, "cold LOAD")
	f.live.tick()
	checkRestoredFacings(t, f, "first tick")
	f.live.tick()
	before := checkRestoredFacings(t, f, "second tick")
	_, name, written := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(path)})
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	fresh := restoredFacingColdLoad(t, written, name)
	after := checkRestoredFacings(t, fresh, "LOAD of the engine SAVE")
	for k := range before {
		if k >= len(after) || before[k] != after[k] {
			t.Fatalf("draw %d before the SAVE %s, after its LOAD %v", k, before[k], after)
		}
	}
	// Loss control: projectile 11, the bolt at dir 5, written at dir 13.
	turned := restoredFacingColdLoad(t, alterSAV(t, written, func(doc *sav.DocumentData) bool {
		for i := range doc.State.ValueRecords {
			if r := &doc.State.ValueRecords[i]; r.Path == "/Prj11/dir" {
				r.Value.Int32 += 8
				return true
			}
		}
		return false
	}), name)
	p, d := turned.live.world.SavedProjectiles().Items[10], turned.live.savedProjectileDraws()[10]
	if frame, mirror := restoredFacingWant(p.Picture, 13, p.Phase); p.ID != 11 || p.Dir != 13 ||
		d.Frame != frame || d.Mirror != mirror || fmt.Sprintf("%v/%d/%t", d.Pos, d.Frame, d.Mirror) == before[10] {
		t.Fatalf("a SAV turning projectile 11 by 8 draws %v/%d/%t, before the turn %s", d.Pos, d.Frame, d.Mirror, before[10])
	}
	for tick := 3; tick <= restoredFlight+1; tick++ {
		f.live.tick()
		fresh.live.tick()
		a, b := checkRestoredFacings(t, f, "continued"), checkRestoredFacings(t, fresh, "continued after LOAD")
		if fmt.Sprint(a) != fmt.Sprint(b) || f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatalf("tick %d: the LOAD of the engine SAVE continues differently", tick)
		}
	}
	if len(f.live.savedProjectileDraws()) != 0 || len(fresh.live.savedProjectileDraws()) != 0 {
		t.Fatal("a landed projectile is still drawn")
	}
}
