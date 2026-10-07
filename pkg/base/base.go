// Package base names the game installs the engine can run on and states what
// each one is missing.
//
// A base is a directory of the original game's archives. A Profile describes one
// known base: the files it ships, the digest of its main archive, and the limits
// the engine has on it. Detect reads the directory (read-only) and returns the
// profile it matches, or an error that says why no profile fits.
//
// The package imports nothing of this repository and reads no game byte beyond
// the size and SHA-256 digest of the main archive. The digests are constants: a
// hash is not an install byte.
package base

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Profile identifiers. Mods name a base by these ids in applies-to.
const (
	ROM1EN   = "rom1-en"
	ROM1RU   = "rom1-ru"
	ROM1Demo = "rom1-demo"

	ROM2EN = "rom2-en"
	ROM2RU = "rom2-ru"

	// ROM1 is the id of a root that holds the five archives and matches no
	// known build and no known language.
	ROM1 = "rom1"

	// ROM2 is that id for a root that carries the second game's files.
	ROM2 = "rom2"
)

// Game names the game a root belongs to. A root's game is read from the files it
// ships, and the empty value of Profile.Game means ROM1.
type Game string

const (
	GameROM1 Game = "rom1"
	GameROM2 Game = "rom2"
)

// rom2Marks are the files a ROM2 root ships and a ROM1 root does not.
var rom2Marks = []string{"allods2.exe", "scenario.dll"}

// MainArchive is the file whose size and digest identify a build.
const MainArchive = "main.res"

// DefaultFirstMission is the campaign mission a new game opens on a base whose
// profile names none.
const DefaultFirstMission = 10

// Build is one known copy of the main archive.
type Build struct {
	Size   int64
	SHA256 string
}

// Limits state what the engine cannot do on a base, as the profile's author
// observed it. The zero value is "no limit".
type Limits struct {
	// NoCharacterGeneration: the base ships no character generation art, so a
	// new game opens FirstMission with the engine's default party and shows no
	// generation screen.
	NoCharacterGeneration bool

	// FirstMission is the mission a new game opens; zero means
	// DefaultFirstMission.
	FirstMission int

	// OriginalSaveRefusal, when not empty, is why loading an original save of
	// this base is refused. It is shown after the engine's own refusal.
	OriginalSaveRefusal string

	// Notes are the other stated limits, one sentence each.
	Notes []string
}

// Profile describes one known base.
type Profile struct {
	ID    string
	Title string

	// Game is the game the profile belongs to; empty means ROM1.
	Game Game

	// Language is the language entry of the base's main archive: "english",
	// "russian", or empty when the profile does not state one.
	Language string

	// Files are the names, beside the five required archives, that a base of
	// this profile ships. Names compare case-insensitively.
	Files []string

	// Builds are the known main archives. A profile with no build is matched by
	// language only.
	Builds []Build

	Limits Limits
}

// GameOf is the profile's game.
func (p Profile) GameOf() Game {
	if p.Game == "" {
		return GameROM1
	}
	return p.Game
}

// Mission is the mission a new game opens on this profile.
func (p Profile) Mission() int {
	if p.Limits.FirstMission > 0 {
		return p.Limits.FirstMission
	}
	return DefaultFirstMission
}

// Known reports whether p names a profile; the zero Profile does not and states
// no limit.
func (p Profile) Known() bool { return p.ID != "" }

// RequiredArchives are the five archives every base holds, in the order the
// engine opens them.
func RequiredArchives() []string {
	return []string{"main.res", "graphics.res", "scenario.res", "world.res", "movies.res"}
}

