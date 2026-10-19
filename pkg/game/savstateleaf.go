package game

import (
	"againrom/pkg/formats/sav"
	"strings"
)

func stateNameMatches(kind uint32, stored, query string) bool {
	return sav.StateNameMatches(kind, stored, query)
}
func splitStatePath(path string) (root, leaf string) {
	root, leaf, _ = strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return
}
func stateRootMatches(kind uint32, stored, root string) bool {
	return sav.StateRootMatches(kind, stored, root)
}
func stateRecordIs(kind uint32, dirs []sav.CityStateDirectoryData, stored, path string) bool {
	return sav.StateRecordIs(kind, dirs, stored, path)
}
func stateLeafRecords(kind uint32, dirs []sav.CityStateDirectoryData, values []sav.CityStateRecordData, path string) []int {
	return sav.StateLeafRecords(kind, dirs, values, path)
}

// patchStateLeaf patches every record the lookup selects: a Document keeps no
// child order, so colliding names all take the value the original reads.
func patchStateLeaf(rootKind uint32, dirs []sav.CityStateDirectoryData, values []sav.CityStateRecordData, path string, patch func(*sav.CityStateValueData) error) (int, error) {
	at := stateLeafRecords(rootKind, dirs, values, path)
	for _, i := range at {
		if err := patch(&values[i].Value); err != nil {
			return 0, err
		}
	}
	return len(at), nil
}

func hasStateRoot(rootKind uint32, dirs []sav.CityStateDirectoryData, root string) bool {
	for _, d := range dirs {
		if stateRootMatches(rootKind, d.Path, root) {
			return true
		}
	}
	return false
}
