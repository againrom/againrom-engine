# Verification

## Checkpoint

Implementation base: `b2e8945eba430eec2264170de3de7c7b5c869b9f`.
Research pin: `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`.
Final reconciliation includes the seat's exact landed and published 1099
master `95648c46dff60a4a15471294ed415e6a52f38d02`, retaining the actual form69
movement tail, 1098 structures and 1099's single correction. The earlier
parallel-preparation parent was `c79a854526e5fca2313622fef64a53d7afdfa6db`.
Candidate checks are complete below. Publication and independent review are
the remaining seat handoff; this document does not assert a story landing.
The installed 1097 compatibility regression below is corrected; its release
assertions remain unchanged.

## Native integration

Form 70 retains the 341-byte form-69 entity width: the 15-byte Human movement
tail remains at offsets 326..340. The new dead section adds 173 bytes per
record and a four-byte backward span before the fixed relation. Its original
74-byte prefix is unchanged, followed by a one-byte Weapon presence and its
98-byte fixed payload. Absence requires all payload bytes zero. Form 70 had
not been published at the earlier 74-byte checkpoint. Migration 69
copies its existing entity and trailing sections, then inserts an empty span.
Migration 68 still widens the entity tail without losing reserved scrolls.

Both independent pinned worlds now include a form-69 migration case. The
nonzero movement/scroll witness imports a negative signed Human speed, retains
a pending scroll and negative structure health, and matches 64 later ticks.
Seven simulation control digests, the map-load control and the native-envelope
SHA-256 remain fixed at their form-69 values after peeling only the new empty
dead section. Current-form digests change; historical decode fixtures do not.
The corrected 1099 `OriginalHuman.Retired` gob descriptor is integrated from
`32661cbd45d93162f04ce308ae523d0e016e121d`. The form-69 control envelope now
hashes to `8a00d92c3e30f90ae1d8071390ffa43271124f6c91955fa0e00d89f77ac7e7fa`;
current form70 hashes to
`fee8a3d790701f2089fef7834864180e288d2366b5347124b3344dcd445400d1`.
The pre-Retired form69 envelope was independently generated from clean
checkpoint `871ae938205f025b4054b027e4e462a97f19e462` and frozen as a compressed
synthetic decode-only fixture. Its decompressed SHA remains
`782e7e3ee1a38e156b120c28a115aa4fe3adc3c8838517030dfc11bc878ab16a`, and both
old and current form69 envelopes upgrade to the same current world bytes.

`go test -trimpath -count=1 ./pkg/sim ./pkg/game ./pkg/mapload ./pkg/formats/sav ./pkg/data`
passes without assets: five affected packages, 5622 test/subtest pass events,
zero failures. This is not the final `./...` chain. The outcome-only synthetic
fixture no longer inserts unrelated heuristic heads into the dead-manager list;
the real 1097 release assertions remain unchanged. This five-package run
preceded the bounded Weapon extension; its focused and release proof follows.

## Witnesses

`go test ./pkg/game ./pkg/sim -run 'TestOriginalDead1100|TestReleaseOriginalDead1100' -count=1`
passes with both lawful roots, using `AGAINROM_SAVE_CORPUS` outside the install.
The input is `2026-08-02/game0002.sav`, SHA-256
`b1ce079cc2b3f1bc101862c1dcf2afd8237421458b3e2474c4447efe5df2e761`.
The same source is imported through two roots, not two independent recordings.

The before/after production result is five fresh living authored actors versus
two loaded late corpses and three absent terminal actors. The independent
oracle is the pinned dead-record census, expressed as literal assertions:

| MapUnitID | Stage | HP | Cell x,y |
|---|---|---|---|
| 20 | 4 | -237 | 32,64 |
| 19 | 4 | -209 | 24,56 |
| 30 | 5 | -10014 | 12,54 |
| 36 | 5 | -10017 | 49,59 |
| 32 | 5 | -10011 | 52,47 |

All five carry timer zero, fine position 128/128, empty inventory, source
archive indices 90..94, and container tails 10000/0. Both production load doors
install the state at tick zero. The app has two saved sacks and gold 100; no
corpse retains fresh map inventory. Ordinary SAVE emits AGS after 33 driver
ticks. LOAD preserves the exact world hash and dead records, and the two
drivers match for another 64 ticks. Terminal source residues stay unchanged.
After the form-69/1098 reconciliation, the exact release test passes on EN and
RU with the final 173-byte stride and cut hash `06ab96efb5c5deaf`. The same report restores 18 structures and
the independent WIN/LOSE and Player outcome fields. No desktop frame is claimed.

Focused sim tests additionally witness atomic invalid-batch rejection, no
RNG/gold/XP/drop replay, active corpse decay independent of immutable source,
passage through its cell, retained terminal state after final decay, reserved
IDs, closed attack/credit references and malformed native rollback. A literal
74-byte prefix, 99-byte absent Weapon arm and four-byte length oracle checks
the native layout. A separately transcribed Weapon sentinel covers all named
members; every one of the 98 native payload bytes changes the world hash and
survives decode/remarshal. Two different Weapon identities/T0E values remain
distinct. Unsupported arms and noncanonical presence/residue fail atomically.

Allocation sweep before and after DIV-682: 32 ledgers, missing answers 0;
floor DIV-690. DIV-683..689 are unused.
The integration ledger resolution was bracketed by the same complete sweep:
32 ledgers, missing answers 0, floor DIV-690 before and after.
The terminal-Weapon amendment was separately bracketed: 32 ledgers, missing
answers 0, current seat floor DIV-698 before and after. It consumes no new ID.

