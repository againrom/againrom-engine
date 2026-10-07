package game

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SaveStore holds authored saves. Only a launch-issued private profile can
// write inside an install; a directory string alone never grants that access.
type SaveStore struct {
	Dir string

	// Selector is the install's language digit, threaded through to
	// asciiLabel/OriginalSaveLabel exactly as text.Font.Selector already is
	// for every other installed string (installtext.go): 0 applies no shift,
	// 1 undoes the RU install's own atlas shift before decoding. A caller
	// that never sets it (every tool and fixture outside the live SAVE/LOAD
	// wiring) reads a SAV label as an EN-install byte string, which is what
	// this build's own writer produces when no install context is wired at
	// all — the same "no language named" default text.Font.Selector documents.
	Selector int

	// files is the production file-operation seam. Its zero value selects the
	// operating system. Tests replace it to force failures after a temporary
	// file exists; no caller outside this package can replace it.
	files   saveFileOps
	profile *runtimeProfileAccess
}

// saveTempFile is the part of *os.File atomic publication uses. Name is part
// of the seam because os.CreateTemp chooses the sibling path exclusively.
type saveTempFile interface {
	io.Writer
	Name() string
	Sync() error
	Close() error
}

// saveFileOps holds create, publish and cleanup behind one production seam, so
// tests can fail those operations at each boundary without replacing
// SaveStore.Write. Directory enumeration remains a direct read-only operation.
type saveFileOps interface {
	CreateTemp(dir, pattern string) (saveTempFile, error)
	Publish(oldPath, newPath string) (savePublishResult, error)
	Remove(path string) error
}

// savePublishResult separates the namespace commit from post-commit cleanup.
// A nil error with committed set means the final path exists and Write must
// report success. cleanupErr says the private source name could not be removed;
// after the active registration is retired, the ordinary next-save recovery
// owns that hidden residue. Returning the cleanup error as a publication error
// would contradict the visible final row on systems which publish by hard
// link.
type savePublishResult struct {
	committed  bool
	cleanupErr error
}

type osSaveFileOps struct{}

var (
	saveTempMu     sync.Mutex
	activeSaveTemp = make(map[uint64]saveTempIdentity)
	nextSaveTempID uint64
)

type saveDirectoryIdentity struct {
	path string
	info os.FileInfo
}

type saveTempIdentity struct {
	dir  os.FileInfo
	name string
}

func (osSaveFileOps) CreateTemp(dir, pattern string) (saveTempFile, error) {
	return os.CreateTemp(dir, pattern)
}

func (osSaveFileOps) Publish(oldPath, newPath string) (savePublishResult, error) {
	return publishSaveFile(oldPath, newPath)
}
func (osSaveFileOps) Remove(path string) error { return os.Remove(path) }

func (s SaveStore) fileOps() saveFileOps {
	if s.files != nil {
		return s.files
	}
	return osSaveFileOps{}
}

// DefaultSaveDir is `saves/` beside the running executable, or the override
// given.
//
// os.Executable FAILING IS REPORTED AND NOT DEFAULTED. A default reached by
// failure — the working directory, say — is how a save lands somewhere nobody
// looks and a load reports an empty list on a machine that has ten.
//
// WHICH DIRECTORIES THOSE ARE IS toolchainRoots' OWN ANSWER and is not the
// temporary directory alone; its doc block carries the failure that widened it.
func DefaultSaveDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot locate the executable to place %s beside it: %w", "saves/", err)
	}
	roots := toolchainRoots()
	var wd string
	if insideAny(roots, filepath.Dir(exe)) {
		if wd, err = os.Getwd(); err != nil {
			return "", fmt.Errorf("cannot locate the working directory to place %s beside it: %w", "saves/", err)
		}
	}
	return saveDirFor(exe, wd, roots...), nil
}

