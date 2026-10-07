# Current-object proof

The player-visible result is successful ordinary mission SAV after current
object changes, including four inputs that previously failed Group retirement.
All source AGS bytes were SHA256-checked before and after each EN/RU run.

| Input under engine/saves | SHA256 | Retired records removed | Current owners | Raw current dead roots |
|---|---|---:|---:|---:|
| 112323232.ags | 522d5cd01dea60a6d64637884f0427b58888c329be3458259c14d17bce5afe35 | 14 | 15 | 57 |
| 122323.ags | 903efb70bf0e042dc90a3b6b3bbd9ce36aca83534dbc662d4b12c678d0477702 | 14 | 13 | 59 |
| 123.ags | 957f66d791c83261931ac31489988414e14189b124cb2ccc2afa9d4502e9cde3 | 11 | 32 | 43 |
| save1234.ags | 199a94ce8024f784cd2cd0945d64081806b1e0f6823e75e72616181ebe01aa9c | 11 | 30 | 45 |

Each case checks ordinary SAVE, fresh-process LOAD, current price/weight/kind,
count and owner/slot order, next drop/pickup and a second SAV. All 43 original
held tuples remain. Complete raw dead-root checks include bodies represented
as live entities before SAVE; internal corpse-list length is not an equality
oracle. Runtime65's active crossing in122323 retains its next position.

The generated-item case preserves code3614 count1/price0/weight0 before the
distinct count1/price-1/weight1 slot, including the next mutation. The configured
boundary case acquires a native count65,536, an ObjectID0 item, consumes one
potion and performs source unequip/equip. Cold SAV carries ordered65,535/1
parts and preserves the total through the next drop/pickup and second SAVE.
Independent synthetic controls cover shared/repeated child references,
deep-copied split children, exclusive retirement cycles, held/live/dying actors,
incomplete documents, malformed references and active reservation protection.

Cloud-before and handoff cases pass through tick402, including paint, pulse,
expiry, the cold second SAVE at tick2 and a final SAVE after expiry. Mixed
delivery cases pass through tick32, second SAV and three further ticks. Existing
single-delivery controls for spells1/2/13/14 also pass in both installed roots.

## Commands and receipts

All Go commands use `GOCACHE=<seat>/.gocache`. Run from the
story worktree; `AGAINROM_ASSETS` selects `../gameversions/en` or `ru`,
`AGAINROM_SAVE_CORPUS` selects `../gameversions/saves`, and the absolute
`AGAINROM_SPELL_WITNESS_DIR` selects `../review/story1217-witness/{en,ru}`.

```text
go test ./pkg/game -run '^TestRelease(CurrentItemMutationSAV|CurrentObjectSAVContinuation|CloudDeliverySAVContinuation|MixedSpellSAVContinuation|SpellSAVContinuation)$' -v -count=1
go test ./pkg/game -run '^TestRelease(CurrentItemMutationSAV|CurrentObjectSAVContinuation)$' -v -count=1
go test ./pkg/game ./pkg/sim ./pkg/formats/sav ./internal/archtest ./internal/storyguard ./internal/divledger -count=1
go test ./pkg/game ./pkg/sim ./internal/storyguard ./internal/archtest ./internal/divledger -run 'TestCurrent|TestSavedActorPlacements|TestConstructSaved|TestOriginalDead|TestItemObjects1115|TestLiveTreeClean|TestLiveTree|TestDiv' -count=1
```

Affected game/sim/SAV suites and architecture checks passed. The initial
storyguard pass reported added references to numbered fixture helpers and
comment growth; helper renames and shorter nearby comments lowered the
committed counts to5,958 test identifiers and7,720,902 comment bytes. Final
focused tests and internal guards pass. New controls and the final raw body
checks were rerun after those changes. The author did not run the seat's full
54-package final chain or the final corpus census.

