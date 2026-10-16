package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"againrom/pkg/mod"
)

func TestReleaseMapEditorWorkspaceRoutesAndTexts(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(640, 480)
	if err := e.Key("new"); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("clear")
	e.TextInput("40x56")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if e.Document().Width != 40 || e.Document().Height != 56 {
		t.Fatal("N did not create a rectangle")
	}
	if err := workspaceField(e, "metadata", 0, "Карта"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "metadata", 1, "Дорога\nДом"); err != nil {
		t.Fatal(err)
	}
	meta := inspectionSectionOffsets(t, e.Document().SourceBytes)[0]
	if got := e.Document().SourceBytes[meta+0x30 : meta+0x36]; !bytes.Equal(got, []byte{0xca, 0xe0, 0xf0, 0xf2, 0xe0, 0}) {
		t.Fatalf("map name bytes %x", got)
	}
	fontName := string([]byte{0xca, 0xe0, 0xf0, 0xf2, 0xe0})
	if x.font.Selector == 1 {
		fontName = string([]byte{0x8a, 0xa0, 0xe0, 0xe2, 0xa0})
	}
	shown := false
	for _, r := range e.Document().Records {
		if r.Section == 0 {
			for _, line := range r.Details {
				shown = shown || line == fontName
			}
		}
	}
	if !shown {
		t.Fatal("map name did not cross the installed-font boundary exactly once")
	}
	pages := map[string]string{"title.txt": "Карта", "briefing.txt": "Задание\nПродолжение", "briefmap.txt": "Дорога\nДом", "event01.txt": "<part=1>\nПривет\nВторая строка", "tips01.txt": "Совет\nДальше", "event100.txt": "<part=1>\nСотое сообщение", "tips100.txt": "Сотый совет"}
	for _, name := range []string{"title.txt", "briefing.txt", "briefmap.txt", "event01.txt", "tips01.txt", "event100.txt", "tips100.txt"} {
		workspacePage(t, e, name, pages[name])
	}
	if err := e.Key("drop"); err != nil {
		t.Fatal(err)
	}
	clickCell(e, image.Pt(20, 20))
	parent := t.TempDir()
	folder := filepath.Join(parent, "workspace")
	if err := workspaceField(e, "package", 1, "workspace"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "package", 0, folder); err != nil {
		t.Fatal(err)
	}
	m := loadWorkspaceMod(t, x.archives.Root, folder)
	if m.Kind != mod.MissionGuest || e.Dirty() {
		t.Fatalf("created route %s dirty %v", m.Kind, e.Dirty())
	}
	check := func(f *FrontEnd, mission mod.Mission) {
		t.Helper()
		address, _ := MissionMap(mission.Number)
		if got, err := f.Archives.Containers.ReadFile(address); err != nil || !bytes.Equal(got, mission.MapBytes) {
			t.Fatalf("runtime map: %v", err)
		}
		for name, value := range pages {
			path := "main/text/battle/m" + strconv.Itoa(mission.Number) + "/" + name
			got, err := f.Archives.Containers.ReadFile(path)
			want := EncodeInstallTextFile([]byte(value), x.font.Selector)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("%s: %x want %x: %v", name, got, want, err)
			}
		}
		if f.live == nil || !f.live.openDialogue(1) || !bytes.Equal(f.live.mission.payload, EncodeInstallTextFile([]byte(pages["event01.txt"]), x.font.Selector)) {
			t.Fatal("live authored event page differs")
		}
		body, ok := dialoguePart(f.live.mission.payload, 1, f.live.mission.dialogueAudience)
		if !ok || !bytes.Equal([]byte(body), EncodeInstallTextFile([]byte("Привет\nВторая строка"), x.font.Selector)) {
			t.Fatalf("authored dialogue part %q (%v)", body, ok)
		}
		f.live.advanceNotice()
		if !f.live.openDialogue(100) || !bytes.Equal(f.live.mission.payload, EncodeInstallTextFile([]byte(pages["event100.txt"]), x.font.Selector)) {
			t.Fatal("event100 authored in the window did not reach live dialogue")
		}
	}
	guest := guestFront(t, parent, "workspace")
	app, next := guestRoute(t, guest, []int{0, 0, 0})
	if next != m.Number {
		t.Fatalf("campaign opened %d, package declares %d", next, m.Number)
	}
	if err := app.OpenMission(guest.MissionOpener(next)); err != nil {
		t.Fatal(err)
	}
	check(guest, m)
	if err := workspaceField(e, "package", 3, "replace"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "package", 4, "20"); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	m = loadWorkspaceMod(t, x.archives.Root, folder)
	replacement := guestFront(t, parent, "workspace")
	replaceRoute(t, replacement)
	check(replacement, m)
	title, brief := worldMapMissionText(t, replacement, 20)
	if title != string(EncodeInstallTextFile([]byte(pages["title.txt"]), x.font.Selector)) || brief != string(EncodeInstallTextFile([]byte(pages["briefmap.txt"]), x.font.Selector)) {
		t.Fatal("world map lost authored title or briefmap")
	}
	fresh := x.Editor()
	if err := fresh.Open("new/32x32"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(fresh, "package", 0, folder); err != nil {
		t.Fatal(err)
	}
	s := fresh.Document().Edits.(*mapInspectionEdits)
	if s.workspace.Package.Kind != mod.MissionReplace || s.workspace.Package.Number != 20 || fresh.Dirty() || fresh.Document().Name != "Карта" {
		t.Fatal("fresh editor lost selected route/name/checkpoint")
	}
	for name, value := range pages {
		if string(s.workspace.Pages[name]) != value {
			t.Fatalf("fresh page %s differs", name)
		}
	}
	if _, err := os.Stat(filepath.Join(folder, editorPackageRecord)); err != nil {
		t.Fatal(err)
	}
	t.Log(fmt.Sprintf("selector=%d; window40x56; CP1251 metadata; %d UTF-8 pages with multiline and event100/tips100; normal guest and replacement missions; live dialogue/world-map title/briefmap; cold editor reopen", x.font.Selector, len(pages)))
}
