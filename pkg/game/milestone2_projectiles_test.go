package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/sim"
)

// This oracle reads the YA1 table/pool itself (REG-FMT-031, REG-REC-032,
// REG-KIND-033/034, REG-VAL-028). File.Store supplies only the framed bytes.
// Neither reg.Parse, File.Projectiles, DocumentData nor an exporter supplies
// its population, expected values, names or types.
type projectile1157Value struct {
	kind uint32
	word uint32
	data []byte
}

type projectile1157Item struct {
	id     uint16
	fields [16]int32
}

type projectile1157Source struct {
	rootKind uint32
	dirs     map[string]uint32
	values   map[string]projectile1157Value
	present  bool
	free     uint16
	ids      []uint16
	items    []projectile1157Item
}

var projectile1157Names = [...]string{
	"x", "y", "z", "picture", "dir", "phase", "lastaction", "action",
	"actiondir", "actiontarget", "actionx", "actiony", "actionz",
	"actionphase", "actionsegments", "actionspell",
}

func projectile1157Path(path string) bool {
	root, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	if root == "Projectiles" {
		return true
	}
	if !strings.HasPrefix(root, "Prj") {
		return false
	}
	_, err := strconv.ParseUint(root[3:], 10, 16)
	return err == nil
}

func projectile1157Read(raw []byte) (projectile1157Source, error) {
	s := projectile1157Source{dirs: map[string]uint32{}, values: map[string]projectile1157Value{}}
	bad := func(message string) (projectile1157Source, error) {
		return s, fmt.Errorf("raw Projectiles: %s", message)
	}
	if len(raw) < 28 || len(raw) > 64<<20 || !bytes.Equal(raw[:4], []byte("&YA1")) {
		return bad("missing YA1 header")
	}
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(raw[off:]) }
	n := uint64(u32(16))
	if n > uint64((len(raw)-28)/32) {
		return bad("record count exceeds bytes or reader bound")
	}
	poolOff := 24 + int(n)*32
	if uint64(poolOff)+4+uint64(u32(poolOff)) != uint64(len(raw)) {
		return bad("pool extent differs from Store extent")
	}
	pool := raw[poolOff+4:]
	s.rootKind = u32(12)
	if s.rootKind&1 == 0 || s.rootKind&14 != 0 {
		return bad("root is not a directory")
	}
	seen := make([]bool, int(n))
	paths := map[string]bool{}
	var copied uint64
	var walk func(uint32, uint32, string, int) error
	walk = func(first, count uint32, parent string, depth int) error {
		if depth > 32 || uint64(first)+uint64(count) > n {
			return fmt.Errorf("child range/depth outside table at %s", parent)
		}
		for i := uint64(first); i < uint64(first)+uint64(count); i++ {
			if seen[i] {
				return fmt.Errorf("record %d reached twice", i)
			}
			seen[i] = true
			off := 24 + 32*int(i)
			name := raw[off+16 : off+32]
			if end := bytes.IndexByte(name, 0); end >= 0 {
				name = name[:end]
			}
			path := parent + "/" + string(name)
			if paths[path] {
				return fmt.Errorf("duplicate path %s", path)
			}
			paths[path] = true
			value, size, kind := u32(off+4), u32(off+8), u32(off+12)
			if kind&1 != 0 {
				if kind&14 != 0 {
					return fmt.Errorf("directory %s claims a value type", path)
				}
				if projectile1157Path(path) {
					s.dirs[path] = kind
				}
				if err := walk(value, size, path, depth+1); err != nil {
					return err
				}
				continue
			}
			if !projectile1157Path(path) {
				continue // Other application values are outside this instrument.
			}
			v := projectile1157Value{kind: kind}
			switch kind & 14 {
			case 2:
				v.word = value
			case 6:
				if size%4 != 0 || uint64(value)+uint64(size) > uint64(len(pool)) || uint64(size) > 64<<20-copied {
					return fmt.Errorf("array %s exceeds/alters pool extent", path)
				}
				copied += uint64(size)
				v.data = bytes.Clone(pool[uint64(value) : uint64(value)+uint64(size)])
			default:
				return fmt.Errorf("selected leaf %s has unsupported kind %#x", path, kind)
			}
			s.values[path] = v
		}
		return nil
	}
	if err := walk(u32(4), u32(8), "", 0); err != nil {
		return bad(err.Error())
	}
	for i, found := range seen {
		if !found {
			return bad(fmt.Sprintf("record %d is unreachable", i))
		}
	}
	free, hasFree := s.values["/Projectiles/FreeIndex"]
	ids, hasIDs := s.values["/Projectiles/IDs"]
	s.present = hasFree || hasIDs
	if hasFree {
		if free.kind&14 != 2 {
			return bad("FreeIndex is not an integer")
		}
		s.free = uint16(free.word)
	}
	if hasIDs {
		switch ids.kind & 14 {
		case 2:
			s.ids = []uint16{uint16(ids.word)}
		case 6:
			for off := 0; off < len(ids.data); off += 4 {
				s.ids = append(s.ids, uint16(binary.LittleEndian.Uint32(ids.data[off:])))
			}
		default:
			return bad("IDs is not an integer or integer array")
		}
	}
	unique := map[uint16]bool{}
	for _, id := range s.ids {
		if unique[id] {
			continue
		}
		unique[id] = true
		item := projectile1157Item{id: id}
		for i, name := range projectile1157Names {
			path := fmt.Sprintf("/Prj%d/%s", id, name)
			if v, present := s.values[path]; present {
				if v.kind&14 != 2 {
					return bad(path + " is not an integer")
				}
				item.fields[i] = int32(v.word)
			}
			// An absent leaf retains its absence in values. Zero here checks
			// this engine's declared default, not an original constructor.
		}
		s.items = append(s.items, item)
	}
	return s, nil
}

