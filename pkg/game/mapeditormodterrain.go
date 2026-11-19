package game

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/formats/pal"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// terrainAddKey is the terrain page's button that adds a ground from the
// picture its value names; a ground's fields are "<group>.<field>". The key
// differs from W's own Add button, which the same window routes by key.
const terrainAddKey = "newground"

// groundSquares is the most squares a ground's picture may hold: four
// strips of the most frames a strip takes.
const groundSquares = 4 * terrain.MaxStripFrames

// freeTerrainGroup is the first group number past the install's that rows
// leave free; false when every one is taken.
func freeTerrainGroup(rows mod.TerrainData) (int, bool) {
	for g := alm.InstallTileGroups; g < alm.TileGroupCount; g++ {
		if _, taken := rows.Group(g); !taken {
			return g, true
		}
	}
	return 0, false
}

// terrainKindWord is the editor's word for land kind class (1 to 10).
func (x *MapInspector) terrainKindWord(class uint8) string {
	if class < 1 || int(class) > len(mod.TerrainKinds) {
		return strconv.Itoa(int(class))
	}
	return x.word("editor.terrain." + mod.TerrainKinds[class-1])
}

// terrainFields is W's terrain page: the button that adds a ground, the
// pressed mod ground's name, kind and delete button, then one grid of every
// ground, the game's then the mod's, each with its variations as its hover
// pictures; none with no mod folder open.
func (s *mapInspectionEdits) terrainFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	line := x.pageLine
	held, err := s.heldRows(s.workspace.Project, nil, mod.TerrainFile)
	if err != nil {
		return []ui.InspectionField{line("error", "", err.Error())}
	}
	rows := held.Terrain
	var out []ui.InspectionField
	if _, ok := freeTerrainGroup(rows); ok {
		out = append(out, ui.InspectionField{Key: terrainAddKey, Label: x.word("editor.terrain.add"), Meaning: x.word("editor.terrain.add.help"), Action: true, Column: 1, Pick: []string{"png", "bmp"}})
	} else {
		out = append(out, line("full", "", x.word("editor.terrain.full")))
	}
	var kinds []ui.InspectionChoice
	for i := range mod.TerrainKinds {
		kinds = append(kinds, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(x.terrainKindWord(uint8(i + 1)))})
	}
	grid := ui.InspectionField{Key: terrainGridKey, Label: x.word("editor.terrain.grounds"), Grid: true, Value: int64(s.ground),
		Icons: map[int64]*image.RGBA{}, Variations: map[int64][]*image.RGBA{}}
	cell := func(value int64, name string, words []uint16) {
		grid.Choices = append(grid.Choices, ui.InspectionChoice{Value: value, Label: x.inspectionUTF8(name)})
		for _, w := range words {
			if pic := swatchOf(x.tiles, w); pic != nil {
				grid.Variations[value] = append(grid.Variations[value], pic)
			}
		}
		if len(grid.Variations[value]) > 0 {
			grid.Icons[value] = grid.Variations[value][0]
		}
	}
	if _, held := rows.Group(s.ground); !held && len(rows.Groups) > 0 && s.ground >= 0 {
		grid.Value = int64(rows.Groups[0].Group)
	}
	for i, p := range mapedit.Palette {
		cell(int64(-i-1), x.terrainKindWord(p.Class), mapedit.Variants(p.Class))
	}
	for _, g := range rows.Groups {
		held := x.mod.strips[g.Group]
		var words []uint16
		if !slices.Contains(held[:], 0) {
			g.Frames = held
			words = groupWords(g)
		}
		cell(int64(g.Group), g.Name, words)
		if int64(g.Group) != grid.Value {
			continue
		}
		key, start := strconv.Itoa(g.Group)+".", len(out)
		if words == nil {
			out = append(out, line(key+"state", "", x.word("editor.terrain.incomplete")))
		}
		name := workspaceText(key+"name", x.word("editor.terrain.name"), g.Name)
		name.Meaning = x.word("editor.terrain.name.help")
		out = append(out, name,
			ui.InspectionField{Key: key + "kind", Label: x.word("editor.terrain.kind"), Meaning: x.word("editor.terrain.kind.help"), Value: int64(g.Class), Choices: kinds},
			ui.InspectionField{Key: key + "remove", Label: x.word("editor.terrain.remove"), Meaning: x.word("editor.terrain.remove.help"), Action: true, Column: 1})
		for i := start; i < len(out); i++ {
			out[i].Group = g.Name
		}
	}
	return append(out, grid)
}

