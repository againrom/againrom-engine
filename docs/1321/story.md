# Paused world presentation

## Intent and authority

Active player pause stops world presentation as well as simulation. The owner
requires retained foliage, well, building, water and floating damage animation,
with a subtle dim that leaves selection, camera, commands and UI usable. This
extends the existing Space and zero-speed routes in story1316. DIV-2213 records
owner direction. No original active-pause behaviour is claimed.

## Behaviour

Viewer holds the shared animation count and its fractional remainder while
player pause is set. Both paced and unpaced modes consume the paused wall
baseline. The first resumed frame retains the same phase and excludes its span
since the last paused sample. Following frames continue at the retained cadence.
This reaches scenery, structure sight phases, water and tower-light pulses.
Actor frames, movement interpolation, projectiles and spell-effect phases remain
on their existing stopped World driver.

Each floating figure retains its offset and independent remaining lifetime.
Paused wall spans shift its birth timestamp before expiry; the first resume span
is also excluded. New damage can still be ingested through the existing wrapper.
Sound delivery and entity retirement still run every frame. A popup alone keeps
the previous policy: drift stops and numeral wall lifetime continues.

A 32/255 black wash covers the world viewport after terrain, art, Heal/Drain and
shroud. Selection, command previews, paths, numerals, HUD, dialogs and cursor are
drawn above it. The wash disappears immediately when pause clears. Messages,
tooltips and cursor keep their existing UI clocks. No persisted field, World
rule, queue policy or save producer changes. Restored player pause supplies the
same existing bit to this presentation route.

## Proof

The first engine commit is 6b5f2116. It bumps only againrom from 0.55.0 to 0.56.0
and regenerates its Windows resource; starter remains 0.3.1.

Three RED behavioural controls measure animation count 1 becoming 164 paced or 102
unpaced during pause, a numeral spending its flight and lifetime, and an absent
world dim. GREEN controls cover exact fractional phase, a one-hour pause plus an
unobserved resume span, inclusive remaining lifetime, audio delivery and absent
entity retirement, zero-speed pause, bright HUD glyphs and resumed dim absence.
Existing selection/camera/order, saved pause, exact cadence and ordered-SAV tests
remain the preservation controls.

TestReleaseActivePauseRenderedWorld is registered in the gated population.
It opens installed mission 20 through App, applies FrontEnd.LiveDamage through
the script health write and real driver tick, and reads GPU pixels from Viewer.
It compares source World bytes and an admitted command, writes current SAV, and
holds the visible damage glyph for 150 frames. A separate scenery App uses the
production LoadMapViewer and the installed map: foliage, well, building and
water each show a moving-pixel control, 150 exact paused frames and a byte-exact
bright frame on resume at retained phase. The hidden GPU child takes no desktop
input. PNGs, SAVs, JSON proofs, source hashes and command receipts stay outside
Git under review/story1321-paused-presentation/author/.

The seat runs the sole review and canonical merge chain. Candidate publication
and focused evidence do not constitute a landing or promoted build.

## Open debt

Transient figure records and ambient frame phase are not added to SAV. Existing
SAV representation limits and original-runtime acceptance remain inherited.
Native sound audibility and owner desktop interaction are not measured by this
offscreen witness. Research pin k159 remains unchanged because this result uses
owner direction and no new claim.
