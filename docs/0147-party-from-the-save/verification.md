# 0147 — verification

Branch `0147-party-from-the-save`, base master `542f27d`. Research pin frozen at `d1e38ad` for the
whole story.

## Gate

| Gate | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l $(git ls-files '*.go')` | prints nothing |
| `go test -trimpath -count=1 ./...` | all packages ok |
| `scripts/check-no-game-assets.sh` | clean (tree scan) |
| `scripts/check-doc-budget.sh` | ok; `0147` plan 7303 <= 1.2 x spec |
| `scripts/check-sdd-audit.sh` | one note, `0147: no tasks.md`; no FAIL for `0147` |
| `scripts/check-hotfix-ledger.sh` | row length, `Owes` column and file size ok. **One FAIL, and it pre-exists on master**: commit `3d60a04` "test(load): witness the caveat…" touches `pkg/cmd/internal`, carries no trailer and has no ledger row. Re-run in `implementation/` at master `542f27d`: the same single FAIL, same commit. Not this story's, and not repaired here. |

`go test -trimpath -count=1 ./...` was green on the untouched worktree before any edit, so the suite
was a baseline and not an assumption.

No `tasks.md`: one lane implemented its own slice, so every `FR` and `DD` is accounted for below.

## The instruments

Two, and they read the same bytes by different routes so a value that agrees across them could have
disagreed (DD-11).

- **`savtool party <file>`** walks the save's object graph and prints what the FILE says. It opens no
  map and builds no world.
- **`savecheck load -orig <install> -name <file>`** goes through the same seam the LOAD GAME window
  calls, restores the save and prints what the WORLD holds, entity by entity.

Corpus: the fourteen `game####.sav` files of the lawful GOG install at
`C:/Program Files (x86)/GOG Galaxy/Games/Rage of Mages`, read-only. Nothing in this story writes to
an install.

## FR-1 — the party comes from the save

Measured, all fourteen files: 13 restore, 1 refused. `game0010.sav` is refused with "this save was
taken BETWEEN missions", which is L-4's unchanged rule.

| File | Characters restored | Placements claimed / withdrawn | World |
|---|---|---|---|
| game0000 | 5 | 4 / 4 | tick 0, hash `273d5c11bfbd57e8`, 57 entities |
| game0001 | 5 | 4 / 4 | hash `1753af56c12bb079`, 57 |
| game0002 | 2 | 1 / 1 | hash `447993bdca714173`, 36 |
| game0003 | 2 | 1 / 1 | hash `27784e9420f8d4d2`, 36 |
| game0004 | 2 | 1 / 1 | hash `fb1b49735986c638`, 36 |
| game0005 | 5 | 4 / 4 | hash `407447b43e2d655e`, 57 |
| game0006 | 1 | 0 / 0 | hash `54ee6094ab6b15a0`, 36 |
| game0007 | 5 | 4 / 4 | hash `83dbf0407c651231`, 57 |
| game0008 | 5 | 4 / 4 | hash `2fa93a3fc90cd836`, 57 |
| game0009 | 5 | 4 / 4 | hash `ed81bf5bc7be0bda`, 57 |
| game0010 | — | — | refused: between missions |
| game0011 | 1 | 0 / 0 | hash `753643f0bfb9cc69`, 36 |
| game0012 | 1 | 0 / 0 | hash `b5181f1cbac23a96`, 36 |
| game9999 | 1 | 0 / 0 | hash `de823c2ffa5e45eb`, 36 |

40 characters restored over 13 files; `game0010`'s own 2 bring the walked total to 42. Every one of
the 27 claimed placements was a record the map held, and all 27 were withdrawn.

The fallback arm is witnessed synthetically:
`pkg/game/originalsave_test.go:TestAFallbackPartyIsSaidOutLoud` requires the report to say
"NO CHARACTER RESTORED" and "fresh party", so a save whose walk reads nothing cannot substitute
silently.

**The walk itself, and the test the programme could have failed.** `SAV-TOPLVL-052` states that the
stream's first record is followed by a back-reference tag and then the `0000` null-objref word. On
**14 of 14** files `savtool party` reports the record ending exactly there:

```
game0011.sav   record 1: Player @0x000057..0x0007f4 (1949 bytes), then 77 00 00 00
```

