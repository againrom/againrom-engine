package game

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type failNamedPublish1173 struct {
	osNamedSaveFiles
	failAt, calls int
}

func parsedSaveName1173(t *testing.T, value string) validatedSaveName {
	t.Helper()
	name, err := namedSaveBase(value)
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func (f *failNamedPublish1173) Publish(a, b string) (savePublishResult, error) {
	f.calls++
	if f.calls == f.failAt {
		return savePublishResult{}, errors.New("injected publication failure")
	}
	return f.osNamedSaveFiles.Publish(a, b)
}

func (f *failNamedPublish1173) Replace(a, b string) error {
	f.calls++
	if f.calls == f.failAt {
		return errors.New("injected replacement failure")
	}
	return f.osNamedSaveFiles.Replace(a, b)
}

func TestNamedSave1173PairRollbackAndExactConfirmation(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "overwrite"}[existing], func(t *testing.T) {
			dir := t.TempDir()
			paths := []string{filepath.Join(dir, "My save.ags"), filepath.Join(dir, "My save.sav")}
			if existing {
				for _, path := range paths {
					if err := os.WriteFile(path, []byte("invalid but occupied "+filepath.Ext(path)), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			files := &failNamedPublish1173{failAt: 2}
			p, err := prepareNamedSave(dir, parsedSaveName1173(t, "My save.ags"), []namedSavePayload{{".ags", []byte("AGS")}, {".sav", []byte("SAV")}}, nil, files)
			if err != nil || !reflect.DeepEqual(p.Paths, paths) || len(p.Existing) != map[bool]int{false: 0, true: 2}[existing] {
				t.Fatalf("prepare: %+v %v", p, err)
			}
			if existing {
				if _, err := p.Commit(false); err == nil || files.calls != 0 {
					t.Fatal("unconfirmed overwrite wrote files")
				}
			}
			if _, err := p.Commit(existing); err == nil {
				t.Fatal("second publication failure was swallowed")
			}
			for _, path := range paths {
				raw, err := os.ReadFile(path)
				if existing && (err != nil || string(raw) != "invalid but occupied "+filepath.Ext(path)) || !existing && !os.IsNotExist(err) {
					t.Fatalf("partial pair at %s: %q %v", path, raw, err)
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != len(p.Existing) {
				t.Fatalf("transaction residue: %v %v", entries, err)
			}
			files.failAt = 0
			if _, err := p.Commit(existing); err != nil {
				// Replacement rollback creates a new file identity, so the old
				// confirmation must be re-prepared after that failed request.
				if !existing {
					t.Fatal(err)
				}
				p, err = prepareNamedSave(dir, parsedSaveName1173(t, "My save"), []namedSavePayload{{".ags", []byte("AGS")}, {".sav", []byte("SAV")}}, nil, files)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := p.Commit(true); err != nil {
					t.Fatal(err)
				}
			}
			for i, path := range paths {
				raw, err := os.ReadFile(path)
				if err != nil || string(raw) != []string{"AGS", "SAV"}[i] {
					t.Fatalf("successful pair: %q %v", raw, err)
				}
			}
		})
	}
}

func TestNamedSave1173CancelChangedTargetAndNames(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "nested")
	list, err := listSaveDirectory(dir, nil, 0)
	if err != nil || list.Path != dir {
		t.Fatal(list, err)
	}
	p, err := prepareNamedSave(dir, parsedSaveName1173(t, "Сохранение"), []namedSavePayload{{".ags", []byte("first")}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("browsing/preparation created directories")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Paths[0], []byte("other writer"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Commit(true); err == nil {
		t.Fatal("late target accepted without fresh confirmation")
	}
	p, err = prepareNamedSave(dir, parsedSaveName1173(t, "Сохранение"), []namedSavePayload{{".ags", []byte("first")}}, nil, nil)
	if err != nil || len(p.Existing) != 1 {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Paths[0], []byte("changed same inode"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Commit(true); err == nil {
		t.Fatal("changed contents accepted under old confirmation")
	}
	for _, name := range []string{"", "../escape", `..\escape`, "con.sav", "LPT1", "a:", "a.", "a ", "/abs"} {
		if _, err := namedSaveBase(name); err == nil {
			t.Errorf("unsafe filename %q accepted", name)
		}
	}
	if _, err := prepareNamedSave(dir, parsedSaveName1173(t, "safe"), []namedSavePayload{{".ags", []byte("first")}}, []string{filepath.Dir(dir)}, nil); err == nil {
		t.Fatal("configured read-only directory fence accepted")
	}
}
