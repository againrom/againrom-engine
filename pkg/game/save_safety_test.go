package game

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"hash/crc32"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

type saveRendezvousOps struct {
	ready   chan struct{}
	release chan struct{}
}

func (o *saveRendezvousOps) CreateTemp(dir, pattern string) (saveTempFile, error) {
	return os.CreateTemp(dir, pattern)
}

func (o *saveRendezvousOps) Publish(oldPath, newPath string) (savePublishResult, error) {
	o.ready <- struct{}{}
	<-o.release
	return publishSaveFile(oldPath, newPath)
}

func (o *saveRendezvousOps) Remove(path string) error { return os.Remove(path) }

// linkedCleanupOps drives the portable hard-link protocol through the exact
// SaveStore seam while this test binary is running on Windows. Only the
// post-commit removal inside Publish is faulted; ordinary recovery uses the
// real removal and therefore proves the residue's next-operation lifecycle.
type linkedCleanupOps struct {
	publishRemoveFailures  int
	recoveryRemoveFailures int
	publishRemovePaths     []string
}

func (o *linkedCleanupOps) CreateTemp(dir, pattern string) (saveTempFile, error) {
	return os.CreateTemp(dir, pattern)
}

func (o *linkedCleanupOps) Publish(oldPath, newPath string) (savePublishResult, error) {
	return publishLinkedSave(oldPath, newPath, os.Link, func(path string) error {
		o.publishRemovePaths = append(o.publishRemovePaths, path)
		if o.publishRemoveFailures > 0 {
			o.publishRemoveFailures--
			return errSaveRemoveBoundary
		}
		return os.Remove(path)
	})
}

func (o *linkedCleanupOps) Remove(path string) error {
	if o.recoveryRemoveFailures > 0 {
		o.recoveryRemoveFailures--
		return errSaveRemoveBoundary
	}
	return os.Remove(path)
}

type concurrentLinkedCleanupOps struct {
	ready                  chan struct{}
	release                chan struct{}
	mu                     sync.Mutex
	recoveryRemoveFailures int
}

func (o *concurrentLinkedCleanupOps) CreateTemp(dir, pattern string) (saveTempFile, error) {
	return os.CreateTemp(dir, pattern)
}

func (o *concurrentLinkedCleanupOps) Publish(oldPath, newPath string) (savePublishResult, error) {
	return publishLinkedSave(oldPath, newPath, os.Link, func(string) error {
		o.ready <- struct{}{}
		<-o.release
		return errSaveRemoveBoundary
	})
}

func (o *concurrentLinkedCleanupOps) Remove(path string) error {
	o.mu.Lock()
	if o.recoveryRemoveFailures > 0 {
		o.recoveryRemoveFailures--
		o.mu.Unlock()
		return errSaveRemoveBoundary
	}
	o.mu.Unlock()
	return os.Remove(path)
}

