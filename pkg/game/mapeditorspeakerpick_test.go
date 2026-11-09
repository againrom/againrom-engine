package game

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The cells of the speaker pick map's four placements.
var (
	pickPaladinCell = image.Pt(16, 16) // the paladin, placed by his own NPC record 25
	pickFighterCell = image.Pt(22, 16) // a man fighter, placed by his Humans row's type
	pickLadyCell    = image.Pt(28, 16) // a woman no speaker record shows
	pickGoblinCell  = image.Pt(34, 16) // a goblin at tier 2
)

// pickGoblinClass is the goblin's class, at the units band.
const pickGoblinClass = 35

// speakerPickMap is the map the speaker pick clicks: the four placements of
// the cells above, and a message action naming page 1.
func speakerPickMap() []byte {
	return speakerPickMapOf([]synth.ALMUnit{
		pickUnit(pickPaladinCell, 1, 25, 1, 1),
		pickUnit(pickFighterCell, 3, 0, 0, 2),
		pickUnit(pickLadyCell, 4, 0, 0, 3),
		pickUnit(pickGoblinCell, pickGoblinClass, 2, 0, 4),
	})
}

// pickUnit is a placement at the middle of cell c.
func pickUnit(c image.Point, class int16, sub uint16, flags uint32, id uint16) synth.ALMUnit {
	return synth.ALMUnit{X: uint32(c.X)<<8 + 128, Y: uint32(c.Y)<<8 + 128, ClassID: class, ClassSubID: sub, Flags: flags, UnitID: id}
}

// speakerPickMapOf is a map of units with a message action naming page 1.
func speakerPickMapOf(units []synth.ALMUnit) []byte {
	script := scriptPayload([]scriptNode{{label: "Message", opcode: 2, id: 1, values: []uint32{1}, tags: []uint32{1}}}, nil)
	return synth.ALM(synth.ALMOptions{Width: 64, Height: 64, Type7Payload: script, Units: units})
}

// speakerPickFaces are the speaker records: 3 states no token and answers
// the first live person; 12 and 13 are the man fighter of face 1 who is no
// hero; 25 is the paladin; 30 is a woman of face 7 nobody on the map is;
// 40 to 42 are the goblin's class portrait at tiers 1, 2 and 2.
func speakerPickFaces() map[int32]data.NPCFace {
	fighter := npcTokenSet(data.NPCTokenHuman, data.NPCTokenFace, data.NPCTokenNotFemale, data.NPCTokenNotMage)
	goblin := npcTokenSet(data.NPCTokenNotHuman, data.NPCTokenPicture, data.NPCTokenFace)
	return map[int32]data.NPCFace{
		3:  {Kind: data.NPCFigure, Dir: data.FigureDirManMage, Face: 1},
		12: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 1, Tokens: fighter},
		13: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 1, Tokens: fighter},
		25: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: dressPaladinFace,
			Tokens: npcTokenSet(data.NPCTokenHero, data.NPCTokenFace, data.NPCTokenNotFemale, data.NPCTokenNotMage)},
		30: {Kind: data.NPCFigure, Dir: data.FigureDirWomanFighter, Face: 7,
			Tokens: npcTokenSet(data.NPCTokenHuman, data.NPCTokenFemale, data.NPCTokenFace)},
		40: {Kind: data.NPCPortrait, Class: pickGoblinClass, HasClass: true, Face: 1, Tokens: goblin},
		41: {Kind: data.NPCPortrait, Class: pickGoblinClass, HasClass: true, Face: 2, Tokens: goblin},
		42: {Kind: data.NPCPortrait, Class: pickGoblinClass, HasClass: true, Face: 2, Tokens: goblin},
	}
}

// speakerPickTable is dressTable with a woman's Humans row of type 4, the
// goblin's Units rows at tiers 1 and 2, and the speaker records.
func speakerPickTable() *mapload.Table {
	t := dressTable()
	lady := figureRow(4, 2, 1)
	lady[24] = 44
	t.Humans = append(slices.Clone(t.Humans.(dbCollection)), dbEntry{name: "M12_Lady", params: lady})
	goblin := func(tier int32) dbEntry {
		p := make([]int32, 41)
		for i := range p {
			p[i] = -1
		}
		p[29], p[30] = pickGoblinClass, tier
		return dbEntry{name: "Goblin", params: p}
	}
	t.Units = dbCollection{{}, goblin(1), goblin(2)}
	faces := speakerPickFaces()
	var entries []data.NPCEntry
	for id := range faces {
		entries = append(entries, data.NPCEntry{ID: id, Name: "speaker"})
	}
	slices.SortFunc(entries, func(a, b data.NPCEntry) int { return int(a.ID - b.ID) })
	t.NPCCatalog = &data.NPCCatalog{Defs: t.NPC, Faces: faces, Entries: entries}
	return t
}

