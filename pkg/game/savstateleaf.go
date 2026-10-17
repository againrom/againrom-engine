package game

import (
	"strings"

	"againrom/pkg/formats/sav"
)

const stateNameBytes = 15

// stateNameMatches is the one name rule of the state store (REG-100). A
// parent whose kind carries bit 4 is sorted: the query is cut to fifteen
// bytes and compared byte for byte. Any other parent is scanned: fifteen bytes
// compared with ASCII case folded.
func stateNameMatches(parentKind uint32, stored, query string) bool {
	if len(query) > stateNameBytes {
		query = query[:stateNameBytes]
	}
	if parentKind&0x10 != 0 {
		return stored == query
	}
	if len(stored) > stateNameBytes {
		stored = stored[:stateNameBytes]
	}
	return asciiFoldEqual(stored, query)
}

func asciiFoldEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 'a' - 'A'
		}
		if 'A' <= y && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}

func splitStatePath(path string) (root, leaf string) {
	root, leaf, _ = strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return root, leaf
}

func stateRootMatches(rootKind uint32, storedPath, root string) bool {
	stored, _ := splitStatePath(storedPath)
	return stateNameMatches(rootKind, stored, root)
}

// stateRecordIs: the root by the root kind, the leaf by its directory's kind.
func stateRecordIs(rootKind uint32, dirs []sav.CityStateDirectoryData, storedPath, path string) bool {
	root, leaf := splitStatePath(path)
	storedRoot, storedLeaf := splitStatePath(storedPath)
	if !stateNameMatches(rootKind, storedRoot, root) {
		return false
	}
	dirPath := "/" + storedRoot
	if strings.HasPrefix(storedPath, "/") {
		dirPath = storedPath[:1+len(storedRoot)]
	}
	kind := uint32(0)
	for _, d := range dirs {
		if d.Path == dirPath {
			kind = d.Kind
			break
		}
	}
	return stateNameMatches(kind, storedLeaf, leaf)
}

func stateLeafRecords(rootKind uint32, dirs []sav.CityStateDirectoryData, values []sav.CityStateRecordData, path string) []int {
	var out []int
	for i, r := range values {
		if stateRecordIs(rootKind, dirs, r.Path, path) {
			out = append(out, i)
		}
	}
	return out
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
