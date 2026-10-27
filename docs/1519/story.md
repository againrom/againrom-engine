# One town host

## Intent and authority

Both campaigns use one town screen and one scene adapter. Descriptions select
host rules. Campaign adapters read and change campaign state; they never
paint, lay out controls or route input. Existing room page producers remain.

Authority is the owner's "ROM1 is a reduced ROM2" ruling in
`pipeline/OWNER-RULINGS.md` and ranks 3 and 14 of
`pipeline/reviews/cross-game-kinds-8b7ee0e5.md` in the seat. This is an engine
refactor. It establishes no new ROM fact. Frames, World hashes and SAV bytes
must remain equal to the base in both languages.

## Contract

Base references below are engine observations at
`5f45c1047c1670e0239342cd16d720e77a511cef`, not ROM evidence. Paths without a
prefix are under `pkg/game/`. The table defines the target contract before
implementation. A later section records which parts have moved.

| Duty | Common contract | First-game value | Second-game value | Base sites |
|---|---|---|---|---|
| Named room and return after talk | Adapter supplies the named current room. The host keeps a talk backdrop and return room. Description page and dialogue rules select the transition. | square, tavern, shop, school, gates; talk returns to its building, gate talk to square | square, inn, shop; talk keeps the campaign room | `townscreen.go:27,540,798,1066`; `secondcampaign.go:14,376,463`; `secondtown.go:153` |
| Dialogue backdrop | A host dialogue rule selects the existing talk producer and the page beneath it. | Building page; a gate conversation keeps square | Current room remains visible | `townscreen.go:533,540,600,606`; `secondcampaign.go:435`; `secondtown.go:217` |
| Page entry and exit | Description `rooms[].enter` and `exit` select host rules; page reset policy is data. | Existing tavern, shop, school and gates steps; retain page scenes | Existing inn and shop steps; reset a room scene on each new entry | `towns/rom1.json` room steps; `towns/rom2-town1.json` room steps; `secondtown.go:245` |
| Scene art | One named-room scene host asks the existing art library for the description's scene and page art. Resource keys remain description data. | Existing square cache and tavern/shop/school page caches | Current town's scene and page art by room name | `townscenehost.go:62`; `townart.go:100`; `secondtown.go:174,636` |
| Audio scope and speech channel | Description audio policy selects room retention, channel routing and silent-device admission. One host owns scopes and loop delivery. | Effects or speech according to SoundRequest; page requests require sound device; square may open its scope silently | Effects channel for scene sounds; no scope without sound device; retain scope until room changes or leaves | `roomsound.go:22`; `townscenehost.go:139,173,202`; `secondtown.go:677,696,714,746` |
| Row source | Description host row rule selects chapter hints or hotspot rows; room page selects its existing offer/seat producer. Labels and hint rules are data. | Four square rows with chapter offer counts; tavern and school rows; shop/gates/talk have no rows | Square rows from hotspots; inn rows from seats followed by EXIT; shop has no rows; retain uncomposed fallback rows | `townscreen.go:898,917,958,994`; `secondcampaign.go:293`; `secondtown.go:376` |
| Save admission | Adapter supplies campaign save admission; the description's save condition and host travel state gate it. | Open town and no homeward journey | Valid town save point and no open dialogue | `townscreen.go:482,488`; `secondcampaign.go:475,477`; `towns/rom1.json` save |
| Leave action | A host leave rule consumes adapter destinations. The adapter performs campaign changes; the host opens the retained map/mission builder. | gates room opens existing world map over available missions/towns | gate changes campaign location, then existing map or direct mission route | `townscreen.go:1020`; `townsquare.go:188`; `worldmap.go:392`; `secondcampaign.go:67`; `secondworldmap.go:159` |
| Music inside rooms and pages | Description music rule chooses room scene or retained town track; adapter supplies current town ID and selected member class. | Square track; shop/tavern/school scenes; school prefers the selected member's class; gates use campaign music | Every town page keeps its town track; no town uses campaign music | `townscreen.go:555`; `townsquare.go:300`; `secondcampaign.go:502,512` |
| Campaign choice of screen | One FrontEnd town-screen constructor binds the campaign's state adapter and profile data. | Chapter state and chapter-keys offers | Scenario-bank state and destinations | `townscreen.go:506,509,1137`; `campaignfirst.go:19`; `campaignsecond.go:22`; `secondcampaign.go:250` |
| Catalog binding | Every supported catalog hook set uses one vocabulary and one host hook table. The offer source chooses campaign state, never a screen builder. | chapters / chapter-keys; required tavern, shop, school, gates | destinations / scenario-bank; named square and pages | `towncatalog.go:15,35,42,129`; `townsquare.go:65,155`; `secondtown.go:50,77` |

