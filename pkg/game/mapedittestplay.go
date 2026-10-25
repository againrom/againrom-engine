package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/launch"
	"againrom/pkg/locale"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// TestPlayModID is the id of the map editor's test mod. Its mods folder is
// TestPlayModID in the editor's player profile, and the mod is the folder of
// the same name inside it.
const TestPlayModID = "testplay"

// testPlayMap and testPlayTexts are where the test mod holds the map and the
// mission's text pages.
const (
	testPlayMap   = "mission/map.alm"
	testPlayTexts = "mission/text"
)

// TestPlayHost is what the map editor's test play takes from its program: the
// player profile the test mod is written under, the game program and the
// launch that starts it.
type TestPlayHost struct {
	// Profile resolves the editor's player profile when a test play writes.
	Profile func() (RuntimeProfile, error)
	// Game is the game program a test play starts.
	Game string
	// Start launches program with args and returns without waiting.
	Start func(program string, args []string) error
}

// SetTestPlay names the host of the editor's test play.
func (x *MapInspector) SetTestPlay(h TestPlayHost) { x.testPlay = &h }

// testPlayEntry is the test mod's entry script.
const testPlayEntry = `def init(game, settings):
    game.data.add("` + mod.MissionsFile + `")
`

// TestPlay writes the map as it now stands into the editor's test mod and
// starts the game in that mission. Nothing is written and nothing starts when
// the report holds an error, the install is not the first game's, or the game
// program is missing. It answers the line the window shows.
func (s *mapInspectionEdits) TestPlay() (string, error) {
	x := s.inspector
	h := x.testPlay
	if h == nil {
		return "", errors.New("Test play: this editor was started without the game program")
	}
	if x.archives == nil {
		return "", errors.New("Test play: no install is open")
	}
	if err := s.TestPlayRefusal(); err != nil {
		return "", err
	}
	if err := s.gate(false); err != nil {
		return "", err
	}
	b, err := s.model.SaveBytes()
	if err != nil {
		return "", fmt.Errorf("Test play: %w", err)
	}
	if len(b) > mod.MaxFileBytes {
		return "", fmt.Errorf("Test play: the map is %d bytes; a mod map holds at most %d", len(b), mod.MaxFileBytes)
	}
	m, err := x.testPlayMission(s.source, b)
	if err != nil {
		return "", fmt.Errorf("Test play: %w", err)
	}
	if s.workspace != nil {
		m.TextFiles = cloneWorkspace(s.workspace).Pages
		if len(m.TextFiles) > 0 {
			m.Texts = testPlayTexts
		} else {
			m.Texts = ""
		}
	}
	// The game loads the test mod through the same rule table; an error there
	// would refuse the mod before the mission opens.
	address, _ := MissionMap(m.Number)
	if rows := x.CheckModMap(m, address); hasError(rows) {
		return "", s.loaderRefusal(rows)
	}
	if st, err := os.Stat(h.Game); err != nil || !st.Mode().IsRegular() {
		return "", fmt.Errorf("Test play: the game program %s is not beside the map editor", h.Game)
	}
	profile, err := h.Profile()
	if err != nil {
		return "", fmt.Errorf("Test play: no player profile: %w", err)
	}
	input := s.source
	if strings.HasPrefix(input, "scenario/") || isNewSource(input) {
		input = ""
	} else if strings.HasPrefix(input, "loose/") {
		input = looseHostPath(x.archives.Root, input)
	}
	var protected []string
	if input != "" {
		protected = append(protected, filepath.Dir(input))
	}
	if x.mod != nil {
		protected = append(protected, x.mod.dir)
	}
	modsDir, err := writeTestMod(profile, x.archives.Root, m, protected...)
	if err != nil {
		return "", fmt.Errorf("Test play: %w", err)
	}
	args, err := TestPlayArgs(x.archives.Root, modsDir, m.Number)
	if err != nil {
		return "", fmt.Errorf("Test play: %w", err)
	}
	if err := h.Start(h.Game, args); err != nil {
		return "", fmt.Errorf("Test play: %s did not start: %v", filepath.Base(h.Game), err)
	}
	return fmt.Sprintf("Test play: %s opens mission %d", filepath.Base(h.Game), m.Number), nil
}

func hasError(rows []ui.InspectionFinding) bool {
	errs, _ := ui.CountFindings(rows)
	return errs > 0
}

