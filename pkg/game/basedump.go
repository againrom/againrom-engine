package game

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"againrom/pkg/mod"
)

// A base dump is a mod folder holding layer 0 of every layered catalog of
// the install, in the format a mod reads: the manifest, an entry script that
// adds each layer file the base reads, and the layer files. A modder copies
// it, keeps the records to change and deletes the rest.

// The refusals of a base dump's folder. Each is wrapped with the path it
// names; nothing is written on a refusal.
var (
	ErrBaseDumpInstall  = errors.New("a base dump never writes into an install")
	ErrBaseDumpNotEmpty = errors.New("a base dump writes only into a new or empty folder")
	ErrBaseDumpExists   = errors.New("a base dump never writes over an existing file")
	ErrBaseDumpName     = errors.New("the folder's name is the dump mod's id: 1 to 64 of a-z, 0-9, '.', '_', '-', starting with a letter or digit")
)

// baseDumpLayer is one layered catalog's layer 0 as the file a mod carries.
// read is false when the base takes no layer of the kind: the file is
// written for reading, the entry script does not add it, and note says so.
type baseDumpLayer struct {
	file string
	data []byte
	read bool
	note string
}

// baseDumpLayers are the layered catalogs a base dump writes, each by its
// kind's own code beside its layer step, in this order.
var baseDumpLayers = []func(*FrontEnd, string) (baseDumpLayer, error){
	(*FrontEnd).baseTownLayer,
}

// BaseDumpReport is what a base dump wrote: its folder, the base it was
// taken from, the files relative to the folder in the order written, and a
// line for each layer the base does not read.
type BaseDumpReport struct {
	Dir   string
	Base  string
	Files []string
	Notes []string
}

// WriteBaseDump writes the base dump of the front end's install into dir, a
// new or empty folder whose name becomes the dump mod's id. A folder inside
// an install, a folder that is not empty, an existing file and a name that
// is not a mod id are refused before anything is written.
func (f *FrontEnd) WriteBaseDump(dir string) (BaseDumpReport, error) {
	if f == nil || f.Archives == nil || f.Archives.Root == "" {
		return BaseDumpReport{}, errors.New("a base dump needs an opened install")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return BaseDumpReport{}, err
	}
	report := BaseDumpReport{Dir: abs, Base: BaseID(InspectInstall(f.Archives.Root))}
	if err := refuseBaseDumpInInstall(f.Archives.Root, abs); err != nil {
		return BaseDumpReport{}, err
	}
	created, err := baseDumpFolderState(abs)
	if err != nil {
		return BaseDumpReport{}, err
	}
	id := filepath.Base(abs)
	if !mod.ValidID(id) {
		return BaseDumpReport{}, fmt.Errorf("%s: %w", abs, ErrBaseDumpName)
	}
	var read, unread []string
	files := map[string][]byte{}
	var order []string
	for _, kind := range baseDumpLayers {
		l, err := kind(f, report.Base)
		if err != nil {
			return BaseDumpReport{}, fmt.Errorf("base dump: %w", err)
		}
		files[l.file], order = l.data, append(order, l.file)
		if l.read {
			read = append(read, l.file)
		} else {
			unread = append(unread, l.file)
			report.Notes = append(report.Notes, l.note)
		}
	}
	files[mod.ManifestName] = baseDumpManifest(id, report.Base)
	files["main.star"] = baseDumpEntry(read, unread)
	order = append([]string{mod.ManifestName, "main.star"}, order...)
	if err := writeNewFiles(abs, created, order, files); err != nil {
		return BaseDumpReport{}, err
	}
	report.Files = order
	return report, nil
}

// refuseBaseDumpInInstall refuses a folder inside the install at root, inside
// a preserved install beside it, or below any folder that holds an install's
// archives.
func refuseBaseDumpInInstall(root, dir string) error {
	install, inside, err := insideInstall(root, dir)
	if err != nil {
		return err
	}
	if inside {
		return fmt.Errorf("%s is inside the install %s: %w", dir, install, ErrBaseDumpInstall)
	}
	return nil
}

