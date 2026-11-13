package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/internal/witness"
	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/pathdialog"
	"againrom/pkg/ui"
)

// speakerPNG is a synthetic picture of the pane's size in one colour.
func speakerPNG(t *testing.T, path string, ink color.RGBA) []byte {
	t.Helper()
	size := speakerPictureSize()
	pic := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	for i := 0; i < len(pic.Pix); i += 4 {
		pic.Pix[i], pic.Pix[i+1], pic.Pix[i+2], pic.Pix[i+3] = ink.R, ink.G, ink.B, ink.A
	}
	b := witness.Bytes(t, pic)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return b
}

// modSpeakerEditor is the speaker pick map opened as the map of a mod whose
// mission names it.
func modSpeakerEditor(t *testing.T) (*ui.MapEditor, *mapInspectionEdits, string) {
	t.Helper()
	return modSpeakerEditorIn(t, t.TempDir(), "ambush")
}

// modSpeakerEditorIn is modSpeakerEditor for a mod of the id given, in the
// mods folder parent.
func modSpeakerEditorIn(t *testing.T, parent, id string) (*ui.MapEditor, *mapInspectionEdits, string) {
	t.Helper()
	e, s := speakerPickEditor(t)
	// The package checks need each unit in the map's one player slot.
	for i := range 4 {
		if err := s.model.SetUnitField(i, "owner", 1); err != nil {
			t.Fatal(err)
		}
	}
	dir := writeNamedEditorMod(t, parent, id, treeModMissions)
	if err := s.inspector.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	entry := s.inspector.mod.missions.Missions[0]
	path := filepath.Join(dir, filepath.FromSlash(entry.Map))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, s.model.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	// The first read found no map at the mission's path, and with it no
	// records; the map is there now.
	if err := s.inspector.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	s.source = path
	s.inspector.archives.Containers.Overlay("scenario/20.alm", func() ([]byte, error) { return s.model.Bytes(), nil })
	digest, err := mod.ContentDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.inspector.mod.digest = digest
	return e, s, dir
}

const (
	speakerLady       = 2 // the placement no speaker record shows
	speakerLadyNumber = 3 // the script number of that placement
)

func TestModSpeakerWriteReadAndResolve(t *testing.T) {
	_, s, dir := modSpeakerEditor(t)
	ink := color.RGBA{R: 10, G: 200, B: 30, A: 255}
	pngFile := filepath.Join(t.TempDir(), "face.png")
	speakerPNG(t, pngFile, ink)
	_, before := s.UnitSpeaker(speakerLady)
	if before.Cause == ui.SpeakerShown {
		t.Fatalf("before the record: %+v", before)
	}
	if !s.CanAddSpeaker(speakerLady, before) {
		t.Fatalf("a map in a mod must offer the record: %+v", before)
	}
	id, err := s.AddSpeaker(speakerLady, "Lady", pngFile)
	if err != nil {
		t.Fatal(err)
	}
	if id != int64(speakerNumberStart("ambush")) {
		t.Fatalf("record number %d", id)
	}
	if got, miss := s.UnitSpeaker(speakerLady); got != id || miss.Cause != ui.SpeakerShown {
		t.Fatalf("the bound unit answers %d %+v", got, miss)
	}
	if got, _ := s.UnitSpeaker(3); got == id {
		t.Fatal("the record shows a unit it is not bound to")
	}
	face := missionStartFace(t, s, int(id))
	if c := face.RGBAAt(face.Bounds().Dx()/2, 10); c != ink {
		t.Fatalf("the message window draws %v", c)
	}
	portrait := s.npcPortraits([]ui.InspectionChoice{{Value: id}})[id]
	if portrait == nil || !bytes.Equal(portrait.Pix, face.Pix) {
		t.Fatal("the info window's face is not the message window's face")
	}

	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	// A cold read: another inspector opens the saved mod.
	_, cold := speakerPickEditor(t)
	if err := cold.inspector.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	if issue := cold.inspector.mod.contextIssue; issue != "" {
		t.Fatal(issue)
	}
	e, ok := cold.inspector.table.NPCCatalog.Entry(int32(id))
	if !ok || e.Mod == "" || !e.Bound || e.Unit != speakerLadyNumber {
		t.Fatalf("cold entry %+v", e)
	}
	if label, _ := cold.inspector.speakerLabel(id); label != "Lady" {
		t.Fatalf("name %q", label)
	}
}

