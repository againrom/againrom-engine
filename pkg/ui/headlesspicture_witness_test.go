package ui

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/frame"
)

// headlessPictureGolden holds the picture hash and the returned extras of
// every headless picture part in every state below, recorded before the
// accessors became one request. Every line is numbers.
const headlessPictureGolden = "testdata/headless_picture.golden"

// headlessPictureRecord writes the golden from this run when set to 1.
const headlessPictureRecord = "AGAINROM_HEADLESS_PICTURE_RECORD"

// headlessWitness is what one accessor call returned.
type headlessWitness struct {
	pix        *image.RGBA
	note       string
	statistics bool
	at         image.Point
	lines      [3]string
	cells      image.Rectangle
	tip        HeadlessTooltipState
	arts       int
	err        error
}

func headlessWitnessText(s string) uint64 {
	if s == "" {
		return 0
	}
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// row is the case's golden line: every field the accessor returned.
func (r headlessWitness) row(part, state int) string {
	var pixHash uint64
	bounds := image.Rectangle{}
	if r.pix != nil {
		h := fnv.New64a()
		h.Write(r.pix.Pix)
		pixHash = h.Sum64()
		bounds = r.pix.Bounds()
	}
	stats := 0
	if r.statistics {
		stats = 1
	}
	visible := 0
	if r.tip.Visible {
		visible = 1
	}
	errText := uint64(0)
	if r.err != nil {
		errText = headlessWitnessText(r.err.Error())
	}
	return fmt.Sprintf("%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
		part, state, bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, pixHash,
		headlessWitnessText(r.note), stats, r.at.X, r.at.Y,
		headlessWitnessText(strings.Join(r.lines[:], "\n")),
		r.cells.Min.X, r.cells.Min.Y, r.cells.Max.X, r.cells.Max.Y,
		r.tip.DelayMS, headlessWitnessText(r.tip.Target), visible,
		r.tip.Bounds.Min.X, r.tip.Bounds.Min.Y, r.tip.Bounds.Max.X, r.tip.Bounds.Max.Y,
		r.arts, errText)
}

type headlessWitnessCase struct {
	part, state int
	get         func() headlessWitness
}

func headlessWitnessMissionApp(v *Viewer) *App {
	return &App{flow: &flow{viewer: v, screen: ScreenMap}}
}

// headlessWitnessCases builds each state on its own synthetic App: the same
// fixtures the accessors' own tests use.
func headlessWitnessCases(t *testing.T) []headlessWitnessCase {
	t.Helper()
	pictureOf := func(a *App, part HeadlessPart) func() headlessWitness {
		return func() headlessWitness {
			pix, extras, err := a.HeadlessPicture(HeadlessRequest{Part: part, Cells: image.Rect(12, 12, 16, 16)})
			return headlessWitness{pix: pix, note: extras.Note, statistics: extras.Statistics, at: extras.At, lines: extras.Lines, cells: extras.Cells, tip: extras.Tooltip, err: err}
		}
	}

	town := newTestApp(t, appRows(1), nil)
	town.SetTown(&fakeStatsSurfaceTown{})
	if !town.flow.showTown("") {
		t.Fatal("setup: town refused")
	}
	gameMenu := newTestApp(t, appRows(1), nil)
	gameMenu.SetTown(&fakeStatsSurfaceTown{})
	if !gameMenu.flow.showTown("") {
		t.Fatal("setup: town refused")
	}
	gameMenu.flow.screen = ScreenGameMenu
	onMap, _ := mkOnMap(t)
	native := nativeMenuApp(t, image.Pt(1024, 768))

	chatOpen, chatV := mkOnMap(t)
	chatV.SetFont(messageFont())
	chatOpen.SetCheatCommands(func(string) {}, nil)
	if err := chatOpen.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := chatOpen.HeadlessType("#x", false); err != nil {
		t.Fatal(err)
	}
	chatClosed, chatClosedV := mkOnMap(t)
	chatClosedV.SetFont(messageFont())

	values := GameOptionValues{}
	var log []optionWrite
	options := optionsApp(t, &values, &log, nil)
	options.flow.menuFont = solidFont15()
	options.SetGameMenuArt(dialogContentArt())
	optionsClosed := optionsApp(t, &values, &log, nil)
	options.flow.screen = ScreenGameMenu
	options.flow.rebuildGameMenu(gameMenuGameOptionsPage, 0)

	notice := noticeViewer(t)
	notice.SetNotice("one", NoticeDialogue)
	noticeNone := noticeViewer(t)

	paneSubject := InventorySubject{ID: 1, Figure: solidPic(160, 200, color.RGBA{R: 0xf0, G: 0x0f, A: 0xff})}
	paneViewer := missionPaneViewer(t)
	paneViewer.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 12, 34)})
	paneViewer.sel = selection{1}
	paneViewer.SetInventorySubject(paneSubject)
	statViewer := missionPaneViewer(t)
	statViewer.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 12, 34)})
	statViewer.sel = selection{1}
	statViewer.SetInventorySubject(paneSubject)
	statViewer.toggleHudPanel(hudPanelDoll)
	column := columnViewer(t)
	hidden := noticeViewer(t)

	minimap := originalViewer(t, 64, 64)
	minimap.SetLocalOwner(1)
	minimap.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(12, 12), Owner: 1, Life: LifeAlive}})
	minimapApp := headlessWitnessMissionApp(minimap)

	readoutApp, readoutV := inspectionFixture(t, image.Pt(1024, 768))
	inspectionHover(t, readoutApp, readoutV, InspectionSubject{InspectionStructure, 7})
	readoutNone, _ := inspectionFixture(t, image.Pt(1024, 768))

	cursorApp, cursorV, _ := atOnMap(t)
	cursorApp.SetCursorRegistry(NewCursorRegistry([]CursorSlot{
		{Name: "default", Frames: []*image.RGBA{solidPic(9, 7, color.RGBA{B: 0xff, A: 0xff})}, Hotspot: image.Pt(5, 5), FrameCount: 1, PeriodMillis: 2000000000},
	}))
	cursorV.SetEntities(mcEntities())
	cursorV.cursorX, cursorV.cursorY, cursorV.hasCursor = 400, 300, true
	cursorV.cursorMgr.SetCursor("default")
	cursorNone, cursorNoneV, _ := atOnMap(t)
	cursorNoneV.armed = true

	tipApp, _ := acApp(t, acBook(0))
	tipV := tipApp.flow.viewer
	tipV.SetFont(messageFont())
	tipV.words.Command[commandCellCast] = "CAST"
	mage := panelEntity(sbUnitID, "Mage", 50, 100, 4, 4)
	mage.CastCapable = true
	tipV.SetEntities([]MapEntity{mage})
	tipV.sel = selection{sbUnitID}
	tipV.selectedSpell = tipV.spellbook[0].ID
	bar, ok := tipV.commandPanelBar()
	if !ok {
		t.Fatal("setup: no command panel")
	}
	p := commandCellRects(bar)[commandCellCast].Min.Add(image.Pt(3, 3))
	in := appInput{CursorX: p.X, CursorY: p.Y, Viewer: Input{CursorX: p.X, CursorY: p.Y}}
	start := time.Unix(100, 0)
	tipApp.step(in, start)
	tipApp.step(in, start.Add(500*time.Millisecond))

	var nilApp *App
	return []headlessWitnessCase{
		{0, 0, pictureOf(nilApp, HeadlessPartFrame)},
		{0, 1, pictureOf(town, HeadlessPartFrame)},
		{0, 2, pictureOf(gameMenu, HeadlessPartFrame)},
		{0, 3, pictureOf(onMap, HeadlessPartFrame)},
		{0, 4, pictureOf(native, HeadlessPartFrame)},
		{1, 0, pictureOf(chatOpen, HeadlessPartChat)},
		{1, 1, pictureOf(chatClosed, HeadlessPartChat)},
		{2, 0, pictureOf(options, HeadlessPartGameOptions)},
		{2, 1, pictureOf(optionsClosed, HeadlessPartGameOptions)},
		{3, 0, pictureOf(headlessWitnessMissionApp(notice), HeadlessPartNotice)},
		{3, 1, pictureOf(headlessWitnessMissionApp(noticeNone), HeadlessPartNotice)},
		{4, 0, pictureOf(headlessWitnessMissionApp(paneViewer), HeadlessPartCharacterPane)},
		{4, 1, pictureOf(headlessWitnessMissionApp(statViewer), HeadlessPartCharacterPane)},
		{4, 2, pictureOf(&App{flow: &flow{}}, HeadlessPartCharacterPane)},
		{4, 3, pictureOf(town, HeadlessPartCharacterPane)},
		{5, 0, pictureOf(headlessWitnessMissionApp(column), HeadlessPartInventoryPack)},
		{5, 1, pictureOf(headlessWitnessMissionApp(hidden), HeadlessPartInventoryPack)},
		{6, 0, pictureOf(headlessWitnessMissionApp(column), HeadlessPartBottomHUD)},
		{6, 1, pictureOf(headlessWitnessMissionApp(hidden), HeadlessPartBottomHUD)},
		{7, 0, pictureOf(minimapApp, HeadlessPartMinimap)},
		{8, 0, pictureOf(minimapApp, HeadlessPartMinimapComposed)},
		{9, 0, pictureOf(minimapApp, HeadlessPartMinimapCells)},
		{9, 1, pictureOf(headlessWitnessMissionApp(hidden), HeadlessPartMinimapCells)},
		{10, 0, pictureOf(readoutApp, HeadlessPartStructureReadout)},
		{10, 1, pictureOf(readoutNone, HeadlessPartStructureReadout)},
		{10, 2, pictureOf(town, HeadlessPartStructureReadout)},
		{11, 0, pictureOf(cursorApp, HeadlessPartMissionCursor)},
		{11, 1, pictureOf(cursorNone, HeadlessPartMissionCursor)},
		{12, 0, pictureOf(headlessWitnessMissionApp(column), HeadlessPartMissionCard)},
		{12, 1, pictureOf(headlessWitnessMissionApp(statViewer), HeadlessPartMissionCard)},
		{12, 2, pictureOf(town, HeadlessPartMissionCard)},
		{13, 0, pictureOf(headlessWitnessMissionApp(column), HeadlessPartDrawnMissionCard)},
		{13, 1, pictureOf(town, HeadlessPartDrawnMissionCard)},
		{14, 0, pictureOf(headlessWitnessMissionApp(column), HeadlessPartCommandPanel)},
		{14, 1, pictureOf(headlessWitnessMissionApp(paneViewer), HeadlessPartCommandPanel)},
		{15, 0, pictureOf(tipApp, HeadlessPartTooltip)},
		{15, 1, pictureOf(town, HeadlessPartTooltip)},
		{16, 0, pictureOf(native, HeadlessPartGameMenuPanel)},
		{16, 1, pictureOf(options, HeadlessPartGameMenuPanel)},
		{17, 0, func() headlessWitness {
			pix, arts, err := column.HeadlessMapFrame(image.Pt(4, 4), frame.W/2, frame.H/2)
			return headlessWitness{pix: pix, arts: len(arts), err: err}
		}},
	}
}

