# Analysis — the sun is not fixed, and this build has it fixed

**Intensity:** spec-anchored / static. **Terrain:** brownfield in `pkg/render/terrain` and `pkg/ui`
— a relief-lighting grid ships and is built once from one constant sun — and greenfield for the sun
model itself, which has no counterpart in the tree.

## What was not known when 0007 shipped, and what closed it

`pkg/render/terrain/light.go` builds its level grid from `DefaultDaytime`, whose angle is the
literal the engine stores **when the cycle is switched off**. The file says so, and it says why the
question was left open: *"Whether shipped play runs with the cycle off is an open research
question; this value is a labelled reference point, not a claim about what a live session uses."*

That question is now closed, and it closed the other way. The switch is a shipped, persisted user
option whose compiled default is **on**, stored by a constructor the static-initializer table runs
before `main`. So a fixed sun is not a feature this build has yet to add — it is a **divergence the
build currently has**, and every level statistic in this tree was measured on the off-by-default
arm.

Two older rows also said the angle swept `+-pi/2`. Both were wrong by a factor of two, both were
retracted, and both were wrong for the same reason: the operands were read off a disassembler's
symbol names while the neighbouring literal was read off its own bytes. The corrected sweep is
`+-0.78539815` — the same truncated-pi quarter this tree already stores as `DefaultTheta`, so the
constant the tree holds was right and the span it wrote in prose was not.

## The reading, re-executed before anything was designed

Six published landmarks were reproduced from the claim rows alone, in a scratch program outside the
repository, before a line of the contract was written. Each discriminates: a wrong band boundary, a
wrong step constant, a wrong `FSUBR` operand order or a wrong modulus fails at least one.

| Landmark | Published | Reproduced |
|---|---|---|
| Full ticks per day carrying a computed angle | 960 of 1440 | 960 |
| Band census over a day | day 720, night 240, dawn 240, dusk 240 | identical |
| theta extremes over a day | `-0.78539815 .. +0.78539815` | identical |
| `tan(shear(theta))` over the day band | `-0.5774 .. +0.5754` | `-0.5774 .. +0.5754` |
| Dead-band full ticks in the day band | 22 + 22 | 44 |
| `tan(shear(theta))` with the cycle off | `0.57735025728078282` | equal to all 17 digits |
| Unforced relights per in-game day | 72 | 72 |

Two arithmetic identities fell out of the same run and are worth recording because neither is
asserted anywhere and both would have caught a mistyped constant: `720 * 0.0021816615277777777`
is `1.5707963`, a truncated pi/2 rather than pi/2 itself, and the night step is **exactly three
times** the day step, which is the "same arc, three times faster" the band structure describes.

The doubles were checked as bytes, not as decimals: `0.78539815` is `3fe921fb4d12d84a` and `pi/4` is
`3fe921fb54442d18`. They differ from the eighth decimal, and the tree already refuses to substitute
one for the other.

## Where the clock came from

The sun model's argument had a shape (`(clock>>4)+0x168`) and no unit. It has one now: the counter
it shifts is the server sub-tick counter, sixteen sub-ticks are one full tick, and **one full tick
is one in-game minute**. That is the same 16:1 cycle `pkg/sim` already runs the mission script and
the engagement decision on, so this tree's own tick is that sub-tick and the conversion needs
nothing new.

It also fixes the phase: at tick zero the argument is 360, the hour is 6, and a mission opens at
**06:00** — the first minute of the daylight band, with the sun at `-pi/4`.

## What the angle can and cannot reach here

The angle reaches the level grid twice, through `stepH = 32/cos|theta|` and the lateral `tan|theta|`
shear, so the relief shading moves across the day and the shear reverses at noon. It reaches sprites
not at all: the sprite ramp row is `ambient>>2` and carries no angle term.

So whether anything **darkens** at night depends entirely on the per-band ambient/tint schedule —
and that schedule is the one part of the routine the round did not read. The dawn and dusk arms
compute their tints through magic-division sequences nobody has traced; the day arm and the
cycle-off arm are the two that are known, and they agree. This story therefore moves the angle and
holds the intensity, and the gap is filed as an open item rather than papered over with an invented
gradient.

## What was considered and not done

Reproducing the shadow shear was considered and dropped, because **this tree draws no shadow**: no
unit, structure or object shadow pass exists, so the shear would ship with no caller. The shear
arithmetic was still re-executed above, as a cross-check on the angle — the cycle-off tangent is a
17-digit discriminator and it passes — and the asymmetry a future shadow pass must reproduce is
recorded in the contract's out-of-scope section so that nobody later "fixes" it.
