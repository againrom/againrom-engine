package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// generatorRecordFile holds what the hero generator answers for both games
// from the fixture tables below: every picture's preset, the party every
// choice makes, and the state an accepted result begins.
const generatorRecordFile = "testdata/generatorprofile.golden"

// generatorRecordStats is the spread every recorded result spends.
var generatorRecordStats = []int{30, 20, 18, 16}

// generatorFirstFront is a first-game front end over a fixture table: three
// archetype sections, one of them an illegal spread, and three of the four
// base rows.
func generatorFirstFront(t *testing.T) *FrontEnd {
	t.Helper()
	section := func(name string, body, reaction, mind, spirit, skill int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: kindDir, Children: []synth.RegNode{
			{Name: "Face", Kind: kindInt, Int: 2}, {Name: "Body", Kind: kindInt, Int: body},
			{Name: "Reaction", Kind: kindInt, Int: reaction}, {Name: "Mind", Kind: kindInt, Int: mind},
			{Name: "Spirit", Kind: kindInt, Int: spirit}, {Name: "Skill", Kind: kindInt, Int: skill}}}
	}
	r, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
		section("MaleFighter", 41, 35, 20, 15, 1),
		section("MaleMage", 29, 22, 40, 34, 3),
		section("FemaleFighter", 32, 40, 31, 24, 5),
		section("FemaleMage", 99, 25, 33, 41, 2),
	}))
	if err != nil {
		t.Fatal(err)
	}
	humans := fourBaseHumans()
	humans[4].name = "Missing"
	f := missionFrontEnd(t)
	f.Table = &mapload.Table{NPC: data.LoadNPCDefs(r), Humans: humans, Shapes: emptyScale{}, Materials: emptyScale{}}
	f.Humans = humans
	f.Bodies, f.Town = data.NewBodyList("unarmed", "mage"), NewTown(Campaign{})
	f.Maps = []MapEntry{{Source: "10.alm", Name: "B", Mission: 10}}
	return f
}

// generatorSecondFront is a second-game front end over a fixture table with
// three of the four template rows and the town text the commit checks.
func generatorSecondFront(t *testing.T) *FrontEnd {
	t.Helper()
	profile, ok := base.Find(base.ROM2EN)
	if !ok {
		t.Fatal("missing ROM2 profile")
	}
	row := func(name string, body, reaction, mind, spirit, face int32) dbEntry {
		e := companionHumansRow(name, 10210, face)
		e.params[0], e.params[1], e.params[2], e.params[3] = body, reaction, mind, spirit
		return e
	}
	humans := dbCollection{{},
		row("Start_MF", 40, 30, 20, 20, 32),
		row("Start_FF", 30, 40, 20, 20, 33),
		row("Start_FM", 20, 20, 40, 30, 34),
	}
	src := townTextFS(t, []synth.File{{Path: "text/town.txt", Data: []byte("[npc1talk1]\r\nHello\r\n")}})
	f := &FrontEnd{InstallResources: InstallResources{
		Archives: &Archives{Base: base.Match{Profile: profile}, Containers: src},
		Campaign: resolved(Campaign{}, nil),
		Table: readUnder(base.GameROM2, &mapload.Table{Humans: humans, Armors: dressArmors(),
			Shapes: emptyScale{}, Materials: emptyScale{}, NPCCatalog: &data.NPCCatalog{}}),
		Bodies: data.NewBodyList("unarmed", "mage"),
	}, CampaignSession: CampaignSession{Town: NewTown(Campaign{})}}
	f.Humans = f.Table.Humans
	f.Maps = []MapEntry{{Source: "10.alm", Name: "B", Mission: 10}}
	return f
}

// generatorRecord is the generator's answer on f as text.
func generatorRecord(t *testing.T, game string, f *FrontEnd) string {
	t.Helper()
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, game+" "+format+"\n", args...) }
	js := func(v any) string {
		out, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("%s: %v", game, err)
		}
		return string(out)
	}
	setup := f.ChargenSetup()
	line("presets %v skills %v accept-shows-town %v", setup.Presets, setup.PresetSkills, setup.AcceptShowsTown)
	for sex := 0; sex < 2; sex++ {
		for class := 0; class < 2; class++ {
			for skill := 0; skill < data.SkillSlots-1; skill++ {
				res := ui.ChargenResult{Name: "Record", Choices: []int{sex, class, skill}, Stats: generatorRecordStats}
				line("party %d %d %d %s", sex, class, skill, js(f.ChargenParty(res)))
			}
		}
	}
	res := ui.ChargenResult{Name: "Start", Choices: []int{1, 1, 2}, Stats: generatorRecordStats, Difficulty: 2}
	if e := f.newGameChargen(0); e != nil {
		open, err := e.Begin(res)
		line("picker begin opener %v error %v", open != nil, err)
	}
	open, err := f.NewGameBegin(10)(res)
	line("begin opener %v error %v", open != nil, err)
	if open != nil {
		v, _, _, _, _, _, _, _, _, _, err := open()
		line("open viewer %v error %v", v != nil, err)
	}
	line("difficulty %d gold %d carried %s", f.Difficulty, f.Town.gold, js(f.Carried))
	if c := f.Town.second; c != nil {
		line("bank 776 %d 781 %d", c.bank[776], c.bank[781])
	}
	return b.String()
}

// The generator's presets, parties and starts of both games equal the record
// taken before the two games' generator hooks became one profile row.
func TestGeneratorProfileKeepsTheRecordedAnswers(t *testing.T) {
	got := generatorRecord(t, "first", generatorFirstFront(t)) + generatorRecord(t, "second", generatorSecondFront(t))
	if os.Getenv("AGAINROM_WRITE_GENERATOR_RECORD") != "" {
		if err := os.WriteFile(filepath.FromSlash(generatorRecordFile), []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(filepath.FromSlash(generatorRecordFile))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := 0; i < len(gl) || i < len(wl); i++ {
			var g, w string
			if i < len(gl) {
				g = gl[i]
			}
			if i < len(wl) {
				w = wl[i]
			}
			if g != w {
				t.Fatalf("line %d:\n got %s\nwant %s", i+1, g, w)
			}
		}
	}
}
