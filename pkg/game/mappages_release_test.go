package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/firstgame"
	"againrom/internal/synth"
)

func TestReleaseMapEditorLooseMapPages(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	lang := x.pageLanguage()
	body := map[string]string{"en": "Map page EN", "ru": "Страница карты"}[lang]
	page := []byte("<part=1>\n" + body)
	script := scriptPayload([]scriptNode{
		{label: "Drop", opcode: dropActionOpcode, id: 99, values: []uint32{16, 16}, tags: []uint32{5, 6}},
		{label: "Message", opcode: 2, id: 1, values: []uint32{1}, tags: []uint32{1}},
	}, []scriptNode{
		{label: "Zero", opcode: 0x10002, id: 1, tags: []uint32{1}},
		{label: "Zero", opcode: 0x10002, id: 2, tags: []uint32{1}},
	})
	trigger := winTrigger("Message", 1)
	binary.LittleEndian.PutUint32(trigger[0x80:], 1)
	binary.LittleEndian.PutUint32(trigger[0x84:], 2)
	script = append(binary.LittleEndian.AppendUint32(script[:len(script)-4], 1), trigger...)
	original := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Name: "Map pages", Type7Payload: script})
	root := t.TempDir()
	input := filepath.Join(root, "input", "map.alm")
	writeTestPlayInput(t, input, original)
	writeTestPlayInput(t, filepath.Join(root, "input", "map.text", lang, "event01.txt"), page)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(input); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if !bytes.Equal(s.workspace.Pages["event01.txt"], page) {
		t.Fatal("Open lost the map page")
	}
	rows, err := x.CheckMap(s.model, input)
	if err != nil || len(ofRule(rows, "mod-message-text")) != 0 {
		t.Fatalf("mapcheck ignored the page: %+v: %v", rows, err)
	}
	missing := filepath.Join(root, "missing", "map.alm")
	writeTestPlayInput(t, missing, original)
	rows, err = x.CheckMap(s.model, missing)
	if err != nil || len(ofRule(rows, "mod-message-text")) != 1 {
		t.Fatalf("missing-page loss control: %+v: %v", rows, err)
	}
	output := filepath.Join(root, "saved", "saved.alm")
	if err := os.MkdirAll(filepath.Dir(output), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAs(output); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(saved, original) {
		t.Fatalf("SaveAs ALM differs: %v", err)
	}
	savedPage, err := os.ReadFile(filepath.Join(root, "saved", "saved.text", lang, "event01.txt"))
	if err != nil || !bytes.Equal(savedPage, page) {
		t.Fatalf("SaveAs page differs: %v", err)
	}
	if err := e.Open(output); err != nil {
		t.Fatal(err)
	}
	s = e.Document().Edits.(*mapInspectionEdits)
	if !bytes.Equal(s.workspace.Pages["event01.txt"], page) {
		t.Fatal("cold Open lost saved page")
	}
	if err := workspaceField(e, "pages", 0, "event01.txt"); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("pages"); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("tab"); err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("AGAINROM_MAP_PAGES_FRAMES"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		pic, _, err := e.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		write := func(name string, img image.Image) {
			t.Helper()
			f, err := os.Create(filepath.Join(dir, lang+"-"+name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, img); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		}
		write("pages", pic)
		write("pages-crop", pic.SubImage(image.Rect(180, 170, 1100, 650)))
	}
	_ = e.Key("escape")
	program := filepath.Join(root, "program.exe")
	writeTestPlayInput(t, program, []byte("recording host"))
	if err := os.Mkdir(filepath.Join(root, "profile"), 0700); err != nil {
		t.Fatal(err)
	}
	starts := 0
	x.SetTestPlay(TestPlayHost{
		Game: program,
		Profile: func() (RuntimeProfile, error) {
			return ResolveRuntimeProfile(filepath.Join(root, "profile", "saves"), "", "", x.archives.Root)
		},
		Start: func(string, []string) error { starts++; return nil },
	})
	if _, err := s.TestPlay(); err != nil || starts != 1 {
		t.Fatalf("test play staging: %v, starts=%d", err, starts)
	}
	modsDir := filepath.Join(root, "profile", TestPlayModID)
	m := loadWorkspaceMod(t, x.archives.Root, filepath.Join(modsDir, TestPlayModID))
	if !bytes.Equal(m.TextFiles["event01.txt"], savedPage) {
		t.Fatal("test play package page differs")
	}
	f := guestFront(t, modsDir, TestPlayModID)
	p, ms, err := StartScenarioMission(f.Archives, HeadlessScenario{Version: 6, Stage: StageMission, Mission: m.Number})
	if err != nil {
		t.Fatal(err)
	}
	if len(ms.Raises) != 1 || ms.Raises[0].Event != 1 {
		t.Fatalf("map raise list: %+v", ms.Raises)
	}
	control, _, err := StartScenarioMission(f.Archives, HeadlessScenario{Version: 6, Stage: StageMission, Mission: m.Number})
	if err != nil {
		t.Fatal(err)
	}
	control.ann = nil
	for range 32 {
		p.step(nil)
		control.step(nil)
		if p.World.Hash() != control.World.Hash() {
			t.Fatal("announcement changed hashed state")
		}
	}
	if len(p.announced) != 1 || !p.announced[0].Shown || !reflect.DeepEqual(p.announced[0].Parts, []string{"\n" + body}) {
		t.Fatalf("map message announcement: %+v", p.announced)
	}
	absent := *x.archives
	absent.Containers = x.archives.Containers.Fork()
	address, _ := MissionMap(m.Number)
	absent.Containers.Overlay(address, func() ([]byte, error) { return bytes.Clone(original), nil })
	loss, _, err := StartScenarioMission(&absent, HeadlessScenario{Version: 6, Stage: StageMission, Mission: m.Number})
	if err != nil {
		t.Fatal(err)
	}
	for range 32 {
		loss.step(nil)
	}
	if len(loss.announced) != 1 || loss.announced[0].Shown || loss.announced[0].Shipped {
		t.Fatalf("absent-page announcement loss control: %+v", loss.announced)
	}
	app, next := guestRoute(t, f, []int{0, 0, 0})
	if next != m.Number {
		t.Fatalf("guest route=%d, want %d", next, m.Number)
	}
	if err := app.OpenMission(f.MissionOpener(next)); err != nil {
		t.Fatal(err)
	}
	savBefore := guestMissionSave(t, f)
	if !f.live.openDialogue(1) {
		t.Fatal("window route lost map page")
	}
	savAfter := guestMissionSave(t, f)
	if !bytes.Equal(savBefore, savAfter) {
		t.Fatal("opening map page changed SAV bytes")
	}
	cold := guestFront(t, modsDir, TestPlayModID)
	open, town, err := cold.RestoreOriginal(savAfter)
	if err != nil || town {
		t.Fatalf("cold LOAD: town=%v: %v", town, err)
	}
	if err := cold.App("Map pages").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if !cold.live.openDialogue(1) || !bytes.Equal(cold.live.mission.payload, EncodeInstallTextFile(page, x.font.Selector)) {
		t.Fatal("cold LOAD lost the map page overlay")
	}
	export := filepath.Join(root, "export")
	if err := os.Mkdir(export, 0700); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(export, s.workspace.Package.ID)
	_, err = s.saveWorkspacePackage(folder)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := os.ReadFile(filepath.Join(folder, "mission", "text", lang, "event01.txt"))
	if err != nil || !bytes.Equal(exported, savedPage) {
		t.Fatalf("W page is not a byte copy: %v", err)
	}
	t.Logf("lang=%s ALM=%d bytes unchanged; page=%d bytes copied; SAV=%d bytes identical before/after dialogue; cold LOAD; mapcheck missing-page control; test-play announcement=%q", lang, len(saved), len(page), len(savBefore), p.announced[0].Parts)
}
