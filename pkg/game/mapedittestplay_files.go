package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"againrom/pkg/hostfile"
	"againrom/pkg/locale"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/namefold"
	"againrom/pkg/words"
)

const testPlayRecord = ".testplay-files.json"

type testPlayFileRecord struct {
	Version int
	Files   []string
}

type missionPackagePlan struct {
	ID, Record, Source string
	Files              map[string][]byte
	Check              func(modrt.Result) error
	Expected           []mod.Mission
	Authoring          bool
	Original           map[string][32]byte
	Receipt            []byte
	InPlace, Owned     bool
	Changed            map[string][]byte
}

func testPlayRecordBytes(names []string) []byte {
	b, _ := json.Marshal(testPlayFileRecord{Version: 1, Files: names})
	return append(b, '\n')
}

// packageGuard is a folder a mission package's folder must not overlap. The
// package folder may not be dir or contain it, since writing there could
// change the author's files. It may lie inside dir unless walked: a reader of
// dir walks its subfolders and would take the package's files for dir's own.
type packageGuard struct {
	dir, name string
	walked    bool
}

func checkMissionPackageOverlap(folder, recordName string, protected []packageGuard) error {
	noun := "the mod package folder"
	if recordName == testPlayRecord {
		noun = "the test folder"
	}
	for _, g := range protected {
		inside, err := sameOrDescendantPhysical(g.dir, folder)
		if err != nil {
			return fmt.Errorf("%s: %w", folder, err)
		}
		contains, err := sameOrDescendantPhysical(folder, g.dir)
		if err != nil {
			return fmt.Errorf("%s: %w", folder, err)
		}
		relation := ""
		switch {
		case inside && contains:
			relation = "is"
		case contains:
			relation = "contains"
		case inside && g.walked:
			relation = "lies inside"
		}
		if relation != "" {
			return fmt.Errorf("%s %s %s %s %s", noun, folder, relation, g.name, g.dir)
		}
	}
	return nil
}

func missionPackageOwnership(folder, recordName string, protected []packageGuard) ([]string, error) {
	if err := checkMissionPackageOverlap(folder, recordName, protected); err != nil {
		return nil, err
	}
	st, err := os.Lstat(folder)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s is not an owned test mod folder", folder)
	}
	physical, err := editorPhysicalDirectory(folder)
	if err != nil || !sameSaveName(physical, folder) {
		return nil, fmt.Errorf("%s redirects out of the test mod", folder)
	}
	name := filepath.Join(folder, recordName)
	st, err = os.Lstat(name)
	if err != nil || !st.Mode().IsRegular() || st.Size() > 65536 {
		return nil, fmt.Errorf("%s was not written by a test play", folder)
	}
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var record testPlayFileRecord
	if err := json.Unmarshal(b, &record); err != nil || record.Version != 1 {
		return nil, fmt.Errorf("%s has an invalid test play ownership record", folder)
	}
	seen := map[string]bool{}
	for _, rel := range record.Files {
		page := strings.TrimPrefix(rel, testPlayTexts+"/")
		if lang, rest, ok := strings.Cut(page, "/"); ok && locale.Known(lang) {
			page = rest
		}
		fixed := rel == mod.ManifestName || rel == mod.EntryName || rel == mod.MissionsFile || rel == testPlayMap || rel == testPlayMapExtended
		projectPath := recordName == editorPackageRecord && rel != "." && filepath.IsLocal(filepath.FromSlash(rel)) && !strings.Contains(rel, "\\")
		if seen[rel] || !projectPath && !fixed && (page == rel || !missionPageName(page)) {
			return nil, fmt.Errorf("%s has an invalid recorded path %q", folder, rel)
		}
		seen[rel] = true
	}
	for _, rel := range []string{mod.ManifestName, mod.MissionsFile, testPlayMap} {
		if recordName == editorPackageRecord && rel != mod.ManifestName {
			continue
		}
		if !seen[rel] && !(rel == testPlayMap && seen[testPlayMapExtended]) {
			return nil, fmt.Errorf("%s has an incomplete test play ownership record", folder)
		}
	}
	return record.Files, nil
}

