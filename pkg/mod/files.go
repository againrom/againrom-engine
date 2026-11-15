package mod

import (
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"againrom/pkg/locale"
	"againrom/pkg/namefold"
)

// MaxInstallFileBytes bounds one install file a mod carries.
const MaxInstallFileBytes = 16 << 20

// FileKind is what an install file is to the game: a picture, a sound or a
// text, by its extension.
type FileKind string

const (
	FilePicture FileKind = "picture"
	FileSound   FileKind = "sound"
	FileText    FileKind = "text"
)

// installFileKind reads extension admission from the tree table.
func installFileKind(ext string) (FileKind, bool) {
	for _, k := range dataKinds {
		if k.install != "" {
			for _, p := range k.Paths {
				if path.Ext(p) == ext {
					return k.install, true
				}
			}
		}
	}
	return "", false
}

// InstallFileKind is the kind of install file at address by its extension;
// false for a file the game takes no install file of.
func InstallFileKind(address string) (FileKind, bool) {
	return installFileKind(namefold.Host(path.Ext(address)))
}

// ChangesState reports whether a file of this kind can change game state.
// A text feeds the definitions the game builds its World from (the body list,
// item names, class and spell tables), so it does. A picture and a sound are
// only drawn and played: no reader on the World-hash or SAV path reads one.
func (k FileKind) ChangesState() bool { return k == FileText }

// FilesChangeState reports whether any of the files can change game state.
func FilesChangeState(files []InstallFile) bool {
	for _, f := range files {
		if f.Kind.ChangesState() {
			return true
		}
	}
	return false
}

// InstallFile is one file a mod carries for an install path.
type InstallFile struct {
	Mod string
	// File is the file's path in the mod folder, slash-separated.
	File string
	// Lang is the language folder the file stands in: CommonFiles or a
	// language code.
	Lang string
	// Address is the install path the file shadows, as the folder spells it:
	// the archive's name, a slash and the path inside the archive.
	Address string
	Kind    FileKind
	Bytes   []byte
}

// Errorf is a refusal of the file, naming the mod and the file.
func (f InstallFile) Errorf(format string, args ...any) error {
	return Source{Mod: f.Mod, File: f.File}.Errorf(format, args...)
}

// FileData is the install files the mods carry for one install's language,
// in load order, with the order the mods state.
type FileData struct {
	Files []InstallFile
	Order Precedence
}

// Empty reports whether no mod carries an install file.
func (d FileData) Empty() bool { return len(d.Files) == 0 }