## As built

`FrontEnd.TownScreen` constructs `townScreen` for both campaigns. The host
owns the room/page state machine, rows, header, footer, input, save admission,
music and scene seams. `secondCampaignScreen` is removed. `townRoomView` is
the square/page scene adapter and `townSceneHost` is its concrete host.

`town.HostSpec` selects the row, header, footer, dialogue, tip, page-cache,
art, audio, shop and leave rules. It also selects initial-town entry,
screen caching, eager square creation, music-track source and optional
click/speech channels. Every shipped description supplies these values.
Hotspots supply fallback row labels and chapter hint names.
The host key `tip-lifecycle` selects `retained` or `entry`; these values
preserve each description's existing tip lifecycle.

`chapterTownCampaign` and `destinationTownCampaign` implement
`townCampaignAdapter`. They supply campaign state: the named room, town ID,
catalog, conditions, values, offer counts, save point and destinations.
They change campaign state on departure or campaign start. The retained
page producers consume their existing chapter or scenario-bank state.
Neither adapter paints, lays out controls or routes input.

Both catalog hook sets use one sorted, deduplicated `townVocabulary` and
one typed `townHooks` table. Hook metadata declares a required offer source.
Catalog validation rejects a page, hook or host profile whose required
campaign state is absent. The existing catalog selects the adapter's state
source and the description selects host rules.

`town.RoomSpec.MusicRule` and `townRoomMusic` select room music. The first
description supplies four room scenes and the school's member-class rule.
An absent room rule inherits town music, as every second-game page does.
`townScreen` owns audio scopes and loop delivery; description values retain
the first game's speech channel and silent-square scope admission.

The retained square/page caches, raw wildlife draw admission, room wire
values and save inputs stay as at base. Deferred talk/map pages hold a
named host pointer; they cannot inherit the town screen's full interface.
The static guard in `internal/archtest/townhost.go` checks one input host,
its scene/music seams, one square adapter and this promotion boundary.

## Acceptance evidence

### Prior acceptance at 74b0c2ae

The common host remains at `d0d04cd49a1bf57efe4025a1792bf67029354d7f`.
Reconciliation fetches main at `a046b32f7a0e0bfcf62d3daec8f5dc090ab9b5e9`.
The only merge conflict is the storyguard baseline. Both independent falls
are retained. All final witnesses and gates use the resolved merged source
tree `7d8872c2198e9a5b5a384be8e1f6ad803bf4547d`; only this proof document
changes afterward. The merge and proof form one local acceptance commit.

`TestReleaseRouteWitness` runs headless on each install. Both first-game
runs also walk the 14 kit/EN town saves. `compare-routes.py` compares raw
TSV and PNG bytes with the corresponding base runs, without normalization.

| Install | TSV files | Data rows | PNG files | Test / wrapper seconds | Raw comparison |
|---|---:|---:|---:|---:|---|
| en | 15 | 1,748 | 1,748 | 283.73 / 317 | Equal |
| rom2-en | 1 | 285 | 285 | 81.73 / 85 | Equal |
| ru | 15 | 1,748 | 1,748 | 277.76 / 281 | Equal |
| rom2-ru | 1 | 285 | 285 | 63.40 / 66 | Equal |

The four `final-*-compare.json` receipts in the notes directory record
32 TSV files, 4,066 rows and 4,066 PNG files in total. Each has zero missing,
extra or different files. These witnesses compare the route's recorded
state and frames; they do not record complete World hashes or SAV bytes.
The base ROM2 EN test passes, but its original wrapper exits 2 after a live
script edit. Its complete artifacts are retained; that wrapper is not PASS.

Two profile loss controls run before reconciliation. Changing the first
description's tavern music scene to `shop` fails
`TestTownMusicFollowsVisibleRoomAndSelectedSchoolClass` in `tavern` and
`tavern_dialogue`. Changing town 1's open-gate hook from `navigate` to
`gate-closed` fails `TestSecondCampaignInputUnlockCancelAndFailedEntry`:
leaving town 1 does not open the map. Each wrapper takes 29 seconds and
returns 1. Each changed file is restored only from the host commit with
`git restore --source`. Final focused tests pass both restored routes.

The four release sets run serially, one invocation per install, without
retries. The actual selected names equal main's 1,483 manifest entries on
every root, with zero missing or extra names. Counts below are top-level
test outcomes, read from raw root JSONL by `summarize-release.py`.

| Install | PASS | FAIL | SKIP | Script exit | Seconds |
|---|---:|---:|---:|---:|---:|
| en | 1,364 | 0 | 119 | 1 | 541 |
| ru | 1,362 | 0 | 121 | 1 | 470 |
| rom2-en | 221 | 4 | 1,258 | 1 | 212 |
| rom2-ru | 221 | 4 | 1,258 | 1 | 235 |

