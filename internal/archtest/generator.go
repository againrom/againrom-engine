package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// generatorHome builds the hero generator's setup, presets, party and start
// for both games from the game's generator row; generatorRowHome holds the
// rows.
const (
	generatorHome    = "pkg/game/chargen.go"
	generatorRowHome = "pkg/game/gamefiles.go"
)

// generatorBuilders are the one setup, preset, party and start builders.
var generatorBuilders = map[string]bool{
	"generatorSetup": true, "generatorPreset": true, "generatorParty": true, "generatorBegin": true,
}

var (
	generatorBuilderName = regexp.MustCompile(`[gG]enerator(Setup|Party|Presets?|Begin)$`)
	perGameGeneratorName = regexp.MustCompile(`(?i)(first|second).*(generat|chargen)`)
)

// generatorCampaignReaders read the description's campaign part, which only
// the second game's description carries; they choose nothing.
var generatorCampaignReaders = map[string]bool{"secondGeneratorCampaign": true, "decodeSecondGenerator": true}

// generatorPresetReads are the reads of a picture's preset source.
var generatorPresetReads = map[string]bool{"HeroArchetype": true, "generatorTemplate": true, "heroTemplate": true}

// generatorSetupFields are the setup's preset and start fields.
var generatorSetupFields = map[string]bool{"Presets": true, "PresetSkills": true, "AcceptShowsTown": true}

// generatorStarts are what an accepted result begins.
var generatorStarts = map[string]bool{"NewGameOpener": true, "startGeneratedTown": true}

// generatorFiles are the files of the generator; none names a game.
var generatorFiles = map[string]bool{
	generatorHome: true, "pkg/game/secondgenerator.go": true, "pkg/game/generator.go": true,
}

// generatorGameNames name a game or its campaign rules.
var generatorGameNames = map[string]bool{
	"firstCampaignRules": true, "secondCampaignRules": true, "firstGameFiles": true, "secondGameFiles": true,
	"CampaignDestinations": true, "CampaignChapters": true, "GameROM1": true, "GameROM2": true, "GameOf": true,
}

// generatorPartyProducers are the two producers that assemble a party: the
// default party and the generated one.
var generatorPartyProducers = map[string]bool{"MissionPartyAs": true, "generatorParty": true}

// generatorWitnessBegins open a witness's own mission with the generated
// party; they choose no game's start.
var generatorWitnessBegins = map[string]bool{"pkg/game/cutscene_witness.go": true}

// GeneratorSites lists the places in production code of pkg/game and pkg/ui
// that choose a game's generator setup, preset, party or start beside the
// game's generator row and its one builder of each. Tests are not scanned.
func GeneratorSites(root string) ([]string, error) {
	var sites []string
	for _, top := range []string{"pkg/game", "pkg/ui"} {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(top)), func(path string, d fs.DirEntry, err error) error {
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
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			sites = append(sites, generatorShapesInSource(rel, src)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

func generatorShapesInSource(name string, src []byte) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	inGame := strings.HasPrefix(name, "pkg/game/")
	seen := map[string]bool{}
	var hits []string
	add := func(pos token.Pos, what string) {
		site := name + ":" + strconv.Itoa(fset.Position(pos).Line) + " " + what
		if !seen[site] {
			seen[site] = true
			hits = append(hits, site)
		}
	}
	inspect := func(fn string, n ast.Node) {
		ast.Inspect(n, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				callee := calleeName(n)
				switch {
				case generatorPresetReads[callee] && name != generatorHome:
					add(n.Pos(), callee+" reads a preset source outside the generator")
				case generatorStarts[callee] && !(name == generatorHome && fn == "generatorBegin"):
					add(n.Pos(), callee+" begins a generator result outside generatorBegin")
				case inGame && callee == "assembleParty" && !generatorPartyProducers[fn]:
					add(n.Pos(), "assembleParty builds a party outside the two producers")
				}
			case *ast.AssignStmt:
				if name == generatorHome {
					return true
				}
				for _, lhs := range n.Lhs {
					if ix, ok := lhs.(*ast.IndexExpr); ok {
						lhs = ix.X
					}
					if sel, ok := lhs.(*ast.SelectorExpr); ok && generatorSetupFields[sel.Sel.Name] {
						add(lhs.Pos(), sel.Sel.Name+" is written outside the generator")
					}
				}
			case *ast.KeyValueExpr:
				if id, ok := n.Key.(*ast.Ident); ok && generatorSetupFields[id.Name] && name != generatorHome {
					add(n.Pos(), id.Name+" is written outside the generator")
				}
				if id, ok := n.Key.(*ast.Ident); ok && id.Name == "generator" && inGame && name != generatorRowHome {
					add(n.Pos(), "a generator row is set outside the game files")
				}
			case *ast.FuncLit:
				if inGame && beginSignature(n.Type) && !(name == generatorHome && fn == "generatorBegin") && !generatorWitnessBegins[name] {
					add(n.Pos(), "a generator Begin is built outside generatorBegin")
				}
			case *ast.CompositeLit:
				if id, ok := n.Type.(*ast.Ident); ok && id.Name == "generatorProfile" && name != generatorRowHome {
					add(n.Pos(), "a generator row is written outside the game files")
				}
			case *ast.MapType:
				if id, ok := n.Value.(*ast.Ident); ok && id.Name == "generatorProfile" {
					add(n.Pos(), "a second table of generator rows")
				}
			case *ast.ArrayType:
				if id, ok := n.Elt.(*ast.Ident); ok && id.Name == "generatorProfile" {
					add(n.Pos(), "a second table of generator rows")
				}
			case *ast.InterfaceType:
				for _, m := range n.Methods.List {
					for _, mn := range m.Names {
						if strings.HasPrefix(mn.Name, "generator") {
							add(mn.Pos(), "interface hook "+mn.Name+" chooses the generator per game")
						}
					}
				}
			case *ast.Ident:
				if generatorFiles[name] && generatorGameNames[n.Name] {
					add(n.Pos(), n.Name+" names a game in a generator file")
				}
			}
			return true
		})
	}
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			inspect("", decl)
			continue
		}
		recv, fn := receiverName(fd.Recv), fd.Name.Name
		switch {
		case strings.HasSuffix(recv, "CampaignRules") && strings.HasPrefix(fn, "generator"):
			add(fd.Pos(), "method "+recv+"."+fn+" is a campaign-rule generator hook")
		case generatorBuilderName.MatchString(fn) && !(name == generatorHome && recv == "FrontEnd" && generatorBuilders[fn]):
			add(fd.Pos(), "func "+fn+" is a second generator builder")
		case perGameGeneratorName.MatchString(fn) && !generatorCampaignReaders[fn]:
			add(fd.Pos(), "func "+fn+" is a per-game generator function")
		}
		inspect(fn, fd)
	}
	return hits
}

// beginSignature reports whether t is func(ui.ChargenResult) (ui.MapOpener, error).
func beginSignature(t *ast.FuncType) bool {
	if t.Params == nil || len(t.Params.List) != 1 || t.Results == nil || len(t.Results.List) != 2 {
		return false
	}
	named := func(e ast.Expr, want string) bool {
		if sel, ok := e.(*ast.SelectorExpr); ok {
			return sel.Sel.Name == want
		}
		id, ok := e.(*ast.Ident)
		return ok && id.Name == want
	}
	return named(t.Params.List[0].Type, "ChargenResult") && named(t.Results.List[0].Type, "MapOpener") &&
		named(t.Results.List[1].Type, "error")
}
