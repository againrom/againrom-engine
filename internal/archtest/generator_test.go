package archtest

import (
	"os"
	"strings"
	"testing"
)

// No production code of the game or its screens chooses a game's generator
// setup, preset, party or start beside the game's generator row: one
// generatorSetup, generatorParty and generatorBegin read the row for both
// games.
func TestGeneratorIsBuiltFromTheGamesRow(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := GeneratorSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s; put the game's difference in its generator row (gamefiles.go) and read it in chargen.go", s)
	}
}

func generatorHits(name, src string) string {
	return strings.Join(generatorShapesInSource(name, []byte(src)), ", ")
}

// One planted instance per shape: each is named, and a look-alike is not.
func TestGeneratorScanSeesEachShape(t *testing.T) {
	const game, home, rows = "pkg/game/x.go", generatorHome, generatorRowHome
	for _, tc := range []struct{ name, file, src, want string }{
		{"a campaign-rule generator method", game, `package x
func (secondCampaignRules) generatorParty(f *FrontEnd, res ui.ChargenResult) []mapload.PartyMember { return nil }`,
			"pkg/game/x.go:2 method secondCampaignRules.generatorParty is a campaign-rule generator hook"},
		{"a first-game generator method", game, `package x
func (firstCampaignRules) generatorBegin() {}`,
			"pkg/game/x.go:2 method firstCampaignRules.generatorBegin is a campaign-rule generator hook"},
		{"a generator hook in an interface", game, `package x
type campaignService interface {
	generatorSetup(f *FrontEnd, setup *ui.ChargenSetup)
}`, "pkg/game/x.go:3 interface hook generatorSetup chooses the generator per game"},
		{"a second setup builder", game, `package x
func secondGeneratorSetup(f *FrontEnd, setup *ui.ChargenSetup) {}`,
			"pkg/game/x.go:2 func secondGeneratorSetup is a second generator builder"},
		{"a second party builder in the home file", home, `package x
func firstGeneratorParty(f *FrontEnd, res ui.ChargenResult) []mapload.PartyMember { return nil }`,
			"pkg/game/chargen.go:2 func firstGeneratorParty is a second generator builder"},
		{"a builder name on another receiver", home, `package x
func (t *Town) generatorSetup() {}`,
			"pkg/game/chargen.go:2 func generatorSetup is a second generator builder"},
		{"a per-game generator function", game, `package x
func (f *FrontEnd) startSecondGenerated(res ui.ChargenResult) error { return nil }`,
			"pkg/game/x.go:2 func startSecondGenerated is a per-game generator function"},
		{"an archetype read outside the generator", game, `package x
func f(t *mapload.Table) { t.NPC.HeroArchetype(false, false) }`,
			"pkg/game/x.go:2 HeroArchetype reads a preset source outside the generator"},
		{"a template read outside the generator", game, `package x
func f(h data.Collection) { generatorTemplate(h, "Start_MF") }`,
			"pkg/game/x.go:2 generatorTemplate reads a preset source outside the generator"},
		{"a preset written outside the generator", game, `package x
func f(s *ui.ChargenSetup) { s.Presets[0] = []int{1, 2, 3, 4} }`,
			"pkg/game/x.go:2 Presets is written outside the generator"},
		{"a start field set in a literal", "pkg/ui/x.go", `package x
var s = ChargenSetup{AcceptShowsTown: true}`,
			"pkg/ui/x.go:2 AcceptShowsTown is written outside the generator"},
		{"a mission start outside generatorBegin", game, `package x
func g(f *FrontEnd, n int, res ui.ChargenResult) { f.NewGameOpener(n, res) }`,
			"pkg/game/x.go:2 NewGameOpener begins a generator result outside generatorBegin"},
		{"a town start in the home file outside generatorBegin", home, `package x
func (f *FrontEnd) NewGameBegin(n int) error { return f.startGeneratedTown(ui.ChargenResult{}) }`,
			"pkg/game/chargen.go:2 startGeneratedTown begins a generator result outside generatorBegin"},
		{"a Begin closure outside generatorBegin", game, `package x
var b = func(res ui.ChargenResult) (ui.MapOpener, error) { return nil, nil }`,
			"pkg/game/x.go:2 a generator Begin is built outside generatorBegin"},
		{"a party assembled outside the producers", game, `package x
func defaultHero() []mapload.PartyMember { return assembleParty(partyInputs{}) }`,
			"pkg/game/x.go:2 assembleParty builds a party outside the two producers"},
		{"a game named in a generator file", "pkg/game/secondgenerator.go", `package x
func f(p base.Profile) bool { return p.GameOf() == base.GameROM2 }`,
			"pkg/game/secondgenerator.go:2 GameOf names a game in a generator file, pkg/game/secondgenerator.go:2 GameROM2 names a game in a generator file"},
		{"a generator row outside the game files", game, `package x
var r = generatorProfile{templates: true}`,
			"pkg/game/x.go:2 a generator row is written outside the game files"},
		{"a row field set outside the game files", game, `package x
var g = gameFiles{generator: r}`,
			"pkg/game/x.go:2 a generator row is set outside the game files"},
		{"a second table of rows", rows, `package x
var t = map[string]generatorProfile{}`,
			"pkg/game/gamefiles.go:2 a second table of generator rows"},
		{"the one builders and the rows", home, `package x
func (f *FrontEnd) generatorSetup(s *ui.ChargenSetup) { s.Presets[0], s.AcceptShowsTown = nil, f.generatorRow().start == generatorStartTown; f.Table.NPC.HeroArchetype(false, false) }
func (f *FrontEnd) generatorParty(res ui.ChargenResult) []mapload.PartyMember { return assembleParty(partyInputs{}) }
func (f *FrontEnd) generatorBegin(s generatorStart, n int) func(ui.ChargenResult) (ui.MapOpener, error) {
	return func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(n, res), f.startGeneratedTown(res) }
}`, ""},
		{"the game rows and a campaign-part reader", rows, `package x
var firstGameFiles = gameFiles{generator: generatorProfile{documents: true}}
func (f *FrontEnd) secondGeneratorCampaign() (*secondGenerator, error) { return nil, nil }`, ""},
		{"a witness's own begin", "pkg/game/cutscene_witness.go", `package x
var b = func(res ui.ChargenResult) (ui.MapOpener, error) { return nil, nil }`, ""},
	} {
		if got := generatorHits(tc.file, tc.src); got != tc.want {
			t.Errorf("%s: hits = %q, want %q", tc.name, got, tc.want)
		}
	}
}
