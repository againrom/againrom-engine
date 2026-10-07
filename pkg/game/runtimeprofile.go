package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// RuntimeProfile keeps all mutable player files under one launch choice.
type RuntimeProfile struct {
	Directory string
	Saves     SaveStore
	Options   OptionsStore
}

// ResolveRuntimeProfile grants only an implicit install-contained executable
// its own private profile. Explicit output paths retain the ordinary fences.
func ResolveRuntimeProfile(saveOverride, executable, userConfigDir, originalDir string) (RuntimeProfile, error) {
	if saveOverride != "" {
		return runtimeProfileStores(saveOverride, originalDir, nil), nil
	}
	if executable == "" {
		return RuntimeProfile{}, fmt.Errorf("cannot locate the executable for the player profile")
	}
	exeDir, err := editorPhysicalDirectory(filepath.Dir(executable))
	if err != nil {
		return RuntimeProfile{}, err
	}
	_, install, err := requiredArchiveAncestor(exeDir)
	if err != nil {
		return RuntimeProfile{}, err
	}
	if !install {
		var wd string
		roots := toolchainRoots()
		if insideAny(roots, filepath.Dir(executable)) {
			wd, err = os.Getwd()
			if err != nil {
				return RuntimeProfile{}, err
			}
		}
		return runtimeProfileStores(saveDirFor(executable, wd, roots...), originalDir, nil), nil
	}
	root := filepath.Join(exeDir, "Againrom")
	access, privateErr := prepareRuntimeProfile(root, originalDir, true)
	if privateErr == nil {
		return runtimeProfileStores(access.directory.path, originalDir, access), nil
	}
	if userConfigDir == "" {
		return RuntimeProfile{}, fmt.Errorf("private profile unavailable and user configuration directory is absent: %w", privateErr)
	}
	root = filepath.Join(userConfigDir, "Againrom")
	if err := refuseOriginalWriteTarget(root, originalDir); err != nil {
		return RuntimeProfile{}, fmt.Errorf("fallback profile: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(root), 0700); err != nil {
		return RuntimeProfile{}, err
	}
	if _, err := prepareRuntimeProfile(root, originalDir, false); err != nil {
		return RuntimeProfile{}, fmt.Errorf("private profile: %v; fallback profile: %w", privateErr, err)
	}
	return runtimeProfileStores(filepath.Join(root, "saves"), originalDir, nil), nil
}

func runtimeProfileStores(saves, originalDir string, access *runtimeProfileAccess) RuntimeProfile {
	return RuntimeProfile{Directory: filepath.Dir(saves),
		Saves:   SaveStore{Dir: saves, profile: access},
		Options: OptionsStore{Path: filepath.Join(filepath.Dir(saves), optionsFileName), originalDir: originalDir, profile: access.rootAccess()}}
}

type runtimeProfileAccess struct {
	root, directory saveDirectoryIdentity
	descendants     bool
}

func (p *runtimeProfileAccess) rootAccess() *runtimeProfileAccess {
	if p == nil {
		return nil
	}
	return &runtimeProfileAccess{root: p.root, directory: p.root}
}

func prepareRuntimeProfile(root, originalDir string, allowInstall bool) (*runtimeProfileAccess, error) {
	if !allowInstall {
		if err := refuseOriginalWriteTarget(root, originalDir); err != nil {
			return nil, err
		}
	}
	anchor, err := createProfileDirectory(root)
	if err != nil {
		return nil, err
	}
	if !allowInstall {
		if err := refuseOriginalWriteTarget(anchor.path, originalDir); err != nil {
			return nil, err
		}
	}
	if install, err := directoryHasRequiredArchives(anchor.path); err != nil || install {
		return nil, fmt.Errorf("private profile is a game install: %s: %v", root, err)
	}
	dir, err := createProfileDirectory(filepath.Join(anchor.path, "saves"))
	if err != nil {
		return nil, err
	}
	access := &runtimeProfileAccess{root: anchor, directory: dir, descendants: true}
	for _, directory := range []saveDirectoryIdentity{anchor, dir} {
		if !allowInstall {
			if err := refuseOriginalWriteTarget(directory.path, originalDir); err != nil {
				return nil, err
			}
		}
		grant := access
		if sameSaveName(directory.path, anchor.path) {
			grant = access.rootAccess()
		}
		if _, err := grant.allows(directory.path); err != nil {
			return nil, err
		}
		probe, err := os.CreateTemp(directory.path, ".againrom-write-probe-*")
		if err != nil {
			return nil, err
		}
		_, writeErr := probe.Write([]byte{0})
		syncErr, closeErr := probe.Sync(), probe.Close()
		removeErr := os.Remove(probe.Name())
		if err := errors.Join(writeErr, syncErr, closeErr, removeErr); err != nil {
			return nil, err
		}
	}
	return access, nil
}

func createProfileDirectory(path string) (saveDirectoryIdentity, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return saveDirectoryIdentity{}, err
	}
	parent, err := editorPhysicalDirectory(filepath.Dir(abs))
	if err != nil {
		return saveDirectoryIdentity{}, err
	}
	path = filepath.Join(parent, filepath.Base(abs))
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(path, 0700)
		if err == nil {
			info, err = os.Lstat(path)
		}
	}
	if err != nil {
		return saveDirectoryIdentity{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return saveDirectoryIdentity{}, fmt.Errorf("profile directory is not a private directory: %s", path)
	}
	physical, err := editorPhysicalDirectory(path)
	if err != nil || !sameSaveName(physical, path) {
		return saveDirectoryIdentity{}, fmt.Errorf("profile directory redirects: %s: %v", path, err)
	}
	return saveDirectoryIdentity{path: path, info: info}, nil
}

func (p *runtimeProfileAccess) allows(path string) (bool, error) {
	if p == nil {
		return false, nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	if !sameSaveName(abs, p.directory.path) && (!p.descendants || !insideDir(p.directory.path, abs)) {
		return false, nil
	}
	for _, held := range []saveDirectoryIdentity{p.root, p.directory} {
		info, err := os.Lstat(held.path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !os.SameFile(held.info, info) {
			return false, fmt.Errorf("profile directory changed: %s", held.path)
		}
	}
	for probe := filepath.Clean(abs); ; probe = filepath.Dir(probe) {
		info, err := os.Lstat(probe)
		if err == nil {
			physical, physicalErr := editorPhysicalDirectory(probe)
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || physicalErr != nil || !sameSaveName(physical, probe) {
				return false, fmt.Errorf("profile target redirects: %s", probe)
			}
			if install, err := directoryHasRequiredArchives(probe); err != nil || install {
				return false, fmt.Errorf("profile contains a game install: %s: %v", probe, err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		if sameSaveName(probe, p.root.path) {
			break
		}
		if filepath.Dir(probe) == probe {
			return false, fmt.Errorf("profile target escaped its root")
		}
	}
	return true, nil
}

func refuseProfileWriteTarget(path, originalDir string, profiles ...*runtimeProfileAccess) error {
	if len(profiles) != 0 {
		allowed, err := profiles[0].allows(path)
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}
	}
	return refuseOriginalWriteTarget(path, originalDir)
}
