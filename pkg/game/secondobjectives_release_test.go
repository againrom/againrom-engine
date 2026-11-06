package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondMapObjectiveActions is, per installed map, the instant 36 (n, mode)
// each trigger runs as R2-ENGINE-894 and R2-ENGINE-895 state them.
var secondMapObjectiveActions = map[int]map[int][][2]int32{
	21: {0: {{1, 1}}, 1: {{2, 1}}, 4: {{2, 2}}, 9: {{1, 2}}},
	31: {2: {{1, 1}}, 3: {{2, 1}}, 5: {{1, 2}}, 13: {{2, 2}}},
}

// Maps 21 and 31 as installed: their triggers run instant 36 where the pages
// say and nowhere else, no node is mode 4 and no check reads an objective
// slot (op 24); the texts run #subobjective0, a line break alone, to 2; and
// map 21's first runner pass opens objective 1, which the window shows with
// the empty-box mark.
func TestReleaseSecondMapObjectiveWiring(t *testing.T) {
	archives := secondGameRoot(t)
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	code := InstallTextCode(archives.Containers, archives.Game().Edition())
	for _, n := range []int{21, 31} {
		party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
		ms, err := StartMission(archives.Containers, n, defs.Table, mapload.DifficultyNormal, party)
		if err != nil {
			t.Fatal(n, err)
		}
		s := ms.World.Script()
		instants := s.Instants()
		got := map[int][][2]int32{}
		for i, tr := range s.Triggers() {
			for _, k := range tr.Instants {
				if k == sim.ScriptNone || instants[k].Op != 36 {
					continue
				}
				got[i] = append(got[i], [2]int32{instants[k].Args[0], instants[k].Args[1]})
			}
		}
		if !reflect.DeepEqual(got, secondMapObjectiveActions[n]) {
			t.Fatalf("map %d runs instant 36 at %v, want %v", n, got, secondMapObjectiveActions[n])
		}
		for _, c := range s.Checks() {
			if c.Op == sim.ScriptCheckObjective {
				t.Fatalf("map %d reads an objective slot", n)
			}
		}
		labels := secondGameObjectiveLabels(archives.Containers, code, n)
		if len(labels) != 3 || labels[0] != "" || labels[1] == "" || labels[2] == "" {
			t.Fatalf("map %d texts %q", n, labels)
		}
		mw := &mapWorld{world: ms.World, mission: &missionNotices{table: defs.Table, objectiveLabels: labels}}
		for range 20 {
			sim.Step(ms.World, nil)
		}
		want := []ui.QuestObjective(nil)
		if n == 21 {
			want = []ui.QuestObjective{{Text: labels[1], Mark: 0}}
		}
		if rows := mw.missionObjectives(); !reflect.DeepEqual(rows, want) {
			t.Fatalf("map %d after the first passes shows %v, want %v", n, rows, want)
		}
	}
}

// secondObjectiveOriginalFiles are the corpus's mission 20 files and the
// objective 2 state each holds (R2-SESSION-348).
var secondObjectiveOriginalFiles = []struct {
	name  string
	state int32
	mark  int
}{
	{"game0009.sav", 1, 0},
	{"game0010.sav", 3, 1},
}

// A LOAD of each original mission 20 file restores the file's objective
// slots, the window shows objective 2 with the mark its state selects, and
// our SAVE of that mission carries the same slots back through both LOADs.
func TestReleaseSecondOriginalMissionFileKeepsTheObjectiveSlots(t *testing.T) {
	dir := filepath.Join(secondCompanionCorpus, "rom2-gog-original-2")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no save corpus: %v", err)
	}
	for _, file := range secondObjectiveOriginalFiles {
		t.Run(file.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, file.name))
			if err != nil {
				t.Skipf("no corpus file: %v", err)
			}
			out := t.TempDir()
			if err := os.WriteFile(filepath.Join(out, file.name), raw, 0o600); err != nil {
				t.Fatal(err)
			}
			f := secondGameFront(t)
			app := f.App("second-objectives")
			app.Layout(1024, 768)
			f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
			_, list, _ := f.SaveSeams(SaveStore{Dir: out}, OriginalStore{}, nil)
			groundAppLoad(t, app, list, file.name)
			if f.live == nil || f.liveMission != 20 {
				t.Fatalf("the LOAD opened mission %d, want 20", f.liveMission)
			}
			want := [16]int32{2: file.state}
			if got := liveObjectiveSlots(t, f); got != want {
				t.Fatalf("the LOAD restores objective slots %v, want %v", got, want)
			}
			rows := f.live.missionObjectives()
			if len(rows) != 1 || rows[0].Mark != file.mark || rows[0].Text == "" {
				t.Fatalf("the window shows %v, want objective 2 with mark %d", rows, file.mark)
			}
			ours := secondObjectiveSave(t, f, out, "ours")
			doc := checkSecondMissionFile(t, ours, 20)
			if got := objectiveSlotsOf(doc.SecondTail.Bank); got != want {
				t.Fatalf("our SAVE carries %v, want %v", got, want)
			}
			back := t.TempDir()
			if err := os.WriteFile(filepath.Join(back, "game0002.sav"), withoutEngineDocument(t, ours), 0o600); err != nil {
				t.Fatal(err)
			}
			cold := secondGameFront(t)
			capp := cold.App("second-objectives-back")
			capp.Layout(1024, 768)
			cold.ConfigureSaveSeams(capp, SaveStore{Dir: back}, OriginalStore{}, nil)
			_, clist, _ := cold.SaveSeams(SaveStore{Dir: back}, OriginalStore{}, nil)
			groundAppLoad(t, capp, clist, "game0002.sav")
			if got := liveObjectiveSlots(t, cold); got != want {
				t.Fatalf("the records' LOAD of our SAVE restores %v, want %v", got, want)
			}
		})
	}
}
