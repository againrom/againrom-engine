package game

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"againrom/pkg/hostfile"
	"againrom/pkg/namefold"
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

// saveTempFile and savePublishResult are the host-file transaction's staged
// file and publication result.
type (
	saveTempFile      = hostfile.TempFile
	savePublishResult = hostfile.PublishResult
)

// saveFileOps holds create, publish and cleanup behind one production seam, so
// tests can fail those operations at each boundary without replacing
// SaveStore.Write. Directory enumeration remains a direct read-only operation.
type saveFileOps interface {
	CreateTemp(dir, pattern string) (saveTempFile, error)
	Publish(oldPath, newPath string) (savePublishResult, error)
	Remove(path string) error
}

// osSaveFileOps is the operating system's file seam.
type osSaveFileOps = hostfile.OS

// claimOps is a save store's seam as the transaction's: a claimed save is
// never published by replacement.
type claimOps struct{ saveFileOps }

func (claimOps) Replace(string, string) error {
	return errors.New("a claimed save never replaces a file")
}

type saveDirectoryIdentity struct {
	path string
	info os.FileInfo
}

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

// insideDir reports whether path is parent itself or lies under it, as the
// install fence compares them.
func insideDir(parent, path string) bool { return hostfile.InsideDir(parent, path) }

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
	if _, err := os.ReadDir(s.Dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []SaveEntry
	for _, e := range scanSaves(saveScan{Dir: s.Dir, Bounded: true}) {
		out = append(out, SaveEntry{Name: e.Name, Label: e.rowLabel(s.Selector), Mod: e.Mod})
	}
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
	// Publication cleanup can fail for more than one concurrent writer.
	// Retirement and recovery share the transaction's lock, so the last
	// writer to leave the active set sees every earlier retired alias, skips
	// paths still active and reaps all paths the OS now permits.
	result, err := hostfile.Commit(hostfile.Set{
		Files:   []hostfile.File{{Path: dir.path, Data: data, Overwrite: hostfile.Refuse}},
		Dir:     dir.path,
		Temp:    saveTempPatternFor(tempBase, ext),
		Sync:    true,
		Claim:   nameAt,
		Recover: isOwnedSaveTempName,
		DirInfo: dir.info,
		Ops:     claimOps{s.fileOps()},
		Check: func(step hostfile.Step, _ int) error {
			if step != hostfile.StepPublish || validate == nil {
				return nil
			}
			current, err := resolveSaveDirectory(dir.path)
			if err != nil {
				return err
			}
			if !os.SameFile(dir.info, current.info) {
				return errors.New("save directory changed before publication")
			}
			return validate(current)
		},
		Words: hostfile.Words{
			Create:       "create temporary save: %w",
			Write:        "write temporary save: %w",
			Sync:         "synchronize temporary save: %w",
			Close:        "close temporary save: %w",
			Publish:      "publish save %s: %w",
			NoCommit:     "publish save %s returned without committing a final path",
			NoName:       "no free original save name in game0000.sav through game9999.sav",
			Cleanup:      "remove failed temporary save %s; it remains hidden and the next save will retry cleanup: %w",
			CloseCleanup: "close failed temporary save during cleanup: %w",
		},
	})
	if err != nil {
		return "", err
	}
	return result.Name, nil
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

func sameSaveName(a, b string) bool { return hostfile.SameName(a, b) }

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

// installFence is the host-file transaction's preserved-install fence over
// this game's archives. The names below bind it for the game's writers.
var installFence = hostfile.Fence{Archives: []string{MainArchive, GraphicsArchive, ScenarioArchive, WorldArchive, MoviesArchive}}

var errWriteUnderInstall = hostfile.ErrWriteUnderInstall

func refuseOriginalWriteTarget(target, originalDir string) error {
	return installFence.RefuseOriginalWriteTarget(target, originalDir)
}

func refuseOriginalWriteTargets(target string, roots []string) error {
	return installFence.RefuseOriginalWriteTargets(target, roots)
}

func requiredArchiveAncestor(target string) (string, bool, error) {
	return installFence.InstallAncestor(target)
}

func resolvePathForSaveSafety(path string) (string, error) { return hostfile.ResolvePath(path) }

func sameOrDescendantPhysical(parent, child string) (bool, error) {
	return hostfile.SameOrDescendantPhysical(parent, child)
}

func directoryHasRequiredArchives(dir string) (bool, error) { return installFence.IsInstall(dir) }

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
	var out []SaveEntry
	for _, e := range scanSaves(saveScan{Dir: s.Dir}) {
		out = append(out, SaveEntry{Name: e.Name, Label: e.rowLabel(s.Selector), Mod: e.Mod})
	}
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
	path, err := InstallFile(s.Dir, name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// IsOriginal reports whether a name from the load window belongs to the original
// game's format rather than ours. It reads the EXTENSION and opens nothing: the
// question is which reader a row belongs to, and a build that had to open a file
// to find out would open every file in the list to draw it.
func IsOriginal(name string) bool {
	return namefold.HostEqual(filepath.Ext(name), originalSaveExt)
}
