# Story `1032` closure — saves written by older builds load again

As-built. The pass-by-pass account of the two adversarial reviews is in git and `pipeline/LOG.md`.

## Twelve-aspect matrix

| Aspect | Status | Note |
|---|---|---|
| Data | N/A | No shipped game data or asset format changes. |
| Runtime state | PASS | Ten sections are substituted when the source form predates them: five entity field groups (defence state, corpse-loot suppression, delayed kill-credit, carry capacity, map placement id), the item-weight table, the spell record's tail, the check record's item reference, the pre-53 area-effect record's shape, and the structure section added at version 58 (story `1033`, repaired by this return). Four are re-declared at the load from the started mission; the other six widen to zero and their sentences stand. The population is the whole of what the seven byte-form bumps changed, measured below. |
| Simulation | PASS | `sim.UpgradeSaveForm` runs on the byte form ahead of the unchanged `UnmarshalBinary`. Round-trip byte-exact against the pin at every readable version (`pkg/sim/upgrade_test.go`, 10 subtests) and against a fixture carrying real spell, check and casting records (`pkg/sim/upgraderestore_test.go`, 5 subtests). Four substituted sections are inside `World.Hash()` (`Hash` is FNV-1a over `MarshalBinary`'s own bytes, so every encoded section is in it by construction): the script section, the item-weight table, `Entity.Load` and, since this return's rung, the structure section. The repairs move the hash. The same mission-20 save resumed to `f657ff0bd11308de` and the mission-151 save to `58aee6b724e42513`, both re-measured after the `1033` merge and identically on both roots. |
| Player input | PASS | `advanceNotice` (`pkg/game/world.go`), the seam all three dismiss inputs reach, gained a paging arm: a load notice with more than one page stays open and advances to the next page on a dismiss press instead of closing after the first. A version-53 save needs three dismiss presses to clear its notice where one closed it before this story. Witnessed by `pkg/game/loadnotice_test.go` (the paging behaviour in isolation) and by `pkg/game/resume_upgrade_test.go` (the same seam reached from a resumed save). This row read N/A until 2026-08-23 (1032 return 2, F-5): `git diff` against the pre-story tip shows `+49/-1` in `world.go` alone, adding `missionNotices.pages`, `openLoadNotice`, the paging arm and its clear in `closeNotice` — an input-handling change the matrix had not counted. |
| AI | N/A | No AI decision code changed. A pre-55 save's lost delayed kill-credit (`DIV-256`) changes which entity receives credit for an in-progress kill, but no AI code path is touched. |
| UI/HUD | PASS | The disclosure is paged through the existing dialogue notice window and every page is drawn whole. Measured through `cmd/savecheck` on the real EN and RU fonts: a version-55 save pages to 2 pages of 5 and 5 drawn lines, a version-53 save to 3 pages of 5, 7 and 5, a version-56 save to 1 page of 7. No page is clipped. The refusal a save this build cannot open puts on the load window's message line names the save and fits the 104-column field for the versions this build refuses outright (too old, the allocated-and-unused 51/52 gap, too new): 101 runes at its longest with a three-digit format number. One arm is over the field: a readable version whose body will not decode — a corrupt file, not a version this build refuses by name — reaches 109 runes at longest (version 50), five over the field. `clipRunes` marks that cut with an ellipsis; the save name still shows. See "The message-line width is not a universal" below. |
| Triggers/scripts | PASS | The check record grew from 73 to 76 bytes at version 57, so an older form's checks widen with `Item` and `HasItem` empty. `RestoreCheckItemRefs` rebinds them from the started mission's own compiled checks and refuses unless the two check arrays agree in every field a pre-57 form carried (`DIV-276`). Measured on mission 151, both roots: 5 of 36 checks bind an item after the repair, 0 before it. Every other script-adjacent section, the script counts, instants, triggers, script state and relation, is unchanged in shape at every version 50 through 57 and is copied whole. |
| Inventory/equipment | PASS | The item-weight table arrived at version 56; an older form widens with a declared count of zero, and `Entity.Load` is written only from that table, so every actor's load is zero until it is re-declared. `DeclareItemWeights` re-declares it from the started mission and recomputes every load (`DIV-257`). Measured on mission 151, both roots: 39 weight entries and 146 actors with a non-zero load after the repair. `Capacity` is an entity field and is not restored (`DIV-275`); its sentence stands. Group, sack, carry, equipment, death-gold and purse are unchanged in shape at every readable version and are copied whole. |
| Persistence/save-load | PASS | The story's own subject: B1 through B5 of `spec.md`. `cmd/savemigrate -w`'s own refusal logic (B5) is narrowed this return: it can no longer rewrite any pre-58 file in place, found while sweeping `RepairableFromMission`'s callers for the new section — see "Remaining open items" below. |
| Campaign/session | PASS | The envelope's mission field and label are unchanged by this story, verified against real saves carrying missions 20 and 151. |
| Shipped content | N/A | No shipped mission or asset content changed. The corpus for B1 is the owner's own 65 save files. |
| Interactions with existing mechanics | PASS | The weight system, corpse-loot system, kill-credit system, script compiler and the notice window are reused, not duplicated. Both install-gated seat gates and the milestone census are unchanged. |

No in-scope `GAP`.

## Integration witness

Re-measured 2026-08-23 (1032 return 2), after the `1033` merge and the 57-to-58 rung, against a
scratch copy of the same two real owner saves this closure has always used, loaded through
`cmd/savecheck load` against both installs. Identical output on `gameversions/en` and
`gameversions/ru` in each case: same hash, same tick, same counts, same disclosure.

The first owner save, byte-form 55, mission 151:

```
savecheck: load notice: 2 page(s)
savecheck:   page 1/2, 5 line(s) produced, 5 drawn: This save was written by an older version of the game. Some things could not be restored. no carry capacity for any character (added at version 56): nobody is slowed by an overloaded pack for the rest of this session
savecheck:   page 2/2, 6 line(s) produced, 6 drawn: no character on this save can be recognised as a specific placed character (added at version 57): a scripted event that triggers on catching, escorting or otherwise acting on one particular character by name will not fire for the rest of this mission
savecheck: resumed at tick 8440, hash 58aee6b724e42513, purse 100, 235 entities, 3 commanded, 185 of 20736 cells explored (0.9%)
savecheck: after upgrade repair: 39 item-weight entries, 235 entities, 146 with a non-zero load, 0 with a non-zero capacity, 0 with a map placement id, 5 of 36 check(s) bind an item, 28 spell rule(s), 91 structure(s)
```

The second owner save, byte-form 53, mission 20:

```
savecheck: load notice: 3 page(s)
savecheck:   page 1/3, 5 line(s) produced, 5 drawn: This save was written by an older version of the game. Some things could not be restored. no corpse-loot suppression for any character (added at version 54): every corpse drops loot
savecheck:   page 2/3, 7 line(s) produced, 7 drawn: no pending delayed kill-credit for any character (added at version 55): a kill in progress when this save was taken will not credit its original attacker no carry capacity for any character (added at version 56): nobody is slowed by an overloaded pack for the rest of this session
savecheck:   page 3/3, 6 line(s) produced, 6 drawn: no character on this save can be recognised as a specific placed character (added at version 57): a scripted event that triggers on catching, escorting or otherwise acting on one particular character by name will not fire for the rest of this mission
savecheck: resumed at tick 111318, hash f657ff0bd11308de, purse 100, 48 entities, 5 commanded, 3452 of 20736 cells explored (16.6%)
savecheck: after upgrade repair: 27 item-weight entries, 48 entities, 32 with a non-zero load, 0 with a non-zero capacity, 0 with a map placement id, 0 of 14 check(s) bind an item, 28 spell rule(s), 30 structure(s)
```

Both hashes moved from the previous landing's own figures (`1c651d33014abee9`, `33ea158b1a50c244`):
the structure section is now part of every byte form, inside `World.Hash()` by construction, and the
57-to-58 rung and its repair run ahead of the unchanged hash function. The disclosure page counts are
unchanged (2 and 3); `lostMapUnitID`'s rewritten sentence (F-7) wraps to one more line than its
predecessor (6 against 5) because it is longer, and no page gains or loses a page as a result.

Before this story `UnmarshalBinary` refused both files outright and neither mission opened.

The `after upgrade repair` line is `cmd/savecheck`'s own, added by story `1032`'s first return and
extended by this return with the structure count. It exists because no committed command printed any
of the numbers `DIV-257`, `DIV-275`, `DIV-276` and the structure repair below rest on: `World.Hash()`
moves, and a hash that moved does not say which section moved it. The fresh construction of the same
mission is printed by the same command with `-rebuild`: mission 151 gives 238 entities, 148 with a
non-zero load, 106 with a non-zero capacity, 237 with a map placement id, 91 structures; mission 20
gives 57 entities, 35 with a non-zero load, 16 with a non-zero capacity, 56 with a map placement id,
30 structures. The structure count agrees between the resumed (widened-then-repaired) world and the
freshly built one for both missions, on both roots: `DeclareStructures` installs the started
mission's own list, and the widened form itself carries none to disagree with (`upgraderestore.go`'s
own (b) argument for this section).