EN and RU have only the expected variable-gated skip,
`TestReleaseActivePauseRenderedWorld`. Their other 118 and 120 top-level
skips lack a subject on that install. ROM2 fails exactly
`TestReleaseSecondMissionSaveLossControls`,
`TestReleaseSecondMissionTwentySaveLossControls`,
`TestReleaseSecondTownSaveLossControls` and
`TestReleaseSecondGameFirstMovieAgainstOracles`. The three save controls
require `AGAINROM_FIRST_ASSETS`; the movie control requires a separately
stamped 386 helper. The brief permits these four failures. They are not
reported as passing tests. All raw attempts and summaries remain in notes.

| Final gate | Result | Wrapper seconds |
|---|---|---:|
| Focused town/catalog/campaign, reset and failed-restore tests | PASS | 5 |
| `scripts/check-fast.sh` | PASS | 57 |
| Explicit archtest/gatedtests/storyguard/divledger command | FAIL: one archtest | 24 |
| `go test -trimpath -count=1 ./...`, run once | FAIL: the same archtest only | 277 |
| `gofmt -l` | PASS: empty output | 2 |
| `go vet` on game, town, ui, camera, archtest and storyguard | PASS | 45 |
| Working and staged `git diff --check` | PASS | less than 1 |
| `scripts/check-no-game-assets.sh` | PASS | less than 1 |
| Darwin arm64, `CGO_ENABLED=0 go build ./...` | PASS | 64 |
| Darwin amd64, `CGO_ENABLED=0 go build ./...` | PASS | 52 |

The resolved tree measures CommentBytes 9,712,160, barestorynumber 1,705
and longcommentgroups 1,694. Every other storyguard counter is unchanged.
Relative to main, these fall by 8,318, one and one respectively. The host
step alone removed 7,952 comment bytes. Composition remains 79 fields and
15 coordinators. The merge baseline retains the measured falls. The lane
diff against main changes no golden, hash, testdata, module, version,
version resource, changelog or knowledge pin. Main's version and resource
changes are inherited without lane edits.

### Schema correction

The seat correction merges main at
`f43dc35b78870b67d5f73cd83f64db38bbf6c3d4`, keeping both hotfixes and the
host. The only conflict is CommentBytes. Its two independent deltas give
9,715,207; barestorynumber stays 1,705 and longcommentgroups stays 1,694.

The host JSON tag, the four shipped descriptions and the validator's rule
name use `tip-lifecycle`. Its values remain `retained` and `entry`. This
removes the schema-key collision with `TestLiveTipsLiveOnlyInTheTipTables`.
The guard and its probes are unchanged. All required correction gates pass;
the prior acceptance blocker is closed.

| Correction gate | Result | Wrapper seconds |
|---|---|---:|
| Focused town/catalog/campaign, reset and failed-restore tests | PASS | 42 |
| Explicit archtest/gatedtests/storyguard/divledger command | PASS | 29 |
| `scripts/check-fast.sh`, including gofmt and asset scan | PASS | 59 |
| `go test -trimpath -count=1 ./...`, run once | PASS | 228 |
| Storyguard measurement after merge | PASS | 5 |

Receipts are `correction-focused.log`, `correction-guards.log`,
`correction-fast.log`, `correction-ordinary.log` and
`correction-measure.log` in the notes directory. The full ordinary run and
the explicit guards both pass `TestLiveTipsLiveOnlyInTheTipTables`.
The measured baseline matches every staged count. Working and staged diff
checks pass. Only this proof document changes after the gates.

Before the schema edit, the merge changes no production file under
`pkg/game/town*`, `pkg/town`, `pkg/ui/town.go` or `pkg/ui/app.go`. Under the
seat's correction scope, route witnesses and release sets are not repeated.
The previous four-root receipts remain tied to their tested revision.

## Open debt

Windowed pause presentation and the four full-chain-input tests remain
outside this headless acceptance. Complete cross-revision World-hash and
SAV-byte equality is not independently measured by the route instrument.
No broader equality claim follows from the recorded route state.

The tavern/inn page, inn offers, hero pane, talk box, room tip and world-map
opening keep their current producers. `Rows`, `Choose`, `paneSquareView`,
`boundTalkSource`, `hostTipView` and `mapPage` call them. Their later
unification remains open. Existing room return, dialogue backdrop, page/tip
lifecycle, audio scope, rows, music, caching and initial-entry differences
remain profile data. This refactor does not decide their ROM authority.
The town Game Options row (`DIV-1269`) and timeless-town policy remain.
No new divergence is introduced. DIV-4102 through DIV-4105 remain unused.
