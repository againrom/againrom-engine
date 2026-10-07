package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// These fixtures construct the YA1 record table directly. Neither the city nor
// world shape/serializer supplies a field name, count, type or expected value.
type literalStateLeaf struct {
	name string
	kind uint32
	word uint32
	data []byte
}

type literalStateDirectory struct {
	name   string
	leaves []literalStateLeaf
}

func literalStateWords(words ...uint32) []byte {
	b := make([]byte, 4*len(words))
	for i, word := range words {
		binary.LittleEndian.PutUint32(b[4*i:], word)
	}
	return b
}

func literalWorldDirectories(ids ...uint32) []literalStateDirectory {
	dirs := []literalStateDirectory{
		{"Character", []literalStateLeaf{{name: "Name", kind: 0, data: []byte{'B', 0xe0, 'r', 'a', 0}}}},
		{"CurrentState", []literalStateLeaf{{name: "InBattle", kind: 2, word: 3}}},
		{"Fog", []literalStateLeaf{{name: "FirstState", kind: 2, word: 0x8000}, {name: "Data", kind: 6, data: literalStateWords(0, 3, 5)}}},
		{"GameOptions", []literalStateLeaf{
			{name: "Wimpy", kind: 2, word: 2}, {name: "Speed", kind: 2, word: 12},
			{name: "Formation", kind: 2, word: 9}, {name: "FlyingHP", kind: 2, word: 0xfffffff9},
			{name: "ShowHP", kind: 2, word: 1}, {name: "ShowTimeFlow", kind: 2, word: 0}}},
		{"Inventory", []literalStateLeaf{{name: "IsOpen", kind: 2, word: 1}}},
		{"Objects", []literalStateLeaf{{name: "Selection", kind: 6, data: literalStateWords(100, 0, 0xffffffff)}}},
		{"Projectiles", []literalStateLeaf{{name: "FreeIndex", kind: 2, word: 0x1234}, {name: "IDs", kind: 6, data: literalStateWords(ids...)}}},
		{"SpellBook", []literalStateLeaf{{name: "IsOpen", kind: 2, word: 1}, {name: "Pressed", kind: 2, word: 5},
			{name: "Shortcuts", kind: 6, data: literalStateWords(0xffffffff, 7, 13, 21)}}},
		{"View", []literalStateLeaf{{name: "X", kind: 2, word: 0xffffff85}, {name: "Y", kind: 2, word: 456}}},
	}
	seen := map[uint16]bool{}
	for _, value := range ids {
		id := uint16(value)
		if seen[id] {
			continue
		}
		seen[id] = true
		d := literalStateDirectory{name: fmt.Sprintf("Prj%d", id)}
		for i, name := range []string{"x", "y", "z", "picture", "dir", "phase", "lastaction", "action", "actiondir", "actiontarget", "actionx", "actiony", "actionz", "actionphase", "actionsegments", "actionspell"} {
			d.leaves = append(d.leaves, literalStateLeaf{name: name, kind: 2, word: uint32(id)*100 + uint32(i)})
		}
		dirs = append(dirs, d)
	}
	return dirs
}

