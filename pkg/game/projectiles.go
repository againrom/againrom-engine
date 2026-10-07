package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/pal"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// The projectile art loader: the registry a cast's picture is looked up in,
// and the sheet behind each row.
//
// It is LoadUnits' own shape one registry over — GPU-free and headless,
// nothing here builds a texture or opens a window — and it is deliberately
// NOT the engine's own shape. The original loads a projectile sheet lazily,
// on its first draw; this loads every one it can with the registry, so a
// -check run reaches them all with no graphics context and no cast stutters
// on its first frame.

// ProjectileRegistry is the ADDRESS the projectile records are loaded from —
// UnitRegistry's sibling and the same shape: the graphics container's identity
// segment and the entry inside it, folded and forward-separated.
const ProjectileRegistry = graphicsPrefix + data.ProjectilePrefix + "projectiles.reg"

// ProjectileArtEvidence is a no-pixels diagnostic of one installed projectile
// row and the decoded sheet behind it. The raw payload is reduced to a digest;
// developer tools can compare lawful roots without writing converted game data.
type ProjectileArtEvidence struct {
	Path             string
	PayloadSHA256    [32]byte
	Frames, Phases   int
	Width, Height    int
	CenterX, CenterY int
}

// InspectProjectileArt resolves one picture through the production registry,
// loader and archive path. It is read-only and opens no window.
func InspectProjectileArt(src terrain.EntrySource, picture int) (ProjectileArtEvidence, error) {
	stream, err := src.ReadFile(ProjectileRegistry)
	if err != nil {
		return ProjectileArtEvidence{}, err
	}
	r, err := reg.Parse(stream)
	if err != nil {
		return ProjectileArtEvidence{}, fmt.Errorf("%s: %w", ProjectileRegistry, err)
	}
	records, err := data.LoadProjectiles(r)
	if err != nil {
		return ProjectileArtEvidence{}, fmt.Errorf("%s: %w", ProjectileRegistry, err)
	}
	record, ok := records.ByID(int32(picture))
	if !ok || record.SpritePath() == "" {
		return ProjectileArtEvidence{}, fmt.Errorf("projectile picture %d names no readable sheet", picture)
	}
	path := strings.TrimSuffix(ProjectileRegistry, data.ProjectilePrefix+"projectiles.reg") + record.SpritePath()
	payload, err := src.ReadFile(path)
	if err != nil {
		return ProjectileArtEvidence{}, err
	}
	set, err := LoadProjectiles(src)
	if err != nil {
		return ProjectileArtEvidence{}, err
	}
	sheet := set.Sheet(picture)
	if sheet == nil {
		return ProjectileArtEvidence{}, fmt.Errorf("projectile picture %d sheet did not decode", picture)
	}
	out := ProjectileArtEvidence{Path: path, PayloadSHA256: sha256.Sum256(payload),
		Frames: len(sheet.Frames), Phases: sheet.Phases, CenterX: sheet.CenterX, CenterY: sheet.CenterY}
	if frame := sheet.Frame(0); frame != nil {
		out.Width, out.Height = frame.Width, frame.Height
	}
	return out, nil
}

// CastFigureEvidence is one drawn tick of one cast, measured through the
// production loader and the production draw (1004). It is a developer tool's
// answer, not a picture: counts and points, no pixels.
type CastFigureEvidence struct {
	Picture   int
	Life      int
	Age       int
	Stamps    int
	Trail     int
	Burst     int
	First     image.Point
	Last      image.Point
	MaxOffset int
}

// InspectCastFigure drives the real cast observation and the real draw over a
// cast of spell from cell (0,0) to cell (cells,0), on the art this install
// ships, and answers what the viewer would be handed on each of the object's
// ticks. It is read-only and opens no window.
//
// It exists because everything this story builds is decided on the far side of
// the seam: the figure, the trail and the burst's wait are all in the tier that
// hands the viewer a list, so a real-install witness of them is a walk of that
// list rather than a screenshot.
func InspectCastFigure(src terrain.EntrySource, spell, cells int) ([]CastFigureEvidence, error) {
	set, err := LoadProjectiles(src)
	if err != nil {
		return nil, err
	}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical,
		nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	mw := &mapWorld{world: w, projectiles: set, swing: map[sim.EntityID]int{},
		phase: map[sim.EntityID]sim.AttackPhase{}, castRun: map[sim.EntityID]castRun{}}
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: uint16(spell),
		FromX: 0, FromY: 0, ToX: int32(cells), ToY: 0}})

	picture := data.CastPicture(spell)
	sheet := set.Sheet(picture)
	burstSheet := set.Sheet(data.BurstPicture(spell))
	var out []CastFigureEvidence
	for age := 0; len(mw.bolts) > 0 && age < 64; age++ {
		row := CastFigureEvidence{Picture: picture, Age: age, Life: mw.bolts[0].life}
		for _, d := range mw.boltDraws(nil) {
			switch {
			case d.Sheet == set.SmokeSheet(0) || d.Sheet == set.SmokeSheet(1):
				row.Trail++
			case burstSheet != nil && d.Sheet == burstSheet:
				row.Burst++
			case d.Sheet == sheet:
				if row.Stamps == 0 {
					row.First = d.Pos
				}
				row.Last = d.Pos
				row.Stamps++
				if o := offsetFromLine(d.Pos); o > row.MaxOffset {
					row.MaxOffset = o
				}
			}
		}
		out = append(out, row)
		mw.advanceBolts()
	}
	return out, nil
}

