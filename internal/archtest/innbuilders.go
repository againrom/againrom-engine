package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

const innBuilderFile = "pkg/game/inn.go"

var innBuilders = map[string]string{
	"innRoster": "talker roster", "innRegistry": "talker roster", "innKeptAccepted": "talker roster",
	"innSeats": "seat list", "innActor": "selected NPC", "platoonTalkObject": "selected NPC",
	"innButtons": "seat dispatch", "innAction": "seat dispatch", "innControl": "seat dispatch",
	"startInnTalk": "talk start",
}

var innAdapterEntries = map[string]bool{
	"pkg/game/town.go:TavernRoster":  true,
	"pkg/game/secondinn.go:speakers": true, "pkg/game/secondinn.go:innSeats": true,
	"pkg/game/taverntalk.go:resolveTavernTalk": true,
	"pkg/game/campaignactor.go:campaignActor":  true,
	"pkg/game/townscreen.go:openOfferDialogue": true, "pkg/game/townscreen.go:openTownDialogue": true,
	"pkg/game/townscreen.go:openMercenaryDialogue": true,
	"pkg/game/secondcampaign.go:setTownDialogue":   true,
}

var innEffectEntries = map[string]map[string]string{
	"pkg/game/hire.go:run":                              {"talkTo": "hire transaction sends its word to TalkTo"},
	"pkg/game/secondcensus.go:secondEngineReachFrom":    {"talkTo": "reachability instrument applies campaign effects without a page"},
	"pkg/game/speech_witness.go:SecondTownSpeechCensus": {"openSection": "speech instrument supplies a cursor without a visit"},
}

// CheckInnBuilders checks the five inn kinds' source shapes and thin adapters.
// It does not attempt to prove arbitrary equivalent programs share a builder.
func CheckInnBuilders(files map[string]string) []Violation {
	var names []string
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		set := token.NewFileSet()
		f, err := parser.ParseFile(set, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			out = append(out, Violation{From: name, Reason: err.Error()})
			continue
		}
		for _, decl := range f.Decls {
			owner := "package scope"
			if fn, ok := decl.(*ast.FuncDecl); ok {
				owner = fn.Name.Name
				if kind := innBuilders[owner]; kind != "" && name != innBuilderFile && !innAdapterEntries[name+":"+owner] {
					out = append(out, Violation{From: name + ":" + strconv.Itoa(set.Position(fn.Pos()).Line), Reason: kind + ": " + owner + " belongs to " + innBuilderFile})
				}
			}
			if name == innBuilderFile {
				continue
			}
			entry := name + ":" + owner
			rosterContext := false
			ast.Inspect(decl, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && id.Name == "innOffer" {
					rosterContext = true
				}
				if call, ok := n.(*ast.CallExpr); ok && lastName(call.Fun) == "innOptions" {
					rosterContext = true
				}
				return true
			})
			add := func(n ast.Node, kind string) {
				out = append(out, Violation{From: name + ":" + strconv.Itoa(set.Position(n.Pos()).Line), Reason: kind + ": " + owner + " bypasses " + innBuilderFile})
			}
			ast.Inspect(decl, func(node ast.Node) bool {
				switch n := node.(type) {
				case *ast.CompositeLit:
					typ := lastName(n.Type)
					if typ == "tavernTalkObject" && len(n.Elts) > 0 {
						add(n, "selected NPC")
					}
					if typ == "innSeat" && len(n.Elts) > 0 && entry != "pkg/game/secondcampaign.go:innAct" {
						add(n, "seat list")
					}
				case *ast.CallExpr:
					called := lastName(n.Fun)
					switch called {
					case "speaker":
						if rosterContext {
							add(n, "talker roster")
						}
					case "pressMercenary", "seatHire":
						add(n, "seat dispatch")
					case "openTalk", "joinAtTalk":
						add(n, "talk start")
					case "talkTo", "openSection":
						if innEffectEntries[entry][called] == "" {
							add(n, "talk start")
						}
					case "openOfferDialogue", "openTownDialogue":
						// A tavern seat's talk is innAction's Talk verb.
						if len(n.Args) > 0 && lastName(n.Args[0]) == "TownTavern" {
							add(n, "seat dispatch")
						}
					}
					if called == "append" {
						for _, arg := range n.Args {
							if call, ok := arg.(*ast.CallExpr); ok && lastName(call.Fun) == "speakers" {
								add(n, "seat list")
							}
						}
					}
				case *ast.IfStmt:
					if innAdapterEntries[entry] {
						ast.Inspect(n.Cond, func(x ast.Node) bool {
							if s, ok := x.(*ast.SelectorExpr); ok {
								switch s.Sel.Name {
								case "npc", "topic", "kind", "high", "NPC", "Mission", "Tokens", "Membership", "Affordable", "Hired":
									add(n, "adapter decision")
								}
							}
							return true
						})
					}
				}
				return true
			})
		}
	}
	return out
}