func testPlayTargets(folder string, old []string, files map[string][]byte) error {
	names := slices.Clone(old)
	for name := range files {
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	for _, rel := range names {
		full := filepath.Join(folder, filepath.FromSlash(rel))
		st, err := os.Lstat(full)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !st.Mode().IsRegular() || !slices.Contains(old, rel) {
			return fmt.Errorf("%s is not a recorded test play file", full)
		}
	}
	return nil
}

func stageMissionPackage(profileDir, folder, root, id, record string, old []string, files map[string][]byte, guard func(string) error, plans ...missionPackagePlan) (stage string, err error) {
	stage, err = hostfile.NewStage(profileDir, ".testplay-*")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(stage)
		}
	}()
	dest := filepath.Join(stage, id)
	if err := os.Mkdir(dest, 0o755); err != nil {
		return stage, err
	}
	source, exclude := folder, old
	var copied []string
	retained := len(plans) != 0 && plans[0].Source != ""
	if retained {
		source, exclude = plans[0].Source, slices.Sorted(maps.Keys(files))
	}
	if len(plans) != 0 && plans[0].InPlace {
		files = plans[0].Changed
		replaced := map[string]bool{}
		for rel := range files {
			replaced[namefold.ModTree(rel)] = true
		}
		tree, err := mod.OpenTree(folder)
		if err != nil {
			return stage, err
		}
		names, err := tree.Files()
		if err != nil {
			return stage, err
		}
		for _, rel := range names {
			if replaced[namefold.ModTree(rel)] {
				continue
			}
			data, err := tree.ReadFile(rel)
			if err != nil {
				return stage, err
			}
			if err := writeTestModFile(guard, profileDir, dest, rel, data, false); err != nil {
				return stage, err
			}
		}
	} else if _, err := os.Stat(source); err == nil {
		if err := copyRetainedTestMod(source, dest, exclude, record, guard, &copied); err != nil {
			return stage, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return stage, err
	}
	// A nil draft removes that file from the retained tree.
	files = maps.Clone(files)
	for rel, data := range files {
		if data == nil {
			delete(files, rel)
		}
	}
	for rel, b := range files {
		if err := writeTestModFile(guard, profileDir, dest, rel, b, false); err != nil {
			return stage, err
		}
	}
	names := slices.Sorted(maps.Keys(files))
	if retained {
		names = append(names, copied...)
		slices.Sort(names)
	}
	if len(plans) != 0 && plans[0].InPlace {
		names = slices.Clone(old)
		for rel, data := range plans[0].Changed {
			if data == nil {
				names = slices.DeleteFunc(names, func(name string) bool { return name == rel })
			} else if !slices.Contains(names, rel) {
				names = append(names, rel)
			}
		}
		if !slices.Equal(names, old) {
			slices.Sort(names)
		}
	}
	if len(plans) == 0 || !plans[0].InPlace || plans[0].Owned {
		receipt := testPlayRecordBytes(names)
		if len(plans) != 0 && plans[0].InPlace && slices.Equal(names, old) {
			receipt = plans[0].Receipt
		}
		if len(names) > 4096 || len(receipt) > 65536 {
			return stage, errors.New("mod ownership record exceeds 65536 bytes")
		}
		if err := writeTestModFile(guard, profileDir, dest, record, receipt, true); err != nil {
			return stage, err
		}
	}
	entries, err := mod.Resolve(stage, []string{id})
	if err != nil {
		return stage, err
	}
	result, err := modrt.Load(entries, BaseID(InspectInstall(root)), nil, modrt.Options{Authoring: len(plans) != 0 && plans[0].Authoring})
	if err == nil && len(plans) != 0 && plans[0].Expected != nil {
		err = checkMissionProjection(plans[0].Expected, result.Missions)
	}
	if err == nil && len(plans) != 0 && plans[0].Check != nil {
		err = plans[0].Check(result)
	}
	return stage, err
}