With full Git Bash PATH `/usr/bin:/mingw64/bin`, `check-no-game-assets.sh`
passes and `check-preserved-installs.sh` reports554 unchanged files.
`AGAINROM_IMPL=<story worktree> bash ../pipeline/check-div-claims.sh` parses
the ledger and reports140 rows with partly retracted citations. The changed
DIV-786 explicitly qualifies the amended Human clause of SAV-RECON-268.
Allocation sweeps before and after the four existing-row edits pass at the
unchanged DIV floor1368.

`go build -o ../review/story1217-witness/missionrun.exe ./cmd/missionrun`,
then `missionrun -mission 10 -trace -ticks 1` and mission20 both report zero
UNSUPPORTED nodes. The same commands against exact published basef7839db
also report0/0, consistent with `pipeline/milestone-baseline.txt`. This tool
runs its minimum64 ticks for that request. The story moves the ordinary SAVE
result above, not the script-gap census.

Untracked receipts, emitted SAVs, semantic JSON, hashes and next inputs live in
`review/story1217-witness/`. `MANIFEST.md` describes exact slots and populations;
`manifest.json` records every seed hash. `intermediate-failures.md` explicitly
labels transcribed writer-only resurrection and corpse-cache failures whose
original rolling log was overwritten. No original executable was launched.
The seat must measure the unchanged-success census and per-reason totals;
four targeted successes do not establish a full-corpus result.

## One bounded review correction

R1 reproduced a finite cursor 7 on six native slots, two of which held 65,536
units. SAVE expanded the container to eight slots while leaving cursor 7,
which changed the next acquisition from append to interior insertion.
The corrected shared actor/Sack projection emits cursor 9. It preserves the
distance beyond the old end and uses wide arithmetic with uint32 saturation.
The native World remains unchanged during SAVE.

Seven durable actor cases cover cursor 0, interior 1, old-end 2, finite 3/9,
maximum-minus-one and maximum values with two wide stacks. Three acquisitions
compare uninterrupted current World against cold SAV state, with another SAV
after the first acquisition. Four Sack cases check the same shared projection,
current counts/values/order/owners, accumulator and a second SAV. Existing
shared-child, split-child, unbound-item and wide-transfer controls still pass.

The EN/RU installed witness configures two stacks through scratch SAV, then
uses actual ScriptInstantAddItem to produce 65,536 units in each. It compares
all current owners directly against fresh-process LOAD, acquires 1911, writes
SAV again, starts a second fresh process, and compares acquisitions 1912/1913.
Both roots pass. The independent raw SAV decoder reads eight slots/cursor 9
in each first seed and nine slots/cursor 9 in each second seed. The last two
acquisitions preserve the finite-cursor ordering rather than merely retaining
the total quantity. This remains configured engine evidence, not an original
game quantity or overflow observation.

New receipts and seeds are under `review/story1217-correction/{en,ru}`;
`raw-cursors.txt` comes from the unchanged reviewer `inspect_cursor.go`.
The 101 prior author/reviewer evidence files remain protected by SHA256.
No original save or previous fixture was edited. The durable cursor 3 regression fails against the unchanged pre-correction
producer with saved cursor 3 instead of 5; its negative receipt is retained.
No new divergence ID,
research authority, or knowledge pin was used.

```text
go test -trimpath -count=1 ./pkg/game ./internal/storyguard ./internal/archtest -run '^(TestCurrentItemCursorExpansionKeepsContinuation|TestCurrentSackCursorExpansion|TestCurrentItemGraph|TestCurrentItemSplit|TestLiveTree|TestLiveTreeClean)$'
go test -trimpath -count=1 ./pkg/game -run '^(TestCurrentItemGraph|TestCurrentItemSplit)'
go test -trimpath -count=1 ./pkg/game -run '^TestReleaseCurrentItemCursorContinuation$' -v
```

The first invocation checks the new cursor regressions and internal guards;
the separate prefix selection checks neighboring item tests. The installed
invocation runs once with each EN/RU asset root, the preserved save corpus,
and a new absolute `AGAINROM_SPELL_WITNESS_DIR`. The seat owns final gates,
serialized landing and promotion; no second review or full author chain ran.
