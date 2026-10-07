# Native trained skill base

## Contract

Native Class0 actors retain six trained inputs independently of their bounded
effective levels. A base of 100 with +200 reads 255 and banks no XP. A base of
90 with +200 or 40 with -100 can train while the effective level stays 255 or
0. Unequip, rearm, mission return and the next mission use the trained input.
This is the trained-base debt in `docs/1322/story.md`, not closure of D16.

Authority: knowledge `013d299b25971475e46ece300d0f959dfd38bc6b`;
`HERO-SKILL-009` distinguishes base and effective slots, `HERO-ORDER-014`
places the base before modifiers, and `HERO-SKILLUP-073` stops awards at the
training cap. DIV-2217..2219 retain the owner's effective bound and ordinary
SAV compatibility policy. These tests establish engine behaviour only.

## State and compatibility

`Entity.NativeTraining` has an explicit presence bit. Native party and roster
constructors seed it from known Hero inputs. A successful training award updates
the base. The derivation cache compares it as well as effective levels. The
award reader, both live rearm readers, post-award recompute and both carry
readers use the same accessor. Class2 awards retain their source Base arithmetic.

Optional binary form105 stores a sparse sorted actor/base table around the
unchanged prior payload. Hash includes that table. Decode bounds counts and
spans, rejects duplicate or unknown actors and checks canonical encoding before
adoption. An absent table keeps the historical bytes and hash.

Historical binary actors without a known base retain the old inverse:
effective minus worn bonus, named slots floored at zero. The first successful
raise materializes that fallback. This is deterministic and cannot reconstruct
history lost at a bound. XP is never inverted. Current SAV reads known native
party and roster bases from ordinary U114 and existing anchored lifts; a new
`LegacyTraining` absence marker preserves an explicitly absent historical base.
Old current SAV with no marker uses its known ordinary base and anchored lifts.
No gob field is renamed and no original SAV field is added.

SAVE projects the current trained named bases into U114. UA6 remains at most
100. UD4 reads current worn skill bonuses, so a second SAVE after unequip cannot
restore the previous bonus. A restored native member retains native town
arithmetic. Existing lifts preserve wide inputs while their ordinary anchors match;
an edited ordinary Base word invalidates the old lift. General keeps its
existing anchored policy because it has no separate original Base word.

## Acceptance and evidence

- `TestNativeTrainingAwardsAtEffectiveBounds`: real award sink at cap100,
  effective255 and effective0; binary/hash identity and the next award.
- `TestNativeTrainingDistinguishesIdenticalEffectiveLevels`: independent base
  affects hash while effective255 stays fixed.
- `TestNativeTrainingLegacyFallbackAndKnownBase` and
  `TestNativeTrainingWireRejectsMalformedPopulation`: deterministic old default,
  known-base replacement and transactional malformed input refusals.
- `TestReleaseNativeTrainingSurvivesBoundedBonuses`: installed reproduction.
  Before the change EN carried 55 instead of100 and100 instead of40.
- `TestReleaseNativeTrainingAwardEquipSaveAndTown`: actual strike/award, cache
  invalidation, equip/unequip, ordinary SAVE, fresh-process LOAD and next unequip,
  second SAVE, town carry/equip/SAVE/LOAD and next mission.
- `TestReleaseNativeTrainingOrdinaryBaseOverridesLift`: wide70000 base survives;
  editing U114 to70 discards its stale lift and second SAVE writes70.
- `TestReleaseNativeTrainingJoinedRoster`: controlled production GiveUnit,
  same-tick join/carry, roster SAVE/LOAD, town return and next mission at255.

External evidence: `review/story1351-trained-skill-base/`. Focused installed EN
and RU witnesses and ordinary touched-package tests are lane gates. Sole review
and final EN/RU, M2 and full chain are landing gates run by the seat once.

## Imported live sheet correction

The owner `saverood.sav` retains known native bases `[0 70 45 15 82 63]`
and effective skills `[0 70 100 100 86 64]`. The worn-item producer cannot
reproduce that sheet. A Heal award previously consumed retained XP, raised
base15 to16, replaced effective100 with16 and reduced max HP110 to108.

For an imported session (retained clock or object carrier) with an explicit
native base, a mismatch between a named effective skill and the known worn-item
calculation defers awards
before XP changes. Cached derivation and a level-only rearm retain that live
sheet while the known base, worn bonus and potion state remain unchanged.
A known producer update that makes the sheet reproducible resumes ordinary
awards. This is the bounded
DIV-675 engine policy, not recovery of the unknown active modifier producer.
No residual bonus or trained history is inferred. Existing session provenance,
base and sheet survive SAVE and cold LOAD without a new flag or wire field.

`TestImportedNativeTrainingKeepsAnUnreproducedSheet` constructs an independent
base and worn source, checks unchanged XP/hash at award and cold decode, then
checks award recovery after a known producer update. The cap100, effective255
and effective0 controls also run with imported-session provenance.
`TestRoodThreeOwnerSaveRescueContinuation` checks cached refresh, level-only
rearm, actual party Heal, retained sheet and max pools, SAVE, cold LOAD, walking,
Guard and second SAVE on EN/RU. The joined-roster control still accepts an
explicit base and equipment producer update.
The independent terminal-actor compatibility witness now peels outer form105
before inspecting the retained form100 payload; its actor assertions stay intact.

## Remaining debt

The fighter-with-mana effect-family mismatch, lifted spellbook Range SAVE,
active spell skill-bonus producer and its native witness stay outside this
story. Ambiguous history already lost from old bounded native state cannot be
recovered. The bound255 remains the owner's accepted deviation.

Reserved DIV-2374..2377 are unused: the correction fulfills existing
DIV-2217..2219 and introduces no new ROM1 policy claim.
