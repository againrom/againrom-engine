package game

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"againrom/pkg/ui"
)

type namedSavePayload struct {
	ext  string
	data []byte
}

type namedSaveTarget struct {
	path    string
	data    []byte
	before  os.FileInfo
	oldHash [32]byte
	old     []byte
}

type namedSaveFileOps interface {
	CreateTemp(string, string) (saveTempFile, error)
	Publish(string, string) (savePublishResult, error)
	Replace(string, string) error
	Remove(string) error
}

type osNamedSaveFiles struct{ osSaveFileOps }

func (osNamedSaveFiles) Replace(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

var namedSaveMu sync.Mutex

// validatedSaveName is a validated basename with one optional format suffix
// removed. Keeping it distinct prevents a later layer from stripping again.
type validatedSaveName string

func namedSaveBase(name string) (validatedSaveName, error) {
	ext := filepath.Ext(name)
	if strings.EqualFold(ext, ".ags") || strings.EqualFold(ext, ".sav") {
		name = name[:len(name)-len(ext)]
	}
	if name == "" || name != strings.TrimSpace(name) || strings.HasSuffix(name, ".") || len(name) > 120 ||
		strings.ContainsAny(name, "<>:\"/\\|?*") || name == "." || name == ".." {
		return "", fmt.Errorf("enter a save name without path separators or a trailing space/dot")
	}
	for _, r := range name {
		if r < 32 {
			return "", fmt.Errorf("save name contains a control character")
		}
	}
	device := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	if device == "CON" || device == "PRN" || device == "AUX" || device == "NUL" ||
		len(device) == 4 && (strings.HasPrefix(device, "COM") || strings.HasPrefix(device, "LPT")) && device[3] >= '0' && device[3] <= '9' {
		return "", fmt.Errorf("save name is reserved by the operating system")
	}
	return validatedSaveName(name), nil
}

func namedSaveDirectory(path string, fences []string, profiles ...*runtimeProfileAccess) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("choose a save directory")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	for _, fence := range fences {
		if err := refuseProfileWriteTarget(abs, fence, profiles...); err != nil {
			return "", err
		}
	}
	// Even callers without a configured fence retain the archive census.
	if err := refuseProfileWriteTarget(abs, "", profiles...); err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func listSaveDirectory(path string, fences []string, selector int, profiles ...*runtimeProfileAccess) (ui.SaveDirectory, error) {
	dir, err := namedSaveDirectory(path, fences, profiles...)
	if err != nil {
		return ui.SaveDirectory{}, err
	}
	out := ui.SaveDirectory{Path: dir}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			out.Directories = append(out.Directories, entry.Name())
		}
	}
	saves, err := (SaveStore{Dir: dir, Selector: selector}).List()
	if err != nil {
		return out, err
	}
	for _, save := range saves {
		out.Entries = append(out.Entries, ui.SaveEntry{Name: save.Name, Label: save.Label})
	}
	return out, nil
}

// prepareNamedSave captures exact targets and bytes without writing. A
// changed target invalidates confirmation, including a corrupt existing file
// that does not appear in SaveStore.List. The old automatic writer is separate.
func prepareNamedSave(directory string, base validatedSaveName, payloads []namedSavePayload, fences []string, files namedSaveFileOps, profiles ...*runtimeProfileAccess) (ui.PreparedSave, error) {
	return prepareOwnedNamedSave(directory, base, payloads, fences, files, nil, profiles...)
}

