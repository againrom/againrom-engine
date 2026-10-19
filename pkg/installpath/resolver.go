// Package installpath maps a name relative to an install root onto the file
// that stands on disk, whatever the case of its name.
//
// The two lawful first-game installs disagree about case (EN ships
// graphics.res, RU ships GRAPHICS.RES), and a copy of either on a
// case-sensitive filesystem keeps the spelling it was shipped with. Every
// reader of an install file builds its host path here, so the answer does not
// depend on whether the host filesystem folds case itself.
package installpath

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ErrAmbiguous is matched by every AmbiguousError.
var ErrAmbiguous = errors.New("names differ only by case")

// AmbiguousError reports a requested name that no directory entry spells
// exactly while two or more entries equal it with case folded. The resolver
// never picks one of them.
type AmbiguousError struct {
	Dir   string   // the directory holding the entries, as the resolver names it
	Name  string   // the requested segment
	Names []string // every entry equal to Name with case folded, sorted
}

func (e *AmbiguousError) Error() string {
	quoted := make([]string, len(e.Names))
	for i, n := range e.Names {
		quoted[i] = fmt.Sprintf("%q", n)
	}
	return fmt.Sprintf("%s: %q matches %s, which differ only by case", e.Dir, e.Name, strings.Join(quoted, " and "))
}

// Is makes errors.Is(err, ErrAmbiguous) hold.
func (e *AmbiguousError) Is(target error) bool { return target == ErrAmbiguous }

// Entry is one resolved name: its slash-separated path relative to the root,
// spelt as the disk spells it, and whether it is a directory.
type Entry struct {
	Rel string
	Dir bool
}

// Resolver resolves names under one root. Each directory is listed once and
// the listing kept; a name the kept listing lacks lists that directory again
// once before it is reported absent, so a file added after the first listing
// is still found. A Resolver is safe for concurrent use.
type Resolver struct {
	root string
	read func(dir string) ([]fs.DirEntry, error)
	open func(rel string) (fs.File, error)
	stat func(rel string) (fs.FileInfo, error)

	mu    sync.Mutex
	lists map[string]*listing
}

// listing is one directory's entries indexed by exact and by folded name.
type listing struct {
	exact  map[string]fs.DirEntry
	folded map[string][]fs.DirEntry
}

// New is a resolver over the host directory root. An empty root resolves
// against the working directory, as a relative path joined to it would.
func New(root string) *Resolver {
	return &Resolver{
		root: root,
		read: func(dir string) ([]fs.DirEntry, error) {
			return os.ReadDir(hostJoin(root, dir))
		},
		open: func(rel string) (fs.File, error) {
			return os.Open(hostJoin(root, rel))
		},
		stat: func(rel string) (fs.FileInfo, error) {
			return os.Stat(hostJoin(root, rel))
		},
		lists: map[string]*listing{},
	}
}

// NewFS is a resolver over fsys. Path then returns the resolved name in the
// host separator, with no root before it.
func NewFS(fsys fs.FS) *Resolver {
	return &Resolver{
		read:  func(dir string) ([]fs.DirEntry, error) { return fs.ReadDir(fsys, dir) },
		open:  func(rel string) (fs.File, error) { return fsys.Open(rel) },
		stat:  func(rel string) (fs.FileInfo, error) { return fs.Stat(fsys, rel) },
		lists: map[string]*listing{},
	}
}

// Root is the host root the resolver was made over.
func (r *Resolver) Root() string { return r.root }

func hostJoin(root, rel string) string {
	if rel == "." {
		if root == "" {
			return "."
		}
		return root
	}
	return filepath.Join(root, filepath.FromSlash(rel))
}

// Fold is the comparison form of a name: Unicode lower case. Two names are the
// same install name when their Fold values are equal.
func Fold(name string) string { return strings.ToLower(name) }

