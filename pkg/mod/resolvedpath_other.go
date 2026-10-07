//go:build !windows

package mod

import "path/filepath"

func resolvedPath(name string) (string, error) { return filepath.EvalSymlinks(name) }
