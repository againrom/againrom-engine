# 0162-wear-rule — provenance

Research pin: `e1b27fd`. All four rows come from EXP-0168 except `SHOP-SCREEN-036`, which EXP-0164
published and EXP-0168 amended.

| Claim | Grade | What the spec takes from it |
|---|---|---|
| `ITEM-WEAR-055` | High | FR-1's predicate, and FR-2's identification of parameter 15 as the item side. The polarity — fighter is bit 0, mage is bit 1 — is fixed at instruction level by the character constructor's addend/base split, not by naming. |
| `ITEM-WEAR-056` | High for the column identity and the per-row values; Medium for reading the partition as fighter/mage from the data alone | AC-2's counts and named rows. The Medium clause is not load-bearing here: the fighter/mage reading is taken from `ITEM-WEAR-055`'s instruction side, and this row is used only as the shipped distribution. |
| `ITEM-WEAR-057` | High for the two sites and the refusal behaviour, High for the simulation's silence, Medium for the client being the only enforcer anywhere | FR-3's refusal shape, FR-5's negative requirement, and DD-5's choice of seam. The Medium clause is why the spec discloses that a save or a script still produces a mismatch: this build mirrors the enforcement the original has, not a stronger one. |
| `SHOP-USABLE-040` | High for the routine and the background arms; Medium for the purchase being unaffected | FR-4's cell background and DD-7's nesting. The Medium clause is disclosed in the spec: this build's shop still sells an unusable item. |
| `SHOP-SCREEN-036` | High for the five arms | DD-7's arm order, and the fact that one routine paints all three grids, which is why all three cell builders take the test. |

## Where this story's own measurement disagrees

`ITEM-WEAR-056` states that weapon row `rem` "carries no parameter array at all, so the slot is
absent and both bits stay clear". Measured here over both preserved roots, that row's parameter
array is present and its cell 15 holds -1. The measurement is recorded in verification.md. The
implementation follows `ITEM-WEAR-055`'s instruction-level rule — bit 0 and bit 1 of the value,
copied independently — which makes -1 read as usable by both. No shipped behaviour depends on it:
no path in this build resolves that row.

## What is authored rather than decoded

The refusal is silent. The original calls a screen method on refusal, which `ITEM-WEAR-057` names by
address but does not identify; whether it is a sound, a message or a cursor change is not decoded, so
this build refuses without feedback. That is a gap in what the player is told, not a divergence in
what happens.

`ShopBackUnusable` as a fourth enumerated state is this build's own structure. The original has one
picture used by two arms; this build needs the arms distinguishable because `Occupied` is derived
from the state rather than from the element's quantity.