// TestPlayArgs is the game's command line of a test play: the install root
// and its base, the test mod from modsDir, and the direct start of mission n.
func TestPlayArgs(root, modsDir string, n int) ([]string, error) {
	return launch.Build(func(st launch.Setting) []string {
		switch st.Key {
		case launch.Assets.Key:
			return []string{root}
		case launch.Base.Key:
			if id := BaseID(InspectInstall(root)); base.Nameable(id) {
				return []string{id}
			}
			return nil
		case launch.Mods.Key:
			return []string{TestPlayModID}
		case launch.ModsDir.Key:
			return []string{modsDir}
		case launch.Mission.Key:
			return []string{strconv.Itoa(n)}
		}
		return []string{st.Load("")}
	})
}

// testPlayMission wraps the map and its shared page source in a guest mission.
func (x *MapInspector) testPlayMission(source string, b []byte) (mod.Mission, error) {
	c, err := LoadCampaign(x.archives.Containers)
	if err != nil {
		return mod.Mission{}, err
	}
	m, err := testPlayGuest(c, x.archives.Containers)
	if err != nil {
		return mod.Mission{}, err
	}
	m.MapBytes = b
	m.TextFiles = map[string][]byte{}
	if p := x.editorPages(source); p != nil {
		if p.err != nil {
			return mod.Mission{}, p.err
		}
		for name, t := range p.files {
			if missionPageName(name) {
				m.TextFiles[name] = t
			}
		}
	}
	if len(m.TextFiles) > 0 {
		m.Texts = testPlayTexts
	}
	return m, nil
}

// testPlayGuest is the guest declaration the test mod plays its map under:
// the first main mission of campaign c whose win opens a town that grants a
// companion, and the lowest number after it that checkGuest admits.
func testPlayGuest(c Campaign, src *vfs.FS) (mod.Mission, error) {
	for _, after := range c.Main {
		offer, ok := c.NextMission(after)
		if !ok || !offer.Town {
			continue
		}
		for _, guest := range c.Chapters[offer.Mission].AddHero {
			m := mod.Mission{Mod: TestPlayModID, File: mod.MissionsFile, Kind: mod.MissionGuest, Key: "map",
				Map: testPlayMap, After: after, Guest: guest, Chapter: offer.Mission, Carry: mod.CarryJoin}
			for n := after + 1; n < after+mainMissionStride; n++ {
				m.Number = n
				if checkGuest(c, src, mod.MissionData{Missions: []mod.Mission{m}}, m) == nil {
					return m, nil
				}
			}
		}
	}
	return mod.Mission{}, errors.New("the campaign has no won mission whose town grants a companion, so no guest mission can be declared")
}

// testPlayMissions is the test mod's data/missions.toml for m.
func testPlayMissions(m mod.Mission) []byte {
	var sb strings.Builder
	fmt.Fprintln(&sb, "[[mission]]")
	if m.Kind == mod.MissionReplace {
		fmt.Fprintf(&sb, "kind = %q\n", mod.MissionReplace)
	}
	fmt.Fprintf(&sb, "key     = %q\nnumber  = %d\nmap     = %q\n", m.Key, m.Number, m.Map)
	if m.Texts != "" {
		fmt.Fprintf(&sb, "texts   = %q\n", m.Texts)
	}
	if m.Kind != mod.MissionReplace {
		fmt.Fprintf(&sb, "after   = %d\nguest   = %d\nchapter = %d\n", m.After, m.Guest, m.Chapter)
		if m.Gold != 0 {
			fmt.Fprintf(&sb, "gold = %d\n", m.Gold)
		}
	}
	return []byte(sb.String())
}

