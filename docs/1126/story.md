# Story 1126 — a real party's town SAVE emits a native SAV

## Status

Landed after one correction pass (review `pipeline/reviews/story1126-pass1.md`,
RETURN). The review found three code defects and a false Player result: a
chargen'd fighter's own worn slot-12 armour piece silently moved onto the
character's Diary reference (R-2); a chapter companion resolved to the wrong
Humans row and so was dropped entirely on reload, not merely shown with the
wrong portrait (R-3a); a hired mercenary squad vanished from the roster on
reload while `Town.mercHired` stayed true, jamming the tavern for that type
permanently (R-3b); and no reachable state in the shipped campaign actually
wrote a native SAV, though the story's own Player result and Status claimed
the ordinary played case did (R-1).

This pass fixes R-2, R-3a and R-3b in code. R-1 is not a code defect this
story can fix: the writer is correct wherever it fires, but two refusals
inherited unmodified from story 1125's hotfix — the tavern's accumulated
mercenary-unlock history and the world-map selection history — between them
cover the entire shipped campaign on both roots. Measured with the
reviewer's own world-map-first walk (15 main missions plus the four
unlocking side missions 41/71/111/121, each mission preceded by the same
`markWorldSelected` call the world map itself makes before `Won`), 0 of 19
states write a native SAV on EN or RU. The Player result below states this
plainly, and it is this pass's own open debt.

## Player result

In town, with a native (never-imported) campaign, SAVE does not currently
write a native SAV anywhere in the shipped campaign. Measured on both EN and
RU, walking the 15 main missions plus the four unlocking side missions (41,
71, 111, 121) in ascending order — `[10 20 30 40 41 50 60 70 71 80 90 100
110 111 120 121 130 140 150]` — each preceded by `markWorldSelected`,
`ExportNativeCitySave` refuses at every one of the 19 resulting states:
`nativeCityMercenarySetMismatch` (mission 10's own `EnableMercenary` unlocks
nine types at once; no chapter's own declared list matches that
nine-type set again before chapter 130) covers the zero-win state through
`Won(41)`; the `WorldSelectedOnce` session-state check becomes live at
`Won(50)` — the first mission in the sequence whose own map marker is
recorded — and, once armed, never clears, so it covers every remaining
state through `Won(150)`. SAVE in town still writes the same lossless
`.ags` envelope this build wrote before this story, with nothing lost.

Where the writer does fire — reachable only by a test that neutralizes both
session-state refusals by hand, not by ordinary play — it is now correct: a
chargen'd hero's worn/carried items, spellbook, non-default stats and
identity round-trip; a granted chapter companion's identity and `Hero`
(Body/Reaction/Mind/Spirit/Skill, compared whole) round-trip through the
same document, which was false before this pass (R-3a); and a hired
mercenary now cleanly forces the `.ags` fallback (R-3b) instead of silently
vanishing from the roster while the tavern still believes it hired, which is
what a native SAV/reload cycle did before this pass.

## As-built behaviour

**Items and containers** (`pkg/game/nativecityitems.go`). `nativeCityAttachItems`
walks `mapload.MemberItemEquipment`/`MemberCarriedItems` for the live
member, enriches each through `mapload.SourceConstructedItem` (story 1109's
own weight/combat-operand constructor — never bytes read from an import),
and mints one `CityObjectData` per item: `Weapon`/`Armor`/`Shield` when the
item's `SourceEquipment.Class` resolves a row in the installed table, else a
plain `Item`. The 37-byte Token head (SAV-TOKENPOS-074, SAV-OBJ-014) carries
price at `+0x1c`; the 12-byte Item fields block carries Code/Stack/Kind/
Weight, the four columns `city_inventory.go`'s own reader consumes
(DIV-893/894, new debt). Worn pieces are wired at `Reference74` (weapon),
`Reference78` (shield) and `Equipment[0:12]` (the twelve armour slots,
`data.EquipSlotFor`'s own 1-based numbering carried through Go index `i` =
slot `n-1`); `Equipment[12]` is reserved for the character's own Diary
reference and this loop never writes it (**R-2 fix**: the original landing
wrote armour slot `n` at `Equipment[n]`, so a slot-12 piece hung off the
Diary reference instead of the array's own last armour index —
`program.go`'s Humanoid programme writes twelve Worn references then one
Diary reference, thirteen references end to end, read back generically as
one run by `city_semantic.go` and split only by array position by
`party.go`). Carried pieces append to `Container`, and their summed weight
lands in `CityHuman.ContainerTails[1]` (`ContainerFlag` written 1
unconditionally, DIV-895, new debt).