and the same `77 00 00 00` on all fourteen. Two published numbers are reproduced exactly on that
file: the decoded stream is **57 972** bytes and the walk's extent measured from body offset 0
through the back-reference tag is `0x7f4 + 2 = ` **2 038** bytes, which are the two figures
`SAV-TOPLVL-052` publishes for `game0011.sav`.

`SAV-UNITLEN-045` states a `Unit` record's fixed part term by term and sums it to 603, floor
`609 + L`. The programme in `pkg/formats/sav/program.go` reproduces that sum term for term in the
claim's own order — 37 + 4 + 2 + 2 + 462 + 2 + 19 + 1 + 55 + 1 + 1 + 17 — and the synthetic test
`TestTheUnitProgrammeReproducesThePublishedFixedPart` measures it off the bytes the walk stepped
rather than off a table.

**The programme is witnessed by mutation.** Changing one raw block's width from 22 to 21
(`Unit+0xbe`) makes all eight walk tests fail, the first at
`Human at 248: HeldWeapon: back-reference to object 19016 at 951, which the walk has not read`.
Reverted; green again.

## FR-2 — the saved cell, and the withdrawal

`savtool party` reads `game0000.sav` character 0 at cell (35, 29); `savecheck load` reports the
entity at `(35,29)`. Character 1 at (33, 30) and entity at `(33,30)`. Two tools, two routes, same
cell.

**A save's own hero carries no map unit id**, which is the defect's cause and is visible in the
instrument: `savtool party` prints `unit 0` for the hero of every file. 15 of the 42 walked
characters carry map unit id 0 and 27 carry a real one — the latter are placements the player took
into his own group, which is what makes the withdrawal necessary rather than theoretical.

**The withdrawal is measurable in the world.** On `game0000.sav` before the withdrawal existed the
resume built **61** entities and reported `joined 54 to map units, MOVED 4`; with it, **57**
entities and `joined 50, MOVED 0`. The four units the previous build reported as MOVED were the
player's own mercenaries, so the only measurable effect that build had on the world was moving the
four characters that should have been the party.

Synthetic pins: `pkg/mapload/saved_test.go:TestASavedMemberStandsWhereTheFilePutHim` places a member
at (44,51) on a map whose authorised drop is (17,20), so the drop walk cannot reach it by accident;
`TestASavedCellIsTakenSoTheRestOfThePartyWalksAwayFromIt` requires no two members to share a cell;
`pkg/game/originalparty_test.go:TestWithdrawRestoredTakesTheDoubleAway` claims two ids the map holds
and one it does not, requires the count to be 2, and requires every surviving unit to appear exactly
once — a filter that wrote past its read cursor would leave a duplicate rather than a gap.

## FR-3 — the decoded state

**The evidence that could have disagreed is a wounded character.** Every party member this tree has
ever minted arrived at full health, because the number came off a fold that writes one value into
both fields. `game0000.sav` character 0 reads `health 107/131` in the file and the restored entity
reads `hp 107/131`. `game0012.sav` reads `138/145`, `game0002.sav`'s second character `119/130`.

**And a mage with mana.** `game9999.sav`'s character arrives `hp 35/35 mana 91/91 class 24` — the
first party member in this build's history to carry a mana pool from a file.

**Item identity, 298 records over the fourteen files.** The file's `item+0x40` is
`(material << 12) | (slot << 8) | (shape << 5) | row` (`ITEM-APPEAR-023`), which is field for field
`data.ItemCode`. Four independent agreements, measured:

- the code's low five bits equal the head's own `+0x0c` definition row index on **298 of 298**
  records — two fields written by different instructions;
- every one of the 71 `Weapon` records carries slot field 1, which is `pkg/data`'s own
  `weaponItemClass`;
- every one of the 18 `Shield` records carries slot field 2, which is `shieldItemClass`;
- every one of the 25 `Item` records carries slot field 14, which is `data.ItemClassCarried`, and
  all 25 are in a container rather than a worn reference;
- the 184 `Armor` records carry slots in {5, 6, 7, 8, 9, 10, 12}, all inside 1..12;
- material spans 0..15 (four bits), shape 0..2 (three bits), row 2..28 (five bits, so no row
  corrupts the shape field).

