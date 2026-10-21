package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
// failure.
func PackMod(folder, outDir string, replace bool) (string, error) {
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
	} else if inside {
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
		return target, createPackedFile(target, data)
	}
	tmp, err := os.CreateTemp(outDir, "."+mod.PackedName(m.ID)+".*")
	if err != nil {
		return "", err
	}
	_, werr := tmp.Write(data)
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), target)
	}
	if werr != nil {
		os.Remove(tmp.Name())
		return "", werr
	}
	return target, nil
}

// createPackedFile creates path with data; an existing file is refused, and a
// failed write removes what it made.
func createPackedFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("%s exists: %w", path, ErrPackExists)
	}
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		os.Remove(path)
	}
	return werr
}
