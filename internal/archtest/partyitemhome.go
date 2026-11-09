package archtest

import (
	"fmt"
	"sort"
	"strings"
)

// PartyItemHomeFile is the one home of what a campaign party carries between
// missions: every unit conversion of a crossing and every pack replacement
// is made there.
const PartyItemHomeFile = "pkg/game/partyitemhome.go"

// partyItemHomeOnly are the conversions and the pack replacement production
// makes in the home's file alone.
var partyItemHomeOnly = []string{"secondshop.KeptUnit", "secondshop.MissionUnit", "memberWithPack("}

// partyItemGraphFollow is the town graph's one follower of a pack. Its
// declaration and its one production call stand in the home.
const partyItemGraphFollow = "followHeroPack("

// CheckPartyItemHome names each production source that converts a party unit
// or replaces a pack outside the home, and a second call of the graph's
// follower.
func CheckPartyItemHome(files map[string]string) []string {
	var findings []string
	for rel, src := range files {
		for _, name := range partyItemHomeOnly {
			if rel != PartyItemHomeFile && strings.Contains(src, name) {
				findings = append(findings, fmt.Sprintf("%s uses %s outside the party item home", rel, name))
			}
		}
		if n := strings.Count(src, partyItemGraphFollow); rel != PartyItemHomeFile && n != 0 {
			findings = append(findings, fmt.Sprintf("%s calls %s outside the party item home", rel, partyItemGraphFollow))
		} else if rel == PartyItemHomeFile && n != 2 {
			findings = append(findings, fmt.Sprintf("the party item home holds %s %d times, want its declaration and one call", partyItemGraphFollow, n))
		}
	}
	sort.Strings(findings)
	return findings
}
