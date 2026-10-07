# Timed city potion through SAV

## Intent and authority

Prove that a potion used in the city survives ordinary F2 SAVE and cold LOAD, then expires once in the next mission. Owner direction requires one current-state SAV producer for native and original-city sessions.

The pinned MAGIC-CONSUME-142 establishes the installed health regeneration potion's +100 modifier and 960-tick duration. MAGIC-CONSUME-144 establishes storage of the remaining timer and applied modifier, followed by one removal at expiry. Its original-runtime reload and town-to-mission transfer remain Unknown. No new ROM1 behavior is asserted.

## As built

Mission entry restores a timer without applying its modifier again when the current Human load already includes that modifier. A party with base-only statistics still uses the existing apply-and-attach path. The scripted mission rebuild keeps the attachment. Invalid or repeated metadata restoration leaves the World unchanged.

City SAVE and mission entry also share construction of missing Spell identities. Both reserve keys against the same ordinary city actor/player records and retained residue, including raw unresolved references. Existing Spell keys and aliases stay intact. The common city record construction is read-only and advances no clock.

Mission entry takes its city object graph from the town being opened. A new campaign uses its fresh town and party without snapshotting the previous session. Ordinary city departure retains the shared current namespace. Preparation leaves the previous session intact until adoption.

The witness buys and drinks one installed potion through the headless App's shelf-to-character gesture. Its fixture selects existing merchant stock and supplies sufficient gold. Native input reaches chapter 30 through the production return route after a controlled victory prerequisite, without a previous city SAVE. The second input is the SHA-pinned original `2026-08-15/game0010.sav`.

Two F2 SAV cycles must preserve the party, payment, current modifier and unadvanced city timer. The ordinary Human modifier is compared directly with changed live state. The uninterrupted and twice-loaded parties enter their next mission; every tick must agree through expiry and one tick beyond it. No private save format is used by this witness.

## Proof

The first EN run passed both city SAV cycles and the ordinary Human modifier assertion. Original-city input then had no potion attachment at mission tick zero, while its modifier remained applied. `initializePotions` called the fresh apply path before the source derive hook was bound and ignored its refusal. `focused-en.jsonl` and `focused-receipt.json` record the failure, 64.401 seconds and zero remaining processes.

The native control failed even without a potion. Its first city SAVE changed generated `SavedObjects.Spells[].This`: one key was `0x62000060` on direct mission entry and `0x01000001` after LOAD. Entity fields, effects, holdings, actions, policy, item weights, script and cells agreed. `native-diagnostic.jsonl` records the exact field and byte difference. The common constructor repairs this mismatch; the full World hash assertion remains unchanged.

`TestReleaseCityPotionF2SAVAndMissionExpiry` passes on EN and RU for native arrival and the original city. Each drinks one purchased potion, performs two F2 city SAV/cold LOAD cycles and enters mission 30. Applied regeneration is +100 through tick 959, removed once at tick 960 and absent at tick 961. Uninterrupted and twice-loaded World hashes agree at every tick. The ordinary Human modifier is checked against changed live state. The unmodified native control also passes. This is the observable result: the temporary city benefit now expires after mission entry and a first city SAVE preserves the next mission's canonical identity state.

Focused sim/mapload tests cover invalid/repeated restoration, native/source Human state, plain/scripted mission starts and expiry. Game tests cover ordinary/raw/retained key collisions, aliases, distinct equal-valued Spells, source immutability, city mission seeding, roster and shop regressions. The existing thirteen-member F2 roster witness passes on both installs. Storyguard, gated population and architecture tests pass without a ratchet change. `constructor-{unit,en,ru,guards}.jsonl` and `constructor-focused-receipt.json` under `review/story1238-city-potion/` record the final focused batch: 101.449 seconds, exit zero, zero remaining processes and peak job memory 1,624,154,112 bytes. EN and RU game packages took 3.922 and 3.947 seconds. `gofmt` and `git diff --check` are clean.

The sole review reproduced NEW GAME refusing a previous pending city return because the seed read the old session's Snapshot. `TestReleaseCityNewGameSessionBoundary` covers native and original cities, settled and pending returns, with an explicit shared Standard autohealing profile. Preparation preserves the old town, party, gold, topology, original source and live hash; adoption opens mission 10. Its World matches a clean campaign with that profile through three ticks. The corrected potion witness retains every tick assertion through 961. `correction-profiled-{unit,en,ru,guards}.jsonl` and its receipt record PASS in 88.078 seconds, peak 1,626,947,584 bytes and zero remaining processes. EN/RU package times are 4.681/4.355 seconds; no selected test failed or skipped.

## Open debt

This slice covers the installed health regeneration potion. The current city policy carries the timer; the ordinary Human carries the applied modifier. Original attachment encoding and executable acceptance remain unproved. Other potion kinds, GUI observation, final merge gates and build promotion are not claimed. No script-gap census change is intended.

An independent default-profile NEW GAME gap remains: `applyFreshGameOptions` inherits the old `f.Carried` Player percentage before adoption. The original city produces AutoHealing 0 while a clean campaign defaults to 50. `correction-{en,ru}.jsonl` and `correction-unprofiled-counterexample_test.go` preserve the mismatch at World byte 57274. The session-boundary control uses the same explicit Standard profile on both paths and does not claim default-profile equivalence.
