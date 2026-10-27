# Native mission-menu windows

## Intent and authority

The owner requires the second game's mission menu and all windows hosted there
to use native pixels, centred over a darker mission without a blue tint. Save
and Load retain the mission behind them. Main menu, town and inn keep their
640x480 presentation; the first game stays unchanged.

The pinned claim tool read R2-SESSION-227, R2-ASSET-150, R2-ENGINE-570,
R2-ENGINE-571, R2-ENGINE-572, R2-ENGINE-438, R2-ENGINE-440, R2-ENGINE-441,
R2-ASSET-141, SESS-VIEW-028, MENU-077, MENU-121 and MENU-127. R2-SESSION-227 establishes the
windows and their transitions. EN-PART3, Mission game menu, supplies the size,
place and darkened map: p3-m01-menu.jpg, p3-m02-save.jpg, p3-m08-delete.jpg,
p3-m13-load.jpg, p3-m15-options.jpg, p3-m17-sound.jpg, p3-m19-quest.jpg,
p3-m21-endquest.jpg and p3-m24-esc.jpg under review/rom2-compare/work.
The snap arithmetic is MENU-077's first-game claim. Applying the existing
builder to the second game's panels follows the owner ruling and crops, not
a claim that the second game's constructor has been recovered.

## As built

GameMenuProfile supplies NativeMissionWindows and MissionBackdrop through
secondGameFiles. The first profile remains the zero value. One host,
App.drawGameMenuOverMap, uploads GameMenuPanel and uses missionMenuPlacement
for both the blit and input mapping. Viewer still owns the map and its dim.
The same composePage and composeSaveScreen build Save, Load and their Delete
state; their backing is transparent only on this mission host. Load retains
the popup and updates the mission pointer through holdMapUnderMenu. Declarative mod pages opened there use the same host and their existing composeModScreen and modPanel.

The existing newDialogGeometry supplies the menu root's 296x296 frame
(288x288 body) and the confirmation's 296x232 frame (288x224 body). Save and
Load retain a 476x344 body and an 8-pixel shadow: the centred frame is
484x352, and its body at 1024x768 is (270,208)-(746,552). Options, sound,
objectives and remaining menu pages use their existing rectangle builders.
Their retained body sizes are 480x480 (Game Options), 480x352 (Sound Options)
and 480x220 (Quest Objectives); full page-layout equality remains unestablished
under DIV-2726.
The complete frame including shadow is centred. Scale is 1 in mission
coordinates; a resized window receives the mission's existing final fit.

The viewer's existing popup backdrop takes black alpha 64/255 while the
mission menu is raised. It supersedes a retained dialogue lookup. RGB
(160,120,80) becomes (120,90,60). DIV-4146 records the unestablished exact
original blend. DIV-210, DIV-211, DIV-2726 and DIV-3078 describe this build.
DIV-4147 is unused.

## Proof

Base: d77853a03767e51897ac9939fb0249f188db7e7a. The required fetch and merge
of origin/main reported Already up to date before final verification.

The ordinary tests check centred native rectangles in 1024x768, 1920x1080
and 640x480 windows, centres of menu, options, sound and objective controls,
Save/Load and both Delete states, transparent backing, pointer updates,
keyboard focus, untinted dim and unchanged town rectangles. The static
guard TestMissionMenuLayersUseOneHost pins one mission-menu blit owner.

The six focused tests pass. The placement loss control fails the two expected
geometry/backing assertions at the former scale 1.6; the backdrop loss control
fails the expected color assertion. These are assertion failures, not build failures.

| Root | Recorded PNGs | Byte-equal | Changed | State-column differences |
|---|---:|---:|---:|---:|
| rom2-en | 350 | 324 | 26 | 0 |
| rom2-ru | 350 | 324 | 26 | 0 |
| en | 282 | 282 | 0 | 0 |
| ru | 282 | 282 | 0 | 0 |

Each first-game route has 283 rows. Row 283 records that AGAINROM_ROUTE_SAVES
names no additional town-save set; it has no PNG on either side. No frame
equality is claimed for that row. PNG bytes are compared without normalization.

The same 26 steps change on both second-game roots: 287, 288, 289, 291, 292, 293, 294, 295, 297, 298, 299, 300, 301, 303, 304, 306, 307, 309, 310, 312, 313, 315, 316, 318, 348, 349.
They are root-menu visits, Save/name/list/Delete states, Load, Game Options,
Sound Options, Quest Objectives, End Quest and the final mission-exit menu.
Every other recorded PNG stays byte-equal, including main menu, town and inn.
The per-step state, control and matching original crop are recorded in
review/story1522-rom2-menu-native/changed-frames.md and comparison.html.

| Root | Base PASS/FAIL/SKIP | Head PASS/FAIL/SKIP | Head seconds |
|---|---|---|---:|
| rom2-en | 46/0/327 | 46/0/327 | 97 |
| rom2-ru | 46/0/327 | 46/0/327 | 103 |
| en | 338/0/35 | 338/0/35 | 625 |
| ru | 337/0/36 | 337/0/36 | 619 |

Each selected release population contains 373 top-level tests across pkg/game
and pkg/ui. TestReleaseSecondTownNamedSaveContinuation passes in the whole set
on both second-game roots at base and head; the brief's known order-dependent
failure was not reproduced in these runs.

The existing Empty Slot pixel assertion now samples GameMenuPanel in native
page coordinates, while its saved artifact remains the actual placed HeadlessFrame.
The old assertion sampled a shifted window region and failed on both head ROM2
roots, while it passed at base. The corrected test passes alone on both roots
and in both complete selected sets. Production blobs did not change for this
fixture adjustment. The first-game runs skip that test before the changed code.

