# Mod authoring in the map editor

Status: implemented; release validation pending. Base: `05289791cc3d281a8ed0dbd59e132d06aab6d593`.

## Approved scope

One W window owns mod identity, mission declarations, explicit After win links,
named companions, party rows, Save mod, and Publish. List reordering does not
change links. The first Save of a handwritten mod creates a complete same-ID
copy under another parent. Publish asks before saving changes or replacing RES.
The output folder is selected once per session. ROM1 EN/RU are in scope, retaining
replacement missions and imported declarations. Tiles and character creation are deferred.

## Builders and slices

The mission writer is `mod.WriteMissions`; it uses the existing TOML parser and
keeps unchanged declaration source. W and F5 use this writer. Party optionals
keep absence distinct from Empty. Existing workspace history deep-copies them.
Folder saving extends `saveWorkspacePackage` and `stageMissionPackage`.
Mission rows use `MissionData` and `Picker`; field rows and windows use
`objectFieldRows`, `popupSpec`, `layoutPopup`, and `drawPopup`. Party resolution
uses `guestParty`. Publish uses `game.PackMod` and `mod.Pack`.

1. Retain a handwritten project's declarations, party, files, and languages;
   save a complete copy and reopen it.
2. Manage mission rows and project history in W.
3. Edit explicit links, named companions, and party equipment rows.
4. Publish through the existing packer and read the RES through the normal loader.
5. Extend the release witnesses for published mission start, continuation, and SAV.

## Evidence

`TestEditorImportedPartySurvivesWorkspaceProjection` reproduced an empty Party
and guest 0 through W on the base. Mission-writer fixtures cover source retention,
one-value edits, comments, localization templates, optional equipment, reordering,
and manifest metadata. Fixtures are not install validation.

`TestEditorCopiesTheWholeHandwrittenModWithoutChangingAnySourceFile` opens a
two-link party fixture, saves a same-ID copy, compares every original file byte,
edits its purse, reloads both declarations, and refuses an external file change
without writing. The working copy retains hidden files and both language maps.
Existing page-preservation fixtures reopen after external fixture edits, as a
real project must before saving over externally changed input.

`TestEditorModMissionOrderAndHistorySurviveMapSwitches` adds a second map,
reorders it without changing After win, switches with pending changes, and
checks Undo/Redo across the switch and after Save. W uses the shared searchable
field rows and popup layout. Its two columns scroll independently. Fixture
tests cover EN/RU labels, canvas blocking, named selection, Ctrl+S, reverse
focus, and the boundary between text undo and project undo.

F5 retains the original mod ID, declaration list, entry scripts, character
definitions and files in the private test-play folder. Save and F5 share
`stageMissionPackage` and `installMissionPackage`. The staged tree passes
`modrt.Load`; an installed inspector also applies `SetMods` and `SetModData`
on an isolated table and archive fork before any destination is replaced.
Fixtures check a fake launch, unchanged source/checkpoint, foreign private
files, unresolved-party refusal, named custom characters, member order, the
eight-member limit and Template versus Empty after Save and reopen.

Publish calls `game.PackMod`, with an explicit output folder kept for the
editor session. Unsaved changes require Save and publish. An existing RES
requires Replace in a second confirmation. Confirmations hold focus on their
two buttons. Source changes, missing maps, broken links, an unresolved party,
install targets, duplicate discovery IDs and scripts omitting edited missions
refuse before replacement. The success receipt names the full path and mission
count. Headless fixtures compare published bytes with `mod.Pack` and load the
RES through `mod.Resolve` and `modrt.Load`.

`TestReleaseEditorPublishedModChainSAV` creates a complete working copy through
W, publishes it through W, then uses the existing mission-chain witness for
start, continuation, town return and cold SAV/LOAD. The complete-scenario
witness also reads its published RES. Owner frames include W, party equipment
and Replace at 640 by 480. These release witnesses are compiled but not run here.
Unsupported default script/dependency contexts have a visible read-only marker.
Add open map also uses the existing page-tree exporter, preserving common,
EN and RU pages plus pending edits for both a first project and an attached
project. Shared-ALM declarations resolve their page root after selecting the
declaration. Independent fixtures cover both boundaries and source immutability.

Page edits retain the declared `texts` path, including a language token inside
the path. `syncModDocument` writes into the resolved declared folder unless the
map's own page supplies that name; that edit goes into `<stem>.text/<lang>/`,
using `mapTextFolder`. A mission without `texts` also uses that folder.
Mission selection restores pending pages in the loader's priority order.
`TestEditorPageEditKeepsLanguageInsideTextsPath` and
`TestEditorPageEditReachesMapSiblingPages` reproduce the review failures on
`5384b788`. They check edits from both EN and RU editors through `modrt.Load`,
the private F5 package with a recording launch, Save and editor reopen. They
also check unchanged declarations, the other language and original source.

Removing an enabled guest reconnects its successors after confirmation.
Removing a replacement or disabled declaration keeps the campaign link and
does not ask to reconnect it. `TestEditorRemovingOnlyGuestMissionsReconnectsSuccessors`
reproduces the replacement failure and covers disabled document state, guest
confirmation and a guest without successors. Disabled state is constructed in
the fixture; a standalone off declaration has no mod layer to disable. The
test saves and reloads each remaining successor with After win still 20.

## Release witnesses

Run for ROM1 EN and RU at the next release, not in this lane:
`TestReleaseMapEditorWorkspaceRoutesAndTexts`,
`TestReleaseMapEditorCompleteScenarioPackageSAV`,
`TestReleaseModMissionChainPlaysAndReturns`,
`TestReleaseEditorPublishedModChainSAV`,
`TestReleaseGuestPartyBuildsCharactersAtTheirLevels`,
`TestReleasePackedModPlaysAsItsFolder`, and `TestReleaseMapEditorOwnerFrames`.

## Release validation and deferred work

Install validation and installed-font fit remain release evidence. Dependency sets
and script/settings contexts not reproducible as one project refuse without
altering source. Disabled declarations remain read-only. New tiles and new
character definitions are deferred. No original-game fact is asserted here;
read pinned tile-capacity claims before a separate tile design.
