package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
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
	sav := module.Packages[savPackagePath]
	if sav == nil {
		t.Fatal("SAV package absent")
	}
	for _, finding := range CheckSaveByteProducers(sav) {
		t.Error(finding)
	}
	var packages []*CheckedPackage
	for _, pkg := range module.Packages {
		packages = append(packages, pkg)
	}
	for _, finding := range CheckSaveByteCallers(packages) {
		t.Error(finding)
	}
}

func TestSaveProducerGuardRejectsUncalledSAVWriter(t *testing.T) {
	fset := token.NewFileSet()
	savFile, err := parser.ParseFile(fset, "sav.go", `package sav; type DocumentData struct{}; type CityUpdate struct{}; type CityProvenance struct{}; type File struct{}; func EncodeDocumentData(DocumentData) ([]byte, error) { return nil, nil }; func (*CityProvenance) Marshal(CityUpdate) ([]byte, error) { return nil, nil }; func (*File) Marshal() []byte { return nil }; func ReserveDocumentKeys(DocumentData, int) ([]uint32, error) { return nil, nil }; func ZZReviewWrite(d DocumentData) ([]byte, error) { return EncodeDocumentData(d) }`, 0)
	if err != nil {
		t.Fatal(err)
	}
	savPkg, err := (&types.Config{}).Check("againrom/pkg/formats/sav", fset, []*ast.File{savFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, source string
		bad          bool
	}{
		{"current producer", `func (f *FrontEnd) ExportCurrentSave() ([]byte, error) { return s.EncodeDocumentData(s.DocumentData{}) }`, false},
		{"equipment load re-encoding", `func (f *FrontEnd) ExportCurrentSave() {}; func repairLoadedEquipmentRows() ([]byte, error) { return s.EncodeDocumentData(s.DocumentData{}) }`, false},
		{"mod load re-encoding", `func (f *FrontEnd) ExportCurrentSave() {}; func applyModMark() ([]byte, error) { return s.EncodeDocumentData(s.DocumentData{}) }`, false},
		{"uncalled town writer", `func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) exportMissionCity(p *s.CityProvenance) ([]byte, error) { return p.Marshal(s.CityUpdate{}) }`, true},
		{"renamed town writer", `func (f *FrontEnd) ExportCurrentSave() {}; func writeTown(p *s.CityProvenance) ([]byte, error) { return p.Marshal(s.CityUpdate{}) }`, true},
		{"uncalled document writer", `func (f *FrontEnd) ExportCurrentSave() {}; func writeDocument() ([]byte, error) { return s.EncodeDocumentData(s.DocumentData{}) }`, true},
		{"uncalled file writer", `func (f *FrontEnd) ExportCurrentSave() {}; func writeFile(p *s.File) []byte { return p.Marshal() }`, true},
		{"callback town writer", `func (f *FrontEnd) ExportCurrentSave() {}; func writeTown(p *s.CityProvenance) func(s.CityUpdate) ([]byte, error) { return p.Marshal }`, true},
		{"package encoder alias", `var writeDocument = s.EncodeDocumentData; func (f *FrontEnd) ExportCurrentSave() {}`, true},
		{"city interface writer", `type originalCityDocument interface { Marshal(s.CityUpdate) ([]byte, error) }; func (f *FrontEnd) ExportCurrentSave() {}; func writeTown(p originalCityDocument) ([]byte, error) { return p.Marshal(s.CityUpdate{}) }`, true},
		{"review helper in a new file", `func (f *FrontEnd) ExportCurrentSave() {}; func zzReviewHelper() ([]byte, error) { return s.EncodeDocumentData(s.DocumentData{}) }; func (f *FrontEnd) zzReviewSave() { zzReviewHelper() }`, true},
		{"review wrapper in the SAV package", `func (f *FrontEnd) ExportCurrentSave() {}; func (f *FrontEnd) zzReviewSave() ([]byte, error) { return s.ZZReviewWrite(s.DocumentData{}) }`, true},
		{"listed key reader", `func (f *FrontEnd) ExportCurrentSave() {}; func keys() ([]uint32, error) { return s.ReserveDocumentKeys(s.DocumentData{}, 1) }`, false},
		{"load name on another receiver", `type Other struct{}; func (f *FrontEnd) ExportCurrentSave() {}; func (Other) applyModMark() ([]byte, error) { return s.EncodeDocumentData(s.DocumentData{}) }`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(fset, "game.go", `package game; import s "againrom/pkg/formats/sav"; type FrontEnd struct{}; var _ s.DocumentData; `+tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
			config := types.Config{Importer: fixedImporter{path: "againrom/pkg/formats/sav", pkg: savPkg}}
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

const syntheticSAV = `package sav
type DocumentData struct{}; type DocumentStateData struct{}; type CityUpdate struct{}; type CityProvenance struct{}; type File struct{}; type DocPayload struct{}; type DocumentFragment struct{}
func EncodeDocumentData(DocumentData) ([]byte, error) { return nil, nil }
func (*File) Marshal() []byte { return nil }
func (*CityProvenance) Marshal(CityUpdate) ([]byte, error) { return nil, nil }
func Compress([]byte) []byte { return nil }
func EncodeDocPayload(*DocPayload) ([]uint32, error) { return nil, nil }
func Decompress([]byte) ([]byte, error) { return nil, nil }
func (*File) CellRecord(int) ([]byte, error) { return nil, nil }
func NativeActions(DocumentStateData) ([]byte, bool, error) { return nil, false, nil }
func NativeMods(DocumentStateData) ([]byte, bool, error) { return nil, false, nil }
func ReserveDocumentKeys(DocumentData, int) ([]uint32, error) { return nil, nil }
func (DocumentFragment) MarshalJSON() ([]byte, error) { return nil, nil }
func Open([]byte) (*File, error) { return nil, nil }
func ZZReviewWrite(d DocumentData) ([]byte, error) { return EncodeDocumentData(d) }
`

func checkSyntheticSAV(t *testing.T, fset *token.FileSet, source string) (*CheckedPackage, *types.Package) {
	t.Helper()
	file, err := parser.ParseFile(fset, "sav.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	pkg, err := (&types.Config{}).Check(savPackagePath, fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	return &CheckedPackage{ImportPath: savPackagePath, Files: []*ast.File{file}, NumProd: 1, Info: info}, pkg
}

func TestSaveByteProducerListRejectsUnlistedProducer(t *testing.T) {
	listed := strings.Replace(syntheticSAV, "func ZZReviewWrite(d DocumentData) ([]byte, error) { return EncodeDocumentData(d) }\n", "", 1)
	for _, tc := range []struct {
		name, source string
		bad          bool
	}{
		{"listed producers only", listed, false},
		{"review wrapper", syntheticSAV, true},
		{"uncalled byte producer", listed + "func ReviewEncode(DocumentData) []byte { return nil }\n", true},
		{"uncalled writer parameter", listed + "type Sink interface{ Write([]byte) (int, error) }; func ReviewWrite(Sink, DocumentData) error { return nil }\n", true},
		{"named byte result", listed + "type Blob []byte; func ReviewBlob() Blob { return nil }\n", true},
		{"uncalled producer method", listed + "func (*File) ReviewBytes() []uint32 { return nil }\n", true},
		{"unexported helper", listed + "func reviewEncode() []byte { return nil }\n", false},
		{"listed producer absent", strings.Replace(listed, "func Compress([]byte) []byte { return nil }\n", "", 1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, _ := checkSyntheticSAV(t, token.NewFileSet(), tc.source)
			if got := CheckSaveByteProducers(p); (len(got) != 0) != tc.bad {
				t.Fatalf("findings %v, want bad=%v", got, tc.bad)
			}
		})
	}
}

func TestSaveByteCallersRejectsNewCaller(t *testing.T) {
	const savtool = `package main; import s "againrom/pkg/formats/sav"; func verify(f *s.File) []byte { return f.Marshal() }; func set(f *s.File) []byte { return f.Marshal() }; func info(f *s.File) ([]byte, error) { return f.CellRecord(0) }; func main() {}`
	const fixture = `package cityfixture; import s "againrom/pkg/formats/sav"; func Original(p *s.CityProvenance) ([]byte, error) { return p.Marshal(s.CityUpdate{}) }`
	for _, tc := range []struct {
		name, savtool, other string
		bad                  bool
	}{
		{"listed callers and a reader", savtool, `func read(b []byte) ([]byte, error) { return s.Decompress(b) }`, false},
		{"new document caller", savtool, `func writeSave() ([]byte, error) { return s.EncodeDocumentData(s.DocumentData{}) }`, true},
		{"new part caller", savtool, `func pack() []byte { return s.Compress(nil) }`, true},
		{"method value at package scope", savtool, `var write = (*s.File).Marshal`, true},
		{"unlisted producer caller", savtool, `func writeSave() ([]byte, error) { return s.ZZReviewWrite(s.DocumentData{}) }`, true},
		{"listed package, other function", strings.Replace(savtool, "func info(f *s.File) ([]byte, error) { return f.CellRecord(0) }", "func info(f *s.File) []byte { return f.Marshal() }", 1), `var _ s.File`, true},
		{"stale listed caller", strings.Replace(savtool, "func set(f *s.File) []byte { return f.Marshal() }", "func set(f *s.File) {}", 1), `var _ s.File`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			_, savPkg := checkSyntheticSAV(t, fset, syntheticSAV)
			var packages []*CheckedPackage
			for _, src := range []struct{ path, source string }{
				{"againrom/cmd/savtool", tc.savtool},
				{"againrom/internal/cityfixture", fixture},
				{"againrom/cmd/other", `package main; import s "againrom/pkg/formats/sav"; ` + tc.other + `; func main() {}`},
			} {
				file, err := parser.ParseFile(fset, "src.go", src.source, 0)
				if err != nil {
					t.Fatal(err)
				}
				info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
				config := types.Config{Importer: fixedImporter{path: savPackagePath, pkg: savPkg}}
				if _, err := config.Check(src.path, fset, []*ast.File{file}, info); err != nil {
					t.Fatal(err)
				}
				packages = append(packages, &CheckedPackage{ImportPath: src.path, Files: []*ast.File{file}, NumProd: 1, Info: info})
			}
			if got := CheckSaveByteCallers(packages); (len(got) != 0) != tc.bad {
				t.Fatalf("findings %v, want bad=%v", got, tc.bad)
			}
		})
	}
}
