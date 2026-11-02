package archtest

import (
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"path/filepath"
	"slices"
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
	"againrom/pkg/game.leaveUnboundEffectsOut":                                 {worldCarrierSet},
	"againrom/pkg/game.restorePointerSlots":                                    {worldCarrierSet},
	"againrom/pkg/game.seatTownPurse":                                          {worldCarrierSet},
	"againrom/pkg/game.applyOriginalDiaries":                                   {worldCarrierSet},
	"againrom/pkg/game.importSavedDiaries":                                     {worldCarrierSet},
	"againrom/pkg/game.seatTownDiaries":                                        {worldCarrierSet},
	"againrom/pkg/game.projectSavedDiaries":                                    {worldCarrierSet},
	"againrom/pkg/game.applyOriginalDeadWithOrigins":                           {worldCarrierSet},
	"againrom/pkg/game.projectDeadActors":                                      {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentTownEffectOwners":                        {worldCarrierSet},
	"againrom/pkg/game.finalizeLoadedTokenReferences":                          {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentObjectIdentityAbsence":                   {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentOrderSpellAbsence":                       {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentBookSelection":                           {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentCarrierAbsence":                          {worldCarrierSet},
	"againrom/pkg/game.restoreCurrentCarrierAbsence":                           {worldCarrierSet},
	"againrom/pkg/game.FrontEnd.finalizeCurrentPlaneResidue":                   {worldCarrierSet},
	"againrom/pkg/game.finalizeCurrentWorldEffectTokens":                       {worldCarrierSet},
	"againrom/pkg/game.campaignService.seatWorld":                              {worldCarrierSet},
	"againrom/pkg/game.firstCampaignRules.seatWorld":                           {worldCarrierSet},
	"againrom/pkg/game.secondCampaignRules.seatWorld":                          {worldCarrierSet},
	"againrom/pkg/game.seatLoadedDocument":                                     {worldCarrierSet},
	"againrom/pkg/game.joinLoadedActors":                                       {worldCarrierSet},
	"againrom/pkg/game.retainLoadedDead":                                       {worldCarrierSet},
	"againrom/pkg/game.restoreCurrentActorIdentities":                          {worldCarrierSet},
	"againrom/pkg/game.importCurrentTerminalActors":                            {worldCarrierSet},
	"againrom/pkg/game.restoreCurrentPlaneResidue":                             {worldCarrierSet},
	"againrom/pkg/game.settleLoadedApplication":                                {worldCarrierSet},
	"againrom/pkg/game.carryTownKnowledge":                                     {worldCarrierSet},
	"againrom/pkg/game.projectTownKnowledge":                                   {worldCarrierSet},
	"againrom/pkg/game.savedPlayerDiary":                                       {worldCarrierSet},
	"againrom/pkg/game.loadedTokenReferenceMisses":                             {worldCarrierSet},
	"againrom/pkg/game.matchCurrentBookSelection":                              {worldCarrierSet},
	"againrom/pkg/game.matchCurrentOrderSpellAbsence":                          {worldCarrierSet},
	"againrom/pkg/game.restoreCurrentWorldEffectTokens":                        {worldCarrierSet},
	"againrom/pkg/game.generatedDocumentBuilder.reserveRetainedDeadEffectKeys": {worldCarrierSet},
	"againrom/pkg/game.generatedDocumentBuilder.reserveWorldEffectKeys":        {worldCarrierSet},
	"againrom/pkg/game.generatedDocumentBuilder.currentWorldEffects":           {worldCarrierSet},
	"againrom/pkg/game.worldEffectFamily":                                      {"againrom/pkg/game.generatedDocumentBuilder.reserveWorldEffectKeys", "againrom/pkg/game.generatedDocumentBuilder.currentWorldEffects"},
	"againrom/pkg/game.keepLoadedPlayerName":                                   {worldCarrierSet},
	"againrom/pkg/game.currentPlayerName":                                      {"againrom/pkg/game.keepLoadedPlayerName"},
	"againrom/pkg/game.projectSavedActorActions":                               {worldCarrierSet},
	"againrom/pkg/game.loadedBodyReference":                                    {"againrom/pkg/game.projectSavedActorActions", "againrom/pkg/game.keepLoadedActionLinks"},
	"againrom/pkg/game.keepLoadedActionLinks":                                  {"againrom/pkg/game.projectSavedActorActions"},
	"againrom/pkg/game.loadedActorRecord":                                      {"againrom/pkg/game.keepLoadedActionLinks"},
	"againrom/pkg/game.graftUnknownObjects":                                    {worldCarrierSet},
	"againrom/pkg/game.graftUnknownSingletons":                                 {"againrom/pkg/game.graftUnknownObjects"},
	"againrom/pkg/game.restoreSavedDeadArt":                                    {worldCarrierSet},
	"againrom/pkg/game.writeDeadArt":                                           {worldCarrierSet},
	"againrom/pkg/game.restoreResumedSessionHead":                              {worldCarrierSet},
	"againrom/pkg/game.publishSessionEntry":                                    {worldCarrierSet},
	"againrom/pkg/game.restoreAbsentSessionHead":                               {worldCarrierSet, "againrom/pkg/game.restoreResumedSessionHead"},
	"againrom/pkg/game.generatedDocumentBuilder.retainDeadValues":              {"againrom/pkg/game.retainLoadedDead"},
	"againrom/pkg/game.generatedDocumentBuilder.retainDeadBook":                {"againrom/pkg/game.retainLoadedDead"},
	"againrom/pkg/game.generatedDocumentBuilder.retainDeadEffects":             {"againrom/pkg/game.retainLoadedDead"},
	"againrom/pkg/game.matchingCurrentPlaneResidue":                            {"againrom/pkg/game.restoreCurrentPlaneResidue", "againrom/pkg/game.readCurrentActions"},
	"againrom/pkg/game.restoreDepartedContents":                                {"againrom/pkg/game.importCurrentTerminalActors"},
	"againrom/pkg/game.mapWorld.restoreApplicationState":                       {"againrom/pkg/game.settleLoadedApplication"},
	"againrom/pkg/game.mapWorld.restoreOriginalActionView":                     {"againrom/pkg/game.settleLoadedApplication"},
	"againrom/pkg/sim.World.ImportCurrentTerminalActors":                       {"againrom/pkg/game.importCurrentTerminalActors"},
	"againrom/pkg/sim.World.RestoreCurrentCellCosts":                           {"againrom/pkg/game.restoreCurrentPlaneResidue"},
	"againrom/pkg/game.savedDiaryRecords":                                      {"againrom/pkg/game.importSavedDiaries", "againrom/pkg/game.projectSavedDiaries", "againrom/pkg/game.currentDiaryRecords"},
	"againrom/pkg/sim.CompareSavedDiaryOwners":                                 {"againrom/pkg/game.importSavedDiaries", "againrom/pkg/game.finalizeCurrentCarrierAbsence"},
	"againrom/pkg/sim.World.ImportOriginalDiaries":                             {"againrom/pkg/game.applyOriginalDiaries", "againrom/pkg/game.importSavedDiaries"},
	"againrom/pkg/sim.World.SetSavedDiaries":                                   {"againrom/pkg/game.seatTownDiaries"},
	"againrom/pkg/sim.World.ImportOriginalDeadActors":                          {"againrom/pkg/game.applyOriginalDeadWithOrigins"},
	"againrom/pkg/sim.World.SetPurse":                                          {"againrom/pkg/game.seatTownPurse", "againrom/pkg/game.importSavedPlayerPurses", "againrom/pkg/game.savedPlayerPurseRows", "againrom/pkg/game.restoreCurrentPlayerSlots"},
	"againrom/pkg/sim.World.RestoreCurrentSackKeyAbsence":                      {"againrom/pkg/game.restoreCurrentCarrierAbsence"},
	"againrom/pkg/sim.World.RestoreCurrentCarrierAbsence":                      {"againrom/pkg/game.restoreCurrentCarrierAbsence"},
	"againrom/pkg/sim.World.SetROM2ScenarioState":                              {"againrom/pkg/game.secondCampaignRules.seatWorld"},
	"againrom/pkg/sim.World.SetSavedProjectiles":                               {"againrom/pkg/game.leaveUnboundEffectsOut", "againrom/pkg/game.seedProjectileCounter"},
	"againrom/pkg/sim.World.SetSavedSpellEffects":                              {"againrom/pkg/game.leaveUnboundEffectsOut", "againrom/pkg/game.applyOriginalSpellEffects"},
	"againrom/pkg/sim.World.RestoreStructureUseMetadata":                       {"againrom/pkg/game.restorePointerSlots", "againrom/pkg/game.mapWorld.restoreStructureUseMetadata"},
	"againrom/pkg/sim.World.SetLoadedRecord":                                   {worldCarrierSet, "againrom/pkg/game.restoreSavedDocumentDecoded"},
	"againrom/pkg/sim.World.LoadedRecord":                                      {worldCarrierSet, "againrom/pkg/game.leaveUnboundEffectsOut", "againrom/pkg/game.secondCampaignRules.seatWorld"},
}

const worldCarrierSet = "againrom/pkg/game.worldCarrierSet"

// worldCarrierSetFile is the one production file that may write a field of
// worldCarrierFields.
const worldCarrierSetFile = "worldcarriers.go"

// worldCarrierFields are the fields a carrier owns outside the World, each
// with the functions outside the set file that may write it: the loaded
// document the mission writers read, the town's carried Diary, and the
// dead body art the dead body row's original LOAD hook decodes, the identity
// step remaps and the autosave's detached mission copies.
var worldCarrierFields = map[string][]string{
	"againrom/pkg/game.Snapshot.loadedDocument": nil,
	"againrom/pkg/game.Town.knowledge":          nil,
	"againrom/pkg/game.Mission.DeadArt": {"againrom/pkg/game.applyOriginalDeadWithOrigins",
		"againrom/pkg/game.restoreCurrentActorIdentities", "againrom/pkg/game.FrontEnd.detachedMission"},
}

// worldCarrierLoadedFields are the Snapshot fields that hold the loaded
// record a SAVE's carriers read.
var worldCarrierLoadedFields = []string{
	"againrom/pkg/game.Snapshot.SavedDocument",
	"againrom/pkg/game.Snapshot.loadedDocument",
}

// worldCarrierLoadedCallees are the functions outside the set that may take
// a read of a worldCarrierLoadedFields field as an argument, each with why
// it keeps no carrier from it.
var worldCarrierLoadedCallees = map[string]string{
	"againrom/pkg/game.cloneSavedDocument":              "the capture chain's copy of the record",
	"againrom/pkg/game.generatedCurrentMissionDocument": "the capture chain's test of whether the World built the record",
	"againrom/pkg/game.restoreSavedDocument":            "the LOAD's binding of the record, through its validators",
	"againrom/pkg/game.applicationActorIDs":             "reads the actor bindings of the document the SAVE has just built",
}

// worldCarrierDocumentNames are the names whose nil test is a test of the
// loaded record: a Mission's or Snapshot's saved document and a saved
// document's Document.
var worldCarrierDocumentNames = []string{"savedDocument", "SavedDocument", "Document"}

// worldCarrierNilDocumentReaders are the production functions outside the set
// file, neither the capture chain nor a LOAD validator, that may test the
// loaded record against nil, each with why it is no carrier step.
var worldCarrierNilDocumentReaders = map[string]string{
	"againrom/pkg/game.mapWorld.virtualDeadDraws": "a draw that reads the loaded Player records and writes no carrier",
	"againrom/pkg/game.loadedTavernStockIDs":      "tests the city file's own decoded document for the tavern stock, not the mission's loaded record",
	"againrom/pkg/game.assertHeadlessState":       "its Document is a headless assertion field that shares the name",
}

// worldCarrierNilDocumentOwners are the production functions outside the set
// file that may test the loaded record against nil: the document capture
// chain and its LOAD validators. A function only the set or a step may call
// (a key of worldCarrierOwners) is a step and may test it too.
var worldCarrierNilDocumentOwners = []string{
	"againrom/pkg/game.CampaignSession.missionCityProvenance",
	"againrom/pkg/game.CampaignSession.retainReturnCity",
	"againrom/pkg/game.FrontEnd.Snapshot",
	"againrom/pkg/game.FrontEnd.captureApplicationState",
	"againrom/pkg/game.armCarriedSpellGraph",
	"againrom/pkg/game.attachSavedObjectAuthority",
	"againrom/pkg/game.captureCurrentObjects",
	"againrom/pkg/game.captureMissionCityBooks",
	"againrom/pkg/game.cloneSavedDocument",
	"againrom/pkg/game.currentAbsentDeadClasses",
	"againrom/pkg/game.currentActorGraph",
	"againrom/pkg/game.currentAdmissionHealthByIdentity",
	"againrom/pkg/game.currentNativePolicy",
	"againrom/pkg/game.currentProfileBases",
	"againrom/pkg/game.currentRetainedDeadKeys",
	"againrom/pkg/game.currentTerminalActorBindings",
	"againrom/pkg/game.currentTerminalActorKeys",
	"againrom/pkg/game.currentTerminalImportRows",
	"againrom/pkg/game.currentTerminalObjectIdentity",
	"againrom/pkg/game.currentWorldDocument",
	"againrom/pkg/game.deadRecordObject",
	"againrom/pkg/game.documentApplication",
	"againrom/pkg/game.generatedCurrentMissionDocument",
	"againrom/pkg/game.importOriginalActorEffects",
	"againrom/pkg/game.importOriginalCellPlanes",
	"againrom/pkg/game.importOriginalWorldEffects",
	"againrom/pkg/game.importSavedActorActions",
	"againrom/pkg/game.importSavedActorMotions",
	"againrom/pkg/game.importSavedDocument",
	"againrom/pkg/game.importSavedPlayerParticipants",
	"againrom/pkg/game.loadedDeadRoot",
	"againrom/pkg/game.mapWorld.restoreOriginalActionView",
	"againrom/pkg/game.originalSource.prepareMission",
	"againrom/pkg/game.projectApplicationState",
	"againrom/pkg/game.projectCurrentUnboundSackRoots",
	"againrom/pkg/game.projectDeadActorOrders",
	"againrom/pkg/game.projectSavedActorMotions",
	"againrom/pkg/game.projectSavedCurrentOrder",
	"againrom/pkg/game.projectSavedGroupGraph",
	"againrom/pkg/game.projectSavedPlayerParticipants",
	"againrom/pkg/game.projectSavedWorldEffects",
	"againrom/pkg/game.reconcileSavedTerminalRegistry",
	"againrom/pkg/game.rejectRetiringSackAliases",
	"againrom/pkg/game.restoreCurrentPlayerSlots",
	"againrom/pkg/game.restoreLegacyNativeObservations",
	"againrom/pkg/game.restoreLoadedRuntimeIDs",
	"againrom/pkg/game.restoreOriginalActions",
	"againrom/pkg/game.restoreOriginalRandomState",
	"againrom/pkg/game.resumeWorld",
	"againrom/pkg/game.retainSavedActorPlacements",
	"againrom/pkg/game.savedActorTiers",
	"againrom/pkg/game.savedFormationWorld",
	"againrom/pkg/game.snapshotSavedDocument",
	"againrom/pkg/game.validateSavedActorMotionsWorld",
	"againrom/pkg/game.validateSavedDocumentWorld",
	"againrom/pkg/game.validateSavedSpellGraph",
	"againrom/pkg/game.validateSavedWorldEffectsWorld",
}

// CheckWorldCarriers reports each production reference to a function of
// worldCarrierOwners from a function the table does not name, and each
// function of the game package a row of the set references that the table
// does not name, so a new row's hook cannot stand outside the rule.
func CheckWorldCarriers(packages []*CheckedPackage) []string {
	findings := checkReferenceOwners(packages, worldCarrierOwners)
	findings = append(findings, checkWorldCarrierFields(packages)...)
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

// checkWorldCarrierFields reports, in the game package's production files
// other than the set file, each write of a field of worldCarrierFields by a
// function its entry does not name, each read of a worldCarrierLoadedFields
// field passed to a function worldCarrierLoadedCallees does not name, and
// each nil test of the loaded record in a function that is no step and that
// worldCarrierNilDocumentOwners and worldCarrierNilDocumentReaders do not
// name. A read of the loaded record is one through the field or through a
// local it was assigned to, whatever the local's name; a nil test is also
// one of a name of worldCarrierDocumentNames.
func checkWorldCarrierFields(packages []*CheckedPackage) []string {
	var findings []string
	for _, p := range packages {
		if p.ImportPath != "againrom/pkg/game" {
			continue
		}
		fields := map[*types.Var]string{}
		for _, obj := range p.Info.Defs {
			name, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			if st, ok := name.Type().Underlying().(*types.Struct); ok {
				for i := range st.NumFields() {
					fields[st.Field(i)] = p.ImportPath + "." + name.Name() + "." + st.Field(i).Name()
				}
			}
		}
		fieldOf := func(id *ast.Ident) string {
			v, _ := p.Info.Uses[id].(*types.Var)
			if v == nil {
				return ""
			}
			return fields[v]
		}
		field := func(e ast.Expr) string {
			e = ast.Unparen(e)
			if x, ok := e.(*ast.IndexExpr); ok {
				e = ast.Unparen(x.X)
			}
			var id *ast.Ident
			switch x := e.(type) {
			case *ast.SelectorExpr:
				id = x.Sel
			case *ast.Ident:
				id = x
			default:
				return ""
			}
			if key := fieldOf(id); key != "" {
				if _, owned := worldCarrierFields[key]; owned {
					return key
				}
			}
			return ""
		}
		for i, file := range p.Files[:p.NumProd] {
			if i < len(p.RelPaths) && path.Base(filepath.ToSlash(p.RelPaths[i])) == worldCarrierSetFile {
				continue
			}
			for _, decl := range file.Decls {
				owner := p.ImportPath + ".package scope"
				if fn, ok := decl.(*ast.FuncDecl); ok {
					if obj, _ := p.Info.Defs[fn.Name].(*types.Func); obj != nil {
						owner = saveCaptureKey(obj)
					}
				}
				write := func(e ast.Expr) {
					if key := field(e); key != "" && !slices.Contains(worldCarrierFields[key], owner) {
						findings = append(findings, owner+" writes "+key+" outside the carrier set")
					}
				}
				// loaded holds the locals assigned a read of the loaded record.
				loaded := map[types.Object]bool{}
				var loadedRead func(e ast.Expr) bool
				loadedRead = func(e ast.Expr) bool {
					switch x := ast.Unparen(e).(type) {
					case *ast.Ident:
						return loaded[p.Info.Uses[x]]
					case *ast.SelectorExpr:
						return slices.Contains(worldCarrierLoadedFields, fieldOf(x.Sel)) || loadedRead(x.X)
					case *ast.StarExpr:
						return loadedRead(x.X)
					case *ast.UnaryExpr:
						return x.Op == token.AND && loadedRead(x.X)
					case *ast.IndexExpr:
						return loadedRead(x.X)
					case *ast.SliceExpr:
						return loadedRead(x.X)
					}
					return false
				}
				for range 2 {
					ast.Inspect(decl, func(node ast.Node) bool {
						switch n := node.(type) {
						case *ast.AssignStmt:
							if len(n.Lhs) == len(n.Rhs) {
								for i, lhs := range n.Lhs {
									if id, ok := lhs.(*ast.Ident); ok && loadedRead(n.Rhs[i]) {
										if obj := p.Info.ObjectOf(id); obj != nil {
											loaded[obj] = true
										}
									}
								}
							}
						case *ast.ValueSpec:
							if len(n.Names) == len(n.Values) {
								for i, id := range n.Names {
									if loadedRead(n.Values[i]) {
										if obj := p.Info.ObjectOf(id); obj != nil {
											loaded[obj] = true
										}
									}
								}
							}
						}
						return true
					})
				}
				_, step := worldCarrierOwners[owner]
				nilAllowed := step || slices.Contains(worldCarrierNilDocumentOwners, owner) || worldCarrierNilDocumentReaders[owner] != ""
				ast.Inspect(decl, func(node ast.Node) bool {
					switch n := node.(type) {
					case *ast.CallExpr:
						var callee *ast.Ident
						switch fun := ast.Unparen(n.Fun).(type) {
						case *ast.Ident:
							callee = fun
						case *ast.SelectorExpr:
							callee = fun.Sel
						}
						fn, _ := p.Info.Uses[callee].(*types.Func)
						if fn == nil {
							return true
						}
						key := saveCaptureKey(fn)
						for _, arg := range n.Args {
							if loadedRead(arg) && worldCarrierLoadedCallees[key] == "" {
								findings = append(findings, owner+" passes the loaded record to "+key+" outside the carrier set")
							}
						}
					case *ast.AssignStmt:
						for _, lhs := range n.Lhs {
							write(lhs)
						}
					case *ast.IncDecStmt:
						write(n.X)
					case *ast.UnaryExpr:
						if n.Op == token.AND {
							write(n.X)
						}
					case *ast.KeyValueExpr:
						if _, ok := n.Key.(*ast.Ident); ok {
							write(n.Key)
						}
					case *ast.BinaryExpr:
						if n.Op != token.EQL && n.Op != token.NEQ {
							return true
						}
						for _, pair := range [][2]ast.Expr{{n.X, n.Y}, {n.Y, n.X}} {
							if nilIdent(pair[1]) && !nilAllowed && (slices.Contains(worldCarrierDocumentNames, lastName(pair[0])) || loadedRead(pair[0])) {
								findings = append(findings, owner+" tests "+lastName(pair[0])+" against nil outside the carrier set")
							}
						}
					}
					return true
				})
			}
		}
	}
	return findings
}

// nilIdent reports whether e is the identifier nil.
func nilIdent(e ast.Expr) bool {
	id, ok := ast.Unparen(e).(*ast.Ident)
	return ok && id.Name == "nil"
}

// lastName is the identifier e ends in: a name or a selector's field.
func lastName(e ast.Expr) string {
	switch x := ast.Unparen(e).(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	}
	return ""
}
