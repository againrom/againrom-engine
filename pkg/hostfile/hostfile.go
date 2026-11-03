// Package hostfile is the one transaction that writes a file a player or a
// modder keeps: a save, a screenshot, the options and hall files, a packed or
// dumped mod, an edited map and its pages, the test-play files and the
// editor and starter profile files. Each caller states its policy as a Set:
// where the bytes are staged, what an existing destination gets, which fence
// and checks run before publication, and what a failure rolls back.
package hostfile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// TempFile is the part of *os.File staging uses. Name is part of the seam
// because os.CreateTemp chooses the sibling path exclusively.
type TempFile interface {
	io.Writer
	Name() string
	Sync() error
	Close() error
}

// PublishResult separates the namespace commit from post-commit cleanup. A
// nil error with Committed set means the final path exists. CleanupErr says
// the private source name could not be removed; next-write recovery owns that
// hidden residue.
type PublishResult struct {
	Committed  bool
	CleanupErr error
}

// Ops is the file-operation seam of a Set's staged files. Tests replace it to
// fail an operation at each boundary.
type Ops interface {
	CreateTemp(dir, pattern string) (TempFile, error)
	Publish(oldPath, newPath string) (PublishResult, error)
	Replace(oldPath, newPath string) error
	Remove(path string) error
}

// OS is the operating system's Ops.
type OS struct{}

func (OS) CreateTemp(dir, pattern string) (TempFile, error) { return os.CreateTemp(dir, pattern) }
func (OS) Publish(oldPath, newPath string) (PublishResult, error) {
	return publishFile(oldPath, newPath)
}
func (OS) Replace(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }
func (OS) Remove(path string) error              { return os.Remove(path) }

// Overwrite is what a file's publication does with an existing final path.
type Overwrite uint8

const (
	// Refuse publishes without replacing: an existing path is a collision.
	Refuse Overwrite = iota
	// Replace puts the new file in the existing one's place whole.
	Replace
	// Backup is Replace with a staged copy of the prior bytes that a
	// rollback restores.
	Backup
)

// Staging is where a file's bytes are written before they are final.
type Staging uint8

const (
	// Sibling stages a hidden temporary file beside the target.
	Sibling Staging = iota
	// Direct writes the final path itself: an exclusive creation under
	// Refuse, an in-place write under Replace.
	Direct
)

// Seen is what preparation saw at a file's final path.
type Seen uint8

const (
	SeenAny     Seen = iota // not looked at: a Sibling file replaces whatever is there
	SeenAbsent              // nothing: the file is published without replacing
	SeenPresent             // a file: it is replaced
)

// Dirs is which folders a Set makes for its Direct files.
type Dirs uint8

const (
	NoDirs  Dirs = iota
	Parents      // each file's folder and its parents are made and kept
	Owned        // the folders under Root are made one by one and a rollback removes them
)

// Step names when a Set's Check runs.
type Step uint8

const (
	StepStage   Step = iota // before a file or its backup is staged
	StepPublish             // before a file is published (each claimed name)
	StepRestore             // before a rollback restores a removed file
)

// File is one row of a Set.
type File struct {
	Path      string // final path; with Set.Claim, the folder the name is claimed in
	Data      []byte
	Perm      fs.FileMode // Direct: creation mode; Sibling: the staged file's mode when not zero
	Overwrite Overwrite
	Staging   Staging
	Seen      Seen
	Prior     []byte // the bytes preparation read at Path: a backup's content
	Remove    bool   // the file at Path is removed; a rollback publishes its backup again
	Folder    string // a staged folder renamed to Path
	Backup    string // with Folder: where the folder at Path waits until the swap is done
	Stale     bool   // with Backup: a name outside the staged folder that an earlier failed swap may have left; it is cleared first
	// Stream, for a Direct file under Refuse, writes the bytes into the
	// created file in place of Data.
	Stream func(io.Writer) error
	// Keep leaves a Direct file under Refuse as written when the Set fails.
	Keep bool
}

