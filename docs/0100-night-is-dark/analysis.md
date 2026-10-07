# 0100 — night is dark: analysis

## The gap this closes, named by the code that carries it

`pkg/render/terrain/sun.go` ships `SunAt` as three lines: `DefaultDaytime` with `Theta` replaced.
Its own comment states the consequence — *"THE SUN MOVES BUT NOTHING DARKENS"* — and story 0092
disclosed it as divergence **D-1** rather than as an omission, because at the time the dawn, dusk
and night arms "compute theirs through sequences nobody has read".

They have been read since. `TERR-LIGHT-119` publishes the whole schedule at High: six arms, every
immediate re-read at the address of the instruction carrying it, with the ramp arithmetic
`m = t mod 120`, `q(k) = (k*m)/120`. So D-1's stated reason for existing no longer holds, and this
story deletes it. Nothing here is invented; the one number that is ours is named in `spec.md` D-2.

## What was not known before opening the code, and what looking answered

**Does the ambient reach sprites?** Yes, and for free. `pkg/ui/statics.go` already returns
`terrain.SpriteRow(v.sun)`, and `SpriteRow` is `ambient>>2` (`TERR-LIGHT-061`). 0092 wired the seam
and its comment says so in as many words: *"the day the per-band intensity schedule is decoded
(0092 D-1), sprites follow the sun because they read the same cache the terrain does."* The row
therefore moves 3 → 8 across the day with no call site changed. `TERR-LIGHT-125` predicts exactly
that: *"objects and units dim in step and never independently."*

**Does the tint reach anything?** Sprites, but only once. `spritePixels` bakes `v.sun.SkyTint` into
the texture while `spriteTextureKey` carries `{frame, row}` alone — 0092 DD-3a names the bug it
would become: *"a tint that varied would serve the first band's textures for the whole session."*
It varies now, so the key is widened. This is the one defect already in the tree that this story's
own change would otherwise activate.

**Does the tint reach the ground?** It does not, at all. The viewer draws terrain through
`DrawTriangles` with per-corner vertex colour multipliers (`withScales`), and a vertex colour is a
multiply — it cannot carry an addend applied *before* a multiply. `SkyTint` has no reader in
`pkg/ui` outside the sprite path. Left alone, this story would ship blue units standing on grey
ground, which reads as a defect rather than as a night. The composition that fixes it is arithmetic
rather than invention and is derived in `plan.md` DD-4.

## What was checked before it was believed

`TERR-LIGHT-123`'s continuity property was re-derived by hand across all six joins before any code
was written, from `TERR-LIGHT-119`'s immediates alone: the largest step in any of R, G, B, ambient
and range is 1, and two joins are exact (night → dawn 1, day → dusk 1). An arithmetic slip in
transcribing the schedule breaks it immediately — the research round's own interim reading produced
`B = -47` and a 95-step discontinuity — so it is carried into the test suite as the discriminator it
is, not as a restatement.

The free agreement was taken as well: `TERR-LIGHT-119`'s cycle-off and day arms are the same seven
immediates, and they are what `DefaultDaytime` has carried since 0007 by a different route. Two
independent readings, one answer.

## What is deliberately not here

The shadow and silhouette pass. `TERR-LIGHT-126` establishes that `0x49c`/`0x4a0` move with the band
too (4/2, 6/3, 8/4). They are carried on `Light` because the schedule is being read here anyway and
two bytes cost nothing, but nothing reads them and no drawing changes. The story that draws shadows
inherits them rather than re-reading `R1813`.
