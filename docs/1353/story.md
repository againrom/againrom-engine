# Current skill SAVE consistency

## Contract

An ordinary mission SAVE of a native table-backed book projects Range from the
same signed level and Mind words that it writes for the Human. Fire school110
with Mind10 keeps live reach13, writes UA6=100/U114=100/UD4=10 and Fire Ball
S09=12, restores live reach13 in a fresh process and writes12 on the second SAVE.
Mana and Defensive instance operands remain independent. An imported book keeps
its cached operands; a changed book cannot rewrite an aliased weapon Spell.

Native skill effects and awards use the known fighter/mage class, independently
of current mana. A fighter with positive mana selects kinds26..31 and weapon
training; a mage without mana selects kinds32..37 and school training. Ordinary
U4C is the class authority at LOAD. Equipment changes, the next eligible award
and a second SAVE retain that class and the released trained base.

Authority: knowledge `fe7d269a9b7e0ccbe75607ae320c156cbb051f07` (k178).
`MAGIC-REACH-178` establishes the signed power0..100 stored-range producer;
`HERO-CLASS-020` establishes the class-selected skill families;
`HERO-ITEMSKILL-096` establishes the award arms. `HERO-CLASS-013` retains its
published constructor frontiers. Popup underflow and the owner's live power255
policy remain separate from the ordinary range producer (DIV-2221).

## State and compatibility

`Entity.NativeClass` has explicit presence and fighter bits. Native party and
resolved Human constructors seed it from the known profile. Source-backed Humans
keep `Source.Fighter` authority and clear native class on source publication.
The native bonus fold, immediate awards and delayed skill attribution read one
class selector. Casting mana eligibility retains its existing contract.

Native class presence requires a Human actor. Current party restoration adopts
ordinary U4C only from a Human record; a native Unit base policy cannot create a
class. Creature setters and binary admission reject Human class presence.

Optional binary form107 stores a sorted sparse actor/class footer around the
unchanged predecessor. An absent class table keeps historical bytes and hash.
Decode bounds counts and spans, rejects duplicate, unordered, unknown or
source-backed actors, checks canonical bytes and adopts only after validation.
Forms105,106 and structure-blocking admission remain composable.

Current SAV stores class presence, not a competing fighter value. A known class
comes from ordinary U4C before native source retirement; party and roster restore
the same authority. New saves of historical actors carry `LegacyClass` absence.
Old current SAV without that marker adopts its known ordinary party/roster class.
Historical binary actors without class retain the old mana-based skill default
(DIV-2386). No gob field is renamed and no original SAV field is added.

`RefreshOriginalBook` materializes only BookLegacy during ordinary mission SAVE
on a detached actor. Named levels project to100, level/Mind narrow to signed
words, power clamps0..100 and cached Range retains its native byte domain. Mod
formula tables keep their existing producer. Imported present books keep their
instance values. The graph writer subtracts removed book edges before deciding
whether a Spell is exclusive; other books and weapon owners prevent a changed
shared Spell from being rewritten. Cold legacy book-root admission accepts the
ordinary projection through the existing live-power upper bound.

Snapshots retain the live BookLegacy range domain. Ordinary SAV retains a
distinct live graph Range with a sparse current-object operand anchored to the
whole ordinary Spell value. An ordinary ID, Range, Mana or Defensive edit wins.
Every Item view of an applied child operand is rebound to that child. Missing
operands keep the prior ordinary-value route; malformed operands are refused
before publication.

## Evidence

External proof is `review/story1353-current-skill-save/`. RED EN/RU used the
installed mission20 F2/SAV route: low-Mind S09 was13 instead of12, power-ceiling
S09 was14 instead of13, and the ordinary fighter bit could not select the worn
bonus while mana remained positive. No-bonus control passed before the fix.

- `TestReleaseCurrentSkillBookRangeProjection`: low Mind, no bonus, power
  ceiling, unchanged World hash, fresh-process LOAD, ticks and second SAVE.
- `TestReleaseCurrentSkillClassOrdinaryBit`: an ordinary U4C edit with unchanged
  continuation metadata, actual unequip/equip/strike, one trained raise, fresh
  process LOAD, equip and second SAVE.
- `TestReleaseCurrentSkillInverseClass`: ordinary mana-less mage, actual rearm
  and two SAVE/LOAD cycles. SIM independently covers its eligible school award.
- `TestReleaseCurrentSkillImportedBookWeaponAlias`: independently edited
  S09/S0A/S0C, two unchanged alias cycles, actual skill-cache refresh splitting
  only the book edge, and cold LOAD of the split.
- `TestReleaseCurrentSkillHistoricalClassDefault`: explicit absence versus old
  current fields, ordinary U4C authority and two SAVE/LOAD cycles.
- `TestNativeClass*` and `TestOriginalBookRangeUsesOrdinarySignedWords`: both
  class arms, post-load award/hash parity, old bytes, combined footers, corrupt
  population refusal and signed-word/range controls.
- `TestReleaseAreaDamageLifecycle1164`: fresh Fire/Poison, signed Poison zero
  crossing, ordinary menu SAVE, cold LOAD, current graph and post-load ticks.
- `TestReleaseOwnerSiegeHireSAVLoads`: both owner siege-hire SAVs, movement,
  unchanged ordinary Unit rows, second SAVE and cold LOAD.
- `TestCurrentSpellRangeProjectionKeepsLiveValueAndOrdinaryEdits` and
  `TestCurrentLegacyBookSnapshotAndOrdinaryRanges`: distinct native/ordinary
  range, four ordinary edits, alias views and six malformed current operands.

Focused EN/RU, ordinary SIM/mapload/GAME and guards are candidate checks.
Sole review, full chain, EN/RU release and source-bound M2 are landing checks
owned by the seat. These witnesses establish engine behaviour only.

## Remaining debt

The active spell skill-bonus producer remains Unknown. The imported-sheet
`NativeTrainingNeedsProducer` guard is unchanged: no residual bonus or lost
training history is inferred (DIV-675). Refresh before every native order and
universal normalization of imported caches remain unestablished. This range
change covers the audited mission BookLegacy materialization, not every city
constructor or a native-runtime LOAD of an engine-written SAV. An unedited
original fighter-with-mana producer remains unobserved.

DIV-2386 records the deterministic historical class default. Reserved
DIV-2387..2389 are retired unused and cannot be reused. Game version0.83.0
remains unreleased; starter0.4.0 is unchanged.
