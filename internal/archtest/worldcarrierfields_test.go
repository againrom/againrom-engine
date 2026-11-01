package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"testing"
)

// The carrier rule names a planted second path of each shape the left
// carriers had: a write of the town's carried Diary or of the kept loaded
// document outside the set file, by assignment or by a literal, a nil test of
// the loaded record in a function the allow list does not name, and a direct
// call of a town step, a build step or the identity remap. It also names a
// write of the dead body art by index outside its row's hooks, the loaded
// record passed to a function outside the set, directly or through a local
// of any name, and a nil test through such a local. The set file may write
// and test them; a step, the capture chain's copy and the readers the allow
// list names with a reason may too.
func TestWorldCarrierRuleNamesAPlantedFieldWrite(t *testing.T) {
	const gameHead = `package game
type Snapshot struct{ loadedDocument, SavedDocument *int }
type Town struct{ knowledge []int }
type Mission struct{ DeadArt map[int]uint16 }
type worldCarrier struct{ town func() error }
func projectTownKnowledge() error { return nil }
func retainLoadedDead() error { return nil }
func restoreCurrentActorIdentities() error { return nil }
func graftUnknownObjects(s *Snapshot) error { if s.loadedDocument == nil { return nil }; return nil }
func cloneSavedDocument(*int) {}
func worldCarrierSet() []worldCarrier { return []worldCarrier{{town: projectTownKnowledge}, {town: retainLoadedDead}, {town: restoreCurrentActorIdentities}, {town: func() error { return graftUnknownObjects(nil) }}} }
func retain(*int) {}
`
	const game = "againrom/pkg/game."
	for _, tc := range []struct {
		name, file, source, want string
	}{
		{"every carrier through the set", "pkg/game/game.go", "", ""},
		{"a LOAD that writes the town's carried Diary", "pkg/game/game.go", "func load(t *Town) { t.knowledge = nil }",
			game + "load writes " + game + "Town.knowledge outside the carrier set"},
		{"a town built with its carried Diary", "pkg/game/game.go", "func town() *Town { return &Town{knowledge: nil} }",
			game + "town writes " + game + "Town.knowledge outside the carrier set"},
		{"a writer that keeps the loaded document itself", "pkg/game/game.go", "func keep(s *Snapshot) { s.loadedDocument = s.SavedDocument }",
			game + "keep writes " + game + "Snapshot.loadedDocument outside the carrier set"},
		{"a nil test of the loaded record outside the allow list", "pkg/game/game.go", "func probe(s *Snapshot) bool { return s.SavedDocument == nil }",
			game + "probe tests SavedDocument against nil outside the carrier set"},
		{"the set file writes and tests them", "pkg/game/worldcarriers.go",
			"func keep(s *Snapshot, t *Town) bool { s.loadedDocument, t.knowledge = s.SavedDocument, nil; return s.SavedDocument == nil }", ""},
		{"a city SAVE that writes the town's Diary itself", "pkg/game/game.go", "func city() error { return projectTownKnowledge() }",
			game + "city references " + game + "projectTownKnowledge"},
		{"a builder that retains a dead record itself", "pkg/game/game.go", "func dead() error { return retainLoadedDead() }",
			game + "dead references " + game + "retainLoadedDead"},
		{"a LOAD that remaps the identities a second time", "pkg/game/game.go", "func restoreOriginalActions() error { return restoreCurrentActorIdentities() }",
			game + "restoreOriginalActions references " + game + "restoreCurrentActorIdentities"},
		{"a LOAD that writes the dead body art itself", "pkg/game/game.go", "func art(ms *Mission) { ms.DeadArt[1] = 2 }",
			game + "art writes " + game + "Mission.DeadArt outside the carrier set"},
		{"the dead body row's original LOAD hook writes the art", "pkg/game/game.go", "func applyOriginalDeadWithOrigins(ms *Mission) { ms.DeadArt[1] = 2 }", ""},
		{"a writer that keeps a carrier from the loaded record", "pkg/game/game.go", "func write(s *Snapshot) { retain(s.loadedDocument) }",
			game + "write passes the loaded record to " + game + "retain outside the carrier set"},
		{"a builder that keeps a carrier through a local of another name", "pkg/game/game.go", "func build(s Snapshot) { state := s.SavedDocument; document := state; retain(document) }",
			game + "build passes the loaded record to " + game + "retain outside the carrier set"},
		{"the capture chain copies the loaded record", "pkg/game/game.go", "func capture(s Snapshot) { cloneSavedDocument(s.SavedDocument) }", ""},
		{"a nil test of the loaded record through a local of another name", "pkg/game/game.go", "func probe(s Snapshot) bool { loaded := s.loadedDocument; return loaded == nil }",
			game + "probe tests loaded against nil outside the carrier set"},
		{"a step tests the loaded record (graftUnknownObjects above)", "pkg/game/game.go", "", ""},
		{"a reader the allow list names with a reason", "pkg/game/game.go", "func loadedTavernStockIDs(s *Snapshot) bool { return s.SavedDocument == nil }", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "game.go", gameHead+tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
			if _, err := (&types.Config{}).Check("againrom/pkg/game", fset, []*ast.File{file}, info); err != nil {
				t.Fatal(err)
			}
			got := CheckWorldCarriers([]*CheckedPackage{{ImportPath: "againrom/pkg/game", Files: []*ast.File{file}, RelPaths: []string{tc.file}, NumProd: 1, Info: info}})
			if tc.want == "" && len(got) != 0 || tc.want != "" && !slices.Contains(got, tc.want) {
				t.Fatalf("findings %q, want %q", got, tc.want)
			}
		})
	}
}
