package game

import (
	"encoding/binary"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// a2cFile is a second-game character file from the rom2-a2c format page:
// the magic, section A's header and its 52 bytes scrambled under key, then
// B's tag. bad selects a broken magic or a wrong checksum.
func a2cFile(id0, id1 uint32, name string, key uint16, bad string) []byte {
	a := make([]byte, 52)
	binary.LittleEndian.PutUint32(a, id0)
	binary.LittleEndian.PutUint32(a[4:], id1)
	copy(a[0x0c:0x2b], name)
	var sum uint32
	for _, c := range a {
		sum = sum*2 + uint32(c)
	}
	if bad == "checksum" {
		sum++
	}
	magic := uint32(0x04507989)
	if bad == "magic" {
		magic = 0x68436c42
	}
	f := binary.LittleEndian.AppendUint32(nil, magic)
	f = binary.LittleEndian.AppendUint32(f, 0xAAAAAAAA)
	f = binary.LittleEndian.AppendUint32(f, 52)
	f = binary.LittleEndian.AppendUint32(f, uint32(key)<<16)
	f = binary.LittleEndian.AppendUint32(f, sum)
	reg := uint32(key)<<16 | uint32(key)
	for i, c := range a {
		f = append(f, c^uint8(reg>>16))
		reg <<= 1
		if i%16 == 15 {
			reg |= uint32(key)
		}
	}
	return binary.LittleEndian.AppendUint32(f, 0x55555555)
}

// alchFile is a first-game network character file from the hero format
// page: AlCh, 8 bytes, the 32-byte name, 32 bytes, flags with bit 0 set, 4
// bytes and two 28-byte blocks; size cuts it.
func alchFile(name string, size int) []byte {
	f := []byte("AlCh")
	f = append(f, make([]byte, 8)...)
	n := make([]byte, 32)
	copy(n, name)
	f = append(f, n...)
	f = append(f, make([]byte, 32)...)
	f = binary.LittleEndian.AppendUint32(f, 1)
	f = append(f, make([]byte, 4+56)...)
	return f[:size]
}

type storedFile struct {
	name string
	b    []byte
	age  time.Duration
}

func writeCharacters(t *testing.T, dir string, files []storedFile) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	base := time.Unix(1_700_000_000, 0)
	for _, f := range files {
		path := filepath.Join(dir, f.name)
		if err := os.WriteFile(path, f.b, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, base.Add(-f.age), base.Add(-f.age)); err != nil {
			t.Fatal(err)
		}
	}
}

// treeState is every path under root, with a file's time and bytes.
func treeState(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			out[path] = "folder"
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[path] = info.ModTime().String() + string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func storeFront(t *testing.T, id string) *FrontEnd {
	t.Helper()
	profile, ok := base.Find(id)
	if !ok {
		t.Fatalf("missing profile %s", id)
	}
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Base: base.Match{Profile: profile}}}}
	f.Words.Engine = words.For("english")
	return f
}

func labels(rows []ui.SaveEntry) (names, labels []string) {
	for _, r := range rows {
		names, labels = append(names, r.Name), append(labels, r.Label)
	}
	return names, labels
}