## The substituted-section population

Two sections were zero-filled where the load path already held the data at the previous landing;
this return adds a third (the structure section, repaired from the freshly started mission). The
population is every byte-form section whose shape or presence moved across the seven version bumps
this table reads, found by diffing every length constant under `pkg/sim` between the seven bump
commits: `0fe514c9` (version 50), `feaa1c4b` (53), `1d5e3a18` (54), `769e25ec` (55), `3ea46f2a` (56),
`dcdb3171` (57), `614d3436` (58, story `1033`, merged into this branch by this return).

| Section | What moved | Disclosed | Data at load | Repaired |
|---|---|---|---|---|
| `SaveFormDefenceState` | `entityLen` 230 to 260 at 53 | yes | entity field | no |
| `SaveFormAreaEffectShape` | `effectRecordLen` 6 to 27 at 53 | yes | per-effect state | no |
| `SaveFormSpellRuleTail` | `spellRecordLen` 21 to 35 at 53 | yes | the mission's spell table | yes |
| `SaveFormCorpseLoot` | `entityLen` 260 to 261 at 54 | yes | entity field | no |
| `SaveFormKillCredit` | `entityLen` 261 to 267 at 55 | yes | entity field | no |
| `SaveFormItemWeights` | `itemWeightCountLen` and `itemWeightRecordLen` added at 56 | yes | the mission's weight table | yes |
| `SaveFormCarryCapacity` | `entityLen` 267 to 275 at 56 | yes | entity field | no |
| `SaveFormCheckItemRef` | `scriptCheckLen` 73 to 76 at 57 | yes | the compiled script | yes |
| `SaveFormMapUnitID` | `entityLen` 275 to 277 at 57 | yes | entity field | no |
| `SaveFormStructures` | structure section added at 58 | no (repair cannot refuse; see below) | the mission's structure list | yes |