// installMissionPackage publishes admitted changes as sibling replacements.
// New packages and disposable test-play copies publish the staged folder.
func installMissionPackage(stage, folder, id string, owned bool, plans ...missionPackagePlan) error {
	if len(plans) != 0 && plans[0].InPlace {
		defer os.RemoveAll(stage)
		plan := plans[0]
		files := maps.Clone(plan.Changed)
		if plan.Owned {
			receipt, err := os.ReadFile(filepath.Join(stage, id, editorPackageRecord))
			if err != nil {
				return err
			}
			if !bytes.Equal(receipt, plan.Receipt) {
				files[editorPackageRecord] = receipt
			}
		}
		set, cleanup, err := editorProjectSet(folder, files, plan.Original)
		if err != nil {
			return err
		}
		defer cleanup()
		_, err = hostfile.Commit(set)
		if err == nil {
			for rel, data := range files {
				if data == nil {
					_ = os.Remove(filepath.Dir(filepath.Join(folder, filepath.FromSlash(rel))))
				}
			}
		}
		return err
	}
	row := hostfile.File{Path: folder, Folder: filepath.Join(stage, id)}
	if owned {
		row.Backup = filepath.Join(stage, "previous")
	}
	_, err := hostfile.Commit(hostfile.Set{
		Files:   []hostfile.File{row},
		Discard: stage,
		Words:   hostfile.Words{RestoreFolder: "package replacement failed: %v; old package retained at %s: %v"},
	})
	return err
}

func copyRetainedTestMod(folder, dest string, old []string, record string, guard func(string) error, written ...*[]string) error {
	return walkRetainedTestMod(folder, old, record, guard, func(rel string, data []byte, dir bool) error {
		if dir {
			return os.Mkdir(filepath.Join(dest, filepath.FromSlash(rel)), 0o755)
		}
		_, err := hostfile.Commit(hostfile.Set{Files: []hostfile.File{{
			Path: filepath.Join(dest, filepath.FromSlash(rel)), Data: data, Perm: 0o644,
			Staging: hostfile.Direct, Overwrite: hostfile.Refuse,
		}}})
		if err == nil && len(written) != 0 {
			*written[0] = append(*written[0], rel)
		}
		return err
	})
}

func walkRetainedTestMod(folder string, old []string, record string, guard func(string) error, visit func(string, []byte, bool) error) error {
	return filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(folder, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s redirects out of the test mod", path)
		}
		if d.IsDir() {
			physical, err := editorPhysicalDirectory(path)
			if err != nil || !sameSaveName(physical, path) {
				return fmt.Errorf("%s redirects out of the test mod", path)
			}
			if guard != nil {
				if err := guard(path); err != nil {
					return err
				}
			}
			return visit(filepath.ToSlash(rel), nil, true)
		}
		if rel == record || slices.Contains(old, filepath.ToSlash(rel)) {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s is not a file of the test mod", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return visit(filepath.ToSlash(rel), data, false)
	})
}

// checkModMapDirectory refuses, before any directory is created, a target
// directory whose deepest existing ancestor resolves outside the mod folder or
// inside the install at root.
func checkModMapDirectory(root, dir, target, rel string) error {
	folder, err := editorPhysicalDirectory(dir)
	if err != nil {
		return fmt.Errorf("resolve mod folder: %w", err)
	}
	lexical, err := filepath.Rel(dir, target)
	if err != nil || lexical == ".." || strings.HasPrefix(lexical, ".."+string(filepath.Separator)) || filepath.IsAbs(lexical) {
		return fmt.Errorf("%s leads outside the mod folder", rel)
	}
	existing := target
	for {
		_, err := os.Lstat(existing)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("inspect map output directory: %w", err)
		}
		next := filepath.Dir(existing)
		if next == existing {
			return fmt.Errorf("%s leads outside the mod folder", rel)
		}
		existing = next
	}
	physical, err := editorPhysicalDirectory(existing)
	if err != nil {
		return fmt.Errorf("resolve map output directory: %w", err)
	}
	inside, err := pathWithin(folder, physical)
	if err != nil {
		return fmt.Errorf("resolve map output boundary: %w", err)
	}
	if !inside {
		return fmt.Errorf("%s leads outside the mod folder", rel)
	}
	return fenceEditorWrite(root, physical, "Save")
}

