package mod

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/res"
)

func treeContents(t *testing.T, tree Tree) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := tree.WalkDir(func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			out[p] = "directory"
			return nil
		}
		b, err := fs.ReadFile(tree.FS(), p)
		if err != nil {
			return err
		}
		inside, _, err := tree.ReadInside(p)
		if err != nil || !bytes.Equal(b, inside) {
			t.Fatalf("ReadInside %s: %q, %v; FS read %q", p, inside, err, b)
		}
		out[p] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPackLongNamesReadAsTheirFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "names")
	writeTreeFile(t, dir, "mod.toml", good("names"))
	for _, parent := range []string{"", "maps/", "data/"} {
		for n := 16; n <= 64; n++ {
			for _, name := range []string{strings.Repeat("a", n), strings.Repeat("я", n/2) + strings.Repeat("x", n%2)} {
				writeTreeFile(t, dir, parent+name+".star", parent+name+".star")
			}
		}
	}
	for _, name := range []string{"assets/items/pictures-of-the-east/日本.png", "maps/длинное-имя-карты.text/ru/briefing.txt", "data/weapon-bodies.toml", "~00000000000001.star", "maps/~00000000000002.star"} {
		body := name
		if name == WeaponBodiesLongFile {
			body = "# empty"
		}
		writeTreeFile(t, dir, name, body)
	}
	folder, err := OpenTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	packedPath := packInto(t, dir, t.TempDir())
	packed, err := OpenTree(packedPath)
	if err != nil {
		t.Fatal(err)
	}
	if want, got := treeContents(t, folder), treeContents(t, packed); !reflect.DeepEqual(got, want) {
		t.Fatalf("packed tree differs: got %v, want %v", got, want)
	}
	want, err := folder.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := packed.Digest(); err != nil || got != want {
		t.Fatalf("digest %s, %v; want %s", got, err, want)
	}
	if b, _, err := packed.ReadInside("MAPS/ДЛИННОЕ-ИМЯ-КАРТЫ.TEXT/RU/BRIEFING.TXT"); err != nil || string(b) != "maps/длинное-имя-карты.text/ru/briefing.txt" {
		t.Fatalf("folded long path: %q, %v", b, err)
	}
	_, first, err := Pack(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := Pack(dir)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("packing is not deterministic: %v", err)
	}
}

func TestPackedLegacyLayoutAndShortPackBytesStayUnchanged(t *testing.T) {
	files := []struct{ name, body string }{{"a.star", "old\n"}, {"mod.toml", "id = \"legacy\"\ntitle = \"Legacy\"\nversion = \"1\"\n"}}
	b := make([]byte, 24)
	var entries []res.WriteEntry
	for _, f := range files {
		b = append(b, f.body...)
		entries = append(entries, res.WriteEntry{Path: f.name, Data: []byte(f.body)})
	}
	le := binary.LittleEndian
	le.PutUint32(b[0:], 0x31415926)
	le.PutUint32(b[8:], 2)
	le.PutUint32(b[12:], 1)
	le.PutUint32(b[16:], uint32(len(b)))
	le.PutUint32(b[20:], 2)
	offset := 24
	for _, f := range files {
		rec := make([]byte, 32)
		le.PutUint32(rec[4:], uint32(offset))
		le.PutUint32(rec[8:], uint32(len(f.body)))
		copy(rec[16:], f.name)
		b = append(b, rec...)
		offset += len(f.body)
	}
	tree, err := openPackedBytes("mod-legacy.res", b)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{".": "directory"}
	h := sha256.New()
	dir := filepath.Join(t.TempDir(), "legacy")
	for _, f := range files {
		want[f.name] = f.body
		fmt.Fprintf(h, "%s\x00%d\x00%s", f.name, len(f.body), f.body)
		writeTreeFile(t, dir, f.name, f.body)
	}
	if got := treeContents(t, tree); !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy tree %v, want %v", got, want)
	}
	if got, err := tree.Digest(); err != nil || got != hex.EncodeToString(h.Sum(nil)) {
		t.Fatalf("legacy digest %s, %v", got, err)
	}
	_, got, err := Pack(dir)
	if err != nil || !bytes.Equal(got, b) {
		t.Fatalf("short pack differs from the legacy layout: %v", err)
	}
	if old, err := res.Write(entries); err != nil || !bytes.Equal(old, b) {
		t.Fatalf("legacy fixture differs from the container writer: %v", err)
	}
}