// TestConcurrentLinkedCommitsReapAllRetiredAliases covers the compound state a
// serial writer cannot produce. Both hard links commit while both private
// unlinks are refused. The first retirement's recovery is also refused; when
// the second writer leaves the active set, its serialized recovery must see and
// remove both inactive aliases without touching either writer while active.
func TestConcurrentLinkedCommitsReapAllRetiredAliases(t *testing.T) {
	dir := t.TempDir()
	ops := &concurrentLinkedCleanupOps{
		ready:                  make(chan struct{}, 2),
		release:                make(chan struct{}),
		recoveryRemoveFailures: 1,
	}
	store := SaveStore{Dir: dir, files: ops}
	at := time.Date(2026, 8, 17, 3, 0, 0, 0, time.UTC)
	type result struct {
		name string
		err  error
	}
	results := make(chan result, 2)
	for _, gold := range []int{1, 2} {
		data, err := EncodeSave(Snapshot{Gold: gold}, fmt.Sprintf("writer %d", gold))
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			name, err := store.Write(at, data)
			results <- result{name: name, err: err}
		}()
	}
	<-ops.ready
	<-ops.ready
	close(ops.release)
	a, b := <-results, <-results
	if a.err != nil || b.err != nil || a.name == "" || b.name == "" || a.name == b.name {
		t.Fatalf("concurrent committed writes = (%q, %v), (%q, %v)", a.name, a.err, b.name, b.err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var finals, temps []string
	for _, entry := range entries {
		switch {
		case looksLikeSaveTemp(entry.Name()):
			temps = append(temps, entry.Name())
		case filepath.Ext(entry.Name()) == saveExt:
			finals = append(finals, entry.Name())
		}
	}
	if len(temps) != 0 || len(finals) != 2 {
		t.Fatalf("after both retirements final=%v private=%v, want two final and no private aliases", finals, temps)
	}
}

// TestLinkedPublicationMakesLinkCreationTheCommitPoint reproduces the
// non-Windows counterexample without depending on this CI host's GOOS. The
// private-name unlink is refused, and a hypothetical rollback unlink would be
// refused too; production must not attempt that second unlink or return an
// error beside a visible final path.
func TestLinkedPublicationMakesLinkCreationTheCommitPoint(t *testing.T) {
	oldPath, newPath := "private.tmp", "final.ags"
	visible := false
	var removePaths []string
	result, err := publishLinkedSave(oldPath, newPath,
		func(old, new string) error {
			if old != oldPath || new != newPath {
				t.Fatalf("link = (%q, %q), want (%q, %q)", old, new, oldPath, newPath)
			}
			visible = true
			return nil
		},
		func(path string) error {
			removePaths = append(removePaths, path)
			if path == newPath {
				return errors.New("injected rollback-final removal boundary")
			}
			return errSaveRemoveBoundary
		})
	if err != nil {
		t.Fatalf("publishLinkedSave = %v, want committed success", err)
	}
	if !result.committed || !errors.Is(result.cleanupErr, errSaveRemoveBoundary) {
		t.Fatalf("result = %+v, want committed with private-name cleanup error", result)
	}
	if !visible {
		t.Fatal("exclusive link did not make the final path visible")
	}
	if !reflect.DeepEqual(removePaths, []string{oldPath}) {
		t.Fatalf("remove paths = %v, want only private name; final rollback must not run", removePaths)
	}
}

// TestSaveStoreReportsCommittedLinkedSaveAndRecoversItsPrivateName carries the
// same refusal through Write. A valid final row means success, not a save
// error; the hidden hard-link alias becomes ordinary same-process residue and
// is removed before the next save creates another temporary file.
func TestSaveStoreReportsCommittedLinkedSaveAndRecoversItsPrivateName(t *testing.T) {
	dir := t.TempDir()
	data, err := EncodeSave(Snapshot{Gold: 77}, "linked commit")
	if err != nil {
		t.Fatal(err)
	}
	ops := &linkedCleanupOps{publishRemoveFailures: 1, recoveryRemoveFailures: 1}
	store := SaveStore{Dir: dir, files: ops}
	at := time.Date(2026, 8, 17, 2, 0, 0, 0, time.UTC)

	name, err := store.Write(at, data)
	if err != nil || name != "save-20260817-020000.ags" {
		t.Fatalf("committed Write = (%q, %v), want final-name success", name, err)
	}
	if len(ops.publishRemovePaths) != 1 || filepath.Base(ops.publishRemovePaths[0]) == name {
		t.Fatalf("post-commit removals = %v, want only the private source name", ops.publishRemovePaths)
	}
	listed, err := listAGS(store)
	if err != nil || len(listed) != 1 || listed[0].Name != name {
		t.Fatalf("list after cleanup refusal = (%v, %v), want committed final row", listed, err)
	}
	if body, err := store.Read(name); err != nil || !reflect.DeepEqual(body, data) {
		t.Fatalf("committed final bytes = (%d, %v), want %d", len(body), err, len(data))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var temps []string
	for _, entry := range entries {
		if looksLikeSaveTemp(entry.Name()) {
			temps = append(temps, entry.Name())
		}
	}
	if len(temps) != 1 {
		t.Fatalf("hidden aliases after committed cleanup refusal = %v, want one", temps)
	}

	ops.recoveryRemoveFailures = 1
	if refused, err := store.Write(at, data); refused != "" || !errors.Is(err, errSaveRemoveBoundary) {
		t.Fatalf("Write while alias recovery is refused = (%q, %v), want no new save", refused, err)
	}
	second, err := store.Write(at, data)
	if err != nil || second != "save-20260817-020000-2.ags" {
		t.Fatalf("Write after recovery = (%q, %v), want second committed name", second, err)
	}
	entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if looksLikeSaveTemp(entry.Name()) {
			t.Fatalf("next Write left recovered private name %q", entry.Name())
		}
	}
	if body, err := store.Read(name); err != nil || !reflect.DeepEqual(body, data) {
		t.Fatalf("first committed save after recovery = (%d, %v), want unchanged", len(body), err)
	}
}

func TestConcurrentSameSecondPublicationIsExclusive(t *testing.T) {
	dir := t.TempDir()
	ops := &saveRendezvousOps{ready: make(chan struct{}, 2), release: make(chan struct{})}
	store := SaveStore{Dir: dir, files: ops}
	at := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)

	type result struct {
		name string
		err  error
	}
	results := make(chan result, 2)
	var started sync.WaitGroup
	started.Add(2)
	for _, data := range [][]byte{[]byte("first"), []byte("second")} {
		data := data
		go func() {
			started.Done()
			name, err := store.Write(at, data)
			results <- result{name: name, err: err}
		}()
	}
	started.Wait()
	<-ops.ready
	<-ops.ready
	close(ops.release)
	a, b := <-results, <-results
	if a.err != nil || b.err != nil {
		t.Fatalf("concurrent writes returned %v and %v", a.err, b.err)
	}
	if a.name == b.name {
		t.Fatalf("concurrent writes both published %q", a.name)
	}
}

