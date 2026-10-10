package mod

import (
	"fmt"
	"regexp"
)

// WeaponBodiesFile is the data file that chooses the hero body a weapon is
// drawn with, and BodiesFile the one that supplies a body sheet of the mod's
// own. A mod loads each from its entry script with game.data.add.
const (
	WeaponBodiesFile = "data/weapon-bodies.toml"
	BodiesFile       = "data/bodies.toml"
)

var bodyNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

// Bounds of a body sheet's geometry.
const (
	maxBodyFrameSide = 512
	maxBodyFrames    = 64
	maxBodyTicks     = 255
	maxWeaponRow     = 31
)

// WeaponBody chooses the body one weapon is drawn with. Weapon is the row's
// name in the weapon table, Row the row itself; a mapping gives one or both.
// Body is a body the install ships or one a loaded mod's BodiesFile supplies.
type WeaponBody struct {
	Mod  string
	File string
	Line int

	Weapon     string
	WeaponLine int
	Row        int
	RowLine    int
	Body       string
	BodyLine   int
}

// BodyAction is one animated action of a body sheet: frames per direction and
// how many ticks each frame is held.
type BodyAction struct {
	Frames int
	Ticks  []int
	Line   int
}

// BodySheet is a body a mod supplies: a PNG sheet for each of the two body
// directories and the geometry both share. The sheet is a grid of frame cells:
// row 0 holds the standing frames (16 at 8 directions, 9 at 5), then one row
// per direction for each of move, attack and idle, in that order, each row
// holding that action's frames.
type BodySheet struct {
	Mod  string
	File string
	Line int
	// Dir is the mod folder, set by the loader that read the file.
	Dir string

	Name     string
	NameLine int
	// Heroes and HeroesLight are the sheets for the heroes and heroes_l
	// directories, relative to the mod folder.
	Heroes          string
	HeroesLine      int
	HeroesLight     string
	HeroesLightLine int

	Width, Height    int
	FrameLine        int
	OriginX, OriginY int
	OriginLine       int
	// Directions is 8, or 5 with directions 5 to 7 drawn mirrored.
	Directions     int
	DirectionsLine int

	Move, Attack, Idle BodyAction

	WeaponLast bool
	// Selection is the selection box in canvas pixels; HasSelection is false
	// when the sheet keeps the box of the class an unknown body falls back on.
	Selection     [4]int
	HasSelection  bool
	SelectionLine int
}

// StandingFrames is how many standing frames the sheet's row 0 holds.
func (b BodySheet) StandingFrames() int {
	if b.Directions == 5 {
		return 9
	}
	return 16
}

// BodyData is the weapon mappings and body sheets the mods of a set declare,
// in load order.
type BodyData struct {
	Weapons []WeaponBody
	Bodies  []BodySheet
}

// Empty reports that no mod maps a weapon or supplies a body.
func (d BodyData) Empty() bool { return len(d.Weapons) == 0 && len(d.Bodies) == 0 }

