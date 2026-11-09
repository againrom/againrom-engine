package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The case-fold scan. A resource or host name folds through a pkg/namefold
// policy. Production code under pkg and cmd, outside pkg/namefold, names no
// standard case function at all: a call, or a function taken as a value, of
// strings, bytes or unicode ToLower, ToUpper, ToTitle, EqualFold, SimpleFold,
// To, ToLowerSpecial, ToUpperSpecial or ToTitleSpecial; a strings.Map or
// bytes.Map, the shape of a fold table; a dot import of strings, bytes or
// unicode, which hides the package name; or an import of the x/text case
// package. The exceptions are the functions of CaseFoldAllowed. Tests are not
// scanned. The hand loop over 'A' to 'Z' is HandFoldSites.

// CaseFoldAllowed are the production functions that fold case of something
// that is not a resource or host name, each with its reason. A key is
// file#function, or file#Type.method for a method; a function literal counts
// with the declaration that holds it.
var CaseFoldAllowed = map[string]string{
	"cmd/againrom/main.go#skillNames":                                         "lists skill names typed on the command line, lower case",
	"cmd/againrom/main.go#skillSlot":                                          "matches a skill name typed on the command line",
	"cmd/buttonframecheck/main.go#driveRooms":                                 "matches a dialog row's text in a witness",
	"cmd/divcensus/main.go#main":                                              "reads a ledger row's status text",
	"cmd/divcensus/main.go#visible":                                           "reads a ledger row's subsystem text",
	"pkg/editorapp/run.go#runAs":                                              "matches a record kind typed on the command line",
	"cmd/savtool/document.go#sameLeaves":                                      "folds the record paths of a save's state tree, which are not file names",
	"cmd/schoolcheck/main.go#driveSchoolRoom":                                 "matches a dialog row's text in a witness",
	"cmd/screenshot/town.go#captureSelectedTavern":                            "matches a button label in a witness",
	"cmd/starter/app.go#app.layout":                                           "compares ini keys of the starter's bases",
	"cmd/starter/settings.go#settings.basePath":                               "compares ini keys of the starter's bases",
	"cmd/starter/settings.go#settings.removeBase":                             "compares ini keys of the starter's bases",
	"cmd/starter/settings.go#settings.store":                                  "compares ini keys and sections of the starter's settings file",
	"cmd/townsquarecheck/main.go#driveDoor":                                   "matches a screen header text in a witness",
	"pkg/data/npcface.go#npcFlags":                                            "matches a data token of the NPC face table",
	"pkg/data/npcface.go#npcTokens":                                           "matches a data token of the NPC face table",
	"pkg/data/spell.go#parseSpellEffect":                                      "matches data tokens of the spell table",
	"pkg/game/campaign.go#ReadCampaign":                                       "matches a section name of the campaign file",
	"pkg/game/campaign.go#childNamed":                                         "matches a key name of the campaign file",
	"pkg/game/campaign.go#sectionMission":                                     "matches a section name of the campaign file",
	"pkg/game/chatcommands.go#mapWorld.runChatCommand":                        "matches the typed name of a party value",
	"pkg/game/headless.go#HeadlessScenario.StageName":                         "reads a stage token of a scenario script",
	"pkg/game/headless.go#HeadlessStep.validate":                              "reads command tokens of a scenario script; the frame file extension folds through namefold",
	"pkg/game/headless.go#HeadlessUntil.frontEndHolds":                        "reads screen and command tokens of a scenario script",
	"pkg/game/headless.go#RunHeadlessScenario":                                "reads a command token of a scenario script",
	"pkg/game/headless.go#assertHeadlessMember":                               "matches a skill name of a scenario assertion",
	"pkg/game/headless.go#assertHeadlessState":                                "matches a screen token of a scenario assertion",
	"pkg/game/headless.go#headlessScreenNamed":                                "matches a screen token of a scenario script",
	"pkg/game/headless.go#runHeadlessStep":                                    "reads command tokens of a scenario script",
	"pkg/game/headlesschargen.go#headlessChargenPicture":                      "matches a control label of a scenario script",
	"pkg/game/headlesschargen.go#headlessChargenStats":                        "matches control labels of a scenario script",
	"pkg/game/headlessframe.go#FrontEnd.checkHeadlessFramePaths":              "reads the frame command token; the frame file paths fold through namefold",
	"pkg/game/headlesskeys.go#fieldType":                                      "matches a JSON field name of a scenario script",
	"pkg/game/headlesspointer.go#HeadlessPoint.resolveBase":                   "reads a base token of a scenario pointer step",
	"pkg/game/headlesspointer.go#HeadlessPoint.validate":                      "reads surface tokens of a scenario pointer step",
	"pkg/game/headlesspointer.go#headlessPointer":                             "reads an action token of a scenario pointer step",
	"pkg/game/installtext.go#secondGameTextSection":                           "matches a section key inside a text payload",
	"pkg/game/mapcheck.go#mapCheck.scriptReference":                           "report text",
	"pkg/game/mapinspection.go#MapInspector.Editor":                           "orders map titles of an inspection list",
	"pkg/game/mapinspectionadd.go#mapInspectionEdits.AddRecord":               "error text",
	"pkg/game/mapinspectionadd.go#newRecordKindOf":                            "error text",
	"pkg/game/mapinspectionedit.go#editKind":                                  "error text",
	"pkg/game/mapinspectionedit.go#mapInspectionEdits.setField":               "error text",
	"pkg/game/mapinspectionscript.go#goalListing":                             "report text",
	"pkg/game/mapinspectionselection.go#selectionKind":                        "error text",
	"pkg/game/mapinspectiontriggers.go#inspectionScript.node":                 "report text",
	"pkg/game/mapinspectionworkspace.go#mapInspectionEdits.SetWorkspaceField": "matches a field name typed in the editor",
	"pkg/game/music_preferences.go#FrontEnd.wireMusicPreferences":             "looks a music title up by its lower-case key",
	"pkg/game/music_preferences.go#musicTitles":                               "keys music titles by lower-case name",
	"pkg/game/savactiondecode.go#checkCurrentWriterCarriers":                  "matches a record name inside a save",
	"pkg/game/savepages.go#secondSavePages":                                   "upper-cases a dialog hint text",
	"pkg/game/savloadprerequisites.go#reservedParticipantName":                "matches the reserved player names of a map",
	"pkg/game/scenario.go#HeadlessScenario.AssetSource":                       "reads an asset source token of a scenario script",
	"pkg/game/scenario.go#HeadlessStep.commands":                              "reads command tokens of a scenario script",
	"pkg/game/scenario.go#HeadlessStep.validateOrder":                         "reads an order token of a scenario script",
	"pkg/game/scenario.go#HeadlessWorldSpec.Build":                            "reads tokens of a scenario world spec",
	"pkg/game/scenario.go#HeadlessWorldSpec.validate":                         "reads tokens of a scenario world spec",
	"pkg/game/scenario.go#RunPlayScenario":                                    "reads a command token of a scenario script",
	"pkg/game/scenario.go#headlessDifficulty":                                 "reads a difficulty token of a scenario script",
	"pkg/game/scenario.go#headlessOutcomeWanted":                              "reads an outcome token of a scenario script",
	"pkg/game/scenario.go#runPlayStep":                                        "reads command tokens of a scenario script",
	"pkg/ini/ini.go#File.Keys":                                                "ini keys are case-insensitive",
	"pkg/ini/ini.go#File.Sections":                                            "ini section names are case-insensitive",
	"pkg/ini/ini.go#File.find":                                                "ini section names are case-insensitive",
	"pkg/ini/ini.go#File.span":                                                "ini keys are case-insensitive",
	"pkg/launch/settings.go#ParseBool":                                        "reads an ini value",
	"pkg/launch/settings.go#Setting.Load":                                     "reads an ini value",
	"pkg/launch/starterini.go#SelectedBase":                                   "compares ini keys of the starter's bases",
	"pkg/launch/starterini.go#SelectedKey":                                    "compares ini keys of the starter's bases",
	"pkg/mapload/itemeffects.go#ParseItemCell":                                "matches a data token of the item effect table",
	"pkg/mapload/party.go#stablePartyName":                                    "derives a party member key from a name",
	"pkg/mod/settings.go#Setting.Parse":                                       "reads a settings value",
	"pkg/ui/ending.go#App.headlessEnding":                                     "matches a label of a scenario script",
	"pkg/ui/gameoptions.go#gameOptionNamed":                                   "matches an option name of a scenario script",
	"pkg/ui/headless.go#App.HeadlessActivate":                                 "matches targets and labels of a scenario script",
	"pkg/ui/headless.go#App.HeadlessGameMenuAction":                           "matches an action token of a scenario script",
	"pkg/ui/headless.go#App.HeadlessKey":                                      "matches a key name of a scenario script",
	"pkg/ui/headless.go#App.headlessActivateTownSurface":                      "matches labels of a scenario script",
	"pkg/ui/headless.go#App.headlessMenuButton":                               "matches labels of a scenario script",
	"pkg/ui/headless.go#headlessWorldMapMissionNumber":                        "matches a target token of a scenario script",
	"pkg/ui/headless.go#headlessWorldMapRecord":                               "matches a target token of a scenario script",
	"pkg/ui/headless.go#matchPickerRow":                                       "matches a row text of a scenario script",
	"pkg/ui/headlesspointer.go#App.HeadlessDocumentPoint":                     "matches a control token of a scenario pointer step",
	"pkg/ui/mapeditorsidepanels.go#MapEditor.treeKey":                         "matches a typed search text of the editor's tree",
	"pkg/ui/mapeditorsidepanels.go#MapEditor.treeMatches":                     "matches a typed search text of the editor's tree and its bar search field",
	"pkg/ui/media_menu.go#App.headlessCutscene":                               "matches a cutscene title of a scenario script",
	"pkg/ui/musicdescription.go#MusicDescription.Known":                       "matches a music track name of the music manifest",
	"pkg/ui/panel.go#panelLabel":                                              "upper-cases a skill name for display",
	"pkg/ui/picker.go#Picker.Find":                                            "matches a typed search text",
	"pkg/ui/save_dialog_headless.go#App.HeadlessSaveAction":                   "matches an action token of a scenario script",
	"pkg/ui/save_dialog_headless.go#App.HeadlessSaveEdit":                     "matches a format token of a scenario script",
}

