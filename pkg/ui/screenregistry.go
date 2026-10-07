package ui

// screenEntry is one Screen value's own registration: either the payload
// composer composeScreen selects for it, or a Reason it has neither a
// composer nor a test.
//
// Two DIFFERENT kinds of test are named, and they answer different
// questions (round 2 adversarial review, finding 2). Test names the
// synthetic test proving composeScreen's own switch reaches this screen's
// composer — SELECTION, not geometry: it compares composeScreen's output
// against a second call to the same composer, so it cannot fail when the
// composer's own destination rectangles move, only when the switch stops
// reaching it at all. GeometryTests names every synthetic test that pins
// one of this screen's own destination rectangles or text origins against a
// hand-transcribed literal, independent of the constant/formula it tests,
// proved by a real +1px mutation and a byte-identical revert — this is what
// actually satisfies contract B2's bar ("shifting the screen's destination
// rectangle by one pixel must fail at least one test"). A screen can have a
// Test with an empty GeometryTests, meaning its selection is witnessed but
// none of its own drawing positions are yet pinned; cmd/screencensus prints
// that case under "unwitnessed remainder" rather than treating a non-empty
// Test as if it covered both questions.
//
// EXACTLY ONE OF Composer OR Reason IS SET, and screenRegistry_test.go
// checks that shape as well as coverage: a Screen value that composes with
// no witness, or refuses with no stated reason, is the same defect this
// registry exists to make visible.
type screenEntry struct {
	// Composer names the function composeScreen (app.go) calls for this
	// Screen — the same symbol a reader would set a breakpoint on.
	Composer string
	// Test names the synthetic (install-free) test function proving
	// composeScreen's own switch selects Composer for this Screen. Empty
	// exactly when Reason is set.
	Test string
	// GeometryTests names every synthetic test pinning one of this screen's
	// own destination rectangles or text origins, proved by mutation. May
	// be empty even when Composer and Test are set — an honest gap, not a
	// silent one; cmd/screencensus's "unwitnessed remainder" column reports
	// it.
	GeometryTests []string
	// SharesGeometry and SharesGeometryWith are set together, for a Screen
	// whose Composer is the SAME function call composeScreen's switch makes
	// for another registered Screen (round 3 fix: "case ScreenTown,
	// ScreenGameMenu: return a.composeTownScreen()" in composeScreen, and
	// drawTown itself calls a.composeTownScreen() directly for both Draw
	// arms too — verified to the leaf, not asserted, 1024 round 3). When
	// set, cmd/screencensus's census (ScreenCensus, below) resolves this
	// entry's own effective geometry witnesses as the UNION of
	// SharesGeometryWith's own GeometryTests and this entry's own
	// GeometryTests, computed at read time rather than the two test-name
	// lists being kept in sync by hand in two places.
	SharesGeometry     bool
	SharesGeometryWith Screen
	// GeometryGapNote, when non-empty, names a known coverage gap this
	// entry's own GeometryTests cannot show on its own: Composer covers more
	// than one page (chargen's pre-create/detailed split) or reachable case,
	// and only some of them have a GeometryTests entry naming them. It is
	// registry DATA, read by cmd/screencensus's own remainder section,
	// rather than a fixed sentence baked into that binary — the census
	// prints exactly this field's own text and a computed count of how many
	// entries carry one, so a later commit that gives the named case its own
	// geometry witness clears this field in the same diff hunk as the
	// GeometryTests edit, rather than the two drifting apart silently the
	// way a hardcoded print statement in cmd/screencensus did (round 2
	// adversarial review round-3 punch list, finding 2).
	GeometryGapNote string
	// Reason is set instead of Composer/Test/GeometryTests for a Screen
	// this build cannot compose on the CPU at all.
	Reason string
}