// modSpeakerProject is modSpeakerEditor with the workspace made as an editor
// opening the mod's map makes it.
func modSpeakerProject(t *testing.T) (*mapInspectionEdits, string) {
	t.Helper()
	_, s, dir := modSpeakerEditor(t)
	s.workspace = nil
	if err := s.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
	if s.workspace.Project == nil {
		t.Fatal("the mod's workspace holds no project")
	}
	return s, dir
}

// A record added in an owned package stays through the next save into the
// package's own folder: the save does not refuse the record's files as
// unowned, and the folder holds the record and the picture afterwards.
func TestModSpeakerSurvivesSavingTheOwnedPackage(t *testing.T) {
	s, _ := modSpeakerProject(t)
	owned := filepath.Join(t.TempDir(), "ambush")
	if _, err := s.saveWorkspacePackage(owned); err != nil {
		t.Fatal(err)
	}
	pngFile := filepath.Join(t.TempDir(), "face.png")
	face := speakerPNG(t, pngFile, color.RGBA{R: 10, G: 200, B: 30, A: 255})
	id, err := s.AddSpeaker(speakerLady, "Lady", pngFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatalf("the save after a new speaker record: %v", err)
	}
	files := tree(t, owned)
	if !strings.Contains(files[mod.SpeakersFile], "Lady") || files[speakerPictureFile(int32(id))] != string(face) {
		t.Fatalf("the saved package holds %q and %d picture bytes", files[mod.SpeakersFile], len(files[speakerPictureFile(int32(id))]))
	}
	// A package built into another folder carries them too.
	other := filepath.Join(t.TempDir(), "ambush")
	if _, err := s.saveWorkspacePackage(other); err != nil {
		t.Fatal(err)
	}
	files = tree(t, other)
	if !strings.Contains(files[mod.SpeakersFile], "Lady") || files[speakerPictureFile(int32(id))] != string(face) {
		t.Fatalf("the copied package holds %q and %d picture bytes", files[mod.SpeakersFile], len(files[speakerPictureFile(int32(id))]))
	}
}

// The whole chain over the real host: the pick, a click on the unit no record
// shows, the name, the picture's window; the record is on disk, the speaker
// field names it, and the applied page holds its number.
func TestModSpeakerClickChoosesThePictureAndWritesTheRecord(t *testing.T) {
	e, s, dir := modSpeakerEditor(t)
	pngFile := filepath.Join(t.TempDir(), "face.png")
	speakerPNG(t, pngFile, color.RGBA{R: 10, G: 200, B: 30, A: 255})
	called := installInspectionDialog(e, t.TempDir(), pathdialog.Result{Path: pngFile})
	openPageOne(t, e)
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: 1, K: 0, Flag: false})
	if !ok {
		t.Fatalf("no button on the speaker row:\n%s", trigLines(e))
	}
	messageClick(t, e, r, ok)
	hover := e.CellPoint(pickLadyCell)
	e.Pointer(ui.Input{CursorX: hover.X, CursorY: hover.Y}, time.Time{})
	p := e.CellPoint(pickLadyCell)
	messageClick(t, e, image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))}, true)
	e.TextInput("Lady")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	request := finishInspectionDialog(t, e, called, pathdialog.OpenFile)
	if request.StartDir != dir || len(request.Filters) != 1 || request.Filters[0].Extensions[0] != "png" {
		t.Fatalf("request %+v", request)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(mod.SpeakersFile))); !os.IsNotExist(err) {
		t.Fatalf("draft wrote a file: %v", err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(mod.SpeakersFile))); err != nil || !strings.Contains(string(b), "Lady") {
		t.Fatalf("the record is not on disk: %v %q", err, b)
	}
	if !strings.Contains(strings.Join(e.PopupLines(), "\n"), "Lady") {
		t.Fatalf("the speaker field does not name the record:\n%s", strings.Join(e.PopupLines(), "\n"))
	}
	messageButton(t, e, 0)
	if page := string(s.workspace.Pages["event01.txt"]); !strings.Contains(page, fmt.Sprintf("npc=%d", speakerNumberStart("ambush"))) {
		t.Fatalf("the applied page is %q", page)
	}
}