## Terminal Unit Weapon compatibility

The first strict-empty implementation refused the previously loadable source
at `original dead actor 0x2ca4880: unsupported effects/carried/worn contents`.
The source is `2026-08-02/game0009.sav`, SHA-256
`60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6`.
No release assertion was removed or weakened.

A read-only shared archive walk measured the dead list at body `[33995:48603]`:
22 entries and 22 unique objects, none also referenced by any Player actor list.
Entry 0 is Unit MapUnitID 59, archive 109, body offset 34001, stage 5,
signed HP -10007, timer 0, runtime 0, cell 32/27, fine 128/128. Its effect count
and inventory count are zero. Its non-null HeldWeapon (archive 110, offset 34539)
causes the projection's aggregate worn count 1. The Weapon has zero effects,
runtime 0, F40=61720, F42=1, F44=2, F46=15, F48=150 and W50=4; the other
F45/F47/F4A values are zero. Inventory is present with tails 10000/0.
Entry 1, Unit MapUnitID 60, archive 111, likewise retains one HeldWeapon at
stage 5, HP -10001 and timer/runtime zero. Its Weapon is archive 112 at offset
35272. The two Weapon source identities are 0x2ca4c80 and 0x2ca7640, and their
T0E values are 715 and 28769. The other 20 entries are stage 4
with timer zero and no effects, inventory or worn contents.

The approved narrow follow-up retains only this terminal Unit held-Weapon arm
as nonplayable source provenance. Its 98-byte payload is archive index2,
Token37, Item scalar12 and Weapon members47; zero Effects and null WeaponSpell
are required input facts, not defaults. All 17 scalar names, three fixed raw
member lengths, zero effect count and absence of references are checked before
projection. Source bytes `[34539:34641]` and `[35272:35374]` independently witness
the complete native payload, omitting only the required zero count/null tag.
No whole object/SAV blob is stored, and no opaque member is given gameplay meaning.

`go test -trimpath -count=1 ./pkg/formats/sav ./pkg/sim ./pkg/game -run 'TestDeadWeapon|TestOriginalDead1100'`
passes without assets. Both EN and RU pass the two `TestReleaseOriginalDead1100`
tests. The new all-22 test observes fresh authored living22 versus loaded
late corpses20 and terminal absent2 through both load doors, at tick zero.
After actual Continue, 33 production ticks, native SAVE and App LOAD, the cut
hash is `9a6b31aab4c68a39`. All 64 following production ticks match; every Weapon
member stays unchanged, Units59/60 remain absent, sacks stay10 and gold600.

Both roots also pass the unchanged 1097 saved Victory and synthetic defeat
tests. Victory's native hash is `1e7178f5af1eb2bc`; its single mission reward
changes gold600 to1100 with party2. Defeat's native hash is `c24e1992c8c9a3ce`,
with no XP/purse change or recovery. These are headless production witnesses,
not original-process or desktop observations.

## Mission census

The worktree-built `cmd/missionrun`, using EN assets with `-trace -ticks 1`,
reports mission 10 UNSUPPORTED=0 and mission 20 UNSUPPORTED=0. Both invocations
exit 0. The runner advances one frame to simulation tick 64. The seat's
`pipeline/milestone-baseline.txt` records script populations 16/27/12 and
14/15/11 checks/instants/triggers; it carries no separate UNSUPPORTED count.
The source populations are unchanged. The seat independently measured prior
master at UNSUPPORTED=0/0 on both roots; its logs are
`review/story1099/mission-{en,ru}-{10,20}.log`. The local EN result is unchanged.

## Limits

This is not full SAV restoration or an original world writer. Early stages,
nonzero timer, effects, carried contents, worn arms other than the bounded
terminal Unit Weapon, non-centered positions, invalid
bounded health/runtime combinations, dynamic actors and ambiguous or missing
authored joins are refused. Source keys remain typed provenance; the complete
original object graph and terrain rebind are not reconstructed. No original
process, GUI input, install write or research change was performed. Headless
production output is witnessed; no on-screen GUI observation is claimed.

## Final candidate checks

Code candidate `6c4c9003772eb11af5b79b9b55f870c71aa69208`, after exact landed
master reconciliation, passes `go test -trimpath -count=1 ./...` once.
Changed Go files are gofmt-clean; `git diff --check` is clean.
`scripts/check-no-game-assets.sh` run from this worktree prints
`check-no-game-assets: clean (tree scan)`.

One final scoped test invocation per EN/RU root passes all six top-level
production witnesses (zero skips): both `TestReleaseOriginalDead1100` tests,
both `TestReleaseOriginalOutcome1097` tests, the three-arm
`TestReleaseOriginalStructures1098HealthRuinAndNativeSave`, and
`TestReleaseImportedFighterTrainingUsesSAVAndFreshProcesses`. The latter retains
four fresh processes, SAV-AGS-SAV identity, school prices200/220 and mission30
pool/combat/movement/native continuation. All dead-state cut hashes above
are unchanged by 1099's additive gob descriptor correction.

`check-div-claims.sh` selects286/286 live rows,420 cited IDs and70 rows citing
partially retracted claims. This is a diagnostic, not a zero-hit gate.
DIV-682 explicitly relies on the unaffected Item/Weapon clauses of
SAV-MEMBER-036; that claim's Human retraction is not used as Weapon authority.
No new allocation or research pin change was made. No files are deleted
relative to the exact landed master, and no Co-Authored-By trailer is present.

The final proof-only commit changes this document, not the tested code.
The seat will run the complete paired release gate after independent review;
neither that gate, review, landing nor build promotion is claimed here.
