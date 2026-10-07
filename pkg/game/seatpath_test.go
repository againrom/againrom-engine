package game

import (
	"os"
	"path/filepath"
)

// seatPath finds rel under the nearest ancestor of the working directory.
func seatPath(rel string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, filepath.FromSlash(rel))
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func reviewDirRoot(name string) string { return seatPath("review/" + name) }
