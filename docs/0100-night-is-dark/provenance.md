# 0100 — night is dark: provenance

Research pin: `research/` at `e6f9ee6`. Every row below was read with `tools/claim` from that pin.

## The schedule

| Claim | Confidence | What it supplies |
|---|---|---|
| `TERR-LIGHT-119` | High | The whole schedule. Six arms of `R1813`, the three tint bytes and the two intensity fields per band, `m = t mod 120`, `q(k) = (k*m)/120`. Every immediate re-read at the address carrying it. This is FR-1 and FR-2 entire. |
| `TERR-LIGHT-121` | High | That there are **six** arms and not five: each twilight band splits at its own `CMP` into two halves that agree on the angle and write different colour ramps. Why FR-1's band table has more rows than `SunAngle`'s. |
| `TERR-LIGHT-122` | High | The ramp divisor is 120 and the hour selector 60, both re-executed from the multiplier and post-shift read out of the PE against exact integer division with 0 mismatches. Also that `m = t mod 120` reduces to `minute mod 120` because `360 mod 120 = 0` and `1440 mod 120 = 0` — which is what lets FR-2 be written in the reduced form. |
| `TERR-LIGHT-123` | High | The continuity property: over all six joins the largest step in any of the five values is 1, two joins are exact, and all five stay in `[0,48]` across a whole day. Carried as AC-4, a discriminator the routine never asserts. |
| `TERR-LIGHT-124` | High | `0x490/0x491/0x492` are **R, G, B** in ascending address order, established twice over and with the rival BGR reading excluded. The night sky is blue, not red. |
| `TERR-LIGHT-128` | High | The reachable-state bound: 24 distinct `(R,G,B,ambient,range)` tuples through the unforced relight cadence, 170 through the forced path. This is what bounds the two texture caches in FR-6/FR-7 (DD-5). |

## The transforms the schedule drives

| Claim | Confidence | What it supplies |
|---|---|---|
| `TERR-LIGHT-127` | High | The tint is masked `AND EAX,0xff` — an **unsigned addend applied before the level multiply**, never a signed offset. All darkening is `0x494`/`0x498` through the level; the tint can only push a channel up. FR-8 and AC-6 are this row. |
| `TERR-LIGHT-019` | High | `out = clamp(((chan + tint) * (LevelCount - level)) / 32, 0, 255)`, already shipped as `ShadeChannel`. DD-4's composition is this expression factored, not a new one. |
| `TERR-LIGHT-020` | High (arithmetic) | Level 64 is the unattenuated identity row and higher level is darker. The identity row is what makes DD-4's factoring exact. |
| `TERR-LIGHT-021` | High | The sky tint enters the shading table and the two intensity bytes do not; their only readers are the per-vertex level's base and swing. Why `Ambient`/`Range` reach the screen only through `LevelGrid`. |
| `TERR-LIGHT-061` | — (via 125) | The sprite ramp's row is `ambient>>2`, a single map-wide byte. Already shipped as `SpriteRow`. |
| `TERR-LIGHT-125` | **Medium** | The figures this story is judged by: flat ground runs `[45,64]` over a day — **46** by day and with the cycle off, **64** at night — and the sprite row runs 3 → 8 through 5, 6, 7. Medium because it is this round's High schedule evaluated through `TERR-LIGHT-013` and `TERR-LIGHT-061`, so it inherits those rows' standing. AC-5 asserts 46 and 64 as **our** arithmetic reproducing them, which is the strongest thing a Medium row supports. |
| `TERR-LIGHT-126` | High (values, schedule, doubling) | `0x49c`/`0x4a0` move with the band: 4/2 cycle-off and day, 6/3 both twilights, 8/4 night, with `0x49c = 2 * 0x4a0` in all four settings and neither computed from the other. Carried as FR-5. The *identification* of what they index is `TERR-SPR-066`'s, and this story asserts nothing about it. |

## Ours by choice, not decoded

- **The ground tint's composition** (DD-4). The engine bakes a `[96][256]` table; this tree carries
  the level as a GPU vertex multiply. Factoring `(chan+tint)*m/32` into a tinted texture times `m/32`
  is exact except where `chan+tint` exceeds 255 at a level above 64, which the 8-bit texture clamps
  early. Bounded and stated as D-2.
- **Both texture cache keys** (DD-5). No published path says anything about caching.
- **That the unshaded diagnostic tints nothing** (FR-9). A diagnostic, not terrain.

## Open, and not opened by this story

- Why the object path's shroud level is exactly twice the unit path's, and whether any consumer reads
  both in one frame (`TERR-LIGHT-126`, "still open"). Nothing here depends on the answer.
- Whether a shipped run is RGB565 (`TERR-LIGHT-019`, Medium). This tree carries 8-bit pre-pack
  channels throughout and is unaffected.
