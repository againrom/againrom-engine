package game

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mod"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The terrain page's keys: a group's fields are "<group>.<field>", a strip
// "<group>.strip.<column>"; "new" adds the typed group and "add" the first
// free one.
const (
	terrainNewKey   = "new"
	terrainAddKey   = "add"
	terrainStripKey = "strip."
)

// terrainRows is the open project's draft of the mod's terrain file: its
// rows and its bytes, the file on disk until the page edits it.
func (s *mapInspectionEdits) terrainRows(w *editorWorkspaceState) (mod.TerrainData, []byte, error) {
	m := s.inspector.mod
	d := w.Project
	if d == nil {
		d = m.document()
	}
	src := d.Files[mod.TerrainFile]
	t, err := mod.ParseTerrain(m.id, mod.TerrainFile, src)
	return t, src, err
}

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

// terrainFields is W's terrain page: each group the mod adds and the row
// that adds the next; none with no mod folder open.
func (s *mapInspectionEdits) terrainFields() []ui.InspectionField {
	x := s.inspector
	if x.mod == nil || x.mod.packed || s.prepareWorkspace() != nil {
		return nil
	}
	line := x.pageLine
	rows, _, err := s.terrainRows(s.workspace)
	if err != nil {
		return []ui.InspectionField{line("error", "", err.Error())}
	}
	var kinds []ui.InspectionChoice
	for i, k := range mod.TerrainKinds {
		kinds = append(kinds, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(k)})
	}
	var out []ui.InspectionField
	for _, g := range rows.Groups {
		head := fmt.Sprintf(x.word("editor.terrain.group"), g.Group, g.Name)
		key, start := strconv.Itoa(g.Group)+".", len(out)
		colour := ""
		if g.HasMinimap {
			colour = mod.TerrainColour(g.Minimap)
		}
		minimap := workspaceText(key+"minimap", x.word("editor.terrain.minimap"), colour)
		minimap.Meaning = x.word("editor.terrain.minimap.hint")
		out = append(out,
			ui.InspectionField{Key: key + "kind", Label: x.word("editor.terrain.kind"), Value: int64(g.Class), Choices: kinds},
			workspaceText(key+"name", x.word("editor.terrain.name"), g.Name),
			minimap)
		n := 0
		for b, frames := range x.mod.strips[g.Group] {
			f := workspaceText(key+terrainStripKey+strconv.Itoa(b), fmt.Sprintf(x.word("editor.terrain.strip"), b+1), "")
			f.Meaning = x.word("editor.terrain.strip.none")
			if frames > 0 {
				f.Meaning = fmt.Sprintf(x.word("editor.terrain.strip.held"), terrain.GroupStripPath(g.Group, b), frames)
				n++
			}
			f.Pick = []string{"bmp"}
			out = append(out, f)
		}
		state := x.word("editor.terrain.complete")
		if n < 4 {
			state = fmt.Sprintf(x.word("editor.terrain.incomplete"), n)
		}
		out = append(out, line(key+"state", "", state),
			ui.InspectionField{Key: key + "remove", Label: fmt.Sprintf(x.word("editor.terrain.remove"), g.Group), Action: true, Column: 1})
		for i := start; i < len(out); i++ {
			out[i].Group = head
		}
	}
	if g, ok := freeTerrainGroup(rows); ok {
		f := workspaceText(terrainNewKey, x.word("editor.terrain.new"), strconv.Itoa(g))
		f.Meaning = fmt.Sprintf(x.word("editor.terrain.new.hint"), alm.InstallTileGroups, alm.TileGroupCount-1)
		out = append(out, f, ui.InspectionField{Key: terrainAddKey, Label: fmt.Sprintf(x.word("editor.terrain.add"), g), Action: true, Column: 1})
	} else {
		out = append(out, line("full", "", x.word("editor.terrain.full")))
	}
	return out
}

