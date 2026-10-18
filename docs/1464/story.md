# Map-owned message pages

## Intent and authority

Owner direction: editor-scripts census plan item 3 and decision 1. Pages live
beside the ALM, per language, and travel with Save As and W. The editor is
this project's tool. The pinned knowledge revision stays unchanged.

`MISSION-TEXT-005` (High / Medium) establishes the installed message path,
the numbered page and silence for an absent page. `TRIG-MSG-023` (High /
Medium) establishes the message producer's packet broadcast and bounds its
no-state-write observation to that producer path. DIV-3633 records the
owner-directed page storage. No new simulation rule follows from it.

## As built

`mod.Mission.ReadTexts` is the shared bounded file reader. Priority is per
file: `<stem>.text/<lang>/`, `<stem>.text/`, declared mod `texts`, the parent
of a declared `/en` or `/ru` folder, then installed pages of a campaign or
replacing mission. Other language folders are not substituted. Empty files
count as present. The mod reader's physical-path and 1 MiB file bounds
apply, with at most 64 files across the selected folders.

Editor Open prepares one workspace. X, report, Test play and W use its
pages. Test play stages the current map and those pages through the same
mission package builder used by W. Mod loading uses ReadTexts before its
existing install-address overlay. An installed campaign opened without a
mod keeps its installed pages.

Save As and W use `exportPageTree`. It copies all source language folders
and unlanguaged pages byte for byte, then overlays changed workspace pages
in the running language's folder. Unchanged unlanguaged pages stay common.
Source pages absent from a tree, such as installed campaign pages, are
materialized in the running language's folder.

Save As writes the existing ALM serialization and that tree into a new map
and new `<stem>.text/` beside it. An existing page folder refuses even when
the workspace has no pages. Page staging and the map share the physical
parent fence. A page-publication failure removes only the newly created map.
The page checkpoint leaves changed mod route settings dirty because they
require W.

W writes the tree into `mission/text/` and declares
`texts = "mission/text/{lang}"`. An owned package preserves its recorded
unlanguaged and other-language pages. The reader fills missing page names
from the declared language folder's parent, ignoring non-page files there.
The declared folder retains its existing filename validation. Reader errors
name the offending file and have one `texts:` prefix. Missing-page reports
name a map or mod as applicable. The report cache includes page contents
from the sidecar tree and overlays workspace changes on the fresh disk view.
ALM and SAV formats, message parameter fields and version resources stay
unchanged. Page prompt words use pkg/words in EN and RU.

## Proof

`TestMapPagesSaveAsCarriesPages` fails with the base SaveAs function's
changed-workspace refusal and passes with the page writer. It compares ALM
bytes, saved page bytes, cold editor Open and the package builder.
`TestMapPagesSourceOrder` tests EN and RU, language and unlanguaged pages,
the absence of both, and the campaign install arm. A page in the other
language is present as a loss control.

`TestMissionPagesLanguageAndLegacyFallback` tests map-side pages, declared
language pages, the legacy folder alone and a missing declared folder.
`TestMapPagesWriteFenceAndCollisions` checks unchanged install and collision
trees, with an outside-install save as the loss control.
`TestMapPagesSaveAsKeepsUnsavedPackageRoute` checks the route checkpoint.
`TestMapPagesPackagePreservesOtherLanguage` updates EN and checks that a
recorded RU page remains byte-identical.

`TestReleaseMapEditorLooseMapPages` runs on each lawful EN/RU install. It
opens a loose map and its own page, checks a missing-page report control,
saves and reopens, stages Test play with a recording launch, and steps the
resulting mission headlessly until its authored message is announced.
It compares the announcement with a consumer-free world's hash, compares
SAV bytes before/after opening the dialogue, cold-loads the mod mission and
checks the page overlay, then checks W's page bytes against Save As.

Owner frames and crops, gate logs and exact revisions are external under
`review/story1464-map-pages/`; INDEX.md identifies the EN/RU X prompt frames.
The required final gate population is the full asset-free tree, touched
packages and vet, one whole EN/RU release set, all mission-stage scenarios,
the asset guard and divergence-claim scan. Existing announcement and
`TestReleaseMapEditor*` tests are unchanged; the new test is in the gated
population manifest. The command's oversized-page witness follows the new
language output path and checks refusal at Open, before any launch or write.
Its in-bound control still invokes the production command's `-check` route.

`TestMissionPagesLanguageTailCompatibility` covers nested and root-level
`{lang}` and literal `en` declarations on EN and RU, with the map in their parent and a common
fallback page. `TestMissionPagesErrorNamesFileOnce` checks the error path.
`TestMapPagesEmptySaveAsRefusesStaleTree` checks a page-less collision and
unchanged destination bytes. `TestMapPagesSaveAsPreservesLanguageTree`
checks an exact EN/RU/common tree copy and the RU reader's result.
`TestMapPagesLegacyPackagePreservesCommon` checks RU before/after W on EN
and the common page's ownership record.
`TestMapPagesPackageCopiesLooseTreeAndEdits` checks W from a loose map,
including the changed EN page and unchanged RU/common pages.
`TestMapPagesDiskEditsRefreshReport` checks a same-length disk change, a
change to the other language's tree, and a workspace override.

Removing non-current-language paths from the shared exporter makes
`TestMapPagesSaveAsPreservesLanguageTree` and
`TestMapPagesLegacyPackagePreservesCommon` fail. The external correction
receipts retain the reproduction, loss control and final gate results.

`TestReleaseActivePauseRenderedWorld` needs an offscreen GPU window. Its GPU
body is NOT RUN under this lane's no-window instruction. The whole release
runner still selects its parent test; a relative output path makes the
unmodified path validation refuse before the child launch. The resulting
named gate failure remains visible in EN/RU receipts. Other tests run with
the release environment unchanged.

The shortened report-cache comment lowers storyguard comment bytes from
9,199,131 to 9,199,001; its baseline moves in the same commit. Composition and command-literal counts
stay unchanged.

## Open debt

Workspace editing changes the running install's language and does not
translate pages. Both writers retain source languages and common fallback
pages; a language absent from both has no substituted page. Save As remains
a new-path operation. Multi-file publication cannot guarantee atomicity
against a process racing the save. Full release acceptance still needs the
GPU witness in a seat permitted to open its window.