// worldState1115Fixture is shared by the envelope tests. Its input is the
// independent literal YA1 builder above, including two complete Prj sections.
func worldState1115Fixture(t *testing.T) *cityState {
	t.Helper()
	state, err := parseWorldState(literalStateStore(literalWorldDirectories(266, 7, 266)))
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func literalStateStore(dirs []literalStateDirectory) []byte {
	n := len(dirs)
	for _, d := range dirs {
		n += len(d.leaves)
	}
	raw := make([]byte, 24+32*n)
	copy(raw, "&YA1")
	binary.LittleEndian.PutUint32(raw[8:], uint32(len(dirs)))
	binary.LittleEndian.PutUint32(raw[12:], 17)
	binary.LittleEndian.PutUint32(raw[16:], uint32(n))
	// Unused transport words and pool debris must not become replay material.
	binary.LittleEndian.PutUint32(raw[20:], 0xdec0de11)
	pool := []byte{0xe1, 0xe2, 0xe3}
	write := func(index int, name string, value, size, kind uint32) {
		o := 24 + index*32
		binary.LittleEndian.PutUint32(raw[o:], 0xf00d1234)
		binary.LittleEndian.PutUint32(raw[o+4:], value)
		binary.LittleEndian.PutUint32(raw[o+8:], size)
		binary.LittleEndian.PutUint32(raw[o+12:], kind)
		copy(raw[o+16:o+32], name)
	}
	child := len(dirs)
	for i, d := range dirs {
		write(i, d.name, uint32(child), uint32(len(d.leaves)), 17)
		for _, leaf := range d.leaves {
			value, size := leaf.word, uint32(0)
			if leaf.kind == 0 || leaf.kind == 6 {
				value, size = uint32(len(pool)), uint32(len(leaf.data))
				pool = append(pool, leaf.data...)
			}
			write(child, leaf.name, value, size, leaf.kind)
			child++
		}
	}
	raw = binary.LittleEndian.AppendUint32(raw, uint32(len(pool)))
	return append(raw, pool...)
}

type literalStateValue struct {
	kind uint32
	word uint32
	data string
}

// Independent shallow YA1 oracle over the test's already bounded tables.
func literalReadState(raw []byte) (map[string]literalStateValue, map[string]int) {
	n := int(binary.LittleEndian.Uint32(raw[16:]))
	pool := raw[24+32*n+4:]
	name := func(i int) string {
		b := raw[24+32*i+16 : 24+32*i+32]
		if end := bytes.IndexByte(b, 0); end >= 0 {
			b = b[:end]
		}
		return string(b)
	}
	values, offsets := map[string]literalStateValue{}, map[string]int{}
	for i := 0; i < int(binary.LittleEndian.Uint32(raw[8:])); i++ {
		o := 24 + 32*i
		start, count := int(binary.LittleEndian.Uint32(raw[o+4:])), int(binary.LittleEndian.Uint32(raw[o+8:]))
		offsets["/"+name(i)] = o
		for j := start; j < start+count; j++ {
			p := 24 + 32*j
			path := "/" + name(i) + "/" + name(j)
			v := literalStateValue{kind: binary.LittleEndian.Uint32(raw[p+12:]), word: binary.LittleEndian.Uint32(raw[p+4:])}
			if v.kind == 0 || v.kind == 6 {
				size := binary.LittleEndian.Uint32(raw[p+8:])
				v.data = string(pool[v.word : v.word+size])
				v.word = 0
			}
			values[path], offsets[path] = v, p
		}
	}
	return values, offsets
}

func TestWorldStateLiteralEmptyAndProjectileValues(t *testing.T) {
	for _, tc := range []struct {
		name           string
		ids            []uint32
		roots, records uint32
	}{{"empty", nil, 9, 28}, {"singleton", []uint32{266}, 10, 45},
		{"ordered-repeated-low-word", []uint32{266, 0x12340007, 266}, 11, 62}} {
		t.Run(tc.name, func(t *testing.T) {
			raw := literalStateStore(literalWorldDirectories(tc.ids...))
			want, _ := literalReadState(raw)
			state, err := parseWorldState(raw)
			if err != nil {
				t.Fatal(err)
			}
			for path, expected := range want {
				got := state.values[path]
				if got.kind != expected.kind || (got.kind == 2 && uint32(got.int32) != expected.word) || (got.kind != 2 && string(got.bytes) != expected.data) {
					t.Fatalf("%s: decoded %+v, want %+v", path, got, expected)
				}
			}
			for i := range raw {
				raw[i] ^= 0xff
			} // Every value must already be detached from the input.
			encoded, err := serializeWorldState(state)
			if err != nil {
				t.Fatal(err)
			}
			if binary.LittleEndian.Uint32(encoded[8:]) != tc.roots || binary.LittleEndian.Uint32(encoded[16:]) != tc.records {
				t.Fatalf("world counts differ: roots=%d records=%d", binary.LittleEndian.Uint32(encoded[8:]), binary.LittleEndian.Uint32(encoded[16:]))
			}
			if size, err := worldStateSerializedSize(state); err != nil || size != uint64(len(encoded)) {
				t.Fatalf("preflight size=%d error=%v, wrote %d bytes", size, err, len(encoded))
			}
			got, _ := literalReadState(encoded)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("writer lost typed values:\ngot %v\nwant %v", got, want)
			}
			if binary.LittleEndian.Uint32(encoded[20:]) != 0 {
				t.Fatal("writer replayed transport header debris")
			}
			second, err := serializeWorldState(state)
			if err != nil || !bytes.Equal(second, encoded) {
				t.Fatalf("non-deterministic writer: %v", err)
			}
			decoded, err := parseWorldState(encoded)
			if err != nil {
				t.Fatal(err)
			}
			v := decoded.values["/View/X"]
			v.int32 = -987
			decoded.values["/View/X"] = v
			updated, err := serializeWorldState(decoded)
			if err != nil {
				t.Fatal(err)
			}
			changed, _ := literalReadState(updated)
			if changed["/View/X"].word != uint32(0xfffffc25) {
				t.Fatal("writer did not use changed detached state")
			}
		})
	}
}

