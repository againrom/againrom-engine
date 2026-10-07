# Retained overlays and projectile render passes

Split the production map composition into the promoted ANIM-047 order:
Wall of Fire/Earth, complete unit shadows, projectile list, Freezing/Poison
clouds, then complete unit bodies. ANIM-047 is Medium; it does not establish
projectile-list traversal. MAGIC-OVERLAYART-051 supplies the four retained-art
arms, selector split, Fire-before-Earth order and Freezing-over-Poison priority
at one cell.

The existing flat-ground pass and the depth merge of objects, structures,
sacks, actor bodies and actor-local marks remain intact. Retained spells gain
presentation-only pass tags; the client keeps its existing projectile sequence.
No simulation, damage, expiry, save or hash changes are in scope. The original
position of unrelated non-unit objects relative to these spell passes remains
outside the two claims; assigning the existing content plane to the body pass
does not claim that broader order is decoded.

Independent tests use literal pass/alpha expectations and reversed-input
producer cases. The production drawArt witness includes two full shadow/body
sweeps, interleaved incoming spell tags and the literal same-cell pixel
(51,88,152,255). Replacing the spell snapshot clears all three spell passes.

The EN/RU witness loads real effect art and exercises actual drawArt submissions
through a CPU test target, with synthetic translucent unit markers. Its raw
palette/coverage oracle checks 441 overlapping pixels for each cloud variant;
all differ from the old final spell band. This is not a GPU screenshot or an
original GUI witness. No generated art or install byte is committed.

DIV-079 closes the five-pass mismatch. DIV-528 retains projectile traversal,
first-seen cell order, non-unit content placement and the separate shower's
broader order limits. DIV-529 through DIV-531 are unused. Fog, relief, phase,
lifetimes and resource algorithms are unchanged, not newly certified here.
See verification.md for candidate and gate evidence.