// ParseWeaponBodies reads data/weapon-bodies.toml of the mod id: [[weapon]]
// tables with weapon and/or row, and body. A refusal names the file and line.
func ParseWeaponBodies(id, file string, data []byte) ([]WeaponBody, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return nil, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return nil, bad(p.line, "%s is outside a [[weapon]] table", p.key)
	}
	var out []WeaponBody
	for _, t := range tables[1:] {
		if t.name != "weapon" || !t.array {
			return nil, bad(t.line, "unknown table %q (this file holds [[weapon]])", t.name)
		}
		w := WeaponBody{Mod: id, File: file, Line: t.line}
		for _, p := range t.pairs {
			var err error
			switch p.key {
			case "weapon":
				w.WeaponLine = p.line
				if p.val.kind != tomlString || p.val.str == "" {
					err = bad(p.line, "weapon must be the name of a weapon table row, as a string")
				}
				w.Weapon = p.val.str
			case "row":
				w.RowLine = p.line
				if p.val.kind != tomlInt {
					err = bad(p.line, "row must be an integer, not %s", p.val.kind)
				} else if p.val.num < 1 || p.val.num > maxWeaponRow {
					err = bad(p.line, "row is %d; a weapon row is 1 to %d", p.val.num, maxWeaponRow)
				}
				w.Row = int(p.val.num)
			case "body":
				w.BodyLine = p.line
				if p.val.kind != tomlString {
					err = bad(p.line, "body must be a string, not %s", p.val.kind)
				} else if w.Body = p.val.str; !bodyNamePattern.MatchString(w.Body) {
					err = bad(p.line, "body %q is not a body name: 1 to 32 of a-z, 0-9 and '_', starting with a letter", w.Body)
				}
			default:
				err = bad(p.line, "unknown key %q in [[weapon]]", p.key)
			}
			if err != nil {
				return nil, err
			}
		}
		if w.Weapon == "" && w.Row == 0 {
			return nil, bad(t.line, "[[weapon]] names no weapon: give weapon, row or both")
		}
		if w.BodyLine == 0 {
			return nil, bad(t.line, "[[weapon]] has no body")
		}
		out = append(out, w)
	}
	return out, nil
}

// ParseBodies reads data/bodies.toml of the mod id: [[body]] tables. A refusal
// names the file and line. Whether the sheets exist and cover every frame is
// decided where the pictures are read.
func ParseBodies(id, file string, data []byte) ([]BodySheet, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return nil, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return nil, bad(p.line, "%s is outside a [[body]] table", p.key)
	}
	var out []BodySheet
	names := map[string]int{}
	for _, t := range tables[1:] {
		if t.name != "body" || !t.array {
			return nil, bad(t.line, "unknown table %q (this file holds [[body]])", t.name)
		}
		b := BodySheet{Mod: id, File: file, Line: t.line}
		have := map[string]bool{}
		for _, p := range t.pairs {
			have[p.key] = true
			ints := func(n int, what string) ([]int, error) {
				if p.val.kind != tomlArray || len(p.val.list) != n {
					return nil, bad(p.line, "%s must be an array of %d integers", p.key, n)
				}
				out := make([]int, n)
				for i, v := range p.val.list {
					if v.kind != tomlInt || v.num < 0 || v.num > maxBodyFrameSide {
						return nil, bad(p.line, "%s must hold %s of 0 to %d", p.key, what, maxBodyFrameSide)
					}
					out[i] = int(v.num)
				}
				return out, nil
			}
			var err error
			switch p.key {
			case "name":
				b.NameLine = p.line
				if p.val.kind != tomlString {
					err = bad(p.line, "name must be a string, not %s", p.val.kind)
				} else if b.Name = p.val.str; !bodyNamePattern.MatchString(b.Name) {
					err = bad(p.line, "name %q is not a body name: 1 to 32 of a-z, 0-9 and '_', starting with a letter", b.Name)
				}
			case "heroes", "heroes_l":
				path := p.val.str
				if p.val.kind != tomlString {
					err = bad(p.line, "%s must be a string, not %s", p.key, p.val.kind)
				} else {
					err = checkModPath(file, p.line, p.key, path, ".png")
				}
				if p.key == "heroes" {
					b.Heroes, b.HeroesLine = path, p.line
				} else {
					b.HeroesLight, b.HeroesLightLine = path, p.line
				}
			case "frame":
				b.FrameLine = p.line
				var v []int
				if v, err = ints(2, "a width and a height"); err == nil {
					b.Width, b.Height = v[0], v[1]
					if b.Width < 1 || b.Height < 1 {
						err = bad(p.line, "frame is %dx%d; each side is 1 to %d pixels", b.Width, b.Height, maxBodyFrameSide)
					}
				}
			case "origin":
				b.OriginLine = p.line
				var v []int
				if v, err = ints(2, "an x and a y"); err == nil {
					b.OriginX, b.OriginY = v[0], v[1]
				}
			case "directions":
				b.DirectionsLine = p.line
				if p.val.kind != tomlInt || p.val.num != 8 && p.val.num != 5 {
					err = bad(p.line, "directions must be 8, or 5 with directions 5 to 7 drawn mirrored")
				}
				b.Directions = int(p.val.num)
			case "move":
				b.Move, err = bodyAction(file, p, 1)
			case "attack":
				b.Attack, err = bodyAction(file, p, 1)
			case "idle":
				b.Idle, err = bodyAction(file, p, 0)
			case "weapon-last":
				if p.val.kind != tomlBool {
					err = bad(p.line, "weapon-last must be true or false, not %s", p.val.kind)
				}
				b.WeaponLast = p.val.flag
			case "selection":
				b.SelectionLine = p.line
				var v []int
				if v, err = ints(4, "x1, y1, x2 and y2"); err == nil {
					copy(b.Selection[:], v)
					b.HasSelection = true
				}
			default:
				err = bad(p.line, "unknown key %q in [[body]]", p.key)
			}
			if err != nil {
				return nil, err
			}
		}
		for _, need := range []string{"name", "heroes", "heroes_l", "frame", "origin", "directions", "move", "attack"} {
			if !have[need] {
				return nil, bad(t.line, "[[body]] has no %s", need)
			}
		}
		if b.OriginX > b.Width || b.OriginY > b.Height {
			return nil, bad(b.OriginLine, "origin (%d, %d) is outside the %dx%d frame", b.OriginX, b.OriginY, b.Width, b.Height)
		}
		if b.HasSelection {
			s := b.Selection
			if s[0] > s[2] || s[1] > s[3] || s[2] > b.Width || s[3] > b.Height {
				return nil, bad(b.SelectionLine, "selection %v is not a box inside the %dx%d frame", s, b.Width, b.Height)
			}
		}
		if first, dup := names[b.Name]; dup {
			return nil, bad(t.line, "body %q is already supplied at line %d", b.Name, first)
		}
		names[b.Name] = t.line
		out = append(out, b)
	}
	return out, nil
}