// insideInstall reports whether the absolute path dir is inside the install
// at root, inside a preserved install beside it, or below any folder that
// holds an install's archives, and which install that is.
func insideInstall(root, dir string) (install string, inside bool, err error) {
	resolvedRoot, err := resolvePathForSaveSafety(root)
	if err != nil {
		return "", false, err
	}
	for _, r := range append([]string{root}, preservedPairRoots(resolvedRoot)...) {
		inside, err := pathWithin(r, dir)
		if err != nil {
			return "", false, err
		}
		if inside {
			return r, true, nil
		}
	}
	resolved, err := resolvePathForSaveSafety(dir)
	if err != nil {
		return "", false, err
	}
	return requiredArchiveAncestor(resolved)
}

// baseDumpFolderState refuses an existing file at dir and a folder there
// that holds anything. It reports whether the folder is still to be made.
func baseDumpFolderState(dir string) (missing bool, err error) {
	st, err := os.Lstat(dir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return true, nil
	case err != nil:
		return false, err
	case !st.IsDir():
		return false, fmt.Errorf("%s is a file: %w", dir, ErrBaseDumpExists)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	if len(entries) > 0 {
		return false, fmt.Errorf("%s holds %s: %w", dir, entries[0].Name(), ErrBaseDumpNotEmpty)
	}
	return false, nil
}

// writeNewFiles creates each file of order, a slash path inside dir, with
// its bytes; dir is made first when missing. A file already there is
// refused. On any failure the files and folders made so far are removed.
func writeNewFiles(dir string, missing bool, order []string, files map[string][]byte) (err error) {
	var made []string
	defer func() {
		if err == nil {
			return
		}
		for i := len(made) - 1; i >= 0; i-- {
			os.Remove(made[i])
		}
	}()
	mkdir := func(p string) error {
		if _, err := os.Lstat(p); err == nil {
			return nil
		}
		if err := os.Mkdir(p, 0o755); err != nil {
			return err
		}
		made = append(made, p)
		return nil
	}
	if missing {
		if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
			return err
		}
		if err := mkdir(dir); err != nil {
			return err
		}
	}
	for _, rel := range order {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		for _, sub := range baseDumpParents(rel) {
			if err := mkdir(filepath.Join(dir, filepath.FromSlash(sub))); err != nil {
				return err
			}
		}
		out, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%s: %w", full, ErrBaseDumpExists)
		}
		if err != nil {
			return err
		}
		made = append(made, full)
		_, werr := out.Write(files[rel])
		if cerr := out.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return werr
		}
	}
	return nil
}

// baseDumpParents are the folders of slash path rel, outermost first.
func baseDumpParents(rel string) []string {
	var out []string
	for d := path.Dir(rel); d != "." && d != "/"; d = path.Dir(d) {
		out = append([]string{d}, out...)
	}
	return out
}

// baseDumpManifest is the dump mod's mod.toml.
func baseDumpManifest(id, baseID string) []byte {
	return []byte(fmt.Sprintf(`# The base dump of %[2]s: layer 0 of each layered catalog of this install,
# in the format a mod reads. To make a mod, copy this folder under a new
# name, set id to that name, keep the records you change and delete the rest.
id         = %[1]q
title      = "Base dump of %[2]s"
version    = "1.0.0"
api        = 1
applies-to = [%[2]q]
entry      = "main.star"
`, id, baseID))
}

// baseDumpEntry is the dump mod's entry script: it adds each file of read;
// the files of unread are named in a comment.
func baseDumpEntry(read, unread []string) []byte {
	var b strings.Builder
	b.WriteString("# Adds each layer file of this mod.\ndef init(game, settings):\n")
	for _, file := range unread {
		fmt.Fprintf(&b, "    # %s is for reading: this game reads no such layer.\n", file)
	}
	for _, file := range read {
		fmt.Fprintf(&b, "    game.data.add(%q)\n", file)
	}
	if len(read) == 0 {
		b.WriteString("    pass\n")
	}
	return []byte(b.String())
}