// Words are a Set's error texts. An empty text returns the operating
// system's error as it is.
type Words struct {
	Create, Write, Sync, Close string // one staging step, "...: %w"
	Stage, StageBackup         string // a staging failure of a file or of its backup, "...: %w"
	Publish                    string // name or path, error
	NoCommit                   string // name or path; a publication that did not commit
	NoName                     string // no claimable name is left
	Cleanup                    string // base name, error: a staged file left behind is reported
	CloseCleanup               string // error
	Restore                    string // path, backup, error: a rollback failure is reported
	RestoreNoCommit            string
	RestoreFolder              string // error, backup, restore error
}

// Set is one file-set transaction: its files and the caller's policy.
type Set struct {
	Files    []File
	Dir      string // where Sibling files and backups are staged; each file's folder when empty
	Temp     string // the staged files' os.CreateTemp pattern
	Sync     bool   // staged and Direct bytes are synchronized before publication
	Dirs     Dirs
	DirPerm  fs.FileMode
	Root     string // Owned: the folder the files lie in
	MakeRoot bool   // Owned: Root is made too
	Discard  string // a staging folder removed when the transaction ends, unless it retains a backup
	// Claim names the at-th candidate for the one file; the first that
	// publishes without a collision is the file's name.
	Claim func(at int) (string, bool)
	// Recover names the hidden staged files of this process in Dir that a
	// failed cleanup left; they are removed before a file is staged.
	Recover func(name string) bool
	DirInfo os.FileInfo // Dir's identity, for Recover
	Check   func(step Step, file int) error
	Ops     Ops
	Words   Words
}

// Result is what a transaction committed.
type Result struct {
	Done int    // files published, in order
	Name string // the claimed name
}

// Commit runs s: every file is staged, then each is checked and published in
// order. A failure rolls back what was published, removes what was staged
// and returns the failure with every rollback failure the Words report.
func Commit(s Set) (Result, error) {
	t := &txn{s: s, ops: s.Ops}
	if t.ops == nil {
		t.ops = OS{}
	}
	return t.run()
}

type txn struct {
	s               Set
	ops             Ops
	staged, backups []string
	undo            []func() error
	ids             []uint64
	retained        bool
	res             Result
}

func (t *txn) run() (Result, error) {
	n := len(t.s.Files)
	t.staged, t.backups = make([]string, n), make([]string, n)
	w := t.s.Words
	for i, f := range t.s.Files {
		if err := t.check(StepStage, i); err != nil {
			return t.end(err, false)
		}
		if f.Staging == Sibling && !f.Remove && f.Folder == "" {
			path, err := t.stage(t.dirOf(f), f.Data, f.Perm)
			if err != nil {
				return t.end(say(w.Stage, err), false)
			}
			t.staged[i] = path
		}
		if f.Remove || f.Overwrite == Backup && f.Seen == SeenPresent && f.Folder == "" {
			path, err := t.stage(t.dirOf(f), f.Prior, 0)
			if err != nil {
				return t.end(say(w.StageBackup, err), false)
			}
			t.backups[i] = path
		}
	}
	if t.s.Dirs == Owned && t.s.MakeRoot {
		if err := os.MkdirAll(filepath.Dir(t.s.Root), t.s.DirPerm); err != nil {
			return t.end(err, true)
		}
		if err := t.mkdir(t.s.Root); err != nil {
			return t.end(err, true)
		}
	}
	for i := range t.s.Files {
		if err := t.publish(i); err != nil {
			return t.end(err, true)
		}
		t.res.Done++
	}
	return t.end(nil, true)
}

func (t *txn) dirOf(f File) string {
	if t.s.Dir != "" {
		return t.s.Dir
	}
	if t.s.Claim != nil {
		return f.Path
	}
	return filepath.Dir(f.Path)
}

func (t *txn) check(step Step, i int) error {
	if t.s.Check == nil {
		return nil
	}
	return t.s.Check(step, i)
}

