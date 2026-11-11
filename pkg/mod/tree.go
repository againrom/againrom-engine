package mod

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"againrom/pkg/formats/res"
	"againrom/pkg/namefold"
)

// PackedPrefix and PackedExt frame the file name of a packed mod in the mods
// directory: mod-<id>.res.
const (
	PackedPrefix = "mod-"
	PackedExt    = ".res"
)

// PackedName is the file name of mod id packed into one RES file.
func PackedName(id string) string { return PackedPrefix + id + PackedExt }

// packedID is the id a packed mod's file name names, when the name is
// mod-<id>.res with a valid id.
func packedID(name string) (string, bool) {
	if !strings.HasPrefix(name, PackedPrefix) || !strings.HasSuffix(name, PackedExt) {
		return "", false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(name, PackedPrefix), PackedExt)
	return id, ValidID(id)
}

// looksPacked reports whether a file of the mods directory is meant as a
// packed mod: its name, case folded, is mod*.res.
func looksPacked(name string) bool {
	low := namefold.Host(name)
	return strings.HasPrefix(low, "mod") && strings.HasSuffix(low, PackedExt)
}

// Tree is the files of one mod: the folder the mod is, or the RES file it is
// packed into. Every reader of a mod's files reads through a Tree, so a
// packed mod is the same tree under the same checks as its folder. Paths are
// slash-separated and relative to the mod's top. In a packed mod a name
// matches without case (ASCII and CP866 Cyrillic, namefold.ModTree), and a
// folder's name matches the same way, not as the host file system folds it. A path with a component starting with a dot is refused by every
// reader, in a folder and in a packed mod.
type Tree struct {
	path   string
	packed *packedTree
}

// OpenTree opens the mod at path: a folder, or a packed mod's RES file. A
// path that is not a regular file is read as a folder, so a missing folder
// fails where it is read, as before.
func OpenTree(path string) (Tree, error) {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return Tree{path: path}, nil
	}
	a, err := res.OpenFileIndex(path)
	if err != nil {
		return Tree{}, fmt.Errorf("%s: %w", path, err)
	}
	p, err := newPackedTree(a)
	if err != nil {
		return Tree{}, fmt.Errorf("%s: %w", path, err)
	}
	return Tree{path: path, packed: p}, nil
}

// openPackedBytes is the tree of a packed mod held in memory.
func openPackedBytes(name string, b []byte) (Tree, error) {
	a, err := res.OpenBytes(b)
	if err != nil {
		return Tree{}, err
	}
	p, err := newPackedTree(a)
	if err != nil {
		return Tree{}, err
	}
	return Tree{path: name, packed: p}, nil
}

// Path is the mod's folder or RES file.
func (t Tree) Path() string { return t.path }

// Packed reports whether the mod is a RES file.
func (t Tree) Packed() bool { return t.packed != nil }

