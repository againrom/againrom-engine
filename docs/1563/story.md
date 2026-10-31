# Every consumer asks the spell-arm table what an arm does

## Intent

Production code decided what a spell does by comparing its arm or spell id with a literal or a
named constant at each consumer. One table in `pkg/sim/spellarms.go` now holds one row per arm,
one field per question a consumer asks. Every consumer asks a row: most read `spellArmOf` with
the arm the spell runs, and the id-as-arm readers listed under open debt read it with the row id,
as the base did. `pkg/game`, `pkg/data`, `pkg/mapload` and, through a hook `pkg/game` sets,
`pkg/ui` ask through exported `sim` functions. No spell, look, sound, hashed state or SAV changes
on either game.

## Authority

- Owner: «Доводим до 50». This is rank 33 of `arch-audit-5809ddc1.md` (B36 of
  `arch-audit-bdac396d.md`), spell-arm identity. Architecture work counts 0 toward the release
  count. No player change.
- One builder per kind (`AGENTS.md`). A per-game difference is data in the row.
- The answers are the base's own (`7e66b6315`). They are recorded in
  `pkg/sim/spellarmmatrix_test.go`, `pkg/sim/spellarmsiterecord_test.go` and
  `pkg/game/spellcaptionrecord_test.go`, each committed before the consumers it records
  changed. The claims the consumers already cited stay beside the fields: `ANIM-111`,
  `ANIM-112`, `AI-FOLLOWHEAL-118`, `MAGIC-237`, `MAGIC-238`, `MAGIC-DMG-005`,
  `MAGIC-EFFECT-015`, `MAGIC-SING-019`, `MAGIC-UNITLIGHT-057`, `SAV-1155`, `TEXT-081`,
  `TEXT-096`, `TEXT-133`, `R2-ENGINE-019`, `R2-ENGINE-022`, `R2-ENGINE-024`, `R2-ENGINE-025`,
  `R2-ENGINE-346`, `R2-ENGINE-347`, `R2-ENGINE-453`, `R2-ENGINE-454`, `R2-ENGINE-653`.

## As built

| kind | builder |
|---|---|
| spell-arm capability, both games | `spellArms` in `pkg/sim/spellarms.go`, read by `spellArmOf`; second-game differences in the row's `second` field, read by `SpellRule.secondArm` |
| area arm | `areaArms` in `pkg/sim/areaarms.go`, unchanged |
| spell values, delivery, targeting, scroll use, auto-cast setting | their producers (`spellvalues.go`, `spelldelivery.go`, `target.go`, `usescroll.go`, `autocastsetting.go`), each asking a row |
| game-side spell answers | `sim.SpellArm*`, `sim.StoneCurseSpellArm`, `sim.InvisibilitySpellArm` |
| book and item popup caption | `sim.SpellCaptionOf`, read by the caption blocks of `pkg/game/spell.go`, which hold the label, format and kind only |
| data spell classification | `data.ClassifySpell`, asking `sim.SpellArmHeals` and `sim.SpellArmDrains` |
| creature aim | the first game's aim is the row's `creatureAim`, read by `creatureAimProfile.aimOf`; the second game's aims by row id stay in `secondCreatureAims`, admitted by the guard's allow list |

Rows: arms 1 to 28 and the second game's Summon arm (`armSummon`). Fields: `applies`, `atActor`,
`drain`, `heals`, `prismatic`, `teleport`, `pointTarget`, `fastRange`, `manualOnly`,
`controlSpirit`, `summon`, `stoneCurse`, `invisibility`, `attachAward`, `selfAimed`, `own` (kind,
scaled), `opposite`, `magnitude`, `formulaMagnitude`, `lastingDuration`, `durationLaw`, `form`,
`autoCast` (level, order, role), `light`, `creatureAim`, `caption` (kind, negate), `burst`,
`speedFlight`, `second` (drawn duration, blank curse, keeps larger, power duration).

Moved into the table file: `SpellRule.arm`, `spellIDOfArm`, `armSpellID`, `secondSpellArms`,
`SecondGameSpellArm`, `armSummon`, `PrismaticSpellID`, the book walk's last id, the per-arm data
of `pointMagnitudes`, `magnitudeArms`, `durationArms`, `creatureAims` and the arm lists of
`autoCastLevelArms`. That table is now `autoCastTiers`: it holds each level's bits and mana costs
and builds the level's arm list from the rows (`autoCastArmsAt`). The caption numbers come from
the row's own magnitude; the scaled own-effect magnitude is one function, `ownScaledMagnitude`,
at the width each reader computes in.

Removed constants: `controlSpiritSpellID` (sim and ui), `teleportSpellID` (sim and game),
`prismaticSpellID`, `lastBookSpellID` (out of `engage.go`), `autoCastHeal`, `autoCastShield`,
`autoCastInvisibility`, `fireSacrificeSpellID`, `teleportArm`, `lightSpellID`,
`darknessSpellID`, `healSpellID`, `drainLifeSpellID` and `fireBallSpell` (sim and game). The
tests keep their own names for their fixtures. Renamed because the guard refuses any constant
named `...Arm`: `ui.AttackAuthoredArm` to `AttackAuthoredReach`, `pkg/game`'s `anyArm` to
`everyGround`, `sim.ScriptSilenceDeadArm` to `ScriptSilenceDeadCheck`.

`pkg/data` now imports `pkg/sim` (`internal/archtest/dag.go`). `pkg/ui` cannot import `pkg/sim`:
`Viewer.SetSpellTakesBody` takes the question, `pkg/game/mapload.go` sets it, and the gesture
carries the answer into `decide`.

## Proof

