# 1052 — destructible structures

## Intent

Placed structures have live health. Area attacks can destroy them and the map
then draws their shipped ruin art; that state survives hashing and save/load.

## As built

- Each placed type-4 record carries current and maximum health plus its authored
  footprint. Both health words start at the Buildings row's `healthMax`.
- Story 1086 supersedes the initial single-cell area slice: mask-selected
  registered references, collision-prefix retention and Building's byte damage
  resolver now serve blast and staged-ring damage. Clouds do not read the
  structure slot. See `../1086/story.md` for the current contract and proof.
- Signed health at or below zero is destroyed. A destructible class draws the
  ruin grid at the end of its sheet; an `Indestructible` class keeps intact art.
  The authored footprint remains in place.
- Structure health, maximum and shape enter the canonical hash and save form.
  Form 65 widens the old six-byte record. Loading form 64 preserves saved
  current health, then restores only immutable shape from the opened mission.

Claims used: `ALM-CLS-053`, `SAV-BLDG-037`, `MAGIC-AREACELL-039`,
`MAGIC-AREAAPPLY-038`, `TERR-STRUCT-102`, and `SPR256-STR-041`.

## Proof

`TestReleaseMission101FireBallDestroysAndDrawsTheShippedSwitch` opens the real
EN and RU mission 101, casts Fire Ball through the production map-input seam at
structure 11 (class 29, cell 54,54), observes health 1 -> 0, independently
checks every live draw entry against the installed ruin block, and reloads the
destroyed state. Focused simulation tests also cover Hail's staged ring, cloud
exclusion, signed health, exact clamp, digest movement, and form-64 migration.

The single adversarial return found that the shared upgrader appended a second
four-byte book-cast lifecycle tail to forms 62 through 64. The corrected walker
adds that tail only to the legacy 20-byte record and copies the complete
24-byte record unchanged. The mission-101 witness now saves on the Fire Ball
impact tick with that record live, then restores the full encoded envelope from
forms 62, 63, 64 and the current form, requiring identical canonical bytes,
hash, destroyed health, repaired footprint and ruin draw.

## Open debt

- Direct physical attacks use a separate ROM1 structure-strike path and are not
  part of this spell-first slice (`DIV-457`).
- Story 1086 closes the multi-cell exclusion (`DIV-458`). Original destruction
  scheduling remains Unknown; retained aliases and terrain-plane limits are
  disclosed in `DIV-552` and `DIV-553`.
