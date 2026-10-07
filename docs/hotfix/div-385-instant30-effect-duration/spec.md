# Hotfix `DIV-385` — canonical instant-30 effect duration

## Result

Script instant 30 changes the remaining duration of every matching attached effect on its referenced
unit. Attached effects are canonical simulation state. Presentation marks do not select or own the
write.

## Behaviour 1 — target resolution and dispatch

The compiled instant retains one unit reference, one spell parameter and one duration parameter.
The script pass dispatches the instant only through its existing opcode-30 arm.

An absent compiled unit reference or a referenced entity that is no longer in the world produces no
state change. The command does not select another entity and does not fall back to an entity index.

## Behaviour 2 — canonical matching and write

The target is the resolved entity id. The spell comparison narrows both the attached record's spell
and the authored spell parameter to 8 bits. The duration store narrows the authored duration to 16
bits.

The mutator scans the complete attached-effect slice and writes every record whose target and
byte-sized spell match. It does not stop after the first record. It preserves slice order, record
count, caster, caster presence, spell, kind, mode and magnitude. It changes only `Remaining`.

An empty attached-effect slice, a different target or a different byte-sized spell produces no state
change. The mutator never creates an attached effect. A duration of zero is stored. The record may
be saved and loaded in that transient state, and the existing effect step removes it on its next
pass.

## Behaviour 3 — presentation separation

`Entity.SpellFX` and `Entity.SpellFXSpell` are presentation state. Instant 30 neither searches those
fields nor changes them. A matching canonical record is updated when the fields are zero or name
another spell. A matching presentation mark without a canonical record changes nothing.

The existing effect step may populate an empty presentation mark from a live canonical record. That
derivation does not make the mark canonical state.

## Behaviour 4 — countdown, hash and persistence

The existing attached-effect step reads the authored `Remaining` value. Values from 1 through 9600
decrement normally. Zero removes the record. Values above 9600 keep the existing no-countdown rule.
Continuous-effect cadence, removal effects, magnitude application, caster attribution, replacement,
annihilation and stacking rules are unchanged.

The existing attached-effect binary record stores `Remaining` as a 16-bit value. Its decoder accepts
zero `Remaining` while still rejecting an empty spell or effect kind. The world hash is computed from
the complete binary form. A save after instant 30 therefore restores every authored duration,
including zero, the same hash and the same next-step result. No field is added and `formatVersion`
does not change.

## Behaviour 5 — shipped campaign matrix

The shipped campaign contains two instant-30 actions on mission 90 per lawful root. Actions 33 and
34 target spell 20 and author durations 60000 and 1. Each exact compiled action runs through
`NewScript`, the normal script pass, `StepTraced` and the canonical-effect oracle.

Both actions have disposition `PASS` on the EN and RU roots. The claim-backed operation denominator
remains 52 rows. No other operation disposition changes.

## Exclusions

The hotfix does not change instant 29, group sub-command 17, effect magnitude or mode, casting,
effect creation, AI, player input, inventory, equipment, presentation art or campaign progression.
It allocates no story number, divergence id or byte-form version.
