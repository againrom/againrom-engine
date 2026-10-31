package editormcp

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteProfileFile replaces file in the editor profile with content: it
// leaves an equal file alone, and otherwise writes a temporary file named by
// pattern beside it and renames it over file. what names the file in the
// error. The MCP record and the editor's panel state both use it.
func WriteProfileFile(file, pattern, what string, content []byte) error {
	if old, err := os.ReadFile(file); err == nil && string(old) == string(content) {
		return nil
	}
	f, err := os.CreateTemp(filepath.Dir(file), pattern)
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(content)
	if werr == nil {
		werr = f.Sync()
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp, file)
	}
	if werr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write %s: %v", what, werr)
	}
	return nil
}
