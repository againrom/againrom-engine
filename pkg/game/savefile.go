package game

import (
	"fmt"
	"os"
)

// ReadSaveFile bounds explicit-path reads with the same ceiling as SaveStore.
func ReadSaveFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("save input is not a regular file")
	}
	return readBoundedSave(f, info.Size())
}