// stage writes data to a new hidden file in dir and returns its path. A
// failure removes the file.
func (t *txn) stage(dir string, data []byte, perm fs.FileMode) (string, error) {
	w := t.s.Words
	var tmp TempFile
	var err error
	if t.s.Recover != nil {
		// Recovery and exclusive creation share one short lock with
		// registration. A concurrent writer may continue writing outside the
		// lock, but its path is then known as active and recovery skips it.
		tmp, err = t.createRegistered(dir)
	} else {
		tmp, err = t.ops.CreateTemp(dir, t.s.Temp)
	}
	var refused *recoveryError
	if errors.As(err, &refused) {
		return "", refused.err
	}
	if err != nil {
		return "", say(w.Create, err)
	}
	path := filepath.Join(dir, filepath.Base(tmp.Name()))
	closed := false
	fail := func(primary error) (string, error) {
		failures := []error{primary}
		if !closed {
			closed = true
			if err := tmp.Close(); err != nil && w.CloseCleanup != "" {
				failures = append(failures, say(w.CloseCleanup, err))
			}
		}
		if err := t.ops.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) && w.Cleanup != "" {
			failures = append(failures, say(w.Cleanup, filepath.Base(path), err))
		}
		t.retire(false)
		return "", joined(failures)
	}
	if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
		return fail(say(w.Write, err))
	}
	if t.s.Sync {
		if err := tmp.Sync(); err != nil {
			return fail(say(w.Sync, err))
		}
	}
	closed = true
	if err := tmp.Close(); err != nil {
		return fail(say(w.Close, err))
	}
	if perm != 0 {
		if err := os.Chmod(path, perm); err != nil {
			return fail(err)
		}
	}
	return path, nil
}

func (t *txn) publish(i int) error {
	f, w := t.s.Files[i], t.s.Words
	if t.s.Claim != nil {
		return t.claim(i)
	}
	if err := t.check(StepPublish, i); err != nil {
		return err
	}
	switch {
	case f.Remove:
		if err := t.ops.Remove(f.Path); err != nil {
			return err
		}
		t.undo = append(t.undo, func() error {
			err := t.check(StepRestore, i)
			if err == nil {
				var result PublishResult
				result, err = t.ops.Publish(t.backups[i], f.Path)
				if err == nil && !result.Committed {
					err = errors.New(w.RestoreNoCommit)
				}
			}
			return t.restoreFailed(i, err)
		})
	case f.Folder != "":
		return t.swapFolder(f)
	case f.Staging == Direct:
		return t.direct(f)
	case f.Overwrite == Refuse || f.Seen == SeenAbsent:
		result, err := t.ops.Publish(t.staged[i], f.Path)
		if err == nil && !result.Committed {
			if w.Publish == "" {
				return say(w.NoCommit, f.Path)
			}
			err = say(w.NoCommit, f.Path)
		}
		if err != nil {
			return say(w.Publish, f.Path, err)
		}
		t.undo = append(t.undo, func() error { return t.restoreFailed(i, t.ops.Remove(f.Path)) })
	default:
		if err := t.ops.Replace(t.staged[i], f.Path); err != nil {
			return say(w.Publish, f.Path, err)
		}
		if t.backups[i] != "" {
			t.undo = append(t.undo, func() error { return t.restoreFailed(i, t.ops.Replace(t.backups[i], f.Path)) })
		}
	}
	return nil
}

// restoreFailed reports a rollback failure of file i when the Words report
// one, and keeps the backup the report names.
func (t *txn) restoreFailed(i int, err error) error {
	if err == nil || t.s.Words.Restore == "" {
		return nil
	}
	report := say(t.s.Words.Restore, t.s.Files[i].Path, t.backups[i], err)
	t.backups[i] = ""
	return report
}

