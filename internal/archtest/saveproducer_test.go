package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"testing"
)

func TestCurrentSaveProducerHasNoLegacyDispatch(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	module, err := loadTypeCheckedModule(root)
	if err != nil {
		t.Fatal(err)
	}
	p := module.Packages["againrom/pkg/game"]
	if p == nil {
		t.Fatal("game package absent")
	}
	for _, finding := range CheckSaveProducer(p) {
		t.Error(finding)
	}
}

func TestSaveProducerGuardRejectsDispatchMigrationAndCallbackFallback(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		bad          bool
	}{
		{"non-return alias", `package game; type FrontEnd struct{}; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) ExportOriginalSave() { f.ExportCurrentSave() }`, true},
		{"direct alias", `package game; type FrontEnd struct{}; func (f *FrontEnd) ExportCurrentSave() int { return 1 }; func (f *FrontEnd) ExportOriginalSave() int { return f.ExportCurrentSave() }`, false},
		{"origin selects writer", `package game; type FrontEnd struct { originalCity bool }; func (f *FrontEnd) ExportCurrentSave() int { return 1 }; func (f *FrontEnd) ExportOriginalSave() int { return f.ExportCurrentSave() }; func (f *FrontEnd) save() int { if f.originalCity { return f.ExportOriginalSave() }; return f.ExportCurrentSave() }`, true},
		{"callback AGS", `package game; type FrontEnd struct{}; func EncodeSave() {}; func construct(f func()) { f() }; func (f *FrontEnd) ExportCurrentSave() { construct(EncodeSave) }`, true},
		{"helper AGS", `package game; type FrontEnd struct{}; func EncodeSave() {}; func helper() { EncodeSave() }; func (f *FrontEnd) ExportCurrentSave() { helper() }`, true},
		{"migration", `package game; type FrontEnd struct{}; func CheckSaveForm() {}; func (f *FrontEnd) ExportCurrentSave() { CheckSaveForm() }`, true},
		{"legacy input only", `package game; type FrontEnd struct{}; func CheckSaveForm() {}; func load() { CheckSaveForm() }; func (f *FrontEnd) ExportCurrentSave() {}`, false},
		{"same method on another writer", `package game; type Other struct{}; type FrontEnd struct { old Other }; func (f *FrontEnd) ExportCurrentSave() int { return 1 }; func (f Other) ExportCurrentSave() int { return 2 }; func (f *FrontEnd) ExportOriginalSave() int { return f.old.ExportCurrentSave() }`, true},
		{"replaced current input", `package game; type FrontEnd struct{}; func (f *FrontEnd) ExportCurrentSave(state int) int { return state }; func (f *FrontEnd) ExportOriginalSave(state int) int { return f.ExportCurrentSave(0) }`, true},
		{"private city writer", `package game; type originalCitySaveState struct{}; func (s *originalCitySaveState) marshal() {}; type FrontEnd struct { originalCity *originalCitySaveState }; func (f *FrontEnd) ExportCurrentSave() { f.originalCity.marshal() }`, true},
		{"dialog current and legacy reader", `package game; type FrontEnd struct{}; type Prepared struct{ Commit func() }; type Seams struct{ List func(); Prepare func() Prepared }; func CheckSaveForm() {}; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) SaveDialogSeams() Seams { return Seams{List: CheckSaveForm, Prepare: func() Prepared { f.ExportCurrentSave(); return Prepared{Commit: func(){}} }} }`, false},
		{"dialog AGS branch", `package game; type FrontEnd struct{ ags bool }; type Seams struct{ Prepare func() }; func EncodeSave() {}; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) SaveDialogSeams() Seams { return Seams{Prepare: func() { if f.ags { EncodeSave() } else { f.ExportCurrentSave() } }} }`, true},
		{"dialog helper", `package game; type FrontEnd struct{}; type Seams struct{ Prepare func() }; func EncodeSave() {}; func helper() { EncodeSave() }; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) SaveDialogSeams() Seams { return Seams{Prepare: helper} }`, true},
		{"dialog commit", `package game; type FrontEnd struct{}; type Prepared struct{ Commit func() }; type Seams struct{ Prepare func() Prepared }; func EncodeSave() {}; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) SaveDialogSeams() Seams { return Seams{Prepare: func() Prepared { f.ExportCurrentSave(); return Prepared{Commit: EncodeSave} }} }`, true},
		{"dialog alternate producer", `package game; type FrontEnd struct{}; type Seams struct{ Prepare func() }; func other() {}; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) SaveDialogSeams() Seams { return Seams{Prepare: other} }`, true},
		{"configured preparation", `package game; type FrontEnd struct{}; type Seams struct{ Prepare func() }; func EncodeSave() {}; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) ConfigureSaveSeams() { s:=Seams{}; s.Prepare=EncodeSave }`, true},
		{"configured confirmation", `package game; type FrontEnd struct{}; type Prepared struct{ Commit func() }; func EncodeSave() {}; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) ConfigureSaveSeams() { p:=Prepared{}; p.Commit=func(){ EncodeSave() } }`, true},
		{"configured load", `package game; type FrontEnd struct{}; type Seams struct{ List func() }; func CheckSaveForm() {}; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) ConfigureSaveSeams() { s:=Seams{}; s.List=CheckSaveForm }`, false},
		{"dialog local commit alias", `package game; type FrontEnd struct{}; type Prepared struct{ Commit func() }; type Seams struct{ Prepare func() Prepared }; func EncodeSave() {}; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) SaveDialogSeams() Seams { commit:=EncodeSave; return Seams{Prepare: func() Prepared { f.ExportCurrentSave(); return Prepared{Commit: commit} }} }`, true},
		{"configured local alias", `package game; type FrontEnd struct{}; type Seams struct{ Prepare func() }; func EncodeSave() {}; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) ConfigureSaveSeams() { s:=Seams{}; fallback:=EncodeSave; s.Prepare=func(){ fallback() } }`, true},
		{"dialog producer alias", `package game; type FrontEnd struct{}; type Seams struct{ Prepare func() }; func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) SaveDialogSeams() Seams { prepare:=f.ExportCurrentSave; return Seams{Prepare: prepare} }`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "guard.go", tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
			config := types.Config{}
			if _, err := config.Check("againrom/pkg/game", fset, []*ast.File{file}, info); err != nil {
				t.Fatal(err)
			}
			p := &CheckedPackage{ImportPath: "againrom/pkg/game", Files: []*ast.File{file}, NumProd: 1, Info: info}
			if got := CheckSaveProducer(p); (len(got) != 0) != tc.bad {
				t.Fatalf("findings %v, want bad=%v", got, tc.bad)
			}
		})
	}
}