// Each game lists its own folder of the profile's character folder: the
// files its decoder accepts by name, newest first, with the count of files
// left out; other extensions and folders are not read, and listing writes
// nothing. The first game's Delete removes the chosen file alone.
func TestCharacterStoreListsEachGamesFolder(t *testing.T) {
	profile := t.TempDir()
	store := CharacterStore{Dir: filepath.Join(profile, "characters")}
	writeCharacters(t, filepath.Join(store.Dir, "rom2"), []storedFile{
		{"10.a2c", a2cFile(1, 0, "Alpha", 0x1234, ""), 3 * time.Hour},
		{"20.a2c", a2cFile(2, 0, "Beta", 0x7fff, ""), time.Hour},
		{"30.A2C", a2cFile(3, 0, "Gamma", 0x0001, ""), 2 * time.Hour},
		{"40.a2c", a2cFile(4, 0, "Sum", 0x2222, "checksum"), 0},
		{"50.a2c", a2cFile(5, 0, "Magic", 0x2222, "magic"), 0},
		{"60.a2c", nil, 0},
		{"game0000.chr", alchFile("First", 0x8c), 0},
		{"notes.txt", []byte("x"), 0},
	})
	if err := os.Mkdir(filepath.Join(store.Dir, "rom2", "70.a2c"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeCharacters(t, filepath.Join(store.Dir, "rom1"), []storedFile{
		{"game0000.chr", alchFile("Gwendolyn", 0x8c), time.Hour},
		{"game0001.chr", alchFile("Bors", 0x8c+40), 2 * time.Hour},
		{"game0002.chr", alchFile("Short", 0x8b), 0},
		{"game0003.chr", a2cFile(6, 0, "Wrong", 0x1111, ""), 0},
		{"10.a2c", a2cFile(1, 0, "Alpha", 0x1234, ""), 0},
	})
	before := treeState(t, profile)
	second := storeFront(t, base.ROM2EN).characterFiles(store)
	rows, message := second.List()
	if names, got := labels(rows); !slices.Equal(got, []string{"Beta", "Gamma", "Alpha"}) ||
		!slices.Equal(names, []string{"20.a2c", "30.A2C", "10.a2c"}) || message != "Unreadable character files left out: 3" {
		t.Errorf("second game rows %q %q message %q", names, got, message)
	}
	if second.CanDelete != nil || second.PrepareDelete != nil {
		t.Errorf("the second game's store deletes")
	}
	first := storeFront(t, base.ROM1EN).characterFiles(store)
	rows, message = first.List()
	if names, got := labels(rows); !slices.Equal(got, []string{"Gwendolyn", "Bors"}) ||
		!slices.Equal(names, []string{"game0000.chr", "game0001.chr"}) || message != "Unreadable character files left out: 2" {
		t.Errorf("first game rows %q %q message %q", names, got, message)
	}
	if after := treeState(t, profile); len(after) != len(before) {
		t.Fatalf("listing changed the profile: %d paths, was %d", len(after), len(before))
	} else {
		for path, state := range before {
			if after[path] != state {
				t.Fatalf("listing changed %s", path)
			}
		}
	}
	for _, name := range []string{"10.a2c", "../rom2/20.a2c", "missing.chr", ""} {
		if first.CanDelete(name) {
			t.Errorf("first game deletes %q", name)
		}
	}
	remove, err := first.PrepareDelete("game0001.chr")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "rom1", "game0001.chr")); err != nil {
		t.Fatalf("preparing removed the file: %v", err)
	}
	if err := remove(); err != nil {
		t.Fatal(err)
	}
	after := treeState(t, profile)
	gone := filepath.Join(store.Dir, "rom1", "game0001.chr")
	for path, state := range before {
		if path == gone {
			if _, ok := after[path]; ok {
				t.Errorf("Delete left %s", path)
			}
		} else if after[path] != state {
			t.Errorf("Delete changed %s", path)
		}
	}
	if len(after) != len(before)-1 {
		t.Errorf("Delete left %d paths, want %d", len(after), len(before)-1)
	}
}

// An absent character folder lists nothing and is not created.
func TestCharacterStoreAbsentFolderListsNothing(t *testing.T) {
	profile := t.TempDir()
	store := CharacterStore{Dir: filepath.Join(profile, "characters")}
	for _, id := range []string{base.ROM1EN, base.ROM2EN} {
		rows, message := storeFront(t, id).characterFiles(store).List()
		if len(rows) != 0 || message != "" {
			t.Errorf("%s: rows %v message %q", id, rows, message)
		}
	}
	if _, err := os.Stat(store.Dir); !os.IsNotExist(err) {
		t.Fatalf("listing created %s: %v", store.Dir, err)
	}
}

// The profile builder names the character folder beside the saves.
func TestRuntimeProfileNamesTheCharacterFolder(t *testing.T) {
	dir := t.TempDir()
	p := runtimeProfileStores(filepath.Join(dir, "saves"), "", nil)
	if p.Characters.Dir != filepath.Join(dir, "characters") {
		t.Fatalf("character folder %q", p.Characters.Dir)
	}
	if p.Characters.path(firstCharacterFiles) != filepath.Join(dir, "characters", "rom1") ||
		p.Characters.path(secondCharacterFiles) != filepath.Join(dir, "characters", "rom2") {
		t.Fatalf("game folders %q %q", p.Characters.path(firstCharacterFiles), p.Characters.path(secondCharacterFiles))
	}
}