// terrainGridKey is the terrain page's grid of grounds; a press picks the
// ground whose controls the page shows above it.
const terrainGridKey = "grounds"

// setTerrainField makes one edit of the terrain page a draft step; a value
// the loader refuses is refused in its words.
func (s *mapInspectionEdits) setTerrainField(key, value string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	if key == terrainGridKey {
		g, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		s.ground = g
		return nil, nil
	}
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return nil, err
	}
	if isModAssetKey(key) {
		d, _, err := s.setModAsset(key, value)
		return d, err
	}
	held, err := s.heldRows(s.workspace.Project, nil, mod.TerrainFile)
	if err != nil {
		return nil, err
	}
	rows := held.Terrain
	if key == terrainAddKey {
		return s.addGround(rows, strings.TrimSpace(value))
	}
	number, field, _ := strings.Cut(key, ".")
	g, err := strconv.Atoi(number)
	if _, held := rows.Group(g); err != nil || !held {
		return nil, fmt.Errorf(x.word("editor.terrain.err.gone"), key)
	}
	var drop []string
	if field == "remove" {
		row, _ := rows.Group(g)
		if painted := s.groupPainters(g); len(painted) > 0 {
			return nil, fmt.Errorf(x.word("editor.terrain.err.painted"), row.Name, strings.Join(painted, ", "))
		}
		for b := range 4 {
			drop = append(drop, modStripFile(g, b))
		}
	}
	return s.editTerrain(func(t *mod.TerrainData) error {
		i := slices.IndexFunc(t.Groups, func(r mod.TerrainGroup) bool { return r.Group == g })
		r := &t.Groups[i]
		switch field {
		case "kind":
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 1 || n > len(mod.TerrainKinds) {
				return fmt.Errorf(x.word("editor.terrain.err.gone"), key)
			}
			r.Class = uint8(n)
		case "name":
			r.Name = strings.TrimSpace(value)
		case "remove":
			t.Groups = slices.Delete(t.Groups, i, i+1)
		default:
			return fmt.Errorf(x.word("editor.terrain.err.gone"), key)
		}
		return nil
	}, drop, nil)
}

// addGround is one draft step that adds a ground from the host picture
// source: its squares in four strips of one colour table, the first free
// group, and a row of land kind 1 named for the file with the picture's
// average colour on the minimap.
func (s *mapInspectionEdits) addGround(rows mod.TerrainData, source string) (*ui.InspectionDocument, error) {
	x := s.inspector
	if source == "" {
		return nil, errors.New(x.word("editor.terrain.err.nofile"))
	}
	g, ok := freeTerrainGroup(rows)
	if !ok {
		return nil, errors.New(x.word("editor.terrain.full"))
	}
	file := filepath.Base(source)
	unread := func(err error) error { return fmt.Errorf(x.word("editor.terrain.err.picture"), file, err) }
	data, err := x.hostFileBytes(source, mod.MaxFileBytes, func([]byte) error { return nil }, unread(errors.New(x.word("editor.err.afolder"))))
	if err != nil {
		return nil, err
	}
	pic, err := groundPicture(data, func(w, h int) error {
		if w%terrain.CellSize != 0 || h%terrain.CellSize != 0 {
			return fmt.Errorf(x.word("editor.terrain.err.size"), file, w, h)
		}
		if n := (w / terrain.CellSize) * (h / terrain.CellSize); n < 1 || n > groundSquares {
			return fmt.Errorf(x.word("editor.terrain.err.count"), file, n, groundSquares)
		}
		return nil
	})
	if err != nil {
		var plain groundSizeError
		if errors.As(err, &plain) {
			return nil, plain.err
		}
		return nil, unread(err)
	}
	strips, err := groundStrips(pic)
	if err != nil {
		return nil, unread(err)
	}
	put := map[string][]byte{}
	for b, data := range strips {
		put[modStripFile(g, b)] = data
	}
	row := mod.TerrainGroup{Group: g, Class: 1, Name: groundName(file), Minimap: averageColour(pic), HasMinimap: true}
	d, err := s.editTerrain(func(t *mod.TerrainData) error {
		t.Groups = append(t.Groups, row)
		return nil
	}, nil, put)
	if err == nil {
		s.ground = g
	}
	return d, err
}

// groundSizeError carries a refusal of a picture's size in the page's words.
type groundSizeError struct{ err error }

func (e groundSizeError) Error() string { return e.err.Error() }