func TestWorldStateKnownProjectileLoaderForms(t *testing.T) {
	for _, name := range []string{"missing-section", "missing-IDs", "missing-FreeIndex", "singleton-integer"} {
		t.Run(name, func(t *testing.T) {
			dirs := literalWorldDirectories()
			switch name {
			case "missing-section":
				dirs = append(dirs[:6], dirs[7:]...)
			case "missing-IDs":
				dirs[6].leaves = dirs[6].leaves[:1]
			case "missing-FreeIndex":
				dirs[6].leaves = dirs[6].leaves[1:]
			case "singleton-integer":
				dirs = literalWorldDirectories(266)
				dirs[6].leaves[1] = literalStateLeaf{name: "IDs", kind: 2, word: 266}
			}
			raw := literalStateStore(dirs)
			want, _ := literalReadState(raw)
			state, err := parseWorldState(raw)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := serializeWorldState(state)
			if err != nil {
				t.Fatal(err)
			}
			got, _ := literalReadState(encoded)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("compatibility form changed: got %v want %v", got, want)
			}
		})
	}
}

func TestWorldStateRejectsMalformedAndIncompleteInputs(t *testing.T) {
	raw := literalStateStore(literalWorldDirectories(266))
	for end := 0; end < len(raw); end++ {
		if state, err := parseWorldState(raw[:end]); err == nil || state != nil {
			t.Fatalf("truncation %d returned partial state=%v error=%v", end, state, err)
		}
	}
	_, offsets := literalReadState(raw)
	for _, tc := range []struct {
		name   string
		mutate func([]byte)
	}{
		{"record-count", func(b []byte) { binary.LittleEndian.PutUint32(b[16:], 0xffffffff) }},
		{"root-count", func(b []byte) { binary.LittleEndian.PutUint32(b[8:], 0xffffffff) }},
		{"root-kind", func(b []byte) { binary.LittleEndian.PutUint32(b[12:], 1) }},
		{"overlap", func(b []byte) { binary.LittleEndian.PutUint32(b[offsets["/Fog"]+4:], 0) }},
		{"nested-directory", func(b []byte) { binary.LittleEndian.PutUint32(b[offsets["/Fog/Data"]+12:], 1) }},
		{"unaligned-array", func(b []byte) { binary.LittleEndian.PutUint32(b[offsets["/Fog/Data"]+8:], 3) }},
		{"wrong-IDs-type", func(b []byte) { binary.LittleEndian.PutUint32(b[offsets["/Projectiles/IDs"]+12:], 4) }},
		{"orphan-projectile", func(b []byte) { copy(b[offsets["/Prj266"]+16:], "Prj267") }},
		{"missing-projectile-field", func(b []byte) { b[offsets["/Prj266/actionspell"]+16] = 'X' }},
		{"duplicate-path", func(b []byte) { p := offsets["/View/Y"] + 16; b[p] = 'X' }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := append([]byte(nil), raw...)
			tc.mutate(input)
			if state, err := parseWorldState(input); err == nil || state != nil {
				t.Fatalf("returned state=%v error=%v", state, err)
			}
		})
	}
	state, err := parseWorldState(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*cityState)
	}{
		{"missing-field", func(s *cityState) { delete(s.values, "/Prj266/actionspell") }},
		{"extra-field", func(s *cityState) { s.values["/Prj266/extra"] = cityStateValue{kind: 2} }},
		{"unaligned-selection", func(s *cityState) { s.values["/Objects/Selection"] = cityStateValue{kind: 6, bytes: []byte{1}} }},
		{"oversize-ID-count", func(s *cityState) {
			s.values["/Projectiles/IDs"] = cityStateValue{kind: 6, bytes: make([]byte, 4*65537)}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := cloneCityState(state)
			tc.mutate(bad)
			if out, err := serializeWorldState(bad); err == nil || out != nil {
				t.Fatalf("returned %d output bytes error=%v", len(out), err)
			}
		})
	}
}

