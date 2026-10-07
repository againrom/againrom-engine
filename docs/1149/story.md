# Fire leaves burned scenery

Fireball and Wall of Fire mark the cells they affect. Destructible objects
there draw their installed dead sprite, and ground retains its scorched
appearance after fire expires and after native SAVE/LOAD.

REG-OBJ-047 establishes DeadObject as the object-class subscript.
TERR-SPR-042 selects its File at frame0 on a tile carrying bit13;
TERR-DIRT-017 supplies the existing ground composite. MAGIC-WALLFIRE-058
establishes Wall of Fire's persistent client mark over destructible objects.
The owner extends burning and persistent ground marks to Fireball. The exact
original Fireball marking footprint and original mark lifetime remain Unknown.
Using the dead class canvas, center and ambient selector is integration policy;
the cited claims establish File/frame replacement without proving those fields.

This slice changes scenery, not terrain passability. Ruined buildings and
charred objects retain the current movement rules. Fountain, lever and arch
interactions are a separate player result.

Form88 stores the sorted affected cells in a bounded footer. Earlier native
forms default to an empty history. It does not reconstruct fire that had
already expired before an earlier executable wrote a save. The map's authored
tile words remain immutable.

The ordinary window texture upload now applies the keyed dirt composite and
includes its four variants in the texture cache key. The CPU terrain renderer
already applied it. Water retains its surface; an object on water can still
select its dead sprite.

Focused proof passes: literal fire footprints, expiry and cold continuation,
old-format migration with unchanged historical87 hashes, atomic malformed
footer rejection, literal dead-sprite geometry, and ground upload palette/key
checks. TestReleaseFireScenery1149 uses actual App spell clicks and installed
mission10 data independently for each spell; both keep their scene and world
hash through ordinary SaveStore and a fresh viewer. The release manifest names
this test. The seat implementation journal owns final gate and executable
evidence for the exact landing. DIV-1026 records the bounded original evidence and remaining policy.