// TestActiveSaveTempIdentitySurvivesDirectoryAliases pauses one writer after
// its temporary file is closed but before publication. A second writer names
// the same physical directory through an absolute alias. Recovery must
// recognise the first path as active rather than delete it as residue.
func TestActiveSaveTempIdentitySurvivesDirectoryAliases(t *testing.T) {
	relative, err := os.MkdirTemp(".", ".save-alias-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(relative) })
	dir, err := filepath.Abs(relative)
	if err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err = filepath.Rel(wd, dir)
	if err != nil {
		t.Skipf("cannot construct a relative alias for %s: %v", dir, err)
	}
	absoluteAlias := dir
	if os.PathSeparator == '\\' {
		absoluteAlias = strings.ToUpper(dir)
	}
	first, err := EncodeSave(Snapshot{Gold: 1}, "relative writer")
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncodeSave(Snapshot{Gold: 2}, "absolute writer")
	if err != nil {
		t.Fatal(err)
	}
	paused := &saveRendezvousOps{ready: make(chan struct{}, 1), release: make(chan struct{})}
	at := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	type result struct {
		name string
		err  error
	}
	firstResult := make(chan result, 1)
	go func() {
		name, err := (SaveStore{Dir: relative, files: paused}).Write(at, first)
		firstResult <- result{name: name, err: err}
	}()
	select {
	case <-paused.ready:
	case result := <-firstResult:
		t.Fatalf("relative writer failed before publication pause: %v", result.err)
	}
	secondName, secondErr := (SaveStore{Dir: absoluteAlias}).Write(at, second)
	close(paused.release)
	firstDone := <-firstResult
	if firstDone.err != nil || secondErr != nil {
		t.Fatalf("aliased writers returned relative=%v absolute=%v", firstDone.err, secondErr)
	}
	if firstDone.name == secondName {
		t.Fatalf("aliased writers both published %q", secondName)
	}
	for name, wantGold := range map[string]int{firstDone.name: 1, secondName: 2} {
		body, err := (SaveStore{Dir: dir}).Read(name)
		if err != nil {
			t.Fatal(err)
		}
		snap, _, err := DecodeSave(body)
		if err != nil || snap.Gold != wantGold {
			t.Fatalf("save %q = gold %d, %v; want %d", name, snap.Gold, err, wantGold)
		}
	}
}

// TestSaveStoreConcurrentProcesses runs independent copies of this test
// binary against one save directory. No process shares a Go lock or file seam,
// so uniqueness comes from the operating-system publication operation alone.
func TestSaveStoreConcurrentProcesses(t *testing.T) {
	dir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	const processes = 6
	commands := make([]*exec.Cmd, 0, processes)
	for i := 0; i < processes; i++ {
		cmd := exec.Command(exe, "-test.run=^TestSaveStoreConcurrentProcessHelper$", "-test.count=1")
		cmd.Env = append(os.Environ(),
			"AGAINROM_SAVE_HELPER_DIR="+dir,
			"AGAINROM_SAVE_HELPER_INDEX="+strconv.Itoa(i))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, cmd)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		ready, err := filepath.Glob(filepath.Join(dir, "ready-*"))
		if err != nil {
			t.Fatal(err)
		}
		if len(ready) == processes {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d/%d helper processes reached the publication barrier", len(ready), processes)
		}
		time.Sleep(time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(dir, "start"), []byte("start"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("save helper: %v", err)
		}
	}

	names := make(map[string]bool, processes)
	store := SaveStore{Dir: dir}
	for i := 0; i < processes; i++ {
		result, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("result-%d", i)))
		if err != nil {
			t.Fatal(err)
		}
		name := string(result)
		if names[name] {
			t.Fatalf("two processes returned %q", name)
		}
		names[name] = true
		body, err := store.Read(name)
		if err != nil {
			t.Fatal(err)
		}
		snap, _, err := DecodeSave(body)
		if err != nil || snap.Gold != i+1 {
			t.Fatalf("process %d save %q decoded as gold %d, error %v", i, name, snap.Gold, err)
		}
	}
	list, err := listAGS(store)
	if err != nil || len(list) != processes {
		t.Fatalf("concurrent process list = (%d, %v), want %d", len(list), err, processes)
	}
}

