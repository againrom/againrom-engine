# The World's game profile

## Intent

A World asks once which game it runs. One `gameProfile` value, chosen by one constructor,
`newGameProfile` in `pkg/sim/gameprofile.go`, holds every rule the World decides by the game, one
field per difference. Each branch that asked `SecondGame()` (a test of `rom2 != nil`), tested
`rom2` for nil as the game, compared the World's script dialect with `ScriptROM2` as the game, or
called a per-feature chooser now reads its field. No player change: every World hash, SAV byte and
LOAD result stays as it was, on both games.

A spell row's `Second` is not the World's game and not a profile field. It is the spell table's own
row format (the second game's columns), set by `AssignSecondGameArms` where the table is loaded
(`pkg/mapload` `SecondGameSpellArms`) or decoded (the form-106 wrapper). The spell rules that read it
keep reading it: a profile read would change the answer for a World with an empty table and for a
cross-game controlled copy, which keeps its source table.

## Authority

- Owner, «Доводим до 50»: the architecture audit `pipeline/reviews/arch-audit-5809ddc1.md` counts
  77 kinds built more than one way; this story is its R25 (B27 of the audit plan), game dialect
  presence. Owner ruling: architecture work counts 0 toward the release count.
- No claim and no DIV row: the change moves where the game is asked, not what either game does.

## As built

| kind | builder |
|---|---|
| game dialect presence | `gameProfile` and its constructor `newGameProfile(ScriptDialect)`; the zero value is the first game's profile |
| per-feature profiles | the fields `spellGame`, `groupRoute` (`groupRouteProfile`) and `creatureAim` (`creatureAimProfile`) of `gameProfile`, chosen by the same constructor; each per-feature profile's zero value is the first game's |
| spell table row format (the second game's columns) | `AssignSecondGameArms`, where the table is loaded or decoded; its reads are named entries of the guard |
| second-game script state | `rom2ScriptState`, unchanged; its reads stay state reads |

The profile is chosen where the second game's script state is set, on every path that builds a
World:

- mission start and the controlled script copy: `initializeROM2Script` sets `rom2` and `game`
  together from the script's dialect;
- decode, base form: the candidate World in `unmarshalBinary` takes `newGameProfile` of its script's
  dialect, always the first game's below the second game's wrapper;
- decode, the second game's wrapper (form 106): `unmarshalROM2ScriptState` sets the script, `rom2`
  and `game` together.

The profile's fields: `spellGame`, `groupRoute`, `creatureAim`, `nativeTick`, `bankedScript`,
`activityMask`, `cullClearsActionClock`, `nativeGroups`, `nativeEffectDriver`, `nativeMovers`,
`nativeFacing`, `nativeCellPlanes`, `nativeObjects`, `nativePointCasts`, `nativeTargetOrders`,
`nativeShopUse`, `nativeWalkRefusal`, `cellCastReach`, `releaseRevalidates`, `blowOrdersAttack`,
`bookWalkAll`, `postKeepsDying`, `savedProjectileAim`, `withdrawMoverPoint`, `withdrawAnchorCells`,
`keepsSuppressedAcquirer`, `noHealArmReach`, `noMageGroupSight`, `keepsWeaponSpell`,
`autoCastFullDefault`, `nativeRegen`, `nativeDecay`, `keepsCastRetain`, `residueOverBookCasts`,
`savesMissionFog`, `noCompanionObjective`. Each field's note in `pkg/sim/gameprofile.go` says what
the second game does.

`pkg/game` reads the profile through six World accessors: `SpellGame`, `NativeMovers`
(`currentdeadconstruct.go`), `KeepsCastRetain` (`savactionwire.go`), `ResidueOverBookCasts`
(`savactoractions.go`), `SavesMissionFog` (`residuecarrier.go`, the fog row of the SAV writer) and
`RestoresCompanionObjective` (`currentscriptbindings.go`, `restoreMission40CompanionObjective` on a
current-SAV LOAD). The last two replace comparisons of the World's script dialect with `ScriptROM2`;
the profile is second-game exactly when that dialect is `ScriptROM2` on every build path, and a nil
script answers the first game in both. `World.SecondGame` is gone.

The spell row format: `AssignSecondGameArms` sets `Arm` and `Second` on every row of a second-game
table. Eight row reads decide a second-game spell rule by the row: `ordinaryEffectPayload`
(Drain Life, R2-ENGINE-026), `applySpellDamage`, `applySpellHealing`, `poisonPointMagnitude`,
`secondArm`, `spellDamageUnder`, `wholeDamageColumns` and `spellPointDurationUnder`. One read tests
the whole table through its first row: `armSpellID` (`len(w.spells) == 0 || !w.spells[0].Second`);
an empty or first-game table carries arm ids.

State, not profile: `groupDialectNative` answers whether the World holds no Group registry. A
second-game World holds one after an original SAV LOAD imports its Groups, so the answer can change
after the World is built. Its 68 calls stay; `WorldGameStateCalls` ratchets them per file. The
`rom2` nil tests that read or write the second game's script state (`ROM2ScenarioValue`,
`ROM2ScenarioState`, `SetROM2ScenarioState`, `setROM2GroupForced`, `currentROM2Policy`,
`rom2ScriptStateFault`, `appendROM2ScriptState`) stay state reads.

The guard is `internal/archtest/worldgame.go`. The brief names `internal/archtest/gameprofile.go`;
that file already holds the edition-profile guard, so the World game guard takes a new name. In
the production files of `pkg/sim` and `pkg/game` it refuses a `SecondGame` call, a `rom2` nil test,
a `savedGroups` nil test, a comparison or case with `ScriptROM1`/`ScriptROM2`, and a game profile
chosen or written (`newGameProfile`, a write of a `game` field in `pkg/sim`, a use of
`secondGameGroupRoute` or `secondGameCreatureAim`), each outside its named `file#function` entries
of `WorldGameAllowed`. In `pkg/sim` it also refuses a spell row's `Second` (a selector or a key) and
a read of `Second` through a constant index (the whole table's format); in every production file of
`pkg` and `cmd` it refuses a use of `AssignSecondGameArms`, called or taken as a value. The allowed
sites are the eight row reads, the writer, `armSpellID` as the one whole-table entry, and the
callers `pkg/mapload/spell.go#SecondGameSpellArms` and
`pkg/sim/rom2statebinary.go#unmarshalROM2ScriptState`, each with its reason. The edition guard's `ProfileDebt` loses its three `pkg/game` entries; the
fog walk guard refuses a `game` read in `fogStamp`.

## Proof

- `TestEveryBuildPathAnswersTheRecordedGameProfile` (`pkg/sim/gameprofile_test.go`), recorded at the
  base before the change: 14 fixture Worlds (start, decoded, loaded from imported original Groups,
  loaded and decoded, the controlled script copy into each game, the loaded script copy) answer the
  game, the Group registry test, the group route, the creature aim and the spell game, with the
  World hash when built and after 300 steps. The rows and hashes are unchanged. A planted
  `spellGame: rules.SpellGameFirst` in the second game's profile fails 8 rows.
- `TestOriginalSAVWorldsAnswerTheRecordedGameProfile` (`pkg/game/gameprofile_test.go`): an
  original-SAV LOAD of each game, recorded at the base, keeps its answers and hashes.
- `go test ./pkg/sim`, `go test -skip '^TestRelease' ./pkg/game`, `./pkg/formats/sav`,
  `./internal/archtest`, `./internal/storyguard`, `./internal/gatedtests` and `./internal/divledger`
  pass. Test fixtures that set `rom2` by hand now set the profile with it.
- The guard's self-tests plant each shape and the state ratchet;
  `TestTheWorldGameScanRefusesEachSpellMarkerShape` plants the spell row read, the `Second` key, the
  whole-table read, and `AssignSecondGameArms` called and taken as a value in `pkg/sim`, `pkg/game`,
  `pkg/mapload` and `cmd`.
- The sim profile test compares the whole profile, so the two new fields are covered without a row
  change; the game profile test's answer reads the two new accessors too, its rows unchanged.

## Open debt

- The spell row format stays a second marker beside the profile, by design: the row's `Second` is
  the table's data, chosen where the table is loaded or decoded. Its nine reads and three uses of
  `AssignSecondGameArms` are named guard entries.
- `groupDialectNative` is state, as above; its calls are ratcheted, not removed.
- The guard is lexical: a dialect held in a variable and compared with something other than the two
  constants, a typed-nil `rom2` test, a new comparison inside `campaignfirst.go` or
  `campaignsecond.go` (admitted as whole files), or a profile accessor read for another purpose
  passes it.
