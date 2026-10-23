# Mission chains and guest parties as mod data

## Intent

A mod's guest mission may follow another guest mission's win with no town between. The missions
make one chain of any length; the chain ends in the town the first win opens, and the waiting party
comes back there when the last mission is won. A guest mission's party may be written as data: one
to eight characters of one character catalog (the install's `npc.reg` records and a mod's own
`[[character]]` keys), each with a level and equipment. The legacy `guest` + `chapter` form plays as
before. `npc.reg` is read through one catalog, and missions and characters merge through
`mod.Merge` like every other layered kind. The base campaign and the second game do not change.

## Authority

- Owner direction: the brief of this story (chains, data parties, one catalog each, persistence),
  and audit `c9cd0cd2` item 3 (one `npc.reg` reader).
- `REG-SCN-063` (active, amended), `SAV-1329` (promoted): an `AutoGetMission` other than -1
  requests that mission and skips completion and the city. A new record loads only when the
  request is above the current main mission. Otherwise the current main record remains and the
  party travels to its map object. The shipped key occurs once, `[Mission10] = 20`; missions 10
  and 20 bypass the town. The original has no guest mission and no chain of them.
- `HERO-JOIN-120` (active): a composed npc record takes server id `26 + selector + 4*floor(mission/40)`,
  four bands of forty missions. A party member's `level` is that band.
- `HERO-JOIN-123` (active): a hero has six skill levels and no overall level, so `level` sets no
  skill.
- `REG-NPC-058` (active): `npc.reg` holds 105 `npc<n>` records.
- Divergences in `docs/divergences/mods-missions.md`: `DIV-4001` (chains), `DIV-4002` (data parties)
  and `DIV-4003` (the stated human template default for records without DataBinID). `DIV-4004`
  remains unused and is returned.

## As built

### One npc.reg catalog

`data.LoadNPCCatalog` (`pkg/data/npccatalog.go`) parses every `npc<n>` section once and answers the
placement half (`Defs`), the portrait half (`Faces`) and the records (`Entries`).
`game.LoadNPCCatalog` replaces `LoadNPCDefs` and `LoadNPCFaces`; `mapload.Table.NPCCatalog` holds it.
The dialogue portraits, the map editor's speaker list (`pkg/game/mapmessagepages.go`), `classdump`
and `wearcheck` read it. Layer 0 of the character catalog is `mod.InstallCharacters`, one record
`npc<n>` per catalog record.

### Characters and missions merge as layers

`pkg/mod/kinds.go`: characters and missions are `LayerMerged`.

- `mod.MergeCharacters` merges catalog characters by `key` over layer 0. Two mods adding one key
  or setting one field require `load-after`; the later wins and the override is reported.
  `off = true` switches a mod's character off. Target edits retain every operation in load order,
  without requiring `load-after`, including overlapping families and rows. A later kind resets
  an earlier face before its own explicit face applies, preserving the base result.
- `mod.MergeMissions` merges by `number`; field `mission` replaces the whole record, so two mods
  declaring one number are refused unless ordered. `checkChains` then refuses two missions after one
  win (naming both mods, files and lines) and a cycle (naming its links).

### Chains

`MissionData.AfterWin(n)` is the guest mission after win `n`; `MissionData.Chain(m)` is the links from
the first to `m`. `checkGuest` (`pkg/game/modmission.go:235`) refuses a chain whose first link does
not follow a main mission that opens a town (the missing link), and a chain with two links in the
guest form. `finishGuestMission` (`pkg/game/modmission.go:589`) carries the party home through the
ordinary carry; when `AfterWin` names a next link, it builds that link's party with no town between
and keeps the stash; otherwise the stash comes back through the one town LOAD with the guest who
joins recorded beside it.

### Guest parties

`party = [{ character, level, weapon, shield, armour }]` names 1 to 8 characters. Every member is
built by `mapload.CharacterMember` (`pkg/mapload/joined.go:45`), the one builder of a persistent
party member: `CampaignNPCMember` and the town's `townCompanion` wrap it, through
`townInstall.character` (`pkg/game/campaignreturn.go:144`), which also names the member.

- `level` 1 to 4: a composed npc record takes the row of mission `(level-1)*40`; a mod character
  whose row is a family takes `<family>_<level>`; without it the band of the chain's town chapter.
  A record or row with one row takes no level (refused).
- `weapon` is cell 0, `shield` cell 1; `armour` replaces cells 2 to 9 and empties those it does not
  name. `mapload.CheckCharacterCells` refuses a name no item of the cell's kind has.
- A member a link names again comes as she ended the previous link (`guestParty`, keyed by
  `npc:<n>` or `char:<key>`).
