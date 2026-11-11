package game

import (
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/fame"
)

// The host-file record: what every writer of a kept file leaves on disk in
// synthetic folders, for a write, an existing destination, a fence refusal and
// a failure at each step of the injectable file seams. The record is compared
// with testdata/hostfile_record.txt; AGAINROM_WRITE_HOSTFILE_RECORD=1 rewrites
// it.
func TestHostFileRecord(t *testing.T) {
	var r hostRecord
	r.saves(t)
	r.namedSets(t)
	r.preferences(t)
	r.modFiles(t)
	r.editorFiles(t)
	r.testPlayFiles(t)
	compareHostRecord(t, filepath.Join("testdata", "hostfile_record.txt"), r.b.String())
}

func compareHostRecord(t *testing.T, golden, got string) {
	t.Helper()
	if os.Getenv("AGAINROM_WRITE_HOSTFILE_RECORD") == "1" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		w, g := strings.Split(string(want), "\n"), strings.Split(got, "\n")
		for i := 0; i < len(w) || i < len(g); i++ {
			var a, b string
			if i < len(w) {
				a = w[i]
			}
			if i < len(g) {
				b = g[i]
			}
			if a != b {
				t.Fatalf("host-file record differs at line %d:\nrecorded %q\nnow      %q", i+1, a, b)
			}
		}
	}
}

type hostRecord struct {
	b     strings.Builder
	roots []string
}

var hostRecordDigits = regexp.MustCompile(`[0-9]{5,}`)

// root is a new synthetic folder whose spelling the record writes as $T.
func (r *hostRecord) root(t *testing.T) string {
	dir := t.TempDir()
	r.roots = append(r.roots, dir)
	if real, err := filepath.EvalSymlinks(dir); err == nil && real != dir {
		r.roots = append(r.roots, real)
	}
	return dir
}

func (r *hostRecord) norm(s string) string {
	for i := len(r.roots) - 1; i >= 0; i-- {
		s = strings.ReplaceAll(s, r.roots[i], "$T")
		s = strings.ReplaceAll(s, filepath.ToSlash(r.roots[i]), "$T")
	}
	// A temporary save's name carries the process ID, of any length.
	s = strings.ReplaceAll(s, ".tmp-"+strconv.Itoa(os.Getpid())+"-", ".tmp-#-")
	return hostRecordDigits.ReplaceAllString(filepath.ToSlash(s), "#")
}

func (r *hostRecord) line(format string, args ...any) {
	r.b.WriteString(r.norm(fmt.Sprintf(format, args...)) + "\n")
}

func (r *hostRecord) err(what string, err error) {
	if err == nil {
		r.line("%s: ok", what)
		return
	}
	fence := strings.HasPrefix(err.Error(), "refusing to write original-format save under") ||
		strings.Contains(err.Error(), ": refusing to write original-format save under")
	r.line("%s: error fence=%v %s", what, fence, err.Error())
}

// tree is every entry under dir, sorted, with each file's size and bytes.
func (r *hostRecord) tree(dir string) {
	var rows []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == dir {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			rows = append(rows, "  "+filepath.ToSlash(rel)+"/")
			return nil
		}
		b, _ := os.ReadFile(p)
		info, _ := d.Info()
		body := string(b)
		if len(body) > 24 {
			body = fmt.Sprintf("%.24s...", body)
		}
		rows = append(rows, fmt.Sprintf("  %s %d %v %q", filepath.ToSlash(rel), len(b), info.Mode().Perm()&0o700, body))
		return nil
	})
	sort.Strings(rows)
	for _, row := range rows {
		r.line("%s", row)
	}
}