- `pkg/sim` `TestSpellArmConsumersKeepTheirRecord` and `TestSpellArmRecordNamesEveryConsumer`
  were committed at `f012403e7` and passed at the base. They pass after the change. A planted
  `drain: true` on arm 12 fails them and `TestSpellArmTable/drain`.
- `pkg/sim` `TestSpellArmSitesKeepTheirRecord` records the first game's creature aim by arm, the
  id an escort's heal and the unarmed auto-heal ask for (6 in both games), Fire Ball's burst and
  its speed-timed flight by row id, and the burst picture 13. `pkg/game`
  `TestSpellCaptionsKeepTheirRecord` records every caption block's answer for every probed row
  id over 26 characteristics in both games. Both ran against an extract of the base before
  they were committed (`365777895`, `b5dd036ea`) and pass unchanged after the change. Planted
  row changes fail each question: arm 18's caption Bless fails the caption record and
  `TestSpellArmTable/caption`; `heals` on arm 5 fails `healID`; arm 9's aim at the victim fails
  `creatureAim`; `burst` and `speedFlight` on arm 3 fail theirs.
- `TestSpellArmTable` has one subtest per capability, 36 in all. `TestSpellArmTableAsksEveryField`
  pins every row field to a recorded question. `TestSpellArmTableRowsAreUnique` pins one row per
  arm.
- The guard `internal/archtest/spellarms.go`, run by `TestLiveSpellArmSitesAreNone`, refuses
  ten shapes: compare, switch, lookup, id-const, const-name, arm-table, field-match, id-match,
  book-id and id-table. Each shape has its own self-test, and the allow list has a self-test of
  its own. Over the base's `pkg/` tree the guard names 190 sites. Over the candidate it names
  none beyond the allow list's one table.
- The census goes from 64 C1 lines and 13 C2 lines to 0, and every lane shape also goes to 0
  (`review/story1563-spell-arms/census-*.txt`).
- `go test -count=1` of `./pkg/sim`, `./pkg/data`, `./pkg/mapload`, `./pkg/ui`,
  `./internal/archtest` and `./internal/storyguard`, and the focused `pkg/game` spell set,
  pass.

Release tests the next release runs on both roots of both games include
`TestReleasePrismaticSpraySecondariesFollowTheGroupsSightAndRank`,
`TestReleaseModRaysHitAndDrawEveryEnemyAndSurviveSaveAndLoad`,
`TestReleaseBookTeleportMovesOnTheCountdownAndDrawsBothEnds`,
`TestReleaseStoneCurseStaysStillThroughSAVColdLoadAndExpiry`,
`TestReleaseSecondCampaignStoneCurseAndSummonSurviveSAVEAndColdLOAD`,
`TestReleaseRaisedGhostSurvivesSAV`, `TestReleaseAutoCastLevelsSurviveSAVEAndColdLOAD`,
`TestReleaseSelfSpellKindsAreShieldAndFireSacrifice`, `TestReleaseCastCursorFollowsTheSpellKind`,
`TestReleaseDrainLifeItemCardUsesLiteralInstalledNamesDamageAndRange`,
`TestReleaseSecondGameSpellsDrawAndSoundTheirOwn`, `TestReleaseCreatureSpellCensus`,
`TestReleaseMilestone2SpellEffects1152`, `TestReleaseMilestone2Projectiles1157`,
`TestReleaseAMissionTenTurnHashMatchesAcrossBothLawfulRoots`,
`TestReleaseSchoolYardDrawsKeepMissionHashes`, the Fire Ball burst and radius release tests
in `pkg/game` (`fireballburst_release_test.go`, `modradius_release_test.go`,
`projectileflight_release_test.go`, `projectilesave_release_test.go`) and `cmd/scenariofixture`
`TestReleaseLegacyMission20EndpointScenarios`. The full list is in the lane's notes.

## Open debt

- Id-as-arm readers keep the base's answers. Each reads a row with the row id where the second
  game's ids differ from its arms:
  - the popup caption, `sim.SpellCaptionOf`: the second game's id 24, which runs the heal arm 6,
    states the Speed caption, and its id 14, which runs arm 17, states the ray count;
  - an escort's heal and the unarmed auto-heal, `healArmAsID`: both games ask for id 6;
  - Fire Ball's burst and speed-timed flight, `rowIDBursts` and `rowIDFliesBySpeed`;
  - the ui body pick, by the selected row id (the second game's Control Spirit is id 22 and its
    Summon id 25);
  - `raysFlag(id)`, and `pkg/mapload/modspell.go` by the first-game row id.
- Unknown: whether a second-game popup reaches the caption blocks in play. If it does, the
  id-as-arm caption is a base defect this story keeps.
- Unknown made visible: second-game escort healing and auto-healing ask for id 6, which is
  Poison Cloud there. This story keeps it; a change needs an owner decision.
- The second game's creature aims stay their own table, keyed by row id: its ids 5 and 16 run
  no arm and still aim, so an arm row cannot hold their aims. It is the guard's one allow-list
  entry.
- The book walk (`bookWalkTakes`, `bookWalkFallback`) is a first-game id range in the table file.
  It is not an arm capability.
- The guard is lexical. It sees an arm in an `arm()`, `ArmID()`, `spellArm`, `SpellArm` or
  `spellArmOf` call, in a variable assigned from one, in a simulation name `arm`, in an `Arm`
  field of a value named for a spell or rule, and a row id in an `ID` field of such a value. An
  arm or id reached any other way is not seen. Sites it leaves as other kinds:
  `secondHeldMarks` in `pkg/game/effectmark.go` (the effect-mark kind's per-game data), the shop
  book exclusions in `pkg/game/shopconsumables.go` (`SHOP-CONSUME-073`), the developer witness
  `pkg/sim/effectwitness.go`, and the book's 1..28 slot range in the book-slot sites.
