package hostfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"againrom/pkg/installpath"
	"againrom/pkg/namefold"
)

// Fence is the preserved-install rule of every host-file write: a target in
// an original directory, or below a directory holding all of Archives, is
// refused. Archives are the file names that mark a game install.
type Fence struct {
	Archives []string
}

// ErrWriteUnderInstall is the start of every install-fence refusal, so a
// caller can tell a refusal by the fence from a failure to inspect the target.
var ErrWriteUnderInstall = errors.New("refusing to write original-format save under")

// RefuseOriginalWriteTarget is the runtime half of the preserved-install rule.
// The explicit originalDir fence catches the configured install and every
// physical descendant. The archive census catches an install accidentally
// supplied without that context, including the upper-case filenames shipped by
// the RU distribution. Missing target directories are resolved through their
// nearest existing parent so a symlink/junction cannot hide a future child.
func (f Fence) RefuseOriginalWriteTarget(target, originalDir string) error {
	return f.RefuseOriginalWriteTargets(target, []string{originalDir})
}

// RefuseOriginalWriteTargets is RefuseOriginalWriteTarget for each of roots in
// order, with the archive census, which does not depend on the root, run once
// where the first root's check ran it. With no roots it is the census alone.
func (f Fence) RefuseOriginalWriteTargets(target string, roots []string) error {
	resolvedTarget, err := resolveOriginalWriteTarget(target)
	if err != nil {
		return err
	}
	for i, root := range roots {
		if err := refuseUnderOriginal(target, resolvedTarget, root); err != nil {
			return err
		}
		if i == 0 {
			if err := f.refuseInstallAncestor(target, resolvedTarget); err != nil {
				return err
			}
		}
	}
	if len(roots) == 0 {
		return f.refuseInstallAncestor(target, resolvedTarget)
	}
	return nil
}

// resolveOriginalWriteTarget is the resolved spelling both halves of
// RefuseOriginalWriteTarget compare.
func resolveOriginalWriteTarget(target string) (string, error) {
	if target == "" {
		return "", errors.New("no save directory configured")
	}
	resolvedTarget, err := ResolvePath(target)
	if err != nil {
		return "", fmt.Errorf("resolve original-save target %s: %w", target, err)
	}
	return resolvedTarget, nil
}

// refuseUnderOriginal is the explicit fence: target, resolved or physical,
// lies in originalDir.
func refuseUnderOriginal(target, resolvedTarget, originalDir string) error {
	if originalDir != "" {
		resolvedOriginal, err := ResolvePath(originalDir)
		if err != nil {
			return fmt.Errorf("resolve read-only game install %s: %w", originalDir, err)
		}
		physical, err := SameOrDescendantPhysical(originalDir, target)
		if err != nil {
			return fmt.Errorf("compare original-save target with read-only game install: %w", err)
		}
		if InsideDir(resolvedOriginal, resolvedTarget) || physical {
			return fmt.Errorf("%w read-only game install %s", ErrWriteUnderInstall, originalDir)
		}
	}
	return nil
}

// refuseInstallAncestor is the archive census: an ancestor of the resolved
// target holds the required archives. It does not depend on originalDir.
func (f Fence) refuseInstallAncestor(target, resolvedTarget string) error {
	installDir, install, err := f.InstallAncestor(resolvedTarget)
	if err != nil {
		return fmt.Errorf("inspect original-save target ancestors for %s: %w", target, err)
	}
	if install {
		return fmt.Errorf("%w game install %s", ErrWriteUnderInstall, installDir)
	}
	return nil
}

// InstallAncestor is the nearest of target and its ancestors that holds the
// fence's archives.
func (f Fence) InstallAncestor(target string) (string, bool, error) {
	for probe := filepath.Clean(target); ; probe = filepath.Dir(probe) {
		install, err := f.IsInstall(probe)
		if err != nil {
			return "", false, err
		}
		if install {
			return probe, true, nil
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", false, nil
		}
	}
}

