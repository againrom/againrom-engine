# One town state and one town art bundle

## Intent

R36 has one ID-keyed town state builder and one description-to-art bundle for
both games. Offers, bank and shop contents, travel, SAV bytes, art sources,
read order and cache lifetimes stay unchanged. Architecture work counts zero
toward the release.

## Authority

Owner direction requires one builder per kind, with the first game expressed
as reduced profile data. The preservation reference is engine
`0427e8c489e425d2270b34a72b97fb1c8c36c7bd`. This comparison establishes engine
preservation, not a new claim about either original game.

## As-built behaviour

`newTownState` builds every catalog row. Chapter-key rows retain their consumed
offers and mercenary pools. Scenario-bank rows refer to the existing
`secondCampaign` bank/shop adapter. `Town.townState` and `Town.others` hold both
profiles by ID. `resetTownStates` binds the rows at the first bank read and the Town keeps
its own row, so the second game's `Town.townID()` stays the chapter town's. Bank and shop remain campaign-wide.
`newSecondCampaign` delegates construction to `newTownState`; LOAD fills that
same adapter from the existing captured fields. Destination services bind the
catalog on first use, preserving new-game admission timing.

The bank adapter is runtime state. `townsRecord` emits no chapter-town session
record for it. The existing SAV producer and field schema remain authoritative.
The `endInnTalk` call site is unchanged.

`newTownArtLoads` constructs the deferred typed surfaces and the manifest
scene/page resolver and the inn patron frames (`townArtLoads.patron`, bound
from the room manifest by class, failures cached as for `room`). `installShare`, archive and mod lookups, the presentation
shop cache and `townScreen.roomArt` all use that constructor. Manifest rooms
retain scene-before-page reads and cache failures. No new game-identity branch
selects an art loader.

| Caller | Source | Lifetime |
|---|---|---|
| Startup square, tavern and school | Install art source | Install share |
| Other chapter towns | Install art source | Archives and description |
| Mod town descriptions | Mod files and install fallback | Town book and description |
| Startup shop | Install art source | Presentation |
| Manifest rooms | Install art library | Town screen and description |

Hire, party identity, offer-list construction and shop rules keep their existing
builders. No new package dependency is introduced.

## Proof

The isolated record commit `6f93a3fd7ac755a4ea04e2ba881cea88f0af8c86`
freezes 30 hashes of state, complete SAV output and deferred art requests.
`TestTownStateAndArtRecord` retains those hashes. Incrementing bank slot 770
fails eight of them. Its strengthened screen-bank probe also passes on the
base archive.

`TestTownRoomArtRecord` freezes 14 additional room request/error records from
the base archive. `TestTownRoomArtReadsSceneBeforePage` uses successful
synthetic pictures for all four descriptions. Both pass on the base archive
and the changed tree. `TestTownBankAdapterKeepsEveryIDAndTheCampaignBank`
checks row identity, shared bank/shop data and rebinding after LOAD.

The focused ordinary selection passes 520 top-level tests with five skips,
excluding the two base failures below. Existing assertions are unchanged;
three constructor call sites in tests receive the new adapter argument.
The build, `TestLiveTreeClean`, and the town-state guard with `Dag` pass.
`CheckTownState` checks allocations, aliases, zero variables, duplicate storage
shapes, art-function aliases and scene/page reads. Its 25 planted-shape cases
name the offending source. Existing leaf decoders are explicit guard entries.

The final candidate requires the fast gate and the town/travel/shop/bank release
selection on EN and RU roots of both games. Additional exact tests that reach
the town services are listed with their routes in the lane handoff. That
handoff holds the final timings and all four install receipts. Synthetic
records alone do not establish installed-data validation.

## Open debt

`TestEditorModUnitNameTravelsAsAPageEdit` fails at its temporary filesystem
operation with `Access is denied` on the unchanged base. It is deferred to the
seat and excluded from further focused runs. Publication and the sole review remain seat-owned.
