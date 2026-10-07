//go:build !windows

package game

import "path/filepath"

func editorPhysicalDirectory(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