// plantInstall makes dir hold the archives that mark a game install.
func plantInstall(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{MainArchive, GraphicsArchive, ScenarioArchive, WorldArchive, MoviesArchive} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func plantFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// recordRemoveRefused refuses to remove a hidden temporary save.
type recordRemoveRefused struct{ osSaveFileOps }

func (recordRemoveRefused) Remove(path string) error {
	if strings.Contains(filepath.Base(path), ".tmp-") {
		return errors.New("planted remove refusal")
	}
	return os.Remove(path)
}

// recordFaults fails the n-th call of a named file operation.
type recordFaults struct {
	osNamedSaveFiles
	fail  map[string]int
	calls map[string]int
}

func faults(fail map[string]int) *recordFaults {
	return &recordFaults{fail: fail, calls: map[string]int{}}
}

func (f *recordFaults) hit(op string) error {
	f.calls[op]++
	if n, ok := f.fail[op]; ok && n == f.calls[op] {
		return fmt.Errorf("planted %s failure", op)
	}
	return nil
}

func (f *recordFaults) CreateTemp(dir, pattern string) (saveTempFile, error) {
	if err := f.hit("create"); err != nil {
		return nil, err
	}
	tmp, err := f.osNamedSaveFiles.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &recordTemp{saveTempFile: tmp, f: f}, nil
}

func (f *recordFaults) Publish(oldPath, newPath string) (savePublishResult, error) {
	if err := f.hit("publish"); err != nil {
		return savePublishResult{}, err
	}
	return f.osNamedSaveFiles.Publish(oldPath, newPath)
}

func (f *recordFaults) Replace(oldPath, newPath string) error {
	if err := f.hit("replace"); err != nil {
		return err
	}
	return f.osNamedSaveFiles.Replace(oldPath, newPath)
}

func (f *recordFaults) Remove(path string) error {
	if err := f.hit("remove"); err != nil {
		return err
	}
	return f.osNamedSaveFiles.Remove(path)
}

type recordTemp struct {
	saveTempFile
	f *recordFaults
}

func (t *recordTemp) Write(b []byte) (int, error) {
	if err := t.f.hit("write"); err != nil {
		return 0, err
	}
	return t.saveTempFile.Write(b)
}

func (t *recordTemp) Sync() error {
	if err := t.f.hit("sync"); err != nil {
		return err
	}
	return t.saveTempFile.Sync()
}

func (t *recordTemp) Close() error {
	err := t.saveTempFile.Close()
	if hit := t.f.hit("close"); hit != nil {
		return hit
	}
	return err
}

func (r *hostRecord) saves(t *testing.T) {
	r.line("## single save")
	dir := r.root(t)
	for i := 0; i < 2; i++ {
		name, err := SaveStore{Dir: dir}.WriteOriginal("", []byte("save-bytes"))
		r.err("write "+name, err)
	}
	plantFile(t, filepath.Join(dir, "game0003.sav"), "planted")
	name, err := SaveStore{Dir: dir}.WriteOriginal("", []byte("third"))
	r.err("write over a gap "+name, err)
	r.tree(dir)

	r.line("## single save fences")
	inst := plantInstall(t, filepath.Join(r.root(t), "inst"))
	_, err = SaveStore{Dir: filepath.Join(inst, "saves")}.WriteOriginal("", []byte("x"))
	r.err("under an install", err)
	orig := filepath.Join(r.root(t), "orig")
	plantFile(t, filepath.Join(orig, "keep.txt"), "k")
	_, err = SaveStore{Dir: filepath.Join(orig, "saves")}.WriteOriginal(orig, []byte("x"))
	r.err("under the original directory", err)
	r.tree(inst)
	r.tree(orig)

	r.line("## single save failures")
	for _, c := range []map[string]int{
		{"create": 1}, {"write": 1}, {"sync": 1}, {"close": 1}, {"publish": 1}, {"publish": 1, "remove": 1},
	} {
		dir := r.root(t)
		_, err := SaveStore{Dir: dir, files: faults(c)}.WriteOriginal("", []byte("lost"))
		r.err(fmt.Sprint(c), err)
		r.tree(dir)
		name, err := SaveStore{Dir: dir}.WriteOriginal("", []byte("next"))
		r.err("next write "+name, err)
		r.tree(dir)
	}

	r.line("## single save recovery")
	// The save folder cannot be listed through the store's Ops, so only the
	// refused removal of a leftover temporary save is pinned.
	dir = r.root(t)
	plantFile(t, filepath.Join(dir, ".save-original.sav.tmp-"+strconv.Itoa(os.Getpid())+"-123"), "left")
	_, err = SaveStore{Dir: dir, files: recordRemoveRefused{}}.WriteOriginal("", []byte("x"))
	r.err("leftover temporary save", err)
	r.tree(dir)

	r.line("## screenshot")
	dir = r.root(t)
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := 0; i < 2; i++ {
		name, err := writeScreenshot(SaveStore{Dir: dir}, nil, img)
		r.err("screenshot "+name, err)
	}
	_, err = writeScreenshot(SaveStore{Dir: filepath.Join(inst, "shots")}, nil, img)
	r.err("screenshot under an install", err)
	r.tree(dir)
}

func (r *hostRecord) namedSets(t *testing.T) {
	both := []namedSavePayload{{".sav", []byte("sav-one")}, {".ags", []byte("ags-one")}}
	next := []namedSavePayload{{".sav", []byte("sav-two")}, {".ags", []byte("ags-two")}}
	commit := func(dir string, payloads []namedSavePayload, fences []string, files namedSaveFileOps, overwrite bool) error {
		base, err := namedSaveBase("slot")
		if err != nil {
			return err
		}
		prepared, err := prepareNamedSave(dir, base, payloads, fences, files)
		if err != nil {
			return err
		}
		_, err = prepared.Commit(overwrite)
		return err
	}
	r.line("## named save set")
	dir := r.root(t)
	r.err("new set", commit(dir, both, nil, nil, false))
	r.err("existing without overwrite", commit(dir, next, nil, nil, false))
	r.tree(dir)
	r.err("existing with overwrite", commit(dir, next, nil, nil, true))
	r.tree(dir)
	inst := plantInstall(t, filepath.Join(r.root(t), "inst"))
	r.err("under an install", commit(filepath.Join(inst, "saves"), both, nil, nil, false))
	orig := filepath.Join(r.root(t), "orig")
	plantFile(t, filepath.Join(orig, "keep.txt"), "k")
	r.err("under the original directory", commit(filepath.Join(orig, "saves"), both, []string{orig}, nil, false))

	r.line("## named save set failures")
	for _, c := range []struct {
		existing bool
		fail     map[string]int
	}{
		{false, map[string]int{"create": 2}},
		{false, map[string]int{"write": 2}},
		{false, map[string]int{"publish": 2}},
		{false, map[string]int{"publish": 2, "remove": 1}},
		{true, map[string]int{"create": 3}},
		{true, map[string]int{"replace": 2}},
		{true, map[string]int{"replace": 2, "remove": 1}},
	} {
		dir := r.root(t)
		if c.existing {
			if err := commit(dir, both, nil, nil, false); err != nil {
				t.Fatal(err)
			}
		}
		r.err(fmt.Sprintf("existing=%v %v", c.existing, c.fail), commit(dir, next, nil, faults(c.fail), true))
		r.tree(dir)
	}
	dir = r.root(t)
	if err := commit(dir, both, nil, nil, false); err != nil {
		t.Fatal(err)
	}
	r.err("existing=true restore fails", commit(dir, next, nil, &restoreFaults{faults(nil)}, true))
	r.tree(dir)

	r.line("## save delete")
	dir = r.root(t)
	plantFile(t, filepath.Join(dir, "game0000.sav"), "doomed")
	run, err := prepareLoadDeleteFiles(dir, localOriginalSavePrefix+"game0000.sav", nil, faults(map[string]int{"remove": 1}))
	r.err("prepare delete", err)
	if run != nil {
		r.err("delete with the remove failing", run())
	}
	r.tree(dir)
	run, err = prepareLoadDelete(dir, localOriginalSavePrefix+"game0000.sav", nil)
	r.err("prepare delete", err)
	if run != nil {
		r.err("delete", run())
	}
	r.tree(dir)
}

// restoreFaults also fails the rollback's restoring replace.
type restoreFaults struct{ *recordFaults }

func (f *restoreFaults) Replace(oldPath, newPath string) error {
	f.calls["replace"]++
	if f.calls["replace"] >= 2 {
		return fmt.Errorf("planted replace failure %d", f.calls["replace"])
	}
	return f.osNamedSaveFiles.Replace(oldPath, newPath)
}

func (r *hostRecord) preferences(t *testing.T) {
	r.line("## options file")
	dir := r.root(t)
	s := OptionsStore{Path: filepath.Join(dir, optionsFileName)}
	r.err("tips on", s.SetTipsMode(true))
	r.err("speed", s.SetGameSpeed(defaultGameSpeedRung()))
	r.tree(dir)
	inst := plantInstall(t, filepath.Join(r.root(t), "inst"))
	r.err("under an install", OptionsStore{Path: filepath.Join(inst, optionsFileName)}.SetTipsMode(true))
	orig := filepath.Join(r.root(t), "orig")
	plantFile(t, filepath.Join(orig, "keep.txt"), "k")
	r.err("under the original directory", OptionsStore{Path: filepath.Join(orig, "p", optionsFileName), originalDir: orig}.SetTipsMode(true))

	r.line("## hall of fame")
	dir = r.root(t)
	for i := int32(1); i <= 2; i++ {
		_, err := fameStore{Dir: dir}.add(fame.Record{Name: fmt.Sprint("hero", i), Score: i})
		r.err(fmt.Sprint("add ", i), err)
	}
	r.tree(dir)
	_, err := fameStore{Dir: filepath.Join(inst, "hall")}.add(fame.Record{Name: "x", Score: 1})
	r.err("under an install", err)
	for _, c := range []map[string]int{{"create": 1}, {"write": 1}, {"replace": 1}} {
		dir := r.root(t)
		if _, err := (fameStore{Dir: dir}).add(fame.Record{Name: "a", Score: 1}); err != nil {
			t.Fatal(err)
		}
		_, err := fameStore{Dir: dir, files: faults(c)}.add(fame.Record{Name: "b", Score: 2})
		r.err(fmt.Sprint(c), err)
		r.tree(dir)
	}
	dir = r.root(t)
	_, err = fameStore{Dir: dir, files: faults(map[string]int{"publish": 1})}.add(fame.Record{Name: "b", Score: 2})
	r.err("new hall publish fails", err)
	r.tree(dir)
}

func (r *hostRecord) modFiles(t *testing.T) {
	r.line("## mod pack")
	src := filepath.Join(r.root(t), "rec")
	plantFile(t, filepath.Join(src, "mod.toml"), "id = \"rec\"\ntitle = \"R\"\nversion = \"1\"\napplies-to = [\"en\"]\n")
	plantFile(t, filepath.Join(src, "data", "a.txt"), "a")
	out := r.root(t)
	for _, replace := range []bool{false, false, true} {
		path, err := PackMod(src, out, replace)
		r.err(fmt.Sprintf("pack replace=%v %s", replace, path), err)
	}
	r.tree(out)
	_, err := PackMod(src, out, false)
	r.line("pack exists is ErrPackExists=%v", errors.Is(err, ErrPackExists))
	inst := plantInstall(t, filepath.Join(r.root(t), "inst"))
	_, err = PackMod(src, inst, true)
	r.err("pack into an install", err)
	r.line("pack into an install is ErrPackInstall=%v", errors.Is(err, ErrPackInstall))
	r.tree(inst)
	_, err = AssembleModMission(inst, "en", 10, inst)
	r.err("assemble into the install", err)

	r.line("## base dump")
	dump := filepath.Join(r.root(t), "dump")
	files := map[string][]byte{"mod.toml": []byte("m"), "data/a.toml": []byte("a"), "data/deep/b.toml": []byte("b")}
	order := []string{"mod.toml", "data/a.toml", "data/deep/b.toml"}
	r.err("dump", writeNewFiles(dump, true, order, files))
	r.tree(filepath.Dir(dump))
	clash := filepath.Join(r.root(t), "dump")
	plantFile(t, filepath.Join(clash, "data", "deep", "b.toml"), "in the way")
	err = writeNewFiles(clash, false, order, files)
	r.err("dump over a file", err)
	r.line("dump over a file is ErrBaseDumpExists=%v", errors.Is(err, ErrBaseDumpExists))
	r.tree(clash)

	r.line("## headless frame paths")
	front := &FrontEnd{}
	front.Archives = &Archives{Root: inst}
	r.err("frame inside the install", front.checkHeadlessFramePath(filepath.Join(inst, "f.png")))
	free := r.root(t)
	plantFile(t, filepath.Join(free, "f.png"), "old")
	r.err("frame over a file", front.checkHeadlessFramePath(filepath.Join(free, "f.png")))
	r.err("frame new", front.checkHeadlessFramePath(filepath.Join(free, "g.png")))
}

func (r *hostRecord) editorFiles(t *testing.T) {
	r.line("## editor map save")
	dir := r.root(t)
	path := filepath.Join(dir, "m.alm")
	got, err := writeEditedMap("", "", path, []byte("map-one"), false)
	r.err("new "+got, err)
	_, err = writeEditedMap("", "", path, []byte("map-two"), false)
	r.err("new over an existing map", err)
	got, err = writeEditedMap("", "", path, []byte("map-three"), true)
	r.err("save over "+got, err)
	r.tree(dir)
	inst := plantInstall(t, filepath.Join(r.root(t), "inst"))
	_, err = writeEditedMap("", "", filepath.Join(inst, "m.alm"), []byte("x"), false)
	r.err("under an install", err)
	_, err = writeEditedMap(inst, "", filepath.Join(inst, "maps", "m.alm"), []byte("x"), false)
	r.err("under the root", err)

	r.line("## editor map pages")
	dir = r.root(t)
	path = filepath.Join(dir, "p.alm")
	pages := map[string][]byte{"en/briefing.txt": []byte("brief"), "title.txt": []byte("name")}
	got, err = writeEditedMapPages("", "", path, []byte("pmap"), pages, []byte("labels-one"), "en", false)
	r.err("new with pages "+got, err)
	r.tree(dir)
	_, err = writeEditedMapPages("", "", path, []byte("pmap2"), pages, nil, "en", false)
	r.err("new over existing pages", err)
	pages2 := map[string][]byte{"en/briefing.txt": []byte("brief-two")}
	got, err = writeEditedMapPages("", "", path, []byte("pmap3"), pages2, []byte("labels-two"), "en", true)
	r.err("save over with pages "+got, err)
	r.tree(dir)
	got, err = writeEditedMapPages("", "", path, []byte("pmap4"), nil, nil, "en", true)
	r.err("save over without pages or labels "+got, err)
	r.tree(dir)

}

func (r *hostRecord) testPlayFiles(t *testing.T) {
	r.line("## test play files")
	top := r.root(t)
	folder := filepath.Join(top, "tp")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	none := func(string) error { return nil }
	r.err("write", writeTestModFile(none, top, folder, "data/a.txt", []byte("one"), false))
	r.err("write durable over", writeTestModFile(none, top, folder, "data/a.txt", []byte("two"), true))
	r.err("write outside", writeTestModFile(none, top, folder, "../../x.txt", []byte("x"), false))
	r.err("guard refuses", writeTestModFile(func(string) error { return errors.New("guarded") }, top, folder, "b.txt", []byte("x"), false))
	r.tree(top)

	r.line("## test play copy")
	dest := filepath.Join(r.root(t), "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	r.err("copy", copyRetainedTestMod(folder, dest, []string{"data/old.txt"}, "record.txt", none))
	r.err("copy again", copyRetainedTestMod(folder, dest, nil, "record.txt", none))
	r.tree(filepath.Dir(dest))

	r.line("## test play package")
	for _, owned := range []bool{false, true} {
		base := r.root(t)
		stage := filepath.Join(base, "stage")
		plantFile(t, filepath.Join(stage, "id", "new.txt"), "new")
		live := filepath.Join(base, "live")
		if owned {
			plantFile(t, filepath.Join(live, "old.txt"), "old")
		}
		r.err(fmt.Sprintf("install owned=%v", owned), installMissionPackage(stage, live, "id", owned))
		r.tree(base)
	}
	base := r.root(t)
	stage := filepath.Join(base, "stage")
	plantFile(t, filepath.Join(stage, "id", "new.txt"), "new")
	live := filepath.Join(base, "live")
	plantFile(t, filepath.Join(live, "old.txt"), "old")
	r.err("install over an unowned folder", installMissionPackage(stage, live, "id", false))
	r.tree(base)

	// A package whose ID is previous fails on the rename, as it did before the
	// transaction; the operating system's words after the paths differ by
	// platform and are not recorded.
	base = r.root(t)
	stage = filepath.Join(base, "stage")
	plantFile(t, filepath.Join(stage, "previous", "new.txt"), "new")
	live = filepath.Join(base, "live")
	plantFile(t, filepath.Join(live, "old.txt"), "old")
	err := installMissionPackage(stage, live, "previous", true)
	if err == nil {
		r.line("install previous: ok")
	} else {
		msg := err.Error()
		if i := strings.Index(msg, "previous: "); i >= 0 {
			msg = msg[:i+len("previous: ")] + "..."
		}
		r.line("install previous: error %s", msg)
	}
	r.tree(base)
}
