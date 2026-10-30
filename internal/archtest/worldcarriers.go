package archtest

// worldCarrierOwners names, for each hook of a World carrier and for the
// World's loaded-record carrier, the production functions that may reference
// it. The SAV producer, the LOAD and the mission seat run a carrier through
// the one carrier set, so a direct call of a hook is a second home; the
// loaded-record carrier has one writer, the SAV LOAD restore, and its
// readers are the set's absence and seat rules.
var worldCarrierOwners = map[string][]string{
	"againrom/pkg/game.leaveUnboundEffectsOut":               {worldCarrierSet},
	"againrom/pkg/game.restorePointerSlots":                  {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentTownEffectOwners":      {worldCarrierSet},
	"againrom/pkg/game.finalizeLoadedTokenReferences":        {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentObjectIdentityAbsence": {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentOrderSpellAbsence":     {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentBookSelection":         {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentCarrierAbsence":        {worldCarrierSet},
	"againrom/pkg/game.FrontEnd.finalizeCurrentPlaneResidue": {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentWorldEffectTokens":     {worldCarrierSet},
	"againrom/pkg/game.campaignService.seatWorld":            {worldCarrierSet},
	"againrom/pkg/game.firstCampaignRules.seatWorld":         {worldCarrierSet},
	"againrom/pkg/game.secondCampaignRules.seatWorld":        {worldCarrierSet},
	"againrom/pkg/sim.World.SetLoadedRecord":                 {worldCarrierSet, "againrom/pkg/game.restoreSavedDocumentDecoded"},
	"againrom/pkg/sim.World.LoadedRecord":                    {worldCarrierSet, "againrom/pkg/game.leaveUnboundEffectsOut", "againrom/pkg/game.secondCampaignRules.seatWorld"},
}

const worldCarrierSet = "againrom/pkg/game.worldCarrierSet"

// CheckWorldCarriers reports each production reference to a function of
// worldCarrierOwners from a function the table does not name.
func CheckWorldCarriers(packages []*CheckedPackage) []string {
	return checkReferenceOwners(packages, worldCarrierOwners)
}
