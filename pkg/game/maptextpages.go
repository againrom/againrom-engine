package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/mod"
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
				if tail := filepath.Base(root); tail == "en" || tail == "ru" {
					root = filepath.ToSlash(filepath.Dir(root))
				}
				if err := readPageTree(x.mod.dir, root, files, false, x.pageLanguage()); err != nil {
					return nil, err
				}
			}
		}
	}
	if !isNewSource(s.source) && !strings.HasPrefix(s.source, "scenario/") {
		source := s.source
		if strings.HasPrefix(source, "loose/") && x.archives != nil {
			source = looseHostPath(x.archives.Root, source)
		}
		if err := readPageTree(filepath.Dir(source), filepath.Base(mapTextFolder(source)), files, true, x.pageLanguage()); err != nil {
			return nil, err
		}
	}
	var saved editorWorkspaceState
	_ = json.Unmarshal(s.savedWorkspace, &saved)
	lang := x.pageLanguage()
	for name, b := range s.workspace.Pages {
		_, localized := files[lang+"/"+name]
		_, common := files[name]
		if before, exists := saved.Pages[name]; !exists || !bytes.Equal(before, b) || (!localized && !common) {
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
		return "en"
	}
	return words.Language(x.archives.Base.Profile.Language)
}

func mapTextFolder(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ".text"
}

func (x *MapInspector) editorPages(source string) *modPages {
	p := &modPages{id: filepath.Base(source), files: map[string][]byte{}}
	n := sourceMission(source)
	if x.mod != nil {
		if entry, ok := x.mod.entryFor(source); ok {
			p.id, p.entry = x.mod.id, entry
			if entry.Kind == mod.MissionReplace {
				n = entry.Number
			}
		}
	}
	if n > 0 {
		if p.entry.Kind == "" {
			p.entry = mod.Mission{Kind: mod.MissionReplace, Number: n}
		}
		if x.archives != nil && x.archives.Containers != nil {
			p.files, p.err = installedPages(x.archives.Containers, n)
			if p.err != nil {
				return p
			}
		}
	}
	merge := func(files map[string][]byte) {
		for name, b := range files {
			if missionPageName(name) {
				p.files[name] = b
			}
		}
	}
	if x.mod != nil {
		if entry, ok := x.mod.entryFor(source); ok {
			p.id, p.entry = x.mod.id, entry
			files, err := entry.ReadTexts(x.mod.dir, x.pageLanguage())
			if err != nil {
				p.err = err
				return p
			}
			merge(files)
		}
	}
	if !strings.HasPrefix(source, "scenario/") && !isNewSource(source) {
		path := source
		if strings.HasPrefix(source, "loose/") && x.archives != nil {
			path = looseHostPath(x.archives.Root, source)
		}
		m := mod.Mission{Map: filepath.Base(path)}
		files, err := m.ReadTexts(filepath.Dir(path), x.pageLanguage())
		if err != nil {
			p.err = err
			return p
		}
		merge(files)
	}
	if len(p.files) > 0 {
		p.entry.Texts = testPlayTexts
	}
	return p
}

func writeEditedMapPages(root, source, path string, data []byte, pages map[string][]byte, lang string, overwrite bool, grants ...*runtimeProfileAccess) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent, err := editorPhysicalDirectory(filepath.Dir(abs))
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
		return writeEditedMap(root, source, path, data, overwrite, grants...)
	}
	if lang != "en" && lang != "ru" {
		return "", fmt.Errorf("unsupported page language %q", lang)
	}
	stage, err := os.MkdirTemp(parent, ".map-pages-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	for name, b := range pages {
		if !pageTreeName(name) || len(b) > mod.MaxFileBytes {
			return "", fmt.Errorf(words.For(lang).Text("editor.pages.invalid"), name)
		}
		target := filepath.Join(stage, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return "", err
		}
		if err := os.WriteFile(target, b, 0600); err != nil {
			return "", err
		}
	}
	target, err := writeEditedMap(root, source, path, data, overwrite, grants...)
	if err != nil {
		return "", err
	}
	if overwrite && existing != nil {
		old := folder + ".old"
		_ = os.RemoveAll(old)
		if err := os.Rename(folder, old); err != nil {
			return "", err
		}
		if err := os.Rename(stage, folder); err != nil {
			_ = os.Rename(old, folder)
			return "", err
		}
		_ = os.RemoveAll(old)
		return target, nil
	}
	if err := os.Rename(stage, folder); err != nil {
		if !overwrite {
			_ = os.Remove(target)
		}
		return "", err
	}
	return target, nil
}