Ten sections, and the sweep finds no eleventh. `attachedRecordLen` (19) and `bookRecordLen` (20) also
arrived at version 53, but they are new record kinds in the casting section rather than a widening. A
pre-53 form carries no attached cast and no spellbook record at all, so nothing is substituted for
them and no section is owed.

The four repairable sections are the four a started mission holds in full. `RepairableFromMission`
names them and is the one declaration of that set, witnessed by
`TestRepairableFromMissionIsExactlyTheSectionsAMissionHolds` (renamed this return from
`...ExactlyTheThreeSections...`, stale the moment the count moved from three to four, on
`pkg/sim/binary_test.go`'s own precedent against baking a live count into a test's name). The other
six are per-entity or per-effect state. An entity of the save can be matched to an entity of the fresh
construction by id alone, which is not a proof of identity: the counts above, 235 against 238, are the
measured case where it would be wrong. `DIV-275` carries that reasoning.

`SaveFormStructures` is disclosed at no version and its sentence never reaches the player
(`upgraderestore.go`'s own comment on `DeclareStructures`): the repair cannot refuse, because no
version below 58 carries any structure bytes to compare the mission's fresh construction against, so
criterion (a) (identity) is vacuous the same way it is for the item-weight table, and criterion (b)
(immutability) holds for the same reason `upgradeSteps`' version-57 table entry gives in full (see
"The 57-to-58 rung" below). The disclosure sentence exists only for `cmd/savemigrate`'s own report,
which names every repaired section for a file it will not rewrite.