// writeTestMod validates the resulting mod before publishing it. Its record
// admits replacement only of its own files and removal only of its old pages.
func writeTestMod(profile RuntimeProfile, root string, m mod.Mission, protected ...string) (string, error) {
	if profile.Directory == "" {
		return "", errors.New("no player profile directory")
	}
	dir, err := filepath.Abs(profile.Directory)
	if err != nil {
		return "", err
	}
	profile.Directory = dir
	modsDir := filepath.Join(profile.Directory, TestPlayModID)
	folder := filepath.Join(modsDir, TestPlayModID)
	old, err := testPlayOwnership(folder, protected)
	if err != nil {
		return "", err
	}
	var grant *runtimeProfileAccess
	if access := profile.Saves.profile; access != nil {
		grant = &runtimeProfileAccess{root: access.root, directory: access.root, descendants: true}
	}
	guard, forget := admittedOnce(func(dir string) error {
		if grant != nil {
			allowed, err := grant.allows(dir)
			if err != nil {
				return err
			}
			if allowed {
				return nil
			}
		}
		return fenceEditorWrite(root, dir, "Test play")
	})
	lang := "en"
	if l, ok := locale.ByBaseSuffix(BaseID(InspectInstall(root))); ok {
		lang = l.Code
	}
	files := missionPackageFiles("Map editor test play", "The map the map editor plays: written by its test play key, replaced by the next one.", m, lang)
	if err := testPlayTargets(folder, old, files); err != nil {
		return "", err
	}
	if err := guard(profile.Directory); err != nil {
		return "", err
	}
	stage, err := stageMissionPackage(profile.Directory, folder, root, TestPlayModID, testPlayRecord, old, files, guard)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	modsDir, grant, err = profile.folder(TestPlayModID, root)
	if err != nil {
		return "", err
	}
	forget()
	folder = filepath.Join(modsDir, TestPlayModID)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, rel := range names {
		if err := writeTestModFile(guard, modsDir, folder, rel, files[rel], false); err != nil {
			return "", err
		}
	}
	for _, rel := range old {
		if _, kept := files[rel]; kept || !strings.HasPrefix(rel, testPlayTexts+"/") {
			continue
		}
		full := filepath.Join(folder, filepath.FromSlash(rel))
		if err := guard(filepath.Dir(full)); err != nil {
			return "", err
		}
		if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	if err := writeTestModFile(guard, modsDir, folder, testPlayRecord, testPlayRecordBytes(names), true); err != nil {
		return "", err
	}
	return modsDir, nil
}

// admittedOnce runs check once per folder spelling until forget: a folder it
// admitted is admitted again without a new install census, and a refusal is
// never remembered. writeTestModFile still checks each folder on a file's path
// for a link or a redirect before it asks the guard.
func admittedOnce(check func(string) error) (guard func(string) error, forget func()) {
	passed := map[string]bool{}
	guard = func(dir string) error {
		if passed[dir] {
			return nil
		}
		if err := check(dir); err != nil {
			return err
		}
		passed[dir] = true
		return nil
	}
	return guard, func() { clear(passed) }
}

// writeTestModFile writes data to the slash path rel in folder, creating the
// directories below top one by one, each checked by guard, and replacing the
// file through a dot-named file moved over it. Only a durable file is flushed
// before the move: the ownership record, written last. Every other file is
// rewritten by the next test play or lives in a staging folder removed after
// validation.
func writeTestModFile(guard func(string) error, top, folder, rel string, data []byte, durable bool) error {
	full := filepath.Join(folder, filepath.FromSlash(rel))
	lexical, err := filepath.Rel(top, filepath.Dir(full))
	if err != nil || !filepath.IsLocal(lexical) && lexical != "." {
		return fmt.Errorf("%s leads outside the test mod", rel)
	}
	dir := top
	if err := guard(dir); err != nil {
		return err
	}
	if lexical != "." {
		for _, part := range strings.Split(lexical, string(filepath.Separator)) {
			dir = filepath.Join(dir, part)
			st, err := os.Lstat(dir)
			if errors.Is(err, os.ErrNotExist) {
				err = os.Mkdir(dir, 0o755)
				if err == nil {
					st, err = os.Lstat(dir)
				}
			}
			if err != nil {
				return err
			}
			if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%s is not a folder of the test mod", dir)
			}
			physical, err := editorPhysicalDirectory(dir)
			if err != nil || !sameSaveName(physical, dir) {
				return fmt.Errorf("%s redirects out of the test mod", dir)
			}
			if err := guard(dir); err != nil {
				return err
			}
		}
	}
	target := filepath.Join(dir, filepath.Base(full))
	if st, err := os.Lstat(target); err == nil && !st.Mode().IsRegular() {
		return fmt.Errorf("%s is not a file", target)
	}
	tmp, err := os.CreateTemp(dir, ".testplay-*")
	if err != nil {
		return err
	}
	_, writeErr := tmp.Write(data)
	if writeErr == nil && durable {
		writeErr = tmp.Sync()
	}
	closeErr := tmp.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr == nil {
		writeErr = os.Rename(tmp.Name(), target)
	}
	if writeErr != nil {
		// This exact file was created by this call, never an input.
		_ = os.Remove(tmp.Name())
		return writeErr
	}
	return nil
}

// TestPlayRefusal is the campaign's refusal of every test play, nil when the
// install's campaign allows it or no install is open.
func (s *mapInspectionEdits) TestPlayRefusal() error {
	if s.inspector.archives == nil {
		return nil
	}
	return campaignOf(s.inspector.archives.Game()).testPlayRefusal()
}