func TestPackNameBoundCountsUTF8Bytes(t *testing.T) {
	for _, name := range []string{strings.Repeat("a", 250) + ".star", strings.Repeat("я", 125) + ".star", strings.Repeat("я", 125) + "x.star"} {
		dir := filepath.Join(t.TempDir(), "names")
		writeTreeFile(t, dir, "mod.toml", good("names"))
		writeTreeFile(t, dir, name, "bounded")
		_, b, err := Pack(dir)
		if len(name) > 255 {
			if err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "256 UTF-8 bytes") {
				t.Fatalf("over the bound: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		tree, err := openPackedBytes("mod-names.res", b)
		if err != nil {
			t.Fatal(err)
		}
		if got, _, err := tree.ReadInside(name); err != nil || string(got) != "bounded" {
			t.Fatalf("at the bound: %q, %v", got, err)
		}
	}
}

func TestPackNameTableEntryIsReserved(t *testing.T) {
	for _, name := range []string{"mod-names.json", "MOD-NAMES.JSON"} {
		for _, directory := range []bool{false, true} {
			dir := filepath.Join(t.TempDir(), "names")
			writeTreeFile(t, dir, "mod.toml", good("names"))
			file := name
			if directory {
				file += "/a.txt"
			}
			writeTreeFile(t, dir, file, "author content")
			if e := Load(dir); e.Err == nil || !strings.Contains(e.Err.Error(), name+" is reserved") {
				t.Fatalf("folder table collision: %v", e.Err)
			}
			if _, _, err := Pack(dir); err == nil || !strings.Contains(err.Error(), name+" is reserved") {
				t.Fatalf("pack table collision: %v", err)
			}
		}
	}
}

func TestPackedNameTableRefusesMalformedMappings(t *testing.T) {
	long := strings.Repeat("a", 16)
	for _, c := range []struct {
		name  string
		table packedNames
	}{
		{"version", packedNames{Version: 2, Entries: []packedName{{"a", long}}}},
		{"empty", packedNames{Version: 1}},
		{"duplicate", packedNames{Version: 1, Entries: []packedName{{"a", long}, {"a", long}}}},
		{"missing", packedNames{Version: 1, Entries: []packedName{{"missing", long}}}},
		{"escape", packedNames{Version: 1, Entries: []packedName{{"../a", long}}}},
		{"separator", packedNames{Version: 1, Entries: []packedName{{"a", "x/y"}}}},
		{"backslash", packedNames{Version: 1, Entries: []packedName{{"a", `x\y`}}}},
		{"control", packedNames{Version: 1, Entries: []packedName{{"a", "x\x00y"}}}},
		{"dot", packedNames{Version: 1, Entries: []packedName{{"a", ".."}}}},
		{"bound", packedNames{Version: 1, Entries: []packedName{{"a", strings.Repeat("я", 128)}}}},
		{"reserved", packedNames{Version: 1, Entries: []packedName{{"a", "MOD-NAMES.JSON"}}}},
		{"fold twins", packedNames{Version: 1, Entries: []packedName{{"a", long}, {"b", strings.ToUpper(long)}}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			data, err := json.Marshal(c.table)
			if err != nil {
				t.Fatal(err)
			}
			b, err := res.Write([]res.WriteEntry{{Path: "mod-names.json", Data: data}, {Path: "a", Data: []byte("a")}, {Path: "b", Data: []byte("b")}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := openPackedBytes("bad.res", b); err == nil {
				t.Fatal("opened a malformed table")
			}
		})
	}
	for _, data := range []string{"author content", `{"version":1,"entries":[]} trailing`, "\xff"} {
		b, err := res.Write([]res.WriteEntry{{Path: "mod-names.json", Data: []byte(data)}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := openPackedBytes("bad.res", b); err == nil || !strings.Contains(err.Error(), "mod-names.json") {
			t.Fatalf("reserved entry: %v", err)
		}
	}
	for _, names := range [][2]string{{long, strings.ToUpper(long)}, {strings.Repeat("я", 8), strings.Repeat("Я", 8)}} {
		if err := refuseFoldTwins("names", []res.WriteEntry{{Path: "maps/" + names[0]}, {Path: "maps/" + names[1]}}); err == nil || !strings.Contains(err.Error(), names[0]) || !strings.Contains(err.Error(), names[1]) {
			t.Fatalf("fold twins: %v", err)
		}
	}
}