// The pictures of the speaker pick map, each its own colour: the man
// fighter's and the man mage's face-1 figures, the woman fighter's faces 7
// and 2, and the goblin's tier 1 and tier 2 portraits.
var (
	pickFighterInk  = color.RGBA{R: 0xc0, G: 0x20, A: 0xff}
	pickMageInk     = color.RGBA{B: 0xc0, G: 0x20, A: 0xff}
	pickLadyInk     = color.RGBA{R: 0xc0, B: 0xc0, A: 0xff}
	pickLadyFaceInk = color.RGBA{R: 0x40, B: 0x80, A: 0xff}
	pickGoblinInk   = color.RGBA{G: 0xff, A: 0xff}
	pickGoblin2Ink  = color.RGBA{G: 0x80, R: 0x40, A: 0xff}
)

// pickFigureSheet is a figure base the size of a speaker picture, every
// pixel opaque in col, so the pane's cut of it is not empty.
func pickFigureSheet(col color.RGBA) []byte {
	pixels := make([]synth.Pixel256, 160*240)
	for i := range pixels {
		pixels[i] = synth.Pixel256{Index: 1, Opaque: true}
	}
	return synth.Sheet256(synth.Sheet256Options{Palette: []color.RGBA{{}, col},
		Frames: []synth.Frame256{{Width: 160, Height: 240, Pixels: pixels}}})
}

// speakerPickEditor opens the speaker pick map in the game's map editor at
// 1920x1080 over the fixture table and pictures, with page 1 a one-card
// message of no speaker.
func speakerPickEditor(t *testing.T) (*ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	return speakerPickEditorOn(t, speakerPickMap())
}

// speakerPickEditorOn is speakerPickEditor over the map m.
func speakerPickEditorOn(t *testing.T, m []byte) (*ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	archives := newTestPlayFixture(t).x.archives
	for path, pic := range map[string][]byte{
		data.ItemFigureBasePath(data.FigureDirManFighter, 1):   pickFigureSheet(pickFighterInk),
		data.ItemFigureBasePath(data.FigureDirManMage, 1):      pickFigureSheet(pickMageInk),
		data.ItemFigureBasePath(data.FigureDirWomanFighter, 7): pickFigureSheet(pickLadyInk),
		data.ItemFigureBasePath(data.FigureDirWomanFighter, 2): pickFigureSheet(pickLadyFaceInk),
	} {
		archives.Containers.Overlay(graphicsPrefix+path, func() ([]byte, error) { return pic, nil })
	}
	archives.Containers.Overlay("graphics/infowindow/Goblin.bmp", func() ([]byte, error) { return synthBMP(160, 240, pickGoblinInk), nil })
	archives.Containers.Overlay("graphics/infowindow/Goblin2.bmp", func() ([]byte, error) { return synthBMP(160, 240, pickGoblin2Ink), nil })
	units := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{pickGoblinClass: {Portrait: "Goblin"}}}
	x := &MapInspector{archives: archives, table: speakerPickTable(), units: units, tiles: &terrain.Tileset{}, rules: []mapRule{}}
	path := filepath.Join(t.TempDir(), "speakers.alm")
	if err := os.WriteFile(path, m, 0o600); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1920, 1080)
	e.Document().Viewer.SetFlat(true)
	closeMapPanel(t, e)
	workspacePage(t, e, "event01.txt", "<part=1>\nHello")
	return e, e.Document().Edits.(*mapInspectionEdits)
}

// missionStartFace is speaker id's face as the game's dialogue shows it when
// a mission opens on s's map: the world the map builds, the driver
// openMission makes of it, and that driver's SpeakerFace, cut as the pane
// cuts it.
func missionStartFace(t *testing.T, s *mapInspectionEdits, id int) *image.RGBA {
	t.Helper()
	face, window, ok := missionStart(t, s).SpeakerFace(id)
	if !ok {
		t.Fatalf("the mission resolves no face for speaker %d", id)
	}
	return ui.SpeakerPortrait(face, window)
}

