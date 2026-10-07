# 0125-experience-from-use — tasks

Five tasks, in order, bottom of the stack upward.

## T1 — the inverse and the two carriers, in `pkg/data`

**FR-10, FR-11, FR-13, FR-14; DD-7, DD-8, DD-9; P-1.** Touch `pkg/data` only.

`recompute.go` step 2 already computes the forward curve per slot. Beside it, add
`SkillLevelFor(xp int32) int32` — the bounded downward scan of DD-7, over `0..100`,
comparing the same truncated `int32` the forward direction produces and calling the same
helper it calls. Its doc block must say the inverse is **authored**, why it cannot be wrong
given a monotone curve and a clamped level, and that the original's own inverse routine is
undecoded.

Add `SkillSlot int32` to `data.Combat` and fill it in the one place `Reach` is filled, from
the same weapon argument, by the same rule the package already uses to say which slot a
melee weapon's attack type names (DD-9). A bare weapon names slot 0.

Add `type Reward struct` carrying the capped Mind and the six per-slot experiences, and
`func (h Hero) Reward() Reward` as a thin accessor over `Recompute` in the exact shape
`Hero.Speed` and `Hero.Sight` already have (DD-8). It must compute nothing of its own.

**Tests.** P-1 in both halves, over the whole level range. That `Reward` agrees slot for slot
with what a recompute already produces. That a hero holding a weapon reports that weapon's
slot on his `Combat`, and slot 0 bare.

## T2 — the state and its byte form, in `pkg/sim`

**FR-1, FR-2, FR-3; DD-3, DD-4; AC-2, P-4.** Touch `pkg/sim`, and `pkg/mapload`'s form test
only where the peel chain below reaches it.

Add the five fields of DD-3 to `Entity`, with doc blocks in the file's own register saying
what each is and where it comes from. Widen `entityLen` by the 34 bytes they take, appending
the block at the **tail** so every existing offset stays where it is, and extend the encode
and decode pair. Take form version **35** and no other number — write the new version's own
doc paragraph in the sequence the constant's file already keeps, saying what arrives at 35.

Add the fault function of DD-4 and call it from **both** the constructor and the decoder, so
neither can produce what the other refuses: slot outside `0..5`, any negative slot
experience, `GainsXP` byte outside `{0,1}`. Mind and the experience value are carried whole
and refused nowhere. Fold the new state into the world hash.

**Tests.** AC-2 in both halves — round-trip equality, and the digest moving when any one of
the six slots does. Each of the three refusals, from the constructor and from the decoder.
Re-pin the form as DD-3's last paragraph instructs, by eye. Confirm the determinism scan
still passes with no float identifier, literal or import added (P-4). Do **not** rename any
test to spell 35.

## T3 — the payment, in `pkg/sim`

**FR-4 to FR-9; DD-5, DD-6; AC-1, AC-3, AC-4, AC-5, AC-6; P-2, P-3.** Touch `pkg/sim` only.

Add `payExperience` as an unexported method on the world (DD-5), taking the attacker and
target indices, the health removed, and whether the target was alive **before** the blow —
read before the subtraction, never recomputed after. Call it as the last act of the routine
that resolves a blow, on the arm that has already proved the removal positive.

Implement FR-5's five refusals as separate guards, FR-6 and FR-7 in the integer forms plan
DD-6 gives, in a width that cannot wrap. The game's own compiled constants — the halving,
the added one, and the two numbers of the Mind scaling — are named once each, in one block,
with the claim id from `provenance.md` beside each. No literal anywhere else.

**Tests.** AC-1, AC-3, AC-4, AC-5, AC-6. FR-9 by comparing the generator's state across an
advance that paid and one that was refused. **P-2 and P-3 by measurement, in a test file**:
sweep the reachable ranges, compare each integer form against a `float64` reference written
in the published real-valued shape, and record the divergence — none expected for P-2, and
for P-3 the count and the first case, which `verification.md` will quote. Sim test files may
use `math` and `float64`; sim **sources** may not.

## T4 — filling the five fields, in `pkg/mapload`

**FR-12, FR-13, FR-14; DD-2, DD-8, DD-9.** Touch `pkg/mapload` only.

Give the per-placement resolution block the three definition-borne numbers and the flag, and
fill them on all three of its arms: a creature row, a person row, and the unresolved arm that
substitutes the definition tier whole. A **person** gains; a creature and an unresolved
placement do not. Read the experience value and Mind off the row that arm already reads,
never off a literal. Take the credited slot from the combat block T1 extended (DD-9).

At the party mint, seed the six slot experiences and Mind from `Hero.Reward()` (DD-8, FR-13),
set the flag, and take the credited slot from the combat block the mint already derives.
Resolve DD-2's authored branch **here, in one expression**: a caster credits slot 0, a
fighter credits the weapon's slot. Name it in the doc block as the authored half of the
story and as the one expression a later reading moves.

**Tests.** A placement down each arm carries the numbers its own definition states. A party
member's minted experience equals what his levels account for. A member trained in a slot,
holding that slot's weapon, is minted crediting that slot.

## T5 — six numbers, live, in `pkg/game` and `pkg/ui`

**FR-15, FR-16, FR-17; DD-10, DD-11; AC-7, AC-8.** Touch `pkg/game` and `pkg/ui` only.

In `pkg/ui`, replace `UnitCharacter`'s trained-slot name and level with a six-wide array of
levels and a local constant for the width (DD-10) — no new import. Keep the field id and its
row literal in the authored layout exactly where they are, still paired with the experience
cell. Change the resolver so the row states the six in slot order, space separated, through a
helper beside the five-wide one, and states it **whenever the panel knows a character** —
never suppressed for being all zeros.

In `pkg/game`, stop taking the two experience-bearing members from the load-time map alone:
where the readout is built, overwrite the six levels and the total from the entity of that
tick (DD-11), converting through T1's inverse. Everything else in that block stays as it is.
Fix every caller and test the field change breaks; change no other behaviour.

**Tests.** AC-7 for a hero trained in one slot and for one trained in none. AC-8 by stepping
a world in which the subject lands blows and asserting the stated number rises, then holds
across ticks with no blow.

## Traceability

T1: FR-10, FR-11, FR-13, FR-14, P-1, DD-7, DD-8, DD-9 · T2: FR-1, FR-2, FR-3, AC-2, P-4,
DD-3, DD-4 · T3: FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, AC-1, AC-3, AC-4, AC-5, AC-6, P-2, P-3,
P-5, DD-5, DD-6 · T4: FR-12, DD-2 · T5: FR-15, FR-16, FR-17, AC-7, AC-8, DD-1, DD-10, DD-11 ·
SC-1 and SC-2 are the build stage's.
