package game

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/pal"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/refraction"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The owner's report: arrows, the bats' attack, sling stones and the Fire
// Arrow spell's projectile were drawn as an orange pixel. Each scenario opens
// an installed mission, gives the party ordinary orders and runs the ordinary
// driver tick. On the tick a ranged swing reaches its class's release tick —
// read here from the installed units.reg, not through the loader under test —
// the frame must carry that class's installed projectile sheet from the
// shooter's cell and no orange mark, and a .256 picture's drawn frame must
// equal, pixel for pixel, the installed frame this test decodes itself. The
// bat's row carries background deformation without a sprite or mark. A staff's
// Fire Arrow wind-up carries its spell's picture and no mark. Each scenario
// logs a digest of its per-tick world hashes, so a run on the base shows the
// simulation did not move, and AGAINROM_PROJECTILE_ARTIFACTS receives an
// offscreen raster of each kind in flight. An attack order keeps its victim
// whoever strikes the unit on the way (AI-RETAL-056), so a scenario that names
// an answer replies to being shot at as a player does: whenever the hero holds
// none of the enemies that hold him as their victim, he is ordered onto the
// nearest of them.
func TestReleaseRangedShotsDrawTheirInstalledSprites(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: the ranged-shot witness needs a lawful install")
	}
	for _, sc := range []shotScenario{
		// The hero's own bow, ordered onto the crossbowman nearest him and
		// answering the slingers and the orc archer who shoot him on the way.
		{name: "bow", mission: 41, party: shotPartyBow, order: shotAttackNearest(15), answers: true,
			ticks: 400, pictures: []int{1, 3, 4}},
		// The default mage's Fire Arrow staff, ordered onto the same target.
		{name: "staff", mission: 41, party: shotPartyMage, order: shotAttackNearest(15), answers: true,
			ticks: 360, casts: true},
		// A march that meets slingers, an orc archer and bats.
		{name: "march41", mission: 41, order: shotMoveTo(58, 66), ticks: 520,
			pictures: []int{3, 4, unitShotDeformationPictureID}},
		// A march that meets two human archers.
		{name: "march110", mission: 110, order: shotMoveTo(30, 50), ticks: 460,
			pictures: []int{1}},
	} {
		t.Run(sc.name, func(t *testing.T) { runShotScenario(t, sc) })
	}
}

// unitShotDeformationPictureID is the bat's row, stated here rather than read from
// the production constant, so the witness names the picture itself.
const unitShotDeformationPictureID = 7

type shotParty int

const (
	shotPartyDefault shotParty = iota
	shotPartyBow
	shotPartyMage
)

type shotScenario struct {
	name     string
	mission  int
	party    shotParty
	order    func(t *testing.T, mw *mapWorld, hero sim.EntityID)
	answers  bool
	ticks    int
	pictures []int
	casts    bool
}

func shotMoveTo(x, y int) func(*testing.T, *mapWorld, sim.EntityID) {
	return func(_ *testing.T, mw *mapWorld, hero sim.EntityID) { mw.enqueue(uint32(hero), x, y) }
}

// shotAttackNearest orders the hero onto the nearest living enemy of class.
func shotAttackNearest(class int32) func(*testing.T, *mapWorld, sim.EntityID) {
	return func(t *testing.T, mw *mapWorld, hero sim.EntityID) {
		h, _ := mw.entity(hero)
		best, bestDist := sim.EntityID(0), -1
		for _, e := range mw.world.EntityView() {
			if e.Class != class || !e.Alive() || e.OffMap {
				continue
			}
			if d := chebyshevDist(int(e.X-h.X), int(e.Y-h.Y)); bestDist < 0 || d < bestDist {
				best, bestDist = e.ID, d
			}
		}
		if bestDist < 0 {
			t.Fatalf("the mission places no class %d to attack", class)
		}
		mw.strike(uint32(hero), uint32(best))
	}
}

// shotAnswer orders the hero onto the nearest enemy that holds him as its
// victim, unless he already holds one of them as his.
func shotAnswer(mw *mapWorld, hero sim.EntityID) {
	h, _ := mw.entity(hero)
	best, bestDist, holds := sim.EntityID(0), -1, false
	for _, e := range mw.world.EntityView() {
		if !e.Alive() || e.OffMap || !e.HasAttackTarget || e.AttackTargetKind != sim.AttackTargetUnit || e.AttackTarget != hero {
			continue
		}
		holds = holds || (h.HasAttackTarget && h.AttackTarget == e.ID)
		if d := chebyshevDist(int(e.X-h.X), int(e.Y-h.Y)); bestDist < 0 || d < bestDist {
			best, bestDist = e.ID, d
		}
	}
	if bestDist >= 0 && !holds {
		mw.strike(uint32(hero), uint32(best))
	}
}