// screenRegistry names every ui.Screen value's own composer and witnesses,
// or the reason it has neither. screenregistry_test.go parses flow.go's own
// Screen const block and requires every declared value to have an entry
// here.
var screenRegistry = map[Screen]screenEntry{
	ScreenCutsceneLibrary: {
		Composer:        "(*App).composeCutsceneLibrary, selected by (*App).composeScreen",
		Test:            "TestCutsceneLibraryEncounterReplayAndInput1184",
		GeometryGapNote: "Movie list and control geometry has an installed render witness but no independent literal pixel test.",
	},
	ScreenCredits: {
		Composer:        "(*App).composeCredits, selected by (*App).composeScreen",
		Test:            "TestCreditsScrollLogoPauseClose1184",
		GeometryGapNote: "Credit motion and logo visibility are tested; exact original positions and cadence remain DIV-1275.",
	},
	ScreenEnding: {
		Composer:      "(*App).composeEndingScreen, selected by (*App).composeScreen",
		Test:          "TestComposeScreenSelectsEndingComposer",
		GeometryTests: []string{"TestEndingControlsAndSourceRowsHaveVisibleGeometry"},
	},
	ScreenSave: {
		Composer:      "(*App).composeSaveDialogScreen, selected by (*App).composeScreen",
		Test:          "TestComposeScreenSelectsSaveDialogComposer",
		GeometryTests: []string{"TestSaveDialogDrawsFieldsAndTitleAtLiteralOrigins"},
	},
	ScreenCutscene: {
		Composer:      "(*App).composeCutscene, selected by Draw and HeadlessFrame overlay guards",
		Test:          "TestCutsceneComposesAuthoredColoursAndReturnsAfterEOF",
		GeometryTests: []string{"TestCutsceneFitMatchesLiteralColoredRectangles"},
	},
	ScreenMenu: {
		Composer: "menu.Assets.Compose, selected by (*App).composeScreen",
		Test:     "TestComposeScreenSelectsMenuComposer",
		// The overlay's own destination rectangles (HoverRects/PressedRects,
		// pkg/render/menu/state.go) are pinned in that package, not here.
		// pkg/render/menu.TestNewGameButton pins HoverRects/PressedRects
		// directly against hand literals and does NOT witness composition:
		// mutated at its own use site (state.go:98, Overlay's own return of
		// HoverRects[i]) it still passes, because it never calls Compose (1024
		// round 3, mutation-verified: a +1px shift on that return statement
		// leaves TestNewGameButton green). pkg/render/menu.TestSelectionAndCompose
		// does witness composition — the same mutation fails five of its
		// subtests at the pixel level (Compose reads Overlay's returned
		// rectangle and draws there) — so that is this screen's real geometry
		// witness, not TestNewGameButton. composeScreen's own ScreenMenu case
		// adds no destination rectangle of its own; it only forwards to
		// a.assets.Compose.
		GeometryTests: []string{"pkg/render/menu.TestSelectionAndCompose"},
	},
	ScreenChargen: {
		Composer: "(*App).composeChargenScreen -> composeChargenPage (pre-create) / composeChargenDetailedPage",
		Test:     "TestComposeScreenSelectsChargenComposer",
		// Pre-create page: TestPreCreateNameControlRegionIsPublished and
		// TestPreCreateNameTextDrawsAtItsOwnOrigin, both pre-existing.
		GeometryTests: []string{
			"TestPreCreateNameControlRegionIsPublished",
			"TestPreCreateNameTextDrawsAtItsOwnOrigin",
			"TestDetailedPageDrawsTheProductionCompactCard",
			"TestDetailedPageDrawsTheProductionMessageStrip",
		},
	},
	ScreenTown: {
		Composer: "(*App).composeTownScreen -> (*App).composeTownRoom -> composeTownRoom -> ComposeTownSurface / ComposeShopScreen / ComposeTownSquare / ComposeWorldMap",
		Test:     "TestComposeScreenSelectsTownComposer",
		// THE CENSUS COUNTS AN EMPTY LIST AND CANNOT READ A SHORT ONE (1027 round
		// 2, D-1). None of them pins the statistics card, so cmd/screencensus
		// printed no unwitnessed remainder for this screen while the card was in
		// fact unwitnessed, and a maintainer reading that output would have
		// rebuilt work that already existed. A registered list is a catalog of
		// what is pinned, not a flag that something is.
		GeometryTests: []string{
			"TestSchoolSkillRectsMatchTheMeasuredLiterals",
			"TestTownButtonTextRectsSplitTheWellIntoLabelThenValue",
			"TestTownSurfaceMessageIsDrawnAtItsOwnRect",
			"TestDrawNinePatchBorderTilesFourDistinctEdges",
			"pkg/game.TestReleaseStatisticsCardFitsAtTheProductionFont",
			"pkg/game.TestReleaseShopDrawsNothingOverTheStatisticsCard",
			"TestCompactPanelLayoutCentredNameDoesNotMoveTheAlignedColumn",
			"TestTheCompactCardDrawsTheWeightRowBetweenTheResistancesAndTheTotals",
			"TestCompactMercenaryCardsFillFromTheBottomAndKeepOnlyCountAndPrice",
			"TestTavernMiniCardDrawsTheProductionSpriteAtTheMeasuredOrigin",
			"TestOnlyTheSelectedTavernMiniatureConsumesTheAnimationFrame",
			"TestTavernMiniCardBorderIsOnlyTheShippedBackground",
			"TestTalkOnlyCardUsesItsShippedBackgroundAndArtTavernAddsNoTitle",
			"TestTavernMiniCardTextUsesTheMeasuredOppositeCorners",
			"TestTavernCandidateInspectionDrawsStatsDollAndNormalHover",
		},
	},
	ScreenGameMenu: {
		// composeScreen's own switch shares this case with ScreenTown
		// ("case ScreenTown, ScreenGameMenu: return a.composeTownScreen()"),
		// and drawTown (app.go) calls a.composeTownScreen() directly for both
		// Draw's ScreenTown and ScreenGameMenu arms too — verified to the
		// leaf, not the first function that looks like it answers (1024
		// round 3). So the frame beneath the menu IS ScreenTown's own
		// composed frame, and SharesGeometryWith below tells the census to
		// count ScreenTown's four GeometryTests as witnessing this screen's
		// frame too, computed rather than duplicated by hand.
		//
		// The in-game menu's own dim and panel ARE NOT in that frame
		// (HeadlessFrame's note branch, headless.go:149): they draw through
		// ebiten's vector package (Draw's own ScreenGameMenu case, app.go),
		// which composeScreen cannot witness. But the panel and row
		// rectangles themselves (gameMenuPanelRect, gameMenuRowRect,
		// gamemenu.go) ARE pinned against hand literals, independent of
		// ebiten's vector draw call: TestTheDecodedRectangles
		// (gamemenusurface_test.go) is that witness, and is this screen's own
		// GeometryTests entry rather than a shared one (round 3 punch list,
		// finding 1: the census previously printed this screen as having no
		// geometry witness at all).
		Composer:           "(*App).composeTownScreen, shared with ScreenTown's own case in composeScreen",
		Test:               "TestComposeScreenSelectsTownComposer",
		SharesGeometry:     true,
		SharesGeometryWith: ScreenTown,
		GeometryTests:      []string{"TestTheDecodedRectangles"},
	},
	ScreenPicker: {
		Reason: "ScreenPicker draws only through ebitenutil.DebugPrintAt, no CPU composite (composeScreen's default arm)",
	},
	ScreenLoad: {
		Composer:      "(*App).composeLoadScreen, selected by (*App).composeScreen",
		Test:          "TestComposeScreenSelectsLoadComposer",
		GeometryTests: []string{"TestLoadDrawsUnicodeInsideFramedListAndKeepsExactLoadToken"},
	},
	ScreenDocuments: {
		Composer: "(*App).composeScreen -> composeDocumentsPanel (documents.go)",
		Test:     "TestComposeScreenSelectsDocumentsComposer",
		// Both entries mutate a production USE site in composeDocumentsPanel
		// and not the rectangle-returning declaration (AGENTS.md rule 6).
		//
		// TestDocumentsPanelDrawsItsThreeControlsAtTheirOwnRects builds the
		// expected frame itself, from four distinctly coloured synthetic
		// bitmaps blitted at hand-written literals, and compares it against
		// the production composite pixel by pixel over the whole 640x480
		// frame. Shifting any one of the three control copyNative calls by
		// one pixel (docLeftRect.Min.Add(image.Pt(1, 0)) and the two beside
		// it) fails it; reverted byte-identical.
		//
		// TestDocumentsPanelDrawsTheTextPageAtItsOwnOriginAndPitch draws its
		// expectation with direct font.Draw calls at a hand-written origin
		// and pitch, not by a second call to composeDocumentsPanel, and
		// compares pixel by pixel over the whole frame. Mutating the page
		// loop's own y (docContentAt.Y+i*pitch+1) fails it, as does changing
		// the pitch term; reverted byte-identical.
		GeometryTests: []string{
			"TestDocumentsPanelDrawsItsThreeControlsAtTheirOwnRects",
			"TestDocumentsPanelDrawsTheTextPageAtItsOwnOriginAndPitch",
		},
	},
	ScreenMap: {
		Reason: "the mission screen: Viewer.Draw composes the whole frame on its own ebiten canvas and then places that canvas in the window, no CPU composite; composeScreen's own switch never sees this value (Draw returns from its mapShowing() branch first). Selection is not witnessed because composeScreen never dispatches here; the frame's own size and placement are, by the first two geometry tests below (story 1026); the right column's own four widgets are, by the third and fourth (story 1023; story 1036 round 3 replaces the authored figure box with the fourth column child's own decoded background and keeps the worn box's own hooked blit); and the character pane's six corner controls and its own two presentations at the shipped frame are, by the last three (story 1036), which are the only part of this screen composed on the CPU at all",
		GeometryTests: []string{
			"TestMissionDrawComposesOnTheFrameAndPlacesIt",
			"TestMissionFramePartitionsIntoViewportAndStrip",
			"TestMissionColumnDrawsThePanelMinimapAndControlPanelAtTheirOwnSlots",
			"TestMissionColumnWornBoxUsesTheHookedBlit",
			"TestMissionCharacterPaneCornersDrawAtTheDecodedRectangles",
			"TestMissionCharacterPaneCornersFollowTheirOwnState",
			"TestMissionCharacterPaneTogglesBetweenFigureAndStatistics",
			"TestMissionEmptySelectionKeepsAnEmptyDollAboveAnEmptyStatisticsPage",
		},
	},
}