// toolchainRoots is every directory the Go toolchain puts a `go run` binary
// under on this machine.
//
// GOTMPDIR is here for the same reason and not because it was observed: it is
// where the toolchain builds when it is set, and a root left out is a root
// nobody notices until a player loses a save in it.
//
// AN UNREADABLE CACHE DIRECTORY IS SIMPLY NOT A ROOT. There is nothing to
// report: every root is a place a save must not go, so failing to name one
// leaves the beside-the-binary rule standing, which is the behaviour that
// shipped and the safe direction to fail in.
func toolchainRoots() []string {
	roots := []string{os.TempDir()}
	if d := os.Getenv("GOTMPDIR"); d != "" {
		roots = append(roots, d)
	}
	if d := os.Getenv("GOCACHE"); d != "" {
		roots = append(roots, d)
	} else if d, err := os.UserCacheDir(); err == nil {
		roots = append(roots, filepath.Join(d, "go-build"))
	}
	return roots
}

// insideAny is insideDir over every root, and an empty root name is not one.
func insideAny(roots []string, path string) bool {
	for _, r := range roots {
		if r != "" && insideDir(r, path) {
			return true
		}
	}
	return false
}

// saveDirFor is DefaultSaveDir's choice with the machine readings handed in, so
// it can be tested over paths this machine does not have.
//
// It is VARIADIC in the roots rather than taking one, because there is more
// than one directory the toolchain builds into and a single name is what let
// the build cache through.
func saveDirFor(exe, wd string, roots ...string) string {
	if insideAny(roots, filepath.Dir(exe)) {
		return filepath.Join(wd, "saves")
	}
	return filepath.Join(filepath.Dir(exe), "saves")
}

// insideDir reports whether path is parent itself or lies under it.
//
// THE COMPARISON IS CASE-INSENSITIVE because both sides are read off the same
// Windows machine through different APIs and neither promises the other's
// casing. Cutting the prefix can split a multi-byte rune, in which case EqualFold
// answers false and the caller keeps the plain beside-the-binary rule — the
// behaviour that shipped, which is the safe direction to fail in.
func insideDir(parent, path string) bool {
	p, c := filepath.Clean(parent), filepath.Clean(path)
	if p == "" || p == "." || len(c) < len(p) || !strings.EqualFold(c[:len(p)], p) {
		return false
	}
	return len(c) == len(p) || os.IsPathSeparator(c[len(p)]) || os.IsPathSeparator(p[len(p)-1])
}

// SaveEntry is one row of what is on disk: the bare file name the store reads
// back by, and the label out of that file's own header.
type SaveEntry struct {
	Name  string
	Label string
	Mod   time.Time
}

// List is what is on disk, NEWEST FIRST.
//
// A FILE THAT WILL NOT READ IS SKIPPED AND THE LIST IS STILL SHOWN. One corrupt
// save is not a reason the player cannot load the other nine, and its own
// refusal sentence is what he gets if he had a row for it and chose it.
//
// A MISSING DIRECTORY IS AN EMPTY LIST AND NOT AN ERROR: the store is created
// on the first write (plan D-9), so "no saves yet" and "no directory yet" are
// one state.
func (s SaveStore) List() ([]SaveEntry, error) {
	if s.Dir == "" {
		return nil, errors.New("no save directory configured")
	}
	ents, err := os.ReadDir(s.Dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []SaveEntry
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(e.Name()), originalSaveExt) {
			continue
		}
		fi, err := e.Info()
		if err != nil || fi.Size() < 0 || fi.Size() > maxSaveBytes {
			continue
		}
		b, err := s.Read(e.Name())
		if err != nil {
			continue
		}
		label, err := OriginalSaveLabel(b, s.Selector)
		if err != nil {
			continue
		}
		out = append(out, SaveEntry{Name: e.Name(), Label: label, Mod: fi.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Mod.Equal(out[j].Mod) {
			return out[i].Name > out[j].Name
		}
		return out[i].Mod.After(out[j].Mod)
	})
	return out, nil
}

// Read reads one save back by the name List reported.
//
// A NAME THAT IS NOT A BARE FILE NAME IS REFUSED. List produces bare names;
// a name carrying a separator or `..` did not come from it and must not
// become a path.
func (s SaveStore) Read(name string) ([]byte, error) {
	if s.Dir == "" {
		return nil, errors.New("no save directory configured")
	}
	if name == "" || name != filepath.Base(name) || name == "." || name == ".." {
		return nil, fmt.Errorf("%q is not a save name", name)
	}
	f, err := os.Open(filepath.Join(s.Dir, name))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return readBoundedSave(f, fi.Size())
}

