# Independent Sack and item acceptance

The milestone-2 gate compares every discovered world SAV's raw Sack population
with restored ground loot and its retained complete Document, separately.
The registered EN/RU witness loads an original through the title menu and the
mission menu, drops one carried item into an existing source Sack, saves through
the ordinary menu, loads into a fresh FrontEnd and compares 20 continuation
hashes. This story adds acceptance instruments and one locator API. It changes
no game rule or save form.

## Authority and independence

`SAV-DOC-053` locates the Sack count immediately after the 4374-byte session.
`SAV-STREAM-013` supplies archive tags. `SAV-TOKEN-034`, `ITEM-SAVE-014`, the
unaffected item clauses of `SAV-MEMBER-036`, and `SAV-SPELL-044` supply the seven
layouts: Sack, Item, Weapon, Armor, Shield, Effect and Spell. The amended claims
and their partial retractions were read at the pinned knowledge snapshot.
`ITEM-STACK-003`, `ITEM-CONT-004`, `SAV-671` and `SAV-672` distinguish quantity,
signed unit weight, insertion index and the stored accumulator.

The expected side trusts `sav.Open` for decompression and structure location.
`DocumentObjectLocations` exposes only archive index, class and body start.
The independent test reader computes every count, tag, child edge, field and
record end from bytes. `DecodeDocumentDataWithOrigins` supplies only the join
from archive index to retained object index. Neither `GroundSacks`, decoded
record values nor a source-backed export supplies expected values.

The Document comparison checks complete field populations, Token bytes,
scalars, counted ordered references, aliases and subclass tails. The World
comparison checks every native Sack, its object registry entry and container,
ordered owned Item/Effect/Spell identities and fields, and every expanded live
item value. An unadopted Sack needs a named coverage reason; its retained
Document is still checked. Input read failures and importer refusals remain
separate and name their file. The gate prints full failing output.

## Proof

`TestMilestone2Sacks` discovers 102 files: 62 world saves, 39 city saves and one
unreadable input. The world population contains 271 raw Sack root slots and 560
reachable records: 271 Sack, 128 Weapon, 129 Armor, 20 Shield, 6 Item, 6 Effect
and no Spell. The source is the preserved save corpus; EN and RU are installed
consumers of those same recordings. All 271 Sacks are adopted; both comparisons
report zero mismatches, zero adoption gaps and zero importer refusals.

The unreadable input is
`2026-08-27/EXP-0261-owner-runs/game9000.sav`: `sav: magic is 42 73 67 26, want
"Asg&"`. It never enters the world/importer comparison.

`TestSacks1151IndependentReaderAndControls` uses a hand-written stream with all
seven classes, ten objects, shared/repeated Effect edges, an owned Spell, a null
Spell slot, nonzero opaque fields, an empty Sack, insertion index `0xffffffff`
and deliberately stale signed load `-17`. Thirty negative controls cover omitted
Sacks, Items, tails and locators; changed counts, fields, order and aliases;
expanded live values; unexplained adoption gaps; and malformed source bounds.
Expected fields never come from the production decoder.

`TestSacks1151RejectsCollapsedEqualEffectOrigins` supplies two distinct Effects
with identical 44-byte bodies. The complete ten-object baseline preserves both
Effects and its repeated child references. Removing one Effect and collapsing
the origin join produces a valid nine-object Document and one-Effect native
registry; both comparisons reject that loss. A second control collapses only
the DTO-to-native bindings and native Effect population, leaving the complete
Document intact. The World comparison rejects it separately. These two
controls bring the total to 32. Each collapsed native graph passes
`SavedObjects.Validate`, so invalid fixture state cannot supply the rejection.
Both comparisons require an injective archive-to-DTO join across distinct
reachable objects, and World also requires injective represented DTO-to-native
bindings. Repeated references to one object retain their aliases.

`TestReleaseMilestone2Sacks1151` uses preserved
`2026-08-15/game0016.sav`, SHA-256
`5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345`.
The source Item at archive index22 has weight2. A production drop command moves
it from the party pack to the unchanged Sack identity `0x2cb8100` at42,39;
the stored accumulator changes41 to43 and Contents gains that exact Item at its
end. Only the actor's approach position is controlled. No source file changes.
Both complete representations survive menu SAVE, fresh LOAD and continuation.

## Limits

This is bounded source-value and native-continuation evidence. It does not prove
original-runtime consumers of opaque Token/scalar fields, original Effect
execution or source-free world writing. Installed equipment Definition binding
is excluded from raw SAV comparison because it comes from the installed tables.
The corpus has no Sack-owned Spell; synthetic coverage is explicitly separate.
Unpublished classes cause a named independent-walk refusal. Back-references and
nondefault Sack tails have synthetic evidence only in this measured population.

An insertion index beyond the item count is valid append state. A stored weight
sum different from the item-derived diagnostic is preserved, never normalized
or refused. Current native nonmerge ground insertion changes the cursor to the
old end index; this is the existing engine policy, not a newly proven original
caller rule. DIV-962 and DIV-804 retain these consumer and lifecycle boundaries.
