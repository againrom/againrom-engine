package launch

import (
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/ini"
)

// Section and key names of the starter's settings file that name the game
// roots; the starter writes them and every program that wants the selected
// root reads them through ReadBases and SelectedBase.
const (
	StarterSection = "starter"
	BasesSection   = "bases"
	LastBaseKey    = "last-base"
)

// Section and key of the starter's settings file that name the mods folder.
const (
	ModsSection = "mods"
	ModsDirKey  = "dir"
)

// ModsFolder is the one mods folder of the programs in the folder programDir,
// read by the starter's scan and the game and written by the editor's Publish:
// the folder named, else the mods folder beside the programs.
func ModsFolder(named, programDir string) string {
	if d := strings.TrimSpace(named); d != "" {
		return d
	}
	return filepath.Join(programDir, "mods")
}

// StarterModsDir is the mods folder the starter.ini in the folder dir names,
// blank when it names none or cannot be read.
func StarterModsDir(dir string) string {
	data, err := os.ReadFile(StarterINI(dir))
	if err != nil {
		return ""
	}
	f, _ := ini.Parse(data)
	named, _ := f.Get(ModsSection, ModsDirKey)
	return named
}

// ProgramModsFolder is ModsFolder for a program the starter did not start
// with a folder: the folder its own flag names, else the one the starter.ini
// beside it names.
func ProgramModsFolder(flag, programDir string) string {
	if strings.TrimSpace(flag) == "" {
		flag = StarterModsDir(programDir)
	}
	return ModsFolder(flag, programDir)
}

// NamedBase is one named game root.
type NamedBase struct {
	Key  string
	Path string
}

// StarterINI is the starter's settings file in the folder dir.
func StarterINI(dir string) string {
	return filepath.Join(dir, "starter.ini")
}

// ReadBases is the game roots the settings file f lists, in file order, and the
// key of the selected one.
func ReadBases(f *ini.File) (bases []NamedBase, last string) {
	last, _ = f.Get(StarterSection, LastBaseKey)
	for _, k := range f.Keys(BasesSection) {
		if p, _ := f.Get(BasesSection, k); p != "" {
			bases = append(bases, NamedBase{k, p})
		}
	}
	return bases, last
}

// SelectedKey is the key of the selected base: last when a base carries that
// key, else the first base's key, else last unchanged when there is no base.
func SelectedKey(bases []NamedBase, last string) string {
	for _, b := range bases {
		if strings.EqualFold(b.Key, last) {
			return last
		}
	}
	if len(bases) > 0 {
		return bases[0].Key
	}
	return last
}

// SelectedBase is the root of the selected base, and false when there is none.
func SelectedBase(bases []NamedBase, last string) (string, bool) {
	key := SelectedKey(bases, last)
	for _, b := range bases {
		if strings.EqualFold(b.Key, key) {
			return b.Path, true
		}
	}
	return "", false
}

// StarterBase is the selected base of the starter.ini in the folder dir. A
// missing or unreadable file, and a file with no base, answer false.
func StarterBase(dir string) (string, bool) {
	data, err := os.ReadFile(StarterINI(dir))
	if err != nil {
		return "", false
	}
	f, _ := ini.Parse(data)
	bases, last := ReadBases(f)
	return SelectedBase(bases, last)
}