## The upgrade table is witnessed cell by cell

`upgradeSteps` has six rows (versions 50, 53, 54, 55, 56, 57) of seven field values plus a loss list:
the source version's entity record width, its spell record width, whether its casting section needs
the pre-53 translation, whether it carries an item-weight section, its check-record width, its
instant-record width, whether it carries a structure section, and its list of losses. Every one of
the resulting 48 cells was mutated in place and required to turn the suite red.

Twelve were green when this story's first return began, and a further class of four was found by
running the sweep then:

- `spellRecordLen`, `legacyCasting` and `checkRecordLen` at each of versions 53, 54, 55 and 56. The
  only older-version fixture in `pkg/sim` peels a world with no spell, no check and no area effect,
  so those twelve cells multiplied by zero. `upgraderestore_test.go` now peels a fixture carrying two
  spell rules with non-zero version-53 columns, three checks with item references and one area
  effect, to every version the table reads rather than to version 50 alone.
- the loss list of versions 50, 53, 54 and 55. Dropping a section from a row changed nothing any test
  asserted. `TestEveryVersionDisclosesExactlyTheSectionsAddedAfterIt` derives the expected set from
  the `SaveFormSection` enum and from each section's own disclosure sentence, which names the version
  that added it, so the expectation does not come from the table under test.

This return adds the version-57 row's own four new cells — `checkRecordLen: checkRecordLenV57`,
`instantRecordLen: legacyInstantRecordLen`, `hasStructures: false`, and `lost: []SaveFormLoss{lostStructures}`
— for the 57-to-58 rung. Each was mutated in place (`checkRecordLenV57` to `scriptCheckLen`,
`hasStructures` to `true`, the loss list to empty, `instantRecordLen` to `scriptInstantLen`) and each
turned `TestUpgradeSaveFormReproducesThePinAtEveryOlderVersion` or
`TestEveryVersionDisclosesExactlyTheSectionsAddedAfterIt` red; each was reverted with the matching
edit and a clean `git diff` confirmed the revert was byte-identical. `DeclareStructures` and its call
site in `pkg/game/resume.go` are outside this table and are witnessed separately: see
`TestDeclareStructuresPutsBackWhatTheUpgradeZeroed` (`pkg/sim/upgraderestore_test.go`) and
`TestResumeWorldRepairsStructuresFromTheStartedMission` (`pkg/game/resume_upgrade_test.go`), both
added this return after a reordering mutation and a no-op mutation respectively passed the suite with
zero failures.

After all of it, every cell in the table turns the suite red. The older record widths the fixture
peels to are written out as test-local literals for the same reason: a peel built from
`spellRecordLenV50` cancels a mutation of `spellRecordLenV50`, measured.

## B1 census, re-measured

`cmd/savemigrate` with no `-w`, against a scratch copy of the owner's 65 save files
(`implementation/saves/` and `implementation/saves/rescued-20260822/`):

```
version census:
  byte-form 41: 1
  byte-form 45: 2
  byte-form 46: 5
  byte-form 53: 23
  byte-form 55: 28
  byte-form 56: 1
  town, no world half: 5
  total: 65
```

52 of the 60 world-half saves load. The 8 at versions 41, 45 and 46 are below
`oldestReadableVersion` and are refused by name. Before this story only version 57 was accepted and
none of the 65 files is at 57, so 0 of 65 loaded.

No `.bak` file was written by any run this lane made. Ten pre-existing `.ags.bak` files sit under
`implementation/saves/` (5, timestamped 2026-08-22 00:26) and `implementation/saves/rescued-20260822/`
(5, 2026-08-22 01:04), all ahead of this story's work. An earlier count of five in this document read
one of the two directories.

## Gate summary

Measured on the clean tree at the pushed commit. Each line is the gate's own printed number, not its
verdict.

Implementation repository, at `wt-1032`:

