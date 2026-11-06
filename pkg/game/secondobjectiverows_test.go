package game

import (
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"golang.org/x/text/encoding/charmap"
)

// objectiveMissionText is a second-game mission text with sub-objective
// tags 0..3 and an empty fourth, its texts encoded as the install holds them.
func objectiveMissionText(t *testing.T, texts []string) []byte {
	t.Helper()
	raw := "#briefing\r\nbriefing\r\n#subobjective0\r\n\r\n"
	for i, s := range texts {
		raw += "#subobjective" + string(rune('1'+i)) + "\r\n" + s + "\r\n"
	}
	raw += "#subobjective4\r\n"
	b, err := charmap.CodePage866.NewEncoder().Bytes([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The objectives window's rows per objective state, in English and Russian
// texts: the texts run from #subobjective0, a line break alone, to the first
// empty text (R2-ENGINE-890); a row shows while its slot is nonzero, with
// frame 1 for bit 2, else frame 2 for bit 4, else frame 0 for bit 1, else no
// mark (R2-ENGINE-888, R2-ENGINE-889). Reading the rows writes no slot.
func TestSecondGameObjectiveRowsPerState(t *testing.T) {
	marks := []struct{ state, mark int32 }{
		{1, 0}, {2, 1}, {3, 1}, {4, 2}, {5, 2}, {6, 1}, {7, 1}, {8, ui.NoObjectiveMark}, {16, ui.NoObjectiveMark}, {-1, 1},
	}
	for _, lang := range []struct {
		name  string
		texts []string
	}{
		{"english", []string{"Find the sack.", "Reach the elder.", "Third."}},
		{"russian", []string{"Найти мешок.", "Дойти до старейшины.", "Третье."}},
	} {
		t.Run(lang.name, func(t *testing.T) {
			src := missionSource{"main/text/mission21.txt": objectiveMissionText(t, lang.texts)}
			labels := secondGameObjectiveLabels(src, TextCode{}, 21)
			if want := append([]string{""}, lang.texts...); !reflect.DeepEqual(labels, want) {
				t.Fatalf("the cached texts are %q, want %q", labels, want)
			}
			w := secondGameScriptWorld(t, []sim.ScriptInstant{{Op: 35, Args: [10]int32{900, 1}}},
				[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
			mw := &mapWorld{world: w, mission: &missionNotices{
				table: readUnder(base.GameROM2, &mapload.Table{}), objectiveLabels: labels,
			}}
			rows := func(states map[int]int32) []ui.QuestObjective {
				t.Helper()
				bank, ok := w.ROM2ScenarioState()
				if !ok {
					t.Fatal("no scenario bank")
				}
				sim.EnterLocation(&bank)
				for slot, v := range states {
					bank[slot] = v
				}
				w.SetROM2ScenarioState(bank)
				got := gameMenuContext(mw, true, "briefing").Objectives
				if after, _ := w.ROM2ScenarioState(); after != bank {
					t.Fatal("reading the rows wrote the bank")
				}
				return got
			}
			if got := rows(nil); got != nil {
				t.Fatalf("no opened objective shows %v", got)
			}
			for _, m := range marks {
				got := rows(map[int]int32{754: m.state})
				if want := []ui.QuestObjective{{Text: lang.texts[1], Mark: int(m.mark)}}; !reflect.DeepEqual(got, want) {
					t.Fatalf("state %d shows %v, want %v", m.state, got, want)
				}
			}
			got := rows(map[int]int32{752: 1, 753: 3, 755: 5, 756: 1})
			want := []ui.QuestObjective{{Text: "", Mark: 0}, {Text: lang.texts[0], Mark: 1}, {Text: lang.texts[2], Mark: 2}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("the rows in cache order are %v, want %v (slot 756 has no text)", got, want)
			}
		})
	}
}
