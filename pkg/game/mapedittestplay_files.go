package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

const testPlayRecord = ".testplay-files.json"

type testPlayFileRecord struct {
	Version int
	Files   []string
}

func testPlayRecordBytes(names []string) []byte {
	b, _ := json.Marshal(testPlayFileRecord{Version: 1, Files: names})
	return append(b, '\n')
}

func testPlayOwnership(folder string, protected []string) ([]string, error) {
	for _, dir := range protected {
		for _, pair := range [][2]string{{dir, folder}, {folder, dir}} {
			inside, err := sameOrDescendantPhysical(pair[0], pair[1])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", folder, err)
			}
			if inside {
				return nil, fmt.Errorf("%s overlaps the opened map or selected mod folder %s", folder, dir)
			}
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
	name := filepath.Join(folder, testPlayRecord)
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
		fixed := rel == mod.ManifestName || rel == "main.star" || rel == mod.MissionsFile || rel == testPlayMap
		if seen[rel] || !fixed && (page == rel || !assembledText.MatchString(page)) {
			return nil, fmt.Errorf("%s has an invalid recorded path %q", folder, rel)
		}
		seen[rel] = true
	}
	for _, rel := range []string{mod.ManifestName, "main.star", mod.MissionsFile, testPlayMap} {
		if !seen[rel] {
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

func validateTestMod(profileDir, folder, root string, old []string, files map[string][]byte, guard func(string) error) error {
	stage, err := os.MkdirTemp(profileDir, ".testplay-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	dest := filepath.Join(stage, TestPlayModID)
	if err := os.Mkdir(dest, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(folder); err == nil {
		if err := copyRetainedTestMod(folder, dest, old, guard); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for rel, b := range files {
		if err := writeTestModFile(guard, profileDir, dest, rel, b); err != nil {
			return err
		}
	}
	entries, err := mod.Resolve(stage, []string{TestPlayModID})
	if err != nil {
		return err
	}
	_, err = modrt.Load(entries, BaseID(InspectInstall(root)), nil, modrt.Options{})
	return err
}

func copyRetainedTestMod(folder, dest string, old []string, guard func(string) error) error {
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
		if rel == testPlayRecord || slices.Contains(old, filepath.ToSlash(rel)) {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s is not a file of the test mod", path)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(filepath.Join(dest, rel), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		return errors.Join(err, out.Close())
	})
}
