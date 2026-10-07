package mod

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// MaxFileBytes bounds a file a mod's script, data or picture is read from.
const MaxFileBytes = 1 << 20

// ReadInside reads a regular file within dir, returning its relative slash path.
// Absolute paths, backslashes and paths that resolve outside dir are refused.
func ReadInside(dir, name string) ([]byte, string, error) {
	if name == "" || strings.ContainsAny(name, "\\:\x00") || path.IsAbs(name) || strings.HasPrefix(name, "/") {
		return nil, "", fmt.Errorf("%q is not a path inside the mod folder", name)
	}
	rel := path.Clean(name)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return nil, "", fmt.Errorf("%q is not a path inside the mod folder", name)
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	root, err := resolvedPath(dir)
	if err != nil {
		return nil, "", err
	}
	real, err := resolvedPath(full)
	if err != nil {
		return nil, "", fmt.Errorf("%s: no such file in the mod folder", rel)
	}
	if real != root && !strings.HasPrefix(real, root+string(filepath.Separator)) {
		return nil, "", fmt.Errorf("%q leads outside the mod folder", name)
	}
	st, err := os.Stat(real)
	if err != nil || !st.Mode().IsRegular() {
		return nil, "", fmt.Errorf("%s is not a file", rel)
	}
	if st.Size() > MaxFileBytes {
		return nil, "", fmt.Errorf("%s is larger than %d bytes", rel, MaxFileBytes)
	}
	data, err := os.ReadFile(real)
	return data, rel, err
}