// claim publishes the one staged file under the first free claimed name.
// The name is reserved by publication itself: a preceding Stat would let two
// processes both observe a missing path and then both rename onto it.
func (t *txn) claim(i int) error {
	w := t.s.Words
	dir := t.dirOf(t.s.Files[i])
	for at := 0; ; at++ {
		name, ok := t.s.Claim(at)
		if !ok {
			return errors.New(w.NoName)
		}
		if err := t.check(StepPublish, i); err != nil {
			return err
		}
		result, err := t.ops.Publish(t.staged[i], filepath.Join(dir, name))
		if err != nil {
			if IsCollision(err) {
				continue
			}
			return say(w.Publish, name, err)
		}
		if !result.Committed {
			return say(w.NoCommit, name)
		}
		// A cleanup failure is not a returned error: the final path is
		// already committed, and retiring the registration makes the hidden
		// source name eligible for recovery before the next write.
		t.staged[i] = ""
		t.res.Name = name
		return nil
	}
}

func (t *txn) direct(f File) error {
	w := t.s.Words
	switch t.s.Dirs {
	case Parents:
		if err := os.MkdirAll(filepath.Dir(f.Path), t.s.DirPerm); err != nil {
			return err
		}
	case Owned:
		rel, err := filepath.Rel(t.s.Root, filepath.Dir(f.Path))
		if err != nil {
			return err
		}
		if rel != "." {
			at := t.s.Root
			for _, part := range strings.Split(rel, string(filepath.Separator)) {
				at = filepath.Join(at, part)
				if err := t.mkdir(at); err != nil {
					return err
				}
			}
		}
	}
	if f.Overwrite != Refuse {
		return os.WriteFile(f.Path, f.Data, f.Perm)
	}
	out, err := os.OpenFile(f.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, f.Perm)
	if err != nil {
		return say(w.Create, err)
	}
	if !f.Keep {
		// This exact file was created exclusively here, never an input.
		t.undo = append(t.undo, func() error { _ = os.Remove(f.Path); return nil })
	}
	if f.Stream != nil {
		err = f.Stream(out)
	} else {
		_, err = out.Write(f.Data)
	}
	if err == nil && t.s.Sync {
		err = out.Sync()
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}

// mkdir makes one Owned folder that is not there yet; a rollback removes it.
func (t *txn) mkdir(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return nil
	}
	if err := os.Mkdir(path, t.s.DirPerm); err != nil {
		return err
	}
	t.undo = append(t.undo, func() error { _ = os.Remove(path); return nil })
	return nil
}

// swapFolder renames the staged folder to f.Path. With a Backup the folder
// there waits at Backup and returns when the swap fails; a failed return
// keeps the staging folder that may hold it.
func (t *txn) swapFolder(f File) error {
	if f.Backup == "" {
		return os.Rename(f.Folder, f.Path)
	}
	if f.Stale {
		_ = os.RemoveAll(f.Backup)
	}
	if err := os.Rename(f.Path, f.Backup); err != nil {
		return err
	}
	if err := os.Rename(f.Folder, f.Path); err != nil {
		if restore := os.Rename(f.Backup, f.Path); restore != nil && t.s.Words.RestoreFolder != "" {
			t.retained = true
			return say(t.s.Words.RestoreFolder, err, f.Backup, restore)
		}
		return err
	}
	_ = os.RemoveAll(f.Backup)
	return nil
}

// end finishes the transaction: a failure in publication rolls back in
// reverse order, then staged files and backups are removed.
func (t *txn) end(primary error, published bool) (Result, error) {
	failures := []error{primary}
	if primary != nil && published {
		for k := len(t.undo) - 1; k >= 0; k-- {
			if err := t.undo[k](); err != nil {
				failures = append(failures, err)
			}
		}
	}
	if t.s.Words.Cleanup != "" {
		// A reported cleanup removes only what a failure left; recovery
		// before the next write retries what the system refused.
		if primary != nil {
			for _, path := range t.staged {
				if path == "" {
					continue
				}
				if err := t.ops.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
					failures = append(failures, say(t.s.Words.Cleanup, filepath.Base(path), err))
				}
			}
		}
		t.retire(primary == nil)
	} else {
		for _, path := range append(t.staged, t.backups...) {
			if path != "" {
				_ = t.ops.Remove(path)
			}
		}
		t.retire(false)
	}
	if t.s.Discard != "" && !t.retained {
		_ = os.RemoveAll(t.s.Discard)
	}
	if primary == nil {
		return t.res, nil
	}
	return t.res, joined(failures)
}

