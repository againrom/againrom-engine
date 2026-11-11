package game

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"againrom/pkg/hostfile"
	"againrom/pkg/locale"
	"againrom/pkg/mod"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

func (s *mapInspectionEdits) exportPageTree(folder string, recorded []string) (map[string][]byte, error) {
	files := map[string][]byte{}
	readRecorded := func(dir string, names []string) error {
		for _, name := range names {
			if !strings.HasPrefix(name, testPlayTexts+"/") {
				continue
			}
			rel := strings.TrimPrefix(name, testPlayTexts+"/")
			if !pageTreeName(rel) {
				continue
			}
			b, _, err := mod.ReadInside(dir, name)
			if err != nil {
				return err
			}
			files[rel] = b
		}
		return nil
	}
	if err := readRecorded(folder, recorded); err != nil {
		return nil, err
	}
	x := s.inspector
	if x.mod != nil {
		if entry, ok := x.mod.entryFor(s.source); ok {
			if x.mod.owned {
				names, err := missionPackageOwnership(x.mod.dir, editorPackageRecord, nil)
				if err != nil {
					return nil, err
				}
				if err := readRecorded(x.mod.dir, names); err != nil {
					return nil, err
				}
			} else if entry.Texts != "" {
				root := entry.Texts
				if tail := filepath.Base(root); locale.Known(tail) {
					root = filepath.ToSlash(filepath.Dir(root))
				}
				if err := readPageTree(x.mod.dir, root, files, false, x.pageLanguage()); err != nil {
					return nil, err
				}
			}
		}
	}
	if source, ok := mapHostPath(ui.ParseMapSource(s.source), x.archives); ok {
		if err := readPageTree(filepath.Dir(source), filepath.Base(mapTextFolder(source)), files, true, x.pageLanguage()); err != nil {
			return nil, err
		}
	}
	var saved editorWorkspaceState
	_ = json.Unmarshal(s.savedWorkspace, &saved)
	lang := x.pageLanguage()
	otherLang := locale.Other(lang)
	for name, b := range s.workspace.Pages {
		_, localized := files[lang+"/"+name]
		_, common := files[name]
		// An unchanged page from the other language's folder stays there.
		other, fromOther := files[otherLang+"/"+name]
		fromOther = fromOther && bytes.Equal(other, b)
		if before, exists := saved.Pages[name]; !exists || !bytes.Equal(before, b) || (!localized && !common && !fromOther) {
			files[lang+"/"+name] = bytes.Clone(b)
		}
	}
	for name, b := range files {
		if !pageTreeName(name) || len(b) > mod.MaxFileBytes {
			return nil, fmt.Errorf(words.For(lang).Text("editor.pages.invalid"), name)
		}
	}
	return files, nil
}

func pageTreeName(name string) bool {
	parts := strings.Split(name, "/")
	return (len(parts) == 1 || len(parts) == 2 && editorOrdinaryFilename(parts[0])) && missionPageName(parts[len(parts)-1]) && editorOrdinaryFilename(parts[len(parts)-1])
}