func prepareOwnedNamedSave(directory string, base validatedSaveName, payloads []namedSavePayload, fences []string, files namedSaveFileOps, validate func([]namedSaveTarget) error, profiles ...*runtimeProfileAccess) (ui.PreparedSave, error) {
	var out ui.PreparedSave
	if base == "" {
		return out, fmt.Errorf("save name has not been prepared")
	}
	dir, err := namedSaveDirectory(directory, fences, profiles...)
	if err != nil {
		return out, err
	}
	resolved, err := resolvePathForSaveSafety(dir)
	if err != nil {
		return out, err
	}
	ancestor := dir
	var ancestorInfo os.FileInfo
	for {
		ancestorInfo, err = os.Stat(ancestor)
		if err == nil {
			if !ancestorInfo.IsDir() {
				return out, fmt.Errorf("save directory has a non-directory parent")
			}
			break
		}
		if !os.IsNotExist(err) || filepath.Dir(ancestor) == ancestor {
			return out, err
		}
		ancestor = filepath.Dir(ancestor)
	}
	if len(payloads) == 0 || len(payloads) > 2 {
		return out, fmt.Errorf("select one format or BOTH")
	}
	targets := make([]namedSaveTarget, 0, len(payloads))
	seen := make(map[string]bool)
	for _, payload := range payloads {
		if payload.ext != ".ags" && payload.ext != ".sav" && payload.ext != timedOwnerExtension && payload.ext != quickOwnerExtension || seen[payload.ext] || len(payload.data) > maxSaveBytes || len(payload.data) == 0 {
			return out, fmt.Errorf("invalid named save payload")
		}
		seen[payload.ext] = true
		target := namedSaveTarget{path: filepath.Join(dir, string(base)+payload.ext), data: bytes.Clone(payload.data)}
		target.before, err = os.Lstat(target.path)
		if err == nil {
			if !target.before.Mode().IsRegular() {
				return out, fmt.Errorf("save target is not a regular file: %s", target.path)
			}
			target.old, err = ReadSaveFile(target.path)
			if err != nil {
				return out, err
			}
			target.oldHash = sha256.Sum256(target.old)
			out.Existing = append(out.Existing, target.path)
		} else if !os.IsNotExist(err) {
			return out, err
		}
		targets = append(targets, target)
		out.Paths = append(out.Paths, target.path)
	}
	if validate != nil {
		if err := validate(targets); err != nil {
			return out, err
		}
	}
	if files == nil {
		files = osNamedSaveFiles{}
	}
	used := false
	out.Commit = func(overwrite bool) ([]string, error) {
		namedSaveMu.Lock()
		defer namedSaveMu.Unlock()
		if used {
			return nil, fmt.Errorf("save request already committed; prepare a new request")
		}
		for _, target := range targets {
			if target.before != nil && !overwrite {
				return nil, fmt.Errorf("confirm replacement of %s", target.path)
			}
		}
		if _, err := namedSaveDirectory(dir, fences, profiles...); err != nil {
			return nil, err
		}
		nowPath, err := resolvePathForSaveSafety(dir)
		if err != nil || !sameSaveName(resolved, nowPath) {
			return nil, fmt.Errorf("save directory changed after preparation")
		}
		info, err := os.Stat(ancestor)
		if err != nil || !os.SameFile(ancestorInfo, info) {
			return nil, fmt.Errorf("save directory parent changed after preparation")
		}
		for _, target := range targets {
			if err := checkNamedSaveTarget(target); err != nil {
				return nil, err
			}
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		actual, err := resolveSaveDirectory(dir)
		if err != nil {
			return nil, err
		}
		if err := commitNamedSaveSet(actual, targets, fences, files, profiles...); err != nil {
			return nil, err
		}
		used = true
		paths := make([]string, len(targets))
		for i := range targets {
			paths[i] = targets[i].path
		}
		return paths, nil
	}
	return out, nil
}

func checkNamedSaveTarget(target namedSaveTarget) error {
	now, err := os.Lstat(target.path)
	if target.before == nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("target appeared after preparation: %s", target.path)
	}
	if err != nil || !now.Mode().IsRegular() || !os.SameFile(target.before, now) {
		return fmt.Errorf("target changed after confirmation: %s", target.path)
	}
	raw, err := ReadSaveFile(target.path)
	if err != nil || sha256.Sum256(raw) != target.oldHash {
		return fmt.Errorf("target contents changed after confirmation: %s", target.path)
	}
	return nil
}

func stageNamedSave(dir string, data []byte, files namedSaveFileOps) (string, error) {
	f, err := files.CreateTemp(dir, ".save-dialog-*")
	if err != nil {
		return "", err
	}
	path := f.Name()
	_, writeErr := io.Copy(f, bytes.NewReader(data))
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		_ = files.Remove(path)
		return "", err
	}
	return path, nil
}

// Both payloads and replacement backups are durable before either final path
// changes. A publication error restores every changed path before returning.
func commitNamedSaveSet(dir saveDirectoryIdentity, targets []namedSaveTarget, fences []string, files namedSaveFileOps, profiles ...*runtimeProfileAccess) error {
	staged := make([]string, len(targets))
	backups := make([]string, len(targets))
	cleanup := func() {
		for _, path := range append(staged, backups...) {
			if path != "" {
				_ = files.Remove(path)
			}
		}
	}
	defer cleanup()
	for i, target := range targets {
		var err error
		staged[i], err = stageNamedSave(dir.path, target.data, files)
		if err != nil {
			return fmt.Errorf("prepare save files: %w", err)
		}
		if target.before != nil {
			backups[i], err = stageNamedSave(dir.path, target.old, files)
			if err != nil {
				return fmt.Errorf("prepare overwrite backup: %w", err)
			}
		}
	}
	committed := 0
	rollback := func(primary error) error {
		failures := []error{primary}
		for i := committed - 1; i >= 0; i-- {
			var err error
			if targets[i].before == nil {
				err = files.Remove(targets[i].path)
			} else {
				err = files.Replace(backups[i], targets[i].path)
			}
			if err != nil {
				failures = append(failures, fmt.Errorf("restore %s from %s: %w", targets[i].path, backups[i], err))
				backups[i] = "" // preserve the recovery file named by the error
			}
		}
		return errors.Join(failures...)
	}
	for i, target := range targets {
		current, err := resolveSaveDirectory(dir.path)
		if err != nil || !os.SameFile(dir.info, current.info) {
			return rollback(fmt.Errorf("save directory changed before publication"))
		}
		if _, err := namedSaveDirectory(dir.path, fences, profiles...); err != nil {
			return rollback(err)
		}
		if err := checkNamedSaveTarget(target); err != nil {
			return rollback(err)
		}
		if target.before != nil {
			err = files.Replace(staged[i], target.path)
		} else {
			var result savePublishResult
			result, err = files.Publish(staged[i], target.path)
			if err == nil && !result.committed {
				err = fmt.Errorf("save publication did not commit")
			}
		}
		if err != nil {
			return rollback(fmt.Errorf("publish %s: %w", target.path, err))
		}
		committed++
	}
	return nil
}
