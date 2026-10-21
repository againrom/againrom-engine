package mod

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/formats/res"
)

// Pack is the mod folder dir packed into one RES file, and its manifest.
// Every folder and regular file of the mod goes in at its path, except names
// that start with a dot, which no mod reader takes. Two paths equal without
// case (foldKey: ASCII and CP866 Cyrillic), which a packed mod could not tell
// apart, are refused naming both. A link or another
// non-regular file, a file over MaxInstallFileBytes, and a name the RES
// cannot hold (over res.MaxNameBytes bytes, or a character CP866 has no byte
// for) are refused naming the file. The packed tree is read back and its
// digest must equal the folder's.
func Pack(dir string) (Manifest, []byte, error) {
	t, err := OpenTree(dir)
	if err != nil {
		return Manifest{}, nil, err
	}
	if t.Packed() {
		return Manifest{}, nil, fmt.Errorf("%s is a packed mod already; pack takes a mod folder", dir)
	}
	e := Load(dir)
	if e.Err != nil {
		return Manifest{}, nil, fmt.Errorf("mod folder %s: %v", dir, e.Err)
	}
	id := e.Manifest.ID
	var entries []res.WriteEntry
	err = t.WalkDir(func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		bad := func(format string, args ...any) error {
			return Source{Mod: id, File: rel}.Errorf(format, args...)
		}
		switch {
		case d.IsDir():
			entries = append(entries, res.WriteEntry{Path: rel, Dir: true})
			return nil
		case !d.Type().IsRegular():
			return bad("is a link or another non-regular file; a packed mod holds folders and files")
		}
		info, err := d.Info()
		if err != nil {
			return bad("%v", err)
		}
		if info.Size() > MaxInstallFileBytes {
			return bad("is larger than %d bytes, more than any mod reader takes", MaxInstallFileBytes)
		}
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return bad("%v", err)
		}
		entries = append(entries, res.WriteEntry{Path: rel, Data: b})
		return nil
	})
	if err != nil {
		return Manifest{}, nil, err
	}
	if err := refuseFoldTwins(id, entries); err != nil {
		return Manifest{}, nil, err
	}
	b, err := res.Write(entries)
	var name *res.NameError
	if errors.As(err, &name) {
		return Manifest{}, nil, Source{Mod: id, File: name.Path}.Errorf("cannot be packed: %s", name.Reason)
	}
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("mod %q: %w", id, err)
	}
	packed, err := openPackedBytes(PackedName(id), b)
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("mod %q: the packed file does not read back: %w", id, err)
	}
	want, err := t.Digest()
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("mod %q: %w", id, err)
	}
	if got, err := packed.Digest(); err != nil || got != want {
		return Manifest{}, nil, fmt.Errorf("mod %q: the packed file reads back as other files than the folder (%v)", id, err)
	}
	return e.Manifest, b, nil
}

// refuseFoldTwins refuses two entries whose paths are one name under foldKey,
// naming both.
func refuseFoldTwins(id string, entries []res.WriteEntry) error {
	seen := map[string]string{}
	for _, e := range entries {
		k := foldKey(e.Path)
		if other, dup := seen[k]; dup {
			return Source{Mod: id, File: e.Path}.Errorf("cannot be packed: names the same file as %s; names compare without case", other)
		}
		seen[k] = e.Path
	}
	return nil
}