// NewStage makes a new hidden folder in parent for files a Set later renames
// into place.
func NewStage(parent, pattern string) (string, error) { return os.MkdirTemp(parent, pattern) }

func joined(failures []error) error {
	if len(failures) == 1 {
		return failures[0]
	}
	return errors.Join(failures...)
}

// say formats a Words text with as many of args as it has verbs, or with all
// of them when it indexes them; an empty text is the last argument when that
// is an error.
func say(format string, args ...any) error {
	if format == "" {
		if len(args) != 0 {
			if err, ok := args[len(args)-1].(error); ok {
				return err
			}
		}
		return errors.New("publication did not commit")
	}
	if strings.Contains(format, "%[") {
		return fmt.Errorf(format, args...)
	}
	n := strings.Count(format, "%") - 2*strings.Count(format, "%%")
	if n > len(args) {
		n = len(args)
	}
	return fmt.Errorf(format, args[:n]...)
}

var (
	tempMu     sync.Mutex
	activeTemp = make(map[uint64]activeName)
	nextTempID uint64
)

type activeName struct {
	dir  os.FileInfo
	name string
}

// recoveryError marks the refusal of temp recovery: it is returned as it is,
// without the Create words that belong to CreateTemp.
type recoveryError struct{ err error }

func (e *recoveryError) Error() string { return e.err.Error() }

// createRegistered recovers, creates and registers one staged file under
// tempMu; a panicking CreateTemp releases the lock.
func (t *txn) createRegistered(dir string) (TempFile, error) {
	tempMu.Lock()
	defer tempMu.Unlock()
	if err := t.recover(); err != nil {
		return nil, &recoveryError{err}
	}
	tmp, err := t.ops.CreateTemp(dir, t.s.Temp)
	if err != nil {
		return nil, err
	}
	nextTempID++
	t.ids = append(t.ids, nextTempID)
	activeTemp[nextTempID] = activeName{dir: t.s.DirInfo, name: filepath.Base(tmp.Name())}
	return tmp, nil
}

// retire ends the registration of this transaction's staged files; reap
// then recovers what earlier writers left.
func (t *txn) retire(reap bool) {
	if len(t.ids) == 0 {
		return
	}
	tempMu.Lock()
	defer tempMu.Unlock()
	for _, id := range t.ids {
		delete(activeTemp, id)
	}
	t.ids = nil
	if reap {
		_ = t.recover()
	}
}

// recover removes only this process's hidden sibling namespace. A prior
// failure can leave one there only when the operating system refused cleanup.
// An in-process registry excludes active callers, and another process has a
// different owner token and remains untouched. Recovery runs before a new
// temporary file is created, so repeated refusals do not accumulate files or
// reach final-name publication. The caller holds tempMu.
func (t *txn) recover() error {
	entries, err := os.ReadDir(t.s.Dir)
	if err != nil {
		return fmt.Errorf("inspect temporary saves before writing: %w", err)
	}
	var failures []error
	for _, entry := range entries {
		if entry.IsDir() || !t.s.Recover(entry.Name()) {
			continue
		}
		path := filepath.Join(t.s.Dir, entry.Name())
		if tempIsActive(t.s.DirInfo, entry.Name()) {
			continue
		}
		if err := t.ops.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, fmt.Errorf("remove hidden temporary save %s: %w", entry.Name(), err))
		}
	}
	if len(failures) != 0 {
		return fmt.Errorf("recover temporary saves before writing; no new save was started: %w", errors.Join(failures...))
	}
	return nil
}

func tempIsActive(dir os.FileInfo, name string) bool {
	for _, active := range activeTemp {
		if os.SameFile(dir, active.dir) && SameName(name, active.name) {
			return true
		}
	}
	return false
}
