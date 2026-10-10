package game

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"sort"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/mod"
	"againrom/pkg/render/terrain"
)

// SetModBodies applies what the mods choose about hero bodies: the body a
// weapon is drawn with, and the body sheets mods supply. It runs once, after
// SetMods and before any game is opened. A refusal names the mod, the file and
// the line.
//
// The choice reaches every reader of the body list through this front end's
// own filesystem (ModBodyChoiceAddress), and every supplied sheet is built
// into this front end's unit bundle under both body directories, so a hero
// drawn with it never reads the archive for it. The class key stays the
// shipped body's (data.HeroAppearance): the mods change pictures only.
func (f *FrontEnd) SetModBodies(bodies mod.BodyData) error {
	if bodies.Empty() {
		return nil
	}
	if f.Table == nil || f.Table.Weapons == nil || f.Archives == nil || f.Archives.Containers == nil || f.Units == nil {
		return errors.New("the install has no weapon table, archives or unit bundle for the mods' bodies")
	}
	src := f.Archives.Containers
	list, built, err := resolveModBodies(src, f.Units, f.Table.Weapons, f.Bodies, bodies)
	if err != nil {
		return err
	}

	fs := src.Fork()
	own := *f.Archives
	own.Containers = fs
	f.Archives = &own
	choice := encodeBodyChoice(list)
	fs.Overlay(ModBodyChoiceAddress, func() ([]byte, error) { return append([]byte(nil), choice...), nil })
	f.Bodies = list
	f.Table.Mods.Bodies = list

	units := cloneCandidateUnits(f.Units)
	if units.Bodies == nil {
		units.Bodies = make(map[string]*terrain.UnitClass, len(built))
	}
	for key, class := range built {
		units.Bodies[key] = class
	}
	f.Units = units
	return nil
}

// resolveModBodies checks the mods' bodies against the install and answers the
// body list carrying their choices and every supplied body built under both
// body directories, keyed as the unit bundle keys bodies.
func resolveModBodies(src entrySource, units *terrain.UnitSet, weapons data.Collection, list data.BodyList,
	bodies mod.BodyData) (data.BodyList, map[string]*terrain.UnitClass, error) {
	if list.Len() == 0 {
		return list, nil, errors.New("the install has no body list for the mods' bodies")
	}
	built := map[string]*terrain.UnitClass{}
	for _, b := range bodies.Bodies {
		name := data.HeroBody(b.Name)
		if shippedBody(src, name) {
			return list, nil, itemFileError(b.Mod, b.File, b.NameLine, "body %q is a body the install ships; a supplied body takes a name of its own", b.Name)
		}
		for _, dir := range []string{data.HeroDirHeroes, data.HeroDirHeroesLight} {
			class, err := modBodyClass(src, units, b, dir)
			if err != nil {
				return list, nil, err
			}
			built[data.HeroBodyKey(dir, name)] = class
		}
		list = list.WithModBody(name, data.ModBody{WeaponLast: b.WeaponLast})
	}
	chosen := map[int]mod.WeaponBody{}
	for _, w := range bodies.Weapons {
		row, err := modWeaponRow(weapons, w)
		if err != nil {
			return list, nil, err
		}
		if first, dup := chosen[row]; dup {
			return list, nil, itemFileError(w.Mod, w.File, w.Line, "weapon row %d already has a body from mod %q (%s:%d)", row, first.Mod, first.File, first.Line)
		}
		if list.Entry(row-1) == "" {
			return list, nil, itemFileError(w.Mod, w.File, w.Line, "weapon row %d has no body in the install's body list; a mapping changes the body of a weapon the game draws", row)
		}
		body := data.HeroBody(w.Body)
		if strings.HasSuffix(w.Body, string(data.HeroShieldSuffix)) {
			return list, nil, itemFileError(w.Mod, w.File, w.BodyLine, "body %q ends in the shield suffix; the shield slot chooses the shield form", w.Body)
		}
		if _, modded := list.ModBodyOf(body); !modded && !shippedBodyInBoth(src, body) {
			return list, nil, itemFileError(w.Mod, w.File, w.BodyLine, "body %q is not a body the install ships under both %s and %s, nor one a loaded mod supplies", w.Body, data.HeroDirHeroes, data.HeroDirHeroesLight)
		}
		chosen[row] = w
		list = list.WithWeaponBody(row, body)
	}

	return list, built, nil
}