Worn set carried through to the world, `game0000.sav` character 0: the file says
`Weapon 0x1106 slot 1`, `Armor 0xb70f slot 7`, `Armor 0xac3c slot 12`, and the entity's equipment
reads `[4358 0 0 0 0 0 46863 0 0 0 0 44092]` — `0x1106`, `0xb70f`, `0xac3c` at indices 0, 6 and 11.
His pack reads `[33076 63013 259 3590 44060 33031 3612]`, which is the file's seven codes in the
file's own order.

`game0012.sav` is the case that shows the pack is not equipment: `armed 0 of 0`, worn only armour,
and the weapon code 259 in the pack. The file records a character who took his sword off, and he
arrives bare-handed with the sword in his pack.

**The two regeneration periods.** Every one of the 42 characters reads `regen 100` and `regen 50`,
which are the two constructor immediates `SAV-UNITFLD-049` anchors its alignment on and exactly what
`data.UnitDefaults()` answers — so carrying them changes no world today. `pkg/mapload/saved_test.go`
gives a fixture 77 and 33 instead and requires the entity to carry those, and separately requires
them NOT to equal the constructor's, so the field is read rather than defaulted.

**The alignment anchor, reproduced.** `SAV-UNITFLD-049`'s "finding only a file could produce" is that
a `Human` record's capacity word is `10 * body + 1`. Measured here: **42 of 42**. Also 42 of 42 for
`health <= healthMax`, `mana <= manaMax`, `health > 0` and death stage 0, which is `SAV-DEATH-051`'s
whole-population claim met on a second corpus.

**The definition row.** All 42 characters resolve their own Humans row (`5 off their own Humans row`
on `game0000.sav`), with row values 26, 28, 29, 58, 200 and 201 — all inside the collection's 215
rows.

## FR-4 — what is not carried, in the reader's units

`OriginalSaveNote` is now:

```
ORIGINAL SAVE: your characters, positions, stats, health and items load -- mission progress does NOT
```

100 columns of the 104 `pkg/ui` can draw, checked by the landed
`TestTheOriginalSaveCaveatIsSayableOnOneLine`, which also requires the words "NOT" and "positions".

The counted form, off `game0000.sav`:

```
        party: 5 characters restored, 5 with statistics and pools, 5 off their own Humans row
        4 claimed a map placement and 4 of those records were withdrawn
        wearing 29 pieces (0 to the pack for want of a slot), carrying 7, armed 4 of 4
        READ AND NOT APPLIED: 0 spellbooks holding 0 spells, 1 journals holding 238 entries,
        5 carry capacities, 10 Unknown statistic words, 5 untrained skill sets
```

`TestARestoredPartyReportsEveryAxisItDidNotApply` pins every one of those phrases.
`TestTheReportNamesWhatIsNotCarried` moved with the contract: it required the report to name
"health" and "inventories" as NOT carried and now requires "skill levels", "spells", "the journal"
and "script state". Both words it dropped are axes this story carries, so the old assertion had
become a false claim about the build.

## FR-5 — the four hotfix rows

`docs/hotfix/LEDGER.md` rows `7a993e9`, `2aef7c3`/`e106956`, `d7aedc6` and `7a69592` now read
`FOLDED into 0147 FR-5` (two with a clause naming which requirement supersedes what).
`scripts/check-hotfix-ledger.sh` reports the `Owes` column ok. The `d7aedc6` rule — the LOAD list
reading the RESOLVED asset root — is witnessed by the build README's commands, every one of which
puts the root in `AGAINROM_ASSETS` and none in `-assets`.

## The decisions

| Decision | Where it is witnessed |
|---|---|
| DD-1 second programme table | `pkg/formats/sav/program.go` reuses `tokenMembers`, `playerMembers`, `buildingMembers`, `effectMembers` from `class.go`; `Classes` is untouched, so `Chain("Building")`'s own landed test still passes |
| DD-2 the walk returns what it read | `TestTheWalkRefusesWhatItCannotStep` requires the refusal to name the class; the mutation run above shows the failing offset |
| DD-3 groups read into the `Player` | `TestTheWalkReadsEveryGroupsActorList` splits three characters over two group records and requires all three |
| DD-4 `Saved` is a pointer field | `TestANilSavedChangesNothing` requires the drop cell, `SpawnHP` on both fields and the constructor's two periods for a member with no `Saved` |
| DD-5 all six pool values or none | `TestASavedMembersPoolsAreTheFilesAndNotTheFolds` gives the member a real profile and hero so the fold has its own answer, then requires all six to be the file's, and separately requires `HP != MaxHP` |
| DD-6 routing by the code's own slot | the 298-record census above; `TestASavedMemberStillWearsAndCarriesWhatHeWasHanded` requires the equipment to reach the world through the same `Stock` entry |
| DD-7 withdraw the claimed placement | the 61 → 57 entity measurement above; `TestWithdrawRestoredTakesTheDoubleAway` |
| DD-8 the party resolved at opener-build time | `RestoreParty` is called before `missionOpener` in `RestoreOriginal`; the closure reads the report it was handed |
| DD-9 `ResumeOriginalSave` takes a `BodyList` | `cmd/missionrun` passes `defs.Bodies`; the resumed drive below |
| DD-10 profile from the file's Humans row | `5 off their own Humans row`, rows 26/28/29/58/200/201 |
| DD-11 two instruments | every FR-2 and FR-3 measurement above is a cross-read of `savtool party` against `savecheck load` |