func TestSaveStoreConcurrentProcessHelper(t *testing.T) {
	dir := os.Getenv("AGAINROM_SAVE_HELPER_DIR")
	if dir == "" {
		return
	}
	index, err := strconv.Atoi(os.Getenv("AGAINROM_SAVE_HELPER_INDEX"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("ready-%d", index)), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "start")); err == nil {
			break
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("publication barrier was not released")
		}
		time.Sleep(time.Millisecond)
	}
	payload, err := EncodeSave(Snapshot{Gold: index + 1}, fmt.Sprintf("process %d", index))
	if err != nil {
		t.Fatal(err)
	}
	name, err := (SaveStore{Dir: dir}).Write(
		time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC), payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("result-%d", index)), []byte(name), 0o644); err != nil {
		t.Fatal(err)
	}
}

func declaredPayloadTail(t *testing.T, encoded, tail []byte) []byte {
	t.Helper()
	out := append([]byte(nil), encoded...)
	labelLen := int(binary.LittleEndian.Uint16(out[len(saveMagic)+1:]))
	checksumAt := saveHeaderLen + labelLen
	lengthAt := checksumAt + 4
	bodyAt := lengthAt + 4
	out = append(out, tail...)
	body := out[bodyAt:]
	binary.LittleEndian.PutUint32(out[checksumAt:], crc32.ChecksumIEEE(body))
	binary.LittleEndian.PutUint32(out[lengthAt:], uint32(len(body)))
	return out
}

