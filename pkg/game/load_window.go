package game

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/ui"
)

func (f *FrontEnd) wireLoadWindow(a *ui.App) {
	w := ui.LoadWindowWords{Title: "Load Saved Game", Subtitle: "Load the Game", OK: "OK", Delete: "Delete", Cancel: "Cancel", Confirm: "Delete selected#saved game"}
	if f.Archives != nil {
		t := LoadTextTable(f.Archives.Containers, DialogsTextPath, f.textCode())
		for i, p := range map[int]*string{151: &w.Title, 24: &w.Subtitle, 0: &w.OK, 157: &w.Delete, 1: &w.Cancel, 158: &w.Confirm} {
			if s, ok := t.At(i); ok {
				*p = s
			}
		}
	}
	a.SetLoadWindowWords(w)
}

func prepareLoadDelete(dir, token string, fences []string, profiles ...*runtimeProfileAccess) (func() error, error) {
	return prepareLoadDeleteFiles(dir, token, fences, osNamedSaveFiles{}, profiles...)
}

func prepareLoadDeleteFiles(dir, token string, fences []string, files namedSaveFileOps, profiles ...*runtimeProfileAccess) (func() error, error) {
	path, err := loadDeletePath(dir, token, fences, profiles...)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	raw, err := ReadSaveFile(path)
	if err != nil {
		return nil, err
	}
	identity, err := resolveSaveDirectory(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	targets := []namedSaveTarget{{path: path, before: info, oldHash: sha256.Sum256(raw), old: raw}}
	if companion, owned := timedSlots.deleteCompanion(path, raw); owned {
		targets = append(targets, companion)
	} else if companion, owned := quickSlots.deleteCompanion(path, raw); owned {
		targets = append(targets, companion)
	}
	return func() error {
		namedSaveMu.Lock()
		defer namedSaveMu.Unlock()
		return deleteNamedSaveSet(identity, targets, fences, files, profiles...)
	}, nil
}

func deleteNamedSaveSet(dir saveDirectoryIdentity, targets []namedSaveTarget, fences []string, files namedSaveFileOps, profiles ...*runtimeProfileAccess) error {
	checkDirectory := func() error {
		for _, path := range []string{dir.path, filepath.Dir(targets[0].path)} {
			current, err := resolveSaveDirectory(path)
			if err != nil || !os.SameFile(dir.info, current.info) {
				return fmt.Errorf("save directory changed before deletion")
			}
			if _, err := namedSaveDirectory(path, fences, profiles...); err != nil {
				return err
			}
		}
		return nil
	}
	if err := checkDirectory(); err != nil {
		return err
	}
	for _, target := range targets {
		if err := checkNamedSaveTarget(target); err != nil {
			return err
		}
	}
	backups := make([]string, len(targets))
	defer func() {
		for _, path := range backups {
			if path != "" {
				_ = files.Remove(path)
			}
		}
	}()
	for i, target := range targets {
		if err := checkDirectory(); err != nil {
			return err
		}
		var err error
		backups[i], err = stageNamedSave(dir.path, target.old, files)
		if err != nil {
			return err
		}
	}
	removed := 0
	rollback := func(primary error) error {
		failures := []error{primary}
		for i := removed - 1; i >= 0; i-- {
			err := checkDirectory()
			if err == nil {
				var result savePublishResult
				result, err = files.Publish(backups[i], targets[i].path)
				if err == nil && !result.committed {
					err = fmt.Errorf("restore did not commit")
				}
			}
			if err != nil {
				failures = append(failures, fmt.Errorf("restore %s from %s: %w", targets[i].path, backups[i], err))
				backups[i] = ""
			}
		}
		return errors.Join(failures...)
	}
	for _, target := range targets {
		if err := checkDirectory(); err != nil {
			return rollback(err)
		}
		if err := checkNamedSaveTarget(target); err != nil {
			return rollback(err)
		}
		if err := files.Remove(target.path); err != nil {
			return rollback(err)
		}
		removed++
	}
	return nil
}

// LOAD tokens distinguish authored SAVs from read-only install saves even when
// both directories contain the same filename. Deletion uses that same boundary.
func loadDeletePath(dir, token string, fences []string, profiles ...*runtimeProfileAccess) (string, error) {
	name := token
	if local, ok := localOriginalSaveName(token); ok {
		name = local
	} else if IsOriginal(token) {
		return "", fmt.Errorf("the original save is read-only")
	}
	if name != filepath.Base(name) || !strings.EqualFold(filepath.Ext(name), ".sav") {
		return "", fmt.Errorf("invalid save name")
	}
	root, err := namedSaveDirectory(dir, fences, profiles...)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, name)
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("save is not a regular file")
	}
	return path, nil
}
