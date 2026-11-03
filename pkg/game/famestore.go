package game

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"againrom/pkg/formats/fame"
	"againrom/pkg/hostfile"
)

// The mutable hall belongs to the player profile. OriginalDir is a read-only
// seed and an output fence; the original WM_DESTROY write is not reproduced.
type fameStore struct {
	Dir, OriginalDir string
	files            namedSaveFileOps
	profile          *runtimeProfileAccess
}

var fameStoreMu sync.Mutex

const fameFileName = "famehall.dat"

func readFameFile(path string) ([]fame.Record, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 5<<20 {
		return nil, nil, fmt.Errorf("invalid hall file")
	}
	b, err := io.ReadAll(io.LimitReader(f, (5<<20)+1))
	if err != nil {
		return nil, nil, err
	}
	if len(b) > 5<<20 {
		return nil, nil, fmt.Errorf("hall file is too large")
	}
	rows, err := fame.Parse(b)
	return rows, b, err
}

// read distinguishes an absent file from a corrupt local table. Only absence
// permits the installed seed; malformed player data must never be overwritten.
func (s fameStore) read() ([]fame.Record, error) {
	if s.Dir != "" {
		rows, _, err := readFameFile(filepath.Join(s.Dir, fameFileName))
		if !errors.Is(err, os.ErrNotExist) {
			return rows, err
		}
	}
	if s.OriginalDir == "" {
		return nil, os.ErrNotExist
	}
	return readInstalledFame(s.OriginalDir)
}

// readInstalledFame reads the hall the install ships, whatever the case of its
// file name.
func readInstalledFame(dir string) ([]fame.Record, error) {
	path, err := InstallFile(dir, fameFileName)
	if err != nil {
		return nil, err
	}
	rows, _, err := readFameFile(path)
	return rows, err
}

// add commits one complete new table. An error before publication preserves
// the previous file. A completed publication is never reported as failure.
func (s fameStore) add(row fame.Record) ([]fame.Record, error) {
	fameStoreMu.Lock()
	defer fameStoreMu.Unlock()
	dir, err := namedSaveDirectory(s.Dir, []string{s.OriginalDir}, s.profile)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, fameFileName)
	before, err := os.Lstat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if before != nil && !before.Mode().IsRegular() {
		return nil, fmt.Errorf("hall target is not a regular file")
	}
	err = nil
	var rows []fame.Record
	var old []byte
	if before != nil {
		rows, old, err = readFameFile(path)
	} else if s.OriginalDir != "" {
		rows, err = readInstalledFame(s.OriginalDir)
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
	}
	if err != nil {
		return nil, err
	}
	rows, err = fame.Insert(rows, row, 10)
	if err != nil {
		return nil, err
	}
	data, err := fame.Marshal(rows)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	identity, err := resolveSaveDirectory(dir)
	if err != nil {
		return nil, err
	}
	if _, err = namedSaveDirectory(identity.path, []string{s.OriginalDir}, s.profile); err != nil {
		return nil, err
	}
	files := s.files
	if files == nil {
		files = osNamedSaveFiles{}
	}
	seen := hostfile.SeenAbsent
	if before != nil {
		seen = hostfile.SeenPresent
	}
	_, err = hostfile.Commit(hostfile.Set{
		Files: []hostfile.File{{Path: path, Data: data, Overwrite: hostfile.Replace, Seen: seen}},
		Dir:   identity.path, Temp: ".againrom-fame-*.tmp", Sync: true, Ops: files,
		Check: func(step hostfile.Step, _ int) error {
			if step != hostfile.StepPublish {
				return nil
			}
			current, err := resolveSaveDirectory(dir)
			if err != nil {
				return err
			}
			if !os.SameFile(identity.info, current.info) {
				return fmt.Errorf("hall directory changed")
			}
			if _, err = namedSaveDirectory(current.path, []string{s.OriginalDir}, s.profile); err != nil {
				return err
			}
			if before == nil {
				return nil
			}
			now, err := os.Lstat(path)
			if err != nil {
				return err
			}
			if !now.Mode().IsRegular() || !os.SameFile(before, now) {
				return fmt.Errorf("hall table changed")
			}
			_, actual, err := readFameFile(path)
			if err != nil {
				return err
			}
			if !bytes.Equal(old, actual) {
				return fmt.Errorf("hall table changed")
			}
			return nil
		},
		Words: hostfile.Words{NoCommit: "hall publication did not commit"},
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}