// groundPicture is the PNG or BMP picture data holds, its size first held to
// fits; a BMP is read as a mod's picture, at the depths the game reads.
func groundPicture(data []byte, fits func(w, h int) error) (*image.RGBA, error) {
	if bytes.HasPrefix(data, []byte("BM")) {
		pic, err := decodeModPicture(data, bmpDepth(data))
		if err != nil {
			return nil, err
		}
		if err := fits(pic.Bounds().Dx(), pic.Bounds().Dy()); err != nil {
			return nil, groundSizeError{err}
		}
		out := image.NewRGBA(image.Rect(0, 0, pic.Bounds().Dx(), pic.Bounds().Dy()))
		draw.Draw(out, out.Bounds(), pic, pic.Bounds().Min, draw.Src)
		return out, nil
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if err := fits(cfg.Width, cfg.Height); err != nil {
		return nil, groundSizeError{err}
	}
	pic, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	out := image.NewRGBA(image.Rect(0, 0, pic.Bounds().Dx(), pic.Bounds().Dy()))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	draw.Draw(out, out.Bounds(), pic, pic.Bounds().Min, draw.Over)
	return out, nil
}

// groundStrips are the four strips of the squares of pic, a whole number of
// cells each way, in reading order: square i goes to strip i mod 4, and with
// fewer than four squares strip b takes square b mod their count. Every
// strip shares the one colour table the reduction makes.
func groundStrips(pic *image.RGBA) ([4][]byte, error) {
	var out [4][]byte
	reduced, err := pal.Reduce(pic, pal.EntryCount)
	if err != nil {
		return out, err
	}
	across := pic.Bounds().Dx() / terrain.CellSize
	n := across * (pic.Bounds().Dy() / terrain.CellSize)
	for b := range out {
		var squares []int
		for i := b; i < n; i += 4 {
			squares = append(squares, i)
		}
		if n < 4 {
			squares = []int{b % n}
		}
		strip := image.NewPaletted(image.Rect(0, 0, terrain.CellSize, terrain.CellSize*len(squares)), reduced.Palette)
		for k, i := range squares {
			at := image.Pt(i%across*terrain.CellSize, i/across*terrain.CellSize)
			draw.Draw(strip, image.Rect(0, k*terrain.CellSize, terrain.CellSize, (k+1)*terrain.CellSize), reduced, at, draw.Src)
		}
		if out[b], err = bmp.EncodePaletted(strip); err != nil {
			return out, err
		}
	}
	return out, nil
}

// averageColour is the mean colour of pic's pixels.
func averageColour(pic *image.RGBA) color.RGBA {
	var sum [3]int
	n := 0
	for i := 0; i+3 < len(pic.Pix); i += 4 {
		sum[0], sum[1], sum[2] = sum[0]+int(pic.Pix[i]), sum[1]+int(pic.Pix[i+1]), sum[2]+int(pic.Pix[i+2])
		n++
	}
	mean := func(v int) uint8 { return uint8((v + n/2) / max(1, n)) }
	return color.RGBA{R: mean(sum[0]), G: mean(sum[1]), B: mean(sum[2]), A: 0xff}
}

// groundName is a new ground's name: the picture's file name without its
// extension, cut to the 32 bytes a name takes.
func groundName(file string) string {
	name := strings.TrimSpace(strings.TrimSuffix(file, filepath.Ext(file)))
	for len(name) > 32 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	if strings.TrimSpace(name) == "" {
		return "ground"
	}
	return name
}

// groupPainters are the mod's maps that paint tile group g: the map these
// edits hold as it stands, every other map as the project or the folder
// holds it.
func (s *mapInspectionEdits) groupPainters(g int) []string {
	d := s.project0()
	if d == nil {
		return nil
	}
	view := projectView(nil, d.Files)
	own := namefold.ModTree(s.ownMap())
	paints := func(m *alm.Map) bool {
		return m != nil && slices.ContainsFunc(m.Tiles, func(w uint16) bool { return alm.TileGroup(w) == g })
	}
	var out []string
	for _, e := range s.modMaps() {
		for _, rel := range e.Files {
			var m *alm.Map
			if own != "" && namefold.ModTree(rel) == own && s.model != nil {
				m, _ = s.model.Map()
			} else {
				b, drafted := view.Draft(rel)
				if !drafted {
					b, _ = modDataFile(s.inspector.mod.dir, rel)
				}
				m, _ = alm.Open(b)
			}
			if paints(m) {
				out = append(out, rel)
				break
			}
		}
	}
	return out
}

// editTerrain is one draft step of the terrain file: edit changes its rows
// through editRows; the files drop names leave the project with it and the
// files put holds enter it.
func (s *mapInspectionEdits) editTerrain(edit func(*mod.TerrainData) error, drop []string, put map[string][]byte) (*ui.InspectionDocument, error) {
	return s.editKind(func(k kindFiles) error {
		err := k.editRows(nil, []string{mod.TerrainFile}, func(d *mod.Data) error { return edit(&d.Terrain) })
		for _, rel := range drop {
			if err == nil {
				err = k.drop(rel)
			}
		}
		for _, rel := range slices.Sorted(maps.Keys(put)) {
			if err == nil {
				err = k.put(rel, put[rel])
			}
		}
		return err
	})
}

// kindFiles is one draft step's hold on the project w's files of the kinds
// the project holds whole (mod.HeldDraft); a file of another kind is
// refused.
type kindFiles struct {
	s *mapInspectionEdits
	w *editorWorkspaceState
}

// editKind is one draft step of the project's files of the held kinds:
// edit writes them through k.
func (s *mapInspectionEdits) editKind(edit func(k kindFiles) error) (*ui.InspectionDocument, error) {
	m := s.inspector.mod
	return s.changeWorkspace(func(w *editorWorkspaceState) error {
		if w.Project == nil {
			w.Project = m.document()
		}
		return edit(kindFiles{s: s, w: w})
	})
}

// put makes out the draft of the file rel names.
func (k kindFiles) put(rel string, out []byte) error {
	if !mod.HeldDraft(rel) {
		return wordErr("editor.err.nokind.held", rel)
	}
	k.w.Project.Files[rel] = out
	return nil
}

// rowText is a page's text lookup over the project d.
type rowText func(d *editorModDocument) (func(key string) (string, bool), error)

// heldRows is the records of the held files rels name in the project d
// (nil: the mod's), each read by its kind's row; text, when set, is the
// lookup a kind citing text asks.
func (s *mapInspectionEdits) heldRows(d *editorModDocument, text rowText, rels ...string) (mod.Data, error) {
	m := s.inspector.mod
	if d == nil {
		d = m.document()
	}
	r := mod.DataReader{Mod: m.id, Lang: s.inspector.pageLanguage()}
	var out mod.Data
	if text != nil {
		look, err := text(d)
		if err != nil {
			return out, err
		}
		r.Text = look
	}
	for _, rel := range rels {
		kind, ok := mod.WrittenKind(rel)
		if !ok {
			return out, wordErr("editor.err.nokind.written", rel)
		}
		if err := kind.ReadHeld(r, &out, rel, d.Files[rel]); err != nil {
			return out, err
		}
	}
	return out, nil
}

// editRows is the one page step of the held files rels name: it reads
// their records (heldRows), lets edit change them, and writes each file
// through its kind's writer over the draft's bytes. A file whose bytes
// change is put, or leaves the mod when it writes blank: each writer reads
// its file back as the records given, so a blank file holds none.
func (k kindFiles) editRows(text rowText, rels []string, edit func(*mod.Data) error) error {
	d, err := k.s.heldRows(k.w.Project, text, rels...)
	if err != nil {
		return err
	}
	d.Before = k.w.Project.data().Before
	if err := edit(&d); err != nil {
		return err
	}
	for _, rel := range rels {
		src := k.w.Project.Files[rel]
		out, err := writeHeld(rel, src, d)
		switch {
		case err != nil:
		case bytes.Equal(out, src):
		case strings.TrimSpace(string(out)) == "":
			err = k.drop(rel)
		default:
			err = k.put(rel, out)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// writeHeld is the file source of rel's kind with the records of data.
func writeHeld(rel string, source []byte, data mod.Data) ([]byte, error) {
	kind, ok := mod.WrittenKind(rel)
	if !ok {
		return nil, wordErr("editor.err.nokind.written", rel)
	}
	return kind.Write(rel, source, data)
}

// drop takes the file rel names out of the project: the next save removes
// it from the folder when the folder holds it.
func (k kindFiles) drop(rel string) error {
	if !mod.HeldDraft(rel) {
		return wordErr("editor.err.nokind.held", rel)
	}
	for _, name := range slices.Collect(maps.Keys(k.w.Project.Files)) {
		if mod.FoldAddress(name) == mod.FoldAddress(rel) {
			delete(k.w.Project.Files, name)
		}
	}
	if st, err := os.Stat(filepath.Join(k.s.inspector.mod.dir, filepath.FromSlash(rel))); err == nil && !st.IsDir() {
		k.w.Project.Files[rel] = nil
	}
	return nil
}

// modStripFile is the mod file of tile group g's strip in column b.
func modStripFile(g, b int) string {
	return mod.FilesDir + "/" + mod.CommonFiles + "/" + terrain.GroupStripPath(g, b)
}
