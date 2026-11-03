package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"againrom/pkg/hostfile"
	"againrom/pkg/namefold"
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

// namedSaveFileOps is the host-file transaction's file seam.
type namedSaveFileOps = hostfile.Ops

type osNamedSaveFiles struct{ hostfile.OS }

var namedSaveMu sync.Mutex

// validatedSaveName is a validated basename with one optional format suffix
// removed. Keeping it distinct prevents a later layer from stripping again.
type validatedSaveName string

func namedSaveBase(name string) (validatedSaveName, error) {
	ext := filepath.Ext(name)
	if namefold.HostEqual(ext, ".ags") || namefold.HostEqual(ext, ".sav") {
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
	device := namefold.HostUpper(strings.SplitN(name, ".", 2)[0])
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
	// A profile grant admits the path for every fence alike. Otherwise each
	// fence is checked, and even callers without a configured fence retain
	// the archive census, which runs once for all of them.
	if len(profiles) != 0 {
		allowed, err := profiles[0].allows(abs)
		if err != nil {
			return "", err
		}
		if allowed {
			return filepath.Clean(abs), nil
		}
	}
	if err := refuseOriginalWriteTargets(abs, fences); err != nil {
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

// namedSaveStage is the staging pattern of a named save set's files and
// backups.
const namedSaveStage = ".save-dialog-*"

// commitNamedSaveSet publishes targets as one host-file set. Both payloads
// and replacement backups are durable before either final path changes. A
// publication error restores every changed path before returning.
func commitNamedSaveSet(dir saveDirectoryIdentity, targets []namedSaveTarget, fences []string, files namedSaveFileOps, profiles ...*runtimeProfileAccess) error {
	rows := make([]hostfile.File, len(targets))
	for i, target := range targets {
		rows[i] = hostfile.File{Path: target.path, Data: target.data, Overwrite: hostfile.Backup, Seen: hostfile.SeenAbsent, Prior: target.old}
		if target.before != nil {
			rows[i].Seen = hostfile.SeenPresent
		}
	}
	_, err := hostfile.Commit(hostfile.Set{
		Files: rows, Dir: dir.path, Temp: namedSaveStage, Sync: true, Ops: files,
		Check: func(step hostfile.Step, i int) error {
			if step != hostfile.StepPublish {
				return nil
			}
			current, err := resolveSaveDirectory(dir.path)
			if err != nil || !os.SameFile(dir.info, current.info) {
				return fmt.Errorf("save directory changed before publication")
			}
			if _, err := namedSaveDirectory(dir.path, fences, profiles...); err != nil {
				return err
			}
			return checkNamedSaveTarget(targets[i])
		},
		Words: hostfile.Words{
			Stage:       "prepare save files: %w",
			StageBackup: "prepare overwrite backup: %w",
			Publish:     "publish %s: %w",
			NoCommit:    "save publication did not commit",
			Restore:     "restore %s from %s: %w",
		},
	})
	return err
}