// FoldAddress is an install path as the archives compare it: ASCII letters
// folded, backslashes read as slashes.
func FoldAddress(a string) string {
	return namefold.Archive(strings.ReplaceAll(a, `\`, "/"))
}

// ReadInstallFiles reads the files/ folder of mod id at dir and answers the
// files that apply on an install of language lang: per install path the
// language folder's file, else the common folder's. Every file of every
// language folder is held to the folder's rules, so a mod refused on one
// install is refused on each: a file outside a language folder, a language
// folder no language names, a file with no archive folder, an extension the
// game takes no file of, a file larger than MaxInstallFileBytes, a text that
// is not UTF-8, two files one install path names, and anything that is not a
// regular file. A mod with no files/ folder carries none.
func ReadInstallFiles(id, dir, lang string) ([]InstallFile, error) {
	t, err := OpenTree(dir)
	if err != nil {
		return nil, Source{Mod: id, File: FilesDir}.Errorf("%v", err)
	}
	st, err := t.Lstat(FilesDir)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, Source{Mod: id, File: FilesDir}.Errorf("%v", err)
	case !st.IsDir():
		return nil, Source{Mod: id, File: FilesDir}.Errorf("is not a folder; install files stand in %s/%s/ or %s/<language>/", FilesDir, CommonFiles, FilesDir)
	}
	return readInstallFiles(id, t.FS(), InstallLanguages(), lang)
}

// InstallLanguages is the language folders of files/: common, then each locale.
func InstallLanguages() []string {
	langs := []string{CommonFiles}
	for _, l := range locale.All() {
		langs = append(langs, l.Code)
	}
	return langs
}

// installFileAt is the file at rel in files/, held to the rules of one
// file; the folder walk and the editor's drafts both ask it.
func installFileAt(id, rel string, size int64, read func() ([]byte, error)) (InstallFile, error) {
	bad := func(format string, args ...any) (InstallFile, error) {
		return InstallFile{}, Source{Mod: id, File: rel}.Errorf(format, args...)
	}
	parts := strings.Split(strings.TrimPrefix(rel, FilesDir+"/"), "/")
	switch {
	case len(parts) == 1:
		return bad("stands outside a language folder; install files stand in %s/%s/ or %s/<language>/", FilesDir, CommonFiles, FilesDir)
	case len(parts) == 2:
		return bad("names no archive: a file stands at its install path, the archive's name and the path inside it")
	}
	kind, ok := InstallFileKind(rel)
	if !ok {
		return bad("the game takes no install file of this kind; a mod carries .bmp pictures, .wav sounds and .txt texts")
	}
	if size > MaxInstallFileBytes {
		return bad("is larger than %d bytes", MaxInstallFileBytes)
	}
	b, err := read()
	if err != nil {
		return bad("%v", err)
	}
	if kind == FileText && !utf8.Valid(b) {
		return bad("is not UTF-8 text")
	}
	return InstallFile{Mod: id, File: rel, Lang: namefold.Host(parts[0]), Address: strings.Join(parts[1:], "/"), Kind: kind, Bytes: b}, nil
}

// DraftInstallFile is the editor's draft b of the file at rel in files/ of
// mod id, held to the rules of one file and of the language folders.
func DraftInstallFile(id, rel string, b []byte) (InstallFile, error) {
	f, err := installFileAt(id, rel, int64(len(b)), func() ([]byte, error) { return b, nil })
	if err != nil {
		return InstallFile{}, err
	}
	if langs := InstallLanguages(); !slices.Contains(langs, f.Lang) {
		return InstallFile{}, f.Errorf("%q is not a language folder: the folders are %s", f.Lang, strings.Join(langs, ", "))
	}
	return f, nil
}

// OverlayInstallFiles is the install files of mod id on language lang with
// the editor's drafts of files/ put over them, held to the folder's rules;
// a nil draft takes its file out.
func OverlayInstallFiles(id, lang string, files []InstallFile, drafts map[string][]byte) ([]InstallFile, error) {
	out := slices.Clone(files)
	for _, rel := range slices.Sorted(maps.Keys(drafts)) {
		if !strings.HasPrefix(rel, FilesDir+"/") {
			continue
		}
		out = slices.DeleteFunc(out, func(f InstallFile) bool { return namefold.ModTree(f.File) == namefold.ModTree(rel) })
		b := drafts[rel]
		if b == nil {
			continue
		}
		f, err := DraftInstallFile(id, rel, b)
		if err != nil {
			return nil, err
		}
		if f.Lang != CommonFiles && f.Lang != lang {
			continue
		}
		at := slices.IndexFunc(out, func(o InstallFile) bool { return FoldAddress(o.Address) == FoldAddress(f.Address) })
		switch {
		case at < 0:
			out = append(out, f)
		case out[at].Lang == f.Lang:
			return nil, f.Errorf("names the install path of %s; one language folder holds one file per install path", out[at].File)
		case f.Lang != CommonFiles:
			out[at] = f
		}
	}
	sort.Slice(out, func(i, j int) bool { return FoldAddress(out[i].Address) < FoldAddress(out[j].Address) })
	return out, nil
}

// readInstallFiles is ReadInstallFiles over the mod folder fsys, whose files/
// folder exists.
func readInstallFiles(id string, fsys fs.FS, langs []string, lang string) ([]InstallFile, error) {
	byLang := map[string][]InstallFile{}
	spelled := map[string]string{}
	err := fs.WalkDir(fsys, FilesDir, func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if rel == FilesDir {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		bad := func(format string, args ...any) error {
			return Source{Mod: id, File: rel}.Errorf(format, args...)
		}
		parts := strings.Split(strings.TrimPrefix(rel, FilesDir+"/"), "/")
		if d.Type()&fs.ModeSymlink != 0 || !d.IsDir() && !d.Type().IsRegular() {
			return bad("is not a regular file or folder")
		}
		if len(parts) == 1 && d.IsDir() {
			low := namefold.Host(parts[0])
			if !slices.Contains(langs, low) {
				return bad("%q is not a language folder: the folders are %s", parts[0], strings.Join(langs, ", "))
			}
			if first, dup := spelled[low]; dup {
				return bad("is the language folder %q, which %s/%s also spells; a mod has one folder per language", low, FilesDir, first)
			}
			spelled[low] = parts[0]
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return bad("%v", err)
		}
		f, err := installFileAt(id, rel, info.Size(), func() ([]byte, error) { return fs.ReadFile(fsys, rel) })
		if err != nil {
			return err
		}
		for _, other := range byLang[f.Lang] {
			if FoldAddress(other.Address) == FoldAddress(f.Address) {
				return bad("names the install path of %s; one language folder holds one file per install path", other.File)
			}
		}
		byLang[f.Lang] = append(byLang[f.Lang], f)
		return nil
	})
	if err != nil {
		return nil, err
	}
	var out []InstallFile
	own := map[string]bool{}
	if lang != CommonFiles {
		for _, f := range byLang[lang] {
			own[FoldAddress(f.Address)] = true
			out = append(out, f)
		}
	}
	for _, f := range byLang[CommonFiles] {
		if !own[FoldAddress(f.Address)] {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return FoldAddress(out[i].Address) < FoldAddress(out[j].Address) })
	return out, nil
}