- `go build ./...` — exit 0, no output.
- `go vet ./...` — exit 0, no output.
- `gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')` — no output.
- `go test -trimpath -count=1 ./...` — exit 0, 43 packages `ok`, 0 `FAIL`.
- `scripts/check-no-game-assets.sh` — exit 0, `clean (tree scan)`.
- `scripts/check-claim-citations.sh` — exit 0, `ok (1236 distinct citations resolve against 1453
  claims and 216 experiments under 786 prefixes)`.

Research repository, at the pinned submodule `753034d`:

- `go build ./...`, `go vet ./...`, `gofmt -l` — exit 0, no output.
- `go test -count=1 ./...` — exit 0, 7 packages `ok`.
- `scripts/check-claim-ids.sh` — exit 0, `ok (1453 ids, all distinct, 31 ledgers, 29 read back
  through tools/claim)`.
- `scripts/check-regen-out.sh` — exit 0, `ok (7 regen.sh, every one honours OUT and writes nowhere
  else)`.
- `scripts/check-retraction-status.sh` — exit 0, `ok (232 overturned ids, every one marked)`.

Seat gates, run from the project folder with `AGAINROM_IMPL` pointed at this worktree:

- `pipeline/check-pin-forward.sh` — exit 0. Master pins `753034d`; 1 unmerged lane branch
  (`1032-old-saves`, this story) pins `753034d`. Re-measured after the `1033` merge (F-6): stories
  `1031` and `1033` had both landed by the time this gate ran, so the branch count fell from the
  return brief's 3 to 1.
- `pipeline/check-preserved-installs.sh` — exit 0, `ok — 162 file(s), both roots as recorded`.
- `pipeline/check-div-claims.sh` — exit 0, 176 live rows of 176, 249 distinct claim ids, retraction
  table 232, 43 rows cite a retracted claim. The script printed the path, sha and pin it read, and
  they are this worktree's.
- `pipeline/check-milestone.sh`, with `AGAINROM_MILESTONE_DRIVE` set to a `missionrun` built from
  this worktree at the merge commit — exit 0, `ok — the script gap and the drive are where they were
  recorded, both roots`. Re-measured after the `1033` merge (F-6): the return brief's own pre-merge
  reading, taken before this story's rung existed, found 24 added `cannot run` lines over 66
  unrunnable nodes on a version-58 world with no table entry; that reading no longer applies once the
  rung is written, and the gate's `ok` here is against the tree as it now stands, not the number the
  brief carried in.
  The mandatory pre-report drive, run separately against the same binary:
  `AGAINROM_ASSETS=gameversions/en missionrun -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED` and
  the same for mission 20, both 0, matching `pipeline/milestone-baseline.txt`. This story reads saves
  and draws a notice; it adds no script opcode and removes none, so the count is expected to be
  unchanged, and the gate's own diff against the baseline file confirms it did not move.
- `pipeline/check-scenarios.sh` — exit 0, `ok (14 of 14)` on the EN root and `ok (14 of 14)` on the
  RU root.
- `pipeline/check-release-tests.sh` — exit 0 on both roots. It selected 41 install-gated tests (38
  `AGAINROM_ASSETS`, 1 `AGAINROM_ORIGINAL_SAVES`, 2 `AGAINROM_SAVE_666`) and printed `ok (41 of 41
  install-gated tests ran and passed, 0 skipped)` on each root.

Adversarial pass 2 measured 41 of 41 and 14 of 14 on both roots at `fe5d3842`. Neither number moved.
The install-gated population is 41 both before and after this return: the tests this return adds are
synthetic and run without an install, so none of them joins that population.

No commit on this branch carries a `Co-Authored-By` or `Claude-Session` trailer.
`git log --format='%h %(trailers:key=Co-Authored-By)' <master>..HEAD` prints a hash and nothing else
on every line.

## Research reconciliation

No claim is cited. `provenance.md` states why: every fact this story depends on is this build's own
recorded byte-form history, not a decoded fact about the original executable's save format.
`docs/DIVERGENCES.md`'s eight rows for this story (`DIV-254` through `DIV-258`, `DIV-274` through
`DIV-276`) each record `—` for ROM1 behaviour with the same reason.