// readBoundedSave checks the same-handle Stat before reading and also limits
// the stream to maxSaveBytes+1. The latter is the TOCTOU guard: a file that
// grows after Stat cannot make io.ReadAll continue allocating without bound.
func readBoundedSave(r io.Reader, reportedSize int64) ([]byte, error) {
	if reportedSize < 0 {
		return nil, fmt.Errorf("save reports an invalid size %d", reportedSize)
	}
	if reportedSize > maxSaveBytes {
		return nil, fmt.Errorf("save is %d bytes; safe maximum is %d", reportedSize, maxSaveBytes)
	}
	b, err := io.ReadAll(io.LimitReader(r, maxSaveBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxSaveBytes {
		return nil, fmt.Errorf("save grew beyond the safe maximum of %d bytes while it was read", maxSaveBytes)
	}
	return b, nil
}

// WriteOriginal atomically claims the lowest free game####.sav slot. The
// originalDir fence admits only this store's launch-issued private profile.
func (s SaveStore) WriteOriginal(originalDir string, data []byte) (string, error) {
	if err := refuseProfileWriteTarget(s.Dir, originalDir, s.profile); err != nil {
		return "", err
	}
	validate := func(dir saveDirectoryIdentity) error {
		return refuseProfileWriteTarget(dir.path, originalDir, s.profile)
	}
	return s.write(data, "save-original", originalSaveExt, validate, func(at int) (string, bool) {
		if at >= 10000 {
			return "", false
		}
		return fmt.Sprintf("game%04d%s", at, originalSaveExt), true
	})
}

type saveNameAt func(int) (string, bool)

func (s SaveStore) write(data []byte, tempBase, ext string, validate func(saveDirectoryIdentity) error,
	nameAt saveNameAt) (string, error) {
	if s.Dir == "" {
		return "", errors.New("no save directory configured")
	}
	if len(data) > maxSaveBytes {
		return "", fmt.Errorf("save is %d bytes; safe maximum is %d", len(data), maxSaveBytes)
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return "", err
	}
	dir, err := resolveSaveDirectory(s.Dir)
	if err != nil {
		return "", err
	}
	if validate != nil {
		if err := validate(dir); err != nil {
			return "", err
		}
	}
	files := s.fileOps()
	// Recovery and exclusive creation share one short lock with registration.
	// A concurrent writer may continue writing outside the lock, but its path
	// is then known as active and recovery will skip it.
	saveTempMu.Lock()
	if err := recoverSaveTemps(dir, files); err != nil {
		saveTempMu.Unlock()
		return "", err
	}
	tmp, err := files.CreateTemp(dir.path, saveTempPatternFor(tempBase, ext))
	if err != nil {
		saveTempMu.Unlock()
		return "", fmt.Errorf("create temporary save: %w", err)
	}
	tmpName := filepath.Base(tmp.Name())
	tmpPath := filepath.Join(dir.path, tmpName)
	nextSaveTempID++
	tmpID := nextSaveTempID
	activeSaveTemp[tmpID] = saveTempIdentity{dir: dir.info, name: tmpName}
	saveTempMu.Unlock()
	retireTemp := func(reap bool) error {
		saveTempMu.Lock()
		defer saveTempMu.Unlock()
		delete(activeSaveTemp, tmpID)
		if reap {
			return recoverSaveTemps(dir, files)
		}
		return nil
	}
	closeAttempted := false
	fail := func(primary error) error {
		failures := []error{primary}
		if !closeAttempted {
			closeAttempted = true
			if err := tmp.Close(); err != nil {
				failures = append(failures, fmt.Errorf("close failed temporary save during cleanup: %w", err))
			}
		}
		if err := files.Remove(tmpPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, fmt.Errorf(
				"remove failed temporary save %s; it remains hidden and the next save will retry cleanup: %w",
				filepath.Base(tmpPath), err,
			))
		}
		_ = retireTemp(false)
		return errors.Join(failures...)
	}

	if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
		return "", fail(fmt.Errorf("write temporary save: %w", err))
	}
	if err := tmp.Sync(); err != nil {
		return "", fail(fmt.Errorf("synchronize temporary save: %w", err))
	}
	closeAttempted = true
	if err := tmp.Close(); err != nil {
		return "", fail(fmt.Errorf("close temporary save: %w", err))
	}
	// THE NAME IS RESERVED BY PUBLICATION ITSELF. A preceding Stat would be a
	// TOCTOU check: two processes can both observe a missing path and then both
	// rename onto it. publishSaveFile refuses an existing destination in the
	// same operation that creates the final name.
	for at := 0; ; at++ {
		name, ok := nameAt(at)
		if !ok {
			return "", fail(fmt.Errorf("no free original save name in game0000.sav through game9999.sav"))
		}
		if validate != nil {
			current, err := resolveSaveDirectory(dir.path)
			if err != nil {
				return "", fail(err)
			}
			if !os.SameFile(dir.info, current.info) {
				return "", fail(errors.New("save directory changed before publication"))
			}
			if err := validate(current); err != nil {
				return "", fail(err)
			}
		}
		published, err := files.Publish(tmpPath, filepath.Join(dir.path, name))
		if err != nil {
			if isPublishCollision(err) {
				continue
			}
			return "", fail(fmt.Errorf("publish save %s: %w", name, err))
		}
		if !published.committed {
			return "", fail(fmt.Errorf("publish save %s returned without committing a final path", name))
		}
		// cleanupErr is deliberately not a returned error: the final path is
		// already committed. Retiring the registration makes the hidden source
		// name eligible for recoverSaveTemps before this process's next write.
		if published.cleanupErr != nil {
			// Publication cleanup can fail for more than one concurrent
			// writer. Retirement and recovery share saveTempMu, so the last
			// writer to leave the active set sees every earlier retired alias,
			// skips paths still active and reaps all paths the OS now permits.
			// A continuing refusal cannot turn this committed final row into a
			// returned failure; the next writer performs the same recovery.
			_ = retireTemp(true)
			return name, nil
		}
		_ = retireTemp(true)
		return name, nil
	}
}