// host is the file system path of name in a folder, found by the mod tree's
// own name rule: each component is looked up in the folder's listing, an exact
// name first, then a name equal under namefold.ModTree. A name the listing
// does not hold under that rule is not found, whatever the host's own folding
// would open, so a folder and its pack answer one way.
func (t Tree) host(name string) (string, error) {
	cur := t.path
	if name == "." || name == "" {
		return cur, nil
	}
	for _, c := range strings.Split(filepath.ToSlash(name), "/") {
		if c == "" || c == "." {
			continue
		}
		entries, err := os.ReadDir(cur)
		if err != nil {
			return "", &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
		found, fold := "", ""
		key := namefold.ModTree(c)
		for _, e := range entries {
			if e.Name() == c {
				found = c
				break
			}
			if fold == "" && namefold.ModTree(e.Name()) == key {
				fold = e.Name()
			}
		}
		if found == "" {
			found = fold
		}
		if found == "" {
			return "", &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
		cur = filepath.Join(cur, found)
	}
	return cur, nil
}

// ReadFile reads the file name, unbounded and without ReadInside's checks;
// the manifest and the settings are read so.
func (t Tree) ReadFile(name string) ([]byte, error) {
	if err := refuseDotted(name); err != nil {
		return nil, err
	}
	if t.packed == nil {
		h, err := t.host(name)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(h)
	}
	return t.packed.readFile(name)
}

// Stat describes name, following a folder's links.
func (t Tree) Stat(name string) (fs.FileInfo, error) {
	if err := refuseDotted(name); err != nil {
		return nil, err
	}
	if t.packed == nil {
		h, err := t.host(name)
		if err != nil {
			return nil, err
		}
		return os.Stat(h)
	}
	return t.packed.stat(name)
}

// Lstat describes name without following a folder's links.
func (t Tree) Lstat(name string) (fs.FileInfo, error) {
	if err := refuseDotted(name); err != nil {
		return nil, err
	}
	if t.packed == nil {
		h, err := t.host(name)
		if err != nil {
			return nil, err
		}
		return os.Lstat(h)
	}
	return t.packed.stat(name)
}

// ReadDir lists the folder name, sorted by name.
func (t Tree) ReadDir(name string) ([]fs.DirEntry, error) {
	if err := refuseDotted(name); err != nil {
		return nil, err
	}
	if t.packed == nil {
		h, err := t.host(name)
		if err != nil {
			return nil, err
		}
		return os.ReadDir(h)
	}
	return t.packed.readDir(name)
}

// FS is the tree as a file system rooted at the mod's top.
func (t Tree) FS() fs.FS {
	if t.packed == nil {
		return os.DirFS(t.path)
	}
	return t.packed
}

// WalkDir walks the whole tree in name order, giving fn each relative slash
// path ("." for the top). A folder is walked as filepath.WalkDir walks it.
func (t Tree) WalkDir(fn fs.WalkDirFunc) error {
	if t.packed != nil {
		return fs.WalkDir(t.packed, ".", fn)
	}
	return filepath.WalkDir(t.path, func(p string, d fs.DirEntry, err error) error {
		rel, rerr := filepath.Rel(t.path, p)
		if rerr != nil {
			return rerr
		}
		return fn(filepath.ToSlash(rel), d, err)
	})
}

// ReadInside reads a regular file within the mod, returning its relative
// slash path. Absolute paths, backslashes and paths that resolve outside the
// mod are refused, and so is a file over MaxFileBytes.
func (t Tree) ReadInside(name string) ([]byte, string, error) {
	if name == "" || strings.ContainsAny(name, "\\:\x00") || path.IsAbs(name) || strings.HasPrefix(name, "/") {
		return nil, "", fmt.Errorf("%q is not a path inside the mod folder", name)
	}
	rel := path.Clean(name)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return nil, "", fmt.Errorf("%q is not a path inside the mod folder", name)
	}
	if err := refuseDotted(rel); err != nil {
		return nil, "", err
	}
	if t.packed != nil {
		n, ok := t.packed.lookup(rel)
		switch {
		case !ok:
			return nil, "", fmt.Errorf("%s: no such file in the mod folder", rel)
		case n.Dir:
			return nil, "", fmt.Errorf("%s is not a file", rel)
		case n.Size > MaxFileBytes:
			return nil, "", fmt.Errorf("%s is larger than %d bytes", rel, MaxFileBytes)
		}
		data, err := t.packed.read(n)
		return data, rel, err
	}
	full, err := t.host(rel)
	if err != nil {
		return nil, "", fmt.Errorf("%s: no such file in the mod folder", rel)
	}
	root, err := resolvedPath(t.path)
	if err != nil {
		return nil, "", err
	}
	real, err := resolvedPath(full)
	if err != nil {
		return nil, "", fmt.Errorf("%s: no such file in the mod folder", rel)
	}
	if real != root && !strings.HasPrefix(real, root+string(filepath.Separator)) {
		return nil, "", fmt.Errorf("%q leads outside the mod folder", name)
	}
	st, err := os.Stat(real)
	if err != nil || !st.Mode().IsRegular() {
		return nil, "", fmt.Errorf("%s is not a file", rel)
	}
	if st.Size() > MaxFileBytes {
		return nil, "", fmt.Errorf("%s is larger than %d bytes", rel, MaxFileBytes)
	}
	data, err := os.ReadFile(real)
	return data, rel, err
}

// packedArchive is what a packed tree reads: the RES index, from a file or
// from memory.
type packedArchive interface {
	Tree() ([]res.Node, []int)
	ReadNode(i int) ([]byte, error)
}

// packedNode is one node of a packed tree.
type packedNode struct {
	index    int
	name     string
	path     string
	Dir      bool
	Size     int64
	children []*packedNode // sorted by name
}

// packedTree is a packed mod's RES read as a tree of names. It implements
// fs.FS, fs.ReadDirFS, fs.ReadFileFS and fs.StatFS.
type packedTree struct {
	a     packedArchive
	root  *packedNode
	byKey map[string]*packedNode // path folded by namefold.ModTree
}

// newPackedTree indexes a RES. A name that is empty, "." or "..", or holds
// a slash or a backslash, and two names of one folder that differ only by
// case (namefold.ModTree) are refused: no folder can hold them.
func newPackedTree(a packedArchive) (*packedTree, error) {
	nodes, roots := a.Tree()
	table := -1
	aliases := map[string]string{}
	for _, i := range roots {
		n := nodes[i]
		if namefold.ModTree(n.Name) != packedNamesFile {
			continue
		}
		bad := func(reason any) error {
			return fmt.Errorf("%s: reserved for the packed mod name table: %v", packedNamesFile, reason)
		}
		if table >= 0 || n.Dir || n.Size > MaxInstallFileBytes {
			return nil, bad("expected one file of at most 16 MiB")
		}
		table = i
		data, err := a.ReadNode(i)
		if err != nil {
			return nil, bad(err)
		}
		var names packedNames
		if !utf8.Valid(data) {
			return nil, bad("the table is not UTF-8 text")
		}
		if err := json.Unmarshal(data, &names); err != nil {
			return nil, bad(err)
		}
		if names.Version != 1 || len(names.Entries) == 0 || len(names.Entries) > len(nodes)-1 {
			return nil, bad("expected version 1 and one entry per renamed node")
		}
		for _, entry := range names.Entries {
			if !fs.ValidPath(entry.Path) || entry.Path == "." || strings.Contains(entry.Path, `\`) {
				return nil, bad(fmt.Sprintf("%q is not an archive path", entry.Path))
			}
			if err := checkPackedComponent(entry.Name); err != nil {
				return nil, bad(err)
			}
			if _, dup := aliases[entry.Path]; dup {
				return nil, bad(fmt.Sprintf("%s is given twice", entry.Path))
			}
			aliases[entry.Path] = entry.Name
		}
	}
	t := &packedTree{a: a, root: &packedNode{index: -1, name: ".", path: ".", Dir: true}, byKey: map[string]*packedNode{}}
	var add func(parent *packedNode, rawParent string, kids []int) error
	add = func(parent *packedNode, rawParent string, kids []int) error {
		for _, i := range kids {
			if i == table {
				continue
			}
			n := nodes[i]
			if n.Name == "" || n.Name == "." || n.Name == ".." || strings.ContainsAny(n.Name, `/\`) {
				return fmt.Errorf("the archive holds the name %q, which no mod folder can hold", n.Name)
			}
			rawPath := path.Join(rawParent, n.Name)
			if name, ok := aliases[rawPath]; ok {
				n.Name = name
				delete(aliases, rawPath)
			}
			if parent == t.root && namefold.ModTree(n.Name) == packedNamesFile {
				return fmt.Errorf("%s is reserved for the packed mod name table", n.Name)
			}
			p := n.Name
			if parent.path != "." {
				p = parent.path + "/" + n.Name
			}
			key := namefold.ModTree(p)
			if other, dup := t.byKey[key]; dup {
				return fmt.Errorf("the archive holds %s and %s, one name in a folder", other.path, p)
			}
			c := &packedNode{index: i, name: n.Name, path: p, Dir: n.Dir, Size: n.Size}
			t.byKey[key] = c
			parent.children = append(parent.children, c)
			if n.Dir {
				if err := add(c, rawPath, n.Children); err != nil {
					return err
				}
			}
		}
		sort.Slice(parent.children, func(i, j int) bool { return parent.children[i].name < parent.children[j].name })
		return nil
	}
	if err := add(t.root, "", roots); err != nil {
		return nil, err
	}
	if len(aliases) != 0 {
		return nil, fmt.Errorf("%s: a name table entry names no archive node", packedNamesFile)
	}
	return t, nil
}

// dottedPath reports whether a clean relative path has a component starting
// with a dot, other than "." itself: no mod reader takes such a name.
func dottedPath(p string) bool {
	for _, c := range strings.Split(p, "/") {
		if c != "." && strings.HasPrefix(c, ".") {
			return true
		}
	}
	return false
}

func refuseDotted(p string) error {
	if dottedPath(filepath.ToSlash(p)) {
		return fmt.Errorf("%s: a name starting with a dot is not read from a mod", filepath.ToSlash(p))
	}
	return nil
}

// lookup finds the clean relative path name.
func (t *packedTree) lookup(name string) (*packedNode, bool) {
	if name == "." {
		return t.root, true
	}
	n, ok := t.byKey[namefold.ModTree(name)]
	return n, ok
}

func (t *packedTree) find(op, name string) (*packedNode, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}
	n, ok := t.lookup(name)
	if !ok {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrNotExist}
	}
	return n, nil
}

func (t *packedTree) read(n *packedNode) ([]byte, error) {
	b, err := t.a.ReadNode(n.index)
	if err != nil {
		return nil, &fs.PathError{Op: "read", Path: n.path, Err: err}
	}
	return b, nil
}

func (t *packedTree) readFile(name string) ([]byte, error) {
	n, err := t.find("read", name)
	if err != nil {
		return nil, err
	}
	if n.Dir {
		return nil, &fs.PathError{Op: "read", Path: name, Err: errors.New("is a folder")}
	}
	return t.read(n)
}

func (t *packedTree) stat(name string) (fs.FileInfo, error) {
	n, err := t.find("stat", name)
	if err != nil {
		return nil, err
	}
	return packedInfo{n}, nil
}

func (t *packedTree) readDir(name string) ([]fs.DirEntry, error) {
	n, err := t.find("readdir", name)
	if err != nil {
		return nil, err
	}
	if !n.Dir {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: errors.New("not a folder")}
	}
	out := make([]fs.DirEntry, len(n.children))
	for i, c := range n.children {
		out[i] = fs.FileInfoToDirEntry(packedInfo{c})
	}
	return out, nil
}

// Open, ReadDir, ReadFile and Stat make the tree an fs.FS.
func (t *packedTree) Open(name string) (fs.File, error) {
	n, err := t.find("open", name)
	if err != nil {
		return nil, err
	}
	if n.Dir {
		return &packedDir{t: t, n: n}, nil
	}
	b, err := t.read(n)
	if err != nil {
		return nil, err
	}
	return &packedFile{n: n, r: bytes.NewReader(b)}, nil
}

func (t *packedTree) ReadDir(name string) ([]fs.DirEntry, error) { return t.readDir(name) }
func (t *packedTree) ReadFile(name string) ([]byte, error)       { return t.readFile(name) }
func (t *packedTree) Stat(name string) (fs.FileInfo, error)      { return t.stat(name) }
func (t *packedTree) String() string                             { return "packed mod" }
func (f *packedFile) Stat() (fs.FileInfo, error)                 { return packedInfo{f.n}, nil }
func (f *packedFile) Read(b []byte) (int, error)                 { return f.r.Read(b) }
func (f *packedFile) Close() error                               { return nil }
func (d *packedDir) Stat() (fs.FileInfo, error)                  { return packedInfo{d.n}, nil }
func (d *packedDir) Close() error                                { return nil }
func (d *packedDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.n.path, Err: errors.New("is a folder")}
}
func (i packedInfo) Name() string       { return i.n.name }
func (i packedInfo) Size() int64        { return i.n.Size }
func (i packedInfo) ModTime() time.Time { return time.Time{} }
func (i packedInfo) IsDir() bool        { return i.n.Dir }
func (i packedInfo) Sys() any           { return nil }
func (i packedInfo) Mode() fs.FileMode {
	if i.n.Dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}

type packedFile struct {
	n *packedNode
	r *bytes.Reader
}

type packedDir struct {
	t    *packedTree
	n    *packedNode
	done int
}

type packedInfo struct{ n *packedNode }

// ReadDir lists the folder's entries; n > 0 reads at most n at a time.
func (d *packedDir) ReadDir(n int) ([]fs.DirEntry, error) {
	rest := d.n.children[d.done:]
	if n > 0 && len(rest) > n {
		rest = rest[:n]
	}
	if n > 0 && len(rest) == 0 {
		return nil, io.EOF
	}
	d.done += len(rest)
	out := make([]fs.DirEntry, len(rest))
	for i, c := range rest {
		out[i] = fs.FileInfoToDirEntry(packedInfo{c})
	}
	return out, nil
}

// Digest is ContentDigest of the tree.
func (t Tree) Digest() (string, error) {
	var paths []string
	err := t.WalkDir(func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if p != "." && strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() || !d.Type().IsRegular() || namefold.HostEqual(name, "README.md") && path.Dir(p) == "." {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, p := range paths {
		data, err := t.ReadFile(p)
		if err != nil {
			return "", err
		}
		switch namefold.Host(path.Ext(p)) {
		case ".toml", ".star", ".txt", ".md":
			data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		}
		fmt.Fprintf(h, "%s\x00%d\x00", p, len(data))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