// InspectWeaponRelease is InspectCastFigure's other producer. It drives the
// real weapon-borne draw over an attacker holding a weapon whose castSpell
// is spell, mid wind-up, with its victim cells away, on the art this install
// ships. One row per swing position within the attack charge.
//
// It exists because a cast picture has two producers and only one of them is a
// cast object, so a walk of the cast list is not a witness of the other. The two
// must agree on what a picture draws.
func InspectWeaponRelease(src terrain.EntrySource, spell, cells int) ([]CastFigureEvidence, error) {
	set, err := LoadProjectiles(src)
	if err != nil {
		return nil, err
	}
	const charge = 8
	ents := []sim.Entity{
		{ID: 1, X: 0, Y: 0, HP: 100, MaxHP: 100, Owner: 1, WeaponSpell: uint16(spell),
			HasAttackTarget: true, AttackTarget: 2, AttackPhase: sim.AttackCasting,
			AttackCharge: charge},
		{ID: 2, X: int32(cells), Y: 0, HP: 100, MaxHP: 100, Owner: 2},
	}
	mw := &mapWorld{projectiles: set, swing: map[sim.EntityID]int{},
		phase: map[sim.EntityID]sim.AttackPhase{}, castRun: map[sim.EntityID]castRun{}}

	picture := data.CastPicture(spell)
	sheet := set.Sheet(picture)
	var out []CastFigureEvidence
	for swing := 0; swing <= charge; swing++ {
		mw.swing[1] = swing
		row := CastFigureEvidence{Picture: picture, Age: swing, Life: charge}
		for _, d := range mw.weaponBoltDraws(ents) {
			switch {
			case d.Sheet == set.SmokeSheet(0) || d.Sheet == set.SmokeSheet(1):
				row.Trail++
			case d.Sheet == sheet:
				if row.Stamps == 0 {
					row.First = d.Pos
				}
				row.Last = d.Pos
				row.Stamps++
				if o := offsetFromLine(d.Pos); o > row.MaxOffset {
					row.MaxOffset = o
				}
			}
		}
		out = append(out, row)
	}
	return out, nil
}

// offsetFromLine is a point's distance from the cast's own straight line, in
// ShotScale units. The line here runs along x, so it is the y term alone.
func offsetFromLine(p image.Point) int {
	if p.Y < 0 {
		return -p.Y
	}
	return p.Y
}