// recoverSaveTemps removes only this process's hidden sibling namespace. A
// prior failure can leave one there only when the operating system refused
// cleanup. An in-process registry excludes active callers, and another process
// has a different owner token and remains untouched. Recovery runs before a
// new temporary file is created, so repeated refusals do not accumulate files
// or reach final-name publication. The caller holds saveTempMu.
func recoverSaveTemps(dir saveDirectoryIdentity, files saveFileOps) error {
	entries, err := os.ReadDir(dir.path)
	if err != nil {
		return fmt.Errorf("inspect temporary saves before writing: %w", err)
	}
	var failures []error
	for _, entry := range entries {
		if entry.IsDir() || !isOwnedSaveTempName(entry.Name()) {
			continue
		}
		path := filepath.Join(dir.path, entry.Name())
		if saveTempIsActive(dir.info, entry.Name()) {
			continue
		}
		if err := files.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, fmt.Errorf("remove hidden temporary save %s: %w", entry.Name(), err))
		}
	}
	if len(failures) != 0 {
		return fmt.Errorf("recover temporary saves before writing; no new save was started: %w", errors.Join(failures...))
	}
	return nil
}

func resolveSaveDirectory(dir string) (saveDirectoryIdentity, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return saveDirectoryIdentity{}, fmt.Errorf("resolve save directory %s: %w", dir, err)
	}
	resolved := abs
	// EvalSymlinks gives publication one stable spelling for junction and
	// symlink aliases when the platform permits the query. Windows can deny
	// that metadata query on an otherwise usable directory, so physical
	// identity below remains the authority and the absolute path is the safe
	// fallback.
	if evaluated, evalErr := filepath.EvalSymlinks(abs); evalErr == nil {
		resolved = evaluated
	}
	resolved = filepath.Clean(resolved)
	info, err := os.Stat(resolved)
	if err != nil {
		return saveDirectoryIdentity{}, fmt.Errorf("identify save directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return saveDirectoryIdentity{}, fmt.Errorf("save directory %s is not a directory", dir)
	}
	return saveDirectoryIdentity{path: resolved, info: info}, nil
}

