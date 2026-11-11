package game

import (
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

func missionPackageOwnership(folder, recordName string, protected []packageGuard) ([]string, error) {
	noun := "the mod package folder"
	if recordName == testPlayRecord {
		noun = "the test folder"
	}
	for _, g := range protected {
		inside, err := sameOrDescendantPhysical(g.dir, folder)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", folder, err)
		}
		contains, err := sameOrDescendantPhysical(folder, g.dir)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", folder, err)
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
			return nil, fmt.Errorf("%s %s %s %s %s", noun, folder, relation, g.name, g.dir)
		}
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
		fixed := rel == mod.ManifestName || rel == "main.star" || rel == mod.MissionsFile || rel == testPlayMap || rel == testPlayMapExtended
		projectPath := recordName == editorPackageRecord && rel != "." && filepath.IsLocal(filepath.FromSlash(rel)) && !strings.Contains(rel, "\\")
		if seen[rel] || !projectPath && !fixed && (page == rel || !missionPageName(page)) {
			return nil, fmt.Errorf("%s has an invalid recorded path %q", folder, rel)
		}
		seen[rel] = true
	}
	for _, rel := range []string{mod.ManifestName, "main.star", mod.MissionsFile, testPlayMap} {
		if recordName == editorPackageRecord && (rel == "main.star" || rel == testPlayMap) {
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
	retained := len(plans) != 0 && plans[0].Source != ""
	if retained {
		source, exclude = plans[0].Source, slices.Sorted(maps.Keys(files))
	}
	if _, err := os.Stat(source); err == nil {
		if err := copyRetainedTestMod(source, dest, exclude, record, guard); err != nil {
			return stage, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return stage, err
	}
	for rel, b := range files {
		if err := writeTestModFile(guard, profileDir, dest, rel, b, false); err != nil {
			return stage, err
		}
	}
	names := slices.Sorted(maps.Keys(files))
	if retained {
		names = nil
		err = filepath.WalkDir(dest, func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			rel, e := filepath.Rel(dest, p)
			if e != nil {
				return e
			}
			if filepath.ToSlash(rel) != record {
				names = append(names, filepath.ToSlash(rel))
			}
			if len(names) > 4096 {
				return errors.New("mod ownership record holds at most 4096 files")
			}
			return nil
		})
		if err != nil {
			return stage, err
		}
		slices.Sort(names)
	}
	receipt := testPlayRecordBytes(names)
	if len(receipt) > 65536 {
		return stage, errors.New("mod ownership record exceeds 65536 bytes")
	}
	if err := writeTestModFile(guard, profileDir, dest, record, receipt, true); err != nil {
		return stage, err
	}
	entries, err := mod.Resolve(stage, []string{id})
	if err != nil {
		return stage, err
	}
	result, err := modrt.Load(entries, BaseID(InspectInstall(root)), nil, modrt.Options{})
	if err == nil && len(plans) != 0 && plans[0].Expected != nil {
		err = checkMissionProjection(plans[0].Expected, result.Missions)
	}
	if err == nil && len(plans) != 0 && plans[0].Check != nil {
		err = plans[0].Check(result)
	}
	return stage, err
}

// installMissionPackage renames the staged package stage/id to folder. An
// owned folder there waits in the stage until the swap is done and returns
// when it fails; the stage is removed unless it still holds that folder. The
// waiting place is inside the stage, so it is not cleared first: a package
// whose ID is previous fails on the rename, and the old package stays.
func installMissionPackage(stage, folder, id string, owned bool) error {
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

func copyRetainedTestMod(folder, dest string, old []string, record string, guard func(string) error) error {
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
			if err := guard(path); err != nil {
				return err
			}
			return os.Mkdir(filepath.Join(dest, rel), 0o755)
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
		_, err = hostfile.Commit(hostfile.Set{Files: []hostfile.File{{
			Path: filepath.Join(dest, rel), Data: data, Perm: 0o644,
			Staging: hostfile.Direct, Overwrite: hostfile.Refuse,
		}}})
		return err
	})
}
