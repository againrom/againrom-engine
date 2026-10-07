# Companion spellbook change through city SAVE

## Intent and authority

A companion's spellbook change made in the city survives two F2 SAVs with a cold LOAD after each, and the changed spell casts in the next mission, on the EN and RU installs. The change covers a merchant book learned by a mage and a school Train purchase. Owner direction: one current-state SAV producer for native and original-city sessions.

Authority is the claims pinned for the earlier spellbook work: `MAGIC-BOOK-002` (the book is a sparse array of spell instances; learning is the teachSpell effect), `MAGIC-SPELL-001` (a spell instance stores range, Defensive and mana cost), `HERO-SKILLBUY-076` (the school Train purchase). No new ROM1 claim is asserted.

## Audit of existing tests

Name and body search of `pkg/game` tests found no test with the full shape. `TestReleaseMageSpellbookTrainSAVEAndContinuation` trains in the city and saves once. Its second SAV is never cold loaded, its mission check compares a 64-tick state trace and issues no cast, and it changes no learned spell. `TestReleaseCityQuickSpellsCurrentWriters1167` changes shortcut bindings on a synthetic mage, not a companion's book or the next real mission. `TestCityBookLearningCommitsPurseAndFreshSlotThroughSAV` covers one SAVE and cold LOAD with no mission cast.

## As built

No writer or loader defect was found. The witness passes on the unchanged production code; this story adds proof, two divergence rows and no behaviour change.

`TestReleaseCityMageBookChangeF2SAVAndNextMissionCast` runs two origins on each install: the original city `2026-08-15/game0010.sav` (SHA256 `89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4`, a source-backed book) and a native arrival (an absent source book, DIV-1687).

1. F2 SAVE of the unchanged city, kept as the loss-control input.
2. A probe city learns every merchant book to pick a spell whose ground cast the next mission admits; the witness itself learns only that one.
3. The mage learns the spell through the production member picker, a shelf-to-table stage, Buy, and a pack-to-figure release (the shelf-to-figure purchase no longer exists), then trains the cheapest affordable school slot through the App Train control.
4. Two F2 SAV cycles. Each compares the saved spell membership and, where a source book exists, every saved spell instance against the live book, then cold LOADs from the main menu and compares the restored book to the live book.
5. The uninterrupted and twice-loaded sessions enter the next mission. World hashes agree at tick 0. Each selects the mage, picks the spell from the open book and clicks a ground cell visible in both sessions. Both queue the same command, run in lockstep to the cast event with equal World hashes every tick, and the cold mage's mana falls.
6. Loss control: the pre-change SAV, cold loaded and entered into the same mission, does not know the spell and the rules refuse the cast. The witness also requires the pre-change SAV to lack the spell bit, so a writer that dropped the change fails steps 4 and 6.

## Proof

Receipts under `review/story1259-sav-m8/`: `witness-en.txt` and `witness-ru.txt` record PASS for both origins. On both installs the mage learns spell 2, trains school slot 1 (gold 70000 to 69800), and the cast lands 9 ticks after input. `gofmt`, `git diff --check`, `internal/storyguard` (comment bytes unchanged), `internal/gatedtests` (the new name is in `testdata/population.txt`) and `internal/divledger` pass. The ordinary `go test` of `pkg/game` (no install root, GOMAXPROCS=4, -p 2) passes in 361.5 seconds (`gates.txt`). The earlier run with an install root set exceeded the default 10-minute timeout under machine load and is not a gate.

## Open debt

- DIV-1687: a native companion has no source book instance. Its learned spell persists as membership and casts from table-derived parameters; first-book allocation stays DIV-690.
- DIV-1688: no original-runtime load of an engine-written changed book exists.
- A quick-spell slot change is covered by the earlier quick-spell witness and is not repeated here.
- The witness proves one learned spell and one school slot, not every merchant spell.
