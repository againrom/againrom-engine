package game

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"againrom/pkg/formats/a2c"
	"againrom/pkg/formats/alch"
	"againrom/pkg/namefold"
	"againrom/pkg/ui"
)

// characterFolder is the player profile's folder of character files.
const characterFolder = "characters"

// maxCharacterRead bounds the bytes a listing reads of one file; the
// decoders need only the header.
const maxCharacterRead = 1 << 16

// CharacterStore is the player profile's character folder. Each game's
// files are in the subfolder its row names; the install is never read
// (DIV-5194).
type CharacterStore struct {
	Dir string
	// profile admits deletion beneath a profile that sits inside an install.
	profile *runtimeProfileAccess
}

// characterStoreRow is one game's row of the character store.
type characterStoreRow struct {
	// dir is the game's subfolder of the character folder.
	dir string
	// ext is the file extension the list reads, in any letter case.
	ext string
	// name decodes a file's name, or refuses the file.
	name func([]byte) ([]byte, error)
	// order sorts the listed files.
	order func(a, b characterFile) int
	// newEntry is the engine word of the list's last row.
	newEntry string
}

// characterFile is one listed file.
type characterFile struct {
	File string
	Name []byte
	Time time.Time
}

// newestFirst orders files by last write time, newest first, then by file
// name. The original orders by CompareFileTime; its direction is Unknown
// (DIV-5195).
func newestFirst(a, b characterFile) int {
	if c := b.Time.Compare(a.Time); c != 0 {
		return c
	}
	return cmp.Compare(a.File, b.File)
}

// firstCharacterFiles reads the AlCh files (MENU-168, HERO-213).
var firstCharacterFiles = characterStoreRow{dir: "rom1", ext: ".chr", order: newestFirst, newEntry: "characters.new",
	name: func(b []byte) ([]byte, error) {
		h, err := alch.ReadHeader(b)
		return h.Name, err
	}}

// secondCharacterFiles reads the .a2c files (R2-SESSION-022, R2-ENGINE-1840).
var secondCharacterFiles = characterStoreRow{dir: "rom2", ext: ".a2c", order: newestFirst, newEntry: "characters.new",
	name: func(b []byte) ([]byte, error) {
		id, err := a2c.ReadIdentity(b)
		return id.Name, err
	}}

// path is the game's folder of the store.
func (s CharacterStore) path(row characterStoreRow) string {
	if s.Dir == "" {
		return ""
	}
	return filepath.Join(s.Dir, row.dir)
}

// listCharacters reads dir's regular files with the row's extension and
// answers those the row's decoder accepts, in the row's order, and the count
// it left out; the original's filter value is Unknown (DIV-5192). An absent
// folder lists nothing. It writes nothing.
func listCharacters(dir string, row characterStoreRow) ([]characterFile, int, error) {
	if dir == "" {
		return nil, 0, nil
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var out []characterFile
	left := 0
	for _, e := range entries {
		if !e.Type().IsRegular() || !namefold.HostEqual(filepath.Ext(e.Name()), row.ext) {
			continue
		}
		file, err := readCharacter(filepath.Join(dir, e.Name()), row)
		if err != nil {
			left++
			continue
		}
		out = append(out, file)
	}
	slices.SortFunc(out, row.order)
	return out, left, nil
}

func readCharacter(path string, row characterStoreRow) (characterFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return characterFile{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return characterFile{}, err
	}
	if !info.Mode().IsRegular() {
		return characterFile{}, fmt.Errorf("character file is not a regular file")
	}
	b, err := io.ReadAll(io.LimitReader(f, maxCharacterRead))
	if err != nil {
		return characterFile{}, err
	}
	name, err := row.name(b)
	if err != nil {
		return characterFile{}, err
	}
	return characterFile{File: filepath.Base(path), Name: name, Time: info.ModTime()}, nil
}

// characterDeletePath is the listed file name inside dir, or a refusal.
func characterDeletePath(dir, name string, row characterStoreRow, fences []string, profile *runtimeProfileAccess) (string, error) {
	if dir == "" || !namefold.HostEqual(filepath.Ext(name), row.ext) {
		return "", fmt.Errorf("invalid character file name")
	}
	return listedFilePath(dir, name, fences, profile.within(dir))
}

// ConfigureCharacters gives the character list page the game's character
// files from store: each row is a file's name, the message counts the files
// left out, and Delete removes the chosen file through the Load page's
// delete path.
func (f *FrontEnd) ConfigureCharacters(app *ui.App, store CharacterStore) {
	if f.Archives != nil {
		app.SetCharacterFiles(f.characterFiles(store))
	}
}

// characterFiles is the page's store over the game's row of store.
func (f *FrontEnd) characterFiles(store CharacterStore) ui.CharacterFiles {
	files := filesOf(f.Archives.Game())
	row := files.characters
	dir := store.path(row)
	selector := f.textSelector()
	left := f.Words.Engine.Text("characters.left")
	var fences []string
	if f.Archives.Root != "" {
		fences = append(fences, f.Archives.Root)
	}
	out := ui.CharacterFiles{
		List: func() ([]ui.SaveEntry, string) {
			files, skipped, err := listCharacters(dir, row)
			if err != nil {
				return nil, err.Error()
			}
			rows := make([]ui.SaveEntry, len(files))
			for i, file := range files {
				rows[i] = ui.SaveEntry{Name: file.File, Label: asciiLabel(file.Name, selector)}
			}
			if skipped == 0 {
				return rows, ""
			}
			return rows, fmt.Sprintf(left, skipped)
		},
	}
	if !files.characterDelete {
		return out
	}
	out.CanDelete = func(name string) bool {
		_, err := characterDeletePath(dir, name, row, fences, store.profile)
		return err == nil
	}
	out.PrepareDelete = func(name string) (func() error, error) {
		path, err := characterDeletePath(dir, name, row, fences, store.profile)
		if err != nil {
			return nil, err
		}
		return prepareFileDelete(path, false, fences, osNamedSaveFiles{}, store.profile.within(dir))
	}
	return out
}