**Spellbooks**, unchanged by this pass. `nativeCityMemberSpells` resolves a
member's `KnownSpells` bitmask against `mapload.SpellRules`. `city.go`'s own
reader requires spellbook slot N (0-indexed) to hold spell ID N+1 exactly, so
`nativeCityAttachItems` sizes `unit.Spells` to the widest known spell ID and
places each `Spell` object reference at `rule.ID-1`.

**Tracked stats**, unchanged by this pass. Speed/Capacity/HealthRegen/
ManaRegen write from the same `data.Derived` value the other eight
`UnitStatWords` already use (DIV-880, closed by story 1125's own correction
pass). `StatU8E`/`StatU90` still write 0; SAV-UNITFLD-049 grades both fields
Unknown, not an established Weight/Load pair.

**Hero, mercenary and companion identity** (`pkg/game/nativecity.go`,
`nativeCityDefRow`). A chargen'd hero resolves through
`data.ChargenBase(table.Humans, mage, female)`, unchanged by this pass. A
chapter companion no longer shares a hired mercenary's TypeID fallback
(**R-3a fix**): `nativeCityDefRow` now also takes the party's own hero (for
`Mage`/`FigureDir`) and, when `member.CompanionNPC != 0`, calls
`data.NPCDefs.CampaignServerID` — REG-SCN-098's own "26 + selector"
arithmetic, the identical call `mapload.CampaignNPCMember` already makes to
construct the companion's own `PartyMember` in the first place — before
ever falling to `data.FindHumanByType`. This lands an existing female
primary's companion on Humans row 28 `PC_Fergard` and an existing male's on
row 29 `PC_Reniesta`, both `PC_`-prefixed, so `persistentOriginalCharacter`
(`originalparty.go`, unmodified, shared with real ROM1 import) now keeps the
companion on reload instead of dropping it as an unnamed ally (closes
DIV-887's own companion half; narrows DIV-890 to the mercenary case alone).
A hired mercenary still resolves through `data.FindHumanByType` when it is
ever called, but it no longer is (**R-3b fix**): `nativeCityAnyMercenaryHired`
(new) checks every `Town.mercHired` entry and makes `ExportNativeCitySave`
refuse outright whenever any type is hired, before the per-member DefRow
loop ever runs. No claim establishes that ROM1 itself restores a hired
mercenary as an individual roster character rather than reconstructing the
squad from the pool count at town entry, so this keeps story 1125's own
lossless-or-refuse contract instead of writing a document
`persistentOriginalCharacter`'s pre-existing, import-shared filter would
silently empty on reload while `mercHired` stayed true.

**Refusals that remain, unchanged from story 1125's hotfix, now measured
against the whole campaign (R-1, unresolved)**: `nativeCityMercenarySetMismatch`
compares session-accumulated `mercEnabled` against the current chapter's own
declared list; mission 10's own `EnableMercenary` unlocks nine types
(`[14 6 13 4 7 3 2 9 12]`) at once, and only chapters 130/140/150 ever
declare a matching set again, and then only after side missions 41/71/111/121
are also won (measured by the review, pipeline/reviews/story1126-pass1.md).
The `WorldSelectedOnce` check refuses on any non-empty world-map selection
history; `selectWorldMission`'s own `markWorldSelected` call is the only
route into a mission on the world map, and it becomes live at `Won(50)`
(review's own finding: missions 50/60/70/100/110 individually carry recorded
marker art, 10/20/30/40/41 do not) and never clears. Together the two cover
every one of the 19 states this pass measured, on both roots (DIV-884 and
DIV-886, both amended this pass to record the check and the result).

## Divergence rows

Amended this pass (review `pipeline/reviews/story1126-pass1.md`; correction
brief): DIV-884 and DIV-886 (record that this pass checked whether
SAV-CAMPMARK-073, `campaignProgress.selectedMarkers()` or the format's own
mercenary pool fields give either R-1 refusal a document home — neither
does; both refusals stay, and DIV-886 now names the exact chapter span the
walk measured). DIV-887 (removes the "only cosmetics are ever affected"
claim the review's D-5 found wrong for a companion — landing on a non-`PC_`
row meant the character was dropped from the roster outright, not shown
with a wrong portrait; records that the companion case is closed by R-3a and
that the residual TypeID fallback is now reachable only by a hired
mercenary, which no longer reaches it at all). DIV-889 (its own witness
description no longer claims the reverse-conversion block proves a hired
mercenary's equipment round-trips, since a hired mercenary no longer reaches
the writer; it now names the companion round-trip instead). DIV-890
(narrowed from "hired mercenary and companion" to "hired mercenary" alone,
since the companion case is closed; reframed from a silent restore-side loss
to a write-time refusal, `nativeCityAnyMercenaryHired`).

New, within the allocated DIV-890..901 range (3 of the remaining 11 spent):

- **DIV-893** — the Item fields block's own four active byte positions
  (Code/Stack/Kind/Weight) cite no claim; `city_inventory.go`'s own reader is
  this tree's own code, not ROM1 evidence (review D-1).
- **DIV-894** — an item's Stack field is written 1 unconditionally; correct
  for `sim.ItemInstance`'s own representation (no stack-count field), but not
  established against whether ROM1 itself ever writes a shipped item's Stack
  above 1 (review D-2).
- **DIV-895** — `ContainerFlag` is written 1 for every member regardless of
  whether anything is carried, where master before story 1126 wrote 0; not
  established against ROM1's own convention for an empty pack (review D-3).

DIV-896 through DIV-901 (6 numbers) are retired unused.

## Proof

- `gofmt -l .`: clean. `go test -trimpath -count=1 ./...`: all packages
  pass, `internal/gatedtests` included (its checked-in population scan now
  also lists the new mercenary-refusal witness). `scripts/check-no-game-assets.sh`:
  clean.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree) against both lawful roots in one invocation: 8 packages, 173
  gated tests, 2 roots; 173 of 173 ran and 0 lacked a subject on EN, the same
  173 of 173 and 0 lacked a subject on RU.