- Records without DataBinID use the explicit default of DIV-4003. Female and Mage select the
  PC_Naira, PC_Fergard or PC_Reniesta family; neither selects PC_Danath. The declared level, else
  the town's band, selects the tier. Its full row supplies statistics, skills and equipment. A
  human-figure record supplies its face; a flat portrait keeps the row's human face and body.
  Name uses the record's Name, else its localized NPC name, else the catalog key. Only the data
  party supplies this row to CharacterMember. The old campaign constructor receives no explicit
  row and retains its results. This default claims no original playable creature behaviour.

### Persistence

`currentGuest` gains `Party` (the link's members after the first) and `Hero` (the waiting party's
starting hero, for a chain of more than one link), both `omitempty`. A chain of one link writes the
record an earlier revision wrote. The chain itself is read back from the active mods' catalog, which
the mod mark's digest pins; a LOAD without the mod is refused by that mark. Every old save has the
default of no `Party` and no `Hero`: a chain of one link.

### Sites before and after

| Site at `d290a371` | After |
|---|---|
| `pkg/game/table.go:235,263` `LoadNPCDefs`, `LoadNPCFaces`: two parses | `game.LoadNPCCatalog`, one parse (`pkg/data/npccatalog.go:39`) |
| `pkg/game/mapmessagepages.go:135` speaker list parses `npc.reg` itself | reads `Table.NPCCatalog.Records()` |
| `pkg/mod/kinds.go:109,182` characters and missions: own merge in the game | `LayerMerged` via `mod.MergeCharacters` (`pkg/mod/characters.go:311`), `mod.MergeMissions` (`pkg/mod/missions.go:191`) |
| `pkg/mod/missions.go:88` `AfterWin`: one guest after a main win | `AfterWin` (`:124`), `Chain` (`:146`), `checkChains` (`:218`) |
| `pkg/game/modmission.go:158` `checkGuest(c, src, m)` | `checkGuest(c, src, d, m)` (`:235`) checks the chain's first link |
| `pkg/game/modmission.go:247` `startGuestMission`: one companion | `:527` builds the link's party (`guestParty`, `:430`) |
| `pkg/game/modmission.go:293` `finishGuestMission`: back to town | `:589` next link or back to town |
| `pkg/game/modmission.go:320,345` `currentGuest`, `restoreGuest` | `Party`, `Hero`; `restoreGuest` (`:674`) rebuilds the chain |
| `pkg/mapload/joined.go:18` `CampaignNPCMember` | wraps `CharacterMember` (`:45`); `CheckCharacterCells` (`:111`) |
| `pkg/game/campaignreturn.go:137` `townCompanion` | wraps `townInstall.character` (`:144`) |

## Proof

- `pkg/game/modchain_release_test.go` `TestReleaseModMissionChainPlaysAndReturns` (EN, RU): the
  fixture mod `guest-chain` plays mission 25 after 20, then 26 after 25 with a party of companion 22
  and the mod's sellsword at level 2 with a set weapon and armour; the second link opens with purse
  100 and both members; the last win brings the waiting party to the chapter-30 town, the guest
  joined as she ended the chain, the sellsword not carried, and the town's digest equal to the town
  without the mod. A SAVE inside mission 26 and a cold LOAD continue the chain to the same town and
  the same joined guest; a LOAD without the mod is refused naming it.
- `TestReleaseGuestPartyBuildsCharactersAtTheirLevels` (EN, RU): companion 22 at level 3 equals the
  town's companion of mission 80; the sellsword at level 1 has row `NPC06_1`; `npc999` is refused.
- `pkg/game/modchain_test.go`: `TestSetModMissionsRefusesABrokenChain` (missing link, cycle, unknown
  character, two mods on one link, two who join, each with file and line);
  `TestGuestChainEntersItsPartyAndJoinsItsGuest`; `TestGuestChainRecordCrossesTheSessionSupplement`
  (the record round-trips; a one-link record has no `Party` or `Hero` key).
- A save of an earlier revision, out of tree because it needs an input no release run supplies: a
  writer test run on `d290a371` wrote SAVEs inside the reniesta guest mission (EN sha256
  `d7d0b25a…`, 235089 bytes; RU `6eab2421…`, 235720 bytes); a loader test on this revision loaded
  each under the mod, won the mission and reached the chapter-30 town with the guest joined and the
  town's digest equal to the town without the mod (EN and RU pass). The same writer run on this
  revision writes byte-equal files, so the one-link record is unchanged and the in-tree
  `TestReleaseModMissionGuestPlaysAndJoins` covers that shape.