func TestWorldStateDoesNotRelaxCityShape(t *testing.T) {
	city, err := serializeCityState(cityTestState("City"))
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(city[8:]) != 7 || binary.LittleEndian.Uint32(city[16:]) != 22 {
		t.Fatal("city counts changed")
	}
	if _, err := parseCityState(city); err != nil {
		t.Fatal(err)
	}
	if _, err := parseWorldState(city); err == nil {
		t.Fatal("city without Fog admitted as complete world state")
	}
	world := literalStateStore(literalWorldDirectories())
	state, err := parseWorldState(world)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseCityState(world); err == nil {
		t.Fatal("city reader admitted world roots")
	}
	if _, err := serializeCityState(state); err == nil {
		t.Fatal("city writer admitted world roots")
	}
}

func TestWorldStateBoundsPoolBeforeSerialization(t *testing.T) {
	state := worldState1115Fixture(t)
	// Both typed leaves share one allocation here; the serializer would have to
	// copy each into its own pool run. Their combined extent exceeds the budget.
	large := make([]byte, maxWorldStateBytes/2)
	state.values["/Objects/Selection"] = cityStateValue{kind: 6, bytes: large}
	state.values["/Fog/Data"] = cityStateValue{kind: 6, bytes: large}
	if _, err := worldStateSerializedSize(state); err == nil {
		t.Fatal("combined leaf budget was not checked")
	}
	if out, err := serializeWorldState(state); err == nil || out != nil {
		t.Fatalf("oversized state returned %d bytes error=%v", len(out), err)
	}
	if _, err := worldStateSerializedSize(nil); err == nil {
		t.Fatal("nil state accepted by size preflight")
	}
}

func TestStateStoreRepeatedPoolAliasesAreBoundedBeforeDetachment(t *testing.T) {
	for _, world := range []bool{false, true} {
		name := "city"
		if world {
			name = "world"
		}
		t.Run(name, func(t *testing.T) {
			dirs := literalWorldDirectories()
			if !world {
				kept := dirs[:0]
				for _, dir := range dirs {
					if dir.name != "Fog" && dir.name != "Projectiles" {
						kept = append(kept, dir)
					}
				}
				dirs = kept
			}
			literal := literalStateStore(dirs)
			roots := int(binary.LittleEndian.Uint32(literal[8:]))
			n := int(binary.LittleEndian.Uint32(literal[16:]))
			tableEnd := 24 + 32*n
			const poolBytes = 8 << 20
			raw := make([]byte, tableEnd+4+poolBytes)
			copy(raw, literal[:tableEnd])
			binary.LittleEndian.PutUint32(raw[tableEnd:], poolBytes)
			// All 15/19 leaves reference the same valid, aligned eight-MiB
			// span. No source bound is violated, but copying each alias would
			// retain 120/152 MiB from an input smaller than nine MiB.
			for i := roots; i < n; i++ {
				off := 24 + 32*i
				binary.LittleEndian.PutUint32(raw[off+4:], 0)
				binary.LittleEndian.PutUint32(raw[off+8:], poolBytes)
				binary.LittleEndian.PutUint32(raw[off+12:], 6)
			}
			parse := parseCityState
			if world {
				parse = parseWorldState
			}
			state, err := parse(raw)
			if state != nil || err == nil || !strings.Contains(err.Error(), "declared detached state bytes") {
				t.Fatalf("expected preflight refusal before record/value allocation, state=%v error=%v", state, err)
			}
		})
	}
}

func TestWorldStateSmallPoolAliasesRemainDetached(t *testing.T) {
	raw := literalStateStore(literalWorldDirectories())
	_, offsets := literalReadState(raw)
	fog, selection := offsets["/Fog/Data"], offsets["/Objects/Selection"]
	copy(raw[selection+4:selection+12], raw[fog+4:fog+12])
	state, err := parseWorldState(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(state.values["/Fog/Data"].bytes, state.values["/Objects/Selection"].bytes) {
		t.Fatal("valid source alias did not produce equal values")
	}
	state.values["/Fog/Data"].bytes[0] = 99
	if state.values["/Objects/Selection"].bytes[0] != 0 {
		t.Fatal("two detached leaves still alias one retained allocation")
	}
}