## Acceptance and prohibitions

| Id | Witness |
|---|---|
| AC-1 | the FR-1 table: 13 files, 40 members, each at the file's cell with the file's pools |
| AC-2 | `game0000.sav` character 0: weapon in slot 1, armour in 7 and 12, seven items in the pack |
| AC-3 | `TestANilSavedChangesNothing`; and the fresh path is unchanged — `againrom -check -mission 10` still prints `party at (17, 66) … health 145/145` |
| AC-4 | the `77 00 00 00` terminator on 14 of 14 files; `TestTheWalkEndsOnTheTerminator` synthetically |
| AC-5 | `TestTheWalkRefusesWhatItCannotStep` |
| AC-6 | 100 of 104 columns; `TestARestoredPartyReportsEveryAxisItDidNotApply` |
| P-1 | the spellbook, the journal, the capacity word and the two Unknown words are counted in the report and reach no field of `sim.Entity`. `sav.Character` carries the spellbook as a BOOL and a COUNT and never a spell id, so there is no value for the game tier to write; the two Unknown words are named `U8E` and `U90` in the programme, which is `Member.Name`'s convention for a field whose offset is the only name anybody has |
| P-2 | the item code is the file's own word; the one axis with no published mapping — a saved `Spell`'s `+0x08` — is reported and asked of research below |
| P-3 | no path in this story opens a save for writing: `sav.File` is read through `Open`, and the only writers in the package (`SetActorPosition`, `setPlayer`, `Marshal`) are reached by no code this story adds. The install's fourteen `.sav` files still carry their original modification times, the newest 2026-08-11 22:53 and twelve of them from 2026-08-01/02, after every run above |
| P-4 | `pkg/sim` is untouched. `git diff --stat master..HEAD -- pkg/sim` is empty, so the byte-form version constant did not move. **Version 42 was allocated to this story and is returned unused.** |
| P-5 | `restoredPools` is the one pool derivation, `restoredLoadout` the one routing loop, `data.HeroAppearance` the one appearance call |

## SC-1 — something the owner can point at

`builds/0147-party-from-the-save/` holds `againrom.exe`, `savtool.exe`, `savecheck.exe` and a README
whose every command was run from that directory with the asset root in `AGAINROM_ASSETS` rather than
in `-assets` — including the windowed launch, which resolved the install and printed
`againrom: saves in …\builds\0147-party-from-the-save\saves` before being stopped after nine seconds.

**The LOAD GAME window's own contents are not witnessed on a screen.** No synthetic keystroke or
click was sent: this is the owner's desktop and the window was not verified to be foreground. What
the window would show is witnessed instead through `savecheck`, which calls the same
`FrontEnd.SaveSeams` seam the window calls and prints the same note text and the same restored
entities. That is a weaker claim than having seen it and it is stated as one.

## SC-2 — the script-gap census and the drive

Run from the worktree, both preserved roots:

```
go build -o /tmp/mr ./cmd/missionrun
for m in 10 20; do AGAINROM_ASSETS=<againrom>/gameversions/<root> /tmp/mr -mission $m -trace -ticks 1 | grep -c UNSUPPORTED; done
```

