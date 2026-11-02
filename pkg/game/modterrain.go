package game

import (
	"image/color"
	"slices"
	"strings"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// modTerrain pairs the terrain groups the mods add with the strips the mods
// serve, and gives each row the frames of its four strips, read from src
// through the tile loader's own strip reader. A strip a mod serves for a
// group no row of that mod adds, and a row whose group lacks one of its four
// strips, are refused naming the mod and the file. served is the mods'
// entries by folded address; nil when no mod serves a file.
func modTerrain(d mod.TerrainData, served map[string]modFileEntry, src terrain.EntrySource) (mod.TerrainData, error) {
	keys := make([]string, 0, len(served))
	for k := range served {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		e := served[k]
		g, _, ok := terrain.ModGroupStrip(k)
		if !ok || e.mod == "" {
			continue
		}
		if row, ok := d.Group(g); !ok || row.Mod != e.mod {
			return mod.TerrainData{}, mod.Source{Mod: e.mod, File: e.file}.Errorf("is a strip of terrain group %d, which no row of this mod's %s adds", g, mod.TerrainFile)
		}
	}
	out := mod.TerrainData{Groups: slices.Clone(d.Groups)}
	for i, row := range out.Groups {
		for b := range row.Frames {
			path := terrain.GroupStripPath(row.Group, b)
			at := mod.Source{Mod: row.Mod, File: row.File, Line: row.Line}
			if _, ok := served[mod.FoldAddress(path)]; !ok {
				return mod.TerrainData{}, at.Errorf("terrain group %d has no strip for column %d; a group carries four strips, files/%s/%s and the next three", row.Group, b, mod.CommonFiles, terrain.GroupStripPath(row.Group, 0))
			}
			data, err := src.ReadFile(path)
			if err != nil {
				return mod.TerrainData{}, at.Errorf("terrain group %d: %s: %v", row.Group, path, err)
			}
			strip, err := terrain.ReadStrip(data)
			if err != nil {
				return mod.TerrainData{}, at.Errorf("terrain group %d: %s: %v", row.Group, path, err)
			}
			out.Groups[i].Frames[b] = len(strip.SubCells)
		}
	}
	return out, nil
}

// setModTerrain gives the table the mods' terrain groups with their frames,
// refusing a strip with no row and a row with no strip.
func (f *FrontEnd) setModTerrain(d mod.TerrainData) error {
	var served map[string]modFileEntry
	var src terrain.EntrySource
	if f.Archives != nil {
		src = f.Archives.Containers
		if f.Archives.mods != nil {
			served = f.Archives.mods.owners
		}
	}
	rows, err := modTerrain(d, served, src)
	if err != nil {
		return err
	}
	if f.Table != nil {
		f.Table.Mods.Terrain = rows
	}
	return nil
}

// groupWords are the words a mod group's brush paints: every frame of each
// of its four columns.
func groupWords(r mod.TerrainGroup) []uint16 {
	var out []uint16
	for b, n := range r.Frames {
		for s := 0; s < n; s++ {
			out = append(out, uint16(r.Group<<6|b<<4|s))
		}
	}
	return out
}

// useModTerrain gives the editor the working mod's terrain: the tileset with
// the mod's terrain strips served over the install, and the mod's groups on
// the table the editor classifies and lists by.
func (x *MapInspector) useModTerrain(result modrt.Result) error {
	if x.archives == nil {
		return nil
	}
	fs := x.archives.Containers.Fork()
	b := &modFiles{fs: fs, order: result.Files.Order, owners: map[string]modFileEntry{}}
	a := *x.archives
	a.Containers, a.mods = fs, b
	var strips []mod.InstallFile
	for _, file := range result.Files.Files {
		if strings.HasPrefix(mod.FoldAddress(file.Address), terrain.TilePathPrefix) {
			strips = append(strips, file)
		}
	}
	if err := b.serveInstallFiles(&a, strips); err != nil {
		return err
	}
	rows, err := modTerrain(result.Terrain, b.owners, fs)
	if err != nil {
		return err
	}
	x.tiles, x.swatches = terrain.LoadTileset(fs), nil
	if x.table != nil {
		t := *x.table
		t.Mods.Terrain = rows
		x.table = &t
	}
	return nil
}

// modBrushTerrains are the terrain list's rows of the mod groups t's mods
// add, each with the swatch of its column 0, frame 0.
func modBrushTerrains(t *mapload.Table, tiles *terrain.Tileset) []ui.BrushTerrain {
	groups := t.ModTerrain()
	if groups == nil {
		return nil
	}
	var out []ui.BrushTerrain
	for _, r := range groups.Groups {
		out = append(out, ui.BrushTerrain{Class: mapedit.GroupBrush(r.Group), Name: r.Name, Swatch: swatchOf(tiles, uint16(r.Group<<6))})
	}
	return out
}

// minimapGroupColours are the minimap colours of the mod groups: the row's
// own colour, else the lookup's colour of its land kind's base word.
func minimapGroupColours(p *ui.MinimapPaint, t *mapload.Table) {
	groups := t.ModTerrain()
	if p == nil || p.Colours == nil || groups == nil {
		return
	}
	for _, r := range groups.Groups {
		c := r.Minimap
		if !r.HasMinimap {
			for _, e := range mapedit.Palette {
				if e.Class == r.Class {
					lb := p.Colours.Bounds()
					c = p.Colours.RGBAAt(lb.Min.X+int(e.Word>>4), lb.Min.Y+int(e.Word&15))
					break
				}
			}
		}
		c.A = 0xff
		p.GroupColours[r.Group] = color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
	}
}