func readPageTree(dir, root string, files map[string][]byte, strict bool, lang string) error {
	read := func(relative, prefix string) ([]os.DirEntry, error) {
		entries, err := os.ReadDir(filepath.Join(dir, filepath.FromSlash(relative)))
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		count := 0
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := prefix + e.Name()
			if !pageTreeName(name) {
				if !strict {
					continue
				}
				return nil, fmt.Errorf(words.For(lang).Text("editor.pages.invalid"), filepath.Join(dir, filepath.FromSlash(relative), e.Name()))
			}
			count++
			if count > mod.MaxMissionTexts {
				return nil, fmt.Errorf("%s: at most %d pages", relative, mod.MaxMissionTexts)
			}
			b, _, err := mod.ReadInside(dir, relative+"/"+e.Name())
			if err != nil {
				return nil, err
			}
			files[name] = b
		}
		return entries, nil
	}
	entries, err := read(root, "")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() && editorOrdinaryFilename(e.Name()) {
			if _, err := read(root+"/"+e.Name(), e.Name()+"/"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (x *MapInspector) pageLanguage() string {
	if x.archives == nil {
		return locale.Fallback
	}
	return words.Language(x.archives.Base.Profile.Language)
}

func mapTextFolder(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ".text"
}

// editorPages is the pages the editor shows for the map at source: the
// install's pages of its mission, then the mod entry's, then the map's own
// folder, resolved by mod.ResolvePages.
func (x *MapInspector) editorPages(source string) *mod.MissionPages {
	p := &mod.MissionPages{ID: filepath.Base(source)}
	lang := x.pageLanguage()
	var layers []mod.PageLayer
	n := sourceMission(source)
	if x.mod != nil {
		if entry, ok := x.mod.entryFor(source); ok {
			p.ID, p.Entry = x.mod.id, entry
			if entry.Kind == mod.MissionReplace {
				n = entry.Number
			}
		}
	}
	if n > 0 {
		if p.Entry.Kind == "" {
			p.Entry = mod.Mission{Kind: mod.MissionReplace, Number: n}
		}
		if x.archives != nil && x.archives.Containers != nil {
			files, err := x.installKeep.pages(x.archives.Containers, n)
			if err != nil {
				p.Err = err
				p.Pages = mod.ResolvePages(missionPageName)
				return p
			}
			layers = append(layers, mod.PageLayer{Source: mod.PageInstall, Lang: lang, Files: files})
		}
	}
	if x.mod != nil {
		if entry, ok := x.mod.entryFor(source); ok {
			p.ID, p.Entry = x.mod.id, entry
			layer, err := entry.ReadPageLayer(mod.PageMod, x.mod.dir, lang)
			if err != nil {
				p.Err = err
				p.Pages = mod.ResolvePages(missionPageName, layers...)
				return p
			}
			layers = append(layers, layer)
		}
	}
	if path, ok := mapHostPath(ui.ParseMapSource(source), x.archives); ok {
		layer, err := mod.Mission{Map: filepath.Base(path)}.ReadPageLayer(mod.PageMap, filepath.Dir(path), lang)
		if err != nil {
			p.Err = err
			p.Pages = mod.ResolvePages(missionPageName, layers...)
			return p
		}
		layers = append(layers, layer)
	}
	p.Pages = mod.ResolvePages(missionPageName, layers...)
	if len(p.Pages) > 0 {
		p.Entry.Texts = testPlayTexts
	}
	return p
}

// mapLabelsFile is the labels file: one entry per labelled register, in
// register order.
type mapLabelsFile struct {
	Variables []mapLabelEntry `json:"variables"`
}

type mapLabelEntry struct {
	Register uint32 `json:"register"`
	Label    string `json:"label"`
}

const (
	maxLabelFileBytes = 64 << 10
	maxLabelRunes     = 64
)

// labelAllowed says whether text can be a variable label: 1 to maxLabelRunes
// characters of UTF-8 with no control character and not only spaces.
func labelAllowed(text string) bool {
	if !utf8.ValidString(text) || strings.TrimSpace(text) == "" || utf8.RuneCountInString(text) > maxLabelRunes {
		return false
	}
	return !strings.ContainsFunc(text, unicode.IsControl)
}

// mapLabels reads the labels file beside the map source names. No file is no
// labels; a file that is not a labels file is an error and no labels.
func (x *MapInspector) mapLabels(source string) (map[uint32]string, error) {
	path, ok := mapHostPath(ui.ParseMapSource(source), x.archives)
	if !ok {
		return nil, nil
	}
	name := mod.MapLabelFile(path)
	fi, err := os.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() || fi.Size() > maxLabelFileBytes {
		return nil, fmt.Errorf("%s is not an ordinary file of at most %d bytes", name, maxLabelFileBytes)
	}
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var f mapLabelsFile
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: data after the labels", name)
	}
	out := map[uint32]string{}
	for _, v := range f.Variables {
		// The bound is the extended array's; liveLabels keeps the labels
		// the map's form holds.
		if _, twice := out[v.Register]; twice || v.Register >= scriptprofile.ExtendedVariables || !labelAllowed(v.Label) {
			return nil, fmt.Errorf("%s: register %d", name, v.Register)
		}
		out[v.Register] = v.Label
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// encodeMapLabels is the labels file of labels; nil when there are none.
func encodeMapLabels(labels map[uint32]string) []byte {
	if len(labels) == 0 {
		return nil
	}
	var f mapLabelsFile
	for r, label := range labels {
		f.Variables = append(f.Variables, mapLabelEntry{Register: r, Label: label})
	}
	slices.SortFunc(f.Variables, func(a, b mapLabelEntry) int { return cmp.Compare(a.Register, b.Register) })
	b, _ := json.MarshalIndent(f, "", "  ")
	return append(b, '\n')
}

// labelsUnwritten is a save whose map is written and whose labels file is
// not.
type labelsUnwritten struct{ err error }

func (e labelsUnwritten) Error() string { return e.err.Error() }
func (e labelsUnwritten) Unwrap() error { return e.err }

// placeMapLabels writes labels beside the written map target, or removes the
// map's old labels file when labels is nil. Empty, not nil, labels leave the
// file as it is: one the open could not read. A new map does not take over a
// file already there.
func placeMapLabels(target string, labels []byte, overwrite bool, lang string) error {
	if labels != nil && len(labels) == 0 {
		return nil
	}
	path := mod.MapLabelFile(target)
	fi, err := os.Lstat(path)
	switch {
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return labelsUnwritten{err}
	case labels == nil && (err != nil || !overwrite || !fi.Mode().IsRegular()):
		return nil // nothing to write and no own file to remove
	case err == nil && (!overwrite || !fi.Mode().IsRegular()):
		return labelsUnwritten{fmt.Errorf(words.For(lang).Text("editor.labels.inway"), path)}
	case labels == nil:
		if err := os.Remove(path); err != nil {
			return labelsUnwritten{err}
		}
		return nil
	}
	if err := replaceMapFile(filepath.Dir(path), path, labels); err != nil {
		return labelsUnwritten{err}
	}
	return nil
}

// writeEditedMapPages writes the map and what the editor keeps beside it: the
// texts folder of pages and the labels file of labels.
func writeEditedMapPages(root, source, path string, data []byte, pages map[string][]byte, labels []byte, lang string, overwrite bool, grants ...*runtimeProfileAccess) (string, error) {
	if err := extendedName(data, path); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent, err := mapOutputDirectory(abs)
	if err != nil {
		return "", err
	}
	if err := fenceEditorWrite(root, parent, "Save map pages", grants...); err != nil {
		return "", err
	}
	folder := mapTextFolder(filepath.Join(parent, filepath.Base(abs)))
	existing, err := os.Lstat(folder)
	if overwrite {
		// The folder beside the map is the map's own; anything there other
		// than a plain directory is not rewritten.
		if len(pages) > 0 && (err == nil && !existing.IsDir() || err != nil && !errors.Is(err, os.ErrNotExist)) {
			return "", fmt.Errorf(words.For(lang).Text("editor.pages.new-folder"), folder)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf(words.For(lang).Text("editor.pages.new-folder"), folder)
	}
	if len(pages) == 0 {
		target, err := writeEditedMap(root, source, path, data, overwrite, grants...)
		if err != nil {
			return "", err
		}
		return target, placeMapLabels(target, labels, overwrite, lang)
	}
	if !locale.Known(lang) {
		return "", fmt.Errorf("unsupported page language %q", lang)
	}
	stage, err := hostfile.NewStage(parent, ".map-pages-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	for name, b := range pages {
		if !pageTreeName(name) || len(b) > mod.MaxFileBytes {
			return "", fmt.Errorf(words.For(lang).Text("editor.pages.invalid"), name)
		}
		if _, err := hostfile.Commit(hostfile.Set{
			Files: []hostfile.File{{
				Path: filepath.Join(stage, filepath.FromSlash(name)), Data: b, Perm: 0600,
				Staging: hostfile.Direct, Overwrite: hostfile.Replace,
			}},
			Dirs: hostfile.Parents, DirPerm: 0700,
		}); err != nil {
			return "", err
		}
	}
	mapParent, target, err := editedMapTarget(root, source, path, overwrite, grants...)
	if err != nil {
		return "", err
	}
	// One Set writes the map, then renames the staged folder into place. An
	// existing folder waits at its backup until the swap is done and returns
	// when it fails. A failed swap removes a new map; a replaced map stays.
	set, err := editedMapSet(mapParent, target, data, overwrite)
	if err != nil {
		return "", err
	}
	pagesRow := hostfile.File{Path: folder, Folder: stage}
	if overwrite && existing != nil {
		pagesRow.Backup = folder + ".old"
		pagesRow.Stale = true
	}
	set.Files = append(set.Files, pagesRow)
	if _, err := hostfile.Commit(set); err != nil {
		return "", err
	}
	return target, placeMapLabels(target, labels, overwrite, lang)
}