- `pipeline/check-div-claims.sh` against this worktree: 370 live rows
  scanned, 0 malformed against the 9-cell header, exit 0.
- **Reachability walk (R-1)**, mirroring the review's own third-walk row: a
  chargen'd party arrives in town; for each of the 15 main missions plus the
  four unlocking side missions, in ascending numeric order —
  `[10 20 30 40 41 50 60 70 71 80 90 100 110 111 120 121 130 140 150]` —
  `townScreen.markWorldSelected(m)` runs before `Town.Won(m)` (the same call
  `selectWorldMission` makes before a real player's own pick), then
  `ExportNativeCitySave` is asked. All 19 `Won` calls are accepted. 0 of 19
  write a native SAV, on EN and again on RU: `nativeCityMercenarySetMismatch`
  refuses the zero-win state through `Won(41)`; `WorldSelectedOnce` refuses
  from `Won(50)` through `Won(150)`. Measured with a throwaway test file,
  deleted before this pass's final commit, matching the review's own
  reproduction convention (its "How to reproduce" section).
- `pkg/game/nativecity_release_test.go`, EN and RU, all six tests pass:
  `TestReleaseNativeTownSaveBarePartyRefusesForSessionStateAndFallsBackToAGS`,
  `TestReleaseNativeTownSaveRosterChangeAcrossAGSFallbackSaves`,
  `TestReleaseNativeTownSaveShopBaredSessionRefusesAndAGSFallbackRoundTrips`
  (session-state refusals, unchanged); `TestReleaseNativeTownSaveFighterWornSlotTwelveRoundTrips`
  (**R-2**: a male fighter's own slot-12 armour piece round-trips at the
  same slot after a real native-SAV round trip, and the document's own
  `HasDiary` stays false); `TestReleaseNativeTownSaveRealPartyEmitsNativeSAVAndRoundTrips`
  (rewritten: a chargen'd mage plus a granted chapter companion, no
  mercenary hired; hero identity/worn/carried/spells/stats round-trip, and
  the companion's own identity and `Hero` — Body/Reaction/Mind/Spirit/Skill
  compared whole — round-trip too, proving **R-3a**);
  `TestReleaseNativeTownSaveHiredMercenaryRefusesAndAGSFallbackPreservesRoster`
  (new, **R-3b**: hiring forces `*originalCityUnsupportedError`, and the
  `.ags` fallback preserves roster membership, the type-14 mercenary count
  and `Town.mercHired[14]` exactly).

  These six witnesses reach the writer only because each neutralizes the
  session-state refusals by hand (`f.Town.mercEnabled[14] = true`,
  `f.addChapterCompanions`) exactly as story 1125's own witnesses already
  did — a session shape no ordinary player reaches, which is R-1's own
  finding and not something this pass's fixtures change.
- Milestone census (`cmd/missionrun -trace -ticks 1`, EN root,
  `grep -c UNSUPPORTED`): 0 for mission 10, 0 for mission 20, measured
  directly on the candidate. Unchanged from the original landing's own
  measurement (0/0): this pass's diff touches only `pkg/game/nativecity.go`,
  `pkg/game/nativecityitems.go`, `pkg/game/nativecity_release_test.go`,
  `internal/gatedtests/testdata/population.txt` and `docs/DIVERGENCES.md` —
  no file under `pkg/sim`, `pkg/mapload`'s script/pathing code,
  `cmd/missionrun` or `scripts/`.

## Open debt

**R-1, unresolved.** 0 of 19 reachable town states in the shipped campaign
write a native SAV, on EN and RU. Two refusals, both pre-existing (story
1125's hotfix), between them cover every state: `nativeCityMercenarySetMismatch`
(DIV-886, open — no claim establishes whether ROM1's own PermanentMercenaries
actually diverges from the session's currently offered set) and the
`WorldSelectedOnce` session-state check (DIV-884, open — SAV-CAMPMARK-073
gives the Markers field's wire shape, not its meaning, and
`campaignProgress.selectedMarkers()` caches only the currently selected
mission, not the accumulated history `WorldSelectedOnce` holds). Closing
either needs a claim this correction pass did not find; until one lands,
SAVE in town writes the same `.ags` envelope this build always has, and the
milestone this story was chartered to reach — a real party's town SAVE
emits a native SAV — is not met by any reachable player action.

DIV-887's residual mercenary-only TypeID fallback is dormant in practice,
since a hired mercenary now refuses export before `nativeCityDefRow` is ever
called on one. DIV-890 (mercenary roster persistence) is now a write-time
refusal rather than a silent loss, but stays open pending a claim on whether
ROM1 restores a hired mercenary as an individual roster character at all.
DIV-891/892 (magic weapon bound spell; item Effect references): unchanged,
no fixture in this tree needs either. DIV-893/894/895 (Item fields byte
positions; Stack=1; ContainerFlag=1): new, no claim yet. DIV-889's own four
session-field refusals (pending world-map return, bound quick spell,
outstanding offered mission — folded into the R-1 finding above alongside
world-selection history) remain out of this story's scope. World writer,
mission-side saves and the original-process write boundary remain out of
scope.

## Touched surfaces

`pkg/game/nativecity.go` (R-3a companion DefRow resolution via
`CampaignServerID`; R-3b `nativeCityAnyMercenaryHired` refusal),
`pkg/game/nativecityitems.go` (R-2 armour-slot index fix; DIV-893/894/895
citations), `pkg/game/nativecity_release_test.go` (R-2 witness added; R-3a
witness rewritten; R-3b witness added), `internal/gatedtests/testdata/population.txt`
(one new entry), `docs/DIVERGENCES.md` (DIV-884/886/887/889/890 amended;
DIV-893/894/895 added), `docs/1126/story.md` (this rewrite). No
formatVersion or pinned-digest constant moved.
