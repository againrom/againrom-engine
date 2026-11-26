package game

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"againrom/pkg/formats/a2c"
	"againrom/pkg/formats/alch"
	"againrom/pkg/hostfile"
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
	// fileName is the at-th name a new file may take, or false past the
	// last; record is the file of a generated character under that name.
	fileName func(at int) (string, bool)
	record   func(f *FrontEnd, res ui.ChargenResult, at int) ([]byte, error)
}

// characterStats are the generator's four statistics in the order both
// character files store them: Body, Reaction, Mind, Spirit.
var characterStats = [4]int{chargenStatBody, chargenStatReaction, chargenStatMind, chargenStatSpirit}

// secondCharacterMoney is section E's money in a new file (R2-ENGINE-1853).
const secondCharacterMoney = 1000

// lastCharacterName bounds the at a file name builder is asked for.
const lastCharacterName = 9999

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
	},
	// The name's prefix text is unread and the original numbers by the
	// selected row; this is the lowest free number (DIV-5242).
	fileName: func(at int) (string, bool) { return fmt.Sprintf("char%04d.chr", at), at <= lastCharacterName },
	record:   firstCharacterRecord}

// secondCharacterFiles reads the .a2c files (R2-SESSION-022, R2-ENGINE-1840).
var secondCharacterFiles = characterStoreRow{dir: "rom2", ext: ".a2c", order: newestFirst, newEntry: "characters.new",
	name: func(b []byte) ([]byte, error) {
		id, err := a2c.ReadIdentity(b)
		return id.Name, err
	},
	// The name is the identity pair's two numbers, as the original's; the
	// pair is (1, at) in place of a counter reading (DIV-5242).
	fileName: func(at int) (string, bool) { return fmt.Sprintf("1%d.a2c", at), at <= lastCharacterName },
	record:   secondCharacterRecord}

// firstCharacterRecord is the short AlCh file: flag bit 0, the name and the
// four statistics, every other span zero (HERO-213, DIV-5240).
func firstCharacterRecord(_ *FrontEnd, res ui.ChargenResult, _ int) ([]byte, error) {
	c := alch.Character{Name: []byte(res.Name), Flags: alch.FlagShort}
	for i, row := range characterStats {
		c.Stats[i] = uint32(chargenStatValue(res, row))
	}
	return alch.Write(c)
}

// secondCharacterRecord is the new .a2c file: A with the identity pair, the
// name and the start unit row of the chosen sex and class, B and C empty, E
// with the start money and the four statistics; D is left out (DIV-5241).
func secondCharacterRecord(f *FrontEnd, res ui.ChargenResult, at int) ([]byte, error) {
	unit := slices.Index(a2c.StartRows[:], f.generatorTemplateName(res))
	if unit < 0 {
		return nil, fmt.Errorf("no start unit row for the generated character")
	}
	c := a2c.Character{
		A:       a2c.A{ID0: 1, ID1: uint32(at), ListKey: a2c.ListKeyAny, Name: []byte(res.Name), Unit: byte(unit) << 6},
		Kills:   make([]byte, a2c.KillsSize),
		Hotkeys: make([]byte, a2c.HotkeysSize),
		E:       a2c.E{Money: secondCharacterMoney},
	}
	for i, row := range characterStats {
		c.E.Base[i] = uint8(chargenStatValue(res, row))
	}
	return a2c.Write(c)
}

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
	return listProfileFiles(profileListing[characterFile]{dir: dir, exts: []string{row.ext}, regular: true, order: row.order,
		read: func(dir string, e os.DirEntry) (characterFile, bool, error) {
			file, err := readCharacter(filepath.Join(dir, e.Name()), row)
			return file, err == nil, err
		}})
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

// characterFences are the install roots no character file is written in.
func (f *FrontEnd) characterFences() []string {
	if f.Archives.Root == "" {
		return nil
	}
	return []string{f.Archives.Root}
}

// storeCharacter writes res as a new file of the game's row in store.
func (f *FrontEnd) storeCharacter(store CharacterStore, res ui.ChargenResult) (string, error) {
	return f.writeCharacter(filesOf(f.Archives.Game()).characters, store, res)
}

// writeCharacter writes res as a new file of row in store, under the lowest
// name the row's file name builder gives that no entry holds, and answers
// that name. An existing file is never replaced: a file that takes the name
// before publication refuses the write, and no free name refuses it with the
// engine's words.
func (f *FrontEnd) writeCharacter(row characterStoreRow, store CharacterStore, res ui.ChargenResult) (string, error) {
	dir, fences := store.path(row), f.characterFences()
	if dir == "" || row.record == nil {
		return "", errors.New(f.Words.Engine.Text("characters.unstored"))
	}
	if _, err := namedSaveDirectory(dir, fences, store.profile); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	identity, err := resolveSaveDirectory(dir)
	if err != nil {
		return "", err
	}
	name, at := "", 0
	for ; ; at++ {
		next, ok := row.fileName(at)
		if !ok {
			return "", errors.New(f.Words.Engine.Text("characters.full"))
		}
		if _, err := os.Lstat(filepath.Join(identity.path, next)); errors.Is(err, os.ErrNotExist) {
			name = next
			break
		} else if err != nil {
			return "", err
		}
	}
	data, err := row.record(f, res, at)
	if err != nil {
		return "", err
	}
	_, err = hostfile.Commit(hostfile.Set{
		Files: []hostfile.File{{Path: filepath.Join(identity.path, name), Data: data, Overwrite: hostfile.Refuse}},
		Dir:   identity.path, Temp: ".againrom-character-*.tmp", Sync: true, Ops: osNamedSaveFiles{},
		Check: func(step hostfile.Step, _ int) error {
			if step != hostfile.StepPublish {
				return nil
			}
			current, err := resolveSaveDirectory(dir)
			if err != nil {
				return err
			}
			if !os.SameFile(identity.info, current.info) {
				return fmt.Errorf("character folder changed")
			}
			_, err = namedSaveDirectory(current.path, fences, store.profile)
			return err
		},
		Words: hostfile.Words{
			Create:       "create temporary character file: %w",
			Write:        "write temporary character file: %w",
			Sync:         "synchronize temporary character file: %w",
			Close:        "close temporary character file: %w",
			Publish:      "publish character file %s: %w",
			NoCommit:     "publish character file %s returned without committing a final path",
			Cleanup:      "remove failed temporary character file %s: %w",
			CloseCleanup: "close failed temporary character file during cleanup: %w",
		},
	})
	if err != nil {
		return "", err
	}
	return name, nil
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
	fences := f.characterFences()
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
	if !f.directNewGame() {
		// The new entry runs the campaign's generator pages and stores the
		// result (MENU-169, R2-ENGINE-1840).
		out.New = func() *ui.ChargenEntry {
			return &ui.ChargenEntry{
				Model: ui.NewChargen(f.chargenSetup(generatorStartCharacter)),
				Begin: f.generatorBegin(generatorStartCharacter, generatorTarget{store: store}),
			}
		}
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
