package mod

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"againrom/pkg/formats/res"
	"againrom/pkg/namefold"
)

// Pack is the mod folder dir packed into one RES file, and its manifest.
// Every folder and regular file of the mod goes in at its path, except names
// that start with a dot, which no mod reader takes. Two paths equal without
// case (namefold.ModTree: ASCII and CP866 Cyrillic), which a packed mod could not tell
// apart, are refused naming both. A link or another
// non-regular file, a file over MaxInstallFileBytes, or an invalid component
// is refused naming the file. Names the RES cannot hold use a name table.
// The packed tree is read back and its digest must equal the folder's.
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
	used := map[string]bool{}
	for _, e := range entries {
		used[namefold.ModTree(e.Path)] = true
	}
	names := packedNames{Version: 1}
	encoded := map[string]string{".": ""}
	serial := 0
	for i := range entries {
		e := &entries[i]
		original := e.Path
		parent, name := path.Dir(original), path.Base(original)
		if err := checkPackedComponent(name); err != nil {
			return Manifest{}, nil, Source{Mod: id, File: original}.Errorf("cannot be packed: %v", err)
		}
		short := name
		if _, err := res.EncodeName(name); len(name) > res.MaxNameBytes || err != nil {
			for {
				serial++
				short = fmt.Sprintf("~%014x", serial)
				key := namefold.ModTree(path.Join(parent, short))
				if !used[key] {
					used[key] = true
					break
				}
			}
			names.Entries = append(names.Entries, packedName{Path: path.Join(encoded[parent], short), Name: name})
		}
		e.Path = path.Join(encoded[parent], short)
		encoded[original] = e.Path
	}
	if len(names.Entries) > 0 {
		data, err := json.Marshal(names)
		if err != nil {
			return Manifest{}, nil, err
		}
		if len(data) > MaxInstallFileBytes {
			return Manifest{}, nil, Source{Mod: id, File: packedNamesFile}.Errorf("name table is larger than %d bytes", MaxInstallFileBytes)
		}
		entries = append(entries, res.WriteEntry{Path: packedNamesFile, Data: data})
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

// refuseFoldTwins refuses two entries whose paths are one name under
// namefold.ModTree, naming both.
func refuseFoldTwins(id string, entries []res.WriteEntry) error {
	seen := map[string]string{}
	for _, e := range entries {
		k := namefold.ModTree(e.Path)
		if other, dup := seen[k]; dup {
			return Source{Mod: id, File: e.Path}.Errorf("cannot be packed: names the same file as %s; names compare without case", other)
		}
		seen[k] = e.Path
	}
	return nil
}
