# Saved non-party current profiles

Original LOAD retains consumed current profiles and regeneration operands for
uniquely matched living non-party actors. New actions consume them; native
SAVE/LOAD retains the profile, fractions, action and explicit authority.

## Contract

FR-1. Both original LOAD doors project every uniquely matched living non-party
Unit, Humanoid and Human reached through the exact all-Player archive walk.
Persistent party members, zero identities, off-map and dead actors are excluded;
unmatched actors are counted. Ambiguous joins and unsupported current selectors
refuse the candidate atomically.

FR-2. Import current hit, three damage components, active physical damage kind,
defence, absorption, five protections, five damage-kind resistances, and the
Reaction/Mind/Spirit consumers. Import both signed regeneration periods,
both signed percentage modifiers and both full fractional bytes as one group.
Run no derive, effect replay, damage, regeneration, book refresh or clock step
on LOAD. Existing bounded pool import remains last.

FR-3. New attacks and regeneration consume these values. Signed regeneration
uses word reads, wrapping dword arithmetic, byte/word stores and signed upper
bound in the promoted order. Native saves retain the state and its source
authority, including during a new action and after a negative pool result.

FR-4. Native gear/skill/potion changes remain legal. A successful native sheet
rebuild explicitly retires source-current authority under the retained native
arithmetic policy; it must not pose as ROM1 re-derivation.

## Design

DD-1. The saved cell overlay precedes stock/rearm, saved books and current
profile import; the existing pool import is last. Both LOAD doors use this
order, and no staged candidate replaces the live session on a late refusal.
The narrow archive view does not use Party or the broad Character projection.
No original source blob or parallel shadow sheet is retained.

DD-2. Dedicated sim batch validation precedes every profile write. Existing
current fields are assigned directly, without SetDerived or SetCombat. Incoming
orders, mover state, clock/action-end phase and skills are outside this slice.
Books and canonical holdings come from their separate imports and remain
unchanged by current-profile assignment. Attack cadence and auto-hit
classification remain the constructed definition's values.

DD-3. Form73 persists source-current/native-retired provenance. Old native
forms default to native authority. Genuine form72 from exact1104 and form73
from the exact1107 WIP remain independent controls, not stripped new encoders.

DD-4. Source-current arithmetic is separate from the widened native policy.
Raw third-component selectors1..5 map to canonical protection indexes0,3,2,1,4.
Source-current target protection stays signed through the third-component
multiply; only the resulting component is clamped. Native/native-retired
target sheets retain the existing0..100 protection-input policy.
Retain raw pool u16 magnitudes on LOAD and read their low words signed during
regeneration. The existing native phase12 schedule and rate1 remain deliberate
limits. Health precedes mana with no repeated health gate. The existing native
death-consistency path handles a nonpositive health result; the original
callback is not reconstructed. Refuse zero mana divisors with nonzero maxima,
even full pools that a later cast could put in deficit. This is a safe admission
policy, not original fault recovery. DIV-634's over-maximum input policy remains.

## Authority and exclusions

Pinned research: 682184eb8e2571ee9b23d2ee614b9f9a459a63d3, moved forward from
1172d41a90ef928aa7345f76d0d3dcf0fc987bc9 at reconciliation with master
3bcf5912b8e7fd7e5b9450462c4e48707c9bffe5. Every claim below was reread at the
new pin; none is retracted or amended.
SAV-UNITFLD-049, SAV-UNITPROG-156, SAV-HUMRUN-444, SAV-HUMLOAD-445,
UNIT-DERIVE-003, SAV-HUMMUT-448, HERO-DMG2-029, HERO-CLAMP-030 and
HERO-RESIST-012 support the current view. Promoted SAV-REGENWIDTH-528 through
SAV-REGENWIRE-532 replace the old widened/normalized regeneration shorthand.

DIV-738 records native retirement, callback substitution, constructed clocks
and faulting-input refusal. Callback effects, loaded action-state16 lifetime,
idle multiplier and the first post-LOAD regeneration tick remain unclosed.
No incoming original clock is claimed restored. Full container state, complete
current-load import and a full-world original SAV writer remain mandatory
separate debt. This is not a complete original-world resume or modifier history.

Verification, candidate state and observable results are in verification.md.