// ResolvePath is path made absolute with the links of its nearest existing
// ancestor resolved, the spelling the fence compares.
func ResolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	probe := abs
	for {
		if _, err := os.Lstat(probe); err == nil {
			resolved, err := filepath.EvalSymlinks(probe)
			if err != nil {
				// Windows can deny reparse-point metadata for an otherwise
				// readable directory. Physical identity is still checked by
				// SameOrDescendantPhysical; retain the absolute spelling.
				resolved = probe
			}
			rel, err := filepath.Rel(probe, abs)
			if err != nil {
				return "", err
			}
			return filepath.Clean(filepath.Join(resolved, rel)), nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return abs, nil
		}
		probe = parent
	}
}

// SameOrDescendantPhysical reports whether child or one of its ancestors is
// the same directory as parent.
func SameOrDescendantPhysical(parent, child string) (bool, error) {
	parentInfo, err := os.Stat(parent)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	absChild, err := filepath.Abs(child)
	if err != nil {
		return false, err
	}
	for probe := filepath.Clean(absChild); ; probe = filepath.Dir(probe) {
		info, err := os.Stat(probe)
		if err == nil && os.SameFile(parentInfo, info) {
			return true, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		parentProbe := filepath.Dir(probe)
		if parentProbe == probe {
			return false, nil
		}
	}
}

// IsInstall reports whether dir holds every one of the fence's archives.
func (f Fence) IsInstall(dir string) (bool, error) {
	want := make(map[string]bool, len(f.Archives))
	for _, name := range f.Archives {
		want[namefold.Host(name)] = false
	}
	// The shipped game and the supported runtime are Windows, where a direct
	// lookup is case-insensitive. Besides being cheaper, these probes still
	// work when the sandbox permits naming a child but refuses enumerating a
	// broad ancestor such as the user profile. The install name resolver below
	// supplies the same case-insensitive census on case-sensitive filesystems.
	direct, absent := 0, 0
	for name := range want {
		info, err := os.Stat(filepath.Join(dir, name))
		if err == nil && !info.IsDir() {
			want[name] = true
			direct++
			continue
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, os.ErrPermission) {
			return false, err
		}
		if errors.Is(err, os.ErrNotExist) {
			absent++
		}
	}
	if direct == len(want) {
		return true, nil
	}
	if runtime.GOOS == "windows" && absent == len(want) {
		return false, nil
	}
	files := installpath.New(dir)
	for name, found := range want {
		if found {
			continue
		}
		e, err := files.Resolve(name)
		if errors.Is(err, fs.ErrNotExist) || (errors.Is(err, os.ErrPermission) && direct == 0) {
			return false, nil
		}
		if errors.Is(err, installpath.ErrAmbiguous) {
			// Two spellings of a required archive still mark an install.
			continue
		}
		if err != nil {
			return false, err
		}
		if e.Dir {
			return false, nil
		}
	}
	return true, nil
}

// SameName reports whether two names in one folder name the same file: case
// folded on Windows, exact elsewhere.
func SameName(a, b string) bool {
	if runtime.GOOS == "windows" {
		return namefold.HostEqual(a, b)
	}
	return a == b
}

// InsideDir reports whether path is parent itself or lies under it.
//
// THE COMPARISON IS CASE-INSENSITIVE because both sides are read off the same
// Windows machine through different APIs and neither promises the other's
// casing. Cutting the prefix can split a multi-byte rune, in which case EqualFold
// answers false and the caller keeps the plain beside-the-binary rule — the
// behaviour that shipped, which is the safe direction to fail in.
func InsideDir(parent, path string) bool {
	p, c := filepath.Clean(parent), filepath.Clean(path)
	if p == "" || p == "." || len(c) < len(p) || !namefold.HostEqual(c[:len(p)], p) {
		return false
	}
	return len(c) == len(p) || os.IsPathSeparator(c[len(p)]) || os.IsPathSeparator(p[len(p)-1])
}