| Measurement | Before (`pipeline/milestone-baseline.txt`) | After, EN | After, RU |
|---|---|---|---|
| mission 10 unrunnable nodes | 17 (1 groupcmd 11, 2 groupcmd 15, 13 instant 2, 1 instant 20) | **17** | **17** |
| mission 20 unrunnable nodes | 13 (1 instant 12, 11 instant 2, 1 instant 23) | **13** | **13** |
| the drive's outcome | lost at tick 224 | **lost at tick 224** | **lost at tick 224** |
| the drive's census | 4 of 36 moved, 1 fell, over 224 ticks | **4 of 36 moved, 1 fell, over 224 ticks** | same |

The drive is `-mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3`, which is
`pipeline/check-milestone.sh`'s own argv. Running it WITHOUT the two waypoints gives
`undecided at tick 400, 2 of 36 moved` — a different drive, not a regression, and worth writing down
because the two answers look like one moving.

**Unchanged, and that is the intended outcome**: this story adds no script arm. The result it owes is
in `builds/`, not in the census.

The resumed drive itself runs: `missionrun -sav game0000.sav -ticks 40 -census` opens 57 entities
with five party members and reports `outcome won at tick 16`, because that save was taken near the
end of mission 20.

## What this story did not do

- **No spellbook is applied.** Two of the 42 characters carry one. `SAV-SPELL-044` grades the
  meaning of a `Spell` record's `+0x08` Unknown, so no spell id is derived from it. Research question
  below.
- **No journal is applied.** 15 of the 42 carry one, 238 entries between the two arrays on the
  hero's. What the arrays hold is Unknown and this tree has no journal.
- **No skill level or experience is restored.** No published claim locates either in a `Unit`
  record. Spec L-1.
- **No carry capacity.** Named at High and this tree has no field for it.
- **The file's own speed word is read and not written.** This tree derives a party member's step rate
  from his Reaction, which IS restored, so the two cannot disagree in the world.
- **Nothing past the terminator.** The map's script state, its trigger latches, its ground loot, its
  sacks and its corpses are a fresh mission start. Spec L-3.
- **A script arm naming a withdrawn unit no longer resolves.** Spec L-2, accepted with the count in
  the report.

## Re-measured at the landing

The corpus at this seat holds **18 distinct save files**: the install's 14, plus
`gameversions/en/game0000.sav`, `game0001.sav`, `game9999.sav` and
`gameversions/saves/2026-08-02/game9999.sav`, which differ from the install's files by MD5. Every
figure above reproduces here first-hand, including all 13 world hashes. Over all 18 the universals
hold on larger populations:

| Measurement | Lane, 14 files | Landing, 18 files |
|---|---|---|
| terminator `77 00 00 00` | 14 of 14 | 18 of 18 |
| characters walked | 42 | 46 |
| capacity `== 10 * body + 1` | 42 of 42 | 46 of 46 |
| pools, `health > 0`, stage 0, regen 100 and 50 | 42 of 42 | 46 of 46 |
| item code's low five bits `==` head `+0x0c` | 298 of 298 | 322 of 322 |
| `Weapon` slot 1 / `Item` slot 14 | 71 / 25 | 79 / 30 |

Unchanged: the `Armor` slot set {5, 6, 7, 8, 9, 10, 12}, `Shield` slot 2 on 18 of 18, material
0..15, shape 0..2, row 2..28, the definition rows 26/28/29/58/200/201, the 27 characters carrying a
real map unit id, and the 10 `Spell` records. The four added files carry one character each, all
with map unit id 0.

**The 298-record census is not `savtool party`'s output.** That listing prints the party's own
pieces, and comes to 264 records over the same 14 files. The 298 is every `Weapon`, `Armor`,
`Shield` and `Item` record in the walked graph, reached through `sav.File.Walk()`. Reproducing it
needs the walk, not the command.

`P-3`'s modification-time count was corrected here from eleven to twelve.

## Question for research

**Does `Spell+0x08` name a Spells-collection row?** `SAV-SPELL-044` reads the load arm looking that
byte up in the collection at `L05046`, and `DAT-OBJ-002` places `L02110 + 0xdc` as Spells; the
claim nonetheless grades the byte's meaning Unknown. If the lookup is the row index, a saved
spellbook maps straight onto `sim.Entity.KnownSpells`, whose bit `i` is already the
Spells-collection index `mapload.SpellIDByToken` resolves a spell name to — so the axis would close
with no new mechanism, only a published meaning. Two of the owner's 42 characters carry a spellbook,
so the population is small; the routine to read is the one `SAV-SPELL-044` names.
