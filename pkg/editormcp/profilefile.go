package editormcp

import (
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/hostfile"
)

// WriteProfileFile replaces file in the editor profile with content: it
// leaves an equal file alone, and otherwise writes a temporary file named by
// pattern beside it and renames it over file. what names the file in the
// error. The MCP record and the editor's panel state both use it.
func WriteProfileFile(file, pattern, what string, content []byte) error {
	if old, err := os.ReadFile(file); err == nil && string(old) == string(content) {
		return nil
	}
	// A failure to create the temporary file is returned as it is; every
	// later failure names what.
	wrap := "write " + strings.ReplaceAll(what, "%", "%%") + ": "
	_, err := hostfile.Commit(hostfile.Set{
		Files: []hostfile.File{{Path: file, Data: content, Overwrite: hostfile.Replace}},
		Dir:   filepath.Dir(file),
		Temp:  pattern,
		Sync:  true,
		Words: hostfile.Words{Write: wrap + "%v", Sync: wrap + "%v", Close: wrap + "%v", Publish: wrap + "%[2]v"},
	})
	return err
}