// A picture that is no 72x96 PNG is refused and nothing is written.
func TestModSpeakerRefusesAPictureOfTheWrongSize(t *testing.T) {
	_, s, dir := modSpeakerEditor(t)
	wrong := filepath.Join(t.TempDir(), "wrong.png")
	if err := os.WriteFile(wrong, witness.Bytes(t, image.NewRGBA(image.Rect(0, 0, 64, 64))), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.AddSpeaker(speakerLady, "Lady", wrong)
	if err == nil || !strings.Contains(err.Error(), "72x96") {
		t.Fatalf("a 64x64 picture answered %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(mod.SpeakersFile))); !os.IsNotExist(err) {
		t.Fatalf("the refusal left a speakers file: %v", err)
	}
}

// remodel gives s the map with these units in this order, keeping the mod
// map's path, as a deletion or a move does.
func remodel(t *testing.T, s *mapInspectionEdits, units []synth.ALMUnit) {
	t.Helper()
	model, err := mapedit.New(speakerPickMapOf(units))
	if err != nil {
		t.Fatal(err)
	}
	s.model, s.speakers = model, nil
}

// A bound record stays with the unit's script number when the list moves
// under it: another unit taking the old place answers no record, the unit
// at its new place still does.
func TestModSpeakerBindingFollowsTheScriptNumber(t *testing.T) {
	_, s, _ := modSpeakerEditor(t)
	pngFile := filepath.Join(t.TempDir(), "face.png")
	speakerPNG(t, pngFile, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	id, err := s.AddSpeaker(speakerLady, "Lady", pngFile)
	if err != nil {
		t.Fatal(err)
	}
	// The paladin is deleted: the lady moves from place 2 to place 1, the
	// goblin from 3 to 2, and no unit keeps the old place's number.
	remodel(t, s, []synth.ALMUnit{
		pickUnit(pickFighterCell, 3, 0, 0, 2),
		pickUnit(pickLadyCell, 4, 0, 0, speakerLadyNumber),
		pickUnit(pickGoblinCell, pickGoblinClass, 2, 0, 4),
	})
	if got, miss := s.UnitSpeaker(1); got != id || miss.Cause != ui.SpeakerShown {
		t.Fatalf("the lady at her new place answers %d %+v", got, miss)
	}
	if got, _ := s.UnitSpeaker(2); got == id {
		t.Fatal("the goblin at the old place answers the record")
	}
}

// A unit with script number 0 and a unit sharing its number are refused, with
// a stated message, and nothing is written.
func TestModSpeakerRefusesAUnitWithNoNumberOfItsOwn(t *testing.T) {
	for _, c := range []struct {
		name string
		lady uint16
		want string
	}{
		{"number 0", 0, "no script number"},
		{"a shared number", 4, "more than one unit"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, s, dir := modSpeakerEditor(t)
			remodel(t, s, []synth.ALMUnit{
				pickUnit(pickPaladinCell, 1, 25, 1, 1),
				pickUnit(pickFighterCell, 3, 0, 0, 2),
				pickUnit(pickLadyCell, 4, 0, 0, c.lady),
				pickUnit(pickGoblinCell, pickGoblinClass, 2, 0, 4),
			})
			refusal := s.SpeakerBindRefusal(speakerLady)
			if refusal == nil || !strings.Contains(refusal.Error(), c.want) {
				t.Fatalf("SpeakerBindRefusal answered %v", refusal)
			}
			pngFile := filepath.Join(t.TempDir(), "face.png")
			speakerPNG(t, pngFile, color.RGBA{A: 255})
			if _, err := s.AddSpeaker(speakerLady, "Lady", pngFile); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("AddSpeaker answered %v", err)
			}
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(mod.SpeakersFile))); !os.IsNotExist(err) {
				t.Fatalf("the refusal left a speakers file: %v", err)
			}
		})
	}
}

