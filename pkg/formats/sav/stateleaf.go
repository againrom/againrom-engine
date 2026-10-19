package sav

import (
	"fmt"
	"strings"

	"againrom/pkg/formats/reg"
)

const stateNameBytes = 15

// StateNameMatches is the one name rule of the state store (REG-100). A
// parent whose kind carries bit 4 is sorted: the query is cut to fifteen
// bytes and compared byte for byte. Any other parent is scanned: fifteen bytes
// compared with ASCII case folded.
func StateNameMatches(parentKind uint32, stored, query string) bool {
	return stateRootKey(parentKind, stored, false) == stateRootKey(parentKind, query, true)
}

func splitStatePath(path string) (root, leaf string) {
	root, leaf, _ = strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return root, leaf
}

func StateRootMatches(rootKind uint32, storedPath, root string) bool {
	stored, _ := splitStatePath(storedPath)
	return StateNameMatches(rootKind, stored, root)
}

// StateRecordIs: the root by the root kind, the leaf by its directory's kind.
func StateRecordIs(rootKind uint32, dirs []CityStateDirectoryData, storedPath, path string) bool {
	root, leaf := splitStatePath(path)
	storedRoot, storedLeaf := splitStatePath(storedPath)
	if !StateNameMatches(rootKind, storedRoot, root) {
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
	return StateNameMatches(kind, storedLeaf, leaf)
}

// StateLeafRecords returns every matching record in input order.
func StateLeafRecords(rootKind uint32, dirs []CityStateDirectoryData, values []CityStateRecordData, path string) []int {
	return indexStateLeaves(rootKind, dirs, values).records(path)
}

type stateLeafIndex struct {
	rootKind uint32
	kinds    map[string]uint32
	values   []CityStateRecordData
	roots    map[string][]int
}

func stateRootKey(kind uint32, name string, query bool) string {
	if (query || kind&0x10 == 0) && len(name) > stateNameBytes {
		name = name[:stateNameBytes]
	}
	if kind&0x10 == 0 {
		b := []byte(name)
		for i, c := range b {
			if c >= 'A' && c <= 'Z' {
				b[i] = c + 'a' - 'A'
			}
		}
		name = string(b)
	}
	return name
}

func indexStateLeaves(rootKind uint32, dirs []CityStateDirectoryData, values []CityStateRecordData) *stateLeafIndex {
	out := &stateLeafIndex{rootKind: rootKind, kinds: map[string]uint32{}, values: values, roots: map[string][]int{}}
	for _, d := range dirs {
		if _, exists := out.kinds[d.Path]; !exists {
			out.kinds[d.Path] = d.Kind
		}
	}
	for i, r := range values {
		root, _ := splitStatePath(r.Path)
		key := stateRootKey(rootKind, root, false)
		out.roots[key] = append(out.roots[key], i)
	}
	return out
}

func (s *stateLeafIndex) records(path string) []int {
	root, leaf := splitStatePath(path)
	var out []int
	for _, i := range s.roots[stateRootKey(s.rootKind, root, true)] {
		storedRoot, storedLeaf := splitStatePath(s.values[i].Path)
		if StateNameMatches(s.kinds["/"+storedRoot], storedLeaf, leaf) {
			out = append(out, i)
		}
	}
	return out
}

func (s *stateLeafIndex) single(path string) (int, bool, error) {
	at := s.records(path)
	if len(at) > 1 {
		return 0, false, fmt.Errorf("sav: ambiguous state leaf %s (%d matches)", path, len(at))
	}
	if len(at) == 0 {
		return 0, false, nil
	}
	return at[0], true, nil
}

func (s *stateLeafIndex) read(path string) (CityStateRecordData, bool, error) {
	i, present, err := s.single(path)
	if !present || err != nil {
		return CityStateRecordData{}, present, err
	}
	return s.values[i], true, nil
}

func (s *cityState) leafIndex() *stateLeafIndex {
	dirs := make([]CityStateDirectoryData, 0, len(s.directoryKinds))
	for path, kind := range s.directoryKinds {
		dirs = append(dirs, CityStateDirectoryData{path, kind})
	}
	values := make([]CityStateRecordData, 0, len(s.values))
	for _, path := range documentSortedKeys(s.values) {
		v := s.values[path]
		values = append(values, CityStateRecordData{path, CityStateValueData{v.kind, v.int32, v.bytes}})
	}
	return indexStateLeaves(s.rootKind, dirs, values)
}

// ReadStateLeaf selects at most one value. Absence is distinct from ambiguity;
// callers supply their own defaults and decode the selected value.
func ReadStateLeaf(rootKind uint32, dirs []CityStateDirectoryData, values []CityStateRecordData, path string) (CityStateValueData, bool, error) {
	r, present, err := indexStateLeaves(rootKind, dirs, values).read(path)
	return r.Value, present, err
}

// StateReader selects raw registry nodes without decoding their values.
type StateReader struct {
	lookup *stateLeafIndex
	nodes  []*reg.Node
}

// StateLeaves adapts the SAV registry to the same lookup as a Document.
func (f *File) StateLeaves() *StateReader {
	out := &StateReader{}
	var dirs []CityStateDirectoryData
	var values []CityStateRecordData
	var kind uint32
	if f.store != nil {
		kind = f.store.Root.Kind
		for _, section := range f.store.Root.Children {
			if !section.Dir {
				continue
			}
			path := "/" + section.Name
			dirs = append(dirs, CityStateDirectoryData{Path: path, Kind: section.Kind})
			for _, leaf := range section.Children {
				values = append(values, CityStateRecordData{Path: path + "/" + leaf.Name})
				out.nodes = append(out.nodes, leaf)
			}
		}
	}
	out.lookup = indexStateLeaves(kind, dirs, values)
	return out
}

func (s *StateReader) ReadLeaf(path string) (*reg.Node, bool, error) {
	i, present, err := s.lookup.single(path)
	if !present || err != nil {
		return nil, present, err
	}
	return s.nodes[i], true, nil
}

func (s DocumentStateData) ReadLeaf(path string) (CityStateValueData, bool, error) {
	return ReadStateLeaf(s.RootKind, s.DirectoryRecords, s.ValueRecords, path)
}
