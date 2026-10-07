package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestHeadlessConvenienceSaveWritesCurrentSAV(t *testing.T) {
	f, _ := originalCityRouteFixture(t)
	f.Words.SaveAcknowledgement = ui.AuthoredWords().SaveAcknowledgement
	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatal(err)
	}
	app := openLocalTownSAV(t, f, dir, name)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	f.Town.gold = 6789
	if err := runHeadlessStep(f, app, HeadlessStep{Command: "save"}, nil); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 2 {
		t.Fatal("convenience SAVE publication", entries, err)
	}
	var saved string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name, "scenario-save-") {
			saved = entry.Name
		}
		if filepath.Ext(entry.Name) != ".sav" {
			t.Fatal("convenience SAVE wrote a legacy file", entry.Name)
		}
	}
	raw, err := store.Read(saved)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || len(doc.Players) != 1 {
		t.Fatal("current SAV document", err)
	}
	if gold, err := savedStructureValue(&doc.Objects[doc.Players[0]-1], "Money"); err != nil || gold != 6789 {
		t.Fatal("convenience SAVE lost current gold", gold, err)
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatal("convenience SAVE did not return to the menu", app.Screen())
	}
}

func TestHeadlessScenarioValidationRefusesMalformedOrAmbiguousSteps(t *testing.T) {
	valid := HeadlessScenario{Version: 1, Steps: []HeadlessStep{{Command: "capture", Name: "start"}}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid scenario: %v", err)
	}
	cases := []struct {
		name string
		in   HeadlessScenario
		want string
	}{
		// THE UNSUPPORTED VERSION IS DERIVED FROM THE CONSTANT and is not a
		// number written here. A case that spells the next version has to be
		// re-spelled at every bump, and a version test carrying a stale
		// number passes over exactly the version it was meant to refuse.
		{"a version this build does not read",
			HeadlessScenario{Version: HeadlessScenarioVersion + 1, Steps: valid.Steps},
			"this build reads versions"},
		{"empty", HeadlessScenario{Version: 1}, "no steps"},
		{"unknown command", HeadlessScenario{Version: 1, Steps: []HeadlessStep{{Command: "transmogrify"}}}, "unknown command"},
		{"missing key", HeadlessScenario{Version: 1, Steps: []HeadlessStep{{Command: "key"}}}, "missing parameter(s): key"},
		{"missing ticks", HeadlessScenario{Version: 1, Steps: []HeadlessStep{{Command: "wait_ticks"}}}, "missing parameter(s): ticks"},
		{"negative ticks", HeadlessScenario{Version: 1, Steps: []HeadlessStep{{Command: "wait_ticks", Ticks: -1}}}, "ticks must be positive"},
		{"parameter for wrong command", HeadlessScenario{Version: 1, Steps: []HeadlessStep{{Command: "save", ID: "hero"}}}, "unexpected parameter"},
		{"missing member", HeadlessScenario{Version: 1, Steps: []HeadlessStep{{Command: "assert_member"}}}, "missing parameter(s): member"},
		{"member without an id",
			HeadlessScenario{Version: 1, Steps: []HeadlessStep{{Command: "assert_member", Member: &HeadlessMemberAssertion{}}}},
			"member.id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.in.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestReadHeadlessScenarioIsStrictAndResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scenario.json")
	if err := os.WriteFile(path, []byte(`{
  "version": 1,
  "original_saves": "original",
  "saves": "native",
  "steps": [{"command": "capture", "name": "start"}]
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := ReadHeadlessScenario(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.OriginalSaves != filepath.Join(dir, "original") || s.Saves != filepath.Join(dir, "native") {
		t.Fatalf("resolved paths = %q, %q; want below %q", s.OriginalSaves, s.Saves, dir)
	}

	for name, source := range map[string]string{
		"unknown field":  `{"version":1,"surprise":true,"steps":[{"command":"save"}]}`,
		"trailing value": `{"version":1,"steps":[{"command":"save"}]} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			bad := filepath.Join(dir, strings.ReplaceAll(name, " ", "-")+".json")
			if err := os.WriteFile(bad, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadHeadlessScenario(bad); err == nil {
				t.Fatal("ReadHeadlessScenario accepted malformed JSON program")
			}
		})
	}
}

func TestHeadlessStateCanRequireTheExactPartySize(t *testing.T) {
	two := 2
	state := HeadlessState{Members: make([]HeadlessMemberSnapshot, 2)}
	if err := assertHeadlessState(state, HeadlessStateAssertion{MemberCount: &two}, nil); err != nil {
		t.Fatalf("exact member count: %v", err)
	}
	one := 1
	if err := assertHeadlessState(state, HeadlessStateAssertion{MemberCount: &one}, nil); err == nil {
		t.Fatal("wrong member count was accepted")
	}
}

// TestHeadlessAMemberAssertionAnswersForTheCarriedLoadAndTheCapacity is what
// makes scenarios/1025-mission10-weight.json a witness rather than a
// decoration. That file states a load and a capacity for a hero in a real
// campaign mission; if either field were ignored here it would pass against
// a build that computed neither.
func TestHeadlessAMemberAssertionAnswersForTheCarriedLoadAndTheCapacity(t *testing.T) {
	state := HeadlessState{Members: []HeadlessMemberSnapshot{{ID: "hero", Load: 25, Capacity: 201}}}
	want := func(load, capacity int32) HeadlessStateAssertion {
		return HeadlessStateAssertion{Members: []HeadlessMemberAssertion{
			{ID: "hero", Load: &load, Capacity: &capacity},
		}}
	}
	if err := assertHeadlessState(state, want(25, 201), nil); err != nil {
		t.Fatalf("the member's own load and capacity: %v", err)
	}
	if err := assertHeadlessState(state, want(24, 201), nil); err == nil {
		t.Error("a wrong load was accepted")
	}
	if err := assertHeadlessState(state, want(25, 200), nil); err == nil {
		t.Error("a wrong capacity was accepted")
	}

	// A FILE THAT NAMES NEITHER ASSERTS NOTHING ABOUT EITHER, which is why
	// both fields are pointers: zero is a real load for a hero with an empty
	// doll and an empty pack.
	if err := assertHeadlessState(state, HeadlessStateAssertion{
		Members: []HeadlessMemberAssertion{{ID: "hero"}},
	}, nil); err != nil {
		t.Errorf("a member assertion naming neither field: %v", err)
	}
}
