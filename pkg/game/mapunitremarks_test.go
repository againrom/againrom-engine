package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func TestUnitWindowDialoguePortraitInDollColumn(t *testing.T) {
	_, s := speakerPickEditor(t)
	for _, unit := range []int{0, 1, 3} {
		n, miss := s.UnitSpeaker(unit)
		if miss.Cause != ui.SpeakerShown {
			t.Fatal(miss)
		}
		field := s.speakerField(unit)
		for _, f := range s.Fields(ui.KindUnit, unit) {
			if f.Key == "speaker" {
				field = f
			}
		}
		want := missionStartFace(t, s, int(n))
		if field.Section != "doll" || field.Picture == nil || field.Picture.Bounds() != want.Bounds() || !bytes.Equal(field.Picture.Pix, want.Pix) {
			t.Errorf("unit %d speaker %d portrait section %q; want dialogue portrait under doll", unit, n, field.Section)
		}
	}
}

func TestUnitCardNameReachesTestPlayMap(t *testing.T) {
	e, s := speakerPickEditor(t)
	e.SetCardFont(viewsCardFont())
	profile := useSpeakerTestPlay(t, s)
	s.AllowErrors()
	e.Select(recordOf(e.Document(), "Unit", 0), true)
	fields := e.ObjectFields()
	c := &editorClicker{t: t}
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: fieldIndex(t, fields, "name")})
	c.click(e, r, ok, "name")
	e.TextInput("Test paladin")
	if err := e.Key("testplay"); err != nil {
		t.Fatal(err)
	}
	id, err := unitID(s.model, 0)
	if err != nil || s.authoredUnitNames()[id] != "Test paladin" {
		t.Fatalf("F5 did not apply the card name for unit %d", id)
	}
	if e.TestPartyOpen() || e.NoticeOpen() {
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
	}
	m := loadWorkspaceMod(t, s.inspector.archives.Root, filepath.Join(profile, TestPlayModID, TestPlayModID))
	names := unitNames(string(m.TextFiles[unitNamesPage]))
	if err != nil || names[id] != "Test paladin" {
		t.Fatalf("test map unit %d name %q; want Test paladin (%v)", id, names[id], err)
	}
	loaded, err := mapedit.New(m.MapBytes)
	if err != nil {
		t.Fatal(err)
	}
	view, err := loaded.Map()
	if err != nil {
		t.Fatal(err)
	}
	trial := *s
	trial.model = loaded
	world := missionStart(t, &trial)
	world.authoredNames = names
	if len(view.Units) == 0 || world.shownName(0) != "Test paladin" {
		t.Fatal("the test mission does not show its authored name")
	}
}

func TestInstalledUnitRemarksFrames(t *testing.T) {
	dir := os.Getenv("AGAINROM_UNIT_REMARK_FRAMES")
	if dir == "" {
		t.Skip("needs AGAINROM_UNIT_REMARK_FRAMES and AGAINROM_ASSETS")
	}
	x := installedInspector(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, who := range []string{"snoot", "animal"} {
		e := x.Editor()
		e.SetPanelProfile([]byte(`{"right":{"minimized":true,"view":"technical","width":520}}`), nil, nil)
		e.Layout(1920, 1080)
		if err := e.Open("scenario/10.alm"); err != nil {
			t.Fatal(err)
		}
		closeMapPanel(t, e)
		s := objectEdits(t, e)
		m, err := s.model.Map()
		if err != nil {
			t.Fatal(err)
		}
		unit := -1
		for i, u := range m.Units {
			if who == "snoot" && u.UnitID == 51 {
				unit = i
				break
			}
			entity, _, _, err := s.UnitSettings(i, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, miss := s.UnitSpeaker(i)
			if who == "animal" && entity.Char.Band == ui.CharacterBandCreature && miss.Cause == ui.SpeakerShown {
				unit = i
				break
			}
		}
		if unit < 0 {
			t.Fatalf("no %s", who)
		}
		e.Select(recordOf(e.Document(), "Unit", unit), true)
		for _, mode := range []string{"detailed", "standard"} {
			if mode == "standard" {
				r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlViewSwitch})
				(&editorClicker{t: t}).click(e, r, ok, "view switch")
			}
			witness.PNG(t, filepath.Join(dir, fmt.Sprintf("installed-%s-%s-%s.png", installLanguage(x), who, mode)), e.ChromeImage())
		}
	}
}

func TestInstalledMissionTenOwnNameInUnitCard(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	for i, u := range m.Units {
		if u.UnitID != 51 {
			continue
		}
		if u.ClassSubID != 52 || len(x.table.NPCNames) < 52 || x.table.NPCNames[51] == "" {
			t.Fatal("mission 10 named placement changed")
		}
		e.Select(recordOf(e.Document(), "Unit", i), true)
		subject, ok := e.UnitCharacterSubject()
		name, _ := ui.PanelSubjectName(subject)
		want := x.table.NPCNames[51]
		if !ok || name != want {
			t.Fatalf("mission10 unit 51 card name %q; want NPC 52 name %q", x.messageUTF8([]byte(name)), x.messageUTF8([]byte(want)))
		}
		t.Logf("unit 51 NPC 52 text/npcnames.txt line 52: %s", x.messageUTF8([]byte(want)))
		return
	}
	t.Fatal("mission10 has no unit 51")
}

func TestNamedPlacementOwnNameInUnitCard(t *testing.T) {
	e, s := speakerPickEditor(t)
	s.inspector.table.NPCNames = make([]string, 26)
	s.inspector.table.NPCNames[24] = "Fixture paladin"
	e.Select(recordOf(e.Document(), "Unit", 0), true)
	subject, ok := e.UnitCharacterSubject()
	name, _ := ui.PanelSubjectName(subject)
	if !ok || name != "Fixture paladin" {
		t.Fatalf("named placement card name %q; want %q", name, "Fixture paladin")
	}
}