- Loss controls on `709e9ada`, EN, restored with `git checkout 709e9ada -- pkg/game/modmission.go`:
  `finishGuestMission` ignoring `AfterWin` fails `TestReleaseModMissionChainPlaysAndReturns` ("the
  first link's win led to 4: mission 25 won - Reniesta returns"); `guestSpec` ignoring level and
  cells fails `TestReleaseModMissionChainPlaysAndReturns` (the sellsword has row `NPC06_1`'s skills)
  and `TestReleaseGuestPartyBuildsCharactersAtTheirLevels` (companion 22 has the town band's row).
  `TestReleaseModMissionGuestPlaysAndJoins`, the legacy form, passes under both breaks.
- Changed existing tests, no asserted value lost: `pkg/modrt/missions_test.go` (the two-mod refusals
  now come from `mod.MergeMissions`, whose message names the merge), `pkg/modrt/mergeddata_test.go`
  and `testdata/merged-data.txt` (the new fields and the fixture), `pkg/game/modmission_release_test.go`
  (`guestMods` extracted into `assembledMods`), `pkg/game/speakersex_release_test.go` (portraits from
  the catalog).

## Correction proof

The correction changes the three returned findings against candidate `cb9705dc`.
The base character-edit outcomes are the outcomes of `f16bf050` reproduced by the review probes.

- `TestCharacterEditsKeepBaseOutcomes` compares complete parameter and equipment arrays for all
  definition rows: unordered disjoint edits; a later kind resetting a face; and interleaved family
  and individual-row edits. All three cases pass. `TestCharacterAdditionsKeepTheirLayerRule`
  keeps the unordered refusal and ordered merge for a new catalog key.
- `TestReleaseCatalogCharactersCanBeGuests` passes on EN and RU for npc1, npc31, npc100 and npc22.
  Each declaration loads, enters the mission through the campaign win, saves, loads cold and wins
  to the chapter-30 town. Identity, name, statistics and figure survive the cold load.
- `TestReleaseCatalogGuestMemberCoverage` retains the review's bounded census input: chapter 30
  and a male fighter reference party. EN and RU each build 105 of 105 catalog records, with zero
  refused; the returned candidate built 23 and refused 82. Existing chain, guest-level, legacy
  guest and held-grant release witnesses pass on both roots.
- Temporary Go overlays restore the returned candidate's characters.go and joined.go. The former
  fails all three edit cases: unordered refusal, face 5 instead of 12, and face 6 instead of 7.
  The latter fails npc1, npc31 and npc100 while npc22 passes. With overlays withdrawn the focused
  EN and RU selections pass. The controls never rewrite production files.
- The exact-base guest SAVE probe passes again on both roots. Candidate bytes equal the review's
  f16bf050-written bytes: EN 235089 bytes, SHA-256
  `d7d0b25a6a87c4304372d3f388dc030990f81ce322e1bbea19a381f7ec8a223f`; RU 235720 bytes,
  `6eab2421f94d79f0ddca7d0c64a181b4c3414bcfdb96c4cf65ed4454b891c3df`.
  Both old saves load cold and continue to the same town and guest. The correction changes no
  SAV reader, writer or supplement shape; milestone-2 is not required by the correction brief.
- CommentBytes falls from 9586915 to 9586637 as the obsolete target-merge comments shrink.
  Other storyguard counters are unchanged. Final gate receipts are retained in the lane notes'
  `correction/` directory. The prior candidate's gate receipts below remain historical.

No new ROM1 fact is inferred for the human template default. DIV-4003 states the default and its
owner authority. A missing default row or invalid definition/equipment is refused with a reason.
The first tier uses the bare PC name; later tiers use suffixes _2 to _4.

After fetching and merging origin/main at f16bf050 (already included), the correction's asset-free
gates pass: ordinary tests of touched packages, vet, build-all, gofmt, diff-check, no-game-assets,
divergence-claim check, and one full `go test -trimpath -count=1 ./...` (83 packages). The whole EN
and RU release sets run on the correction commit; their immutable-SHA receipts remain in the lane
notes. No window or audio witness is authorized.

## Gates

On `9f458e1a`, the merge of `origin/main` at `f16bf050`:

- `gofmt -l`, `git diff --check`, `go vet ./...`, `scripts/check-no-game-assets.sh`: clean.
- `go test -trimpath -count=1 ./...`: 83 packages ok.
- `pipeline/check-div-claims.sh`: DIV-4001 and DIV-4002 cite `REG-SCN-063` and `REG-NPC-058`, which
  carry amendments; the amended clauses (the `LastMission` readers and the `!= -1` arm's scope;
  the `Portrait*` presence counts and the price readers) are not the facts the rows use.
- `pipeline/check-release-tests.sh`, 1421 gated tests: EN and RU, no failure on the EN set with fresh
  output directories; RU one `.map-pages-*` "Access is denied" rename in
  `TestReleaseInstalledMapsSaveUnchangedWithoutAStroke`, which passes alone. Skipped:
  `TestReleaseActivePauseRenderedWorld` (expected) and `TestReleaseAPlayersCastsSeeItsOwnInvisibleUnit`,
  merged from `origin/main`, whose `AGAINROM_INVIZ_SAV` the lane environment does not set.
- `scripts/check-milestone2-acceptance.sh`: EN and RU ok, 51 instruments each.

## Open debt

- Whether a chain may pass through a town, end in another town, or take a second guest who joins is
  an owner decision (DIV-4001).
- A level that sets skills, or a member other than the guest who joins coming home, is an owner
  decision (DIV-4002).