// missionStart is the driver openMission makes of the world s's map builds.
func missionStart(t *testing.T, s *mapInspectionEdits) *mapWorld {
	t.Helper()
	model, err := mapedit.New(s.model.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	m, err := model.Map()
	if err != nil {
		t.Fatal(err)
	}
	built, _ := model.Map()
	w, err := mapload.FromALMWith(built, s.inspector.table, openDifficulty)
	if err != nil {
		t.Fatal(err)
	}
	return openMission(&Mission{Number: 1, Map: m, World: w}, s.inspector.table, s.inspector.units,
		worldFixtureViewer(t, m), s.inspector.archives.artSource(), nil, speakerFaces(s.inspector.table))
}

// A person takes his own NPC record when it resolves to him, though a lower
// record resolves to him too; a person placed by his row takes the lowest
// record that resolves to him; a creature takes the lowest record whose class
// portrait is the picture his info window shows at his tier; a person no
// record resolves to takes none.
func TestUnitSpeakerIsTheRecordThatShowsTheUnit(t *testing.T) {
	_, s := speakerPickEditor(t)
	for _, tc := range []struct {
		name string
		unit int
		want int64
		miss ui.SpeakerCause
	}{
		{name: "the paladin's own record", unit: 0, want: 25},
		{name: "the fighter's lowest record", unit: 1, want: 12},
		{name: "no record shows the woman", unit: 2, miss: ui.SpeakerNotFirst},
		{name: "the goblin's tier 2 portrait", unit: 3, want: 41},
		{name: "no placement", unit: 4, miss: ui.SpeakerNoMission},
	} {
		got, why := s.UnitSpeaker(tc.unit)
		if tc.miss != ui.SpeakerShown {
			if tc.miss == ui.SpeakerNotFirst && (why.Record != 3 || why.Cell != pickPaladinCell) {
				t.Errorf("%s: record %d first at %v, want record 3 at %v", tc.name, why.Record, why.Cell, pickPaladinCell)
			}
			if why.Cause != tc.miss {
				t.Errorf("%s: unit %d takes speaker %d with cause %d, want cause %d", tc.name, tc.unit, got, why.Cause, tc.miss)
			}
			continue
		}
		if why.Cause != ui.SpeakerShown || got != tc.want {
			t.Errorf("%s: unit %d takes speaker %d (cause %d), want %d", tc.name, tc.unit, got, why.Cause, tc.want)
		}
	}
	if rec, ok := s.speakers.cast.resolveNPC(3, speakerPickFaces()[3]); !ok || rec.id != 0 {
		t.Fatal("the fixture's record 3 does not resolve to the paladin, so the own record is not preferred over a lower one")
	}
	mw := missionStart(t, s)
	info := mw.unitPicture(3, pickGoblinClass)
	if info == nil || info.RGBAAt(80, 120) != pickGoblin2Ink {
		t.Fatal("the goblin's info window does not show his tier 2 picture")
	}
	for id, same := range map[int]bool{41: true, 40: false} {
		if face, _, ok := mw.SpeakerFace(id); !ok || bytes.Equal(face.Pix, info.Pix) != same {
			t.Errorf("speaker %d's face equals the goblin's info picture: %v, want %v", id, !same, same)
		}
	}
}

// The speaker list's portraits are the faces a dialogue shows at mission
// start on this map, pixel for pixel: a record a live person answers wears
// that person's figure, not its own. A change to the map's units changes the
// faces.
func TestSpeakerListPortraitsAreTheMissionStartFaces(t *testing.T) {
	_, s := speakerPickEditor(t)
	check := func(when string, ids ...int) map[int64]*image.RGBA {
		t.Helper()
		fields := s.MessageFields(1)
		if len(fields) != 2 {
			t.Fatalf("%s: page 1 has %d fields", when, len(fields))
		}
		icons := fields[1].Icons
		for _, id := range ids {
			got, want := icons[int64(id)], missionStartFace(t, s, id)
			if got == nil || got.Rect != want.Rect || !bytes.Equal(got.Pix, want.Pix) {
				t.Errorf("%s: speaker %d's portrait is not the face the mission shows", when, id)
			}
		}
		return icons
	}
	before := check("fresh", 3, 12, 25, 30, 41)
	own := gameSpeakerPortrait(t, s.inspector, speakerFaces(s.inspector.table), 3)
	if bytes.Equal(own.Pix, before[3].Pix) {
		t.Fatal("the fixture's record 3 looks the same live and bare, so the test cannot tell the two apart")
	}
	if pick, why := s.UnitSpeaker(1); why.Cause != ui.SpeakerShown || !bytes.Equal(before[pick].Pix, missionStartFace(t, s, 12).Pix) {
		t.Fatal("the fighter's picked speaker does not carry the fighter's face")
	}
	// The paladin and the fighter wear the same face-1 figure, so both go.
	for range 2 {
		if _, err := s.Delete("Unit", 0); err != nil {
			t.Fatal(err)
		}
	}
	after := check("after the paladin's and the fighter's removal", 3, 12, 25, 41)
	if bytes.Equal(before[3].Pix, after[3].Pix) {
		t.Error("record 3 kept the removed paladin's face")
	}
}

// openPageOne opens page 1 from the message action's popup.
func openPageOne(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	e.Select(recordIndex(t, e, "Action", 0), false)
	for i, line := range e.PopupLines() {
		if strings.HasPrefix(line, "Open Message") {
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupRow, N: i})
			messageClick(t, e, r, ok)
			return
		}
	}
	t.Fatalf("the action shows no Open Message link:\n%s", trigLines(e))
}

