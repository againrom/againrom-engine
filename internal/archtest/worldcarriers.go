package archtest

import (
	"go/ast"
	"go/types"
	"sort"
)

// worldCarrierOwners names, for each hook of a carrier and for the World
// state a carrier owns, the production functions that may reference it. The
// SAV producer, the LOAD and the mission seat run a carrier through the one
// carrier set, so a direct call of a hook is a second home, and so is a direct
// call of a World writer a row owns. The loaded-record carrier has one
// writer, the SAV LOAD restore; the Diaries, the dead bodies, the carrier
// absences and the Scenario bank enter a World only through their rows'
// hooks. The town purse is seated only by its row; the other purse writers
// bind the Player purses a loaded document carries. The effect and Pointer
// writers also serve the original LOAD, the counter seed and the map start.
var worldCarrierOwners = map[string][]string{
	"againrom/pkg/game.leaveUnboundEffectsOut":               {worldCarrierSet},
	"againrom/pkg/game.restorePointerSlots":                  {worldCarrierSet},
	"againrom/pkg/game.seatTownPurse":                        {worldCarrierSet},
	"againrom/pkg/game.applyOriginalDiaries":                 {worldCarrierSet},
	"againrom/pkg/game.importSavedDiaries":                   {worldCarrierSet},
	"againrom/pkg/game.seatTownDiaries":                      {worldCarrierSet},
	"againrom/pkg/game.projectSavedDiaries":                  {worldCarrierSet},
	"againrom/pkg/game.applyOriginalDeadWithOrigins":         {worldCarrierSet, "againrom/pkg/game.applyOriginalDead"},
	"againrom/pkg/game.projectDeadActors":                    {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentTownEffectOwners":      {worldCarrierSet},
	"againrom/pkg/game.finalizeLoadedTokenReferences":        {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentObjectIdentityAbsence": {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentOrderSpellAbsence":     {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentBookSelection":         {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentCarrierAbsence":        {worldCarrierSet},
	"againrom/pkg/game.restoreCurrentCarrierAbsence":         {worldCarrierSet},
	"againrom/pkg/game.FrontEnd.finalizeCurrentPlaneResidue": {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentWorldEffectTokens":     {worldCarrierSet},
	"againrom/pkg/game.campaignService.seatWorld":            {worldCarrierSet},
	"againrom/pkg/game.firstCampaignRules.seatWorld":         {worldCarrierSet},
	"againrom/pkg/game.secondCampaignRules.seatWorld":        {worldCarrierSet},
	"againrom/pkg/game.savedDiaryRecords":                    {"againrom/pkg/game.importSavedDiaries", "againrom/pkg/game.projectSavedDiaries", "againrom/pkg/game.currentDiaryRecords"},
	"againrom/pkg/sim.CompareSavedDiaryOwners":               {"againrom/pkg/game.importSavedDiaries", "againrom/pkg/game.finalizeCurrentCarrierAbsence"},
	"againrom/pkg/sim.World.ImportOriginalDiaries":           {"againrom/pkg/game.applyOriginalDiaries", "againrom/pkg/game.importSavedDiaries"},
	"againrom/pkg/sim.World.SetSavedDiaries":                 {"againrom/pkg/game.seatTownDiaries"},
	"againrom/pkg/sim.World.ImportOriginalDeadActors":        {"againrom/pkg/game.applyOriginalDeadWithOrigins"},
	"againrom/pkg/sim.World.SetPurse":                        {"againrom/pkg/game.seatTownPurse", "againrom/pkg/game.importSavedPlayerPurses", "againrom/pkg/game.savedPlayerPurseRows", "againrom/pkg/game.restoreCurrentPlayerSlots"},
	"againrom/pkg/sim.World.RestoreCurrentSackKeyAbsence":    {"againrom/pkg/game.restoreCurrentCarrierAbsence"},
	"againrom/pkg/sim.World.RestoreCurrentCarrierAbsence":    {"againrom/pkg/game.restoreCurrentCarrierAbsence"},
	"againrom/pkg/sim.World.SetROM2ScenarioState":            {"againrom/pkg/game.secondCampaignRules.seatWorld"},
	"againrom/pkg/sim.World.SetSavedProjectiles":             {"againrom/pkg/game.leaveUnboundEffectsOut", "againrom/pkg/game.seedProjectileCounter"},
	"againrom/pkg/sim.World.SetSavedSpellEffects":            {"againrom/pkg/game.leaveUnboundEffectsOut", "againrom/pkg/game.applyOriginalSpellEffects"},
	"againrom/pkg/sim.World.RestoreStructureUseMetadata":     {"againrom/pkg/game.restorePointerSlots", "againrom/pkg/game.mapWorld.restoreStructureUseMetadata"},
	"againrom/pkg/sim.World.SetLoadedRecord":                 {worldCarrierSet, "againrom/pkg/game.restoreSavedDocumentDecoded"},
	"againrom/pkg/sim.World.LoadedRecord":                    {worldCarrierSet, "againrom/pkg/game.leaveUnboundEffectsOut", "againrom/pkg/game.secondCampaignRules.seatWorld"},
}

const worldCarrierSet = "againrom/pkg/game.worldCarrierSet"

// CheckWorldCarriers reports each production reference to a function of
// worldCarrierOwners from a function the table does not name, and each
// function of the game package a row of the set references that the table
// does not name, so a new row's hook cannot stand outside the rule.
func CheckWorldCarriers(packages []*CheckedPackage) []string {
	findings := checkReferenceOwners(packages, worldCarrierOwners)
	for _, p := range packages {
		if p.ImportPath != "againrom/pkg/game" {
			continue
		}
		for _, file := range p.Files[:p.NumProd] {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if obj, _ := p.Info.Defs[fn.Name].(*types.Func); obj == nil || saveCaptureKey(obj) != worldCarrierSet {
					continue
				}
				ast.Inspect(fn, func(node ast.Node) bool {
					id, ok := node.(*ast.Ident)
					if !ok {
						return true
					}
					used, _ := p.Info.Uses[id].(*types.Func)
					if used == nil || used.Pkg() == nil || used.Pkg().Path() != p.ImportPath {
						return true
					}
					if key := saveCaptureKey(used); worldCarrierOwners[key] == nil {
						findings = append(findings, worldCarrierSet+" runs "+key+", which the carrier rule does not name")
					}
					return true
				})
			}
		}
	}
	sort.Strings(findings)
	return findings
}