// bodyAction reads one action: { frames = n, ticks = t } where ticks is one
// count for every frame or an array of one count per frame.
func bodyAction(file string, p tomlPair, minFrames int) (BodyAction, error) {
	bad := func(format string, args ...any) error {
		return &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf(format, args...)}
	}
	a := BodyAction{Line: p.line}
	if p.val.kind != tomlInline {
		return a, bad("%s must be an inline table { frames = n, ticks = t }", p.key)
	}
	var ticks *tomlValue
	for i, q := range p.val.tbl {
		switch q.key {
		case "frames":
			if q.val.kind != tomlInt || q.val.num < int64(minFrames) || q.val.num > maxBodyFrames {
				return a, bad("%s.frames must be an integer of %d to %d", p.key, minFrames, maxBodyFrames)
			}
			a.Frames = int(q.val.num)
		case "ticks":
			ticks = &p.val.tbl[i].val
		default:
			return a, bad("unknown key %q in %s", q.key, p.key)
		}
	}
	if a.Frames < minFrames {
		return a, bad("%s has no frames", p.key)
	}
	if ticks == nil {
		return a, bad("%s has no ticks", p.key)
	}
	tick := func(v tomlValue) (int, error) {
		if v.kind != tomlInt || v.num < 1 || v.num > maxBodyTicks {
			return 0, bad("%s.ticks must hold integers of 1 to %d", p.key, maxBodyTicks)
		}
		return int(v.num), nil
	}
	switch ticks.kind {
	case tomlInt:
		n, err := tick(*ticks)
		if err != nil {
			return a, err
		}
		for i := 0; i < a.Frames; i++ {
			a.Ticks = append(a.Ticks, n)
		}
	case tomlArray:
		if len(ticks.list) != a.Frames {
			return a, bad("%s.ticks holds %d counts for %d frames", p.key, len(ticks.list), a.Frames)
		}
		for _, v := range ticks.list {
			n, err := tick(v)
			if err != nil {
				return a, err
			}
			a.Ticks = append(a.Ticks, n)
		}
	default:
		return a, bad("%s.ticks must be an integer or an array of integers", p.key)
	}
	return a, nil
}
