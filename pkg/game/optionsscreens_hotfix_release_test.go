package game

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/fame"
	"againrom/pkg/ui"
)

func TestReleaseQuestObjectivesAndMainMenuHall(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	profile := t.TempDir()
	f.hallStore = fameStore{Dir: profile, OriginalDir: f.Archives.Root}
	a := f.App("objectives and hall")
	a.Layout(640, 480)
	artifacts := t.TempDir()
	if out := os.Getenv("AGAINROM_OPTIONS_ARTIFACTS"); out != "" {
		artifacts = filepath.Join(out, filepath.Base(f.Archives.Root))
		if err := os.MkdirAll(artifacts, 0755); err != nil {
			t.Fatal(err)
		}
	}
	capture := func(name string, pix *image.RGBA) {
		t.Helper()
		if pix == nil {
			t.Fatal("missing composition", name)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, pix); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(artifacts, name+".png"), buf.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}
	seed, err := os.ReadFile(filepath.Join(f.Archives.Root, "famehall.dat"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := fame.Parse(seed)
	if err != nil {
		t.Fatal(err)
	}
	for visit := 0; visit < 2; visit++ {
		if err := a.HeadlessActivate("hall of fame"); err != nil || a.Screen() != ui.ScreenEnding {
			t.Fatal("main-menu hall", err, a.Screen())
		}
		view := f.hallOfFame()
		if !view.HallAvailable || len(view.Hall) != len(parsed) {
			t.Fatal("hall seed", view)
		}
		for i, row := range view.Hall {
			if row.Score != parsed[i].Score || row.Name != parsed[i].Name {
				t.Fatal("hall reordered or changed", i, row, parsed[i])
			}
		}
		pix, note, err := a.HeadlessFrame()
		if err != nil || note != "" {
			t.Fatal(err, note)
		}
		capture("hall", pix)
		if visit == 0 {
			err = a.HeadlessKey("escape")
		} else {
			err = a.HeadlessActivate(a.HeadlessRows()[0].Text)
		}
		if err != nil || a.Screen() != ui.ScreenMenu {
			t.Fatal("hall return", err, a.Screen())
		}
	}
	if _, err := os.Stat(filepath.Join(profile, "famehall.dat")); !os.IsNotExist(err) {
		t.Fatal("viewing hall wrote the profile", err)
	}
	if f.fame.Result != nil || f.completedCampaign() {
		t.Fatal("hall view manufactured a result")
	}
	for _, n := range []int{10, 31, 110} {
		if err := a.OpenMission(f.MissionOpener(n)); err != nil {
			t.Fatal(n, err)
		}
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		before := f.live.world.Hash()
		if err := a.HeadlessGameMenuAction("objectives"); err != nil {
			t.Fatal(err)
		}
		raw, err := f.Archives.Containers.ReadFile("main/text/battle/m" + strconv.Itoa(n) + "/briefing.txt")
		if err != nil {
			t.Fatal(err)
		}
		rows := a.HeadlessRows()
		if len(rows) < 3 || rows[0].Text != f.Words.QuestHeading || rows[len(rows)-1].Text != f.Words.NoticeButton {
			t.Fatal("objective controls", rows)
		}
		var body []string
		for _, row := range rows[1 : len(rows)-1] {
			if row.Choosable {
				t.Fatal("objective text is a control")
			}
			body = append(body, row.Text)
		}
		want := strings.Join(strings.Fields(strings.ReplaceAll(string(raw), "#", " ")), " ")
		if strings.Join(body, " ") != want {
			t.Fatalf("mission %d objective differs from its installed briefing: got % x want % x", n, []byte(strings.Join(body, " ")), []byte(want))
		}
		capture("quest-"+strconv.Itoa(n), a.QuestObjectivePanel())
		for i := 0; i < 8; i++ {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		if f.live.world.Hash() != before {
			t.Fatal("objectives advanced world")
		}
		if err := a.HeadlessGameMenuAction("page-return"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("return"); err != nil || a.Screen() != ui.ScreenMap {
			t.Fatal("objective return", err, a.Screen())
		}
	}
	got, err := os.ReadFile(filepath.Join(f.Archives.Root, "famehall.dat"))
	if err != nil || !bytes.Equal(seed, got) {
		t.Fatal("read-only original hall changed", err)
	}
}