// A speaker picked on the map and then applied writes the same page as the
// same speaker chosen from the list.
func TestPickedSpeakerWritesTheListsMarkup(t *testing.T) {
	pages := map[string]string{}
	for _, how := range []string{"pick", "list"} {
		e, s := speakerPickEditor(t)
		openPageOne(t, e)
		if how == "pick" {
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: 1, K: 0, Flag: false})
			if !ok {
				t.Fatalf("no button on the speaker row:\n%s", trigLines(e))
			}
			messageClick(t, e, r, ok)
			// The pointer's first move over the map settles its camera, as a
			// player's pointer does on the way to the unit.
			hover := e.CellPoint(pickFighterCell)
			e.Pointer(ui.Input{CursorX: hover.X, CursorY: hover.Y}, time.Time{})
			p := e.CellPoint(pickFighterCell)
			messageClick(t, e, image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))}, true)
		} else {
			messageField(t, e, 1)
			messageChoiceClick(t, e, "12: ")
		}
		messageButton(t, e, 0)
		pages[how] = string(s.workspace.Pages["event01.txt"])
	}
	if pages["pick"] != pages["list"] || !strings.Contains(pages["pick"], "npc=12") {
		t.Fatalf("the pick wrote %q, the list %q", pages["pick"], pages["list"])
	}
}

// A person placed dead is no speaker: record 3, which names no token and so
// answers the first live person, shows its own picture in the list and the
// dead man is picked for no record. A dead man and a live man of the same
// look: the records resolve to the live one and a click on it sets the first.
func TestSpeakerCastIsTheGamesLiveCast(t *testing.T) {
	dead := func(c image.Point, id uint16) synth.ALMUnit {
		u := pickUnit(c, 3, 0, 0, id)
		u.HasCurrentHP = true
		return u
	}
	t.Run("a corpse alone", func(t *testing.T) {
		_, s := speakerPickEditorOn(t, speakerPickMapOf([]synth.ALMUnit{dead(pickFighterCell, 1)}))
		own := gameSpeakerPortrait(t, s.inspector, speakerFaces(s.inspector.table), 3)
		if got := s.MessageFields(1)[1].Icons[3]; got == nil || !bytes.Equal(got.Pix, own.Pix) {
			t.Error("record 3's list face is not its own picture")
		}
		if got, why := s.UnitSpeaker(0); why.Cause != ui.SpeakerDown {
			t.Errorf("the corpse is picked for record %d with cause %d, want down", got, why.Cause)
		}
	})
	t.Run("a corpse and a live man", func(t *testing.T) {
		_, s := speakerPickEditorOn(t, speakerPickMapOf([]synth.ALMUnit{dead(pickFighterCell, 1), pickUnit(pickLadyCell, 3, 0, 0, 2)}))
		if got, why := s.UnitSpeaker(1); why.Cause != ui.SpeakerShown || got != 3 {
			t.Errorf("the live man is picked for record %d (cause %d), want 3", got, why.Cause)
		}
		if got, why := s.UnitSpeaker(0); why.Cause != ui.SpeakerDown {
			t.Errorf("the corpse is picked for record %d with cause %d, want down", got, why.Cause)
		}
		a, ok := s.speakers.cast.resolveNPC(12, speakerPickFaces()[12])
		if !ok || a.id != 1 {
			t.Errorf("record 12 resolves to %v (%v), want the live man 1", a.id, ok)
		}
	})
	t.Run("a withdrawn border placement", func(t *testing.T) {
		_, s := speakerPickEditorOn(t, speakerPickMapOf([]synth.ALMUnit{
			pickUnit(image.Pt(2, 2), 3, 0, 0, 1), pickUnit(pickFighterCell, 3, 0, 0, 2)}))
		if got, why := s.UnitSpeaker(0); why.Cause != ui.SpeakerBorder {
			t.Errorf("the withdrawn placement is picked for record %d with cause %d, want border", got, why.Cause)
		}
		if got, why := s.UnitSpeaker(1); why.Cause != ui.SpeakerShown || got != 3 {
			t.Errorf("the placement after it is picked for record %d (cause %d), want 3", got, why.Cause)
		}
	})
}