func (s projectile1157Source) worldDifferences(got sim.SavedProjectiles) []string {
	var differences []string
	if got.FreeIndex != s.free {
		differences = append(differences, "allocator differs")
	}
	if !slices.Equal(got.IDs, s.ids) {
		differences = append(differences, "ordered IDs differ")
	}
	if len(got.Items) != len(s.items) {
		differences = append(differences, "distinct item population differs")
	}
	for i, want := range s.items {
		if i >= len(got.Items) {
			break
		}
		p := got.Items[i]
		if p.ID != want.id {
			differences = append(differences, fmt.Sprintf("item %d identity/order differs", i))
		}
		fields := [...]int32{p.X, p.Y, p.Z, p.Picture, p.Dir, p.Phase, p.LastAction, p.Action,
			p.ActionDir, p.ActionTarget, p.ActionX, p.ActionY, p.ActionZ, p.ActionPhase, p.ActionSegments, p.ActionSpell}
		for j, value := range fields {
			if value != want.fields[j] {
				differences = append(differences, fmt.Sprintf("Prj%d/%s differs", want.id, projectile1157Names[j]))
			}
		}
	}
	return differences
}

func (s projectile1157Source) documentDifferences(state *SnapshotSAVDocument) []string {
	if state == nil || state.Unavailable != "" || state.Document == nil {
		return []string{"complete Document unavailable"}
	}
	var differences []string
	doc := state.Document.State
	if doc.RootKind != s.rootKind {
		differences = append(differences, "application root kind differs")
	}
	dirs, values := map[string]bool{}, map[string]bool{}
	for _, r := range doc.DirectoryRecords {
		if !projectile1157Path(r.Path) {
			continue
		}
		want, exists := s.dirs[r.Path]
		if !exists || dirs[r.Path] || r.Kind != want {
			differences = append(differences, "directory differs: "+r.Path)
		}
		dirs[r.Path] = true
	}
	for _, r := range doc.ValueRecords {
		if !projectile1157Path(r.Path) {
			continue
		}
		want, exists := s.values[r.Path]
		if !exists || values[r.Path] || r.Value.Kind != want.kind ||
			uint32(r.Value.Int32) != want.word || !bytes.Equal(r.Value.Bytes, want.data) {
			differences = append(differences, "leaf differs: "+r.Path)
		}
		values[r.Path] = true
	}
	for path := range s.dirs {
		if !dirs[path] {
			differences = append(differences, "missing directory: "+path)
		}
	}
	for path := range s.values {
		if !values[path] {
			differences = append(differences, "missing leaf: "+path)
		}
	}
	slices.Sort(differences)
	return differences
}