func runShotScenario(t *testing.T, sc shotScenario) {
	f := releaseFront(t)
	switch sc.party {
	case shotPartyBow:
		bow, err := resolveWeaponForSlot(false, f.Table.Shapes, f.Table.Materials, f.Table.Weapons, data.SkillShoot)
		if err != nil {
			t.Fatal(err)
		}
		f.Carried = MissionPartyAs(false, bow, f.Bodies, f.Table)
	case shotPartyMage:
		f.Carried = MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	}
	if err := f.App("ranged-shot-" + sc.name).OpenMission(f.MissionOpener(sc.mission)); err != nil {
		t.Fatal(err)
	}
	shots := installedClassShots(t, f)
	w := &shotWitness{t: t, mw: f.live, shots: shots, art: installedShotArt(t, f, shots),
		tiles: f.Tiles,
		was:   map[sim.EntityID]sim.AttackPhase{}, run: map[sim.EntityID]sim.AttackPhase{},
		checked: map[int]int{}, shown: map[string]bool{}, name: sc.name}
	if dir := os.Getenv("AGAINROM_PROJECTILE_ARTIFACTS"); dir != "" {
		w.dir = filepath.Join(dir, filepath.Base(f.Archives.Root))
		if err := os.MkdirAll(w.dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	hero := f.live.mission.ids[0]
	sc.order(t, f.live, hero)
	digest := fnv.New64a()
	for tick := 1; tick <= sc.ticks; tick++ {
		if sc.answers {
			shotAnswer(f.live, hero)
		}
		f.LiveAdvance(1)
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], f.live.world.Hash())
		digest.Write(b[:])
		w.observe(tick)
	}
	var kinds []string
	for p, n := range w.checked {
		kinds = append(kinds, fmt.Sprintf("picture %d x%d", p, n))
	}
	sort.Strings(kinds)
	t.Logf("%s: mission %d, %d ticks, world hash digest %016x, final world hash %016x; checked %v, casting wind-up ticks %d",
		sc.name, sc.mission, sc.ticks, digest.Sum64(), f.live.world.Hash(), kinds, w.casts)
	for _, p := range sc.pictures {
		if w.checked[p] == 0 {
			t.Errorf("%s: no release of picture %d was reached; the scenario does not witness it", sc.name, p)
		}
	}
	if sc.casts && w.casts == 0 {
		t.Errorf("%s: no Fire Arrow casting wind-up was reached", sc.name)
	}
}

// installedClassShots is every units.reg class's Projectile and ShootDelay,
// read from the install through the registry decoder.
func installedClassShots(t *testing.T, f *FrontEnd) map[int32][2]int {
	t.Helper()
	stream, err := f.Archives.Containers.ReadFile(UnitRegistry)
	if err != nil {
		t.Fatal(err)
	}
	r, err := reg.Parse(stream)
	if err != nil {
		t.Fatal(err)
	}
	classes, err := data.LoadUnitClasses(r)
	if err != nil {
		t.Fatal(err)
	}
	out := map[int32][2]int{}
	for _, c := range classes.All() {
		out[c.ID] = [2]int{int(c.Projectile), int(c.ShootDelay)}
	}
	return out
}

type shotWitness struct {
	t        *testing.T
	mw       *mapWorld
	name     string
	shots    map[int32][2]int
	art      map[int][]*terrain.EffectFrame
	was, run map[sim.EntityID]sim.AttackPhase
	checked  map[int]int
	casts    int
	dir      string
	shown    map[string]bool
	tiles    *terrain.Tileset
}

