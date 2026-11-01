package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The objectives scan: the objectives panel, the outcome text and the
// per-mission text are built by the functions of one file over a game's
// gameFiles.objectives row. No campaign rules type carries a hook of their
// names, no second function builds the panel, and the three functions do not
// branch on the game.
const objectivesHome = "pkg/game/questobjectives.go"

// objectivesHooks are the names a campaign hook once had.
var objectivesHooks = map[string]bool{"objectives": true, "outcomeText": true, "loadMissionText": true}

// objectivesHomeFuncs are the three functions of the home file.
var objectivesHomeFuncs = map[string]bool{"missionObjectives": true, "outcomeText": true, "loadMissionText": true}

// objectivesState are the mission fields only the home file reads.
var objectivesState = map[string]bool{"objectiveLabels": true, "failureText": true}

var objectivesPanelName = regexp.MustCompile(`(?i:objectives?panel)|^missionObjectives$`)

var objectivesGameIdent = map[string]bool{
	"GameOf": true, "GameROM1": true, "GameROM2": true, "secondCampaignRules": true,
	"firstCampaignRules": true, "campaign": true, "campaignOf": true, "tableCampaign": true,
}

// CheckObjectives reports every production shape of a second objectives
// panel or outcome text builder in pkg/game: a method of a campaign rules
// type or of the campaign service named objectives, outcomeText or
// loadMissionText; a function of those names or of a panel producer's name
// outside the home file's one set; a read of the objectives state outside the
// home file; and a game identity read inside the home file's three
// functions. files is keyed by module-relative slash path.
func CheckObjectives(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the objectives scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/game/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		home := name == objectivesHome
		for _, decl := range f.Decls {
			fn, _ := decl.(*ast.FuncDecl)
			owner := ""
			if fn != nil {
				owner = fn.Name.Name
				if fn.Recv != nil && len(fn.Recv.List) == 1 {
					recv := objectivesTypeName(fn.Recv.List[0].Type)
					if strings.HasSuffix(recv, "CampaignRules") && objectivesHooks[owner] {
						vs = append(vs, Violation{From: at(fn), Reason: recv + "." + owner + " is a per-game objectives hook; the game's gameFiles.objectives row gives the one producer its difference"})
					}
				}
				if !home && fn.Recv == nil && objectivesHomeFuncs[owner] {
					vs = append(vs, Violation{From: at(fn), Reason: owner + " is a second objectives builder; " + objectivesHome + " holds the one"})
				}
				if !home && objectivesPanelName.MatchString(owner) {
					vs = append(vs, Violation{From: at(fn), Reason: owner + " is a second objectives panel producer; missionObjectives is the one"})
				}
				if home && objectivesPanelName.MatchString(owner) && owner != "missionObjectives" {
					vs = append(vs, Violation{From: at(fn), Reason: owner + " is a second objectives panel producer beside missionObjectives"})
				}
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.InterfaceType:
					for _, m := range x.Methods.List {
						for _, id := range m.Names {
							if objectivesHooks[id.Name] {
								vs = append(vs, Violation{From: at(id), Reason: "an interface method " + id.Name + " is a per-game objectives hook"})
							}
						}
					}
				case *ast.SelectorExpr:
					if objectivesState[x.Sel.Name] && !home {
						vs = append(vs, Violation{From: at(x), Reason: owner + " reads the mission's " + x.Sel.Name + "; only " + objectivesHome + " builds the panel and outcome text from it"})
					}
				case *ast.Ident:
					if home && objectivesHomeFuncs[owner] && objectivesGameIdent[x.Name] {
						vs = append(vs, Violation{From: at(x), Reason: owner + " reads " + x.Name + "; the game difference is the objectives row"})
					}
				}
				return true
			})
		}
	}
	return vs
}

// objectivesTypeName is the type name of a method receiver.
func objectivesTypeName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.StarExpr:
		return objectivesTypeName(x.X)
	case *ast.Ident:
		return x.Name
	}
	return ""
}
