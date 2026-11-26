# Corpses decay at the same health thresholds in both games

## Result

A body keeps its fallen appearance at exactly -10 health even after dropping its loot.
In both games the next three stages begin below -10, -20 and -40 health.
Saving and loading preserves the drop. Older saves keep the stage their bodies had reached.

## Authority

- Owner, board card `o-r1-corpse-stages`, «Как рекомендуешь». Save teardown independently and retain older saves.
- HERO-207 (High per the brief) and R2-ENGINE-1825 (High for strict comparisons) give the ladder.
- SAV-DEADLOAD-126 as cited in DIV-1187 distinguishes stage, health and timer. DIV-5252 covers the saved flag.

## Rules

1. Both games change stages below -10, -20 and -40 health and reach the terminal stage below -600.
2. A body at -10 drops its loot once, including across a save and load.
3. Older saves keep the stage at -20 and -40 and resume with the loot already spent.
4. A living actor carries no corpse flag and its world keeps its previous byte form.

## Code

| kind | its one builder | change |
|---|---|---|
| decay ladder | `decayLadder.stage` | strict comparisons for both profiles and the existing hero floor |
| corpse teardown | `clearFelled`, `decayPass` | both dispatch points use `readyForTeardown` and `markCorpseDecayed` |
| death loot | `dropTerminalLoot` | successful transfer sets the flag before a body can drop again |
| dead list | `corpseList`, `rootDeadActors` | a torn-down fallen body joins the ordinary dead roots |
| SAV | `ExportCurrentSave`, `projectCurrentActions`, `restoreValues` | one optional actor value in the existing actions record |
| world byte form | `encodeInto`, `unmarshalForm` | sparse section for torn-down bodies still at stage one |

## Proof

| test | rules |
|---|---|
| `TestBothGamesCorpseStagesBeginBelowTheirThresholds` | 1 |
| `TestBothGamesBodyAtMinusTenDropsLootOnceAcrossLoad`, `TestSAVCorpseDecayedAtMinusTenDropsLootOnce` | 2 |
| `TestOldSaveDeadUnitsAtThresholdBoundariesLoad`, `TestSAVOldSaveDeadUnitsAtMinusTwentyAndMinusFortyLoad` | 3 |
| `TestCorpseDecayedByteFormRoundTripAndRefusal`, `TestMarshalledBytesArePinned`, `TestSAVCorruptCorpseDecayedOnLivingActorRefused` | 4 |

Release tests at the release: `TestReleaseNewCorpseSAVContinuation`, `TestReleaseSecondDeathDropUnitsRows`.

## Open debt

- DIV-5252. The original SAV location of an independent teardown flag needs evidence.