// CaseFoldSite is one use of a standard case function.
type CaseFoldSite struct {
	File string // slash path under the module root
	Func string // declaration holding the use
	Line int
	Use  string // package.name
}

// Key is the CaseFoldAllowed key of the site.
func (s CaseFoldSite) Key() string { return s.File + "#" + s.Func }

var caseFoldPackages = map[string]bool{"strings": true, "bytes": true, "unicode": true}

var caseFoldNames = map[string]bool{
	"ToLower": true, "ToUpper": true, "ToTitle": true, "EqualFold": true, "SimpleFold": true,
	"To": true, "ToLowerSpecial": true, "ToUpperSpecial": true, "ToTitleSpecial": true,
	"Map": true,
}

const caseFoldXText = "golang.org/x/text/cases"

// CaseFoldUnallowed are the sites that no entry of allowed names: a site is
// allowed only in its own file and function.
func CaseFoldUnallowed(sites []CaseFoldSite, allowed map[string]string) []CaseFoldSite {
	var out []CaseFoldSite
	for _, s := range sites {
		if _, ok := allowed[s.Key()]; !ok {
			out = append(out, s)
		}
	}
	return out
}

// CaseFoldStale are the keys of allowed that no site uses.
func CaseFoldStale(sites []CaseFoldSite, allowed map[string]string) []string {
	used := map[string]bool{}
	for _, s := range sites {
		used[s.Key()] = true
	}
	var out []string
	for k := range allowed {
		if !used[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// CaseFoldSites lists every use of a standard case function in the production
// code of pkg and cmd, outside pkg/namefold, allowed or not.
func CaseFoldSites(root string) ([]CaseFoldSite, error) {
	var sites []CaseFoldSite
	for _, top := range []string{"pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				return rerr
			}
			rel = filepath.ToSlash(rel)
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || strings.HasPrefix(rel, "pkg/namefold/") {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			sites = append(sites, caseFoldInSource(rel, src)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].File != sites[j].File {
			return sites[i].File < sites[j].File
		}
		return sites[i].Line < sites[j].Line
	})
	return sites, nil
}

// caseFoldInSource lists every use in one file, allowed or not.
func caseFoldInSource(name string, src []byte) []CaseFoldSite {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	local := map[string]string{} // file-local package name -> standard package
	var sites []CaseFoldSite
	for _, imp := range file.Imports {
		p, uerr := strconv.Unquote(imp.Path.Value)
		if uerr != nil {
			continue
		}
		switch {
		case p == caseFoldXText:
			sites = append(sites, CaseFoldSite{File: name, Func: "(imports)", Line: fset.Position(imp.Pos()).Line, Use: p})
		case caseFoldPackages[p]:
			n := p
			if imp.Name != nil {
				n = imp.Name.Name
			}
			if n == "." {
				sites = append(sites, CaseFoldSite{File: name, Func: "(imports)", Line: fset.Position(imp.Pos()).Line, Use: "dot import of " + p})
				continue
			}
			local[n] = p
		}
	}
	scan := func(holder string, root ast.Node) {
		ast.Inspect(root, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			id, ok := sel.X.(*ast.Ident)
			if !ok || id.Obj != nil {
				return true
			}
			pkg, ok := local[id.Name]
			if !ok || !caseFoldNames[sel.Sel.Name] {
				return true
			}
			if sel.Sel.Name == "Map" && pkg == "unicode" || sel.Sel.Name == "To" && pkg != "unicode" {
				return true
			}
			sites = append(sites, CaseFoldSite{File: name, Func: holder, Line: fset.Position(sel.Pos()).Line, Use: pkg + "." + sel.Sel.Name})
			return true
		})
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			scan(funcHolder(d), d)
		case *ast.GenDecl:
			scan("(package)", d)
		}
	}
	return sites
}

// funcHolder names a declaration as CaseFoldAllowed keys it.
func funcHolder(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	t := fn.Recv.List[0].Type
	for {
		switch x := t.(type) {
		case *ast.StarExpr:
			t = x.X
			continue
		case *ast.IndexExpr:
			t = x.X
			continue
		case *ast.IndexListExpr:
			t = x.X
			continue
		case *ast.Ident:
			return x.Name + "." + fn.Name.Name
		}
		return "?." + fn.Name.Name
	}
}