// LoadProjectiles decodes the projectile records and their sheets' every
// frame out of the graphics container into the render tier's bundle.
//
// The bundle is keyed by PICTURE ID, which is the id a spell computes and the id
// the engine's own record array is indexed by, so a lookup downstream is a
// subscript rather than a search.
//
// ONLY AN UNREADABLE OR UNPARSEABLE REGISTRY IS AN ERROR, and the read error
// is returned unwrapped so a front end's -check can test it with errors.Is.
// Every other failure is a SKIP that leaves the picture without a sheet:
//
//   - an entry that is not in the archive;
//   - a stream the decoder refuses;
//   - a .16a sheet with no colour table of its own. Every shipped .16a row
//     carries one and sets the registry's Palette bit;
//   - a .256 sheet whose row names a table that is not there: its own when
//     the sheet carries none, or the shared projectiles.pal when the install
//     holds no readable one (indexedEffectFrames);
//   - a sheet holding no frames.
//
// A skipped picture draws nothing, which is a state the drawing side already has
// an answer for: 21 of the 28 spell ids compute a picture with no record at all.
func LoadProjectiles(src terrain.EntrySource) (*terrain.EffectSet, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no graphics archive", ProjectileRegistry)
	}
	stream, err := src.ReadFile(ProjectileRegistry)
	if err != nil {
		return nil, err
	}
	r, err := reg.Parse(stream)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ProjectileRegistry, err)
	}
	records, err := data.LoadProjectiles(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ProjectileRegistry, err)
	}

	set := &terrain.EffectSet{Sheets: make(map[int]*terrain.EffectSheet), Pictures: make(map[int]bool)}
	// The sheets are memoised on their ADDRESS and the table they are drawn
	// through, so two rows naming one file hold one slice of frames — pointer
	// for pointer — and the frame-keyed texture cache downstream uploads it
	// once. Two of the shipped rows do name one file.
	type sheetKey struct {
		path   string
		shared bool
	}
	decoded := make(map[sheetKey][]*terrain.EffectFrame)
	shared := sharedProjectileTable(src)
	for id := int32(0); id <= projectileMaxID; id++ {
		p, ok := records.ByID(id)
		if !ok {
			continue
		}
		set.Pictures[int(id)] = true
		path := p.SpritePath()
		if path == "" {
			continue
		}
		key := sheetKey{path: path, shared: p.A16 == 0 && p.Palette == 0}
		frames, tried := decoded[key]
		if !tried {
			if p.A16 != 0 {
				frames = effectFrames(src, graphicsPrefix+path)
			} else {
				frames = indexedEffectFrames(src, graphicsPrefix+path, key.shared, shared)
			}
			decoded[key] = frames
		}
		if len(frames) == 0 {
			continue
		}
		set.Sheets[int(id)] = &terrain.EffectSheet{
			Frames:         frames,
			Phases:         int(p.Phases),
			RotationPhases: int(p.RotationPhases),
			Flip:           p.Flip != 0,
			Clock:          effectClock(int(id)),
			// The registry's own Width and Height HALVED, which is what the draw
			// subtracts from the object's point. They are the centring offsets and
			// not the art's size, and on shipped rows the two disagree — carried
			// across as stated, never reconciled with the frame.
			CenterX: int(p.Width) / 2,
			CenterY: int(p.Height) / 2,
		}
	}
	loadSmokeSheets(src, set)
	return set, nil
}

// smokeEntry is the address of one trail sheet inside the graphics
// container. The two sheets are loaded by CONSTRUCTED PATH and not through
// the registry, which is what the engine's own registry loader does with
// them: it reads projectiles.reg's 31 rows and then, outside the loop, opens
// graphics\projectiles\smoke%d\sprites.16a for 0 and 1 into a two-slot
// global (`REG-PROJ-086`). They are the only projectile art no ID addresses.
const smokeEntry = graphicsPrefix + data.ProjectilePrefix + "smoke%d/sprites.16a"

// loadSmokeSheets fills the bundle's two trail sheets. A sheet that is absent or
// will not decode is left nil, on the loader's own rule above: nothing here is
// an error, and a nil trail sheet draws no trail.
//
// A trail sheet has NO REGISTRY ROW, so it states no Phases, no RotationPhases
// and no centring halves. The phase count is its own frame count, the rotation
// count is one — the trail is stamped without a facing — and the halves are the
// art's own, because the registry's Width and Height are what the draw would
// otherwise subtract and this sheet has none.
func loadSmokeSheets(src terrain.EntrySource, set *terrain.EffectSet) {
	for slot := range set.Smoke {
		frames := effectFrames(src, fmt.Sprintf(smokeEntry, slot))
		if len(frames) == 0 {
			continue
		}
		set.Smoke[slot] = &terrain.EffectSheet{
			Frames:         frames,
			Phases:         len(frames),
			RotationPhases: 1,
			Clock:          terrain.EffectClockDirect,
			CenterX:        frames[0].Width / 2,
			CenterY:        frames[0].Height / 2,
		}
	}
}

// loadProjectilesOrNil is the front end's own call: the bundle, or nil for an
// install whose projectile registry is absent or will not parse.
//
// IT SWALLOWS THE ERROR ON PURPOSE, unlike the object, unit and structure
// loaders beside it, whose failure stops a window from opening. Those three are
// the map's own furniture and a run without them is a run missing its ground. A
// projectile sheet is what a cast looks like, and the font's rule holds: a
// mission does not fail to open over a cosmetic asset.
func loadProjectilesOrNil(src terrain.EntrySource) *terrain.EffectSet {
	set, err := LoadProjectiles(src)
	if err != nil {
		return nil
	}
	return set
}