// setTerrainField makes one edit of the terrain page a draft step; a value
// the loader refuses is refused in its words.
func (s *mapInspectionEdits) setTerrainField(key, value string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return nil, err
	}
	if isModAssetKey(key) {
		d, _, err := s.setModAsset(key, value)
		return d, err
	}
	rows, _, err := s.terrainRows(s.workspace)
	if err != nil {
		return nil, err
	}
	if key == terrainNewKey || key == terrainAddKey {
		g, ok := freeTerrainGroup(rows)
		if key == terrainNewKey {
			g, err = strconv.Atoi(strings.TrimSpace(value))
			ok = err == nil && g >= alm.InstallTileGroups && g < alm.TileGroupCount
		}
		if !ok {
			return nil, fmt.Errorf(x.word("editor.terrain.err.number"), alm.InstallTileGroups, alm.TileGroupCount-1)
		}
		if _, taken := rows.Group(g); taken {
			return nil, fmt.Errorf(x.word("editor.terrain.err.taken"), g)
		}
		return s.editTerrain(func(t *mod.TerrainData) error {
			t.Groups = append(t.Groups, mod.TerrainGroup{Group: g, Class: 1, Name: fmt.Sprintf(x.word("editor.terrain.newname"), g)})
			return nil
		}, nil)
	}
	number, field, _ := strings.Cut(key, ".")
	g, err := strconv.Atoi(number)
	if _, held := rows.Group(g); err != nil || !held {
		return nil, fmt.Errorf(x.word("editor.terrain.err.gone"), key)
	}
	if b, ok := strings.CutPrefix(field, terrainStripKey); ok {
		column, err := strconv.Atoi(b)
		if err != nil || column < 0 || column > 3 {
			return nil, fmt.Errorf(x.word("editor.terrain.err.gone"), key)
		}
		_, d, err := s.ImportTerrainStrip(g, column, strings.TrimSpace(value), false)
		return d, err
	}
	edit := func(t *mod.TerrainData) error {
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
		case "minimap":
			r.HasMinimap = strings.TrimSpace(value) != ""
			if r.HasMinimap {
				c, err := mod.ParseTerrainColour(strings.TrimSpace(value))
				if err != nil {
					return fmt.Errorf(x.word("editor.terrain.err.colour"), g, strings.TrimSpace(value))
				}
				r.Minimap = c
			}
		case "remove":
			t.Groups = slices.Delete(t.Groups, i, i+1)
		default:
			return fmt.Errorf(x.word("editor.terrain.err.gone"), key)
		}
		return nil
	}
	var strips []string
	if field == "remove" {
		for b := range 4 {
			strips = append(strips, modStripFile(g, b))
		}
	}
	return s.editTerrain(edit, strips)
}

// editTerrain is one draft step of the terrain file: edit changes its rows
// and the kind's writer writes them over the draft's bytes; the files drop
// names leave the project with it. A file with no row left leaves the mod.
func (s *mapInspectionEdits) editTerrain(edit func(*mod.TerrainData) error, drop []string) (*ui.InspectionDocument, error) {
	return s.editKind(func(k kindFiles) error {
		rows, src, err := s.terrainRows(k.w)
		if err != nil {
			return err
		}
		rows.Groups = slices.Clone(rows.Groups)
		if err := edit(&rows); err != nil {
			return err
		}
		kind, _ := mod.WrittenKind(mod.TerrainFile)
		out, err := kind.Write(mod.TerrainFile, src, mod.Data{Terrain: rows})
		if err != nil {
			return err
		}
		if len(rows.Groups) == 0 && strings.TrimSpace(string(out)) == "" {
			err = k.drop(mod.TerrainFile)
		} else {
			err = k.put(mod.TerrainFile, out)
		}
		for _, rel := range drop {
			if err == nil {
				err = k.drop(rel)
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
		return fmt.Errorf("the project holds no kind of %s", rel)
	}
	k.w.Project.Files[rel] = out
	return nil
}

// drop takes the file rel names out of the project: the next save removes
// it from the folder when the folder holds it.
func (k kindFiles) drop(rel string) error {
	if !mod.HeldDraft(rel) {
		return fmt.Errorf("the project holds no kind of %s", rel)
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

// ImportTerrainStrip copies the host file source into the open mod as the
// strip of tile group g's column b, held to the loader's rule, through the
// one file import.
func (s *mapInspectionEdits) ImportTerrainStrip(g, b int, source string, replace bool) (string, *ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return "", nil, err
	}
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return "", nil, err
	}
	if source == "" {
		return "", nil, errors.New(x.word("editor.terrain.err.nofile"))
	}
	check := func(data []byte) error {
		if _, err := terrain.ReadStrip(data); err != nil {
			return fmt.Errorf(x.word("editor.terrain.err.strip"), b+1, g, err)
		}
		return nil
	}
	data, err := x.hostFileBytes(source, mod.MaxFileBytes, check, fmt.Errorf(x.word("editor.terrain.err.strip"), b+1, g, errors.New(source)))
	if err != nil {
		return "", nil, err
	}
	rel := modStripFile(g, b)
	return s.importModFile(rel, data, replace, func() (*ui.InspectionDocument, error) {
		_, d, err := s.ImportTerrainStrip(g, b, source, true)
		return d, err
	})
}
