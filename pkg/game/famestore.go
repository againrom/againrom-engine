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
	rows, _, err := readFameFile(filepath.Join(s.OriginalDir, fameFileName))
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
		rows, _, err = readFameFile(filepath.Join(s.OriginalDir, fameFileName))
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
	tmp, err := files.CreateTemp(identity.path, ".againrom-fame-*.tmp")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	closed := false
	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		_ = files.Remove(tmpPath)
	}()
	if _, err = io.Copy(tmp, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	if err = tmp.Sync(); err != nil {
		return nil, err
	}
	closed = true
	if err = tmp.Close(); err != nil {
		return nil, err
	}
	current, err := resolveSaveDirectory(dir)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(identity.info, current.info) {
		return nil, fmt.Errorf("hall directory changed")
	}
	if _, err = namedSaveDirectory(current.path, []string{s.OriginalDir}, s.profile); err != nil {
		return nil, err
	}
	if before == nil {
		result, err := files.Publish(tmpPath, path)
		if !result.committed {
			if err == nil {
				err = fmt.Errorf("hall publication did not commit")
			}
			return nil, err
		}
	} else {
		now, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !now.Mode().IsRegular() || !os.SameFile(before, now) {
			return nil, fmt.Errorf("hall table changed")
		}
		_, actual, err := readFameFile(path)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(old, actual) {
			return nil, fmt.Errorf("hall table changed")
		}
		if err := files.Replace(tmpPath, path); err != nil {
			return nil, err
		}
	}
	return rows, nil
}