func headlessWitnessRows(t *testing.T, plant func(part, state int, r *headlessWitness)) []string {
	t.Helper()
	var rows []string
	for _, c := range headlessWitnessCases(t) {
		r := c.get()
		if plant != nil {
			plant(c.part, c.state, &r)
		}
		rows = append(rows, r.row(c.part, c.state))
	}
	return rows
}

func readHeadlessPictureGolden(t *testing.T) []string {
	t.Helper()
	f, err := os.Open(headlessPictureGolden)
	if err != nil {
		t.Fatalf("headless picture golden: %v", err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		if strings.Trim(sc.Text(), "0123456789 -") != "" {
			t.Fatalf("golden line holds more than numbers: %q", sc.Text())
		}
		out = append(out, sc.Text())
	}
	return out
}

func diffHeadlessPicture(want, got []string) string {
	if len(want) != len(got) {
		return fmt.Sprintf("%d golden lines, %d computed", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			return fmt.Sprintf("line %d:\n golden   %s\n computed %s", i+1, want[i], got[i])
		}
	}
	return ""
}

// TestHeadlessPictureWitness fails when any part's picture or returned extras
// differ from the golden recorded before the accessors became one request.
func TestHeadlessPictureWitness(t *testing.T) {
	got := headlessWitnessRows(t, nil)
	if os.Getenv(headlessPictureRecord) == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(headlessPictureGolden, []byte(strings.Join(got, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want := readHeadlessPictureGolden(t)
	if d := diffHeadlessPicture(want, got); d != "" {
		t.Fatal(d)
	}
	drawn := 0
	for _, line := range want {
		var part, state, minX, minY, maxX, maxY int
		if _, err := fmt.Sscan(line, &part, &state, &minX, &minY, &maxX, &maxY); err != nil {
			t.Fatal(err)
		}
		if maxX > minX && maxY > minY {
			drawn++
		}
	}
	if drawn < 20 {
		t.Fatalf("%d cases drew a picture, want at least 20", drawn)
	}
}

// A planted fault in any field of a case must reach the golden comparison.
func TestHeadlessPictureWitnessFailsOnAPlantedFault(t *testing.T) {
	want := readHeadlessPictureGolden(t)
	faults := []func(r *headlessWitness){
		func(r *headlessWitness) { r.pix.Pix[0] ^= 1 },
		func(r *headlessWitness) { r.note = "planted" },
		func(r *headlessWitness) { r.at.X++ },
		func(r *headlessWitness) { r.lines[0] = "planted" },
		func(r *headlessWitness) { r.tip.Visible = !r.tip.Visible },
		func(r *headlessWitness) { r.err = fmt.Errorf("planted") },
	}
	for i, fault := range faults {
		got := headlessWitnessRows(t, func(part, state int, r *headlessWitness) {
			if part == 4 && state == 0 {
				fault(r)
			}
		})
		if diffHeadlessPicture(want, got) == "" {
			t.Errorf("planted fault %d passed the witness", i)
		}
	}
}
