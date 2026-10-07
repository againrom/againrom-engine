# 0161-spell-art — provenance

## Claims this story builds on

| Claim | Confidence | What is taken from it |
|---|---|---|
| `MAGIC-PIC-026` | High | The picture a cast names is computed: `2*spellId + 8` for the flying object, `2*spellId + 9` for the burst. Seven spells compute an id with no row at either parity. |
| `MAGIC-CASTSPAWN-033` | High (Medium for the muzzle table) | The caster builds the cast object at the `ShootDelay` tick; `picture = actionspell`; the flight length comes from a 51-byte index table with six arms — `dist/200` for picture 10, `dist/384` for 12, `1` for 20 and 30, `13` for 34 and 36, `21` for 60, `0` for the other 44. A zero-length object executes no driver arm. Picture 60 allocates a second object at the caster's own bounding-box centre. |
| `ANIM-PROJ-025` | High (Medium per-arm) | The driver: `step = (actionx - x) / actionsegments` per axis, `actionphase` incremented, `actionsegments` decremented, finished at zero. So `actionsegments` is a lifetime in ticks and the object reaches the target at the end of it. |
| `ANIM-PROJ-026` | High (Medium for the palette fork, the lazy load and the smoke trail) | The draw: art centred on the object by registry `Width / 2` and `Height / 2`; `facing = (dir - 8) & 0xf`; `Flip` mirrors facings 9..15 onto 7..1, leaving nine stored; `frame = Phases * facing + phase`, overridden to `frame = phase` when `RotationPhases == 1`; a picture past the array and an empty slot both draw nothing. |
| `ANIM-PHASECLOCK-028` | High | A projectile advances one sheet frame every two game ticks: `phase = (actionphase / 2) % Phases`. Picture 60 takes `actionphase - 1` with no modulus, picture 51 takes `actionphase` raw, pictures 34 and 36 take a 13-entry ramp yielding the constants 4, 3, 2, 1. |
| `MAGIC-BURSTLIFE-034` | High for the fields, the stationarity and the three lifetimes; Medium for reading `effect+0x0c == 2` as spell 2 | The burst is a stationary object: `actiontarget` cleared, `actionx`/`actiony` set to its own position, so every driver `IDIV` divides zero. `actionsegments` is a lifetime in ticks: 16 by default, 18 on the `acid_stream` arm, 22 from the second sender. A ring advances every 2 ticks. Sound id `500 + picture`. |
| `REG-PROJ-086` | High for the field-by-field loader; Medium for the per-key present-or-absent counts | `projectiles.reg`: `[Global] Count`, sections `Projectile%d`, `File` as a string, and the defaults each absent key takes — `ID` -1, `Phases` -1, `RotationPhases` 0x10, `Width` 0x40, `Height` 0x40, `Palette` 0, `Homing` 0, `Flip` 0, `SFX` 0. The record array is keyed by `ID`, so every consumer indexes by picture id. |
| `REG-PROJ-087` | High | `Palette` is a boolean: the sheet carries its own colour table. It agrees with the sheet trailer's bit 31 on 31 of 31 rows. The three rows that default to 0 are `archer\arrow`, `xbowman\arrow` and `orc\arrow`; their frames begin at offset 0, not 1024. `A16` selects the sprite class, and therefore the extension. |
| `SPR16A-CAST-028` | Medium | The spell-to-sheet join: 15 cast rows and 8 burst rows over the 28 spell ids; exactly two rows rotate (`firebolt` and `fireball`, `RotationPhases = 16`, `Flip = 1`, `Phases = 4`, 36 frames each); every other spell-reachable row has `RotationPhases = 1`. Registry `Width`/`Height` are centring halves, not art dimensions. The 26 spell-reachable resources are byte-identical EN/RU. |
| `SPR16A-PROJ-024` | High | 24 of the 31 rows are `.16a`; frame geometry is uniform inside a sheet; `healing` holds 8 frames against `Phases = 7`, one frame the draw can never reach. |
| `SPR16A-ALPHA-025` | — (already implemented) | The `.16a` container and its 16-step blend. The tree's decoder is `pkg/formats/spr16` and the blend is `cursorPixel` in `pkg/game/cursor.go`, reused rather than restated. |
| `MAGIC-DELIVER-035` | High for the field's role; Medium for the four-spell agreement | The simulation computes its own flight time from `Delivery System` and parameter 7, and on the normal path the client's own switch wins. This story draws the client's, so the `Data.bin` column is read by nothing here. |

## Ours by choice

- The 16-way direction a bolt flies at is derived from the flight vector by an
  angle split into sixteen 22.5-degree sectors. The engine's own routine
  (`L02828`) is not published, so the sector boundaries are ours. What is
  decoded is the space they land in: the fold to nine stored facings and the
  `Phases * facing + phase` index are `ANIM-PROJ-026`'s, and the wheel is the
  eight-way sheet ordering this tree already carries in `sheetOctant`, doubled.
- The distance a flight length divides is computed as an integer square root of
  the cell delta scaled by 256. The engine calls `ftol` on a distance term; the
  rounding of that term is not published.
- Pictures 34 and 36 use the default one-frame-per-two-ticks clock. The decoded
  ramp's per-index constants are not published — `ANIM-PHASECLOCK-028` names the
  four values it yields and the 13-entry table's bound, not the mapping.
- A burst is drawn where a cast this build applies has a defined burst row. In
  the original the burst object is built by the two area senders, and this build
  models no area delivery, so the trigger is ours and the object is the decoded
  one.

## Open, and how it is closed here

- Whether a zero-length cast object is drawn for one frame before the driver
  reports finished (`EXP-0167`). Closed by authoring: it is not drawn. A picture
  with a zero arm executes no driver arm, so it has no phase clock and no
  displacement to run.
- The `+0x124` / `+0x128` effect lists (`EXP-0167`). Omitted.
- `MAGIC-CASTSPAWN-033`'s muzzle table (`classRecord+0xec`, `+0xf0`) is Medium
  and its field names are not established. The bolt leaves the caster's own cell,
  which is where 0154 already put it.

## Removed

- 0154's authored bolt square and expanding burst ring (`pkg/ui/spellbolt.go`).
  0154 wrote them because nothing in the tree loaded `projectiles.reg` or the
  sheets behind it and disclosed them as a stand-in. Both are gone.
- 0154's `boltBurst` constant, four ticks. The burst's life is decoded.