// shippedBody reports whether name is a body the original's name chain knows
// or the install carries a sheet for under either body directory.
func shippedBody(src entrySource, name data.HeroBody) bool {
	if _, known := data.HeroBodyClass(name); known {
		return true
	}
	for _, dir := range []string{data.HeroDirHeroes, data.HeroDirHeroesLight} {
		if _, err := src.ReadFile(graphicsPrefix + data.HeroSheetPath(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// shippedBodyInBoth reports whether the install carries a sheet for name under
// both body directories, which a fighter's body needs.
func shippedBodyInBoth(src entrySource, name data.HeroBody) bool {
	for _, dir := range []string{data.HeroDirHeroes, data.HeroDirHeroesLight} {
		if _, err := src.ReadFile(graphicsPrefix + data.HeroSheetPath(dir, name)); err != nil {
			return false
		}
	}
	return true
}

// modWeaponRow resolves a mapping's weapon to its definition row: the row
// whose name the mapping gives, the row it gives, and both agreeing when it
// gives both.
func modWeaponRow(weapons data.Collection, w mod.WeaponBody) (int, error) {
	row := 0
	if w.Weapon != "" {
		want := strings.Join(strings.Fields(w.Weapon), " ")
		for i := 1; i < weapons.Len(); i++ {
			if weapons.EntryName(i) == want {
				row = i
				break
			}
		}
		if row == 0 {
			return 0, itemFileError(w.Mod, w.File, w.WeaponLine, "the weapon table has no row named %q", w.Weapon)
		}
	}
	if w.Row != 0 {
		if w.Row >= weapons.Len() || weapons.EntryName(w.Row) == "" {
			return 0, itemFileError(w.Mod, w.File, w.RowLine, "the weapon table has no row %d", w.Row)
		}
		if row != 0 && row != w.Row {
			return 0, itemFileError(w.Mod, w.File, w.RowLine, "row %d is %q, not %q (row %d)", w.Row, weapons.EntryName(w.Row), w.Weapon, row)
		}
		row = w.Row
	}
	return row, nil
}

// modBodyClass builds one supplied body under one directory: its sheet cut
// into frames in the order the shipped block arithmetic reads them
// (data.UnitClass.Anim), on the geometry the mod states, through the same
// composition a shipped body takes (composeHeroBody). Everything the mod does
// not state comes from the class a body name the chain does not know is drawn
// as (data.HeroUnmatchedClass); a fallen hero takes the shipped dying body of
// the same directory.
func modBodyClass(src entrySource, units *terrain.UnitSet, b mod.BodySheet, dir string) (*terrain.UnitClass, error) {
	path, line := b.Heroes, b.HeroesLine
	if dir == data.HeroDirHeroesLight {
		path, line = b.HeroesLight, b.HeroesLightLine
	}
	bad := func(format string, args ...any) error {
		return itemFileError(b.Mod, b.File, line, "%s %q: %s", dir, path, fmt.Sprintf(format, args...))
	}
	base := units.Classes[data.HeroUnmatchedClass]
	if base == nil {
		return nil, bad("the install's unit registry has no class %d to take the body's other fields from", data.HeroUnmatchedClass)
	}
	raw, _, err := mod.ReadInside(b.Dir, path)
	if err != nil {
		return nil, bad("%v", err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, bad("not a readable PNG: %v", err)
	}
	frames, err := cutBodySheet(img, b)
	if err != nil {
		return nil, bad("%v", err)
	}
	rec := *base
	rec.Width, rec.Height = b.Width, b.Height
	rec.CenterX, rec.CenterY = b.OriginX, b.OriginY
	rec.Anim = unitAnim(modBodyAnim(b))
	rec.OwnerShaded = false
	if b.HasSelection {
		s := b.Selection
		rec.Selection = image.Rect(s[0], s[1], s[2], s[3])
	}
	sheets := sheetCache{src: src, decoded: map[string]*spr256.Sprite{}, converted: map[string][]*terrain.StaticFrame{}}
	return composeHeroBody(&rec, frames, nil, heroDyingBody(&sheets, units, dir, data.HeroBody(b.Name))), nil
}

// modBodyAnim is a supplied body's descriptor, derived by the same arithmetic
// a registry class takes: no wind-up, no dying or bone block (a fallen hero is
// drawn from the dying body), and one track per action from its ticks.
func modBodyAnim(b mod.BodySheet) data.UnitAnim {
	c := data.UnitClass{
		MoveBeginPhases: 0, MovePhases: int32(b.Move.Frames),
		AttackPhases: int32(b.Attack.Frames), IdlePhases: int32(b.Idle.Frames),
		DyingPhases: 0, BonePhases: 0,
	}
	if b.Directions == 5 {
		c.Flip = 1
	}
	track := func(a mod.BodyAction) (times, frames []int32) {
		for i, t := range a.Ticks {
			times = append(times, int32(t))
			frames = append(frames, int32(i))
		}
		return times, frames
	}
	c.MoveAnimTime, c.MoveAnimFrame = track(b.Move)
	c.AttackAnimTime, c.AttackAnimFrame = track(b.Attack)
	c.IdleAnimTime, c.IdleAnimFrame = track(b.Idle)
	return c.Anim()
}

// cutBodySheet cuts img into the body's frames in sheet order: the standing
// row, then for move, attack and idle one row per direction. Every frame the
// descriptor addresses must lie inside the picture and hold a visible pixel;
// the refusal names the first that does not.
func cutBodySheet(img image.Image, b mod.BodySheet) ([]*terrain.StaticFrame, error) {
	type cell struct {
		what     string
		col, row int
	}
	var cells []cell
	for i := 0; i < b.StandingFrames(); i++ {
		cells = append(cells, cell{fmt.Sprintf("standing frame %d", i), i, 0})
	}
	row := 1
	for _, a := range []struct {
		name string
		n    int
	}{{"move", b.Move.Frames}, {"attack", b.Attack.Frames}, {"idle", b.Idle.Frames}} {
		if a.n == 0 {
			continue
		}
		for d := 0; d < b.Directions; d++ {
			for j := 0; j < a.n; j++ {
				cells = append(cells, cell{fmt.Sprintf("%s direction %d frame %d", a.name, d, j), j, row})
			}
			row++
		}
	}
	bounds := img.Bounds()
	rgba := make([][]color.NRGBA, len(cells))
	for i, c := range cells {
		r := image.Rect(c.col*b.Width, c.row*b.Height, (c.col+1)*b.Width, (c.row+1)*b.Height).Add(bounds.Min)
		if !r.In(bounds) {
			return nil, fmt.Errorf("the %dx%d picture does not cover %s (the cell at column %d, row %d of %dx%d frames)",
				bounds.Dx(), bounds.Dy(), c.what, c.col, c.row, b.Width, b.Height)
		}
		px := make([]color.NRGBA, 0, b.Width*b.Height)
		visible := false
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				p := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				visible = visible || p.A != 0
				px = append(px, p)
			}
		}
		if !visible {
			return nil, fmt.Errorf("%s (the cell at column %d, row %d) holds no visible pixel", c.what, c.col, c.row)
		}
		rgba[i] = px
	}
	palette, index := bodyPalette(rgba)
	out := make([]*terrain.StaticFrame, len(cells))
	for i, px := range rgba {
		f := &terrain.StaticFrame{Width: b.Width, Height: b.Height, Palette: palette,
			Pixels: make([]terrain.StaticPixel, len(px))}
		for k, p := range px {
			if p.A != 0 {
				f.Pixels[k] = terrain.StaticPixel{Index: index(p), Opaque: true}
			}
		}
		out[i] = f
	}
	return out, nil
}

// bodyPalette is one palette for every frame of a sheet: each colour a visible
// pixel holds, with the low bits of each channel dropped until 256 entries
// hold them all, in ascending colour order. A pixel with no alpha is not
// drawn; every other pixel is drawn opaque.
func bodyPalette(frames [][]color.NRGBA) ([256]color.RGBA, func(color.NRGBA) uint8) {
	var keys [][3]uint8
	var mask uint8
	for drop := 0; drop <= 8; drop++ {
		mask = uint8(0xff) << drop
		seen := map[[3]uint8]bool{}
		keys = keys[:0]
		for _, px := range frames {
			for _, p := range px {
				k := [3]uint8{p.R & mask, p.G & mask, p.B & mask}
				if p.A != 0 && !seen[k] {
					seen[k] = true
					keys = append(keys, k)
				}
			}
		}
		if len(keys) <= 256 {
			break
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		if a[1] != b[1] {
			return a[1] < b[1]
		}
		return a[2] < b[2]
	})
	var palette [256]color.RGBA
	at := make(map[[3]uint8]uint8, len(keys))
	for i, k := range keys {
		palette[i] = color.RGBA{R: k[0], G: k[1], B: k[2], A: 0xff}
		at[k] = uint8(i)
	}
	return palette, func(p color.NRGBA) uint8 { return at[[3]uint8{p.R & mask, p.G & mask, p.B & mask}] }
}