Base EN was measured with the frozen clean-base test binaries and each package's
working directory. Their hashes are retained in freeze.log and PROGRESS.md.
Invalid runner attempts are excluded; their logs remain in the notes directory.

| Final gate | Seconds | Result |
|---|---:|---|
| focused | 2 | PASS |
| ordinary | 183 | PASS |
| fast | 60 | PASS |
| format | 2 | PASS |
| vet | 6 | PASS |
| diff | 0 | PASS |
| staged-diff | 0 | PASS |
| assets | 0 | PASS |
| build | 28 | PASS |
| mac-amd64 | 28 | PASS |
| mac-arm64 | 26 | PASS |
| internal | 19 | PASS |
| div-claims | 1 | PASS |

The macOS builds use CGO_ENABLED=0 for amd64 and arm64. The touched-package vet
covers pkg/ui, pkg/game and internal/archtest. The internal gate covers archtest,
gatedtests, storyguard and divledger. Every Go command uses the prescribed cache,
GOMAXPROCS=4, GOFLAGS=-p=2 and the notes temporary directory.
The exact final Go blob set is recorded in candidate-go-blobs.tsv.

The storyguard baseline falls from 1694 to 1693 long comment groups and
from 9738875 to 9736754 comment bytes because obsolete host and headless
contract comments were replaced with the current contract. No guard rises.

## Open debt

All runs are headless by instruction. The route witness records menu layers,
not a full GPU mission behind them. Geometry, transparent backing and the
production backdrop decision are checked; actual GPU blending and final
text smoothing require the EN and RU release witness with App.Draw readback
when window execution is authorized: TestReleaseRouteWitness on en, ru,
rom2-en and rom2-ru through App.Draw/readback. These GPU runs are NOT RUN.
No original program was run.

DIV-4130 retains the pre-existing Delete question layout difference: its
text is inside the page, rather than a separate Yes/No box. Option control
content and other existing page differences are outside this change.


## Whole-frame wash and native mod rows

The menu wash covers the whole logical mission frame, including the right
column, open pack or spellbook bar and bottom band. The menu layer, tooltip
and pointer remain above it. noticeBackdropOf keeps the camera rectangle
for the first game's zero profile and all other notices. DIV-4146 states
this extent beside the alpha reading. The original m1.24-key-esc.png and
m1.25-key-esc-again.png pair, measured in the sole review, establishes the
HUD and bottom-band extent; the exact blend formula remains Unknown.

gameMenuPanelRectFor centres the grown native panel, including its shadow,
in the 640x480 canvas. gameMenuRowRect takes that panel for both paint and
hit tests. fitMenuRowLabel uses the profile's row width, with textfit's end
mark. Non-native panel coordinates retain their previous formula.

The ordinary test checks zero through five mod rows, the painted bounds,
shadow, every enabled row's drawn-centre hit and the last mod page opening.
It uses synthetic rows beyond the existing two-entry registration limit;
that limit is unchanged. A long label fits the native row and ends in three
full stops. The wash test checks the complete 1024x768 rectangle and neutral
darkening at map, right-column, open-pack and bottom-band pixels.

Three final-source loss overlays restore only the old wash extent, downward
growth or fixed label width. Each compiles and fails its matching assertion.
Storyguard comment bytes fall 9736754 to 9736430 because obsolete rectangle
comments are shortened; all other measured counts remain unchanged.


### Correction receipts

Baseline: e3d8dd9825ac4e4226889d70d969c047d7d708c8. Fetch and merge of
origin/main again reported Already up to date at d77853a0. The correction
reuses that candidate's recorded route PNGs; its 23 Go blob IDs were verified
against the committed tree. Each PNG pair is compared as raw bytes and SHA-256.

| Route root | Equal PNGs | Changed steps | State-column differences | Seconds |
|---|---:|---:|---:|---:|
| en | 282 | 0 | 0 | 47 |
| rom2-en | 350 | 0 | 0 | 103 |

No step changed against the returned candidate. The EN route's row 283 has
no PNG on either side. ROM2 menu layers without mod rows are unchanged;
the full-frame wash is not present in the headless layer witness. The
ordinary pixel test, not PNG equality, proves its rectangle and alpha.

| Selected release root | PASS | FAIL | SKIP | Seconds |
|---|---:|---:|---:|---:|
| rom2-en | 46 | 0 | 327 | 90 |
| rom2-ru | 46 | 0 | 327 | 98 |

The selection is TestRelease.*(GameMenu|Menu|Save|Load|Delete|Quest|Options|Abort)
in pkg/game and pkg/ui. All three NamedSaveContinuation tests pass in each
whole set. The review's order-dependent failures were not reproduced; no
isolated rerun was needed. Roots ran sequentially with the prescribed runner.

| Correction gate | Seconds | Result |
|---|---:|---|
| focused | 1 | PASS |
| ordinary | 166 | PASS |
| fast | 62 | PASS |
| format | 2 | PASS |
| vet | 6 | PASS |
| diff | 0 | PASS |
| staged-diff | 0 | PASS |
| assets | 0 | PASS |
| build | 26 | PASS |
| mac-amd64 | 39 | PASS |
| mac-arm64 | 30 | PASS |
| internal | 18 | PASS |
| div-claims | 0 | PASS |

The seven focused tests pass. The final-source extent, position and label
loss controls fail the intended assertions in 10, 9 and 10 seconds. No
production or test source changed after those controls and gates. Twelve
changed Go blob IDs are retained in correction-go-blobs.json.

GPU blending/readback and the original program remain NOT RUN. The exact
original blend, existing Delete layout DIV-4130 and existing options/page
layout debt remain open. No second host, page composer or game branch was
added. DIV-4147 stays unused. The registration limit is unchanged.