// screenConstName is the identifier flow.go's own const block spells each
// Screen value with. screenregistry_test.go parses that block from source
// and looks every identifier it finds up here; an identifier this map does
// not know fails the test closed, the same way an unlisted package fails
// internal/archtest's Check.
var screenConstName = map[string]Screen{
	"ScreenCutsceneLibrary": ScreenCutsceneLibrary,
	"ScreenCredits":         ScreenCredits,
	"ScreenEnding":          ScreenEnding,
	"ScreenMenu":            ScreenMenu,
	"ScreenPicker":          ScreenPicker,
	"ScreenMap":             ScreenMap,
	"ScreenChargen":         ScreenChargen,
	"ScreenTown":            ScreenTown,
	"ScreenGameMenu":        ScreenGameMenu,
	"ScreenLoad":            ScreenLoad,
	"ScreenDocuments":       ScreenDocuments,
	"ScreenCutscene":        ScreenCutscene,
	"ScreenSave":            ScreenSave,
}

// ScreenCensusEntry is screenRegistry's own row, exported for
// cmd/screencensus. It is the only exported read of screenRegistry; nothing
// inside this package needs one, since production code never consults its
// own registry — the registry describes tests, it is not read by the
// composer it names.
type ScreenCensusEntry struct {
	Screen   Screen
	Name     string
	Composer string
	Test     string
	// GeometryTests is this entry's OWN GeometryTests, unresolved — the
	// list screenregistry_test.go polices existence for. Reported separately
	// from EffectiveGeometryTests so a reader can tell "this screen's own
	// witness" from "witnessed because another screen's composer produces
	// this screen's frame too".
	GeometryTests []string
	// EffectiveGeometryTests is GeometryTests, plus SharedWith's own
	// GeometryTests when SharedWith is non-empty — computed here, once, so
	// cmd/screencensus's remainder section and every other reader answer
	// "is this screen witnessed" from one place rather than each
	// re-implementing the union.
	EffectiveGeometryTests []string
	// SharedWith names the other Screen (by String()) whose composer call
	// this entry's Composer is, when SharesGeometry is set on the registry
	// row. Empty for an entry with no shared composer.
	SharedWith string
	// GeometryGapNote is the registry row's own GeometryGapNote, passed
	// through unresolved.
	GeometryGapNote string
	Reason          string
}