// Profiles are the known bases, in the order Detect tries them. The release
// profiles come first only for their digests; the demo is told apart from them
// by its digest and its own extra file.
var Profiles = []Profile{
	{
		ID:       ROM1Demo,
		Title:    "Rage of Mages demo 1.01",
		Language: "english",
		Files:    []string{"video4.res"},
		Builds:   []Build{{Size: 4822928, SHA256: "3601698a93d694d1482ad9af5a6bffd832848e647b9c296a0f013604d955ad2d"}},
		Limits: Limits{
			NoCharacterGeneration: true,
			FirstMission:          41,
			OriginalSaveRefusal:   "the demo's own save carries campaign inn arrays of different lengths, which the engine's original-save reader refuses",
			Notes: []string{
				"ships missions 41, 51 and 91 only; the campaign registry declares the full set",
				"ships no character generation, document or training art",
			},
		},
	},
	{
		ID:       ROM1EN,
		Title:    "Rage of Mages, English",
		Language: "english",
		Builds:   []Build{{Size: 5938096, SHA256: "c67fd7585503c9239a160493dbe5622a1b5d93afd60a5fcb0b39b4997507861b"}},
	},
	{
		ID:       ROM1RU,
		Title:    "Rage of Mages, Russian",
		Language: "russian",
		Builds:   []Build{{Size: 4922540, SHA256: "be9199163c2331a74091415d60fd54b833f2be6275ecc35c9b0079aacaa95c40"}},
	},
	{
		ID:       ROM2EN,
		Title:    "Rage of Mages II, English",
		Game:     GameROM2,
		Language: "english",
		Builds:   []Build{{Size: 6619459, SHA256: "27b841ab71e6132eff7acc4fb75a6a2294260c0696b93ceec7f96773fae2d857"}},
		Limits:   rom2Limits,
	},
	{
		ID:       ROM2RU,
		Title:    "Rage of Mages II, Russian",
		Game:     GameROM2,
		Language: "russian",
		Builds:   []Build{{Size: 6416437, SHA256: "d51b031c9fbc09d2b15bca837c615619d7987fa09f6daa3c2ef248b86cb4c3e2"}},
		Limits:   rom2Limits,
	},
}

var rom2Limits = Limits{
	NoCharacterGeneration: true,
	FirstMission:          10,
	OriginalSaveRefusal:   "native ROM2 SAV import is unavailable; authored first-mission SAV requires its current continuation",
	Notes: []string{
		"supports the initial town TALK, mission 10 victory and selection of mission 20; later campaign continuation and native party construction are unavailable",
	},
}

// unrecognised is the profile of a root that holds the five archives and
// matches nothing known by language.
var unrecognised = Profile{ID: ROM1, Title: "Rage of Mages, unrecognised build"}

var unrecognisedROM2 = Profile{ID: ROM2, Title: "Rage of Mages II, unrecognised build", Game: GameROM2, Limits: rom2Limits}

// Match is the profile a root was detected as.
type Match struct {
	Profile Profile
	// Exact: the main archive's size and digest are a known build of the
	// profile. A profile matched by language alone is not exact.
	Exact bool
	// Digest is the main archive's SHA-256, empty when it was not read.
	Digest string
	// Main is the path of the main archive as the directory spells it.
	Main string
}

// ID is the matched profile's id, empty for the zero Match.
func (m Match) ID() string { return m.Profile.ID }

// String is the one-line statement -check and the starter print.
func (m Match) String() string {
	if !m.Profile.Known() {
		return "no base detected"
	}
	how := "exact build"
	if !m.Exact {
		how = "unrecognised build, matched by language"
		if m.Profile.ID == ROM1 || m.Profile.ID == ROM2 {
			how = "unrecognised build and language"
		}
	}
	return fmt.Sprintf("%s (%s; %s)", m.Profile.ID, m.Profile.Title, how)
}

// NotInstallError is a root that lacks required archives.
type NotInstallError struct {
	Root    string
	Missing []string
}

func (e *NotInstallError) Error() string {
	return fmt.Sprintf("%s is not a game base: missing %s", e.Root, strings.Join(e.Missing, ", "))
}

// PartialError is a root whose main archive is a known build but which lacks a
// file the profile ships.
type PartialError struct {
	Root    string
	Profile string
	Missing []string
}

func (e *PartialError) Error() string {
	return fmt.Sprintf("%s holds the main archive of %s but is missing %s", e.Root, e.Profile, strings.Join(e.Missing, ", "))
}