func saveTempIsActive(dir os.FileInfo, name string) bool {
	for _, active := range activeSaveTemp {
		if os.SameFile(dir, active.dir) && sameSaveName(name, active.name) {
			return true
		}
	}
	return false
}

func sameSaveName(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func saveTempPatternFor(base, ext string) string {
	return "." + base + ext + ".tmp-" + strconv.Itoa(os.Getpid()) + "-*"
}

// ownedSaveTempExtensions are the save extensions whose private staging names
// this process may recover.
var ownedSaveTempExtensions = []string{originalSaveExt}

func isOwnedSaveTempName(name string) bool {
	if !strings.HasPrefix(name, ".save-") {
		return false
	}
	for _, ext := range ownedSaveTempExtensions {
		marker := ext + ".tmp-" + strconv.Itoa(os.Getpid()) + "-"
		i := strings.Index(name, marker)
		if i > len(".save-") && i+len(marker) < len(name) {
			return true
		}
	}
	return false
}

// refuseOriginalWriteTarget is the runtime half of the preserved-install rule.
// The explicit originalDir fence catches the configured install and every
// physical descendant. The archive census catches an install accidentally
// supplied without that context, including the upper-case filenames shipped by
// the RU distribution. Missing target directories are resolved through their
// nearest existing parent so a symlink/junction cannot hide a future child.
func refuseOriginalWriteTarget(target, originalDir string) error {
	if target == "" {
		return errors.New("no save directory configured")
	}
	resolvedTarget, err := resolvePathForSaveSafety(target)
	if err != nil {
		return fmt.Errorf("resolve original-save target %s: %w", target, err)
	}
	if originalDir != "" {
		resolvedOriginal, err := resolvePathForSaveSafety(originalDir)
		if err != nil {
			return fmt.Errorf("resolve read-only game install %s: %w", originalDir, err)
		}
		physical, err := sameOrDescendantPhysical(originalDir, target)
		if err != nil {
			return fmt.Errorf("compare original-save target with read-only game install: %w", err)
		}
		if insideDir(resolvedOriginal, resolvedTarget) || physical {
			return fmt.Errorf("refusing to write original-format save under read-only game install %s", originalDir)
		}
	}
	installDir, install, err := requiredArchiveAncestor(resolvedTarget)
	if err != nil {
		return fmt.Errorf("inspect original-save target ancestors for %s: %w", target, err)
	}
	if install {
		return fmt.Errorf("refusing to write original-format save under game install %s", installDir)
	}
	return nil
}

func requiredArchiveAncestor(target string) (string, bool, error) {
	for probe := filepath.Clean(target); ; probe = filepath.Dir(probe) {
		install, err := directoryHasRequiredArchives(probe)
		if err != nil {
			return "", false, err
		}
		if install {
			return probe, true, nil
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", false, nil
		}
	}
}

func resolvePathForSaveSafety(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	probe := abs
	for {
		if _, err := os.Lstat(probe); err == nil {
			resolved, err := filepath.EvalSymlinks(probe)
			if err != nil {
				// Windows can deny reparse-point metadata for an otherwise
				// readable directory. Physical identity is still checked by
				// sameOrDescendantPhysical below; retain the absolute spelling.
				resolved = probe
			}
			rel, err := filepath.Rel(probe, abs)
			if err != nil {
				return "", err
			}
			return filepath.Clean(filepath.Join(resolved, rel)), nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return abs, nil
		}
		probe = parent
	}
}

func sameOrDescendantPhysical(parent, child string) (bool, error) {
	parentInfo, err := os.Stat(parent)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	absChild, err := filepath.Abs(child)
	if err != nil {
		return false, err
	}
	for probe := filepath.Clean(absChild); ; probe = filepath.Dir(probe) {
		info, err := os.Stat(probe)
		if err == nil && os.SameFile(parentInfo, info) {
			return true, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		parentProbe := filepath.Dir(probe)
		if parentProbe == probe {
			return false, nil
		}
	}
}

func directoryHasRequiredArchives(dir string) (bool, error) {
	want := map[string]bool{
		strings.ToLower(MainArchive):     false,
		strings.ToLower(GraphicsArchive): false,
		strings.ToLower(ScenarioArchive): false,
		strings.ToLower(WorldArchive):    false,
		strings.ToLower(MoviesArchive):   false,
	}
	// The shipped game and the supported runtime are Windows, where a direct
	// lookup is case-insensitive. Besides being cheaper, these probes still
	// work when the sandbox permits naming a child but refuses enumerating a
	// broad ancestor such as the user profile. ReadDir below supplies the same
	// case-insensitive census on case-sensitive development filesystems.
	direct, absent := 0, 0
	for name := range want {
		info, err := os.Stat(filepath.Join(dir, name))
		if err == nil && !info.IsDir() {
			want[name] = true
			direct++
			continue
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, os.ErrPermission) {
			return false, err
		}
		if errors.Is(err, os.ErrNotExist) {
			absent++
		}
	}
	if direct == len(want) {
		return true, nil
	}
	if runtime.GOOS == "windows" && absent == len(want) {
		return false, nil
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) || (errors.Is(err, os.ErrPermission) && direct == 0) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if _, ok := want[name]; ok && !entry.IsDir() {
			want[name] = true
		}
	}
	for _, found := range want {
		if !found {
			return false, nil
		}
	}
	return true, nil
}

// originalSaveExt is what a save file is called. SAV is the only save format;
// the original game and this one name it the same way.
const originalSaveExt = ".sav"

// OriginalStore reads installed saves and has no write capability.
type OriginalStore struct {
	Dir string

	// Selector carries the same install language digit SaveStore.Selector
	// documents, for the same reason: a genuinely original .sav's label byte
	// page is Unknown (asciiLabel's own doc), and the current install's own
	// code page is this reader's one principled default rather than a claim
	// about what wrote the file.
	Selector int
}

// List is every original save in the directory, newest first, labelled from its
// own head.
//
// A FILE THAT WILL NOT READ IS SKIPPED, on SaveStore.List's reasoning: one save
// this tree cannot open is not a reason to hide the others. A MISSING OR UNSET
// DIRECTORY IS AN EMPTY LIST AND NOT AN ERROR — a build pointed at no install,
// or an install with no saves in it, are one state to a player.
func (s OriginalStore) List() []SaveEntry {
	if s.Dir == "" {
		return nil
	}
	ents, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil
	}
	var out []SaveEntry
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), originalSaveExt) {
			continue
		}
		b, err := s.Read(e.Name())
		if err != nil {
			continue
		}
		label, err := OriginalSaveLabel(b, s.Selector)
		if err != nil {
			continue
		}
		var mod time.Time
		if fi, err := e.Info(); err == nil {
			mod = fi.ModTime()
		}
		out = append(out, SaveEntry{Name: e.Name(), Label: label, Mod: mod})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Mod.Equal(out[j].Mod) {
			return out[i].Name > out[j].Name
		}
		return out[i].Mod.After(out[j].Mod)
	})
	return out
}

// Read reads one original save back by the name List reported.
//
// A NAME THAT IS NOT A BARE FILE NAME IS REFUSED, on SaveStore.Read's reasoning
// and with more at stake: this directory is the player's own game install, so a
// name carrying a separator or `..` is the one way a chosen row could reach a
// file that is not a save at all.
//
// THE WHOLE FILE IS READ. Unlike our own format there is no short head to peek
// at — the label sits past a compressed body's own header — and these files are
// tens of kilobytes.
func (s OriginalStore) Read(name string) ([]byte, error) {
	if s.Dir == "" {
		return nil, errors.New("no game install configured to read saves from")
	}
	if name != filepath.Base(name) || name == "." || name == ".." {
		return nil, fmt.Errorf("%q is not a save name", name)
	}
	return os.ReadFile(filepath.Join(s.Dir, name))
}

// IsOriginal reports whether a name from the load window belongs to the original
// game's format rather than ours. It reads the EXTENSION and opens nothing: the
// question is which reader a row belongs to, and a build that had to open a file
// to find out would open every file in the list to draw it.
func IsOriginal(name string) bool {
	return strings.EqualFold(filepath.Ext(name), originalSaveExt)
}