// projectileMaxID is the highest picture id this loader walks. It is the highest
// a SPELL can compute — 2*28 + 9 — and the walk is over ids rather than over the
// record map so the bundle is built in numeric order and does not depend on a
// map walk. A row above it exists on the shipped registry (id 62) and is
// reachable only through a unit class's own key, which is not a cast.
const projectileMaxID = 65

// effectClock is which phase clock a picture runs. The default is one frame
// per two ticks and it is what every picture but two takes; the two
// overrides are the engine's own per-picture switch arms.
//
// It sits here, in the tier that reads the registry, because the render tier has
// no picture ids: the id is resolved into a clock once, at load, and the
// selection downstream reads a field.
func effectClock(picture int) terrain.EffectClock {
	switch picture {
	case 60:
		return terrain.EffectClockDirect
	case 51:
		return terrain.EffectClockRaw
	}
	return terrain.EffectClockHalf
}

// effectFrames is one sheet's every frame, or nil for any of the ways it
// cannot be built.
//
// THE BLEND IS cursorPixel's, called and not copied. This package already
// resolves a pixel of this format to a premultiplied colour, for the attack
// pointer and the inventory icon, and a second resolution here that
// disagreed about what a level means would be invisible on screen: nobody
// looking at a rendered bolt can tell that coverage 7 came out one sixteenth
// too bright. One walk, three callers.
//
// An UNPAINTED cell is the zero value and is left alone, which is the decoder's
// own distinction: a painted level of 0 is a written pixel that is nearly
// transparent, and an unpainted cell is no pixel at all.
func effectFrames(src terrain.EntrySource, addr string) []*terrain.EffectFrame {
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil
	}
	sprite, err := spr16.DecodeA(b, true)
	if err != nil || sprite == nil || len(sprite.Palette) == 0 {
		return nil
	}
	out := make([]*terrain.EffectFrame, 0, len(sprite.Frames))
	for _, f := range sprite.Frames {
		frame := &terrain.EffectFrame{
			Width:  f.Width,
			Height: f.Height,
			Pixels: make([]color.RGBA, len(f.Pixels)),
		}
		for i, p := range f.Pixels {
			if !p.Painted {
				continue
			}
			frame.Pixels[i] = cursorPixel(sprite.Palette, p)
		}
		out = append(out, frame)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// projectileTableEntry is the shared colour table a .256 row with Palette 0
// is drawn through (PAL-PROJ-011). projectile_.pal beside it is loaded by the
// engine and never drawn with, so it is not read here.
const projectileTableEntry = graphicsPrefix + data.ProjectilePrefix + "projectiles.pal"

// sharedProjectileTable is projectiles.pal's 256 colours at full opacity, or
// nil for an install without a readable one.
func sharedProjectileTable(src terrain.EntrySource) *[256]color.RGBA {
	raw, err := src.ReadFile(projectileTableEntry)
	if err != nil {
		return nil
	}
	t, err := pal.Decode(raw)
	if err != nil {
		return nil
	}
	table := tableRGBA(t)
	return &table
}

// indexedEffectFrames is one .256 sheet's every frame, resolved through the
// table its row names (REG-PROJ-087, ANIM-CAST-027): the shared table when
// shared is set, whatever the sheet carries, and the sheet's own otherwise.
// It is nil when that table is missing or the stream will not decode.
//
// The format has no coverage, so an opaque pixel takes its entry at full
// opacity and a hole stays the zero value. The draw's light level, the own
// table's tint and the "b" sibling sheet are not applied: which shade level
// the engine passes is not published (DIV-1454).
func indexedEffectFrames(src terrain.EntrySource, addr string, shared bool, sharedTable *[256]color.RGBA) []*terrain.EffectFrame {
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil
	}
	sprite, err := spr256.Decode(b)
	if err != nil {
		return nil
	}
	var table [256]color.RGBA
	switch {
	case shared && sharedTable != nil:
		table = *sharedTable
	case !shared && sprite.HasPalette && len(sprite.Palette) == len(table):
		for i, e := range sprite.Palette {
			table[i] = color.RGBA{R: e.R, G: e.G, B: e.B, A: 0xff}
		}
	default:
		return nil
	}
	out := make([]*terrain.EffectFrame, 0, len(sprite.Frames))
	for _, f := range sprite.Frames {
		frame := &terrain.EffectFrame{Width: f.Width, Height: f.Height, Pixels: make([]color.RGBA, len(f.Pixels))}
		for i, p := range f.Pixels {
			if p.Opaque {
				frame.Pixels[i] = table[p.Index]
			}
		}
		out = append(out, frame)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