// speakerMod writes a mod with one speaker record of a synthetic picture
// beside the given manifest and returns its parent folder.
func speakerMod(t *testing.T, id int, ink color.RGBA) string {
	t.Helper()
	parent := t.TempDir()
	dir := filepath.Join(parent, "ambush")
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "speakers"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"mod.toml":           editorModManifest,
		"main.star":          "def init(game, settings):\n    pass\n",
		"data/speakers.toml": fmt.Sprintf("[[speaker]]\nid = %d\nname = \"Karl\"\npicture = \"speakers/karl.png\"\n", id),
	} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	speakerPNG(t, filepath.Join(dir, "speakers", "karl.png"), ink)
	return parent
}

// A game that starts with such a mod loads it the mods' way, applies the
// declared kinds, and the record's face is in the table every reader of a
// speaker uses.
func TestModSpeakerReachesAGameStartedWithTheMod(t *testing.T) {
	ink := color.RGBA{R: 200, G: 10, B: 30, A: 255}
	parent := speakerMod(t, mod.FirstModSpeaker, ink)
	entries, err := mod.Resolve(parent, []string{"ambush"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := modrt.Load(entries, BaseID(InspectInstall(parent)), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{}
	f.Table = speakerPickTable()
	f.NPCFaces = speakerFaces(f.Table)
	if _, held := f.NPCFaces[mod.FirstModSpeaker]; held {
		t.Fatal("the table holds the record before the mod applies")
	}
	if _, err := f.setModSpeakers(result.Data.Speakers); err != nil {
		t.Fatal(err)
	}
	face, held := f.NPCFaces[mod.FirstModSpeaker]
	if !held || face.Kind != data.NPCModPicture || face.Picture == nil {
		t.Fatalf("the started game's face is %+v (held %v)", face, held)
	}
	if got := face.Picture.RGBAAt(face.Picture.Bounds().Dx()/2, 10); got != ink {
		t.Fatalf("the face draws %v", got)
	}
	if e, ok := f.Table.NPCCatalog.Entry(mod.FirstModSpeaker); !ok || e.Mod != "ambush" {
		t.Fatalf("the catalog's record is %+v (held %v)", e, ok)
	}
}

// useSpeakerTestPlay gives s a test play whose launch is recorded, and
// answers the profile folder the test mod is written under.
func useSpeakerTestPlay(t *testing.T, s *mapInspectionEdits) string {
	t.Helper()
	profile := t.TempDir()
	game := filepath.Join(t.TempDir(), "againrom.exe")
	if err := os.WriteFile(game, []byte("program"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := s.inspector.archives.Root
	s.inspector.SetTestPlay(TestPlayHost{
		Profile: func() (RuntimeProfile, error) {
			return ResolveRuntimeProfile(filepath.Join(profile, "saves"), "", "", root)
		},
		Game:  game,
		Start: func(string, []string) error { return nil },
	})
	return profile
}

// speakerFileUnder finds, by its slash suffix, a written file under dir.
func speakerFileUnder(t *testing.T, dir, suffix string) (string, bool) {
	t.Helper()
	for name, body := range tree(t, dir) {
		if strings.HasSuffix(name, "/"+suffix) {
			return body, true
		}
	}
	return "", false
}

// A test play of a map in such a mod writes the record and its picture into
// the test mod on both routes: the project's folder copy and the single
// mission's carried files.
func TestModSpeakerSurvivesATestPlay(t *testing.T) {
	for _, c := range []struct {
		name    string
		project bool
	}{{"the project route", true}, {"the single mission route", false}} {
		t.Run(c.name, func(t *testing.T) {
			var s *mapInspectionEdits
			if c.project {
				s, _ = modSpeakerProject(t)
			} else {
				_, s, _ = modSpeakerEditor(t)
			}
			profile := useSpeakerTestPlay(t, s)
			pngFile := filepath.Join(t.TempDir(), "face.png")
			face := speakerPNG(t, pngFile, color.RGBA{R: 7, G: 8, B: 9, A: 255})
			id, err := s.AddSpeaker(speakerLady, "Lady", pngFile)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.TestPlay(); err != nil {
				t.Fatal(err)
			}
			if body, ok := speakerFileUnder(t, profile, mod.SpeakersFile); !ok || !strings.Contains(body, "Lady") {
				t.Fatalf("the test mod holds no record: %q %v\n%v", body, ok, tree(t, profile))
			}
			if body, ok := speakerFileUnder(t, profile, speakerPictureFile(int32(id))); !ok || body != string(face) {
				t.Fatalf("the test mod's picture: %d bytes, held %v", len(body), ok)
			}
		})
	}
}

// carryModSpeakers lays the record and its picture on the single mission, and
// a mod with no speakers file carries nothing.
func TestCarryModSpeakersPutsTheFilesOnTheMission(t *testing.T) {
	_, s, _ := modSpeakerEditor(t)
	var empty mod.Mission
	if err := s.carryModSpeakers(&empty); err != nil || empty.Carried != nil {
		t.Fatalf("no speakers file carried %v, %v", empty.Carried, err)
	}
	pngFile := filepath.Join(t.TempDir(), "face.png")
	face := speakerPNG(t, pngFile, color.RGBA{R: 7, G: 8, B: 9, A: 255})
	id, err := s.AddSpeaker(speakerLady, "Lady", pngFile)
	if err != nil {
		t.Fatal(err)
	}
	var m mod.Mission
	if err := s.carryModSpeakers(&m); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(m.Carried[mod.SpeakersFile]), "Lady") || !bytes.Equal(m.Carried[speakerPictureFile(int32(id))], face) {
		t.Fatalf("carried %d files: %q", len(m.Carried), m.Carried[mod.SpeakersFile])
	}
}

func TestModSpeakerOutsideAModKeepsTheRefusal(t *testing.T) {
	_, s := speakerPickEditor(t)
	_, miss := s.UnitSpeaker(speakerLady)
	if s.CanAddSpeaker(speakerLady, miss) {
		t.Fatal("a map outside a mod offered a record")
	}
	if _, err := s.AddSpeaker(speakerLady, "Lady", "x.png"); err == nil {
		t.Fatal("a map outside a mod wrote a record")
	}
}

func TestTwoModsNamingOneSpeaker(t *testing.T) {
	rec := func(modID string, name string) mod.Speaker {
		return mod.Speaker{Mod: modID, File: mod.SpeakersFile, ID: mod.FirstModSpeaker, Name: name,
			Picture: "a.png", PictureBytes: []byte{1}, Fields: []mod.PatchField{{Name: "name"}, {Name: "picture"}}}
	}
	d := mod.SpeakerData{Speakers: []mod.Speaker{rec("a", "A"), rec("b", "B")}}
	if _, _, err := mod.MergeSpeakers(d); err == nil {
		t.Fatal("two mods with no load order must be refused")
	}
	d.Order = mod.PrecedenceOf([]mod.Entry{
		{Manifest: mod.Manifest{ID: "a"}},
		{Manifest: mod.Manifest{ID: "b", LoadAfter: []string{"a"}}},
	})
	got, overrides, err := mod.MergeSpeakers(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "B" || len(overrides) == 0 {
		t.Fatalf("merged %+v overrides %v", got, overrides)
	}
}