func (x *MapInspector) writeEditorProject(folder, id string, old []string, files map[string][]byte, plan missionPackagePlan, replace bool) error {
	plan.Authoring = true
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	parent, err := editorPhysicalDirectory(filepath.Dir(folder))
	if err != nil {
		return err
	}
	guard, _ := admittedOnce(func(path string) error { return fenceEditorWrite(root, path, "Save mod package") })
	if err := guard(parent); err != nil {
		return err
	}
	if _, err := os.Lstat(folder); err == nil {
		for rel := range files {
			full := filepath.Join(folder, filepath.FromSlash(rel))
			if err := checkModMapDirectory(root, folder, filepath.Dir(full), rel); err != nil {
				return err
			}
		}
		plan.InPlace = true
		plan.Owned = len(old) != 0
		if x.mod != nil && sameSaveName(x.mod.dir, folder) {
			plan.Receipt = x.mod.ownershipRecord
			if x.archives != nil {
				plan.Original = x.mod.snapshot
			}
		}
		plan.Changed, err = changedProjectFiles(folder, files, plan.Original)
		if err != nil {
			return err
		}
		if _, changed := plan.Changed[mod.MissionsFile]; !changed {
			plan.Expected = nil
		}
	}
	stage, err := stageMissionPackage(parent, folder, root, id, editorPackageRecord, old, files, guard, plan)
	if err != nil {
		return err
	}
	return installMissionPackage(stage, folder, id, replace, plan)
}

func writeMapInMod(root, target string, data []byte, pages map[string][]byte, labels []byte, overwrite, withLabels bool) (bool, error) {
	for folder := filepath.Dir(target); ; folder = filepath.Dir(folder) {
		if _, err := os.Lstat(filepath.Join(folder, mod.ManifestName)); err == nil {
			x := &MapInspector{archives: &Archives{Root: root}}
			if err := x.SetMod(folder); err != nil {
				return true, err
			}
			if _, err := os.Lstat(target); !overwrite && !errors.Is(err, os.ErrNotExist) {
				return true, fmt.Errorf("create new map: %w", os.ErrExist)
			}
			rel, err := filepath.Rel(folder, target)
			if err != nil {
				return true, err
			}
			files := map[string][]byte{filepath.ToSlash(rel): data}
			for name, b := range pages {
				if !pageTreeName(name) || len(b) > mod.MaxFileBytes {
					return true, fmt.Errorf("invalid map page %q", name)
				}
				path, err := filepath.Rel(folder, filepath.Join(mapTextFolder(target), filepath.FromSlash(name)))
				if err != nil {
					return true, err
				}
				files[filepath.ToSlash(path)] = b
			}
			if withLabels && (labels == nil || len(labels) != 0) {
				path, err := filepath.Rel(folder, mod.MapLabelFile(target))
				if err != nil {
					return true, err
				}
				if labels != nil || overwrite {
					files[filepath.ToSlash(path)] = labels
				}
			}
			if overwrite {
				for name := range files {
					key := namefold.ModTree(name)
					if _, known := x.mod.snapshot[key]; !known {
						data, err := os.ReadFile(filepath.Join(folder, filepath.FromSlash(name)))
						if err == nil {
							x.mod.snapshot[key] = sha256.Sum256(data)
						} else if !errors.Is(err, os.ErrNotExist) {
							return true, err
						}
					}
				}
			}
			var old []string
			if x.mod.owned {
				old, err = missionPackageOwnership(folder, editorPackageRecord, nil)
				if err != nil {
					return true, err
				}
			}
			return true, x.writeEditorProject(folder, x.mod.id, old, files, missionPackagePlan{Source: folder}, true)
		}
		if next := filepath.Dir(folder); next == folder {
			return false, nil
		}
	}
}

