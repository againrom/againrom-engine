package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"againrom/pkg/hostfile"
	"againrom/pkg/mod"
)

// The refusals of PackMod.
var (
	ErrPackInstall = errors.New("a packed mod is never written into an install")
	ErrPackExists  = errors.New("a packed mod is written over an existing file only when asked")
)

// PackMod packs the mod folder folder into outDir/mod-<id>.res and returns
// that path. outDir is an existing folder below no folder that holds an
// install's archives. An existing file there is refused unless replace is
// set; then the new file takes its place whole. Nothing is left behind on a
// failure. A folder one of grants admits is written.
func PackMod(folder, outDir string, replace bool, grants ...*runtimeProfileAccess) (string, error) {
	st, err := os.Stat(outDir)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%s is not a folder", outDir)
	}
	resolved, err := resolvePathForSaveSafety(outDir)
	if err != nil {
		return "", err
	}
	if install, inside, err := requiredArchiveAncestor(resolved); err != nil {
		return "", err
	} else if inside && !admittedByGrant(outDir, grants) {
		return "", fmt.Errorf("%s is inside the install %s: %w", outDir, install, ErrPackInstall)
	}
	m, data, err := mod.Pack(folder)
	if err != nil {
		return "", err
	}
	target := filepath.Join(outDir, mod.PackedName(m.ID))
	if st, err := os.Lstat(target); err == nil {
		if !replace {
			return "", fmt.Errorf("%s exists: %w", target, ErrPackExists)
		}
		if !st.Mode().IsRegular() {
			return "", fmt.Errorf("%s is not a file", target)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if !replace {
		// A new file is created exclusively; a failed write removes it.
		_, err := hostfile.Commit(hostfile.Set{Files: []hostfile.File{{
			Path: target, Data: data, Perm: 0o644, Staging: hostfile.Direct, Overwrite: hostfile.Refuse,
		}}})
		if errors.Is(err, os.ErrExist) {
			return target, fmt.Errorf("%s exists: %w", target, ErrPackExists)
		}
		return target, err
	}
	// A replacement is staged beside the target and renamed over it whole.
	if _, err := hostfile.Commit(hostfile.Set{
		Files: []hostfile.File{{Path: target, Data: data, Overwrite: hostfile.Replace}},
		Dir:   outDir,
		Temp:  "." + mod.PackedName(m.ID) + ".*",
	}); err != nil {
		return "", err
	}
	return target, nil
}
