# Story 1055: authored current health and rescue triggers

## Player result

Units placed wounded or fallen by a mission now enter with that exact current
health instead of being silently filled to maximum. The fallen Cavalrymen in
mission 71 and Ogre in mission 81 can be healed, get back up without losing
their equipment, and let the shipped mission scripts deliver their reactions.

## Authority and data contract

`UNIT-PLACE-034` identifies type-6 offset `+0x20` as signed current health and
raw `-1` as the derive-from-maximum sentinel. `MAGIC-TARGET-017`,
`HERO-REVIVE-068`, `HERO-ZERO-070`, `TRIG-PROPERTY-036`, `TRIG-CHECK-051` and
`TRIG-REAP-017` define the target, revival, zero-health and script boundaries.

- `alm.Unit.CurrentHP` retains the signed word and `HasCurrentHP` distinguishes
  an authored value from raw `-1`. Synthetic maps write `-1` unless explicitly
  given an override.
- Placement derives and difficulty-adjusts `MaxHP` exactly as before, then
  applies authored current health without scaling or clamping. This also keeps
  positive custom-map overrides and unresolved classes intact.
- An authored non-positive actor begins at the fallen presentation and receives
  the death-time Defence half once. This setup never calls `clearFelled`, so
  authored stock and worn equipment remain on the actor.

The independent raw-frame census pins 30 overrides on seven maps in both
lawful installs. At Normal difficulty, mission 71 map-unit ids 42 and 43 start
at `0/248`; mission 81 map-unit id 50 starts at `0/320`.

## Lifecycle

- Exactly zero health is a stable point under decay. Negative bodies continue
  through the existing decay ladder.
- Heal admits a target with positive maximum health at current health 0 or -1
  through -9. It refuses -10 and below and actors without a health system.
  Buffs and other support still refuse bodies; damaging finishers and Control
  Spirit retain their separate routes.
- A body in that restorative band keeps its movement-layer cell and cell-record
  actor slot after Dwell expires. A mover cannot take the coordinates before a
  Heal or script health write restores the actor. A terminal body at -10 or
  below still releases the cell after Dwell.
- A Heal or script property write crossing from non-positive to positive clears
  Decay and Dwell and restores Defence exactly once. A later Heal cannot repeat
  that restoration.
- Script death check 18 requires the dying window to have closed and health to
  be at or below -10. Mission 71 then fires `Horsemans Healed`; mission 81 fires
  the Ogre reward path after the production Heal crossing.

Potions and continuous positive effects are not broadened into resurrection.
Native saves already carry health, Decay, Dwell, Defence, equipment and the
world hash: resume preserves those saved values and does not reapply ALM. No
guess is made for an old save whose full health could mean either legacy load
behaviour or a genuinely healed actor.

## Proof and closure matrix

| Aspect | Result and witness |
|---|---|
| Format | Raw `-1`, `0`, `-10` and positive values decode distinctly; document round-trip leaves all four frames intact. |
| Synthetic input | Type-6 synthesis defaults to `-1` and explicitly emits signed overrides. |
| Placement | Human, creature and unresolved placements prove exact current health while Easy/Normal/Hard still alter only derived statistics. |
| Initial body state | Duplicate preparation and repeated world construction prove one Defence half, fallen presentation and stable HP 0. |
| Occupancy | After Dwell, a mover cannot enter an authored HP-0 body's cell; Heal and script restoration produce one living actor per cell, while terminal HP -10 releases the corridor. |
| Combat targeting | Heal accepts 0/-1/-9 and refuses -10/no-health; damaging finishers keep their boundary. |
| Spell effects | Support buffs remain forbidden on bodies and continuous effects cannot resurrect. |
| Script lifecycle | Instant/property HP `1` revives; signed `0xffff` becomes -1; VIP death waits for the -10 floor. |
| Equipment | Initial body and revival retain worn/carried instances and create no death sack. |
| Persistence | Initial and healed forms round-trip with equal bytes/hash, health, decay, defence and stock. |
| Session resume | A native snapshot restored over a freshly loaded ALM world keeps saved initial or healed state and never reapplies the placement override. |
| Shipped scripts | The production mission-71 and mission-81 Heal commands fire the exact latched script events and expected raises. |
| Lawful installs | One gated, package-independent raw-frame walker asserts the exact 30-record corpus and runs the trigger/save/hash witness on EN and RU. |

## Review and divergence disposition

After push this story receives exactly one fresh-context adversarial pass. Only
a player-visible or hashed-state `P` finding returns one bounded correction;
pass or non-`P` findings stop the review. No second review is requested.

Story 1055 closes `DIV-220`, `DIV-221` and the superseded Heal conflict in
`DIV-442`. `DIV-219` remains open and is amended: rescue now exists, while this
build still deliberately defers mission loss and has no ROM1 re-placement arm.
Allocated `DIV-475` through `DIV-482` were not needed and are returned unused;
they remain retired.

Open debt is deliberately narrow: no old-save health inference, no
potion/continuous resurrection, no change to mission-loss re-placement, and no
decode of unrelated ALM type-6 words.