## Remaining open items

**The 57-to-58 rung's own argument does not generalise to a future 58-to-59 rung.** The rung repairs
the structure section by writing the mission's own fresh construction over the widened form's empty
one, and it is safe only because every version-57 form was written before the concept of a structure
section existed in any build of this tree: there is no session value to lose, so the immutability
argument holds by construction rather than by inspection. A later widening of a save that DID carry a
structure section once the concept existed faces the same identity problem F-1 found for per-entity
fields — an id match is not a proof, and a fresh mission's construction is not the value a resumed
session actually held. `upgrade.go`'s own comment on the version-57 table entry states this and warns
against copying its reasoning forward. No file in the owner's corpus is at version 57; every save this
build writes now is at 58.

**`cmd/savemigrate -w` cannot currently upgrade any file below version 58 in place.** Found while
sweeping `RepairableFromMission`'s callers after adding `SaveFormStructures` to that set, not by
either adversarial review. `holdsSectionData` decides whether `-w` may rewrite a file by asking
whether the widened world holds the data a lost section's repair would write; for the first three
repairable sections that question is answerable from the save's own other decoded fields (does the
compiled script have checks, does it have spells, does any actor carry or wear something). For
structures there is no such signal: a structure is per-map state seeded from the class table at
mission start, a widened pre-58 world's own structure count is always zero regardless of what the
target mission actually has, and this tool opens no map and starts no mission, so it cannot ask the
question that would settle it. It now answers `true` unconditionally for `SaveFormStructures`
(`cmd/savemigrate/main.go`, `holdsSectionData`), on the same "refuse rather than guess" rule the
other three repairs already follow. The alternative was measured before the fix: without it, `-w`
would silently rewrite a real mission's save with zero structures baked in permanently, and every
later load of that rewritten file opens with no buildings at all, because a file already at
`formatVersion` is never widened and never repaired. Since every version this table reads (50 through
57) loses the structure section, this closes `-w`'s upgrade-and-write success path for every one of
them; report mode, `cmd/saverepair`, and `pkg/game`'s own load-time repair are unaffected.
`TestSavemigrateWriteRefusesAnOlderFileForItsMissingStructures` (renamed from
`...WriteUpgradesAndKeepsABak`, `cmd/savemigrate/main_test.go`) now proves the refusal instead of the
withdrawn success case; `TestHoldsSectionDataAsksWhatTheWorldCarries` gained two structures subtests,
one against a world with no structures and one against a world holding one, both `true`, proving the
signal genuinely does not distinguish the two cases. Both were mutated (`true` to `false`) and turned
the suite red, reverted cleanly. No divergence row: this is this build's own dev-tool behaviour, not
a ROM1 fact.

**Six of the ten sections are not repaired and their sentences stand.** They are per-entity or
per-effect state and cannot be matched between the save and the fresh construction by anything
stronger than an entity id. `DIV-275` is the row; the removal condition is an entity identity that
survives a Control Spirit raise, or a save format that carries the field at every version.

**An upgraded save is not bit-identical to a native current-version save of the same session.** Six
sections widen to zero rather than to the value a later save of the same session would have carried,
and nothing recomputes them. This is a property of substituting honest defaults for fields a
pre-upgrade form never wrote, and it is covered by the divergence rows above rather than by a new one.

**Nine of the 65 save labels carry a UTF-8 em dash, and this story left them alone.** The label is
stored in the envelope and reads back correctly: `cmd/savecheck` prints
`"mission 40 — tick 15040 — gold 1847"` for one of them. Every draw path for a save row
is `ebitenutil.DebugPrintAt`, whose debug font covers ASCII, so U+2014 has no glyph there. The nine
files are all from 2026-08-12 to 2026-08-14 and predate this story; the label formatter stopped
producing em dashes on 2026-08-15. This closure did not see the screen and does not claim what it
draws. A fix belongs in the row text the picker composes, not in the owner's files, and is
hotfix-shaped rather than story-shaped.

**Versions 51 and 52 remain refused.** `binary.go`'s own header records both as allocated and returned
unused, and no build of this tree ever wrote either shape. Unchanged from `contract.md`.