func changedProjectFiles(folder string, files map[string][]byte, baseline map[string][32]byte) (map[string][]byte, error) {
	changed := map[string][]byte{}
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		data := files[rel]
		prior, known := baseline[namefold.ModTree(rel)]
		if baseline != nil && (known && data != nil && sha256.Sum256(data) == prior || !known && data == nil) {
			continue
		}
		path := filepath.Join(folder, filepath.FromSlash(rel))
		if st, err := os.Lstat(path); err == nil && !st.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a file", rel)
		}
		current, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if bytes.Equal(current, data) && (err == nil) == (data != nil) {
			continue
		}
		if baseline != nil && (known && sha256.Sum256(current) != prior || (err == nil) != known) {
			return nil, fmt.Errorf("%s changed outside the editor; reopen the mod before saving", rel)
		}
		changed[rel] = data
	}
	return changed, nil
}

func editorProjectSet(folder string, files map[string][]byte, snapshot map[string][32]byte) (hostfile.Set, func(), error) {
	set := hostfile.Set{Temp: ".mod-save-", Sync: true, Words: hostfile.Words{Restore: "restore %s from %s: %v"}}
	files, err := changedProjectFiles(folder, files, snapshot)
	if err != nil {
		return set, func() {}, err
	}
	var created []string
	cleanup := func() {
		for i := len(created) - 1; i >= 0; i-- {
			_ = os.Remove(created[i])
		}
	}
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		path := filepath.Join(folder, filepath.FromSlash(rel))
		var missing []string
		for at := filepath.Dir(path); !sameSaveName(at, folder); at = filepath.Dir(at) {
			if _, err := os.Lstat(at); err == nil {
				break
			} else if !errors.Is(err, os.ErrNotExist) {
				cleanup()
				return set, func() {}, err
			}
			missing = append(missing, at)
		}
		for i := len(missing) - 1; i >= 0; i-- {
			if err := os.Mkdir(missing[i], 0o755); err != nil {
				cleanup()
				return set, func() {}, err
			}
			created = append(created, missing[i])
		}
		prior, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanup()
			return set, func() {}, err
		}
		row := hostfile.File{Path: path, Data: files[rel], Prior: prior, Perm: 0o644, Seen: hostfile.SeenAbsent, Overwrite: hostfile.Backup, Remove: files[rel] == nil}
		if err == nil {
			row.Seen = hostfile.SeenPresent
			if st, err := os.Lstat(path); err == nil {
				row.Perm = st.Mode().Perm()
			}
		}
		set.Files = append(set.Files, row)
	}
	set.Check = func(step hostfile.Step, i int) error {
		row := set.Files[i]
		if err := checkModMapDirectory("", folder, filepath.Dir(row.Path), row.Path); err != nil {
			return err
		}
		if step == hostfile.StepRestore {
			return nil
		}
		current, err := os.ReadFile(row.Path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if !bytes.Equal(current, row.Prior) || (err == nil) != (row.Seen == hostfile.SeenPresent) {
			return fmt.Errorf("%s changed outside the editor; reopen the mod before saving", row.Path)
		}
		if snapshot != nil {
			rel, err := filepath.Rel(folder, row.Path)
			if err != nil {
				return err
			}
			stamp, known := snapshot[namefold.ModTree(filepath.ToSlash(rel))]
			if (row.Seen == hostfile.SeenPresent) != known || known && sha256.Sum256(row.Prior) != stamp {
				return fmt.Errorf("%s changed outside the editor; reopen the mod before saving", row.Path)
			}
		}
		return nil
	}
	return set, cleanup, nil
}

// writeEditedMap writes a host file, never an input or a preserved install,
// even through an alias.
func writeEditedMap(root, source, path string, data []byte, overwrite bool, grants ...*runtimeProfileAccess) (string, error) {
	if err := extendedName(data, path); err != nil {
		return "", err
	}
	parent, target, err := editedMapTarget(root, source, path, overwrite, grants...)
	if err != nil {
		return "", err
	}
	if handled, err := writeMapInMod(root, target, data, nil, nil, overwrite, false); handled {
		if err != nil {
			return "", err
		}
		return target, nil
	}
	set, err := editedMapSet(parent, target, data, overwrite)
	if err != nil {
		return "", err
	}
	if _, err := hostfile.Commit(set); err != nil {
		return "", err
	}
	return target, nil
}

