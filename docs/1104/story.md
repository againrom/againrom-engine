# Source-backed second physical damage

An eligible imported Human carries its saved second physical base/spread into
combat. School Train can retain the modifier-derived pair, ordinary SAVE writes
SAV, and fresh SAV/AGS LOAD and mission entry preserve the result.

## Contract and authority

- The live pair projects through `data.Combat`, `sim.Entity` and `CombatBlock`.
  Canonical form 72 appends two bytes to each form-71 entity; older forms default
  them to zero. Books and original-dead records retain their existing layouts.
  LOAD rejects impossible entity spans before narrowing the wire count or
  allocating migration state, including 32-bit overflow boundaries.
- `HERO-DMG2-029` admits the second roll whenever either byte is nonzero,
  independently of the primary hit, absorption and weapon resistance. It uses
  signed Water protection and truncation after adding 0.75, then clamps at zero.
  `HERO-CLAMP-030` preserves the negative primary contribution and final clamp.
  The existing third-component admission rule does not change.
- `SAV-HUMRUN-444` and `SAV-HUMFOLD-446` distinguish current and modifier pairs.
  LOAD projects current bytes; Train clears then folds the modifier bytes.
  `HERO-RESIST-012` identifies the Water word at target +0xc6.
- `SAV-HUMGAPS-449` leaves nonzero modifier producers Unknown. Live native
  rearm/skill recompute retains the observed canonical pair under DIV-675,
  without claiming those producers. SetCombat itself remains full replacement.

## Boundaries

Retired or invalid source bases still refuse native fallback Train/mission
entry when either current or modifier pair is nonzero. Native zero-pair training
remains available. CarryParty still clears the source basis (DIV-675); this
contract ends at the entered mission and its native continuation, not a later
mission settlement or original-world SAV export. No original process or lawful
install writes are required. Release probes may explicitly mutate a scratch
copy of a lawful city save; those are synthetic inputs, not natural occurrences.

## Proof

`verification.md` records independent scalar/RNG cases, literal Train fields,
canonical old-form and continuation checks, paired EN/RU App routes, and gates.