func TestDecodeRejectsTrailingDataInsideDeclaredPayload(t *testing.T) {
	first, err := EncodeSave(Snapshot{Gold: 7}, "inside tail")
	if err != nil {
		t.Fatal(err)
	}
	var second bytes.Buffer
	if err := gob.NewEncoder(&second).Encode(Snapshot{Gold: 8}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		tail []byte
	}{
		{name: "second gob value", tail: second.Bytes()},
		{name: "unread byte", tail: []byte{0xa5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snap, label, err := DecodeSave(declaredPayloadTail(t, first, tc.tail))
			if err == nil {
				t.Fatal("DecodeSave accepted data after the first gob value")
			}
			if !strings.Contains(err.Error(), "trailing data") {
				t.Fatalf("inside-payload refusal = %q, want trailing data", err)
			}
			if !reflect.DeepEqual(snap, Snapshot{}) || label != "" {
				t.Fatalf("inside-payload refusal returned snapshot %+v and label %q", snap, label)
			}
		})
	}
}

type endlessSaveReader struct{}

func (endlessSaveReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestBoundedSaveReadStopsFileGrowthAfterStat(t *testing.T) {
	if body, err := readBoundedSave(endlessSaveReader{}, 1); err == nil ||
		!strings.Contains(err.Error(), "grew beyond") || body != nil {
		t.Fatalf("post-Stat growth read = (%d bytes, %v), want bounded growth refusal", len(body), err)
	}
}

func TestPreparedOpenerCommitsOnceAcrossConcurrentCalls(t *testing.T) {
	var calls atomic.Int32
	enteredCommit := make(chan struct{}, 2)
	releaseCommit := make(chan struct{})
	opener := openPrepared(preparedMap{}, func() {
		calls.Add(1)
		enteredCommit <- struct{}{}
		<-releaseCommit
	})
	start := make(chan struct{})
	var ready, done sync.WaitGroup
	ready.Add(2)
	done.Add(2)
	for range 2 {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			_, _, _, _, _, _, _, _, _, _, _ = opener()
		}()
	}
	ready.Wait()
	close(start)
	<-enteredCommit
	select {
	case <-enteredCommit:
		close(releaseCommit)
		done.Wait()
		t.Fatalf("simultaneous opener calls entered commit %d times", calls.Load())
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseCommit)
	done.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("simultaneous opener calls committed %d times, want one", got)
	}
}

func TestPublicationRefusesAndPreservesAnExistingSave(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "new.tmp"), filepath.Join(dir, "old.ags")
	if err := os.WriteFile(src, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := publishSaveFile(src, dst)
	if !isPublishCollision(err) {
		t.Fatalf("publication onto an existing save = %v, want a collision", err)
	}
	if result.committed {
		t.Fatal("publication collision reported a committed final path")
	}
	for path, want := range map[string]string{src: "new", dst: "old"} {
		got, readErr := os.ReadFile(path)
		if readErr != nil || string(got) != want {
			t.Fatalf("%s after refusal = (%q, %v), want %q", path, got, readErr, want)
		}
	}
}

