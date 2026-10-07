# Multi-cell structure area damage

## Intent and authority

Area spells reach each registered structure cell, change live HP and ruin art,
and continue identically through native saves. The authority is research pin
`26c755b2be9513b9ec90edba6527db4fdd613dc4`: `UNIT-STRUCTCELL-070`,
`UNIT-AREAVISIT-071`, `UNIT-AREADIRECT-072`, `UNIT-AREAHP-073`,
`UNIT-STRUCTDETACH-074`, `UNIT-AREAPOP-075` and corrected
`MAGIC-AREAAPPLY-038`. These establish local mechanism, not original live
destruction scheduling or a universal rectangle-times-damage formula.

## As built

The simulation derives structure cell aliases from immutable saved placement
records in ID order. A mask-selected cell uses the five-bit mask index and
low-word registration key. The first occupied cell ends that placement's
registration without discarding its prefix. Neither missing mask cells nor a
refused suffix receive damage. Ring and blast walks read the current slot after
ground and air. Clouds do not read it. Blast visits are x-major with independent
byte truncation; rings retain their byte truncation and eight-cell inset gate.

Damage calls use Building's byte base/spread resolver and word HP subtraction
with signed clamp. Building's Fire Ball divisor is one. Fire Sacrifice's stored
damage pair is already constructed and is not power-scaled a second time.
Ordinary timed effects
retain their own replacement and refresh rules; the cell walker does not
deduplicate damage objects. Existing native structure records contain enough
immutable data to rebuild the slots. No record width is changed. Old native
saves preserve HP and shape but resume under the corrected rules, not a frozen
emulation of the former 1x1-only implementation.
The written form remains 67, inherited from the reconciled physical-attack
landing; the structure record remains 22 bytes.

Againrom retains ruined structures and their aliases (`DIV-552`), including subsequent
damage calls at zero HP. This is a continuation policy, not evidence that the
original immediately detaches or persists forever. Runtime building creation,
destruction scheduling, passability recompute timing, town interaction and an
original-SAV writer are outside this slice. The loader's pre-existing playable
terrain clipping remains distinct from un-clipped damage aliases (`DIV-553`).

## Proof and remaining debt

Asset-free expectations independently enumerate mask holes, five-bit mask
wrapping, collision prefixes, low-word edge keys, x-major blast order, ring
inset rejection, per-reference RNG/damage, zero-spread gates and zero-HP slot
retention. They distinguish direct damage from timed replacement and untimed
ordinary effects. A two-stage ring crosses native save/load before landing,
between damaging stages and after completion, comparing live-vs-loaded reports,
hashes and canonical bytes. Legacy shape repair rebuilds aliases without
resurrecting saved zero HP. A lethal multi-cell ring also cancels a physical
pursuit after native load, preserving the current crossing's transit payment.

`TestReleaseMultiCellStructureAreaSpell` passes on both preserved roots. A
controlled one-mage arena retains the real mission-10 map, all placed
structures, terrain, spell rows and art. Book and map clicks pass through
`App.step`; structure 4 has a 3x3 mask `0x1ff`, HP 1000 and no HP fixture writes.
Three Fire Ball casts produce `1000 -> 674 -> 359 -> 0`. All nine draw entries
select installed ruin frames and the live HP card pixels change. SaveStore
round trips before casting, during book windup and after ruin compare canonical
bytes and twelve subsequent reports/hashes. This is an installed-data headless
drive, not an original runtime witness or a physical desktop observation.

The former blanket exclusion `DIV-458` is closed. Exact commands, reconciliation
and remaining final gates are in `verification.md`. No shared current build is
written by this lane.
