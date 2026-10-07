# City SAVE keeps the complete hired roster

## Intent and authority

A legitimate city SAVE retains every current party member, including hired squads whose total exceeds twelve. Owner direction requires SAV at every city save point and permits authored campaign data. No ROM1 party maximum is asserted. DIV-1444 names the remaining original-runtime question.

## As built

The common city producer bounds the explicit roster by archive object space. The current city reader bounds population against represented archive objects, then checks distinct member identities and actor bindings. Legacy reconstruction of absent squads from flags keeps its twelve-member per-type guard. No wire field or simulation rule changes.

## Proof

The pre-fix production SAVE in `TestCityRosterSaveTwelveAndThirteenTotalMembers/13` failed with `native city save has 13 party members`. `review/story1237-city-roster/red-admitted.jsonl` preserves that assertion.

The focused tests pass for twelve and thirteen total members, thirteen hired actors in one squad, and two squads hired in both orders. Two ordinary SAV cycles preserve unique actors, group order, party identities, item/effect/spell records, gold and hire flags. Live and cold return/rehire agree. Malformed current metadata and the old flag-only overpopulation guard reject atomically. `green-defined.jsonl`: package 0.562 seconds, bounded run 61.023 seconds including compilation.

`TestReleaseCityRosterMultipleHiredSquadsF2SAV` passes on EN and RU through the headless App's F2 route. Chapter 50 offers types `[6,14,13]` with counts `[4,3,4]`; those eleven hires plus two characters produce thirteen members. Two SAV cycles, cold return/rehire, next mission and three tick hashes agree. `release-en.jsonl` and `release-ru.jsonl`: 1.203 and 1.238 seconds. This is the player-visible result: an offered roster rejected before now saves and continues.

The gated population and story guard pass. The comment ratchet falls by 144 bytes. The paired release/guard batch took 67.022 seconds; the corrected guard took 2.842 seconds. Every bounded receipt records zero remaining processes. `gofmt` and `git diff --check` are clean. The release witness compares installed gameplay state, not original executable acceptance; no GUI observation or original-runtime LOAD was performed. No script-gap census change is intended.

## Open debt

DIV-1411 remains open: unavailable city provenance falls back to constructed group order. A synthetic companion with no matching type definition reproduced that fallback; `review/story1237-city-roster/GROUP-COUNTEREXAMPLE.md` retains the input and logs. The complete-roster tests use a matching definition and retain exact group-order assertions. The original runtime's accepted party size and legacy flag-only reconstruction above the retained guard remain unknown. Sole review, final merge gates and build promotion are pending.
