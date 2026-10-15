package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/formats/sav"
)

// Test helpers observe production SAV LOAD and publish explicitly named SAV
// fixtures through the install fence and no-replace atomic protocol.

// LoadObservedSAV runs a cold LOAD of SAV bytes on a fresh FrontEnd and returns
// the state it installed.
func (f *FrontEnd) LoadObservedSAV(saved []byte) (Snapshot, error) {
	open, town, err := f.RestoreOriginal(saved)
	if err != nil {
		return Snapshot{}, err
	}
	if !town {
		if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
			return Snapshot{}, err
		}
	}
	return f.ObserveLoadedState(!town)
}

// CheckConvertedSaveTarget checks the install fence and archive ancestors,
// including aliases of not-yet-created children, without writing the output.
func CheckConvertedSaveTarget(path, assetsRoot string) error {
	if path == "" {
		return fmt.Errorf("conversion output path is required")
	}
	if assetsRoot == "" {
		return fmt.Errorf("conversion asset root is required for the read-only install fence")
	}
	return refuseOriginalWriteTarget(filepath.Dir(path), assetsRoot)
}

// WriteConvertedSave publishes one explicitly named SAV output through
// SaveStore's synced, no-replace atomic protocol. There is no directory or
// filename default. It validates SAV bytes before publishing them.
func WriteConvertedSave(path string, data []byte, assetsRoot string) error {
	if err := CheckConvertedSaveTarget(path, assetsRoot); err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".sav" {
		return fmt.Errorf("conversion output must have .sav extension")
	}
	if _, err := sav.DecodeDocumentData(data); err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("conversion output already exists: %w", os.ErrExist)
	} else if !os.IsNotExist(err) {
		return err
	}
	store := SaveStore{Dir: filepath.Dir(path)}
	validate := func(dir saveDirectoryIdentity) error {
		return refuseOriginalWriteTarget(dir.path, assetsRoot)
	}
	_, err := store.write(data, "save-conversion", ext, validate, func(at int) (string, bool) {
		return filepath.Base(path), at == 0
	})
	if err != nil {
		return fmt.Errorf("publish conversion output without replacement: %w", err)
	}
	return nil
}