// observe reads one tick's frame. The run a swing belongs to is followed from
// the phases the world reports: a run opens on entry into a wind-up and ends
// when the swing clock is cleared.
func (w *shotWitness) observe(tick int) {
	mw := w.mw
	ents := mw.world.EntityView()
	for _, e := range ents {
		if windUp(e.AttackPhase) && w.was[e.ID] != e.AttackPhase {
			w.run[e.ID] = e.AttackPhase
		} else if !e.Alive() || e.Turning() || (!e.HasAttackTarget && !mw.casting(e.ID)) {
			delete(w.run, e.ID)
		}
		w.was[e.ID] = e.AttackPhase
	}
	var draws []ui.MapEntity
	var bolts []ui.SpellBolt
	read := func() {
		if draws == nil {
			before := mw.world.Hash()
			draws, bolts = mw.entityDraws(), mw.boltDraws(ents)
			if after := mw.world.Hash(); after != before {
				w.t.Fatalf("tick %d: reading the frame moved the world hash %016x -> %016x", tick, before, after)
			}
		}
	}
	for _, e := range ents {
		if !e.Alive() || !e.HasAttackTarget || e.AttackTargetKind != sim.AttackTargetUnit {
			continue
		}
		victim, ok := entityIn(ents, e.AttackTarget)
		if !ok {
			continue
		}
		from, to := image.Pt(int(e.X), int(e.Y)), image.Pt(int(victim.X), int(victim.Y))
		switch {
		case w.run[e.ID] == sim.AttackCharging && e.Reach > 1:
			shot := w.shots[mw.spellClientClass(e.ID, e.Class)]
			if shot[0] == 0 || mw.swing[e.ID] != shot[1] || !sim.InReach(e, victim) {
				continue
			}
			read()
			w.release(tick, e, shot[0], from, to, draws, bolts)
		case e.AttackPhase == sim.AttackCasting && e.WeaponSpell == 1 && chebyshevDist(to.X-from.X, to.Y-from.Y) >= 2:
			read()
			w.cast(tick, e, from, to, draws, bolts)
		}
	}
}

// release checks one ranged swing on its release tick.
func (w *shotWitness) release(tick int, e sim.Entity, picture int, from, to image.Point, draws []ui.MapEntity, bolts []ui.SpellBolt) {
	w.checked[picture]++
	sheet := w.mw.projectiles.Sheet(picture)
	// The record the release tick just built is the newest id: its point is
	// where the frame draws the sprite.
	var at image.Point
	released := w.mw.world.SavedProjectiles()
	if p, ok := savedProjectileByID(w.mw.world, released.FreeIndex-1); ok && int(p.Picture) == picture {
		at = image.Pt(int(p.X), int(p.Y))
	} else {
		w.t.Errorf("%s tick %d: entity %d released picture %d and no record %d of that picture is in the World", w.name, tick, e.ID, picture, released.FreeIndex-1)
	}
	var own []ui.SpellBolt
	for _, b := range bolts {
		if b.Pos == at && b.Sheet != nil && b.Sheet == sheet {
			own = append(own, b)
		}
	}
	mark := shotMarkOf(draws, e.ID)
	w.t.Logf("%s tick %d: entity %d, class %d, releases picture %d from %v at %v", w.name, tick, e.ID, e.Class, picture, from, to)
	w.raster(fmt.Sprintf("picture%d", picture), tick, from, to, draws, bolts)
	if mark != nil {
		w.t.Errorf("%s tick %d: entity %d (picture %d) at %v draws the orange mark at %v", w.name, tick, e.ID, picture, from, *mark)
	}
	if picture == unitShotDeformationPictureID {
		if len(own) != 0 {
			w.t.Errorf("%s tick %d: the bat's row drew %d sprite(s); its draw arm blits nothing", w.name, tick, len(own))
		}
		effects := 0
		for _, b := range bolts {
			if b.Pos == at && b.Effect == ui.SpellBackgroundDeformation && b.Sheet == nil {
				effects++
			}
		}
		if effects == 0 {
			w.t.Errorf("%s tick %d: the bat released no travelling background deformation", w.name, tick)
		}
		return
	}
	if sheet == nil || len(own) != 1 {
		w.t.Errorf("%s tick %d: entity %d at %v released picture %d and the frame carries %d of its installed sprites (sheet loaded: %t)",
			w.name, tick, e.ID, from, picture, len(own), sheet != nil)
		return
	}
	drawn := sheet.Frame(own[0].Frame)
	if paintedPixels(drawn) == 0 {
		w.t.Errorf("%s tick %d: picture %d frame %d paints nothing", w.name, tick, picture, own[0].Frame)
	}
	art, ok := w.art[picture]
	if !ok {
		return
	}
	if own[0].Frame < 0 || own[0].Frame >= len(art) {
		w.t.Errorf("%s tick %d: picture %d draws frame %d; the installed sheet holds %d", w.name, tick, picture, own[0].Frame, len(art))
		return
	}
	if want := art[own[0].Frame]; !sameEffectFrame(drawn, want) {
		w.t.Errorf("%s tick %d: picture %d frame %d (%dx%d, %d painted) is not the installed frame (%dx%d, %d painted)",
			w.name, tick, picture, own[0].Frame, drawn.Width, drawn.Height, paintedPixels(drawn),
			want.Width, want.Height, paintedPixels(want))
	}
}