// ScreenCensus returns every ui.Screen value's own registry row, in the
// fixed declaration order of flow.go's const block (screenCensusOrder),
// so cmd/screencensus's output is stable run to run.
func ScreenCensus() []ScreenCensusEntry {
	out := make([]ScreenCensusEntry, 0, len(screenCensusOrder))
	for _, s := range screenCensusOrder {
		e := screenRegistry[s]
		effective := append([]string(nil), e.GeometryTests...)
		sharedWith := ""
		if e.SharesGeometry {
			sharedWith = e.SharesGeometryWith.String()
			shared := screenRegistry[e.SharesGeometryWith].GeometryTests
			effective = append(append([]string(nil), shared...), effective...)
		}
		out = append(out, ScreenCensusEntry{
			Screen:                 s,
			Name:                   s.String(),
			Composer:               e.Composer,
			Test:                   e.Test,
			GeometryTests:          e.GeometryTests,
			EffectiveGeometryTests: effective,
			SharedWith:             sharedWith,
			GeometryGapNote:        e.GeometryGapNote,
			Reason:                 e.Reason,
		})
	}
	return out
}

// screenCensusOrder is flow.go's own declaration order, so ScreenCensus and
// cmd/screencensus's printed table read in the same order every run.
var screenCensusOrder = []Screen{
	ScreenMenu, ScreenPicker, ScreenMap, ScreenChargen, ScreenTown, ScreenGameMenu, ScreenLoad,
	ScreenDocuments,
	ScreenCutscene,
	ScreenSave,
	ScreenEnding,
	ScreenCutsceneLibrary,
	ScreenCredits,
}