func candidateBodyFixture(t *testing.T) (*FrontEnd, Snapshot, string, *terrain.UnitSet) {
	t.Helper()
	dir := t.TempDir()
	pal := make([]color.RGBA, 4)
	pal[1] = color.RGBA{R: 0xff, A: 0xff}
	archives := map[string][]byte{
		ScenarioArchive: synth.Archive([]synth.File{
			{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{Width: 24, Height: 24})},
			{Path: "npc.reg", Data: synth.NPCReg(nil)},
		}),
		"graphics.res": synth.Archive([]synth.File{{
			Path: "units/heroes_l/swordsman/sprites.256",
			Data: synth.Sheet256(synth.Sheet256Options{Palette: pal,
				Frames: []synth.Frame256{{Width: 2, Height: 2}}}),
		}}),
	}
	var hosts []string
	for name, body := range archives {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
		hosts = append(hosts, path)
	}
	containers, err := vfs.Open(hosts, nil)
	if err != nil {
		t.Fatal(err)
	}
	units := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
		3: {Width: 2, Height: 2, CenterX: 1, CenterY: 1, OwnerShaded: true},
	}, HasOwnerPalettes: true}
	units.OwnerPalettes[3][17] = color.RGBA{R: 0x31, G: 0x42, B: 0x53, A: 0xff}
	units.OwnerPalettes[15][255] = color.RGBA{R: 0xa4, G: 0xb5, B: 0xc6, A: 0xff}
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Containers: containers}, Tiles: &terrain.Tileset{}, Units: units, Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: saveTown(t), Carried: saveParty()}}
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	app := f.App("candidate-cache")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	s.Party = []mapload.PartyMember{{Body: "swordsman", BodyDir: "heroes_l"}}
	key := data.HeroBodyKey("heroes_l", "swordsman")
	if _, exists := f.Units.Bodies[key]; exists {
		t.Fatal("fixture body was already cached")
	}
	return f, s, key, f.live.units
}

func TestFailedCandidateDoesNotPopulateTheLiveBodyCache(t *testing.T) {
	f, s, key, oldUnits := candidateBodyFixture(t)
	s.World = append([]byte(nil), s.World[:8]...)
	if _, err := f.prepareRestore(s); err == nil {
		t.Fatal("corrupt candidate world was accepted")
	}
	if f.live.units != oldUnits || f.live.units != f.Units {
		t.Fatal("failed candidate replaced the live or front-end unit set")
	}
	if _, exists := f.Units.Bodies[key]; exists {
		t.Fatal("failed candidate populated the live body cache")
	}
}

func TestCandidateBodyCacheIsAdoptedOnlyAtCommit(t *testing.T) {
	f, s, key, oldUnits := candidateBodyFixture(t)
	open, town, err := f.Restore(s)
	if err != nil {
		t.Fatal(err)
	}
	if town || open == nil {
		t.Fatalf("prepared mission = (town %v, opener %v), want mission opener", town, open != nil)
	}
	if f.Units != oldUnits {
		t.Fatal("mission preparation replaced the front-end unit set before commit")
	}
	if _, exists := oldUnits.Bodies[key]; exists {
		t.Fatal("mission preparation populated the old unit set")
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	if f.Units == oldUnits || f.live.units != f.Units {
		t.Fatal("commit did not adopt one candidate unit set for the front end and live map")
	}
	if _, exists := f.Units.Bodies[key]; !exists {
		t.Fatal("committed candidate unit set does not hold its loaded body")
	}
	if body := f.Units.Bodies[key]; body == nil || !body.OwnerShaded {
		t.Fatalf("committed candidate body = %+v, want owner-shade eligibility retained", body)
	}
	if !f.Units.HasOwnerPalettes || !f.live.units.HasOwnerPalettes {
		t.Fatal("candidate commit lost the owner-palette presence bit")
	}
	for _, sentinel := range []struct {
		table, entry int
		want         color.RGBA
	}{
		{3, 17, color.RGBA{R: 0x31, G: 0x42, B: 0x53, A: 0xff}},
		{15, 255, color.RGBA{R: 0xa4, G: 0xb5, B: 0xc6, A: 0xff}},
	} {
		if got := f.Units.OwnerPalettes[sentinel.table][sentinel.entry]; got != sentinel.want {
			t.Errorf("committed owner table %d entry %d = %+v, want sentinel %+v",
				sentinel.table, sentinel.entry, got, sentinel.want)
		}
		if got := f.live.units.OwnerPalettes[sentinel.table][sentinel.entry]; got != sentinel.want {
			t.Errorf("live owner table %d entry %d = %+v, want sentinel %+v",
				sentinel.table, sentinel.entry, got, sentinel.want)
		}
	}
	if _, exists := oldUnits.Bodies[key]; exists {
		t.Fatal("commit wrote the candidate body into the old unit set")
	}
}
