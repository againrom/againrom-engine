package mod

import (
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"strings"
)

// The tile groups a mod may add. A tile word names its group in 13 bits, but
// the tile loader holds groups 0 to 31 (TERR-TILE-291), and the installs use
// groups 0 to 12 (ALM-TILE-336); the groups between are free.
const (
	FirstModTerrainGroup = 13
	LastModTerrainGroup  = 31
)

// TerrainKinds are the land kinds a group row names, in the terrain
// classifier's class order: kind i is class i+1 (Land 1 to Road 10). A cell
// of the group moves, blocks and prices as a cell of that class.
var TerrainKinds = []string{"land", "grass", "flowers", "sand", "cracked", "stones", "savanna", "mountain", "water", "road"}

// TerrainGroup is one row of a mod's terrain file: tile group Group, whose
// cells are of land kind Class (1 to 10, TerrainKinds), shown in the map
// editor's terrain list as Name and on the minimap in Minimap when HasMinimap.
type TerrainGroup struct {
	Mod  string
	File string
	Line int

	Group      int
	Class      uint8
	Name       string
	Minimap    color.RGBA
	HasMinimap bool
	// Frames are the frames of the strip of each column 0 to 3, set when
	// the game serves the strips; 0 before.
	Frames [4]int
}

// TerrainData is the terrain groups the mods of a set add, in load order.
type TerrainData struct {
	Groups []TerrainGroup
}

// Empty reports whether no mod adds a terrain group.
func (d TerrainData) Empty() bool { return len(d.Groups) == 0 }

// Group is the row of tile group g; false when no mod adds g.
func (d TerrainData) Group(g int) (TerrainGroup, bool) {
	i := slices.IndexFunc(d.Groups, func(r TerrainGroup) bool { return r.Group == g })
	if i < 0 {
		return TerrainGroup{}, false
	}
	return d.Groups[i], true
}

// readTerrain reads one mod's terrain file and adds its groups to d. A group
// is one mod's: a group another mod already adds is refused naming both.
func readTerrain(r DataReader, d *Data, rel string, src []byte) error {
	groups, err := ParseTerrain(r.Mod, rel, src)
	if err != nil {
		return err
	}
	for _, g := range groups.Groups {
		if prior, ok := d.Terrain.Group(g.Group); ok {
			return Source{Mod: g.Mod, File: g.File, Line: g.Line}.Errorf("terrain group %d is added by mod %q (%s:%d) too; a group is one mod's", g.Group, prior.Mod, prior.File, prior.Line)
		}
		d.Terrain.Groups = append(d.Terrain.Groups, g)
	}
	return nil
}

// ParseTerrain reads a terrain file: [[group]] tables of group, kind, name
// and an optional minimap colour "#rrggbb". Each refusal names the file and
// the line.
func ParseTerrain(id, file string, data []byte) (TerrainData, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return TerrainData{}, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return TerrainData{}, bad(p.line, "%s is outside a [[group]] table", p.key)
	}
	var out TerrainData
	for _, t := range tables[1:] {
		if t.name != "group" || !t.array {
			return TerrainData{}, bad(t.line, "unknown table %q (this file holds [[group]])", t.name)
		}
		g := TerrainGroup{Mod: id, File: file, Line: t.line}
		have := map[string]bool{}
		for _, p := range t.pairs {
			have[p.key] = true
			text := func() (string, error) {
				if p.val.kind != tomlString {
					return "", bad(p.line, "%s must be a string, not %s", p.key, p.val.kind)
				}
				return p.val.str, nil
			}
			var err error
			switch p.key {
			case "group":
				switch {
				case p.val.kind != tomlInt:
					err = bad(p.line, "group must be an integer, not %s", p.val.kind)
				case p.val.num < FirstModTerrainGroup || p.val.num > LastModTerrainGroup:
					err = bad(p.line, "group is %d; a mod adds groups %d to %d (the installs hold 0 to 12)", p.val.num, FirstModTerrainGroup, LastModTerrainGroup)
				default:
					g.Group = int(p.val.num)
				}
			case "kind":
				var s string
				if s, err = text(); err == nil {
					if i := slices.Index(TerrainKinds, s); i >= 0 {
						g.Class = uint8(i + 1)
					} else {
						err = bad(p.line, "kind is %q; use one of %s", s, strings.Join(TerrainKinds, ", "))
					}
				}
			case "name":
				if g.Name, err = text(); err == nil && (strings.TrimSpace(g.Name) == "" || len(g.Name) > 32) {
					err = bad(p.line, "name must be 1 to 32 bytes")
				}
			case "minimap":
				var s string
				if s, err = text(); err == nil {
					if g.Minimap, err = parseColour(s); err != nil {
						err = bad(p.line, "minimap is %q; write a colour as \"#rrggbb\"", s)
					}
					g.HasMinimap = true
				}
			default:
				err = bad(p.line, "unknown key %q in [[group]]", p.key)
			}
			if err != nil {
				return TerrainData{}, err
			}
		}
		for _, need := range []string{"group", "kind", "name"} {
			if !have[need] {
				return TerrainData{}, bad(t.line, "[[group]] has no %s", need)
			}
		}
		if prior, ok := out.Group(g.Group); ok {
			return TerrainData{}, bad(t.line, "group %d is already a row at line %d", g.Group, prior.Line)
		}
		out.Groups = append(out.Groups, g)
	}
	return out, nil
}

// parseColour reads "#rrggbb".
func parseColour(s string) (color.RGBA, error) {
	if len(s) != 7 || s[0] != '#' {
		return color.RGBA{}, fmt.Errorf("not #rrggbb")
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return color.RGBA{}, err
	}
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}, nil
}