// editedMapSet is the Set that writes data at target in folder parent. A new
// map is created exclusively and synchronized; a failed write removes it.
// With overwrite the bytes go to a temporary file beside target, keep the
// mode of the file they replace and are renamed over it, so a crash leaves
// the old or the new file whole. An existing target must be an ordinary
// file, not a link or a directory.
func editedMapSet(parent, target string, data []byte, overwrite bool) (hostfile.Set, error) {
	if !overwrite {
		return hostfile.Set{
			Files: []hostfile.File{{Path: target, Data: data, Perm: 0600, Staging: hostfile.Direct, Overwrite: hostfile.Refuse}},
			Sync:  true,
			Words: hostfile.Words{Create: "create new map: %w"},
		}, nil
	}
	perm := os.FileMode(0600)
	if fi, err := os.Lstat(target); err == nil {
		if !fi.Mode().IsRegular() {
			return hostfile.Set{}, fmt.Errorf("Save cannot replace %s: not an ordinary file", filepath.Base(target))
		}
		perm = fi.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return hostfile.Set{}, err
	}
	return hostfile.Set{
		Files: []hostfile.File{{Path: target, Data: data, Perm: perm, Overwrite: hostfile.Replace}},
		Dir:   parent,
		Temp:  ".map-save-",
		Sync:  true,
		Words: hostfile.Words{Create: "create temporary map: %w"},
	}, nil
}

// replaceMapFile writes data over target through its replacing Set.
func replaceMapFile(parent, target string, data []byte) error {
	set, err := editedMapSet(parent, target, data, true)
	if err != nil {
		return err
	}
	_, err = hostfile.Commit(set)
	return err
}

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
		_, target, err := editedMapTarget(root, source, path, overwrite, grants...)
		if err != nil {
			return "", err
		}
		if handled, err := writeMapInMod(root, target, data, pages, labels, overwrite, true); handled {
			if err != nil {
				return "", err
			}
			return target, nil
		}
		target, err = writeEditedMap(root, source, path, data, overwrite, grants...)
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
	if handled, err := writeMapInMod(root, target, data, pages, labels, overwrite, true); handled {
		if err != nil {
			return "", err
		}
		return target, nil
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

// writeTestModFile writes data to the slash path rel in folder, creating the
// directories below top one by one, each checked by guard, and replacing the
// file through a dot-named file moved over it. Only a durable file is flushed
// before the move: the ownership record, written last. Every other file is
// rewritten by the next test play or lives in a staging folder removed after
// validation.
func writeTestModFile(guard func(string) error, top, folder, rel string, data []byte, durable bool) error {
	full := filepath.Join(folder, filepath.FromSlash(rel))
	lexical, err := filepath.Rel(top, filepath.Dir(full))
	if err != nil || !filepath.IsLocal(lexical) && lexical != "." {
		return fmt.Errorf("%s leads outside the test mod", rel)
	}
	dir := top
	if err := guard(dir); err != nil {
		return err
	}
	if lexical != "." {
		for _, part := range strings.Split(lexical, string(filepath.Separator)) {
			dir = filepath.Join(dir, part)
			st, err := os.Lstat(dir)
			if errors.Is(err, os.ErrNotExist) {
				err = os.Mkdir(dir, 0o755)
				if err == nil {
					st, err = os.Lstat(dir)
				}
			}
			if err != nil {
				return err
			}
			if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%s is not a folder of the test mod", dir)
			}
			physical, err := editorPhysicalDirectory(dir)
			if err != nil || !sameSaveName(physical, dir) {
				return fmt.Errorf("%s redirects out of the test mod", dir)
			}
			if err := guard(dir); err != nil {
				return err
			}
		}
	}
	target := filepath.Join(dir, filepath.Base(full))
	if st, err := os.Lstat(target); err == nil && !st.Mode().IsRegular() {
		return fmt.Errorf("%s is not a file", target)
	}
	_, err = hostfile.Commit(hostfile.Set{
		Files: []hostfile.File{{Path: target, Data: data, Overwrite: hostfile.Replace}},
		Dir:   dir,
		Temp:  ".testplay-*",
		Sync:  durable,
	})
	return err
}