// MismatchError is a requested profile that differs from the detected one.
type MismatchError struct {
	Want string
	Got  Match
}

func (e *MismatchError) Error() string {
	return fmt.Sprintf("base %s was requested but the root is %s", e.Want, e.Got)
}

// Require reports an error unless m is the profile id want. An empty want
// accepts any base.
func Require(want string, m Match) error {
	if want == "" || m.Profile.ID == want {
		return nil
	}
	return &MismatchError{Want: want, Got: m}
}

// Find returns the known profile with id.
func Find(id string) (Profile, bool) {
	for _, p := range Profiles {
		if p.ID == id {
			return p, true
		}
	}
	if id == ROM1 {
		return unrecognised, true
	}
	if id == ROM2 {
		return unrecognisedROM2, true
	}
	return Profile{}, false
}

// IDs lists the profile ids a -base flag accepts.
func IDs() []string {
	ids := make([]string, 0, len(Profiles))
	for _, p := range Profiles {
		ids = append(ids, p.ID)
	}
	return ids
}

// Language reads the language entry of the main archive at path: "english",
// "russian", or "" when it cannot be read. It is supplied by the caller because
// reading an archive is not this package's business.
type Language func(mainArchive string) string

// Detect matches root against the known profiles. It lists the directory, stats
// the main archive and, only when its size equals a known build's, reads it once
// to take its SHA-256. It never writes.
//
// The language function is asked only when no digest matches; a nil one is the
// same as an unreadable archive.
func Detect(root string, language Language) (Match, error) {
	return DetectIn(root, Profiles, language)
}

// DetectIn is Detect over the given profiles.
func DetectIn(root string, profiles []Profile, language Language) (Match, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return Match{}, &NotInstallError{Root: root, Missing: RequiredArchives()}
	}
	actual := make(map[string]string, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			actual[strings.ToLower(e.Name())] = e.Name()
		}
	}
	var missing []string
	for _, name := range RequiredArchives() {
		if _, ok := actual[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Match{}, &NotInstallError{Root: root, Missing: missing}
	}
	game := GameROM2
	for _, name := range rom2Marks {
		if _, ok := actual[name]; !ok {
			game = GameROM1
		}
	}
	profiles = inGame(profiles, game)
	mainPath := filepath.Join(root, actual[MainArchive])
	st, err := os.Stat(mainPath)
	if err != nil {
		return Match{}, fmt.Errorf("%s: %w", mainPath, err)
	}
	var digest string
	for _, p := range profiles {
		for _, b := range p.Builds {
			if b.Size != st.Size() {
				continue
			}
			if digest == "" {
				if digest, err = fileDigest(mainPath); err != nil {
					return Match{}, fmt.Errorf("%s: %w", mainPath, err)
				}
			}
			if digest != b.SHA256 {
				continue
			}
			var absent []string
			for _, f := range p.Files {
				if _, ok := actual[strings.ToLower(f)]; !ok {
					absent = append(absent, f)
				}
			}
			if len(absent) > 0 {
				return Match{}, &PartialError{Root: root, Profile: p.ID, Missing: absent}
			}
			return Match{Profile: p, Exact: true, Digest: digest, Main: mainPath}, nil
		}
	}
	var lang string
	if language != nil {
		lang = language(mainPath)
	}
	for _, p := range profiles {
		if len(p.Files) == 0 && p.Language != "" && p.Language == lang {
			return Match{Profile: p, Digest: digest, Main: mainPath}, nil
		}
	}
	if game == GameROM2 {
		return Match{Profile: unrecognisedROM2, Digest: digest, Main: mainPath}, nil
	}
	return Match{Profile: unrecognised, Digest: digest, Main: mainPath}, nil
}

func inGame(profiles []Profile, game Game) []Profile {
	var out []Profile
	for _, p := range profiles {
		if p.GameOf() == game {
			out = append(out, p)
		}
	}
	return out
}

// fileDigest is the SHA-256 of the file at path, in lower-case hex.
func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
