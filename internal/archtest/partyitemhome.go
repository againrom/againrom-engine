package archtest

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strconv"
)

// PartyItemHomeFile is the one home of what a campaign party carries between
// missions: every unit conversion of a crossing and every pack replacement
// is made there.
const PartyItemHomeFile = "pkg/game/partyitemhome.go"

// partyItemHomeOnly are the conversions and the pack replacement production
// references in the home's file alone, by the function's own identity: an
// import alias, a method value or a function value reaches the same function.
var partyItemHomeOnly = []string{
	"againrom/pkg/secondshop.KeptUnit",
	"againrom/pkg/secondshop.MissionUnit",
	"againrom/pkg/game.memberWithPack",
}

// partyItemOwnPackage are the functions their own package may reference
// beside the home: the conversion's implementation is not a crossing.
var partyItemOwnPackage = map[string]string{
	"againrom/pkg/secondshop.MissionUnit": "againrom/pkg/secondshop",
}

// partyItemGraphFollow is the town graph's one follower of a pack. Its
// declaration and its one production reference stand in the home.
const partyItemGraphFollow = "againrom/pkg/game.CampaignSession.followHeroPack"

// CheckPartyItemHome names each production reference to a party unit
// conversion or the pack replacement outside the home, and a second
// reference to the graph's follower. A reference is any use of the function's
// name that the type-checker resolves to it.
func CheckPartyItemHome(m *TypeCheckedModule) []string {
	var findings []string
	homeSeen, followInHome := false, 0
	home := map[string]bool{}
	for _, key := range partyItemHomeOnly {
		home[key] = true
	}
	for _, p := range m.Packages {
		for i, file := range p.Files[:p.NumProd] {
			rel := p.RelPaths[i]
			if rel == PartyItemHomeFile {
				homeSeen = true
			}
			ast.Inspect(file, func(node ast.Node) bool {
				id, ok := node.(*ast.Ident)
				if !ok {
					return true
				}
				used, _ := p.Info.Uses[id].(*types.Func)
				if used == nil {
					return true
				}
				key := saveCaptureKey(used)
				switch {
				case key == partyItemGraphFollow && rel == PartyItemHomeFile:
					followInHome++
				case key == partyItemGraphFollow && rel != PartyItemHomeFile:
					findings = append(findings, fmt.Sprintf("%s:%s references %s outside the party item home", rel, line(m, id), key))
				case home[key] && rel != PartyItemHomeFile && partyItemOwnPackage[key] != p.ImportPath:
					findings = append(findings, fmt.Sprintf("%s:%s references %s outside the party item home", rel, line(m, id), key))
				}
				return true
			})
		}
	}
	if !homeSeen {
		findings = append(findings, "the party item home "+PartyItemHomeFile+" is missing")
	} else if followInHome != 1 {
		findings = append(findings, fmt.Sprintf("the party item home references %s %d times, want its one call", partyItemGraphFollow, followInHome))
	}
	sort.Strings(findings)
	return findings
}

func line(m *TypeCheckedModule, n ast.Node) string {
	return strconv.Itoa(m.Fset.Position(n.Pos()).Line)
}