// segments splits rel, with either separator, into its names. An empty name,
// "." or ".." is refused: an install-relative name never leaves its root.
func segments(rel string) ([]string, error) {
	parts := strings.Split(strings.ReplaceAll(rel, `\`, "/"), "/")
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return nil, &fs.PathError{Op: "resolve", Path: rel, Err: fs.ErrInvalid}
		}
	}
	return parts, nil
}

// Resolve maps rel onto the entry that stands on disk, one segment at a time.
// In each directory the entry spelt exactly as requested wins; otherwise the
// one entry equal to it with case folded is taken; two or more such entries
// are an AmbiguousError. A name nothing matches is an error satisfying
// errors.Is(err, fs.ErrNotExist); a directory that cannot be listed returns
// the listing's own error.
func (r *Resolver) Resolve(rel string) (Entry, error) {
	parts, err := segments(rel)
	if err != nil {
		return Entry{}, err
	}
	dir := "."
	var last fs.DirEntry
	for _, seg := range parts {
		e, err := r.child(dir, seg)
		if err != nil {
			return Entry{}, err
		}
		dir = path.Join(dir, e.Name())
		last = e
	}
	return Entry{Rel: dir, Dir: last.IsDir()}, nil
}

func (r *Resolver) child(dir, seg string) (fs.DirEntry, error) {
	for refresh := false; ; refresh = true {
		l, err := r.list(dir, refresh)
		if err != nil {
			return nil, err
		}
		if e, ok := l.exact[seg]; ok {
			return e, nil
		}
		switch m := l.folded[Fold(seg)]; len(m) {
		case 0:
			if refresh {
				return nil, &fs.PathError{Op: "open", Path: hostJoin(r.root, path.Join(dir, seg)), Err: fs.ErrNotExist}
			}
		case 1:
			return m[0], nil
		default:
			names := make([]string, len(m))
			for i, e := range m {
				names[i] = e.Name()
			}
			sort.Strings(names)
			return nil, &AmbiguousError{Dir: hostJoin(r.root, dir), Name: seg, Names: names}
		}
	}
}

func (r *Resolver) list(dir string, refresh bool) (*listing, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.lists[dir]; ok && !refresh {
		return l, nil
	}
	entries, err := r.read(dir)
	if err != nil {
		delete(r.lists, dir)
		return nil, err
	}
	l := &listing{exact: make(map[string]fs.DirEntry, len(entries)), folded: make(map[string][]fs.DirEntry, len(entries))}
	for _, e := range entries {
		l.exact[e.Name()] = e
		k := Fold(e.Name())
		l.folded[k] = append(l.folded[k], e)
	}
	r.lists[dir] = l
	return l, nil
}

// Name is the relative name to open for rel: the resolved spelling when an
// entry matches, and the requested spelling when none does or a directory on
// the way cannot be listed, so that opening it fails, or succeeds, exactly as
// the host decides and reports. Only an ambiguous or invalid name is an error.
func (r *Resolver) Name(rel string) (string, error) {
	e, err := r.Resolve(rel)
	if err == nil {
		return e.Rel, nil
	}
	var invalid *fs.PathError
	if errors.Is(err, ErrAmbiguous) || errors.As(err, &invalid) && errors.Is(invalid.Err, fs.ErrInvalid) {
		return "", err
	}
	return strings.ReplaceAll(rel, `\`, "/"), nil
}

// Path is the host path of Name(rel) under the root.
func (r *Resolver) Path(rel string) (string, error) {
	name, err := r.Name(rel)
	if err != nil {
		return "", err
	}
	return hostJoin(r.root, name), nil
}

// Open opens Name(rel) for reading.
func (r *Resolver) Open(rel string) (fs.File, error) {
	name, err := r.Name(rel)
	if err != nil {
		return nil, err
	}
	return r.open(name)
}

// Stat describes Name(rel), following a symbolic link.
func (r *Resolver) Stat(rel string) (fs.FileInfo, error) {
	name, err := r.Name(rel)
	if err != nil {
		return nil, err
	}
	return r.stat(name)
}

// ReadFile returns the bytes of Name(rel).
func (r *Resolver) ReadFile(rel string) ([]byte, error) {
	f, err := r.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