// installedShotArt is, for each unit-shot picture drawn from a .256 sheet, the
// frames the install holds for it: the sheet its projectiles.reg row names,
// resolved through the table its Palette bit selects — the sheet's own, or
// projectiles.pal (REG-PROJ-087, PAL-PROJ-011). It is decoded here from the
// registry and the format decoders, not through the sheet loader under test.
func installedShotArt(t *testing.T, f *FrontEnd, shots map[int32][2]int) map[int][]*terrain.EffectFrame {
	t.Helper()
	read := func(name string) []byte {
		t.Helper()
		b, err := f.Archives.Containers.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	r, err := reg.Parse(read("graphics/projectiles/projectiles.reg"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.LoadProjectiles(r)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := pal.Decode(read("graphics/projectiles/projectiles.pal"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[int][]*terrain.EffectFrame{}
	for _, s := range shots {
		row, ok := rows.ByID(int32(s[0]))
		if s[0] == 0 || !ok || row.A16 != 0 || out[s[0]] != nil {
			continue
		}
		sprite, err := spr256.Decode(read("graphics/projectiles/" + strings.ReplaceAll(row.File, `\`, "/") + ".256"))
		if err != nil {
			t.Fatalf("picture %d: %v", s[0], err)
		}
		table := shared
		if row.Palette != 0 {
			if !sprite.HasPalette {
				t.Fatalf("picture %d: the row names the sheet's own table and the sheet carries none", s[0])
			}
			for i, c := range sprite.Palette {
				table[i] = pal.Color{R: c.R, G: c.G, B: c.B}
			}
		}
		var frames []*terrain.EffectFrame
		for _, fr := range sprite.Frames {
			ef := &terrain.EffectFrame{Width: fr.Width, Height: fr.Height, Pixels: make([]color.RGBA, len(fr.Pixels))}
			for i, px := range fr.Pixels {
				if px.Opaque {
					c := table[px.Index]
					ef.Pixels[i] = color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff}
				}
			}
			frames = append(frames, ef)
		}
		out[s[0]] = frames
	}
	return out
}

func sameEffectFrame(a, b *terrain.EffectFrame) bool {
	if a == nil || b == nil || a.Width != b.Width || a.Height != b.Height || len(a.Pixels) != len(b.Pixels) {
		return false
	}
	for i := range a.Pixels {
		if a.Pixels[i] != b.Pixels[i] {
			return false
		}
	}
	return true
}

// cast checks one tick of a staff's Fire Arrow wind-up.
func (w *shotWitness) cast(tick int, e sim.Entity, from, to image.Point, draws []ui.MapEntity, bolts []ui.SpellBolt) {
	w.casts++
	sheet := w.mw.projectiles.Sheet(data.CastPicture(1))
	drawn := 0
	for _, b := range bolts {
		if b.Cell == from && sheet != nil && b.Sheet == sheet {
			drawn++
		}
	}
	w.raster("firearrow", tick, from, to, draws, bolts)
	if mark := shotMarkOf(draws, e.ID); mark != nil {
		w.t.Errorf("%s tick %d: entity %d's Fire Arrow wind-up draws the orange mark at %v", w.name, tick, e.ID, *mark)
	}
	if drawn != 1 {
		w.t.Errorf("%s tick %d: entity %d's Fire Arrow wind-up carries %d firebolt sprite(s), want 1", w.name, tick, e.ID, drawn)
	}
}

func shotMarkOf(draws []ui.MapEntity, id sim.EntityID) *image.Point {
	for _, d := range draws {
		if d.ID == uint32(id) {
			return d.Shot
		}
	}
	return nil
}

func paintedPixels(f *terrain.EffectFrame) int {
	if f == nil {
		return 0
	}
	n := 0
	for _, p := range f.Pixels {
		if p.A != 0 {
			n++
		}
	}
	return n
}

// raster writes, once per kind and scenario, an offscreen picture of the cells
// around a shot: every drawn unit frame, every spell object and every orange
// mark, placed by the viewer's own ground points, at twice the world scale.
// It is evidence for the owner and asserts nothing.
func (w *shotWitness) raster(kind string, tick int, from, to image.Point, draws []ui.MapEntity, bolts []ui.SpellBolt) {
	if w.dir == "" || w.shown[kind] {
		return
	}
	w.shown[kind] = true
	lo := image.Pt(min(from.X, to.X)-2, min(from.Y, to.Y)-3)
	hi := image.Pt(max(from.X, to.X)+3, max(from.Y, to.Y)+2)
	origin := lo.Mul(terrain.CellSize)
	canvas := image.NewRGBA(image.Rectangle{Max: hi.Sub(lo).Mul(terrain.CellSize)})
	for i := 0; i < len(canvas.Pix); i += 4 {
		copy(canvas.Pix[i:], []byte{0x30, 0x3c, 0x2c, 0xff})
	}
	if w.tiles != nil && w.mw.mission != nil && w.mw.mission.state != nil && w.mw.mission.state.Map != nil {
		m := w.mw.mission.state.Map
		words := terrain.RenderTileWords(m.Tiles)
		for y := max(0, lo.Y); y < min(m.Height, hi.Y); y++ {
			for x := max(0, lo.X); x < min(m.Width, hi.X); x++ {
				ref := terrain.Resolve(words[y*m.Width+x])
				if tile := w.tiles.Slot(ref.Slot).SubCell(ref.Sub); tile != nil {
					at := image.Pt(x*terrain.CellSize, y*terrain.CellSize).Sub(origin)
					draw.Draw(canvas, image.Rectangle{Min: at, Max: at.Add(image.Pt(32, 32))}, tile, tile.Rect.Min, draw.Src)
				}
			}
		}
	}
	sorted := append([]ui.MapEntity(nil), draws...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Cell.Y < sorted[j].Cell.Y })
	for _, d := range sorted {
		if d.Art == nil || d.Frame == nil {
			continue
		}
		if p, ok := terrain.UnitPlace(d.Cell.X, d.Cell.Y, d.Art, d.Frame, d.Mirror, 0, 0); ok {
			overPic(canvas, d.Frame.RGBA(), p.TopLeft.Sub(origin), d.Mirror)
		}
	}
	for _, b := range bolts {
		if b.Effect == ui.SpellBackgroundDeformation {
			center := image.Pt(b.Pos.X*terrain.CellSize/ui.ShotScale+terrain.CellSize/2, b.Pos.Y*terrain.CellSize/ui.ShotScale+terrain.CellSize/2).Sub(origin)
			mapping := b.Mapping
			if mapping == nil {
				mapping = refraction.Picture7Map()
			}
			refraction.Apply(canvas, mapping, center, canvas.Rect)
			continue
		}
		if f := b.Sheet.Frame(b.Frame); f != nil {
			px, py := ui.EffectGroundPoint(b.Pos, b.Sheet)
			overPic(canvas, f.RGBA(), image.Pt(px, py).Sub(origin), b.Mirror)
		}
	}
	for _, d := range draws {
		if d.Shot == nil {
			continue
		}
		c := image.Pt(d.Shot.X*terrain.CellSize/ui.ShotScale+terrain.CellSize/2,
			d.Shot.Y*terrain.CellSize/ui.ShotScale+terrain.CellSize/2).Sub(origin)
		for y := -ui.ShotMarkSize / 2; y < ui.ShotMarkSize/2; y++ {
			for x := -ui.ShotMarkSize / 2; x < ui.ShotMarkSize/2; x++ {
				canvas.SetRGBA(c.X+x, c.Y+y, ui.ShotMarkerColor)
			}
		}
	}
	scaled := image.NewRGBA(image.Rectangle{Max: canvas.Rect.Max.Mul(2)})
	for y := 0; y < scaled.Rect.Max.Y; y++ {
		for x := 0; x < scaled.Rect.Max.X; x++ {
			scaled.SetRGBA(x, y, canvas.RGBAAt(x/2, y/2))
		}
	}
	path := filepath.Join(w.dir, fmt.Sprintf("%s-%s-tick%d.png", w.name, kind, tick))
	out, err := os.Create(path)
	if err != nil {
		w.t.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, scaled); err != nil {
		w.t.Fatal(err)
	}
}

// overPic composites a premultiplied picture onto dst at at, mirrored left to
// right when mirror is set.
func overPic(dst, src *image.RGBA, at image.Point, mirror bool) {
	b := src.Rect
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			sx := x
			if mirror {
				sx = b.Dx() - 1 - x
			}
			s := src.RGBAAt(sx, y)
			if s.A == 0 {
				continue
			}
			p := image.Pt(at.X+x, at.Y+y)
			if !p.In(dst.Rect) {
				continue
			}
			d := dst.RGBAAt(p.X, p.Y)
			k := 255 - uint32(s.A)
			dst.SetRGBA(p.X, p.Y, color.RGBA{
				R: uint8(uint32(s.R) + uint32(d.R)*k/255),
				G: uint8(uint32(s.G) + uint32(d.G)*k/255),
				B: uint8(uint32(s.B) + uint32(d.B)*k/255),
				A: 0xff,
			})
		}
	}
}
